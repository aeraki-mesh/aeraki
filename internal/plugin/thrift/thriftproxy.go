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

	"github.com/aeraki-mesh/aeraki/internal/model"
)

func buildOutboundProxy(context *model.EnvoyFilterContext) *thrift.ThriftProxy {
	route := buildOutboundRouteConfig(context)

	return newThriftProxy(context, route, model.TrafficDirectionOutbound)
}

func buildInboundProxy(context *model.EnvoyFilterContext) *thrift.ThriftProxy {
	route := buildInboundRouteConfig(context)
	return newThriftProxy(context, route, model.TrafficDirectionInbound)
}

func newThriftProxy(context *model.EnvoyFilterContext, route *thrift.RouteConfiguration,
	trafficDirection model.TrafficDirection) *thrift.ThriftProxy {
	return &thrift.ThriftProxy{
		StatPrefix: model.BuildClusterName(trafficDirection, "",
			context.ServiceEntry.Spec.Hosts[0], int(context.ServiceEntry.Spec.Ports[0].Number)),
		Transport:   thrift.TransportType_AUTO_TRANSPORT,
		Protocol:    thrift.ProtocolType_AUTO_PROTOCOL,
		RouteConfig: route,
		ThriftFilters: []*thrift.ThriftFilter{
			{
				Name: "envoy.filters.thrift.router",
			},
		},
	}
}
