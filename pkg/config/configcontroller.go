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

package config

import (
	"fmt"

	"istio.io/istio/pkg/config/schema/collection"

	"github.com/aeraki-framework/aeraki/pkg/model/protocol"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	meshconfig "istio.io/api/mesh/v1alpha1"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/config/memory"
	istiomodel "istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/adsc"
	"istio.io/istio/pkg/config/schema/collections"
	"istio.io/pkg/log"
)

var configCollection = collection.NewSchemasBuilder().MustAdd(collections.IstioNetworkingV1Alpha3Serviceentries).
	MustAdd(collections.IstioNetworkingV1Alpha3Virtualservices).MustAdd(collections.IstioNetworkingV1Alpha3Destinationrules).Build()

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

func (c *Controller) Run(stop <-chan struct{}) error {
	xdsMCP, err := adsc.New(&meshconfig.ProxyConfig{
		DiscoveryAddress: c.configServerAddr,
	}, &adsc.Config{
		Meta: istiomodel.NodeMetadata{
			Generator: "api",
			//ClusterID: "Kubernetes",
		}.ToStruct(),
	})
	xdsMCP.Store = istiomodel.MakeIstioStore(c.controller)

	if err != nil {
		return fmt.Errorf("failed to dial XDS %s %v", c.configServerAddr, err)
	}
	log.Infof("Start watching xDS MCP server at %s", c.configServerAddr)
	xdsMCP.WatchConfig()
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
func (c *Controller) RegisterEventHandler(instance protocol.Instance, handler func(istiomodel.Config, istiomodel.Config, istiomodel.Event)) {
	schemas := configCollection.All()
	for _, schema := range schemas {
		c.controller.RegisterEventHandler(schema.Resource().GroupVersionKind(), func(prev istiomodel.Config, curr istiomodel.Config, event istiomodel.Event) {
			//log.Infof("####Kind:%v, Name:%v", curr.GroupVersionKind.String(), curr.Name)
			if curr.GroupVersionKind == collections.IstioNetworkingV1Alpha3Serviceentries.Resource().GroupVersionKind() {
				service, ok := curr.Spec.(*networking.ServiceEntry)
				if !ok { // should never happen
					log.Errorf("Failed in getting a virtual service: %v", curr.Labels)
				}
				for _, port := range service.Ports {
					if protocol.GetLayer7ProtocolFromPortName(port.Name) == instance {
						handler(prev, curr, event)
					}
				}
			}

		})
	}
}
