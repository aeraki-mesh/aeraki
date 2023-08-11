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
	"math"

	spec "github.com/aeraki-mesh/api/redis/v1alpha1"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoycore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoytype "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/networking/util"

	"github.com/aeraki-mesh/aeraki/internal/model"
)

func getOrCreateIstioMetadata(cluster *cluster.Cluster) *structpb.Struct {
	if cluster.Metadata == nil {
		cluster.Metadata = &envoycore.Metadata{
			FilterMetadata: map[string]*structpb.Struct{},
		}
	}
	// Create Istio metadata if does not exist yet
	if _, ok := cluster.Metadata.FilterMetadata[util.IstioMetadataKey]; !ok {
		cluster.Metadata.FilterMetadata[util.IstioMetadataKey] = &structpb.Struct{
			Fields: map[string]*structpb.Value{},
		}
	}
	return cluster.Metadata.FilterMetadata[util.IstioMetadataKey]
}

// Build a struct which contains service metadata and will be added into cluster label.
func buildServiceMetadata(host, name, namespace string) *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_StructValue{
			StructValue: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					// service fqdn
					"host": {
						Kind: &structpb.Value_StringValue{
							StringValue: host,
						},
					},
					// short name of the service
					"name": {
						Kind: &structpb.Value_StringValue{
							StringValue: name,
						},
					},
					// namespace of the service
					"namespace": {
						Kind: &structpb.Value_StringValue{
							StringValue: namespace,
						},
					},
				},
			},
		},
	}
}

func findServicePort(svc *networking.ServiceEntry, listenPort uint32, listenPortName string) uint32 {
	if svc == nil {
		return listenPort
	}
	if len(svc.Ports) == 1 {
		return svc.Ports[0].Number
	}
	for _, port := range svc.Ports {
		if port.Number == listenPort {
			return listenPort
		}
		if port.Name == listenPortName {
			return port.Number
		}
	}
	return listenPort
}

// translatePercentToFractionalPercent translates an v1alpha3 Percent instance
// to an envoy.type.FractionalPercent instance.
func translatePercentToFractionalPercent(p *spec.Percent) *envoytype.FractionalPercent {
	return &envoytype.FractionalPercent{
		Numerator:   uint32(p.Value * 10000),
		Denominator: envoytype.FractionalPercent_MILLION,
	}
}

// inlineString build a inlineString DataSource
func inlineString(str string) *envoycore.DataSource {
	if str == "" {
		return nil
	}
	return &envoycore.DataSource{Specifier: &envoycore.DataSource_InlineString{InlineString: str}}
}

// HostPort represents a address with host and port.
type HostPort struct {
	Host string
	Port uint32
}

func toLbEndpoints(addrs ...HostPort) (endpoints []*endpoint.LbEndpoint) {
	for _, addr := range addrs {
		ep := &endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: &envoycore.Address{
						Address: &envoycore.Address_SocketAddress{
							SocketAddress: &envoycore.SocketAddress{
								Address:       addr.Host,
								PortSpecifier: &envoycore.SocketAddress_PortValue{PortValue: addr.Port},
							},
						},
					},
				},
			},
		}
		endpoints = append(endpoints, ep)
	}
	return endpoints
}

// getDefaultCircuitBreakerThresholds returns a copy of the default circuit breaker thresholds for the given traffic
// direction.
func getDefaultCircuitBreakerThresholds() *cluster.CircuitBreakers_Thresholds {
	return &cluster.CircuitBreakers_Thresholds{
		// DefaultMaxRetries specifies the default for the Envoy circuit breaker parameter max_retries. This
		// defines the maximum number of parallel retries a given Envoy will allow to the upstream cluster. Envoy
		// defaults this value to 3, however that has shown to be insufficient during periods of pod churn (e.g.
		// rolling updates), where multiple endpoints in a cluster are terminated. In these scenarios the circuit
		// breaker can kick in before Pilot is able to deliver an updated endpoint list to Envoy,
		// leading to client-facing 503s.
		MaxRetries:         &wrappers.UInt32Value{Value: math.MaxUint32},
		MaxRequests:        &wrappers.UInt32Value{Value: math.MaxUint32},
		MaxConnections:     &wrappers.UInt32Value{Value: math.MaxUint32},
		MaxPendingRequests: &wrappers.UInt32Value{Value: math.MaxUint32},
	}
}

func setKeepAliveSettings(cluster *cluster.Cluster,
	keepalive *networking.ConnectionPoolSettings_TCPSettings_TcpKeepalive) {
	if keepalive.Probes > 0 {
		cluster.UpstreamConnectionOptions.TcpKeepalive.KeepaliveProbes = &wrappers.UInt32Value{Value: keepalive.Probes}
	}

	if keepalive.Time != nil {
		cluster.UpstreamConnectionOptions.TcpKeepalive.KeepaliveTime =
			&wrappers.UInt32Value{Value: uint32(keepalive.Time.Seconds)}
	}

	if keepalive.Interval != nil {
		cluster.UpstreamConnectionOptions.TcpKeepalive.KeepaliveInterval =
			&wrappers.UInt32Value{Value: uint32(keepalive.Interval.Seconds)}
	}
}

func outboundClusterName(host string, port uint32) string {
	return model.BuildClusterName(model.TrafficDirectionOutbound, "", host, int(port))
}
