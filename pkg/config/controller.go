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

package config

import (
	"reflect"
	"strings"
	"time"

	"github.com/aeraki-mesh/aeraki/pkg/envoyfilter"
	"github.com/aeraki-mesh/aeraki/pkg/model/protocol"
	"github.com/cenkalti/backoff"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/config/memory"
	istiomodel "istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/adsc"
	istioconfig "istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/collection"
	"istio.io/istio/pkg/config/schema/collections"
	"istio.io/pkg/log"
)

var (
	controllerLog = log.RegisterScope("config-controller", "config-controller debugging", 0)
	// We need serviceentry and virtualservice to generate the envoyfiters
	configCollection = collection.NewSchemasBuilder().MustAdd(collections.IstioNetworkingV1Alpha3Serviceentries).
				MustAdd(collections.IstioNetworkingV1Alpha3Virtualservices).
				MustAdd(collections.IstioNetworkingV1Alpha3Destinationrules).
				MustAdd(collections.IstioNetworkingV1Alpha3Envoyfilters).Build()
)

// Controller watches Istio config xDS server and notifies the listeners when config changes.
type Controller struct {
	configServerAddr string
	xdsMCP           *adsc.ADSC
	Store            istiomodel.ConfigStore
	controller       istiomodel.ConfigStoreCache
}

// NewController creates a new Controller instance based on the provided arguments.
func NewController(configServerAddr string) *Controller {
	store := memory.Make(configCollection)
	return &Controller{
		configServerAddr: configServerAddr,
		Store:            store,
		controller:       memory.NewController(store),
	}
}

// Run until a signal is received, this function won't block
func (c *Controller) Run(stop <-chan struct{}) {
	go c.controller.Run(stop)
	go func() {
		c.connectIstio()
		for {
			time.Sleep(30 * time.Minute)
			c.reconnectIstio()
		}
	}()
}

func (c *Controller) reconnectIstio() {
	controllerLog.Info("Close connection to Istio MCP over xDS server")
	c.closeConnection()
	c.connectIstio()
	controllerLog.Info("Reconnect to Istio MCP over xDS server")
}

func (c *Controller) closeConnection() {
	c.xdsMCP.Close()
}

func (c *Controller) connectIstio() {
	var err error
	for {
		c.xdsMCP, err = adsc.New(c.configServerAddr, &adsc.Config{
			Meta: istiomodel.NodeMetadata{
				Generator: "api",
			}.ToStruct(),
			InitialDiscoveryRequests: c.configInitialRequests(),
			BackoffPolicy:            backoff.NewConstantBackOff(time.Second),
		})
		if err != nil {
			controllerLog.Errorf("failed to dial XDS %s %v", c.configServerAddr, err)
			time.Sleep(5 * time.Second)
			continue
		}
		c.xdsMCP.Store = istiomodel.MakeIstioStore(c.controller)
		if err = c.xdsMCP.Run(); err != nil {
			controllerLog.Errorf("adsc: failed running %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		return
	}
}

func (c *Controller) configInitialRequests() []*discovery.DiscoveryRequest {
	schemas := configCollection.All()
	requests := make([]*discovery.DiscoveryRequest, len(schemas))
	for i, schema := range schemas {
		requests[i] = &discovery.DiscoveryRequest{
			TypeUrl: schema.Resource().GroupVersionKind().String(),
		}
	}
	return requests
}

// RegisterEventHandler adds a handler to receive config update events for a configuration type
func (c *Controller) RegisterEventHandler(protocols map[protocol.Instance]envoyfilter.Generator, handler func(istioconfig.Config, istioconfig.Config, istiomodel.Event)) {
	handlerWrapper := func(prev istioconfig.Config, curr istioconfig.Config, event istiomodel.Event) {
		if event == istiomodel.EventUpdate && reflect.DeepEqual(prev.Spec, curr.Spec) {
			return
		}
		// Now we only care about ServiceEntry and VirtualService
		if curr.GroupVersionKind == collections.IstioNetworkingV1Alpha3Serviceentries.Resource().GroupVersionKind() {
			//controllerLog.Infof("Service Entry changed: %s %s", event.String(), curr.Name)
			service, ok := curr.Spec.(*networking.ServiceEntry)
			if !ok {
				// This should never happen
				controllerLog.Errorf("failed in getting a virtual service: %v", curr.Name)
				return
			}
			for _, port := range service.Ports {
				if !strings.HasPrefix(port.Name, "tcp") {
					continue
				}
				handler(prev, curr, event)
			}
		} else if curr.GroupVersionKind == collections.IstioNetworkingV1Alpha3Virtualservices.Resource().GroupVersionKind() {
			controllerLog.Infof("virtual service changed: %s %s", event.String(), curr.Name)
			vs, ok := curr.Spec.(*networking.VirtualService)
			if !ok {
				// This should never happen
				controllerLog.Errorf("failed in getting a virtual service: %v", event.String(), curr.Name)
				return
			}
			serviceEntries, err := c.Store.List(collections.IstioNetworkingV1Alpha3Serviceentries.Resource().GroupVersionKind(), "")
			if err != nil {
				controllerLog.Errorf("failed to list configs: %v", err)
				return
			}
			for _, config := range serviceEntries {
				service, ok := config.Spec.(*networking.ServiceEntry)
				if !ok { // should never happen
					controllerLog.Errorf("failed in getting a service entry: %s: %v", config.Labels, err)
					return
				}
				if len(vs.Hosts) > 0 {
					for _, host := range service.Hosts {
						if host == vs.Hosts[0] {
							for _, port := range service.Ports {
								if _, ok := protocols[protocol.GetLayer7ProtocolFromPortName(port.Name)]; ok {
									handler(prev, curr, event)
								}
							}
						}
					}
				}
			}
		}
	}

	schemas := configCollection.All()
	for _, schema := range schemas {
		c.controller.RegisterEventHandler(schema.Resource().GroupVersionKind(), handlerWrapper)
	}
}
