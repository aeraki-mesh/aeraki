// Copyright Aeraki Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"fmt"

	"github.com/aeraki-mesh/aeraki/lazyxds/pkg/model"
	"github.com/aeraki-mesh/aeraki/lazyxds/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (c *AggregationController) syncService(ctx context.Context, clusterName string, service *corev1.Service) (err error) {
	selectors := c.selectors
	id := utils.FQDN(service.Name, service.Namespace)

	matched, err := c.matchDiscoverySelector(selectors, service)
	if err != nil {
		return err
	}
	if !matched {
		c.services.Delete(id)
		c.log.Info("Namespace label of service not match DiscoverySelector, delete it", "service", service.Name, "namespace", service.Namespace)
		return nil
	}

	v, _ := c.services.LoadOrStore(id, model.NewService(service))
	svc := v.(*model.Service)
	svc.UpdateClusterService(clusterName, service)

	return c.doSyncService(ctx, svc)
}

func (c *AggregationController) matchDiscoverySelector(selectors []labels.Selector, service *corev1.Service) (bool, error) {
	isNsLabelsMatch := false
	if len(selectors) > 0 {
		v, ok := c.namespaces.Load(service.Namespace)
		if !ok {
			return false, fmt.Errorf("namespace %s not found", service.Namespace)
		}
		ns := v.(*model.Namespace)
		for _, selector := range selectors {
			if selector.Matches(labels.Set(ns.Labels)) {
				isNsLabelsMatch = true
			}
		}
	} else {
		// omitting discoverySelectors indicates discovering all namespaces
		isNsLabelsMatch = true
	}
	return isNsLabelsMatch, nil
}

func (c *AggregationController) deleteService(ctx context.Context, clusterName, svcID string) (err error) {
	v, ok := c.services.Load(svcID)
	if !ok {
		return nil
	}
	svc := v.(*model.Service)
	svc.DeleteFromCluster(clusterName)

	return c.doSyncService(ctx, svc)
}

func (c *AggregationController) doSyncService(ctx context.Context, svc *model.Service) error {
	// todo 可以并发
	if err := c.reconcileService(ctx, svc); err != nil {
		return err
	}

	if err := c.tryReconcileLazyService(ctx, svc); err != nil {
		return err
	}

	return nil
}

func (c *AggregationController) reconcileService(ctx context.Context, svc *model.Service) (err error) {
	defer func() {
		if len(svc.Distribution) == 0 {
			c.services.Delete(svc.ID())
		}
	}()
	if !svc.NeedReconcileService() {
		return nil
	}

	defer func() {
		if err == nil {
			svc.FinishReconcileService()
		}
	}()

	if err = c.syncServiceRuleOfEgress(ctx, svc); err != nil {
		return err
	}

	// todo we haven't consider tcp
	//if len(svc.TCPPorts) > 0 {
	//} else {
	//}

	// UpdateClusterService global service
	if err = c.buildPlaceHolderService(ctx); err != nil {
		return err
	}

	if err = c.reconcileAllLazyServices(ctx); err != nil { // todo 检查必要性
		return err
	}

	return nil
}
