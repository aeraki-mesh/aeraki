/*
 * // Copyright Aeraki Authors
 * //
 * // Licensed under the Apache License, Version 2.0 (the "License");
 * // you may not use this file except in compliance with the License.
 * // You may obtain a copy of the License at
 * //
 * //     http://www.apache.org/licenses/LICENSE-2.0
 * //
 * // Unless required by applicable law or agreed to in writing, software
 * // distributed under the License is distributed on an "AS IS" BASIS,
 * // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * // See the License for the specific language governing permissions and
 * // limitations under the License.
 */

package accesslog

import (
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils"
	envoycore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	al "github.com/envoyproxy/go-control-plane/envoy/data/accesslog/v3"
	als "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v3"
)

// AccessHandler ...
type AccessHandler interface {
	HandleAccess(fromIP, svcID, toIP string) error
}

// StreamAccessLogs accept access log from lazy xds egress gateway
func (server *Server) StreamAccessLogs(logStream als.AccessLogService_StreamAccessLogsServer) error {
	for {
		data, err := logStream.Recv()
		if err != nil {
			return err
		}

		httpLog := data.GetHttpLogs()
		if httpLog != nil {
			for _, entry := range httpLog.LogEntry {
				server.log.V(4).Info("http log entry", "entry", entry)
				fromIP := getDownstreamIP(entry)
				if fromIP == "" {
					continue
				}

				upstreamCluster := entry.CommonProperties.UpstreamCluster
				svcID := utils.UpstreamCluster2ServiceID(upstreamCluster)

				toIP := getUpstreamIP(entry)

				if err := server.handler.HandleAccess(fromIP, svcID, toIP); err != nil {
					server.log.Error(err, "handle access error")
				}
			}
		}
	}
}

func getDownstreamIP(entry *al.HTTPAccessLogEntry) string {
	if entry.CommonProperties.DownstreamRemoteAddress == nil {
		return ""
	}
	downstreamSock, ok := entry.CommonProperties.DownstreamRemoteAddress.Address.(*envoycore.Address_SocketAddress)
	if !ok {
		return ""
	}
	if downstreamSock == nil || downstreamSock.SocketAddress == nil {
		return ""
	}
	return downstreamSock.SocketAddress.Address
}

func getUpstreamIP(entry *al.HTTPAccessLogEntry) string {
	if entry.CommonProperties.UpstreamRemoteAddress == nil {
		return ""
	}
	upstreamSock, ok := entry.CommonProperties.UpstreamRemoteAddress.Address.(*envoycore.Address_SocketAddress)
	if !ok {
		return ""
	}
	if upstreamSock == nil || upstreamSock.SocketAddress == nil {
		return ""
	}

	return upstreamSock.SocketAddress.Address
}
