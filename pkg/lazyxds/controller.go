// Copyright Istio Authors
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

package lazyxds

import (
	"time"

	kubelib "istio.io/istio/pkg/kube"

	"istio.io/istio/pkg/config/schema/collections"

	"github.com/zhaohuabing/debounce"
	"istio.io/istio/pilot/pkg/model"
	istiomodel "istio.io/istio/pilot/pkg/model"

	metaprotocolmodel "github.com/aeraki-mesh/aeraki/pkg/model/metaprotocol"
	"github.com/aeraki-mesh/aeraki/pkg/model/protocol"
	networking "istio.io/api/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	istioconfig "istio.io/istio/pkg/config"
	"istio.io/pkg/log"
)

const (
	// ResourcePrefix ...
	ResourcePrefix = "lazyxds-"
	LazyXdsManager = "lazyxds-manager"
	ManagedByLabel = "app.kubernetes.io/managed-by"

	// debounceAfter is the delay added to events to wait after a registry event for debouncing.
	// This will delay the push by at least this interval, plus the time getting subsequent events.
	// If no change is detected the push will happen, otherwise we'll keep delaying until things settle.
	debounceAfter = 1 * time.Second

	// debounceMax is the maximum time to wait for events while debouncing.
	// Defaults to 10 seconds. If events keep showing up with no break for this time, we'll trigger a push.
	debounceMax = 10 * time.Second
)

var (
	controllerLog = log.RegisterScope("lazyxds-controller", "lazyxds-controller debugging", 0)
)

// Controller is the controller for lazyxds.
type Controller struct {
	kubeClient   kubelib.Client
	istioClient  *istioclient.Clientset
	configStore  istiomodel.ConfigStore
	eventChannel chan *istioconfig.Config
}

// NewServiceCache creates a new Service Cache.
func NewController(kubeClient kubelib.Client, istioClient *istioclient.Clientset,
	store istiomodel.ConfigStore) *Controller {
	return &Controller{
		kubeClient:   kubeClient,
		istioClient:  istioClient,
		configStore:  store,
		eventChannel: make(chan *istioconfig.Config, 100),
		//cache:       make(map[uint32]map[string]*istioconfig.Config),
	}
}

// Run until a signal is received, this function won't block
func (c *Controller) Run(stop <-chan struct{}) {
	go func() {
		c.mainLoop(stop)
	}()
}

func (c *Controller) mainLoop(stop <-chan struct{}) {
	callback := func() {
		c.syncListeners()
	}
	debouncer := debounce.New(debounceAfter, debounceMax, callback, stop)
	for {
		select {
		case e := <-c.eventChannel:
			controllerLog.Debugf("receive a service change event: %v", e.Name)
			debouncer.Bounce()
		case <-stop:
			break
		}
	}
}

func (c *Controller) ServiceChange(config *istioconfig.Config, event model.Event) {
	c.eventChannel <- config
}

func (c *Controller) syncListeners() {
	// It's a two-layer map, the key of the first layer is the port number,
	//the key of the second layer is the service host
	ports := make(map[uint32]map[string]*istioconfig.Config)
	serviceEntries, err := c.configStore.List(collections.IstioNetworkingV1Alpha3Serviceentries.Resource().
		GroupVersionKind(), "")
	if err != nil {
		controllerLog.Errorf("can't list serviceentry: %v", err)
	}
	for i := range serviceEntries {
		ports = c.collectServicePorts(&serviceEntries[i], ports)
	}
	for port, services := range ports {
		// generate a catch-all listener on each serive port for sidecar outbound traffic
		c.syncCatchAllListener(port, services)
		// generate a listener on lazyxds gateway to forward the request to the real destination
		c.syncGatewayListener(port, services)
	}
	c.sycGatewayService(ports)
}

func (c *Controller) collectServicePorts(config *istioconfig.Config,
	ports map[uint32]map[string]*istioconfig.Config) map[uint32]map[string]*istioconfig.Config {
	service, ok := config.Spec.(*networking.ServiceEntry)
	if !ok { // should never happen
		controllerLog.Errorf("failed to convert config to service entry: %s", config.Name)
		return ports
	}

	for _, port := range service.Ports {
		if protocol.GetLayer7ProtocolFromPortName(port.Name).IsMetaProtocol() {
			applicationProtocol, err := metaprotocolmodel.GetApplicationProtocolFromPortName(port.Name)
			if applicationProtocol == "dubbo" { //Currently only support Dubbo
				if err != nil {
					controllerLog.Errorf("failed to parse applicationProtocol from port name: %s %s",
						config.Name, port.Name)
				}
				if ports[port.Number] == nil {
					ports[port.Number] = make(map[string]*istioconfig.Config)
				}
				if ports[port.Number][service.Hosts[0]] == nil {
					ports[port.Number][service.Hosts[0]] = config
				}
			}
		}
	}
	return ports
}
