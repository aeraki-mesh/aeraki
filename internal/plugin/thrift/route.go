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

package thrift

import (
	thrift "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/thrift_proxy/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	networking "istio.io/api/networking/v1alpha3"

	"github.com/aeraki-mesh/aeraki/internal/model"
)

func buildOutboundRouteConfig(context *model.EnvoyFilterContext) *thrift.RouteConfiguration {
	var route []*thrift.Route
	clusterName := model.BuildClusterName(model.TrafficDirectionOutbound, "",
		context.ServiceEntry.Spec.Hosts[0], int(context.ServiceEntry.Spec.Ports[0].Number))

	if context.VirtualService == nil {
		route = []*thrift.Route{defaultRoute(clusterName)}
	} else {
		route = buildRoute(context)
	}

	return &thrift.RouteConfiguration{
		Name:   clusterName,
		Routes: route,
	}
}

func buildInboundRouteConfig(context *model.EnvoyFilterContext) *thrift.RouteConfiguration {
	clusterName := model.BuildClusterName(model.TrafficDirectionInbound, "",
		context.ServiceEntry.Spec.Hosts[0], int(context.ServiceEntry.Spec.Ports[0].Number))

	return &thrift.RouteConfiguration{
		Name: clusterName,
		Routes: []*thrift.Route{
			defaultRoute(clusterName),
		},
	}
}

func defaultRoute(clusterName string) *thrift.Route {
	return &thrift.Route{
		Match: &thrift.RouteMatch{
			MatchSpecifier: &thrift.RouteMatch_MethodName{
				MethodName: "", // empty string matches any request method name
			},
		},
		Route: &thrift.RouteAction{
			ClusterSpecifier: &thrift.RouteAction_Cluster{
				Cluster: clusterName,
			},
		},
	}
}

func buildRoute(context *model.EnvoyFilterContext) []*thrift.Route {
	service := context.ServiceEntry.Spec
	vs := context.VirtualService.Spec

	routes := make([]*thrift.Route, 0)
	for _, http := range vs.Http {
		var routeAction *thrift.RouteAction

		if len(http.Route) > 1 {
			routeAction = buildWeightedCluster(http, service)
		} else {
			routeAction = buildSingleCluster(http, service)
		}

		routes = append(routes, &thrift.Route{
			//todo: convert virtual service HTTP Route Match to Dubbo Route Match
			Match: &thrift.RouteMatch{
				MatchSpecifier: &thrift.RouteMatch_MethodName{
					MethodName: "", // empty string matches any request method name
				},
			},
			Route: routeAction,
		})
	}
	return routes
}

func buildSingleCluster(http *networking.HTTPRoute, service *networking.ServiceEntry) *thrift.RouteAction {
	clusterName := model.BuildClusterName(model.TrafficDirectionOutbound, http.Route[0].Destination.Subset,
		service.Hosts[0], int(service.Ports[0].Number))
	return &thrift.RouteAction{
		ClusterSpecifier: &thrift.RouteAction_Cluster{
			Cluster: clusterName,
		},
	}
}

func buildWeightedCluster(http *networking.HTTPRoute, service *networking.ServiceEntry) *thrift.RouteAction {
	var clusterWeights []*thrift.WeightedCluster_ClusterWeight
	var totalWeight uint32

	for _, route := range http.Route {
		clusterName := model.BuildClusterName(model.TrafficDirectionOutbound, route.Destination.Subset,
			service.Hosts[0], int(service.Ports[0].Number))
		clusterWeight := &thrift.WeightedCluster_ClusterWeight{
			Name:   clusterName,
			Weight: &wrappers.UInt32Value{Value: uint32(route.Weight)},
		}
		clusterWeights = append(clusterWeights, clusterWeight)
		totalWeight += uint32(route.Weight)
	}

	return &thrift.RouteAction{
		ClusterSpecifier: &thrift.RouteAction_WeightedClusters{
			WeightedClusters: &thrift.WeightedCluster{
				Clusters: clusterWeights,
			},
		},
	}
}
