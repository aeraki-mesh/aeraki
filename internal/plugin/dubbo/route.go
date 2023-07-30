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
	"fmt"

	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	dubbo "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/dubbo_proxy/v3"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	networking "istio.io/api/networking/v1alpha3"

	"github.com/aeraki-mesh/aeraki/internal/model"
)

var (
	// TODO: In the current version of Envoy, MaxProgramSize has been deprecated. However even if we do not send
	// MaxProgramSize, Envoy is enforcing max size of 100 via runtime.
	// See https://www.envoyproxy.io/docs/envoy/latest/api-v3/type/matcher/v3/regex.proto.html
	// #type-matcher-v3-regexmatcher-googlere2.
	regexEngine = &matcher.RegexMatcher_GoogleRe2{GoogleRe2: &matcher.RegexMatcher_GoogleRE2{}}
)

func buildOutboundRouteConfig(context *model.EnvoyFilterContext) (*dubbo.RouteConfiguration, error) {
	// dubbo service interface should be passed in via serviceentry annotation
	var serviceInterface string
	var exist bool
	if serviceInterface, exist = context.ServiceEntry.Annotations["interface"]; !exist {
		err := fmt.Errorf("no interface annotation")
		return nil, err
	}

	var route []*dubbo.Route
	clusterName := model.BuildClusterName(model.TrafficDirectionOutbound, "",
		context.ServiceEntry.Spec.Hosts[0], int(context.ServiceEntry.Spec.Ports[0].Number))

	if context.VirtualService == nil {
		route = []*dubbo.Route{defaultRoute(clusterName)}
	} else {
		route = buildRoute(context)
	}

	return &dubbo.RouteConfiguration{
		Name: clusterName,
		// To make this work, Dubbo Interface should have been registered to the Istio service registry as a service
		Interface: serviceInterface,
		Routes:    route,
	}, nil
}

func buildInboundRouteConfig(context *model.EnvoyFilterContext) *dubbo.RouteConfiguration {
	clusterName := model.BuildClusterName(model.TrafficDirectionInbound, "", "",
		int(context.ServiceEntry.Spec.Ports[0].Number))
	route := []*dubbo.Route{defaultRoute(clusterName)}
	return &dubbo.RouteConfiguration{
		Name:      clusterName,
		Interface: "*", // Use wildcard to catch all the dubbo interfaces at this inbound port
		Routes:    route,
	}
}

func defaultRoute(clusterName string) *dubbo.Route {
	return &dubbo.Route{
		Match: &dubbo.RouteMatch{
			Method: &dubbo.MethodMatch{
				Name: &matcher.StringMatcher{
					MatchPattern: &matcher.StringMatcher_SafeRegex{
						SafeRegex: &matcher.RegexMatcher{
							EngineType: regexEngine,
							Regex:      ".*",
						},
					},
				},
			},
		},
		Route: &dubbo.RouteAction{
			ClusterSpecifier: &dubbo.RouteAction_Cluster{
				Cluster: clusterName,
			},
		},
	}
}

func buildRoute(context *model.EnvoyFilterContext) []*dubbo.Route {
	service := context.ServiceEntry.Spec
	vs := context.VirtualService.Spec

	routes := make([]*dubbo.Route, 0)
	for _, http := range vs.Http {
		var routeAction *dubbo.RouteAction

		if len(http.Route) > 1 {
			routeAction = buildWeightedCluster(http, service)
		} else {
			routeAction = buildSingleCluster(http, service)
		}

		dubboRoute := &dubbo.Route{
			Match: &dubbo.RouteMatch{
				Method:  buildMethodMatch(http),
				Headers: buildHeaderMatch(http),
			},
			Route: routeAction,
		}
		routes = append(routes, dubboRoute)
	}
	return routes
}

