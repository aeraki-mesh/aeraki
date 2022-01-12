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

	"github.com/aeraki-mesh/aeraki/lazyxds/pkg/model"
	"github.com/aeraki-mesh/aeraki/lazyxds/pkg/utils/log"
	corev1 "k8s.io/api/core/v1"
)

func (c *AggregationController) syncNamespace(ctx context.Context, clusterName string, namespace *corev1.Namespace) (err error) {
	id := namespace.Name
	v, _ := c.namespaces.LoadOrStore(id, model.NewNamespace(namespace))
	ns := v.(*model.Namespace)

	ns.Update(clusterName, namespace)

	return c.reconcileNamespace(ctx, ns)
}

func (c *AggregationController) deleteNamespace(ctx context.Context, clusterName string, name string) (err error) {
	logger := log.FromContext(ctx)

	logger.Info("Namespace has been deleted")
	id := name

	v, ok := c.namespaces.Load(id)
	if !ok {
		return nil
	}
	ns := v.(*model.Namespace)

	ns.Delete(clusterName)
	if len(ns.Distribution) == 0 {
		c.namespaces.Delete(name)
		return nil
	}

	return c.reconcileNamespace(ctx, ns)
}

// reconcileAllNamespace do this when namespace labels updated
func (c *AggregationController) reconcileNamespace(ctx context.Context, ns *model.Namespace) (err error) {
	c.serviceController.ReconcileServices(ns)
	return c.reconcileAllLazyServices(ctx)
}

// reconcileAllNamespace do this when discoverySelector updated
func (c *AggregationController) reconcileAllNamespaces(ctx context.Context) (err error) {
	c.serviceController.ReconcileAllServices()
	return c.reconcileAllLazyServices(ctx)
}
