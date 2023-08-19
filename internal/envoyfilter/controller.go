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

package envoyfilter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aeraki-mesh/api/metaprotocol/v1alpha1"
	metaprotocol "github.com/aeraki-mesh/client-go/pkg/apis/metaprotocol/v1alpha1"
	"github.com/zhaohuabing/debounce"
	"google.golang.org/protobuf/proto"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	istiomodel "istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/mesh"
	"istio.io/istio/pkg/config/schema/gvk"
	"istio.io/pkg/log"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aeraki-mesh/aeraki/internal/config/constants"
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

var (
	controllerLog = log.RegisterScope("envoyfilter-controller", "envoyfilter-controller debugging", 0)
)

// Controller contains the runtime configuration for the envoyFilter controller.
type Controller struct {
	istioClientset             *istioclient.Clientset
	MetaRouterControllerClient client.Client
	configStore                istiomodel.ConfigStore
	generators                 map[protocol.Instance]Generator
	namespaceScoped            bool
	namespace                  string
	// Sending on this channel results in a push.
	pushChannel chan istiomodel.Event
	meshConfig  mesh.Holder
}

// NewController creates a new controller instance based on the provided arguments.
func NewController(istioClientset *istioclient.Clientset, store istiomodel.ConfigStore,
	generators map[protocol.Instance]Generator, namespaceScoped bool, namespace string) *Controller {
	controller := &Controller{
		istioClientset:  istioClientset,
		configStore:     store,
		generators:      generators,
		namespaceScoped: namespaceScoped,
		namespace:       namespace,
		pushChannel:     make(chan istiomodel.Event, 100),
	}
	return controller
}

// InitMeshConfig global mesh configuration
func (c *Controller) InitMeshConfig(meshConfig mesh.Holder) {
	c.meshConfig = meshConfig
}

// Run until a signal is received, this function won't block
func (c *Controller) Run(stop <-chan struct{}) {
	go func() {
		c.mainLoop(stop)
	}()
}

func (c *Controller) mainLoop(stop <-chan struct{}) {
	const maxRetries = 3
	retries := 0
	callback := func() {
		controllerLog.Debugf("create envoyfilter")
		err := c.pushEnvoyFilters2APIServer()
		if err != nil {
			controllerLog.Errorf("failed to create envoyFilters: %v", err)
			// Retry if failed to create envoyFilters
			if retries >= maxRetries {
				retries = 0
				return
			}
			retries++
			c.ConfigUpdated(istiomodel.EventUpdate)
			return
		}
		retries = 0
	}
	debouncer := debounce.New(debounceAfter, debounceMax, callback, stop)
	for {
		select {
		case e := <-c.pushChannel:
			controllerLog.Debugf("receive event from push chanel : %v", e)
			debouncer.Bounce()
		case <-stop:
			break
		}
	}
}

