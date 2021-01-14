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
	"github.com/aeraki-framework/aeraki/pkg/envoyfilter"
	"github.com/aeraki-framework/aeraki/pkg/model"
	"istio.io/pkg/log"
)

var generatorLog = log.RegisterScope("dubbo-generator", "dubbo generator", 0)

// Generator defines a dubbo envoyfilter Generator
type Generator struct {
}

// NewGenerator creates an new Dubbo Generator instance
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate create EnvoyFilters for Dubbo services
func (*Generator) Generate(context *model.EnvoyFilterContext) []*model.EnvoyFilterWrapper {
	return envoyfilter.GenerateReplaceNetworkFilter(
		context.ServiceEntry.Spec,
		buildOutboundProxy(context),
		buildInboundProxy(context),
		"envoy.filters.network.dubbo_proxy",
		"type.googleapis.com/envoy.extensions.filters.network.dubbo_proxy.v3.DubboProxy")
}
