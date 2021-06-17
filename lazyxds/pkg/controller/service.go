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
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/model"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils"
	corev1 "k8s.io/api/core/v1"
)

func (c *AggregationController) syncService(ctx context.Context, clusterName string, service *corev1.Service) (err error) {
	id := utils.FQDN(service.Name, service.Namespace)

	v, _ := c.services.LoadOrStore(id, model.NewService(service))
	svc := v.(*model.Service)
	svc.UpdateClusterService(clusterName, service)

	return c.doSyncService(ctx, svc)
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