func (c *Controller) pushEnvoyFilters2APIServer() error {
	generatedEnvoyFilters, err := c.generateEnvoyFilters()
	controllerLog.Debugf("create envoyfilter: %v", len(generatedEnvoyFilters))
	if err != nil {
		return fmt.Errorf("failed to generate EnvoyFilter: %v", err)
	}

	existingEnvoyFilters, _ := c.istioClientset.NetworkingV1alpha3().EnvoyFilters("").List(context.TODO(), v1.ListOptions{
		LabelSelector: "manager=" + constants.AerakiFieldManager,
	})

	// Deleted envoyFilters
	for i := range existingEnvoyFilters.Items {
		oldEnvoyFilter := existingEnvoyFilters.Items[i]
		if _, ok := generatedEnvoyFilters[envoyFilterMapKey(oldEnvoyFilter.Name, oldEnvoyFilter.Namespace)]; !ok {
			controllerLog.Infof("deleting EnvoyFilter: namespace: %s name: %s %v", oldEnvoyFilter.Namespace,
				oldEnvoyFilter.Name, model.Struct2JSON(oldEnvoyFilter))
			err = c.istioClientset.NetworkingV1alpha3().EnvoyFilters(oldEnvoyFilter.Namespace).Delete(context.TODO(),
				oldEnvoyFilter.Name,
				v1.DeleteOptions{})
		}
	}

	// Changed envoyFilters
	for i := range existingEnvoyFilters.Items {
		oldEnvoyFilter := existingEnvoyFilters.Items[i]
		mapKey := envoyFilterMapKey(oldEnvoyFilter.Name, oldEnvoyFilter.Namespace)
		if newEnvoyFilter, ok := generatedEnvoyFilters[mapKey]; ok {
			if !proto.Equal(newEnvoyFilter.Envoyfilter, &oldEnvoyFilter.Spec) {
				controllerLog.Infof("updating EnvoyFilter: namespace: %s name: %s %v", newEnvoyFilter.Namespace,
					newEnvoyFilter.Name, model.Struct2JSON(newEnvoyFilter.Envoyfilter))
				_, err = c.istioClientset.NetworkingV1alpha3().EnvoyFilters(newEnvoyFilter.Namespace).Update(context.TODO(),
					c.toEnvoyFilterCRD(newEnvoyFilter, oldEnvoyFilter),
					v1.UpdateOptions{FieldManager: constants.AerakiFieldManager})
			} else {
				controllerLog.Infof("envoyFilter: namespace: %s name: %s unchanged", oldEnvoyFilter.Namespace,
					oldEnvoyFilter.Name)
			}
			delete(generatedEnvoyFilters, mapKey)
		}
	}

	// New envoyFilters
	for _, wrapper := range generatedEnvoyFilters {
		controllerLog.Infof("creating EnvoyFilter: namespace: %s name: %s %v", wrapper.Namespace, wrapper.Name,
			model.Struct2JSON(wrapper.Envoyfilter))
		_, err = c.istioClientset.NetworkingV1alpha3().EnvoyFilters(wrapper.Namespace).Create(context.TODO(),
			c.toEnvoyFilterCRD(wrapper,
				nil),
			v1.CreateOptions{FieldManager: constants.AerakiFieldManager})
	}
	return err
}

func (c *Controller) toEnvoyFilterCRD(newEf *model.EnvoyFilterWrapper,
	oldEf *v1alpha3.EnvoyFilter) *v1alpha3.EnvoyFilter {
	envoyFilter := &v1alpha3.EnvoyFilter{
		ObjectMeta: v1.ObjectMeta{
			Name:      newEf.Name,
			Namespace: newEf.Namespace,
			Labels: map[string]string{
				"manager": constants.AerakiFieldManager,
			},
		},
		Spec: *newEf.Envoyfilter.DeepCopy(),
	}
	if oldEf != nil {
		envoyFilter.ResourceVersion = oldEf.ResourceVersion
	}
	return envoyFilter
}

func (c *Controller) generateEnvoyFilters() (map[string]*model.EnvoyFilterWrapper, error) {
	envoyFilters := make(map[string]*model.EnvoyFilterWrapper)
	serviceEntries := c.configStore.List(gvk.ServiceEntry, "")

	for i := range serviceEntries {
		service, ok := serviceEntries[i].Spec.(*networking.ServiceEntry)
		if !ok { // should never happen
			return envoyFilters, fmt.Errorf("failed in getting a service entry: %s", serviceEntries[i].Labels)
		}

		if len(service.Hosts) == 0 {
			controllerLog.Errorf("host should not be empty: %s", serviceEntries[i].Name)
			// We can't retry in this scenario
			return envoyFilters, nil
		}

		if len(service.Hosts) > 1 {
			controllerLog.Warnf("multiple hosts found for service: %s, only the first one will be processed",
				serviceEntries[i].Name)
		}

		for _, port := range service.Ports {
			instance := protocol.GetLayer7ProtocolFromPortName(port.Name)
			if generator, ok := c.generators[instance]; ok {
				controllerLog.Infof("found generator for port: %s", port.Name)

				ctx, err := c.envoyFilterContext(service, &serviceEntries[i])
				if err != nil {
					return envoyFilters, err
				}
				if ctx == nil {
					return envoyFilters, err
				}
				envoyFilterWrappers, err := generator.Generate(ctx)
				if err == nil {
					for _, wrapper := range envoyFilterWrappers {
						envoyFilters = c.createEnvoyFiltersOnExportNSs(ctx, wrapper, envoyFilters)
					}
				} else {
					controllerLog.Errorf("failed to generate envoy filter: service: %s, port: %s, error: %v",
						serviceEntries[i].Name,
						port.Name, err)
				}
				break
			}
		}
	}

	// generate envoyFilters for gateway with tcp-metaprotocol server
	err := c.generateGatewayEnvoyFilters(envoyFilters)

	return envoyFilters, err
}

