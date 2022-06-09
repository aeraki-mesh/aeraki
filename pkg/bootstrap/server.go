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
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/aeraki-mesh/aeraki/pkg/leaderelection"

	aerakischeme "github.com/aeraki-mesh/aeraki/client-go/pkg/clientset/versioned/scheme"
	"github.com/aeraki-mesh/aeraki/pkg/controller/istio"
	"github.com/aeraki-mesh/aeraki/pkg/controller/kube"

	istioscheme "istio.io/client-go/pkg/apis/networking/v1alpha3"
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

	"github.com/aeraki-mesh/aeraki/pkg/envoyfilter"
	"github.com/aeraki-mesh/aeraki/pkg/model/protocol"
	"github.com/aeraki-mesh/aeraki/pkg/xds"
	"github.com/aeraki-mesh/aeraki/plugin/dubbo"
	"github.com/aeraki-mesh/aeraki/plugin/redis"
)

var (
	aerakiLog = log.RegisterScope("aeraki-server", "aeraki-server debugging", 0)
)

// Server contains the runtime configuration for the Aeraki service.
type Server struct {
	args                  *AerakiArgs
	kubeClient            kubernetes.Interface
	configController      *istio.Controller
	envoyFilterController *envoyfilter.Controller
	xdsCacheMgr           *xds.CacheMgr
	xdsServer             *xds.Server
	httpsServer           *http.Server // webhooks HTTPS Server.
	scalableCtrlMgr       manager.Manager
	singletonCtrlMgr      manager.Manager
	// httpsMux listens on the httpsAddr(15017), handling webhooks
	// If the address os empty, the webhooks will be set on the default httpPort.
	httpsMux        *http.ServeMux // webhooks
	certMu          sync.RWMutex
	istiodCert      *tls.Certificate
	CABundle        *bytes.Buffer
	stopControllers func()
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
	configController := istio.NewController(&istio.Options{
		ClusterID:  args.ClusterID,
		IstiodAddr: args.IstiodAddr,
		NameSpace:  args.RootNamespace,
	})

	// envoyFilterController watches changes on config and create/update corresponding EnvoyFilters
	envoyFilterController := envoyfilter.NewController(client, configController.Store, args.Protocols,
		args.EnableEnvoyFilterNSScope)
	configController.RegisterEventHandler(func(_, curr *istioconfig.Config, event model.Event) {
		envoyFilterController.ConfigUpdated(event)
	})

	// routeCacheMgr watches service entry and generate the routes for meta protocol services
	routeCacheMgr := xds.NewCacheMgr(configController.Store)
	configController.RegisterEventHandler(func(prev *istioconfig.Config, curr *istioconfig.Config,
		event model.Event) {
		routeCacheMgr.ConfigUpdated(prev, curr, event)
	})

	// xdsServer is the RDS server for metaProtocol proxy
	xdsServer := xds.NewServer(args.XdsAddr, routeCacheMgr)

	// crdCtrlMgr watches Aeraki CRDs,  such as MetaRouter, ApplicationProtocol, etc.
	scalableCtrlMgr, err := createScalableControllers(args, kubeConfig, envoyFilterController, routeCacheMgr)
	if err != nil {
		return nil, err
	}
	// routeCacheMgr uses controller manager client to get route configuration in MetaRouters
	routeCacheMgr.MetaRouterControllerClient = scalableCtrlMgr.GetClient()
	// envoyFilterController uses controller manager client to get the rate limit configuration in MetaRouters
	envoyFilterController.MetaRouterControllerClient = scalableCtrlMgr.GetClient()

	//todo replace config with cached client
	cfg := scalableCtrlMgr.GetConfig()
	args.Protocols[protocol.Dubbo] = dubbo.NewGenerator(scalableCtrlMgr.GetConfig())
	args.Protocols[protocol.Redis] = redis.New(cfg, configController.Store)

	// singletonCtrlMgr
	singletonCtrlMgr, err := createSingletonControllers(args, kubeConfig)
	if err != nil {
		return nil, err
	}
	server := &Server{
		args:                  args,
		configController:      configController,
		envoyFilterController: envoyFilterController,
		scalableCtrlMgr:       scalableCtrlMgr,
		singletonCtrlMgr:      singletonCtrlMgr,
		xdsCacheMgr:           routeCacheMgr,
		xdsServer:             xdsServer,
	}

	if err := server.initKubeClient(); err != nil {
		return nil, fmt.Errorf("error initializing kube client: %v", err)
	}
	if err := server.initRootCA(); err != nil {
		return nil, fmt.Errorf("error initializing root ca: %v", err)
	}
	if err := server.initSecureWebhookServer(args); err != nil {
		return nil, fmt.Errorf("error initializing webhook server: %v", err)
	}
	if err := server.initConfigValidation(args); err != nil {
		return nil, fmt.Errorf("error initializing config validator: %v", err)
	}
	return server, err
}

