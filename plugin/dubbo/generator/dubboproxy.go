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

package generator

import (
	dubbo "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/dubbo_proxy/v3"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config/host"
)

var (
	// TODO: In the current version of Envoy, MaxProgramSize has been deprecated. However even if we do not send
	// MaxProgramSize, Envoy is enforcing max size of 100 via runtime.
	// See https://www.envoyproxy.io/docs/envoy/latest/api-v3/type/matcher/v3/regex.proto.html#type-matcher-v3-regexmatcher-googlere2.
	regexEngine = &matcher.RegexMatcher_GoogleRe2{GoogleRe2: &matcher.RegexMatcher_GoogleRE2{}}
)

func buildProxy(hostname host.Name, port int) *dubbo.DubboProxy {
	clusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", hostname, port)

	//clusterName := model.BuildSubsetKey(model.TrafficDirectionInbound, hostname, port)

	return &dubbo.DubboProxy{
		StatPrefix:        clusterName,
		ProtocolType:      dubbo.ProtocolType_Dubbo,
		SerializationType: dubbo.SerializationType_Hessian2,
		RouteConfig: []*dubbo.RouteConfiguration{
			buildRouteConfig(clusterName, string(hostname)),
		},
	}
}

func buildRouteConfig(clusterName string, interfaceName string) *dubbo.RouteConfiguration {
	return &dubbo.RouteConfiguration{
		Name:      clusterName,
		Interface: interfaceName, // To make this work, Dubbo Interface should have been registered to the Istio service registry as a service
		Routes: []*dubbo.Route{
			defaultRoute(clusterName),
		},
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
