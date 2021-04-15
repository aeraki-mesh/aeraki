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
	"context"
	"fmt"

	"istio.io/pkg/log"

	"github.com/aeraki-framework/aeraki/pkg/config/serviceentry"

	"github.com/aeraki-framework/aeraki/pkg/envoyfilter"

	"github.com/aeraki-framework/aeraki/pkg/config"
	"github.com/aeraki-framework/aeraki/pkg/kube/controller"
	"github.com/aeraki-framework/aeraki/pkg/model/protocol"
	"github.com/aeraki-framework/aeraki/plugin/redis"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	"istio.io/istio/pilot/pkg/model"
	istioconfig "istio.io/istio/pkg/config"
	kubeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	aerakiLog = log.RegisterScope("aeraki-server", "aeraki-server debugging", 0)
)

// Server contains the runtime configuration for the Aeraki service.
type Server struct {
	args                   *AerakiArgs
	configController       *config.Controller
	serviceEntryController *serviceentry.Controller
	envoyFilterController  *envoyfilter.Controller
	crdController          manager.Manager
	stopCRDController      func()
}

// NewServer creates a new Server instance based on the provided arguments.
func NewServer(args *AerakiArgs) (*Server, error) {
	ic, err := getIstioClient()
	if err != nil {
		return nil, err
	}
	configController := config.NewController(args.IstiodAddr)
	envoyFilterController := envoyfilter.NewController(ic, configController.Store, args.Protocols)
	crdController := controller.NewManager(args.Namespace, args.ElectionID, func() error {
		envoyFilterController.ConfigUpdate(model.EventUpdate)
		return nil
	})

	cfg := crdController.GetConfig()
	args.Protocols[protocol.Redis] = redis.New(cfg, configController.Store)

	configController.RegisterEventHandler(args.Protocols, func(_, curr istioconfig.Config, event model.Event) {
		envoyFilterController.ConfigUpdate(event)
	})

	serviceEntryController := serviceentry.NewController(ic)

	return &Server{
		args:                   args,
		configController:       configController,
		envoyFilterController:  envoyFilterController,
		crdController:          crdController,
		serviceEntryController: serviceEntryController,
	}, nil
}

// Start starts all components of the Aeraki service. Serving can be canceled at any time by closing the provided stop channel.
// This method won't block
func (s *Server) Start(stop <-chan struct{}) {
	aerakiLog.Info("Staring Aeraki Server")

	go func() {
		aerakiLog.Infof("Starting Envoy Filter Controller")
		s.envoyFilterController.Run(stop)
	}()

	go func() {
		aerakiLog.Infof("Watching xDS resource changes at %s", s.args.IstiodAddr)
		s.configController.Run(stop)
	}()

	go func() {
		aerakiLog.Infof("Starting ServiceEntry controller")
		s.serviceEntryController.Run(stop)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	s.stopCRDController = cancel
	go func() {
		_ = s.crdController.Start(ctx)
	}()

	s.waitForShutdown(stop)
}

// Wait for the stop, and do cleanups
func (s *Server) waitForShutdown(stop <-chan struct{}) {
	go func() {
		<-stop
		s.stopCRDController()
	}()
}

func getIstioClient() (*istioclient.Clientset, error) {
	config, err := kubeconfig.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("can not get kubernetes config: %v", err)
	}

	ic, err := istioclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create istio client: %v", err)
	}
	return ic, nil
}
