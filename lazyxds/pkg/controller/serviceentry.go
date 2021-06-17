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
	istio "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"reflect"
)

func (c *AggregationController) syncServiceEntry(ctx context.Context, serviceEntry *istio.ServiceEntry) error {
	if serviceEntry.Namespace == config.IstioNamespace { // ignore istio system, it's always exported
		return nil
	}

	id := utils.ObjectID(serviceEntry.Name, serviceEntry.Namespace)

	var oldHosts, newHosts []string
	if value, ok := c.serviceEntries.Load(id); ok {
		oldHosts = value.([]string)
	}
	newHosts = append(newHosts, serviceEntry.Spec.Hosts...)

	if reflect.DeepEqual(oldHosts, newHosts) {
		return nil
	}

	c.serviceEntries.Store(id, newHosts)

	return c.reconcileAllLazyServices(ctx)
}

func (c *AggregationController) deleteServiceEntry(ctx context.Context, name, ns string) error {
	if ns == config.IstioNamespace { // ignore istio system, it's always exported
		return nil
	}

	id := utils.ObjectID(name, ns)

	if _, found := c.serviceEntries.LoadAndDelete(id); found {
		return c.reconcileAllLazyServices(ctx)
	}

	return nil
}
