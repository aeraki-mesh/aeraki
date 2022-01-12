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

package controller

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/aeraki-mesh/aeraki/lazyxds/cmd/lazyxds/app/config"
	"github.com/aeraki-mesh/aeraki/lazyxds/pkg/model"
	"github.com/aeraki-mesh/aeraki/lazyxds/pkg/utils"
	"github.com/aeraki-mesh/aeraki/lazyxds/pkg/utils/log"
	networking "istio.io/api/networking/v1alpha3"
	istio "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// syncServiceRuleOfEgress add/update virtualService of egress gateway
func (c *AggregationController) syncServiceRuleOfEgress(ctx context.Context, svc *model.Service) error {
	logger := log.FromContext(ctx)
	vs, err := c.istioClient.NetworkingV1alpha3().VirtualServices(config.IstioNamespace).
		Get(ctx, config.EgressVirtualServiceName, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		vs = &istio.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.EgressVirtualServiceName,
				Namespace: config.IstioNamespace,
				Labels: map[string]string{
					config.ManagedByLabel: config.LazyXdsManager,
				},
			},
			Spec: networking.VirtualService{
				Hosts:    []string{"*"},
				Gateways: []string{EgressGatewayFullName},
			},
		}
	} else if err != nil {
		return err
	}

	newSpec := networking.VirtualService{
		Hosts:    []string{"*"},
		Gateways: []string{EgressGatewayFullName},
	}
	routedPorts := make(map[string]bool)

	for _, hr := range vs.Spec.Http {
		serviceID := hr.Route[0].Destination.Host
		portNum := fmt.Sprint(hr.Route[0].Destination.Port.Number)

		if serviceID != svc.ID() {
			newSpec.Http = append(newSpec.Http, hr)
			continue
		}

		if _, ok := svc.Spec.HTTPPorts[portNum]; ok {
			logger.Info("Updating port rule of egress VirtualService", "port", portNum)
			routedPorts[portNum] = true
			newSpec.Http = append(newSpec.Http, httpRouteOfServicePort(svc, portNum))
		} else {
			logger.Info("Deleting port rule from egress VirtualService", "port", portNum)
		}
	}

	for num := range svc.Spec.HTTPPorts {
		if !routedPorts[num] {
			logger.Info("Adding port rule to egress VirtualService", "port", num)
			newSpec.Http = append(newSpec.Http, httpRouteOfServicePort(svc, num))
		}
	}

	return c.syncEgressVirtualService(ctx, vs, newSpec)
}

func (c *AggregationController) syncEgressVirtualService(
	ctx context.Context,
	vs *istio.VirtualService,
	spec networking.VirtualService,
) error {
	logger := log.FromContext(ctx)

	// use fixed order to avoid unnecessary update
	sort.Slice(spec.Http, func(i, j int) bool {
		di := spec.Http[i].Route[0].Destination
		dj := spec.Http[j].Route[0].Destination

		if di.Host == dj.Host {
			return di.Port.Number > dj.Port.Number
		}
		return di.Host > dj.Host
	})

	if len(spec.Http) == 0 {
		logger.Info("Deleting egress VirtualService")
		err := c.istioClient.NetworkingV1alpha3().VirtualServices(config.IstioNamespace).
			Delete(ctx, vs.Name, metav1.DeleteOptions{})
		if err != nil && errors.IsNotFound(err) {
			return nil
		}
		return err
	} else if vs.ResourceVersion == "" {
		logger.Info("Creating egress VirtualService")
		vs.Spec = spec
		_, err := c.istioClient.NetworkingV1alpha3().VirtualServices(config.IstioNamespace).
			Create(ctx, vs, metav1.CreateOptions{
				//FieldManager: LazyXdsManager,
			})
		return err
	} else {
		if reflect.DeepEqual(vs.Spec, spec) {
			return nil
		}
		vs.Spec = spec
		_, err := c.istioClient.NetworkingV1alpha3().VirtualServices(config.IstioNamespace).
			Update(ctx, vs, metav1.UpdateOptions{
				//FieldManager: LazyXdsManager,
			})
		return err
	}
}

