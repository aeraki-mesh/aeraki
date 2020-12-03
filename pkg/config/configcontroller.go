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
	"fmt"
	"reflect"
	"time"

	"istio.io/istio/pkg/config/schema/collection"

	"github.com/aeraki-framework/aeraki/pkg/model/protocol"
	"github.com/cenkalti/backoff"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/config/memory"
	istiomodel "istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/adsc"
	istioconfig "istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/collections"
	"istio.io/pkg/log"
)

var (
	controllerLog = log.RegisterScope("config-controller", "mcp debugging", 0)
	// We need serviceentry and virtualservice to generate the envoyfiters
	configCollection = collection.NewSchemasBuilder().MustAdd(collections.IstioNetworkingV1Alpha3Serviceentries).
				MustAdd(collections.IstioNetworkingV1Alpha3Virtualservices).MustAdd(collections.IstioNetworkingV1Alpha3Destinationrules).Build()
)

// Controller watches Istio config xDS server and notifies the listeners when config changes
type Controller struct {
	configServerAddr string
	Store            istiomodel.ConfigStore
	controller       istiomodel.ConfigStoreCache
}

func NewController(configServerAddr string) *Controller {
	store := memory.Make(configCollection)
	return &Controller{
		configServerAddr: configServerAddr,
		Store:            store,
		controller:       memory.NewController(store),
	}
}

// Run until a signal is received, this function won't block
func (c *Controller) Run(stop <-chan struct{}) error {
	xdsMCP, err := adsc.New(c.configServerAddr, &adsc.Config{
		Meta: istiomodel.NodeMetadata{
			Generator: "api",
		}.ToStruct(),
		InitialDiscoveryRequests: c.configInitialRequests(),
		BackoffPolicy:            backoff.NewConstantBackOff(time.Second),
	})
	if err != nil {
		return fmt.Errorf("failed to dial XDS %s %v", c.configServerAddr, err)
	}
	xdsMCP.Store = istiomodel.MakeIstioStore(c.controller)
	if err = xdsMCP.Run(); err != nil {
		return fmt.Errorf("MCP: failed running %v", err)
	}
	c.controller.Run(stop)
	return nil
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
func (c *Controller) RegisterEventHandler(instance protocol.Instance, handler func(istioconfig.Config, istioconfig.Config, istiomodel.Event)) {
	handlerWrapper := func(prev istioconfig.Config, curr istioconfig.Config, event istiomodel.Event) {
		if event == istiomodel.EventUpdate && reflect.DeepEqual(prev.Spec, curr.Spec) {
			return
		}
		// Now we only care about ServiceEntry and VirtualService
		if curr.GroupVersionKind == collections.IstioNetworkingV1Alpha3Serviceentries.Resource().GroupVersionKind() {
			controllerLog.Infof("Service Entry changed: %s %s", event.String(), curr.Name)
			service, ok := curr.Spec.(*networking.ServiceEntry)
			if !ok {
				// This should never happen
				controllerLog.Errorf("Failed in getting a virtual service: %v", curr.Name)
				return
			}
			for _, port := range service.Ports {
				if protocol.GetLayer7ProtocolFromPortName(port.Name) == instance {
					handler(prev, curr, event)
				}
			}
		} else if curr.GroupVersionKind == collections.IstioNetworkingV1Alpha3Virtualservices.Resource().GroupVersionKind() {
			controllerLog.Infof("Virtual Service changed: %s %s", event.String(), curr.Name)
			vs, ok := curr.Spec.(*networking.VirtualService)
			if !ok {
				// This should never happen
				controllerLog.Errorf("Failed in getting a virtual service: %v", event.String(), curr.Name)
				return
			}
			serviceEntries, err := c.Store.List(collections.IstioNetworkingV1Alpha3Serviceentries.Resource().GroupVersionKind(), "")
			if err != nil {
				controllerLog.Errorf("Failed to list configs: %v", err)
				return
			}
			for _, config := range serviceEntries {
				service, ok := config.Spec.(*networking.ServiceEntry)
				if !ok { // should never happen
					controllerLog.Errorf("failed in getting a service entry: %s: %v", config.Labels, err)
					return
				}
				if len(service.Hosts) > 0 {
					for _, host := range service.Hosts {
						if host == vs.Hosts[0] {
							for _, port := range service.Ports {
								if protocol.GetLayer7ProtocolFromPortName(port.Name) == instance {
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
