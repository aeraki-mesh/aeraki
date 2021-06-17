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

func (c *AggregationController) syncEndpoints(ctx context.Context, endpoints *corev1.Endpoints) error {
	ep := model.NewEndpoints(endpoints)
	c.endpoints.Store(ep.ID(), ep)

	return nil
}

func (c *AggregationController) deleteEndpoints(ctx context.Context, name, ns string) error {
	id := utils.FQDN(name, ns)
	c.endpoints.Delete(id)
	return nil
}

// IP2ServiceID ...
func (c *AggregationController) IP2ServiceID(targetIP string) string {
	var svcID string
	c.endpoints.Range(func(key, value interface{}) bool {
		ep := value.(*model.Endpoints)
		for _, ip := range ep.IPList {
			if targetIP == ip {
				svcID = ep.ID()
				return false
			}
		}
		return true
	})

	return svcID
}
