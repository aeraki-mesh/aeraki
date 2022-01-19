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
	"github.com/aeraki-mesh/aeraki/pkg/envoyfilter"
	"github.com/aeraki-mesh/aeraki/pkg/model"
	"istio.io/pkg/log"
)

var generatorLog = log.RegisterScope("metaprotocol-generator", "metaprotocol generator", 0)

// Generator defines a MetaProtocol envoyfilter Generator
type Generator struct {
}

// NewGenerator creates an new MetaProtocol Generator instance
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate create EnvoyFilters for MetaProtocol services
func (*Generator) Generate(context *model.EnvoyFilterContext) ([]*model.EnvoyFilterWrapper, error) {
	outboundProxy, err := buildOutboundProxy(context)
	if err != nil {
		return nil, err
	}
	inboundProxy, err := buildInboundProxy(context)
	if err != nil {
		return nil, err
	}
	return envoyfilter.GenerateReplaceNetworkFilter(
		context.ServiceEntry,
		outboundProxy,
		inboundProxy,
		"envoy.filters.network.meta_protocol_proxy",
		"type.googleapis.com/aeraki.meta_protocol_proxy.v1alpha.MetaProtocolProxy"), nil
}