// syncEnvoyFilterOfLazySource add/update RDS rule of lazy source using envoyFilter
func (c *AggregationController) syncEnvoyFilterOfLazySource(ctx context.Context, lazySvc *model.Service) error {
	logger := log.FromContext(ctx).WithValues("lazyservice", lazySvc.ID())
	envoyFilterName := ResourcePrefix + lazySvc.Name

	envoyFilter, err := c.istioClient.NetworkingV1alpha3().EnvoyFilters(lazySvc.Namespace).
		Get(ctx, envoyFilterName, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		envoyFilter = &istio.EnvoyFilter{
			ObjectMeta: metav1.ObjectMeta{
				Name:      envoyFilterName,
				Namespace: lazySvc.Namespace,
				Labels: map[string]string{
					config.ManagedByLabel: config.LazyXdsManager,
				},
			},
		}
	} else if err != nil {
		return err
	}

	newSpec := networking.EnvoyFilter{
		WorkloadSelector: &networking.WorkloadSelector{
			Labels: lazySvc.Spec.LazySelector,
		},
		ConfigPatches: c.envoyFilterPatchOfLazySource(lazySvc),
	}

	if len(newSpec.ConfigPatches) == 0 {
		return c.removeEnvoyFilter(ctx, lazySvc.Name, lazySvc.Namespace)
	} else if envoyFilter.ResourceVersion == "" {
		logger.Info("Creating egress envoy filter",
			"name", envoyFilter.Name, "ns", envoyFilter.Namespace)
		envoyFilter.Spec = newSpec
		_, err := c.istioClient.NetworkingV1alpha3().EnvoyFilters(lazySvc.Namespace).
			Create(ctx, envoyFilter, metav1.CreateOptions{
				FieldManager: config.LazyXdsManager,
			})
		return err
	} else {
		if reflect.DeepEqual(envoyFilter.Spec, newSpec) {
			return nil
		}
		envoyFilter.Spec = newSpec
		_, err := c.istioClient.NetworkingV1alpha3().EnvoyFilters(lazySvc.Namespace).
			Update(ctx, envoyFilter, metav1.UpdateOptions{
				FieldManager: config.LazyXdsManager,
			})
		return err
	}
}

func (c *AggregationController) removeEnvoyFilter(ctx context.Context, svcName, svcNamespace string) error {
	envoyFilterName := ResourcePrefix + svcName
	logger := log.FromContext(ctx).WithValues("envoyfilter", envoyFilterName, "ns", svcNamespace)
	logger.Info("Deleting envoyFilter")

	err := c.istioClient.NetworkingV1alpha3().EnvoyFilters(svcNamespace).
		Delete(ctx, envoyFilterName, metav1.DeleteOptions{})
	if err != nil && errors.IsNotFound(err) {
		return nil
	}

	return err
}

func (c *AggregationController) syncSidecarOfLazySource(ctx context.Context, lazySvc *model.Service) error {
	logger := log.FromContext(ctx).WithValues("lazyservice", lazySvc.ID())
	sidecarName := ResourcePrefix + lazySvc.Name

	// some service may be deleted
	for e := range lazySvc.EgressService {
		if _, ok := c.services.Load(e); !ok {
			delete(lazySvc.EgressService, e)
		}
	}

	sidecar, err := c.istioClient.NetworkingV1alpha3().Sidecars(lazySvc.Namespace).
		Get(ctx, sidecarName, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		sidecar = &istio.Sidecar{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sidecarName,
				Namespace: lazySvc.Namespace,
				Labels: map[string]string{
					config.ManagedByLabel: config.LazyXdsManager,
				},
			},
		}
	} else if err != nil {
		return err
	}

	newSpec := networking.Sidecar{
		WorkloadSelector: &networking.WorkloadSelector{
			Labels: lazySvc.Spec.LazySelector,
		},
		Egress: []*networking.IstioEgressListener{
			{
				Hosts: c.egressListOfLazySource(lazySvc),
			},
		},
	}

	if sidecar.ResourceVersion == "" {
		logger.Info("Creating sidecar")
		sidecar.Spec = newSpec
		_, err := c.istioClient.NetworkingV1alpha3().Sidecars(lazySvc.Namespace).
			Create(ctx, sidecar, metav1.CreateOptions{
				FieldManager: config.LazyXdsManager,
			})
		return err
	}

	if reflect.DeepEqual(sidecar.Spec, newSpec) {
		return nil
	}

	sidecar.Spec = newSpec
	_, err = c.istioClient.NetworkingV1alpha3().Sidecars(lazySvc.Namespace).
		Update(ctx, sidecar, metav1.UpdateOptions{
			FieldManager: config.LazyXdsManager,
		})
	return err
}

