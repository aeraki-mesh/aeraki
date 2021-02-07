package redis

import (
	"github.com/aeraki-framework/aeraki/pkg/model"
	redis "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/redis_proxy/v3"
)

func (g *Generator) buildInboundProxy(context *model.EnvoyFilterContext, port uint32, portName string) *redis.RedisProxy {
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
