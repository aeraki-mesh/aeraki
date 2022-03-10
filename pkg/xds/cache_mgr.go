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

	"github.com/aeraki-mesh/aeraki/pkg/model/protocol"

	metaprotocolapi "github.com/aeraki-mesh/aeraki/api/metaprotocol/v1alpha1"
	metaprotocol "github.com/aeraki-mesh/aeraki/client-go/pkg/apis/metaprotocol/v1alpha1"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	istioconfig "istio.io/istio/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aeraki-mesh/aeraki/pkg/model"
	httproute "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"

	metaroute "github.com/aeraki-mesh/meta-protocol-control-plane-api/meta_protocol_proxy/config/route/v1alpha"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
	networking "istio.io/api/networking/v1alpha3"

	"github.com/zhaohuabing/debounce"
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

// CacheMgr contains the runtime configuration for the envoyFilter controller.
type CacheMgr struct {
	MetaRouterControllerClient client.Client
	configStore                istiomodel.ConfigStore
	routeCache                 cachev3.SnapshotCache
	// Sending on this channel results in a push.
	pushChannel chan istiomodel.Event
}

// NewCacheMgr creates a new controller instance based on the provided arguments.
func NewCacheMgr(store istiomodel.ConfigStore) *CacheMgr {
	controller := &CacheMgr{
		configStore: store,
		routeCache:  cachev3.NewSnapshotCache(false, cachev3.IDHash{}, logger{}),
		pushChannel: make(chan istiomodel.Event, 100),
	}
	return controller
}

// Cache return th route cache
func (c *CacheMgr) Cache() cachev3.SnapshotCache {
	return c.routeCache
}

// Run until a signal is received, this function won't block
func (c *CacheMgr) Run(stop <-chan struct{}) {
	go func() {
		c.mainLoop(stop)
	}()
}

