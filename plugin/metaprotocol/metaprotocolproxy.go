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
	"fmt"

	envoyconfig "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoytype "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	istionetworking "istio.io/api/networking/v1alpha3"

	metaprotocol "github.com/aeraki-mesh/meta-protocol-control-plane-api/meta_protocol_proxy/v1alpha"

	"github.com/aeraki-mesh/aeraki/pkg/model"
	metaprotocolmodel "github.com/aeraki-mesh/aeraki/pkg/model/metaprotocol"
)

func buildOutboundProxy(context *model.EnvoyFilterContext,
	port *istionetworking.Port) (*metaprotocol.MetaProtocolProxy, error) {
	applicationProtocol, err := metaprotocolmodel.GetApplicationProtocolFromPortName(port.Name)
	if err != nil {
		return nil, err
	}
	codec, err := metaprotocolmodel.GetApplicationProtocolCodec(applicationProtocol)
	if err != nil {
		return nil, err
	}
	metaProtocolProy := &metaprotocol.MetaProtocolProxy{
		StatPrefix: model.BuildClusterName(model.TrafficDirectionOutbound, "",
			context.ServiceEntry.Spec.Hosts[0], int(port.Number)),
		RouteSpecifier: &metaprotocol.MetaProtocolProxy_Rds{
			Rds: &metaprotocol.Rds{
				RouteConfigName: model.BuildMetaProtocolRouteName(context.ServiceEntry.Spec.Hosts[0],
					int(port.Number)),
				ConfigSource: &envoyconfig.ConfigSource{
					ResourceApiVersion: envoyconfig.ApiVersion_V3,
					ConfigSourceSpecifier: &envoyconfig.ConfigSource_ApiConfigSource{
						ApiConfigSource: &envoyconfig.ApiConfigSource{
							ApiType:             envoyconfig.ApiConfigSource_GRPC,
							TransportApiVersion: envoyconfig.ApiVersion_V3,
							GrpcServices: []*envoyconfig.GrpcService{
								{
									TargetSpecifier: &envoyconfig.GrpcService_EnvoyGrpc_{
										EnvoyGrpc: &envoyconfig.GrpcService_EnvoyGrpc{
											ClusterName: "aeraki-xds", // TODO make this configurable
										},
									},
								},
							},
						},
					},
				},
			},
		},
		ApplicationProtocol: applicationProtocol,
		Codec: &metaprotocol.Codec{
			Name: codec,
		},
		MetaProtocolFilters: buildOutboundFilters(),
	}
	configTracing(context, metaProtocolProy)
	return metaProtocolProy, nil
}

func buildInboundProxy(context *model.EnvoyFilterContext,
	port *istionetworking.Port) (*metaprotocol.MetaProtocolProxy, error) {
	route := buildInboundRouteConfig(context, port)
	applicationProtocol, err := metaprotocolmodel.GetApplicationProtocolFromPortName(port.
		Name)
	if err != nil {
		return nil, err
	}
	codec, err := metaprotocolmodel.GetApplicationProtocolCodec(applicationProtocol)
	if err != nil {
		return nil, err
	}

	filters, err := buildInboundFilters(context.MetaRouter)
	if err != nil {
		return nil, err
	}

	metaProtocolProy := &metaprotocol.MetaProtocolProxy{
		StatPrefix: model.BuildClusterName(model.TrafficDirectionInbound, "",
			context.ServiceEntry.Spec.Hosts[0], int(port.Number)),
		RouteSpecifier: &metaprotocol.MetaProtocolProxy_RouteConfig{
			RouteConfig: route,
		},
		ApplicationProtocol: applicationProtocol,
		Codec: &metaprotocol.Codec{
			Name: codec,
		},
		MetaProtocolFilters: filters,
	}
	configTracing(context, metaProtocolProy)
	return metaProtocolProy, nil
}

func configTracing(context *model.EnvoyFilterContext, metaProtocolProy *metaprotocol.MetaProtocolProxy) {
	if context.MeshConfig.Mesh().EnableTracing {
		randomSampling := 1.0
		fmt.Printf("%v", context.MeshConfig.Mesh().DefaultConfig.Tracing)
		if context.MeshConfig.Mesh().DefaultConfig.Tracing != nil {
			randomSampling = context.MeshConfig.Mesh().DefaultConfig.Tracing.Sampling
		}
		metaProtocolProy.Tracing = &metaprotocol.Tracing{
			ClientSampling: &envoytype.Percent{
				Value: 100,
			},
			RandomSampling: &envoytype.Percent{
				Value: randomSampling,
			},
			OverallSampling: &envoytype.Percent{
				Value: 100,
			},
		}
	}
}
