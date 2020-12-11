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

package bootstrap

import (
	"github.com/aeraki-framework/aeraki/pkg/mcp"
	"istio.io/istio/pilot/pkg/model"
	istioconfig "istio.io/istio/pkg/config"

	"github.com/aeraki-framework/aeraki/pkg/config"
	"istio.io/pkg/log"
)

var (
	aerakiLog = log.RegisterScope("aeraki-server", "mcp debugging", 0)
)

// Server contains the runtime configuration for the Aeraki service.
type Server struct {
	args             *AerakiArgs
	configController *config.Controller
	mcpServer        *mcp.Server
}

// NewServer creates a new Server instance based on the provided arguments.
func NewServer(args *AerakiArgs) *Server {
	configController := config.NewController(args.IstiodAddr)
	mcpServer := mcp.NewServer(args.ListenAddr, configController.Store, args.Protocols)
	configController.RegisterEventHandler(args.Protocols, func(_, curr istioconfig.Config, event model.Event) {
		mcpServer.ConfigUpdate(event)
	})

	return &Server{
		args:             args,
		configController: configController,
		mcpServer:        mcpServer,
	}
}

// Start starts all components of the Aeraki service. Serving can be canceled at any time by closing the provided stop channel.
// This method won't block
func (s *Server) Start(stop <-chan struct{}) {
	aerakiLog.Info("Staring Aeraki Server")

	go func() {
		aerakiLog.Infof("Starting MCP service at %s", s.args.ListenAddr)
		if err := s.mcpServer.Start(); err != nil {
			aerakiLog.Warn(err)
		}
	}()

	go func() {
		aerakiLog.Infof("Watching xDS resource changes at %s", s.args.IstiodAddr)
		s.configController.Run(stop)
	}()
	s.waitForShutdown(stop)
}

// Wait for the stop, and do cleanups
func (s *Server) waitForShutdown(stop <-chan struct{}) {
	go func() {
		<-stop
		s.mcpServer.Stop()
	}()
}
