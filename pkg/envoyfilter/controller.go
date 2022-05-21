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
	"time"

	"github.com/gogo/protobuf/proto"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metaprotocol "github.com/aeraki-mesh/aeraki/client-go/pkg/apis/metaprotocol/v1alpha1"
	"github.com/aeraki-mesh/aeraki/pkg/model"
	"github.com/aeraki-mesh/aeraki/pkg/model/protocol"
	"github.com/zhaohuabing/debounce"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	istiomodel "istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config/schema/collections"
	"istio.io/pkg/log"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// debounceAfter is the delay added to events to wait after a registry event for debouncing.
	// This will delay the push by at least this interval, plus the time getting subsequent events.
	// If no change is detected the push will happen, otherwise we'll keep delaying until things settle.
	debounceAfter = 1 * time.Second

	// debounceMax is the maximum time to wait for events while debouncing.
	// Defaults to 10 seconds. If events keep showing up with no break for this time, we'll trigger a push.
	debounceMax = 10 * time.Second

	// configRootNS is the root config root namespace
	configRootNS = "istio-system"

	// aerakiFieldManager is the FileldManager for Aeraki CRDs
	aerakiFieldManager = "Aeraki"
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
	// Sending on this channel results in a push.
	pushChannel chan istiomodel.Event
}

// NewController creates a new controller instance based on the provided arguments.
func NewController(istioClientset *istioclient.Clientset, store istiomodel.ConfigStore,
	generators map[protocol.Instance]Generator, namespaceScoped bool) *Controller {
	controller := &Controller{
		istioClientset:  istioClientset,
		configStore:     store,
		generators:      generators,
		namespaceScoped: namespaceScoped,
		pushChannel:     make(chan istiomodel.Event, 100),
	}
	return controller
}

// Run until a signal is received, this function won't block
func (c *Controller) Run(stop <-chan struct{}) {
	go func() {
		c.mainLoop(stop)
	}()
}