func (c *Controller) generateGatewayEnvoyFilters(envoyFilters map[string]*model.EnvoyFilterWrapper) error {
	var envoyFilterContexts []*model.EnvoyFilterContext
	gateways := c.configStore.List(gvk.Gateway, "")

	for i := range gateways {
		gw, ok := gateways[i].Spec.(*networking.Gateway)
		if !ok { // should never happen
			log.Errorf("failed in getting a gateway: %s", gateways[i].Labels)
		}
		if gw.Servers == nil || len(gw.Servers) == 0 {
			continue
		}
		for _, server := range gw.Servers {
			if server.Port == nil {
				continue
			}
			instance := protocol.GetLayer7ProtocolFromPortName(server.Port.Name)
			if !instance.IsMetaProtocol() {
				// server l7 port name must be MetaProtocol to generate EnvoyFilter.
				continue
			}
			if generator, ok := c.generators[instance]; ok {
				controllerLog.Infof("found generator for router port: %s", server.Port.Name)

				ctxs, err := c.gatewayEnvoyFilterContexts(gw, &gateways[i], server.Port.Number)
				if err != nil {
					log.Errorf("failed to build EnvoyFilter Context router: %s, port: %s, error: %v",
						gateways[i].Name,
						server.Name, err)
					return nil
				}
				if len(ctxs) == 0 {
					continue
				}
				envoyFilterContexts = append(envoyFilterContexts, ctxs...)
				for _, ctx := range ctxs {
					envoyFilterWrappers, err := generator.Generate(ctx)
					if err != nil {
						controllerLog.Errorf("failed to generate router envoy filter: router: %s, port: %s, error: %v",
							gateways[i].Name,
							server.Name, err)
						continue
					}
					for _, wrapper := range envoyFilterWrappers {
						envoyFilters[envoyFilterMapKey(wrapper.Name, wrapper.Namespace)] = wrapper
					}
				}
			}
		}
	}

	// must create listeners for gateway before generate EnvoyFilters
	return c.generateListenerForGateway(envoyFilterContexts)
}

func (c *Controller) createEnvoyFiltersOnExportNSs(ctx *model.EnvoyFilterContext, wrapper *model.EnvoyFilterWrapper,
	envoyFilters map[string]*model.EnvoyFilterWrapper) map[string]*model.EnvoyFilterWrapper {
	var exportNSs []string
	if ctx.MetaRouter != nil {
		exportNSs = ctx.MetaRouter.Spec.ExportTo
	}
	if len(exportNSs) == 0 {
		// create an envoyfilter in the default export NS, which can be either the Root NS or the NS in which the
		// service is located, depends on the aeraki command option
		wrapper.Namespace = c.defaultEnvoyFilterNS(ctx.ServiceEntry.Namespace)
		envoyFilters[envoyFilterMapKey(wrapper.Name, wrapper.Namespace)] = wrapper
	} else {
		// create an envoyfilter in each exported NS
		for _, exportNS := range exportNSs {
			if exportNS == "." {
				exportNS = ctx.MetaRouter.Namespace
			} else if exportNS == "*" {
				exportNS = c.namespace
			}
			wrapperClone := &model.EnvoyFilterWrapper{
				Name:        wrapper.Name,
				Namespace:   exportNS,
				Envoyfilter: wrapper.Envoyfilter,
			}
			envoyFilters[envoyFilterMapKey(wrapperClone.Name, wrapperClone.Namespace)] = wrapperClone
		}
	}
	return envoyFilters
}

// envoyFilterContext wraps all the resources needed to create the EnvoyFilter
func (c *Controller) envoyFilterContext(service *networking.ServiceEntry,
	serviceEntry *config.Config) (*model.EnvoyFilterContext, error) {
	relatedVs, err := c.findRelatedVirtualService(service)
	if err != nil {
		return nil, fmt.Errorf("failed in finding the related virtual service : %s: %v", service.Hosts[0], err)
	}
	relatedMr, err := c.findRelatedMetaRouter(service)
	if err != nil {
		return nil, fmt.Errorf("failed in finding the related meta router : %s: %v", service.Hosts[0], err)
	}
	return &model.EnvoyFilterContext{
		MeshConfig: c.meshConfig,
		ServiceEntry: &model.ServiceEntryWrapper{
			Meta: serviceEntry.Meta,
			Spec: service,
		},
		VirtualService: relatedVs,
		MetaRouter:     relatedMr,
	}, nil
}

