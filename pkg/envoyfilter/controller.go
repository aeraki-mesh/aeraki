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

	"github.com/aeraki-framework/aeraki/pkg/model"
	"github.com/aeraki-framework/aeraki/pkg/model/protocol"
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
	istioClientset *istioclient.Clientset
	configStore    istiomodel.ConfigStore
	generators     map[protocol.Instance]Generator
	// Sending on this channel results in a push.
	pushChannel chan istiomodel.Event
}

// NewController creates a new controller instance based on the provided arguments.
func NewController(istioClientset *istioclient.Clientset, store istiomodel.ConfigStore,
	generators map[protocol.Instance]Generator) *Controller {
	controller := &Controller{
		istioClientset: istioClientset,
		configStore:    store,
		generators:     generators,
		pushChannel:    make(chan istiomodel.Event, 100),
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
			c.ConfigUpdate(istiomodel.EventUpdate)
		}
	}
	debouncer := debounce.New(debounceAfter, debounceMax, callback, stop)
	for {
		select {
		case e := <-c.pushChannel:
			controllerLog.Debugf("Receive event from push chanel : %v", e)
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

	existingEnvoyFilters, _ := c.istioClientset.NetworkingV1alpha3().EnvoyFilters(configRootNS).List(context.TODO(), v1.ListOptions{
		LabelSelector: "manager=" + aerakiFieldManager,
	})

	for _, oldEnvoyFilter := range existingEnvoyFilters.Items {
		if newEnvoyFilter, ok := generatedEnvoyFilters[oldEnvoyFilter.Name]; !ok {
			controllerLog.Infof("Deleting EnvoyFilter: %v", model.Struct2JSON(oldEnvoyFilter))
			err = c.istioClientset.NetworkingV1alpha3().EnvoyFilters(configRootNS).Delete(context.TODO(), oldEnvoyFilter.Name,
				v1.DeleteOptions{})
			if err != nil {
				err = fmt.Errorf("failed to delete EnvoyFilter: %v", err)
			}
		} else {
			if !proto.Equal(newEnvoyFilter.Envoyfilter, &oldEnvoyFilter.Spec) {
				controllerLog.Infof("Updating EnvoyFilter: %v", model.Struct2JSON(*newEnvoyFilter.Envoyfilter))
				_, err = c.istioClientset.NetworkingV1alpha3().EnvoyFilters(configRootNS).Update(context.TODO(),
					c.toEnvoyFilterCRD(newEnvoyFilter, &oldEnvoyFilter),
					v1.UpdateOptions{FieldManager: aerakiFieldManager})
				if err != nil {
					err = fmt.Errorf("failed to update EnvoyFilter: %v", err)
				}
			} else {
				controllerLog.Infof("EnvoyFilter: %s unchanged", oldEnvoyFilter.Name)
			}
			delete(generatedEnvoyFilters, oldEnvoyFilter.Name)
		}
	}

	for _, wrapper := range generatedEnvoyFilters {
		_, err = c.istioClientset.NetworkingV1alpha3().EnvoyFilters(configRootNS).Create(context.TODO(), c.toEnvoyFilterCRD(wrapper,
			nil),
			v1.CreateOptions{FieldManager: aerakiFieldManager})
		controllerLog.Infof("Creating EnvoyFilter: %v", model.Struct2JSON(*wrapper.Envoyfilter))
		if err != nil {
			err = fmt.Errorf("failed to create EnvoyFilter: %v", err)
		}
	}
	return err
}

func (c *Controller) toEnvoyFilterCRD(new *model.EnvoyFilterWrapper, old *v1alpha3.EnvoyFilter) *v1alpha3.EnvoyFilter {
	envoyFilter := &v1alpha3.EnvoyFilter{
		ObjectMeta: v1.ObjectMeta{
			Name:      new.Name,
			Namespace: configRootNS,
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

		relatedVs, err := c.findRelatedVirtualService(service)
		if err != nil {
			return envoyFilters, fmt.Errorf("failed in finding the related virtual service : %s: %v", config.Name, err)
		}
		context := &model.EnvoyFilterContext{
			ServiceEntry: &model.ServiceEntryWrapper{
				Meta: config.Meta,
				Spec: service,
			},
			VirtualService: relatedVs,
		}
		for _, port := range service.Ports {
			instance := protocol.GetLayer7ProtocolFromPortName(port.Name)
			if generator, ok := c.generators[instance]; ok {
				envoyFilterWrappers := generator.Generate(context)
				for _, wrapper := range envoyFilterWrappers {
					envoyFilters[wrapper.Name] = wrapper
				}
				break
			}
		}
	}
	return envoyFilters, nil
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

// ConfigUpdate sends a config change event to the pushChannel of connections
func (c *Controller) ConfigUpdate(event istiomodel.Event) {
	c.pushChannel <- event
}
