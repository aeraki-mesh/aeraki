package accesslog

import (
	"errors"
	"fmt"
	"github.com/aeraki-framework/aeraki/lazyxds/cmd/lazyxds/app/config"
	"github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v3"
	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"k8s.io/klog/v2/klogr"
	"net"
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
