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
	"net"
	"strconv"
	"strings"

	spec "github.com/aeraki-mesh/api/redis/v1alpha1"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoycore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	redis "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/redis_proxy/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/duration"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/protobuf/types/known/anypb"
	"istio.io/istio/pilot/pkg/xds/filters"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/aeraki-mesh/aeraki/internal/model"
)

// nolint: funlen,gocyclo
func (g *Generator) buildOutboundCluster(ctx context.Context, c *model.EnvoyFilterContext,
	listenPort uint32) *cluster.Cluster {
	const defaultConnectTimeout = 10

	cl := &cluster.Cluster{
		Name:           outboundClusterName(c.ServiceEntry.Spec.Hosts[0], listenPort),
		ConnectTimeout: &duration.Duration{Seconds: defaultConnectTimeout},
		ClusterDiscoveryType: &cluster.Cluster_Type{
			Type: cluster.Cluster_EDS,
		},
		EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
			EdsConfig: &envoycore.ConfigSource{
				ConfigSourceSpecifier: &envoycore.ConfigSource_Ads{},
				ResourceApiVersion:    envoycore.ApiVersion_V3,
			},
		},
		LbPolicy: cluster.Cluster_MAGLEV,
		CircuitBreakers: &cluster.CircuitBreakers{
			Thresholds: []*cluster.CircuitBreakers_Thresholds{
				getDefaultCircuitBreakerThresholds(),
			},
		},
	}

	destinations, err := g.redis.RedisDestinations(c.ServiceEntry.Namespace).List(ctx, v1.ListOptions{
		LabelSelector: labels.Everything().String(),
	})
	if err != nil {
		generatorLog.Errorf("could not list RedisDestinations: %e", err)
		return nil
	}
	var targetDestination *spec.RedisDestination
L:
	for i := range destinations.Items {
		destination := destinations.Items[i]
		for _, host := range c.ServiceEntry.Spec.Hosts {
			if destination.Spec.Host == host {
				targetDestination = &destination.Spec
				break L
			}
		}
	}
	if targetDestination == nil {
		return cl
	}
	if targetDestination.TrafficPolicy == nil {
		return cl
	}

	connPool := targetDestination.TrafficPolicy.ConnectionPool
	if connPool == nil {
		return cl
	}

	if connPool.Redis != nil {
		if connPool.Redis.Mode == spec.RedisSettings_CLUSTER {
			cl.LbPolicy = cluster.Cluster_CLUSTER_PROVIDED
			cl.ClusterDiscoveryType = &cluster.Cluster_ClusterType{ClusterType: &cluster.Cluster_CustomClusterType{
				Name: "envoy.clusters.redis",
			}}
			cl.EdsClusterConfig = nil
			var hostports []HostPort
			if len(connPool.Redis.DiscoveryEndpoints) != 0 {
				for _, ep := range connPool.Redis.DiscoveryEndpoints {
					hp := HostPort{}
					host, port, err := net.SplitHostPort(ep)
					if err != nil {
						if !strings.Contains(err.(*net.AddrError).Err, "missing port") {
							return nil
						}
						hp.Host = ep
						hp.Port = listenPort
					} else {
						hp.Host = host
						p, _ := strconv.Atoi(port)
						hp.Port = uint32(p)
					}
					hostports = append(hostports, hp)
				}
			} else {
				hostports = []HostPort{{
					targetDestination.Host, listenPort,
				}}
			}
			cl.LoadAssignment = &endpoint.ClusterLoadAssignment{
				ClusterName: cl.Name,
				Endpoints: []*endpoint.LocalityLbEndpoints{{
					LbEndpoints: toLbEndpoints(hostports...)}},
			}
		}
		if connPool.Redis.Auth != nil {
			username, password, err := g.password(c.ServiceEntry.Namespace, connPool.Redis.Auth)
			if err != nil {
				generatorLog.Errorf("get password from auth: %e", err)
				return nil
			}
			RedisProtocolOptions, err := anypb.New(&redis.RedisProtocolOptions{
				AuthPassword: inlineString(password),
				AuthUsername: inlineString(username),
			})
			if err != nil {
				generatorLog.Errorf("RedisProtocolOptions create failed: %e", err)
				return nil
			}
			cl.TypedExtensionProtocolOptions = map[string]*any.Any{
				"envoy.filters.network.redis_proxy": RedisProtocolOptions,
			}
		}
	}
	if connPool.Tcp != nil {
		threshold := getDefaultCircuitBreakerThresholds()
		timeout := connPool.Tcp.ConnectTimeout
		if timeout != nil {
			cl.ConnectTimeout = timeout
		}
		if connPool.Tcp.MaxConnections > 0 {
			threshold.MaxConnections = &wrappers.UInt32Value{Value: uint32(connPool.Tcp.MaxConnections)}
			cl.CircuitBreakers = &cluster.CircuitBreakers{
				Thresholds: []*cluster.CircuitBreakers_Thresholds{threshold},
			}
		}
		if connPool.Tcp.TcpKeepalive != nil {
			setKeepAliveSettings(cl, connPool.Tcp.TcpKeepalive)
		}
	}

	g.addIstioFilter(cl, listenPort, targetDestination.Host, c.ServiceEntry.Name, c.ServiceEntry.Namespace)

	// The `mTLS` cannot works with redis proxy now.
	// So we just use raw buffer
	// see https://github.com/istio/istio/issues/30022 .
	cl.TransportSocketMatches = []*cluster.Cluster_TransportSocketMatch{
		{
			Name:            "tlsMode-disabled",
			TransportSocket: &envoycore.TransportSocket{Name: wellknown.TransportSocketRawBuffer},
		},
	}
	return cl
}

func (g *Generator) addIstioFilter(cl *cluster.Cluster, port uint32, host, name, namespace string) {
	name = strings.TrimPrefix(name, "synthetic-")
	metadata := getOrCreateIstioMetadata(cl)
	// Add original_port field into istio metadata
	// Endpoint could override this port but the chance should be small.
	metadata.Fields["default_original_port"] = &structpb.Value{
		Kind: &structpb.Value_NumberValue{
			NumberValue: float64(port),
		},
	}
	metadata.Fields["services"] = &structpb.Value{
		Kind: &structpb.Value_ListValue{
			ListValue: &structpb.ListValue{
				Values: []*structpb.Value{},
			},
		},
	}
	svcList := metadata.Fields["services"].GetListValue()
	svcList.Values = append(svcList.Values, buildServiceMetadata(host, name, namespace))

	cl.Filters = append(cl.Filters, &cluster.Filter{
		Name:        filters.TCPClusterMx.Name,
		TypedConfig: filters.TCPClusterMx.GetTypedConfig(),
	})
}
