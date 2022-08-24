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

package zookeeper

import (
	zookeeper "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/zookeeper_proxy/v3"

	"github.com/aeraki-mesh/aeraki/pkg/model"
)

func buildOutboundProxy(context *model.EnvoyFilterContext) *zookeeper.ZooKeeperProxy {
	return &zookeeper.ZooKeeperProxy{
		StatPrefix: model.BuildClusterName(model.TrafficDirectionOutbound, "",
			context.ServiceEntry.Spec.Hosts[0], int(context.ServiceEntry.Spec.Ports[0].Number)),
	}
}

func buildInboundProxy(context *model.EnvoyFilterContext) *zookeeper.ZooKeeperProxy {
	return &zookeeper.ZooKeeperProxy{
		StatPrefix: model.BuildClusterName(model.TrafficDirectionInbound, "",
			context.ServiceEntry.Spec.Hosts[0], int(context.ServiceEntry.Spec.Ports[0].Number)),
	}
}
