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
	"reflect"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	versionedclient "istio.io/client-go/pkg/clientset/versioned"
	"k8s.io/client-go/rest"

	"github.com/aeraki-framework/aeraki/pkg/model"
	"github.com/aeraki-framework/aeraki/pkg/model/protocol"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	istiomodel "istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config/schema/collections"
	"istio.io/pkg/log"
)

const (
	// debounceAfter is the delay added to events to wait after a registry event for debouncing.
	// This will delay the push by at least this interval, plus the time getting subsequent events.
	// If no change is detected the push will happen, otherwise we'll keep delaying until things settle.
	debounceAfter = 500 * time.Millisecond

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
	configStore istiomodel.ConfigStore
	generators  map[protocol.Instance]Generator
	// Sending on this channel results in a push.
	pushChannel chan istiomodel.Event
}

// NewController creates a new controller instance based on the provided arguments.
func NewController(store istiomodel.ConfigStore, generators map[protocol.Instance]Generator) *Controller {
	controller := &Controller{
		configStore: store,
		generators:  generators,
		pushChannel: make(chan istiomodel.Event, 100),
	}
	return controller
}

// Run until a signal is received, this function won't block
func (s *Controller) Run(stop <-chan struct{}) {
	go func() {
		s.mainLoop(stop)
	}()
}

func (s *Controller) mainLoop(stop <-chan struct{}) {
	var timeChan <-chan time.Time
	var startDebounce time.Time
	var lastResourceUpdateTime time.Time
	pushCounter := 0
	debouncedEvents := 0

	for {
		select {
		case <-stop:
			break
		case e := <-s.pushChannel:
			controllerLog.Debugf("Receive event from push chanel : %v", e)
			lastResourceUpdateTime = time.Now()
			if debouncedEvents == 0 {
				controllerLog.Debugf("This is the first debounced event")
				startDebounce = lastResourceUpdateTime
			}
			timeChan = time.After(debounceAfter)
			debouncedEvents++
		case <-timeChan:
			controllerLog.Debugf("Receive event from time chanel")
			eventDelay := time.Since(startDebounce)
			quietTime := time.Since(lastResourceUpdateTime)
			// it has been too long since the first debounced event or quiet enough since the last debounced event
			if eventDelay >= debounceMax || quietTime >= debounceAfter {
				if debouncedEvents > 0 {
					pushCounter++
					controllerLog.Infof("Push debounce stable[%d] %d: %v since last change, %v since last push",
						pushCounter, debouncedEvents, quietTime, eventDelay)
					err := s.pushEnvoyFilters2APIServer()
					if err != nil {
						controllerLog.Errorf("%v", err)
						// Retry if failed to push envoyFilters to AP IServer
						s.ConfigUpdate(istiomodel.EventUpdate)
					}
					debouncedEvents = 0
				}
			} else {
				timeChan = time.After(debounceAfter - quietTime)
			}
		}
	}
}

func (s *Controller) pushEnvoyFilters2APIServer() error {
	generatedEnvoyFilters, err := s.generateEnvoyFilters()
	if err != nil {
		return fmt.Errorf("failed to generate EnvoyFilter: %v", err)
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("can not get kubernetes config: %v", err)
	}

	ic, err := versionedclient.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create istio client: %v", err)
	}

	existingEnvoyFilters, _ := ic.NetworkingV1alpha3().EnvoyFilters(configRootNS).List(context.TODO(), v1.ListOptions{
		LabelSelector: "manager=" + aerakiFieldManager,
	})

	for _, oldEnvoyFilter := range existingEnvoyFilters.Items {
		if newEnvoyFilter, ok := generatedEnvoyFilters[oldEnvoyFilter.Name]; !ok {
			controllerLog.Infof("Deleting EnvoyFilter: %v", oldEnvoyFilter)
			err = ic.NetworkingV1alpha3().EnvoyFilters(configRootNS).Delete(context.TODO(), oldEnvoyFilter.Name, v1.DeleteOptions{})
			if err != nil {
				err = fmt.Errorf("failed to create istio client: %v", err)
			}
		} else {
			if !reflect.DeepEqual(newEnvoyFilter.Envoyfilter, &oldEnvoyFilter.Spec) {
				controllerLog.Infof("Updating EnvoyFilter: %v", *newEnvoyFilter.Envoyfilter)
				_, err = ic.NetworkingV1alpha3().EnvoyFilters(configRootNS).Update(context.TODO(), s.toEnovyFilterCRD(newEnvoyFilter, &oldEnvoyFilter),
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
		_, err = ic.NetworkingV1alpha3().EnvoyFilters(configRootNS).Create(context.TODO(), s.toEnovyFilterCRD(wrapper, nil),
			v1.CreateOptions{FieldManager: aerakiFieldManager})
		controllerLog.Infof("Creating EnvoyFilter: %v", *wrapper.Envoyfilter)
		if err != nil {
			err = fmt.Errorf("failed to create EnvoyFilter: %v", err)
		}
	}
	return err
}

func (s *Controller) toEnovyFilterCRD(new *model.EnvoyFilterWrapper, old *v1alpha3.EnvoyFilter) *v1alpha3.EnvoyFilter {
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

func (s *Controller) generateEnvoyFilters() (map[string]*model.EnvoyFilterWrapper, error) {
	envoyFilters := make(map[string]*model.EnvoyFilterWrapper)
	serviceEntries, err := s.configStore.List(collections.IstioNetworkingV1Alpha3Serviceentries.Resource().GroupVersionKind(), "")
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

		relatedVs, err := s.findRelatedVirtualService(service)
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
			if generator, ok := s.generators[instance]; ok {
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

func (s *Controller) findRelatedVirtualService(service *networking.ServiceEntry) (*model.VirtualServiceWrapper, error) {
	virtualServices, err := s.configStore.List(collections.IstioNetworkingV1Alpha3Virtualservices.Resource().GroupVersionKind(), "")
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
func (s *Controller) ConfigUpdate(event istiomodel.Event) {
	s.pushChannel <- event
}
