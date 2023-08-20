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
	"context"
	"strings"
	"time"

	spec "github.com/aeraki-mesh/api/redis/v1alpha1"
	"github.com/aeraki-mesh/client-go/pkg/apis/redis/v1alpha1"
	envoycore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	redis "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/redis_proxy/v3"
	"google.golang.org/protobuf/types/known/durationpb"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pkg/config/schema/gvk"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/aeraki-mesh/aeraki/internal/model"
)

const (
	defaultUsername  = "username"
	defaultPassword  = "password"
	defaultPassword2 = "token"
)

var (
	defaultOpTimeout        = durationpb.New(time.Minute)
	defaultInboundOpTimeout = durationpb.New(time.Hour)
)

func (g *Generator) buildOutboundProxyWithFallback(ctx context.Context, c *model.EnvoyFilterContext, listenPort uint32,
	listenPortName string) *redis.RedisProxy {
	proxy, err := g.buildOutboundProxy(ctx, c, listenPort, listenPortName)
	if err != nil {
		generatorLog.Errorf("build outbound %s/%s :%e", c.ServiceEntry.Namespace, c.ServiceEntry.Name, err)
	}
	if proxy == nil {
		generatorLog.Debugf("build outbound %s/%s redisservice not found", c.ServiceEntry.Namespace, c.ServiceEntry.Name)
		return &redis.RedisProxy{
			Settings: &redis.RedisProxy_ConnPoolSettings{
				OpTimeout: defaultOpTimeout,
			},
			StatPrefix: outboundClusterName(c.ServiceEntry.Spec.Hosts[0], listenPort),
			PrefixRoutes: &redis.RedisProxy_PrefixRoutes{
				CatchAllRoute: &redis.RedisProxy_PrefixRoutes_Route{
					Cluster: outboundClusterName(c.ServiceEntry.Spec.Hosts[0], listenPort),
				},
			},
		}
	}
	return proxy
}

func (g *Generator) buildOutboundProxy(ctx context.Context, c *model.EnvoyFilterContext, listenPort uint32,
	listenPortName string) (*redis.RedisProxy, error) {
	targetHost, rs, err := g.findTargetHostAndRedisService(ctx, c.ServiceEntry.Namespace, c.ServiceEntry.Spec.Hosts)
	if err != nil {
		return nil, err
	}
	if rs == nil {
		generatorLog.Infof("no matched RedisService")
		return nil, nil
	}

	hostServices := g.hostServices(c.ServiceEntry.Namespace)

	proxy := &redis.RedisProxy{
		StatPrefix:   outboundClusterName(targetHost, listenPort),
		PrefixRoutes: &redis.RedisProxy_PrefixRoutes{},
		Settings:     &redis.RedisProxy_ConnPoolSettings{OpTimeout: defaultOpTimeout},
	}

	for _, fault := range rs.Spec.Faults {
		proxy.Faults = append(proxy.Faults, g.convertFault(fault))
	}

	if rs.Spec.Settings != nil {
		err = g.buildSettings(proxy, rs)
		if err != nil {
			return nil, err
		}
	}

	if len(rs.Spec.Redis) == 0 {
		proxy.PrefixRoutes.CatchAllRoute = &redis.RedisProxy_PrefixRoutes_Route{
			Cluster: outboundClusterName(targetHost, listenPort),
		}
		return proxy, nil
	}
	generatorLog.Debugf("redis service: %s", &rs.Spec)
	for _, r := range rs.Spec.Redis {
		route, all := g.buildPrefixRoute(r, hostServices, listenPort, listenPortName)
		if all {
			proxy.PrefixRoutes.CatchAllRoute = route
		} else {
			proxy.PrefixRoutes.Routes = append(proxy.PrefixRoutes.Routes, route)
		}
	}
	return proxy, nil
}

const maxWaitSecret = time.Second

// get password information from auth.
func (g *Generator) password(ns string, auth *spec.Auth) (username, password string, err error) {
	plain := auth.GetPlain()
	if plain != nil {
		return plain.Username, plain.Password, nil
	}
	secret := auth.GetSecret()
	if secret == nil {
		return "", "", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), maxWaitSecret)
	defer cancel()
	s, err := g.secretsGetter.Secrets(ns).Get(ctx, secret.Name, v1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	pf := secret.PasswordField
	if pf == "" {
		pf = defaultPassword
		if s.Data[pf] == nil {
			pf = defaultPassword2
		}
	}
	uf := secret.UsernameField
	if pf == "" {
		pf = defaultUsername
	}

	p := s.Data[pf]
	if p != nil {
		password = strings.TrimSpace(string(p))
	}

	u := s.Data[uf]
	if u != nil {
		username = strings.TrimSpace(string(u))
	}
	return username, password, nil
}

func (g *Generator) buildAuth(proxy *redis.RedisProxy, rs *v1alpha1.RedisService) error {
	username, password, err := g.password(rs.Namespace, rs.Spec.Settings.Auth)
	if err != nil {
		return err
	}
	if password != "" {
		// nolint: staticcheck
		proxy.DownstreamAuthPassword = inlineString(password)
	}
	if username != "" {
		proxy.DownstreamAuthUsername = inlineString(username)
	}
	return nil
}

