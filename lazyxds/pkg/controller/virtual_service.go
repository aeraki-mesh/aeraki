/*
 * // Copyright Aeraki Authors
 * //
 * // Licensed under the Apache License, Version 2.0 (the "License");
 * // you may not use this file except in compliance with the License.
 * // You may obtain a copy of the License at
 * //
 * //     http://www.apache.org/licenses/LICENSE-2.0
 * //
 * // Unless required by applicable law or agreed to in writing, software
 * // distributed under the License is distributed on an "AS IS" BASIS,
 * // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * // See the License for the specific language governing permissions and
 * // limitations under the License.
 */

package controller

import (
	"context"
	"github.com/aeraki-framework/aeraki/lazyxds/cmd/lazyxds/app/config"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils/log"
	istio "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"reflect"
	"strings"
)

// todo miss tls
func (c *AggregationController) syncVirtualService(ctx context.Context, vs *istio.VirtualService) error {
	logger := log.FromContext(ctx)

	// k8s.AddFinalizer(endpoints, meshv1alpha1.ClusterFinalizer)
	if !vs.DeletionTimestamp.IsZero() {
		logger.Info("todo endpoints deleted, 需要 Finalizer")
	}

	if utils.FQDN(vs.Name, vs.Namespace) == utils.FQDN(config.EgressVirtualServiceName, config.IstioNamespace) {
		return nil
	}

	httpChanged := c.updateHTTPServiceBinding(vs)
	tcpChanged := c.updateTCPServiceBinding(vs)

	if httpChanged || tcpChanged {
		return c.reconcileAllLazyServices(ctx)
	}

	return nil

}

// return whether the value changed.
func (c *AggregationController) updateHTTPServiceBinding(vs *istio.VirtualService) bool {
	id := utils.FQDN(vs.Name, vs.Namespace)
	binding := make(map[string]struct{})

	// todo consider delegate VirtualService and sort domain
	// Note for Kubernetes users: When short names are used
	// (e.g. “reviews” instead of “reviews.default.svc.cluster.local”),
	// Istio will interpret the short name based on the namespace of the rule, not the service.
	// A rule in the “default” namespace containing a host “reviews” will be interpreted as
	// “reviews.default.svc.cluster.local”, irrespective of the actual namespace associated with the reviews service.
	// To avoid potential misconfigurations, it is recommended to always use fully qualified domain names
	for _, host := range vs.Spec.Hosts {
		if !strings.HasSuffix(host, "svc.cluster.local") { // hard code
			continue
		}
		if strings.Contains(host, "*") {
			continue
		}
		binding[host] = struct{}{}
	}

	if len(binding) == 0 {
		_, ok := c.httpServicesBinding.LoadAndDelete(id)
		return ok
	}

	for _, hr := range vs.Spec.Http {
		for _, r := range hr.Route {
			host := r.Destination.Host
			if !strings.HasSuffix(host, "svc.cluster.local") {
				continue
			}
			binding[host] = struct{}{}
		}
	}
	if len(binding) <= 1 {
		_, ok := c.httpServicesBinding.LoadAndDelete(id)
		return ok
	}

	val, ok := c.httpServicesBinding.Load(id)
	if !ok {
		c.httpServicesBinding.Store(id, binding)
		return true
	}
	old := val.(map[string]struct{})

	if reflect.DeepEqual(binding, old) {
		return false
	}
	c.httpServicesBinding.Store(id, binding)
	return true
}

// return whether the value changed.
func (c *AggregationController) updateTCPServiceBinding(vs *istio.VirtualService) bool {
	id := utils.FQDN(vs.Name, vs.Namespace)
	binding := make(map[string]struct{})

	for _, host := range vs.Spec.Hosts {
		if !strings.HasSuffix(host, "svc.cluster.local") { // hard code
			continue
		}
		if strings.Contains(host, "*") {
			continue
		}
		binding[host] = struct{}{}
	}

	if len(binding) == 0 {
		_, ok := c.tcpServicesBinding.LoadAndDelete(id)
		return ok
	}

	for _, tr := range vs.Spec.Tcp {
		for _, r := range tr.Route {
			host := r.Destination.Host
			if !strings.HasSuffix(host, "svc.cluster.local") {
				continue
			}
			binding[host] = struct{}{}
		}
	}
	if len(binding) <= 1 {
		_, ok := c.tcpServicesBinding.LoadAndDelete(id)
		return ok
	}

	val, ok := c.tcpServicesBinding.Load(id)
	if !ok {
		c.tcpServicesBinding.Store(id, binding)
		return true
	}

	old := val.(map[string]struct{})
	if reflect.DeepEqual(binding, old) {
		return false
	}
	c.tcpServicesBinding.Store(id, binding)
	return true
}

func (c *AggregationController) deleteVirtualService(ctx context.Context, id string) error {
	if id == utils.FQDN(config.EgressVirtualServiceName, config.IstioNamespace) {
		return nil
	}
	logger := log.FromContext(ctx)

	logger.V(2).Info("VirtualService has been deleted")
	_, httpDeleted := c.httpServicesBinding.LoadAndDelete(id)
	_, tcpDeleted := c.tcpServicesBinding.LoadAndDelete(id)

	if httpDeleted || tcpDeleted {
		return c.reconcileAllLazyServices(ctx)
	}
	return nil
}

// this is only one level binding
func (c *AggregationController) getHTTPServiceBinding(svcID string) map[string]struct{} {
	binding := make(map[string]struct{})
	binding[svcID] = struct{}{}

	c.httpServicesBinding.Range(func(key, value interface{}) bool {
		b := value.(map[string]struct{})
		if _, ok := b[svcID]; ok {
			for id := range b {
				if _, ok := c.services.Load(id); ok { // todo service in vs may not exist, do we need check if it's http
					binding[id] = struct{}{}
				}
			}
		}
		return true
	})

	return binding
}