// These controllers are horizontally scalable, multiple instances can be deployed to share the load
func createScalableControllers(args *AerakiArgs, kubeConfig *rest.Config,
	envoyFilterController *envoyfilter.Controller, xdsCacheMgr *xds.CacheMgr) (manager.Manager, error) {
	mgr, err := kube.NewManager(kubeConfig, args.RootNamespace, false, "")
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
	err = kube.AddRedisServiceController(mgr, updateEnvoyFilter)
	if err != nil {
		aerakiLog.Fatalf("could not add RedisServiceController: %e", err)
	}
	err = kube.AddRedisDestinationController(mgr, updateEnvoyFilter)
	if err != nil {
		aerakiLog.Fatalf("could not add RedisDestinationController: %e", err)
	}
	err = kube.AddDubboAuthorizationPolicyController(mgr, updateEnvoyFilter)
	if err != nil {
		aerakiLog.Fatalf("could not add DubboAuthorizationPolicyController: %e", err)
	}
	err = kube.AddApplicationProtocolController(mgr, updateEnvoyFilter)
	if err != nil {
		aerakiLog.Fatalf("could not add ApplicationProtocolController: %e", err)
	}
	err = kube.AddMetaRouterController(mgr, func() error {
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
	err = aerakischeme.AddToScheme(mgr.GetScheme())
	if err != nil {
		aerakiLog.Fatalf("could not add schema: %e", err)
	}
	return mgr, nil
}

// The Service Entry Controller is used to assign a globally unique VIP to a service entry,
// hence only one instance can get the lock to run
//
// Istio can allocate a VIP for a serviceentry, but the IPs are allocated in a sidecar scope, hence the IP of a
// service is not consistent across sidecar border.
// Since Aeraki is using the VIP of a serviceEntry as match condition when generating EnvoyFilter,
// the VIP must be unique and consistent in the mesh.
func createSingletonControllers(args *AerakiArgs, kubeConfig *rest.Config) (manager.Manager, error) {
	mgr, err := kube.NewManager(kubeConfig, args.RootNamespace, true, leaderelection.AllocateVIPController)
	if err != nil {
		return nil, err
	}
	err = kube.AddServiceEntryController(mgr)
	if err != nil {
		aerakiLog.Fatalf("could not add ServiceEntryController: %e", err)
	}
	err = kube.AddNamespaceController(mgr)
	if err != nil {
		aerakiLog.Fatalf("could not add NamespaceController: %e", err)
	}
	err = istioscheme.AddToScheme(mgr.GetScheme())
	if err != nil {
		aerakiLog.Fatalf("could not add schema: %e", err)
	}
	return mgr, nil
}

// Start starts all components of the Aeraki service. Serving can be canceled at any time by closing the provided stop channel.
// This method won't block
func (s *Server) Start(stop <-chan struct{}) {
	aerakiLog.Info("staring Aeraki Server")

	// Only create EnvoyFilters and assign VIP when running as in master mode
	if s.args.Master {
		aerakiLog.Infof("aeraki is running as the master")
		go func() {
			leaderelection.
				NewLeaderElection(s.args.RootNamespace, s.args.ServerID, leaderelection.EnvoyFilterController,
					s.kubeClient).
				AddRunFunction(func(leaderStop <-chan struct{}) {
					aerakiLog.Infof("starting EnvoyFilter creation controller")
					s.envoyFilterController.Run(stop)
				}).Run(stop)
		}()
	} else {
		aerakiLog.Infof("aeraki is running as a slave, only xds server will be started")
	}
	go func() {
		aerakiLog.Infof("watching xDS resource changes at %s", s.args.IstiodAddr)
		s.configController.Run(stop)
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
	s.stopControllers = cancel
	go func() {
		err := s.scalableCtrlMgr.Start(ctx)
		if err != nil {
			aerakiLog.Errorf("failed to start controllers: %v", err)
		}
	}()
	go func() {
		err := s.singletonCtrlMgr.Start(ctx)
		if err != nil {
			aerakiLog.Errorf("failed to start controllers: %v", err)
		}
	}()

	httpsListener, err := net.Listen("tcp", s.httpsServer.Addr)
	if err != nil {
		aerakiLog.Errorf("failed to start webhook server: %v", err)
	}
	go func() {
		log.Infof("starting webhook service at %s", httpsListener.Addr())
		if err := s.httpsServer.ServeTLS(httpsListener, "", ""); isUnexpectedListenerError(err) {
			log.Errorf("error serving https server: %v", err)
		}
	}()

	s.waitForShutdown(stop)
}

func isUnexpectedListenerError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, net.ErrClosed) {
		return false
	}
	if errors.Is(err, http.ErrServerClosed) {
		return false
	}
	return true
}

// Wait for the stop, and do cleanups
func (s *Server) waitForShutdown(stop <-chan struct{}) {
	go func() {
		<-stop
		s.stopControllers()
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
	if args.RootNamespace != "" && args.ConfigStoreSecret != "" {
		client, err := kubernetes.NewForConfig(kubeConfig)
		if err != nil {
			err = fmt.Errorf("failed to get Kube client: %v", err)
			return nil, err
		}
		secret, err := client.CoreV1().Secrets(args.RootNamespace).Get(context.TODO(), args.ConfigStoreSecret,
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