func (c *AggregationController) removeSidecar(ctx context.Context, svcName, svcNamespace string) error {
	sidecarName := ResourcePrefix + svcName
	logger := log.FromContext(ctx).WithValues("sidecar", sidecarName, "ns", svcNamespace)
	logger.Info("Deleting sidecar")

	err := c.istioClient.NetworkingV1alpha3().Sidecars(svcNamespace).
		Delete(ctx, sidecarName, metav1.DeleteOptions{})
	if err != nil && errors.IsNotFound(err) {
		return nil
	}

	return err
}

func httpRouteOfServicePort(svc *model.Service, port string) *networking.HTTPRoute {
	_, ok := svc.Spec.HTTPPorts[port]
	if !ok {
		return nil
	}

	num, _ := strconv.Atoi(port)
	route := &networking.HTTPRoute{
		Match: []*networking.HTTPMatchRequest{
			{
				//Gateways: []string{EgressGatewayFullName},
				Headers: map[string]*networking.StringMatch{
					ServiceAddressKey: {
						MatchType: &networking.StringMatch_Exact{
							Exact: utils.PortID(svc.ID(), port),
						},
					},
				},
			},
		},
		Route: []*networking.HTTPRouteDestination{
			{
				Destination: &networking.Destination{
					Host: svc.ID(),
					Port: &networking.PortSelector{Number: uint32(num)},
				},
				Headers: &networking.Headers{
					Request: &networking.Headers_HeaderOperations{
						Remove: []string{ServiceAddressKey},
					},
				},
			},
		},
	}

	return route
}

func (c *AggregationController) envoyFilterPatchOfLazySource(source *model.Service) []*networking.EnvoyFilter_EnvoyConfigObjectPatch {
	var newPatches []*networking.EnvoyFilter_EnvoyConfigObjectPatch
	egressWithBinding := c.visibleServiceOfLazyService(source)
	index := c.httpServicePortIndex()

	// we want sorted
	ports := make([]string, 0, len(index))
	for p := range index {
		ports = append(ports, p)
	}
	sort.Strings(ports)

	for _, num := range ports {
		services := index[num]
		var virtualHosts []string
		for _, svc := range services {
			if _, ok := egressWithBinding[svc.ID()]; ok { // services exported to current source
				continue
			}

			var domains []string
			for _, domain := range svc.DomainListOfPort(num, source.Namespace) {
				domains = append(domains, fmt.Sprintf(`"%s"`, domain))
			}
			domainList := strings.Join(domains, ",")
			portID := utils.PortID(svc.ID(), num)
			vh := fmt.Sprintf(`{
						"name":"%s",
						"domains":[%s],
						"routes":[{
							"name":"lazy_egress",
							"match":{"prefix":"/"},
							"route":{"cluster":"%s"},
							"request_headers_to_add": [{"header":{"key":"%s","value":"%s"},"append":false}]
                        }]
                    }`,
				portID, domainList, config.GetEgressCluster(), ServiceAddressKey, portID)

			virtualHosts = append(virtualHosts, vh)
		}

		if len(virtualHosts) == 0 {
			continue
		}

		patchValue := fmt.Sprintf(`{"virtual_hosts":[%s]}`, strings.Join(virtualHosts, ","))
		patch, err := utils.BuildPatchStruct(patchValue)
		if err != nil {
			// todo handle error
			continue
		}
		portPatch := &networking.EnvoyFilter_EnvoyConfigObjectPatch{
			ApplyTo: networking.EnvoyFilter_ROUTE_CONFIGURATION,
			Match: &networking.EnvoyFilter_EnvoyConfigObjectMatch{
				ObjectTypes: &networking.EnvoyFilter_EnvoyConfigObjectMatch_RouteConfiguration{
					RouteConfiguration: &networking.EnvoyFilter_RouteConfigurationMatch{
						Name: num,
					},
				},
			},
			Patch: &networking.EnvoyFilter_Patch{
				Operation: networking.EnvoyFilter_Patch_MERGE,
				Value:     patch,
			},
		}

		newPatches = append(newPatches, portPatch)
	}

	return newPatches
}

func (c *AggregationController) httpServicePortIndex() map[string][]*model.Service {
	index := make(map[string][]*model.Service)

	c.services.Range(func(key, value interface{}) bool {
		svc := value.(*model.Service)
		if svc.Namespace == config.IstioNamespace {
			return true
		}
		for port := range svc.Spec.HTTPPorts {
			index[port] = append(index[port], svc)
		}
		return true
	})

	// we want sorted
	for _, services := range index {
		sort.Slice(services, func(i, j int) bool {
			return services[i].Name > services[j].Name
		})
	}

	return index
}