func buildMethodMatch(route *networking.HTTPRoute) *dubbo.MethodMatch {
	var methodName *matcher.StringMatcher
	if len(route.Match) > 0 {
		method := route.Match[0].Method
		if method != nil {
			switch method.MatchType.(type) {
			case *networking.StringMatch_Exact:
				methodName = &matcher.StringMatcher{
					MatchPattern: &matcher.StringMatcher_Exact{
						Exact: method.GetExact(),
					},
				}
			case *networking.StringMatch_Prefix:
				methodName = &matcher.StringMatcher{
					MatchPattern: &matcher.StringMatcher_Prefix{
						Prefix: method.GetPrefix(),
					},
				}
			case *networking.StringMatch_Regex:
				methodName = &matcher.StringMatcher{
					MatchPattern: &matcher.StringMatcher_SafeRegex{
						SafeRegex: &matcher.RegexMatcher{
							EngineType: regexEngine,
							Regex:      method.GetRegex()},
					},
				}
			}
		}
	}

	// Set a default match-all MethodMatch, otherwise dubbo proxy will complain
	if methodName == nil {
		methodName = &matcher.StringMatcher{
			MatchPattern: &matcher.StringMatcher_SafeRegex{
				SafeRegex: &matcher.RegexMatcher{
					EngineType: regexEngine,
					Regex:      ".*",
				},
			},
		}
	}

	return &dubbo.MethodMatch{
		Name: methodName,
	}
}

func buildHeaderMatch(route *networking.HTTPRoute) []*routepb.HeaderMatcher {
	headerMatchers := make([]*routepb.HeaderMatcher, 0)
	if len(route.Match) > 0 {
		for name, value := range route.Match[0].Headers {
			switch value.MatchType.(type) {
			case *networking.StringMatch_Exact:
				headerMatchers = append(headerMatchers, &routepb.HeaderMatcher{
					Name: name,
					HeaderMatchSpecifier: &routepb.HeaderMatcher_ExactMatch{
						ExactMatch: value.GetExact(),
					},
				})
			case *networking.StringMatch_Prefix:
				headerMatchers = append(headerMatchers, &routepb.HeaderMatcher{
					Name: name,
					HeaderMatchSpecifier: &routepb.HeaderMatcher_PrefixMatch{
						PrefixMatch: value.GetPrefix(),
					},
				})
			case *networking.StringMatch_Regex:
				headerMatchers = append(headerMatchers, &routepb.HeaderMatcher{
					Name: name,
					HeaderMatchSpecifier: &routepb.HeaderMatcher_SafeRegexMatch{
						SafeRegexMatch: &matcher.RegexMatcher{
							EngineType: regexEngine,
							Regex:      value.GetRegex()},
					},
				})
			}
		}
	}
	return headerMatchers
}

func buildSingleCluster(http *networking.HTTPRoute, service *networking.ServiceEntry) *dubbo.RouteAction {
	clusterName := model.BuildClusterName(model.TrafficDirectionOutbound, http.Route[0].Destination.Subset,
		service.Hosts[0], int(service.Ports[0].Number))
	return &dubbo.RouteAction{
		ClusterSpecifier: &dubbo.RouteAction_Cluster{
			Cluster: clusterName,
		},
	}
}

func buildWeightedCluster(http *networking.HTTPRoute, service *networking.ServiceEntry) *dubbo.RouteAction {
	var clusterWeights []*routepb.WeightedCluster_ClusterWeight
	var totalWeight uint32

	for _, route := range http.Route {
		clusterName := model.BuildClusterName(model.TrafficDirectionOutbound, route.Destination.Subset,
			service.Hosts[0], int(service.Ports[0].Number))
		clusterWeight := &routepb.WeightedCluster_ClusterWeight{
			Name:   clusterName,
			Weight: &wrappers.UInt32Value{Value: uint32(route.Weight)},
		}
		clusterWeights = append(clusterWeights, clusterWeight)
		totalWeight += uint32(route.Weight)
	}

	return &dubbo.RouteAction{
		ClusterSpecifier: &dubbo.RouteAction_WeightedClusters{
			WeightedClusters: &routepb.WeightedCluster{
				Clusters:    clusterWeights,
				TotalWeight: &wrappers.UInt32Value{Value: totalWeight},
			},
		},
	}
}
