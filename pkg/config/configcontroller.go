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

// Run until a signal is received, this function won't block
func (c *Controller) Run(stop <-chan struct{}) error {
	xdsMCP, err := adsc.New(&meshconfig.ProxyConfig{
		DiscoveryAddress: c.configServerAddr,
	}, &adsc.Config{
		Meta: istiomodel.NodeMetadata{
			Generator: "api",
			ClusterID: "Kubernetes",
		}.ToStruct(),
		BackoffPolicy: backoff.NewConstantBackOff(time.Second),
	})
	xdsMCP.Store = istiomodel.MakeIstioStore(c.controller)

	if err != nil {
		return fmt.Errorf("failed to dial XDS %s %v", c.configServerAddr, err)
	}
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
			if event == istiomodel.EventUpdate && reflect.DeepEqual(prev.Spec, curr.Spec) {
				log.Infof("Ignore this update because there is no change to the Spec: %v", curr)
				return
			}
			if curr.GroupVersionKind == collections.IstioNetworkingV1Alpha3Serviceentries.Resource().GroupVersionKind() {
				service, ok := curr.Spec.(*networking.ServiceEntry)
				if !ok {
					// This should never happen
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
