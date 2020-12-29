package dubbo

import (
	"fmt"

	"github.com/aeraki-framework/aeraki/pkg/model"
	envoy "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	dubbo "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/dubbo_proxy/v3"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	networking "istio.io/api/networking/v1alpha3"
)

var (
	// TODO: In the current version of Envoy, MaxProgramSize has been deprecated. However even if we do not send
	// MaxProgramSize, Envoy is enforcing max size of 100 via runtime.
	// See https://www.envoyproxy.io/docs/envoy/latest/api-v3/type/matcher/v3/regex.proto.html#type-matcher-v3-regexmatcher-googlere2.
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
	clusterName := model.BuildClusterName(model.TrafficDirectionOutbound, "", context.ServiceEntry.Spec.Hosts[0], int(context.ServiceEntry.Spec.Ports[0].Number))

	if context.VirtualService == nil {
		route = []*dubbo.Route{defaultRoute(clusterName)}
	} else {
		route = buildRoute(context)
	}

	return &dubbo.RouteConfiguration{
		Name:      clusterName,
		Interface: serviceInterface, // To make this work, Dubbo Interface should have been registered to the Istio service registry as a service
		Routes:    route,
	}, nil
}

func buildInboundRouteConfig(context *model.EnvoyFilterContext) (*dubbo.RouteConfiguration, error) {
	// dubbo service interface should be passed in via serviceentry annotation
	var serviceInterface string
	var exist bool
	if serviceInterface, exist = context.ServiceEntry.Annotations["interface"]; !exist {
		err := fmt.Errorf("no interface annotation")
		return nil, err
	}
	clusterName := model.BuildClusterName(model.TrafficDirectionInbound, "", "", int(context.ServiceEntry.Spec.Ports[0].Number))
	route := []*dubbo.Route{defaultRoute(clusterName)}
	return &dubbo.RouteConfiguration{
		Name:      clusterName,
		Interface: serviceInterface, // To make this work, Dubbo Interface should have been registered to the Istio service registry as a service
		Routes:    route,
	}, nil
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

		routes = append(routes, &dubbo.Route{
			//todo: convert virtual service HTTP Route Match to Dubbo Route Match
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
			Route: routeAction,
		})
	}
	return routes
}

func buildSingleCluster(http *networking.HTTPRoute, service *networking.ServiceEntry) *dubbo.RouteAction {
	clusterName := model.BuildClusterName(model.TrafficDirectionOutbound, http.Route[0].Destination.Subset, service.Hosts[0], int(service.Ports[0].Number))
	return &dubbo.RouteAction{
		ClusterSpecifier: &dubbo.RouteAction_Cluster{
			Cluster: clusterName,
		},
	}
}

func buildWeightedCluster(http *networking.HTTPRoute, service *networking.ServiceEntry) *dubbo.RouteAction {
	var clusterWeights []*envoy.WeightedCluster_ClusterWeight
	var totalWeight uint32

	for _, route := range http.Route {
		clusterName := model.BuildClusterName(model.TrafficDirectionOutbound, route.Destination.Subset, service.Hosts[0], int(service.Ports[0].Number))
		clusterWeight := &envoy.WeightedCluster_ClusterWeight{
			Name:   clusterName,
			Weight: &wrappers.UInt32Value{Value: uint32(route.Weight)},
		}
		clusterWeights = append(clusterWeights, clusterWeight)
		totalWeight += uint32(route.Weight)
	}

	return &dubbo.RouteAction{
		ClusterSpecifier: &dubbo.RouteAction_WeightedClusters{
			WeightedClusters: &envoy.WeightedCluster{
				Clusters:    clusterWeights,
				TotalWeight: &wrappers.UInt32Value{Value: totalWeight},
			},
		},
	}
}