// gatewayEnvoyFilterContexts wraps all the resources needed to create the EnvoyFilters for a gateway
func (c *Controller) gatewayEnvoyFilterContexts(gatewaySpec *networking.Gateway, gateway *config.Config,
	portNumber uint32) ([]*model.EnvoyFilterContext, error) {
	var ctxs []*model.EnvoyFilterContext
	metaRouterList := &metaprotocol.MetaRouterList{}
	err := c.MetaRouterControllerClient.List(context.TODO(), metaRouterList, &client.ListOptions{})
	if err != nil {
		return nil, err
	}
	gatewayName := fmt.Sprintf("%s/%s", gateway.Namespace, gateway.Name)
	for i := range metaRouterList.Items {
		for _, gw := range metaRouterList.Items[i].Spec.Gateways {
			if j := strings.Index(gw, "/"); j < 0 {
				gw = metaRouterList.Items[i].Namespace + "/" + gw
			}
			if gw != gatewayName {
				continue
			}
			// the port in the MetaRouter destination must match with gateway server's port
			if !isMatchPort(portNumber, metaRouterList.Items[i].Spec.Routes) {
				continue
			}
			ctxs = append(ctxs, &model.EnvoyFilterContext{
				MeshConfig: c.meshConfig,
				Gateway: &model.GatewayWrapper{
					Meta: gateway.Meta,
					Spec: gatewaySpec,
				},
				ServiceEntry: &model.ServiceEntryWrapper{
					Spec: &networking.ServiceEntry{
						Hosts:     metaRouterList.Items[i].Spec.Hosts,
						Addresses: []string{"0.0.0.0"},
					},
				},
				MetaRouter: metaRouterList.Items[i],
				VirtualService: buildVirtualServiceWrapper(portNumber, gateway.Name, gateway.Namespace,
					metaRouterList.Items[i].Spec.Hosts[0]),
			})
			// A gateway port can only be defined by a MetaRouter
			break
		}
	}
	return ctxs, nil
}

func buildVirtualServiceWrapper(portNumber uint32, gatewayName, gatewayNs, host string) *model.VirtualServiceWrapper {
	return &model.VirtualServiceWrapper{
		Meta: config.Meta{
			Name:      fmt.Sprintf("aeraki-vs-%s.%s-%d", gatewayNs, gatewayName, portNumber),
			Namespace: gatewayNs,
			Labels: map[string]string{
				"manager": constants.AerakiFieldManager,
			},
		},
		Spec: &networking.VirtualService{
			Hosts:    []string{host},
			Gateways: []string{fmt.Sprintf("%s/%s", gatewayNs, gatewayName)},
			Tcp: []*networking.TCPRoute{
				{
					Match: []*networking.L4MatchAttributes{
						{
							Port: portNumber,
						},
					},
					Route: []*networking.RouteDestination{
						{
							Destination: &networking.Destination{
								Host: host,
								Port: &networking.PortSelector{
									Number: portNumber,
								},
							},
						},
					},
				},
			},
		},
	}
}

func isMatchPort(portNumber uint32, routes []*v1alpha1.MetaRoute) bool {
	if routes == nil {
		return false
	}
	for _, route := range routes {
		if route.Route == nil {
			continue
		}
		for _, destination := range route.Route {
			if destination.Destination != nil && destination.Destination.Port != nil &&
				destination.Destination.Port.Number == portNumber {
				return true
			}
		}
	}
	return false
}

func (c *Controller) defaultEnvoyFilterNS(serviceNS string) string {
	if c.namespaceScoped {
		return serviceNS
	}
	return c.namespace
}

func envoyFilterMapKey(name, ns string) string {
	return ns + "-" + name
}

func (c *Controller) findRelatedVirtualService(service *networking.ServiceEntry) (*model.VirtualServiceWrapper, error) {
	virtualServices := c.configStore.List(gvk.VirtualService, "")

	for i := range virtualServices {
		vs, ok := virtualServices[i].Spec.(*networking.VirtualService)
		if !ok { // should never happen
			return nil, fmt.Errorf("failed in getting a virtual service: %s", virtualServices[i].Name)
		}

		//Todo: we may need to deal with delegate Virtual services
		for _, host := range vs.Hosts {
			if host == service.Hosts[0] {
				return &model.VirtualServiceWrapper{
					Meta: virtualServices[i].Meta,
					Spec: vs,
				}, nil
			}
		}
	}
	return nil, nil
}

