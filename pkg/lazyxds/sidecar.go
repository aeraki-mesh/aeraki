// Copyright Istio Authors
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

package lazyxds

import (
	"context"

	networking "istio.io/api/networking/v1alpha3"
	istio "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *namespaceController) createDefaultSidecar(ctx context.Context, namespace string) (err error) {
	defaultSidecarName := ResourcePrefix + "default"
	sidecar, err := c.istioClient.NetworkingV1alpha3().Sidecars(namespace).
		Get(ctx, defaultSidecarName, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		sidecar = &istio.Sidecar{
			ObjectMeta: metav1.ObjectMeta{
				Name:      defaultSidecarName,
				Namespace: namespace,
				Labels: map[string]string{
					ManagedByLabel: LazyXdsManager,
				},
			},
			Spec: networking.Sidecar{
				Egress: []*networking.IstioEgressListener{
					{
						Hosts: []string{"istio-system/*"},
					},
				},
			},
		}
		_, err = c.istioClient.NetworkingV1alpha3().Sidecars(namespace).
			Create(context.TODO(), sidecar, metav1.CreateOptions{
				FieldManager: LazyXdsManager,
			})
		return err
	} else if err != nil {
		return err
	}
	return nil
}
