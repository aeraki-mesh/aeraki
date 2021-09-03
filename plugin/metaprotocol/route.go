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
	"github.com/aeraki-framework/aeraki/pkg/model"
	metaroute "github.com/aeraki-framework/meta-protocol-control-plane-api/meta_protocol_proxy/config/route/v1alpha"
)

func buildInboundRouteConfig(context *model.EnvoyFilterContext) (*metaroute.RouteConfiguration, error) {
	clusterName := model.BuildClusterName(model.TrafficDirectionInbound, "", context.ServiceEntry.Spec.Hosts[0], int(context.ServiceEntry.Spec.Ports[0].Number))

	return &metaroute.RouteConfiguration{
		Name: clusterName,
		Routes: []*metaroute.Route{
			defaultRoute(clusterName),
		},
	}, nil
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
