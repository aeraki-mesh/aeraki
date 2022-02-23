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
	"errors"
	"fmt"

	"github.com/aeraki-mesh/aeraki/pkg/leaderelection"

	"github.com/aeraki-mesh/aeraki/client-go/pkg/clientset/versioned/scheme"
	"github.com/aeraki-mesh/aeraki/pkg/config"
	"github.com/aeraki-mesh/aeraki/pkg/config/serviceentry"
	"github.com/aeraki-mesh/aeraki/pkg/envoyfilter"
	"github.com/aeraki-mesh/aeraki/pkg/kube/controller"
	"github.com/aeraki-mesh/aeraki/pkg/model/protocol"
	"github.com/aeraki-mesh/aeraki/pkg/xds"
	"github.com/aeraki-mesh/aeraki/plugin/dubbo"
	"github.com/aeraki-mesh/aeraki/plugin/redis"
	"istio.io/client-go/pkg/clientset/versioned"
	"istio.io/istio/pilot/pkg/model"
	istioconfig "istio.io/istio/pkg/config"
	"istio.io/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kubeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	aerakiLog = log.RegisterScope("aeraki-server", "aeraki-server debugging", 0)
)

// Server contains the runtime configuration for the Aeraki service.
type Server struct {
	args                   *AerakiArgs
	kubeClient             kubernetes.Interface
	configController       *config.Controller
	serviceEntryController *serviceentry.Controller
	envoyFilterController  *envoyfilter.Controller
	xdsCacheMgr            *xds.CacheMgr
	xdsServer              *xds.Server
	crdCtrlMgr             manager.Manager
	stopCRDController      func()
}

// NewServer creates a new Server instance based on the provided arguments.
func NewServer(args *AerakiArgs) (*Server, error) {
	kubeConfig, err := getConfigStoreKubeConfig(args)
	if err != nil {
		return nil, fmt.Errorf("failed to get Istio kube config store : %v", err)
	}
	client, err := versioned.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create istio client: %v", err)
	}

	// configController watches Istiod through MCP over xDS to get service entry and virtual service updates
	configController := config.NewController(&config.Options{
		ClusterID:  args.ClusterID,
		IstiodAddr: args.IstiodAddr,
		NameSpace:  args.Namespace,
	})

	// Todo remove serviceEntryController, now Istio can allocate IP
	serviceEntryController := serviceentry.NewController(client)

	// envoyFilterController watches changes on config and create/update corresponding EnvoyFilters
	envoyFilterController := envoyfilter.NewController(client, configController.Store, args.Protocols)
	configController.RegisterEventHandler(args.Protocols, func(_, curr istioconfig.Config, event model.Event) {
		envoyFilterController.ConfigUpdated(event)
	})

	// routeCacheMgr watches service entry and generate the routes for meta protocol services
	routeCacheMgr := xds.NewCacheMgr(configController.Store)
	configController.RegisterEventHandler(args.Protocols, func(prev istioconfig.Config, curr istioconfig.Config,
		event model.Event) {
		routeCacheMgr.ConfigUpdated(prev, curr, event)
	})

	// xdsServer is the RDS server for metaProtocol proxy
	xdsServer := xds.NewServer(args.XdsAddr, routeCacheMgr)

	// crdCtrlMgr watches Aeraki CRDs,  such as MetaRouter, ApplicationProtocol, etc.
	crdCtrlMgr, err := createCrdControllers(args, kubeConfig, envoyFilterController, routeCacheMgr)
	if err != nil {
		return nil, err
	}
	// routeCacheMgr uses controller manager client to get route configuration in MetaRouters
	routeCacheMgr.MetaRouterControllerClient = crdCtrlMgr.GetClient()
	// envoyFilterController uses controller manager client to get the rate limit configuration in MetaRouters
	envoyFilterController.MetaRouterControllerClient = crdCtrlMgr.GetClient()

	//todo replace config with cached client
	cfg := crdCtrlMgr.GetConfig()
	args.Protocols[protocol.Dubbo] = dubbo.NewGenerator(crdCtrlMgr.GetConfig())
	args.Protocols[protocol.Redis] = redis.New(cfg, configController.Store)

	server := &Server{
		args:                   args,
		configController:       configController,
		envoyFilterController:  envoyFilterController,
		crdCtrlMgr:             crdCtrlMgr,
		serviceEntryController: serviceEntryController,
		xdsCacheMgr:            routeCacheMgr,
		xdsServer:              xdsServer,
	}
	err = server.initKubeClient()
	return server, err
}

