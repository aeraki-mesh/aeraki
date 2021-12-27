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
	metaprotocolmodel "github.com/aeraki-framework/aeraki/pkg/model/metaprotocol"
	metaprotocol "github.com/aeraki-framework/meta-protocol-control-plane-api/meta_protocol_proxy/v1alpha"
	envoyconfig "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
)

func buildOutboundProxy(context *model.EnvoyFilterContext) (*metaprotocol.MetaProtocolProxy, error) {
	applicationProtocol, err := metaprotocolmodel.GetApplicationProtocolFromPortName(context.ServiceEntry.Spec.Ports[0].
		Name)
	if err != nil {
		return nil, err
	}
	codec, err := metaprotocolmodel.GetApplicationProtocolCodec(applicationProtocol)
	if err != nil {
		return nil, err
	}
	return &metaprotocol.MetaProtocolProxy{
		StatPrefix: model.BuildClusterName(model.TrafficDirectionOutbound, "",
			context.ServiceEntry.Spec.Hosts[0], int(context.ServiceEntry.Spec.Ports[0].Number)),
		RouteSpecifier: &metaprotocol.MetaProtocolProxy_Rds{
			Rds: &metaprotocol.Rds{
				RouteConfigName: model.BuildMetaProtocolRouteName(context.ServiceEntry.Spec.Hosts[0],
					int(context.ServiceEntry.Spec.Ports[0].Number)),
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
											ClusterName: "aeraki-xds", //TODO make this configurable
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
		MetaProtocolFilters: buildOutboundFilters(context.MetaRouter),
	}, nil
}

func buildInboundProxy(context *model.EnvoyFilterContext) (*metaprotocol.MetaProtocolProxy, error) {
	route, err := buildInboundRouteConfig(context)
	if err != nil {
		return nil, err
	}
	applicationProtocol, err := metaprotocolmodel.GetApplicationProtocolFromPortName(context.ServiceEntry.Spec.Ports[0].
		Name)
	if err != nil {
		return nil, err
	}
	codec, err := metaprotocolmodel.GetApplicationProtocolCodec(applicationProtocol)
	if err != nil {
		return nil, err
	}
	return &metaprotocol.MetaProtocolProxy{
		StatPrefix: model.BuildClusterName(model.TrafficDirectionInbound, "",
			context.ServiceEntry.Spec.Hosts[0], int(context.ServiceEntry.Spec.Ports[0].Number)),
		RouteSpecifier: &metaprotocol.MetaProtocolProxy_RouteConfig{
			RouteConfig: route,
		},
		ApplicationProtocol: applicationProtocol,
		Codec: &metaprotocol.Codec{
			Name: codec,
		},
		MetaProtocolFilters: buildInboundFilters(context.MetaRouter),
	}, nil
}
