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

package metaprotocol

import (
	metaroute "github.com/aeraki-mesh/meta-protocol-control-plane-api/aeraki/meta_protocol_proxy/config/route/v1alpha"
	istionetworking "istio.io/api/networking/v1alpha3"

	"github.com/aeraki-mesh/aeraki/internal/model"
)

func buildInboundRouteConfig(context *model.EnvoyFilterContext,
	port *istionetworking.ServicePort) *metaroute.RouteConfiguration {
	clusterName := model.BuildClusterName(model.TrafficDirectionInbound, "",
		context.ServiceEntry.Spec.Hosts[0], int(port.Number))

	return &metaroute.RouteConfiguration{
		Name: clusterName,
		Routes: []*metaroute.Route{
			defaultRoute(clusterName),
		},
	}
}

func defaultRoute(clusterName string) *metaroute.Route {
	return &metaroute.Route{
		Route: &metaroute.RouteAction{
			ClusterSpecifier: &metaroute.RouteAction_Cluster{
				Cluster: clusterName,
			},
		},
	}
}
