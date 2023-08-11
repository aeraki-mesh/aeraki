// Copyright 2020 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package xds

import (
	"strconv"
	"time"

	httpcore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	httproute "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"

	userapi "github.com/aeraki-mesh/api/metaprotocol/v1alpha1"
	metaroute "github.com/aeraki-mesh/meta-protocol-control-plane-api/aeraki/meta_protocol_proxy/config/route/v1alpha"
)

var regexEngine = &matcher.RegexMatcher_GoogleRe2{GoogleRe2: &matcher.RegexMatcher_GoogleRE2{}}

// We use Envoy RDS(HTTP RouteConfiguration) to transmit Meta Protocol Configuration from the RDS server to the Proxy
func metaProtocolRoute2HttpRoute(metaRoute *metaroute.RouteConfiguration) *httproute.RouteConfiguration {
	httpRoute := &httproute.RouteConfiguration{
		Name: metaRoute.Name,
		VirtualHosts: []*httproute.VirtualHost{
			{
				Name:    "dummy",
				Domains: []string{"*"},
			},
		},
	}

	for _, route := range metaRoute.Routes {
		routeMatch := httpRouteMatch(route)
		routeAction := httpRouteAction(route)
		requestHeaders := httpRequestHeaders(route)
		responseHeaders := httpResponseHeaders(route)
		httpRoute.VirtualHosts[0].Routes = append(httpRoute.VirtualHosts[0].Routes, &httproute.Route{
			Name:  route.Name,
			Match: routeMatch,
			Action: &httproute.Route_Route{
				Route: routeAction,
			},
			RequestHeadersToAdd:  requestHeaders,
			ResponseHeadersToAdd: responseHeaders,
		})
	}
	return httpRoute
}

func httpResponseHeaders(route *metaroute.Route) []*httpcore.HeaderValueOption {
	var responseHeaders []*httpcore.HeaderValueOption
	for _, mutation := range route.ResponseMutation {
		responseHeaders = append(responseHeaders, &httpcore.HeaderValueOption{
			Header: &httpcore.HeaderValue{
				Key:   mutation.Key,
				Value: mutation.Value,
			},
		})
	}
	return responseHeaders
}

func httpRequestHeaders(route *metaroute.Route) []*httpcore.HeaderValueOption {
	var requestHeaders []*httpcore.HeaderValueOption
	for _, mutation := range route.RequestMutation {
		requestHeaders = append(requestHeaders, &httpcore.HeaderValueOption{
			Header: &httpcore.HeaderValue{
				Key:   mutation.Key,
				Value: mutation.Value,
			},
		})
	}
	return requestHeaders
}

func httpRouteAction(route *metaroute.Route) *httproute.RouteAction {
	var routeAction = &httproute.RouteAction{}
	if route.Route.GetWeightedClusters() != nil {
		routeAction.ClusterSpecifier = &httproute.RouteAction_WeightedClusters{
			WeightedClusters: route.Route.GetWeightedClusters(),
		}
	} else {
		routeAction.ClusterSpecifier = &httproute.RouteAction_Cluster{
			Cluster: route.Route.GetCluster(),
		}
	}
	routeAction.RequestMirrorPolicies = make([]*httproute.
		RouteAction_RequestMirrorPolicy, len(route.Route.RequestMirrorPolicies))
	for i, mirrorPolicy := range route.Route.RequestMirrorPolicies {
		routeAction.RequestMirrorPolicies[i] = &httproute.RouteAction_RequestMirrorPolicy{
			Cluster:         mirrorPolicy.Cluster,
			RuntimeFraction: mirrorPolicy.RuntimeFraction,
		}
	}
	if route.Route.HashPolicy != nil && len(route.Route.HashPolicy) > 0 {
		for _, hashPolicy := range route.Route.HashPolicy {
			routeAction.HashPolicy = append(routeAction.HashPolicy, &httproute.RouteAction_HashPolicy{
				PolicySpecifier: &httproute.RouteAction_HashPolicy_Header_{
					Header: &httproute.RouteAction_HashPolicy_Header{
						HeaderName: hashPolicy,
					},
				},
			})
		}
	}
	return routeAction
}

func httpRouteMatch(route *metaroute.Route) *httproute.RouteMatch {
	routeMatch := &httproute.RouteMatch{
		PathSpecifier: &httproute.RouteMatch_Prefix{
			Prefix: "/",
		},
	}
	for _, metadata := range route.Match.Metadata {
		routeMatch.Headers = append(routeMatch.Headers, &httproute.HeaderMatcher{
			Name:                 metadata.Name,
			HeaderMatchSpecifier: metadata.HeaderMatchSpecifier,
		})
	}
	return routeMatch
}

func generateSnapshot(metaRoutes []*metaroute.RouteConfiguration) (*cache.Snapshot, error) {
	var httpRoutes []types.Resource
	for _, route := range metaRoutes {
		httpRoutes = append(httpRoutes, metaProtocolRoute2HttpRoute(route))
	}
	return cache.NewSnapshot(
		strconv.FormatInt(time.Now().Unix(), 10),
		map[resource.Type][]types.Resource{
			resource.RouteType: httpRoutes,
		},
	)
}

// MetaMatch2HttpHeaderMatch converts MetaMatch to HttpHeaderMatch
func MetaMatch2HttpHeaderMatch(matchCrd *userapi.MetaRouteMatch) []*httproute.HeaderMatcher {
	var headerMatchers []*httproute.HeaderMatcher
	if matchCrd != nil {
		for name, attribute := range matchCrd.Attributes {
			headerMatcher := &httproute.HeaderMatcher{}
			headerMatcher.Name = name
			if isCatchAllHeaderMatch(attribute) {
				headerMatcher.HeaderMatchSpecifier = &httproute.HeaderMatcher_PresentMatch{PresentMatch: true}
			} else {
				switch attribute.GetMatchType().(type) {
				case *userapi.StringMatch_Exact:
					headerMatcher.HeaderMatchSpecifier = &httproute.HeaderMatcher_ExactMatch{
						ExactMatch: attribute.GetExact(),
					}
				case *userapi.StringMatch_Prefix:
					headerMatcher.HeaderMatchSpecifier = &httproute.HeaderMatcher_PrefixMatch{
						PrefixMatch: attribute.GetPrefix(),
					}
				case *userapi.StringMatch_Regex:
					headerMatcher.HeaderMatchSpecifier = &httproute.HeaderMatcher_SafeRegexMatch{
						SafeRegexMatch: &matcher.RegexMatcher{
							EngineType: regexEngine,
							Regex:      attribute.GetRegex(),
						},
					}
				default:
					continue
				}
			}
			headerMatchers = append(headerMatchers, headerMatcher)
		}
	}
	return headerMatchers
}

func isCatchAllHeaderMatch(in *userapi.StringMatch) bool {
	catchall := false

	if in == nil {
		return true
	}

	switch m := in.MatchType.(type) {
	case *userapi.StringMatch_Regex:
		catchall = m.Regex == "*"
	}

	return catchall
}

func translatePercentToFractionalPercent(p float64) *typev3.FractionalPercent {
	return &typev3.FractionalPercent{
		Numerator:   uint32(p * 10000),
		Denominator: typev3.FractionalPercent_MILLION,
	}
}
