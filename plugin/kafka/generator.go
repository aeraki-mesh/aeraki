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

package kafka

import (
	"github.com/aeraki-framework/aeraki/pkg/envoyfilter"
	"github.com/aeraki-framework/aeraki/pkg/model"
)

// Generator defines a kafka envoyfilter Generator
type Generator struct {
}

// NewGenerator creates an new kafka Generator instance
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate create EnvoyFilters for Dubbo services
func (*Generator) Generate(context *model.EnvoyFilterContext) []*model.EnvoyFilterWrapper {
	return envoyfilter.GenerateInsertBeforeNetworkFilter(
		context.ServiceEntry,
		buildOutboundProxy(context),
		buildInboundProxy(context),
		"envoy.filters.network.kafka_broker",
		"type.googleapis.com/envoy.extensions.filters.network.kafka_broker.v3.KafkaBroker")
}