func (c *CacheMgr) mainLoop(stop <-chan struct{}) {
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

func (c *CacheMgr) updateRouteCache() error {
	if len(c.routeCache.GetStatusKeys()) == 0 {
		xdsLog.Infof("no rds subscriber, ignore this update")
		return nil
	}
	serviceEntries, err := c.configStore.List(collections.IstioNetworkingV1Alpha3Serviceentries.Resource().
		GroupVersionKind(), "")
	if err != nil {
		return fmt.Errorf("failed to list service entries from the config store: %v", err)
	}

	var routes []*metaroute.RouteConfiguration

	for _, config := range serviceEntries {
		service, ok := config.Spec.(*networking.ServiceEntry)
		if !ok { // should never happen
			xdsLog.Errorf("failed in getting a service entry: %s: %v", config.Labels, err)
			continue
		}

		if len(service.Ports) == 0 {
			xdsLog.Errorf("service has no ports: %s", config.Name)
			continue
		}
		if !isMetaProtocolService(service) {
			continue
		}
		if len(service.Hosts) == 0 {
			xdsLog.Errorf("host should not be empty: %s", config.Name)
			continue
		}
		if len(service.Hosts) > 1 {
			xdsLog.Warnf("multiple hosts found for service: %s, only the first one will be processed", config.Name)
		}

		metaRouter, err := c.findRelatedMetaRouter(service)
		if err != nil {
			xdsLog.Errorf("failed to list meta router for service: %s", config.Name)
		}

		for _, port := range service.Ports {
			if protocol.GetLayer7ProtocolFromPortName(port.Name).IsMetaProtocol() {
				if metaRouter != nil && len(metaRouter.Spec.Routes) > 0 {
					xdsLog.Debugf("find meta router ：%s for : %s", metaRouter.Name, config.Name)
					routes = append(routes, c.constructRoute(service, port, metaRouter))
				} else {
					xdsLog.Debugf("no meta router for : %s", config.Name)
					routes = append(routes, c.defaultRoute(service, port))
				}
			}
		}
	}

	new := generateSnapshot(routes)
	for _, node := range c.routeCache.GetStatusKeys() {
		xdsLog.Debugf("set route cahe for: %s", node)
		if err := c.routeCache.SetSnapshot(context.TODO(), node, new); err != nil {
			xdsLog.Errorf("failed to set route cache: %v", err)
			// We can't retry in this scenario
		}
	}
	return nil
}

func isMetaProtocolService(service *networking.ServiceEntry) bool {
	for _, port := range service.Ports {
		if protocol.GetLayer7ProtocolFromPortName(port.Name).IsMetaProtocol() {
			return true
		}
	}
	return false
}

func (c *CacheMgr) constructRoute(service *networking.ServiceEntry,
	port *networking.Port, metaRouter *metaprotocol.MetaRouter) *metaroute.
	RouteConfiguration {
	var routes []*metaroute.Route
	for _, route := range metaRouter.Spec.Routes {
		routes = append(routes, &metaroute.Route{
			Name: route.Name,
			Match: &metaroute.RouteMatch{
				Metadata: MetaMatch2HttpHeaderMatch(route.Match),
			},
			Route:            c.constructAction(service, port, route),
			RequestMutation:  c.constructMutation(route.RequestMutation),
			ResponseMutation: c.constructMutation(route.ResponseMutation),
		})
	}
	// Currently, the routes for different port are the same, but we may need different routes for different ports in
	// the future
	metaRoute := metaroute.RouteConfiguration{
		Name:   model.BuildMetaProtocolRouteName(service.Hosts[0], int(port.Number)),
		Routes: routes,
	}
	return &metaRoute
}

func (c *CacheMgr) constructAction(service *networking.ServiceEntry, port *networking.Port, route *metaprotocolapi.MetaRoute) *metaroute.RouteAction {
	var routeAction *metaroute.RouteAction
	if len(route.Route) == 1 {
		subset := route.Route[0].Destination.Subset
		host := route.Route[0].Destination.Host
		dstPort := port.Number
		if route.Route[0].Destination.Port != nil && route.Route[0].Destination.Port.Number != 0 {
			dstPort = route.Route[0].Destination.Port.Number
		}
		routeAction = &metaroute.RouteAction{
			ClusterSpecifier: &metaroute.RouteAction_Cluster{
				Cluster: model.BuildClusterName(model.TrafficDirectionOutbound, subset,
					host, int(dstPort)),
			},
		}
	} else {
		var clusters []*routev3.WeightedCluster_ClusterWeight
		var totalWeight uint32
		for _, routeDestination := range route.Route {
			subset := routeDestination.Destination.Subset
			host := routeDestination.Destination.Host
			dstPort := port.Number
			if routeDestination.Destination.Port != nil && routeDestination.Destination.Port.Number != 0 {
				dstPort = routeDestination.Destination.Port.Number
			}
			clusters = append(clusters, &routev3.WeightedCluster_ClusterWeight{
				Name: model.BuildClusterName(model.TrafficDirectionOutbound, subset,
					host, int(dstPort)),
				Weight: &wrappers.UInt32Value{
					Value: routeDestination.Weight,
				},
			})
			totalWeight += routeDestination.Weight
		}

		routeAction = &metaroute.RouteAction{
			ClusterSpecifier: &metaroute.RouteAction_WeightedClusters{
				WeightedClusters: &routev3.WeightedCluster{
					Clusters: clusters,
					TotalWeight: &wrappers.UInt32Value{
						Value: totalWeight,
					},
				},
			},
		}
	}
	return routeAction
}
func (c *CacheMgr) defaultRoute(service *networking.ServiceEntry, port *networking.Port) *metaroute.RouteConfiguration {
	metaRoute := metaroute.RouteConfiguration{
		Name: model.BuildMetaProtocolRouteName(service.Hosts[0], int(port.Number)),
		Routes: []*metaroute.Route{
			{
				Name: "default",
				Match: &metaroute.RouteMatch{
					Metadata: []*httproute.HeaderMatcher{},
				},
				Route: &metaroute.RouteAction{
					ClusterSpecifier: &metaroute.RouteAction_Cluster{
						Cluster: model.BuildClusterName(model.TrafficDirectionOutbound, "",
							service.Hosts[0], int(port.Number)),
					},
				},
			},
		},
	}
	return &metaRoute
}

func (c *CacheMgr) findRelatedMetaRouter(service *networking.ServiceEntry) (*metaprotocol.MetaRouter, error) {
	metaRouterList := metaprotocol.MetaRouterList{}
	err := c.MetaRouterControllerClient.List(context.TODO(), &metaRouterList, &client.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, metaRouter := range metaRouterList.Items {
		for _, host := range metaRouter.Spec.Hosts {
			if host == service.Hosts[0] {
				return &metaRouter, nil
			}
		}
	}
	return nil, nil
}

// ConfigUpdated sends a config change event to the pushChannel when Istio config changed
func (c *CacheMgr) ConfigUpdated(_ istioconfig.Config, curr istioconfig.Config, event istiomodel.Event) {
	if curr.GroupVersionKind == collections.IstioNetworkingV1Alpha3Serviceentries.Resource().GroupVersionKind() {
		service, ok := curr.Spec.(*networking.ServiceEntry)
		if !ok {
			xdsLog.Errorf("Failed in getting a virtual service: %v", curr.Name)
			return
		}
		for _, port := range service.Ports {
			if strings.HasPrefix(port.Name,
				"tcp-metaprotocol") {
				c.pushChannel <- event
			}
		}
	}
}

// UpdateRoute sends a config change event to the pushChannel when Meta Router changed
func (c *CacheMgr) UpdateRoute() {
	c.pushChannel <- istiomodel.EventUpdate
}

func (c *CacheMgr) initNode(node string) {
	// send a update event to pushChannel to trigger initialization of cache for a node.
	// we use update event here because update events are debounced, so the initialization of a large number of nodes
	// won't cause high cpu consumption.
	c.pushChannel <- istiomodel.EventUpdate
}

func (c *CacheMgr) hasNode(node string) bool {
	if _, error := c.routeCache.GetSnapshot(node); error != nil {
		return false
	}
	return true
}

func (c *CacheMgr) cache() cachev3.SnapshotCache {
	return c.routeCache
}

func (c *CacheMgr) constructMutation(mutation []*metaprotocolapi.KeyValue) []*metaroute.KeyValue {
	var result []*metaroute.KeyValue
	for _, keyValue := range mutation {
		result = append(result, &metaroute.KeyValue{Key: keyValue.Key, Value: keyValue.Value})
	}
	return result
}
