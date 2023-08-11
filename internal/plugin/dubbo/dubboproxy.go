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

package dubbo

import (
	dubbov1alpha1 "github.com/aeraki-mesh/client-go/pkg/clientset/versioned/typed/dubbo/v1alpha1"
	dubbo "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/dubbo_proxy/v3"
	"istio.io/istio/pilot/pkg/security/trustdomain"
	"istio.io/istio/pkg/spiffe"

	"github.com/aeraki-mesh/aeraki/internal/model"
	"github.com/aeraki-mesh/aeraki/internal/plugin/dubbo/authz/builder"
)

func buildOutboundProxy(context *model.EnvoyFilterContext) *dubbo.DubboProxy {
	route, err := buildOutboundRouteConfig(context)
	if err != nil {
		generatorLog.Errorf("Failed to generate Dubbo EnvoyFilter: %v, %v", context.ServiceEntry, err)
		return nil
	}

	return &dubbo.DubboProxy{
		StatPrefix: model.BuildClusterName(model.TrafficDirectionOutbound, "",
			context.ServiceEntry.Spec.Hosts[0], int(context.ServiceEntry.Spec.Ports[0].Number)),
		ProtocolType:      dubbo.ProtocolType_Dubbo,
		SerializationType: dubbo.SerializationType_Hessian2,
		// we only support one to one mapping of interface and service. If there're multiple interfaces in one process,
		// these interfaces can be defined in separated services, one service for one interface.
		RouteConfig: []*dubbo.RouteConfiguration{
			route,
		},
		DubboFilters: []*dubbo.DubboFilter{
			{
				Name: "envoy.filters.dubbo.router",
			},
		},
	}
}

func buildInboundProxy(context *model.EnvoyFilterContext,
	client dubbov1alpha1.DubboV1alpha1Interface) *dubbo.DubboProxy {
	route := buildInboundRouteConfig(context)

	// Todo support Domain alias
	tdBundle := trustdomain.NewBundle(spiffe.GetTrustDomain(), []string{})
	builder := builder.New(tdBundle, context.ServiceEntry.Namespace, client)
	dubboFilters := builder.BuildDubboFilter()
	dubboFilters = append(dubboFilters, &dubbo.DubboFilter{
		Name: "envoy.filters.dubbo.router",
	})

	return &dubbo.DubboProxy{
		StatPrefix: model.BuildClusterName(model.TrafficDirectionInbound, "",
			context.ServiceEntry.Spec.Hosts[0], int(context.ServiceEntry.Spec.Ports[0].Number)),
		ProtocolType:      dubbo.ProtocolType_Dubbo,
		SerializationType: dubbo.SerializationType_Hessian2,
		// we only support one to one mapping of interface and service. If there're multiple interfaces in one process,
		// these interfaces can be defined in separated services, one service for one interface.
		RouteConfig: []*dubbo.RouteConfiguration{
			route,
		},
		DubboFilters: dubboFilters,
	}
}
