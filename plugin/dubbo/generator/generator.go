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
	"bytes"
	"strconv"

	"github.com/aeraki-framework/aeraki/pkg/model"

	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/types"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/pkg/log"
)

var generatorLog = log.RegisterScope("dubbo-generator", "mcp debugging", 0)

type Generator struct {
}

func NewGenerator() *Generator {
	return &Generator{}
}

func (*Generator) Generate(context *model.EnvoyFilterContext) *networking.EnvoyFilter {

	serviceEntry := context.ServiceEntry

	service := serviceEntry.Spec
	dubboProxy := buildProxy(context)

	buf := &bytes.Buffer{}
	_ = (&jsonpb.Marshaler{OrigName: true}).Marshal(buf, dubboProxy)
	var out = &types.Struct{}
	if err := (&jsonpb.Unmarshaler{AllowUnknownFields: false}).Unmarshal(buf, out); err != nil {
		//This should not happen
		generatorLog.Errorf("Failed to generate Dubbo EnvoyFilter: %v", err)
		return nil
	}
	out.Fields["@type"] = &types.Value{Kind: &types.Value_StringValue{
		StringValue: "type.googleapis.com/envoy.extensions.filters.network.dubbo_proxy.v3.DubboProxy",
	}}

	Value := &types.Struct{
		Fields: map[string]*types.Value{
			"name": {
				Kind: &types.Value_StringValue{
					StringValue: "envoy.filters.network.dubbo_proxy",
				},
			},
			"typed_config": {
				Kind: &types.Value_StructValue{StructValue: out},
			},
		},
	}

	listenerName := service.GetAddresses()[0] + "_" + strconv.Itoa(int(service.Ports[0].Number))

	return &networking.EnvoyFilter{
		ConfigPatches: []*networking.EnvoyFilter_EnvoyConfigObjectPatch{
			&networking.EnvoyFilter_EnvoyConfigObjectPatch{
				ApplyTo: networking.EnvoyFilter_NETWORK_FILTER,
				Match: &networking.EnvoyFilter_EnvoyConfigObjectMatch{
					ObjectTypes: &networking.EnvoyFilter_EnvoyConfigObjectMatch_Listener{
						Listener: &networking.EnvoyFilter_ListenerMatch{
							Name: listenerName,
							FilterChain: &networking.EnvoyFilter_ListenerMatch_FilterChainMatch{
								Filter: &networking.EnvoyFilter_ListenerMatch_FilterMatch{
									Name: wellknown.TCPProxy,
								},
							},
						},
					},
				},
				Patch: &networking.EnvoyFilter_Patch{
					Operation: networking.EnvoyFilter_Patch_REPLACE,
					Value:     Value,
				},
			},
		},
	}
}
