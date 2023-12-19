// Copyright 2020 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package xds

import (
	"context"
	"crypto/tls"
	"net"

	"google.golang.org/grpc/credentials"

	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"google.golang.org/grpc"
	"istio.io/pkg/log"
)

var xdsLog = log.RegisterScope("xds-server", "Aeraki xds server", 0)

const (
	grpcMaxConcurrentStreams = 1000000
)

type cacheMgr interface {
	initNode(node string)
	hasNode(node string) bool
	cache() cachev3.SnapshotCache
	clearNode(node string)
}

// Server serves xDS resources to Envoy sidecars
type Server struct {
	addr      string
	cacheMgr  cacheMgr
	TLSConfig tls.Config
}

// NewServer creates a xDS server
func NewServer(addr string, cacheMgr cacheMgr) *Server {
	return &Server{
		cacheMgr: cacheMgr,
		addr:     addr,
	}
}

// Run an xDS server until an event is received over stopCh.
func (s *Server) Run(stopCh <-chan struct{}) {
	// gRPC golang library sets a very small upper bound for the number gRPC/h2
	// streams over a single TCP connection. If a proxy multiplexes requests over
	// a single connection to the management server, then it might lead to
	// availability problems.
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams))
	tlsCreds := credentials.NewTLS(&s.TLSConfig)
	grpcOptions = append(grpcOptions, grpc.Creds(tlsCreds))
	grpcServer := grpc.NewServer(grpcOptions...)

	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		xdsLog.Fatal(err)
	}
	srv := serverv3.NewServer(context.Background(), s.cacheMgr.cache(), newCallbacks(s.cacheMgr))
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, srv)

	xdsLog.Infof("management server listening on %s\n", s.addr)
	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			log.Errorf("failed to start grpc server: %v", err)
		}
	}()

	<-stopCh
	grpcServer.Stop()
}