func (g *Generator) findTargetHostAndRedisService(ctx context.Context, ns string, hosts []string) (targetHost string,
	rs *v1alpha1.RedisService, err error) {
	generatorLog.Debugf("try find target host and RedisService %s %v", ns, hosts)
	redisServices, err := g.redis.RedisServices(ns).List(ctx, v1.ListOptions{
		LabelSelector: labels.Everything().String(),
	})
	if err != nil {
		return "", nil, err
	}
	svcs := map[string]*v1alpha1.RedisService{}
	for i := range redisServices.Items {
		service := redisServices.Items[i]
		for _, svcHost := range service.Spec.Host {
			generatorLog.Debugf("related host: %s => %s", svcHost, service.Name)
			svcs[svcHost] = service
		}
	}
	for _, host := range hosts {
		rs = svcs[host]
		if rs != nil {
			targetHost = host
			return targetHost, rs, nil
		}
	}
	return "", nil, nil
}

func (g *Generator) hostServices(ns string) (hostServices map[string]*networking.ServiceEntry) {
	hostServices = map[string]*networking.ServiceEntry{}
	entries := g.store.List(gvk.ServiceEntry, ns)
	for i := range entries {
		se := entries[i].Spec.(*networking.ServiceEntry)
		for _, host := range se.Hosts {
			hostServices[host] = se
		}
	}
	return hostServices
}

func (g *Generator) convertPolicy(policy spec.RedisService_ReadPolicy) redis.RedisProxy_ConnPoolSettings_ReadPolicy {
	switch policy {
	case spec.RedisService_MASTER:
		return redis.RedisProxy_ConnPoolSettings_MASTER
	case spec.RedisService_PREFER_MASTER:
		return redis.RedisProxy_ConnPoolSettings_PREFER_MASTER
	case spec.RedisService_REPLICA:
		return redis.RedisProxy_ConnPoolSettings_REPLICA
	case spec.RedisService_PREFER_REPLICA:
		return redis.RedisProxy_ConnPoolSettings_PREFER_MASTER
	case spec.RedisService_ANY:
		return redis.RedisProxy_ConnPoolSettings_ANY
	}
	return redis.RedisProxy_ConnPoolSettings_MASTER
}

func (g *Generator) buildPrefixRoute(r *spec.RedisService_Route, hostServices map[string]*networking.ServiceEntry,
	listenPort uint32, listenPortName string) (route *redis.RedisProxy_PrefixRoutes_Route, all bool) {
	port := r.Route.Port
	if port == 0 {
		port = findServicePort(hostServices[r.Route.Host], listenPort, listenPortName)
	}
	route = &redis.RedisProxy_PrefixRoutes_Route{
		Cluster: outboundClusterName(r.Route.Host, port),
	}
	for _, mirror := range r.Mirror {
		policy := &redis.RedisProxy_PrefixRoutes_Route_RequestMirrorPolicy{}
		policy.ExcludeReadCommands = mirror.ExcludeReadCommands
		policy.RuntimeFraction = &envoycore.RuntimeFractionalPercent{
			DefaultValue: translatePercentToFractionalPercent(mirror.Percentage),
		}
		mirrorPort := r.Route.Port
		if mirrorPort == 0 {
			mirrorPort = findServicePort(hostServices[r.Route.Host], listenPort, listenPortName)
		}
		policy.Cluster = outboundClusterName(mirror.Route.Host, mirrorPort)
		route.RequestMirrorPolicy = append(route.RequestMirrorPolicy, policy)
	}
	generatorLog.Debugf("router match: %s", r.Match)
	if r.Match == nil || r.Match.GetKey() == nil {
		return route, true
	}
	route.Prefix = r.Match.GetKey().Prefix
	route.RemovePrefix = r.Match.GetKey().RemovePrefix
	return route, false
}

func (g *Generator) buildSettings(proxy *redis.RedisProxy, rs *v1alpha1.RedisService) (err error) {
	if rs.Spec.Settings.Auth != nil {
		err = g.buildAuth(proxy, rs)
		if err != nil {
			return err
		}
	}
	if rs.Spec.Settings.OpTimeout != nil {
		proxy.Settings.OpTimeout = rs.Spec.Settings.OpTimeout
	}

	proxy.Settings.EnableCommandStats = rs.Spec.Settings.EnableCommandStats
	proxy.Settings.EnableHashtagging = rs.Spec.Settings.EnableHashtagging
	proxy.Settings.EnableRedirection = rs.Spec.Settings.EnableRedirection

	if rs.Spec.Settings.BufferFlushTimeout != nil {
		proxy.Settings.BufferFlushTimeout = rs.Spec.Settings.BufferFlushTimeout
	}

	proxy.Settings.MaxBufferSizeBeforeFlush = rs.Spec.Settings.MaxBufferSizeBeforeFlush

	proxy.Settings.ReadPolicy = g.convertPolicy(rs.Spec.Settings.ReadPolicy)
	return nil
}

func (g *Generator) convertFault(fault *spec.Fault) *redis.RedisProxy_RedisFault {
	envoyFault := &redis.RedisProxy_RedisFault{Commands: fault.Commands}
	envoyFault.Delay = fault.Delay
	envoyFault.FaultEnabled = &envoycore.RuntimeFractionalPercent{
		DefaultValue: translatePercentToFractionalPercent(fault.Percentage),
	}
	switch fault.Type {
	case spec.Fault_DELAY:
		envoyFault.FaultType = redis.RedisProxy_RedisFault_DELAY
	case spec.Fault_ERROR:
		envoyFault.FaultType = redis.RedisProxy_RedisFault_ERROR
	}
	return envoyFault
}
