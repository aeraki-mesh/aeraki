// Copyright 2020 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package xds

import (
	"context"
	"fmt"
	"strings"
	"time"

	istioconfig "istio.io/istio/pkg/config"

	"github.com/aeraki-framework/aeraki/pkg/model"
	httproute "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"

	networking "istio.io/api/networking/v1alpha3"

	metaroute "github.com/aeraki-framework/meta-protocol-control-plane-api/meta_protocol_proxy/config/route/v1alpha"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"

	"github.com/zhaohuabing/debounce"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	istiomodel "istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config/schema/collections"
)

const (
	// debounceAfter is the delay added to events to wait after a registry event for debouncing.
	// This will delay the push by at least this interval, plus the time getting subsequent events.
	// If no change is detected the push will happen, otherwise we'll keep delaying until things settle.
	debounceAfter = 1 * time.Second

	// debounceMax is the maximum time to wait for events while debouncing.
	// Defaults to 10 seconds. If events keep showing up with no break for this time, we'll trigger a push.
	debounceMax = 10 * time.Second
)

// Controller contains the runtime configuration for the envoyFilter controller.
type Controller struct {
	istioClientset *istioclient.Clientset
	configStore    istiomodel.ConfigStore
	routeCache     cachev3.SnapshotCache
	// Sending on this channel results in a push.
	pushChannel chan istiomodel.Event
}

// NewController creates a new controller instance based on the provided arguments.
func NewController(istioClientset *istioclient.Clientset, store istiomodel.ConfigStore) *Controller {
	controller := &Controller{
		istioClientset: istioClientset,
		configStore:    store,
		routeCache:     cachev3.NewSnapshotCache(false, cachev3.IDHash{}, logger{}),
		pushChannel:    make(chan istiomodel.Event, 100),
	}
	return controller
}

// Cache return th route cache
func (c *Controller) Cache() cachev3.SnapshotCache {
	return c.routeCache
}

// Run until a signal is received, this function won't block
func (c *Controller) Run(stop <-chan struct{}) {
	go func() {
		c.mainLoop(stop)
	}()
}

func (c *Controller) mainLoop(stop <-chan struct{}) {
	callback := func() {
		err := c.updateRouteCache()
		if err != nil {
			xdsLog.Errorf("%v", err)
			// Retry if failed to push envoyFilters to AP IServer
			c.pushChannel <- istiomodel.EventUpdate
		} else {
			xdsLog.Infof("route cache updated")
		}
	}
	debouncer := debounce.New(debounceAfter, debounceMax, callback, stop)
	for {
		select {
		case e := <-c.pushChannel:
			xdsLog.Debugf("receive event from push chanel : %v", e)
			debouncer.Bounce()
		case <-stop:
			break
		}

	}
}

func (c *Controller) updateRouteCache() error {
	if len(c.routeCache.GetStatusKeys()) == 0 {
		xdsLog.Infof("no rds subscriber, ignore this update")
		return nil
	}
	serviceEntries, err := c.configStore.List(collections.IstioNetworkingV1Alpha3Serviceentries.Resource().
		GroupVersionKind(), "")
	if err != nil {
		return fmt.Errorf("failed to service entry configs: %v", err)
	}

	var routes []*metaroute.RouteConfiguration

	for _, config := range serviceEntries {
		service, ok := config.Spec.(*networking.ServiceEntry)
		if !ok { // should never happen
			xdsLog.Errorf("failed in getting a service entry: %s: %v", config.Labels, err)
			return nil
		}

		if len(service.Hosts) == 0 {
			xdsLog.Errorf("host should not be empty: %s", config.Name)
			// We can't retry in this scenario
			return nil
		}
		if len(service.Hosts) > 1 {
			xdsLog.Warnf("multiple hosts found for service: %s, only the first one will be processed", config.Name)
		}

		metaRoute := metaroute.RouteConfiguration{
			Name: model.BuildMetaProtocolRouteName(service.Hosts[0], int(service.Ports[0].Number)),
			Routes: []*metaroute.Route{
				{
					Name: "default",
					Match: &metaroute.RouteMatch{
						Metadata: []*httproute.HeaderMatcher{},
					},
					Route: &metaroute.RouteAction{
						ClusterSpecifier: &metaroute.RouteAction_Cluster{
							Cluster: model.BuildClusterName(model.TrafficDirectionOutbound, "",
								service.Hosts[0], int(service.Ports[0].Number)),
						},
					},
				},
			},
		}
		routes = append(routes, &metaRoute)
	}

	new := generateSnapshot(routes)
	for _, node := range c.routeCache.GetStatusKeys() {
		if err := c.routeCache.SetSnapshot(context.TODO(), node, new); err != nil {
			xdsLog.Errorf("failed to set route cache: %v", err)
			// We can't retry in this scenario
		}
	}
	return nil
}

// ConfigUpdate sends a config change event to the pushChannel of connections
func (c *Controller) ConfigUpdate(_ istioconfig.Config, curr istioconfig.Config, event istiomodel.Event) {
	if curr.GroupVersionKind == collections.IstioNetworkingV1Alpha3Serviceentries.Resource().GroupVersionKind() {
		service, ok := curr.Spec.(*networking.ServiceEntry)
		if !ok {
			xdsLog.Errorf("Failed in getting a virtual service: %v", curr.Name)
			return
		}
		if strings.HasPrefix(service.Ports[0].Name,
			"tcp-metaprotocol") { //@todo we may need to handle multiple ports in the future
			c.pushChannel <- event
		}
	}
}

func (c *Controller) initNode(node string) {
	// send a update event to pushChannel to trigger initialization of cache for a node.
	// we use update event here because update events are debounced, so the initialization of a large number of nodes
	// won't cause high cpu consumption.
	c.pushChannel <- istiomodel.EventUpdate
}

func (c *Controller) hasNode(node string) bool {
	if _, error := c.routeCache.GetSnapshot(node); error != nil {
		return false
	}
	return true
}

func (c *Controller) cache() cachev3.SnapshotCache {
	return c.routeCache
}
