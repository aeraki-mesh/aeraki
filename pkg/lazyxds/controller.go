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
	// It's a two-layer map, the key of the first layer is the port number,
	//the key of the second layer is the service host
	cache       map[uint32]map[string]*istioconfig.Config
	istioClient *istioclient.Clientset
}

// NewServiceCache creates a new Service Cache.
func NewController(istioClient *istioclient.Clientset) *Controller {
	return &Controller{
		istioClient: istioClient,
		cache:       make(map[uint32]map[string]*istioconfig.Config),
	}
}

func (c *Controller) ServiceChange(config *istioconfig.Config) {
	service, ok := config.Spec.(*networking.ServiceEntry)
	if !ok { // should never happen
		controllerLog.Fatalf("failed to convert config to service entry: %s", config.Name)
		return
	}
	for _, port := range service.Ports {
		if protocol.GetLayer7ProtocolFromPortName(port.Name).IsMetaProtocol() {
			applicationProtocol, err := metaprotocolmodel.GetApplicationProtocolFromPortName(port.Name)
			if applicationProtocol == "dubbo" { //Currently only support Dubbo
				if err != nil {
					controllerLog.Errorf("failed to parse applicationProtocol from port name: %s %s",
						config.Name, port.Name)
				}
				if c.cache[port.Number] == nil {
					c.cache[port.Number] = make(map[string]*istioconfig.Config)
				}
				if c.cache[port.Number][service.Hosts[0]] == nil {
					c.cache[port.Number][service.Hosts[0]] = config
					//create sidecar
					//update the aggregated listeners on the lazyxds gateway and sidecars
					c.syncCatchAllListenersForSidecars()
					c.syncSidecars(config)
				}
			}
		}
	}
}
