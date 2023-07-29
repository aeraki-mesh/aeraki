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
	kafka "github.com/envoyproxy/go-control-plane/contrib/envoy/extensions/filters/network/kafka_broker/v3"

	"github.com/aeraki-mesh/aeraki/internal/model"
)

func buildOutboundProxy(context *model.EnvoyFilterContext) *kafka.KafkaBroker {
	return &kafka.KafkaBroker{
		StatPrefix: model.BuildClusterName(model.TrafficDirectionOutbound, "",
			context.ServiceEntry.Spec.Hosts[0], int(context.ServiceEntry.Spec.Ports[0].Number)),
	}
}

func buildInboundProxy(context *model.EnvoyFilterContext) *kafka.KafkaBroker {
	return &kafka.KafkaBroker{
		StatPrefix: model.BuildClusterName(model.TrafficDirectionInbound, "",
			context.ServiceEntry.Spec.Hosts[0], int(context.ServiceEntry.Spec.Ports[0].Number)),
	}
}
