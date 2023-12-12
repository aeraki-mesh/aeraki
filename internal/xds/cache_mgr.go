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

	metaprotocolapi "github.com/aeraki-mesh/api/metaprotocol/v1alpha1"
	metaprotocol "github.com/aeraki-mesh/client-go/pkg/apis/metaprotocol/v1alpha1"
	metaroute "github.com/aeraki-mesh/meta-protocol-control-plane-api/aeraki/meta_protocol_proxy/config/route/v1alpha"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/zhaohuabing/debounce"
	networking "istio.io/api/networking/v1alpha3"
	istiomodel "istio.io/istio/pilot/pkg/model"
	istioconfig "istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/gvk"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aeraki-mesh/aeraki/internal/model"
	"github.com/aeraki-mesh/aeraki/internal/model/protocol"
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
	const maxRetries = 3
	retries := 0

	callback := func() {
		err := c.updateRouteCache()
		if err != nil {
			xdsLog.Errorf("failed to update route cache: %v", err)
			// Retry if failed to update route cache
			if retries >= maxRetries {
				retries = 0
				return
			}
			retries++
			c.pushChannel <- istiomodel.EventUpdate
			return
		}
		retries = 0
		xdsLog.Infof("route cache updated")
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

	serviceEntries := c.configStore.List(gvk.ServiceEntry, "")

	routes := c.generateMetaRoutes(serviceEntries)
	snapshot, err := generateSnapshot(routes)
	if err != nil {
		xdsLog.Errorf("failed to generate route cache: %v", err)
		// We don't retry in this scenario
		return err
	}

	for _, node := range c.routeCache.GetStatusKeys() {
		xdsLog.Debugf("set route cahe for: %s", node)
		if err := c.routeCache.SetSnapshot(context.TODO(), node, snapshot); err != nil {
			xdsLog.Errorf("failed to set route cache: %v", err)
			return err
		}
	}
	return nil
}

