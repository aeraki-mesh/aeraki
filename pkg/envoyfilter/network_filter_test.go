package envoyfilter

import (
	"reflect"
	"testing"

	istioconfig "istio.io/istio/pkg/config"

	"github.com/aeraki-framework/aeraki/pkg/model"
	networking "istio.io/api/networking/v1alpha3"
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inboudEnvoyFilterWorkloadSelector(tt.service); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("inboudEnvoyFilterWorkloadSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}
