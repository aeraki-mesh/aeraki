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
	"fmt"
	"github.com/aeraki-framework/aeraki/lazyxds/cmd/lazyxds/app/config"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/model"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils"
	"sort"
)

func (c *AggregationController) syncLazyService(ctx context.Context, id string) (err error) {
	// todo if something wrong, need put back to queue
	lazySvc := c.lazyServices[id]
	if lazySvc == nil {
		return c.deleteLazyService(ctx, id)
	}
	return c.reconcileLazyService(ctx, lazySvc)
}

func (c *AggregationController) tryReconcileLazyService(ctx context.Context, svc *model.Service) (err error) {
	id := svc.ID()
	v, ok := c.namespaces.Load(svc.Namespace)
	if !ok {
		return fmt.Errorf("namespace %s not found", svc.Namespace)
	}
	ns := v.(*model.Namespace)
	svc.UpdateNSLazy(ns.LazyStatus)

	if !svc.Status.LazyEnabled && svc.Spec.LazyEnabled {
		c.lazyServices[id] = svc
		c.lazyServiceController.Add(id)
	} else if svc.Status.LazyEnabled && !svc.Spec.LazyEnabled {
		delete(c.lazyServices, id)
		c.lazyServiceController.Add(id)
	}

	return nil
}

func (c *AggregationController) reconcileLazyService(ctx context.Context, lazySvc *model.Service) (err error) {
	defer func() {
		if err == nil {
			lazySvc.FinishReconcileLazy()
		}
	}()
	if err := c.syncEnvoyFilterOfLazySource(ctx, lazySvc); err != nil {
		return err
	}
	if err := c.syncSidecarOfLazySource(ctx, lazySvc); err != nil {
		return err
	}
	return nil
}

func (c *AggregationController) reconcileAllLazyServices(ctx context.Context) error {
	var err error
	for _, ls := range c.lazyServices {
		c.lazyServiceController.Add(ls.ID())
	}

	return err
}

func (c *AggregationController) deleteLazyService(ctx context.Context, id string) error {
	name, namespace := utils.ParseID(id)

	if err := c.removeEnvoyFilter(ctx, name, namespace); err != nil {
		return err
	}
	if err := c.removeSidecar(ctx, name, namespace); err != nil {
		return err
	}
	return nil
}

func (c *AggregationController) visibleServiceOfLazyService(lazySvc *model.Service) map[string]struct{} {
	egress := make(map[string]struct{})

	for svcID := range lazySvc.EgressService {
		binding := c.getHTTPServiceBinding(svcID)
		for id := range binding {
			egress[id] = struct{}{}
		}
	}
	// currently all tcp service should be visible
	c.services.Range(func(key, value interface{}) bool {
		svc := value.(*model.Service)
		if svc.Namespace == config.IstioNamespace { // istio-system always exported
			return true
		}

		if len(svc.Spec.TCPPorts) > 0 {
			egress[key.(string)] = struct{}{}
		}
		return true
	})

	return egress
}

func (c *AggregationController) egressListOfLazySource(lazySvc *model.Service) []string {
	var list []string

	for id := range c.visibleServiceOfLazyService(lazySvc) {
		list = append(list, utils.ServiceID2EgressString(id))
	}

	c.serviceEntries.Range(func(key, value interface{}) bool {
		_, ns := utils.ParseID(key.(string))
		hosts := value.([]string)
		for _, host := range hosts {
			list = append(list, fmt.Sprintf("%s/%s", ns, host))
		}
		return true
	})

	sort.Slice(list, func(i, j int) bool {
		return list[i] < list[j]
	})

	list = append([]string{"istio-system/*"}, list...)
	return list
}