func (c *CacheMgr) generateMetaRoutes(serviceEntries []istioconfig.Config) []*metaroute.RouteConfiguration {
	var routes []*metaroute.RouteConfiguration

	for i := range serviceEntries {
		config := serviceEntries[i]
		service, ok := config.Spec.(*networking.ServiceEntry)
		if !ok { // should never happen
			xdsLog.Errorf("failed in getting a service entry: %s", config.Name)
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
		destinationRule, err := c.findRelatedDestinationRule(&model.ServiceEntryWrapper{
			Meta: config.Meta,
			Spec: service,
		})
		if err != nil {
			xdsLog.Errorf("failed to list destination rule for service: %s", config.Name)
		}

		for _, port := range service.Ports {
			if protocol.GetLayer7ProtocolFromPortName(port.Name).IsMetaProtocol() {
				if metaRouter != nil {
					xdsLog.Debugf("find meta router ：%s for : %s", metaRouter.Name, config.Name)
				}
				if destinationRule != nil {
					xdsLog.Debugf("find destination rule ：%s for : %s", destinationRule.Name, config.Name)
				}
				if metaRouter != nil {
					routes = append(routes, c.constructRoute(service, port, metaRouter, destinationRule))
				} else {
					xdsLog.Debugf("no meta router for : %s", config.Name)
					routes = append(routes, c.defaultRoute(service, port, destinationRule))
				}
			}
		}
	}
	return routes
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
	port *networking.ServicePort, metaRouter *metaprotocol.MetaRouter, dr *model.DestinationRuleWrapper) *metaroute.
	RouteConfiguration {
	var routes []*metaroute.Route
	for _, route := range metaRouter.Spec.Routes {
		routes = append(routes, &metaroute.Route{
			Name: route.Name,
			Match: &metaroute.RouteMatch{
				Metadata: MetaMatch2HttpHeaderMatch(route.Match),
			},
			Route:            c.constructAction(port, route, dr),
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

func (c *CacheMgr) constructAction(port *networking.ServicePort,
	route *metaprotocolapi.MetaRoute, dr *model.DestinationRuleWrapper) *metaroute.RouteAction {
	var routeAction = &metaroute.RouteAction{}

	if route != nil {
		if len(route.Route) == 1 {
			subset := route.Route[0].Destination.Subset
			host := route.Route[0].Destination.Host
			dstPort := port.Number
			if route.Route[0].Destination.Port != nil && route.Route[0].Destination.Port.Number != 0 {
				dstPort = route.Route[0].Destination.Port.Number
			}
			routeAction.ClusterSpecifier = &metaroute.RouteAction_Cluster{
				Cluster: model.BuildClusterName(model.TrafficDirectionOutbound, subset,
					host, int(dstPort)),
			}
			policy := model.GetHashPolicy(dr, subset)
			if policy != "" {
				routeAction.HashPolicy = []string{policy}
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
				policy := model.GetHashPolicy(dr, subset)
				if policy != "" {
					routeAction.HashPolicy = append(routeAction.HashPolicy, policy)
				}
				totalWeight += routeDestination.Weight
			}
			routeAction.ClusterSpecifier = &metaroute.RouteAction_WeightedClusters{
				WeightedClusters: &routev3.WeightedCluster{
					Clusters: clusters,
					TotalWeight: &wrappers.UInt32Value{
						Value: totalWeight,
					},
				},
			}
		}

		if route.Mirror != nil {
			dstPort := port.Number
			if route.Mirror.Port != nil && route.Mirror.Port.Number != 0 {
				dstPort = route.Mirror.Port.Number
			}
			routeAction.RequestMirrorPolicies = []*metaroute.RouteAction_RequestMirrorPolicy{
				{
					Cluster: model.BuildClusterName(model.TrafficDirectionOutbound, route.Mirror.Subset,
						route.Mirror.Host, int(dstPort)),
				},
			}
			var mirrorPercent float64
			mirrorPercent = 100
			if route.MirrorPercentage != nil && route.MirrorPercentage.Value != 0 {
				mirrorPercent = route.MirrorPercentage.Value
			}
			routeAction.RequestMirrorPolicies[0].RuntimeFraction = &corev3.RuntimeFractionalPercent{
				DefaultValue: translatePercentToFractionalPercent(mirrorPercent),
			}
		}
	}

	return routeAction
}

func (c *CacheMgr) defaultRoute(service *networking.ServiceEntry, port *networking.ServicePort,
	dr *model.DestinationRuleWrapper) *metaroute.RouteConfiguration {
	metaRoute := metaroute.RouteConfiguration{
		Name: model.BuildMetaProtocolRouteName(service.Hosts[0], int(port.Number)),
		Routes: []*metaroute.Route{
			{
				Name: "default",
				Match: &metaroute.RouteMatch{
					Metadata: []*routev3.HeaderMatcher{},
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
	if dr != nil && dr.Spec.TrafficPolicy != nil && dr.Spec.TrafficPolicy.LoadBalancer != nil && dr.Spec.TrafficPolicy.
		LoadBalancer.GetConsistentHash() != nil && dr.Spec.TrafficPolicy.
		LoadBalancer.GetConsistentHash().GetHttpHeaderName() != "" {
		metaRoute.Routes[0].Route.HashPolicy = []string{dr.Spec.TrafficPolicy.LoadBalancer.GetConsistentHash().
			GetHttpHeaderName()}
	}
	return &metaRoute
}

func (c *CacheMgr) findRelatedServiceEntry(dr *model.DestinationRuleWrapper) (*model.ServiceEntryWrapper, error) {
	serviceEntries := c.configStore.List(gvk.ServiceEntry, "")

	for i := range serviceEntries {
		se, ok := serviceEntries[i].Spec.(*networking.ServiceEntry)
		if !ok { // should never happen
			return nil, fmt.Errorf("failed in getting a service entry: %s", serviceEntries[i].Name)
		}
		if model.IsFQDNEquals(dr.Spec.Host, dr.Namespace, se.Hosts[0], serviceEntries[i].Namespace) {
			return &model.ServiceEntryWrapper{
				Meta: serviceEntries[i].Meta,
				Spec: se,
			}, nil
		}
	}
	return nil, nil
}

func (c *CacheMgr) findRelatedMetaRouter(service *networking.ServiceEntry) (*metaprotocol.MetaRouter, error) {
	metaRouterList := metaprotocol.MetaRouterList{}
	err := c.MetaRouterControllerClient.List(context.TODO(), &metaRouterList, &client.ListOptions{})
	if err != nil {
		return nil, err
	}

	for i := range metaRouterList.Items {
		for _, host := range metaRouterList.Items[i].Spec.Hosts {
			if host == service.Hosts[0] {
				if len(metaRouterList.Items[i].Spec.Routes) > 0 {
					return metaRouterList.Items[i], nil
				}
				xdsLog.Warnf("no route in metaRouter: %v", metaRouterList.Items[i])
				return nil, nil
			}
		}
	}
	return nil, nil
}

func (c *CacheMgr) findRelatedDestinationRule(service *model.ServiceEntryWrapper) (*model.DestinationRuleWrapper,
	error) {
	drs := c.configStore.List(gvk.DestinationRule, "")

	for i := range drs {
		dr, ok := drs[i].Spec.(*networking.DestinationRule)
		if !ok { // should never happen
			return nil, fmt.Errorf("failed in getting a destination rule: %s", drs[i].Name)
		}
		if model.IsFQDNEquals(dr.Host, drs[i].Namespace, service.Spec.Hosts[0], service.Namespace) {
			return &model.DestinationRuleWrapper{
				Meta: drs[i].Meta,
				Spec: dr,
			}, nil
		}
	}
	return nil, nil
}

// ConfigUpdated sends a config change event to the pushChannel when Istio config changed
func (c *CacheMgr) ConfigUpdated(prev, curr *istioconfig.Config, event istiomodel.Event) {
	if c.shouldUpdateCache(curr) {
		c.pushChannel <- event
	} else if c.shouldUpdateCache(prev) {
		c.pushChannel <- event
	}
}

func (c *CacheMgr) shouldUpdateCache(config *istioconfig.Config) bool {
	var serviceEntry *networking.ServiceEntry
	if config.GroupVersionKind == gvk.ServiceEntry {
		service, ok := config.Spec.(*networking.ServiceEntry)
		if !ok {
			xdsLog.Errorf("Failed in getting a service entry: %v", config.Name)
			return false
		}
		serviceEntry = service
	}

	// Cache needs to be updated if dr changed, the hash policy in the dr is used to generate routes
	if config.GroupVersionKind == gvk.DestinationRule {
		dr, ok := config.Spec.(*networking.DestinationRule)
		if !ok {
			xdsLog.Errorf("Failed in getting a destination rule: %v", config.Name)
			return false
		}

		se, err := c.findRelatedServiceEntry(&model.DestinationRuleWrapper{
			Meta: config.Meta,
			Spec: dr,
		})
		if err != nil {
			xdsLog.Errorf("Failed to find service entry for dr %s, %v", config.Namespace, err)
		}
		if se != nil {
			serviceEntry = se.Spec
		}
	}

	if serviceEntry != nil {
		for _, port := range serviceEntry.Ports {
			if strings.HasPrefix(port.Name,
				"tcp-metaprotocol") {
				return true
			}
		}
	}
	return false
}

// UpdateRoute sends a config change event to the pushChannel when Meta Router changed
func (c *CacheMgr) UpdateRoute() {
	c.pushChannel <- istiomodel.EventUpdate
}

func (c *CacheMgr) initNode(_ string) {
	// send a update event to pushChannel to trigger initialization of cache for a node.
	// we use update event here because update events are debounced, so the initialization of a large number of nodes
	// won't cause high cpu consumption.
	c.pushChannel <- istiomodel.EventUpdate
}

func (c *CacheMgr) clearNode(node string) {
	c.routeCache.ClearSnapshot(node)
}

func (c *CacheMgr) hasNode(node string) bool {
	if _, err := c.routeCache.GetSnapshot(node); err != nil {
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
