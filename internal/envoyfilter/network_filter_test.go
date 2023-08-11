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

package envoyfilter

import (
	"reflect"
	"testing"

	networking "istio.io/api/networking/v1alpha3"
	istioconfig "istio.io/istio/pkg/config"

	"github.com/aeraki-mesh/aeraki/internal/model"
)

func Test_inboudEnvoyFilterWorkloadSelector(t *testing.T) {
	tests := []struct {
		name    string
		service *model.ServiceEntryWrapper
		want    *networking.WorkloadSelector
	}{
		{
			name: "test1",
			service: &model.ServiceEntryWrapper{
				Spec: &networking.ServiceEntry{
					WorkloadSelector: &networking.WorkloadSelector{
						Labels: map[string]string{
							"app": "dubbo-sample-provider",
						},
					},
				},
			},
			want: &networking.WorkloadSelector{
				Labels: map[string]string{
					"app": "dubbo-sample-provider",
				},
			},
		},
		{
			name: "test2",
			service: &model.ServiceEntryWrapper{
				Meta: istioconfig.Meta{
					Annotations: map[string]string{
						"workloadSelector": "dubbo-sample-provider",
					},
				},
				Spec: &networking.ServiceEntry{},
			},
			want: &networking.WorkloadSelector{
				Labels: map[string]string{
					"app": "dubbo-sample-provider",
				},
			},
		},
		{
			name: "test3",
			service: &model.ServiceEntryWrapper{
				Meta: istioconfig.Meta{
					Annotations: map[string]string{
						"workloadSelector": "app.io: dubbo-sample-provider",
					},
				},
				Spec: &networking.ServiceEntry{},
			},
			want: &networking.WorkloadSelector{
				Labels: map[string]string{
					"app.io": "dubbo-sample-provider",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inboundEnvoyFilterWorkloadSelector(tt.service); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("inboudEnvoyFilterWorkloadSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}