func (c *Controller) mainLoop(stop <-chan struct{}) {
	callback := func() {
		err := c.pushEnvoyFilters2APIServer()
		if err != nil {
			controllerLog.Errorf("%v", err)
			// Retry if failed to push envoyFilters to AP IServer
			c.ConfigUpdated(istiomodel.EventUpdate)
		}
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
	if err != nil {
		return fmt.Errorf("failed to generate EnvoyFilter: %v", err)
	}

	//TODO use cached informer instead of calling k8s API server
	existingEnvoyFilters, _ := c.istioClientset.NetworkingV1alpha3().EnvoyFilters("").List(context.TODO(), v1.ListOptions{
		LabelSelector: "manager=" + aerakiFieldManager,
	})

	// Deleted envoyFilters
	for _, oldEnvoyFilter := range existingEnvoyFilters.Items {
		if _, ok := generatedEnvoyFilters[envoyFilterMapKey(oldEnvoyFilter.Name, oldEnvoyFilter.Namespace)]; !ok {
			controllerLog.Infof("deleting EnvoyFilter: namespace: %s name: %s %v", oldEnvoyFilter.Namespace,
				oldEnvoyFilter.Name, model.Struct2JSON(oldEnvoyFilter))
			err = c.istioClientset.NetworkingV1alpha3().EnvoyFilters(oldEnvoyFilter.Namespace).Delete(context.TODO(),
				oldEnvoyFilter.Name,
				v1.DeleteOptions{})
		}
	}

	// Changed envoyFilters
	for _, oldEnvoyFilter := range existingEnvoyFilters.Items {
		mapKey := envoyFilterMapKey(oldEnvoyFilter.Name, oldEnvoyFilter.Namespace)
		if newEnvoyFilter, ok := generatedEnvoyFilters[mapKey]; ok {
			if !proto.Equal(newEnvoyFilter.Envoyfilter, &oldEnvoyFilter.Spec) {
				controllerLog.Infof("updating EnvoyFilter: namespace: %s name: %s %v", newEnvoyFilter.Namespace,
					newEnvoyFilter.Name, model.Struct2JSON(*newEnvoyFilter.Envoyfilter))
				_, err = c.istioClientset.NetworkingV1alpha3().EnvoyFilters(newEnvoyFilter.Namespace).Update(context.TODO(),
					c.toEnvoyFilterCRD(newEnvoyFilter, &oldEnvoyFilter),
					v1.UpdateOptions{FieldManager: aerakiFieldManager})
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
			v1.CreateOptions{FieldManager: aerakiFieldManager})
	}
	return err
}

func (c *Controller) deleteEnvoyFilter(ef *model.EnvoyFilterWrapper) (err error) {
	err = c.istioClientset.NetworkingV1alpha3().EnvoyFilters(ef.Namespace).Delete(context.TODO(), ef.Name,
		v1.DeleteOptions{})
	return err
}

func (c *Controller) toEnvoyFilterCRD(new *model.EnvoyFilterWrapper, old *v1alpha3.EnvoyFilter) *v1alpha3.EnvoyFilter {
	envoyFilter := &v1alpha3.EnvoyFilter{
		ObjectMeta: v1.ObjectMeta{
			Name:      new.Name,
			Namespace: new.Namespace,
			Labels: map[string]string{
				"manager": aerakiFieldManager,
			},
		},
		Spec: *new.Envoyfilter,
	}
	if old != nil {
		envoyFilter.ResourceVersion = old.ResourceVersion
	}
	return envoyFilter
}

func (c *Controller) generateEnvoyFilters() (map[string]*model.EnvoyFilterWrapper, error) {
	envoyFilters := make(map[string]*model.EnvoyFilterWrapper)
	serviceEntries, err := c.configStore.List(collections.IstioNetworkingV1Alpha3Serviceentries.Resource().
		GroupVersionKind(), "")
	if err != nil {
		return envoyFilters, fmt.Errorf("failed to list configs: %v", err)
	}

	for _, config := range serviceEntries {
		service, ok := config.Spec.(*networking.ServiceEntry)
		if !ok { // should never happen
			return envoyFilters, fmt.Errorf("failed in getting a service entry: %s: %v", config.Labels, err)
		}

		if len(service.Hosts) == 0 {
			controllerLog.Errorf("host should not be empty: %s", config.Name)
			// We can't retry in this scenario
			return envoyFilters, nil
		}
		if len(service.Hosts) > 1 {
			controllerLog.Warnf("multiple hosts found for service: %s, only the first one will be processed", config.Name)
		}

		for _, port := range service.Ports {
			instance := protocol.GetLayer7ProtocolFromPortName(port.Name)
			if generator, ok := c.generators[instance]; ok {
				controllerLog.Infof("found generator for port: %s", port.Name)

				relatedVs, err := c.findRelatedVirtualService(service)
				if err != nil {
					return envoyFilters, fmt.Errorf("failed in finding the related virtual service : %s: %v", config.Name, err)
				}
				relatedMr, err := c.findRelatedMetaRouter(service)
				if err != nil {
					return envoyFilters, fmt.Errorf("failed in finding the related meta router : %s: %v", config.Name, err)
				}
				context := &model.EnvoyFilterContext{
					ServiceEntry: &model.ServiceEntryWrapper{
						Meta: config.Meta,
						Spec: service,
					},
					VirtualService: relatedVs,
					MetaRouter:     relatedMr,
				}

				envoyFilterWrappers, err := generator.Generate(context)
				if err != nil {
					controllerLog.Errorf("failed to generate envoy filter: service: %s, port: %s, error: %v",
						config.Name,
						port.Name, err)
				} else {
					for _, wrapper := range envoyFilterWrappers {
						var exportNSs []string
						if context.MetaRouter != nil {
							exportNSs = context.MetaRouter.Spec.ExportTo
						}
						if len(exportNSs) == 0 {
							wrapper.Namespace = c.defaultEnvoyFilterNS(context.ServiceEntry.Namespace)
							envoyFilters[envoyFilterMapKey(wrapper.Name, wrapper.Namespace)] = wrapper
						} else {
							for _, exportNS := range exportNSs {
								if exportNS == "." {
									exportNS = context.MetaRouter.Namespace
								} else if exportNS == "*" {
									exportNS = configRootNS
								}
								wrapper.Namespace = exportNS
								envoyFilters[envoyFilterMapKey(wrapper.Name, exportNS)] = wrapper
							}
						}
					}
				}
				break
			}
		}
	}
	return envoyFilters, nil
}

func (c *Controller) defaultEnvoyFilterNS(serviceNS string) string {
	if c.namespaceScoped {
		return serviceNS
	}
	return configRootNS
}

func envoyFilterMapKey(name, ns string) string {
	return ns + "-" + name
}

func (c *Controller) findRelatedVirtualService(service *networking.ServiceEntry) (*model.VirtualServiceWrapper, error) {
	virtualServices, err := c.configStore.List(
		collections.IstioNetworkingV1Alpha3Virtualservices.Resource().GroupVersionKind(), "")
	if err != nil {
		return nil, fmt.Errorf("failed to list configs: %v", err)
	}

	for _, vsConfig := range virtualServices {
		vs, ok := vsConfig.Spec.(*networking.VirtualService)
		if !ok { // should never happen
			return nil, fmt.Errorf("failed in getting a virtual service: %s: %v", vsConfig.Name, err)
		}

		//Todo: we may need to deal with delegate Virtual services
		for _, host := range vs.Hosts {
			if host == service.Hosts[0] {
				return &model.VirtualServiceWrapper{
					Meta: vsConfig.Meta,
					Spec: vs,
				}, nil
			}
		}
	}
	return nil, nil
}

func (c *Controller) findRelatedMetaRouter(service *networking.ServiceEntry) (*metaprotocol.MetaRouter, error) {
	metaRouterList := metaprotocol.MetaRouterList{}
	err := c.MetaRouterControllerClient.List(context.TODO(), &metaRouterList, &client.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, metaRouter := range metaRouterList.Items {
		for _, host := range metaRouter.Spec.Hosts {
			// Aeraki now only supports one host in the MetaRouter
			if host == service.Hosts[0] {
				return &metaRouter, nil
			}
		}
	}
	return nil, nil
}

// ConfigUpdated sends a config change event to the pushChannel to trigger the generation of envoyfilters
func (c *Controller) ConfigUpdated(event istiomodel.Event) {
	c.pushChannel <- event
}