func createCrdControllers(args *AerakiArgs, kubeConfig *rest.Config,
	envoyFilterController *envoyfilter.Controller, xdsCacheMgr *xds.CacheMgr) (manager.Manager, error) {
	crdCtrlMgr, err := controller.NewManager(kubeConfig, args.Namespace)
	if err != nil {
		return nil, err
	}

	updateEnvoyFilter := func() error {
		envoyFilterController.ConfigUpdated(model.EventUpdate)
		return nil
	}
	updateCache := func() error {
		xdsCacheMgr.UpdateRoute()
		return nil
	}
	err = controller.AddRedisServiceController(crdCtrlMgr, updateEnvoyFilter)
	if err != nil {
		aerakiLog.Fatalf("could not add RedisServiceController: %e", err)
	}
	err = controller.AddRedisDestinationController(crdCtrlMgr, updateEnvoyFilter)
	if err != nil {
		aerakiLog.Fatalf("could not add RedisDestinationController: %e", err)
	}
	err = controller.AddDubboAuthorizationPolicyController(crdCtrlMgr, updateEnvoyFilter)
	if err != nil {
		aerakiLog.Fatalf("could not add DubboAuthorizationPolicyController: %e", err)
	}
	err = controller.AddApplicationProtocolController(crdCtrlMgr, updateEnvoyFilter)
	if err != nil {
		aerakiLog.Fatalf("could not add ApplicationProtocolController: %e", err)
	}
	err = controller.AddMetaRouterController(crdCtrlMgr, func() error {
		if err := updateEnvoyFilter(); err != nil { //MetaRouter Rate limit config will cause update on EnvoyFilters
			return err
		}
		if err := updateCache(); err != nil { //MetaRouter route config will cause update on RDS cache
			return err
		}
		return nil
	})
	if err != nil {
		aerakiLog.Fatalf("could not add MetaRouterController: %e", err)
	}
	err = scheme.AddToScheme(crdCtrlMgr.GetScheme())
	if err != nil {
		aerakiLog.Fatalf("could not add schema: %e", err)
	}
	return crdCtrlMgr, nil
}

// Start starts all components of the Aeraki service. Serving can be canceled at any time by closing the provided stop channel.
// This method won't block
func (s *Server) Start(stop <-chan struct{}) {
	aerakiLog.Info("staring Aeraki Server")

	go func() {
		leaderelection.
			NewLeaderElection(s.args.Namespace, s.args.ServerID, leaderelection.EnvoyFilterController, s.kubeClient).
			AddRunFunction(func(leaderStop <-chan struct{}) {
				aerakiLog.Infof("starting EnvoyFilter creation controller")
				s.envoyFilterController.Run(stop)
			}).Run(stop)
	}()

	go func() {
		aerakiLog.Infof("watching xDS resource changes at %s", s.args.IstiodAddr)
		s.configController.Run(stop)
	}()

	go func() {
		leaderelection.
			NewLeaderElection(s.args.Namespace, s.args.ServerID, leaderelection.AllocateVIPController, s.kubeClient).
			AddRunFunction(func(leaderStop <-chan struct{}) {
				aerakiLog.Infof("starting ServiceEntry IP allocation controller")
				s.serviceEntryController.Run(stop)
			}).Run(stop)
	}()

	go func() {
		aerakiLog.Infof("starting MetaProtocol routes controller")
		s.xdsCacheMgr.Run(stop)
	}()

	go func() {
		aerakiLog.Infof("starting MetaProtocol RDS server, listening on %s", s.args.XdsAddr)
		s.xdsServer.Run(stop)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	s.stopCRDController = cancel
	go func() {
		_ = s.crdCtrlMgr.Start(ctx)
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

func (s *Server) initKubeClient() error {
	kubeConfig, err := kubeconfig.GetConfig()
	if err != nil {
		return err
	}
	s.kubeClient, err = kubernetes.NewForConfig(kubeConfig)
	return err
}

func getConfigStoreKubeConfig(args *AerakiArgs) (*rest.Config, error) {
	kubeConfig, err := kubeconfig.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("can not get kubernetes config: %v", err)
	}

	// Aeraki allows to use a dedicated API Server as the Istio config store.
	// The credential to access this dedicated Istio config store should be stored in a secret
	if args.Namespace != "" && args.ConfigStoreSecret != "" {
		client, err := kubernetes.NewForConfig(kubeConfig)
		if err != nil {
			err = fmt.Errorf("failed to get Kube client: %v", err)
			return nil, err
		}
		secret, err := client.CoreV1().Secrets(args.Namespace).Get(context.TODO(), args.ConfigStoreSecret,
			metav1.GetOptions{})
		if err != nil {
			err = fmt.Errorf("failed to get Istio config store secret: %v", err)
			return nil, err
		}

		rawConfig := secret.Data["kubeconfig.admin"]
		kubeConfig, err = getRestConfig(rawConfig)
		if err != nil {
			err = fmt.Errorf("failed to get Istio config store secret: %v", err)
			return nil, err
		}
	}

	return kubeConfig, nil
}

func getRestConfig(kubeConfig []byte) (*rest.Config, error) {
	if len(kubeConfig) == 0 {
		return nil, errors.New("kubeconfig is empty")
	}

	rawConfig, err := clientcmd.Load(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("kubeconfig cannot be loaded: %v", err)
	}

	if err := clientcmd.Validate(*rawConfig); err != nil {
		return nil, fmt.Errorf("kubeconfig is not valid: %v", err)
	}

	clientConfig := clientcmd.NewDefaultClientConfig(*rawConfig, &clientcmd.ConfigOverrides{})
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create kube clients: %v", err)
	}
	return restConfig, nil
}
