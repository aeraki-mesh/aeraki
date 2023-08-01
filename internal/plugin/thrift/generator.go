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
	"github.com/aeraki-mesh/aeraki/internal/envoyfilter"
	"github.com/aeraki-mesh/aeraki/internal/model"
)

// Generator defines a Thrift envoyfilter Generator
type Generator struct {
}

// NewGenerator creates an new Thrift Generator instance
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate create EnvoyFilters for Thrift services
func (*Generator) Generate(context *model.EnvoyFilterContext) ([]*model.EnvoyFilterWrapper, error) {
	return envoyfilter.GenerateReplaceNetworkFilter(
		context.ServiceEntry,
		context.ServiceEntry.Spec.Ports[0],
		buildOutboundProxy(context),
		buildInboundProxy(context),
		"envoy.filters.network.thrift_proxy",
		"type.googleapis.com/envoy.extensions.filters.network.thrift_proxy.v3.ThriftProxy"), nil
}
