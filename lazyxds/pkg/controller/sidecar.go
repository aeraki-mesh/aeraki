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
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/model"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils"
	istio "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

func (c *AggregationController) syncSidecar(ctx context.Context, sidecar *istio.Sidecar) (err error) {
	v, ok := c.namespaces.Load(sidecar.Namespace)
	if !ok {
		return fmt.Errorf("namespace of sidecar(%s/%s) is not exist", sidecar.Namespace, sidecar.Name)
	}
	ns := v.(*model.Namespace)
	ns.AddSidecar(sidecar.Name)

	return c.reconcileNamespace(ctx, ns)
}

func (c *AggregationController) deleteSidecar(ctx context.Context, id string) (err error) {
	name, namespace := utils.ParseID(id)

	v, ok := c.namespaces.Load(namespace)
	if !ok {
		return fmt.Errorf("namespace of sidecar(%s/%s) is not exist", namespace, name)
	}
	ns := v.(*model.Namespace)
	ns.DeleteSidecar(name)

	return c.reconcileNamespace(ctx, ns)
}