func (c *Controller) findRelatedMetaRouter(service *networking.ServiceEntry) (*metaprotocol.MetaRouter, error) {
	metaRouterList := &metaprotocol.MetaRouterList{}
	err := c.MetaRouterControllerClient.List(context.TODO(), metaRouterList, &client.ListOptions{})
	if err != nil {
		return nil, err
	}

	for i := range metaRouterList.Items {
		for _, host := range metaRouterList.Items[i].Spec.Hosts {
			// Aeraki now only supports one host in the MetaRouter
			if host == service.Hosts[0] {
				return metaRouterList.Items[i], nil
			}
		}
	}
	return nil, nil
}

// ConfigUpdated sends a config change event to the pushChannel to trigger the generation of envoyfilters
func (c *Controller) ConfigUpdated(event istiomodel.Event) {
	c.pushChannel <- event
}

// generateListenerForGateway generate listeners for gateway by create VirtualService
func (c *Controller) generateListenerForGateway(ctxs []*model.EnvoyFilterContext) error {
	generatedVirtualService := make(map[string]*v1alpha3.VirtualService)
	for _, ctx := range ctxs {
		if ctx.VirtualService == nil {
			continue
		}
		vs := &v1alpha3.VirtualService{
			ObjectMeta: v1.ObjectMeta{
				Name:      ctx.VirtualService.Name,
				Namespace: ctx.VirtualService.Namespace,
				Labels:    ctx.VirtualService.Labels,
			},
			Spec: networking.VirtualService{
				Hosts:    ctx.VirtualService.Spec.Hosts,
				Gateways: ctx.VirtualService.Spec.Gateways,
				Tcp:      ctx.VirtualService.Spec.Tcp,
			},
		}
		generatedVirtualService[virtualServiceMapKey(ctx.VirtualService.Name, ctx.VirtualService.Namespace)] = vs
	}

	existingVirtualService, _ := c.istioClientset.NetworkingV1alpha3().VirtualServices("").
		List(context.TODO(), v1.ListOptions{
			LabelSelector: "manager=" + constants.AerakiFieldManager,
		})

	// Deleted virtualServices
	var err error
	for i := range existingVirtualService.Items {
		oldVirtualService := existingVirtualService.Items[i]
		if _, ok := generatedVirtualService[virtualServiceMapKey(oldVirtualService.Name, oldVirtualService.Namespace)]; !ok {
			controllerLog.Infof("deleting VirtualService: namespace: %s name: %s %v", oldVirtualService.Namespace,
				oldVirtualService.Name, model.Struct2JSON(oldVirtualService))
			err = c.istioClientset.NetworkingV1alpha3().VirtualServices(oldVirtualService.Namespace).Delete(context.TODO(),
				oldVirtualService.Name,
				v1.DeleteOptions{})
		}
	}

	// Changed virtualServices
	for i := range existingVirtualService.Items {
		oldVirtualService := existingVirtualService.Items[i]
		mapKey := virtualServiceMapKey(oldVirtualService.Name, oldVirtualService.Namespace)
		if newVirtualService, ok := generatedVirtualService[mapKey]; ok {
			if !proto.Equal(&newVirtualService.Spec, &oldVirtualService.Spec) {
				controllerLog.Infof("updating VirtualService: namespace: %s name: %s %v", newVirtualService.Namespace,
					newVirtualService.Name, model.Struct2JSON(newVirtualService))
				_, err = c.istioClientset.NetworkingV1alpha3().VirtualServices(newVirtualService.Namespace).Update(context.TODO(),
					newVirtualService, v1.UpdateOptions{FieldManager: constants.AerakiFieldManager})
			} else {
				controllerLog.Infof("VirtualService: namespace: %s name: %s unchanged", newVirtualService.Namespace,
					newVirtualService.Name)
			}
			delete(generatedVirtualService, mapKey)
		}
	}

	// New envoyFilters
	for _, vs := range generatedVirtualService {
		controllerLog.Infof("creating VirtualService: namespace: %s name: %s %v", vs.Namespace, vs.Name,
			model.Struct2JSON(vs))
		_, err = c.istioClientset.NetworkingV1alpha3().VirtualServices(vs.Namespace).Create(context.TODO(),
			vs, v1.CreateOptions{FieldManager: constants.AerakiFieldManager})
	}
	return err
}

func virtualServiceMapKey(name, namespace string) string {
	return namespace + "/" + name
}
