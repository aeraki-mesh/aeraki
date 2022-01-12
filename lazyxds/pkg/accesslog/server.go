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

package accesslog

import (
	"errors"
	"fmt"
	"net"

	"github.com/aeraki-mesh/aeraki/lazyxds/cmd/lazyxds/app/config"
	envoy_service_accesslog_v3 "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v3"
	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"k8s.io/klog/v2/klogr"
)

// Server ...
type Server struct {
	log     logr.Logger
	handler AccessHandler
}

// NewAccessLogServer ...
func NewAccessLogServer(handler AccessHandler) *Server {
	return &Server{
		log:     klogr.New().WithName("access-log-server"),
		handler: handler,
	}
}

// Serve ...
func (server *Server) Serve() error {
	port := fmt.Sprintf(":%d", config.AccessLogServicePort)
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return errors.New("failed to listen")
	}
	svc := grpc.NewServer()
	envoy_service_accesslog_v3.RegisterAccessLogServiceServer(svc, server)

	go func() {
		if err := svc.Serve(lis); err != nil {
			server.log.Error(err, "serve error")
		}
	}()
	// todo maybe graceful shutdown
	return nil
}
