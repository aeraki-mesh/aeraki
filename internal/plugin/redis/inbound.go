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

package redis

import (
	redis "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/redis_proxy/v3"

	"github.com/aeraki-mesh/aeraki/internal/model"
)

func (g *Generator) buildInboundProxy(context *model.EnvoyFilterContext) *redis.RedisProxy {
	name := model.BuildClusterName(model.TrafficDirectionInbound, "", "", int(context.ServiceEntry.Spec.Ports[0].Number))
	proxy := &redis.RedisProxy{
		StatPrefix: name,
		Settings: &redis.RedisProxy_ConnPoolSettings{
			OpTimeout: defaultInboundOpTimeout,
		},
		PrefixRoutes: &redis.RedisProxy_PrefixRoutes{
			CatchAllRoute: &redis.RedisProxy_PrefixRoutes_Route{Cluster: name},
		},
	}
	return proxy
}
