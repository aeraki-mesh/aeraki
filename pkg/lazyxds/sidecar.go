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
	"reflect"

	networking "istio.io/api/networking/v1alpha3"
	istio "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioconfig "istio.io/istio/pkg/config"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) syncSidecars(service *istioconfig.Config) error {
	sidecarName := ResourcePrefix + service.Name

	sidecar, err := c.istioClient.NetworkingV1alpha3().Sidecars(service.Namespace).
		Get(context.TODO(), sidecarName, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		sidecar = &istio.Sidecar{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sidecarName,
				Namespace: service.Namespace,
				Labels: map[string]string{
					ManagedByLabel: LazyXdsManager,
				},
			},
		}
	} else if err != nil {
		return err
	}

	newSpec := networking.Sidecar{
		//TODO 只设置需要启用lazyxds的service/ns
		/*WorkloadSelector: &networking.WorkloadSelector{
			Labels: lazySvc.Spec.LazySelector,
		},*/
		Egress: []*networking.IstioEgressListener{
			{
				Hosts: c.egressListOfLazySource(service),
			},
		},
	}

	if sidecar.ResourceVersion == "" {
		controllerLog.Info("create sidecar")
		sidecar.Spec = newSpec
		_, err := c.istioClient.NetworkingV1alpha3().Sidecars(service.Namespace).
			Create(context.TODO(), sidecar, metav1.CreateOptions{
				FieldManager: LazyXdsManager,
			})
		return err
	}

	if reflect.DeepEqual(sidecar.Spec, newSpec) {
		return nil
	}

	sidecar.Spec = newSpec
	_, err = c.istioClient.NetworkingV1alpha3().Sidecars(service.Namespace).
		Update(context.TODO(), sidecar, metav1.UpdateOptions{
			FieldManager: LazyXdsManager,
		})
	return err
}

func (c *Controller) egressListOfLazySource(service *istioconfig.Config) []string {
	var list []string

	//TODO add egress services extracted from access log
	list = append([]string{"istio-system/*"}, list...)
	return list
}
