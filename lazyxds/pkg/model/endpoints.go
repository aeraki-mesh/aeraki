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

package model

import (
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils"
	corev1 "k8s.io/api/core/v1"
)

// Endpoints contains the k8s endpoints info which lazyxds just need
type Endpoints struct {
	Name      string
	Namespace string

	IPList []string
}

// NewEndpoints creates new Endpoints from k8s endpoints
func NewEndpoints(endpoints *corev1.Endpoints) *Endpoints {
	ep := &Endpoints{
		Name:      endpoints.Name,
		Namespace: endpoints.Namespace,
	}

	for _, subset := range endpoints.Subsets {
		for _, address := range subset.Addresses {
			ep.IPList = append(ep.IPList, address.IP)
		}
	}

	return ep
}

// ID use service fqdn as endpoints id
func (ep *Endpoints) ID() string {
	return utils.FQDN(ep.Name, ep.Namespace)
}
