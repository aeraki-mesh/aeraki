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

package lazyxds

import (
	"context"
	"reflect"
	"strconv"

	"github.com/aeraki-mesh/aeraki/pkg/model"

	route "github.com/aeraki-mesh/meta-protocol-control-plane-api/aeraki/meta_protocol_proxy/config/route/v1alpha"
	metaprotocol "github.com/aeraki-mesh/meta-protocol-control-plane-api/aeraki/meta_protocol_proxy/v1alpha"
	v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"istio.io/istio/pilot/pkg/networking/util"
	istioconfig "istio.io/istio/pkg/config"

	metaprotocolmodel "github.com/aeraki-mesh/aeraki/pkg/model/metaprotocol"
	udpa "github.com/cncf/xds/go/udpa/type/v1"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/conversion"
	networking "istio.io/api/networking/v1alpha3"
	istio "istio.io/client-go/pkg/apis/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) sycGatewayService(ports map[uint32]map[string]*istioconfig.Config) {
	servicePorts := make([]corev1.ServicePort, 0)
	for port, _ := range ports {
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name: strconv.Itoa(int(port)),
			Port: int32(port),
		})
	}

	gatewaySvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lazyxds-gateway",
			Namespace: "istio-system",
		},
		Spec: corev1.ServiceSpec{
			Ports: servicePorts,
			Selector: map[string]string{
				"app": "lazyxds-gateway",
			},
		},
	}

	old, err := c.kubeClient.CoreV1().Services("istio-system").Get(context.TODO(), "lazyxds-gateway",
		metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		controllerLog.Infof("create service for lazyxds gateway ")
		c.kubeClient.CoreV1().Services("istio-system").Create(context.TODO(), gatewaySvc, metav1.CreateOptions{
			FieldManager: LazyXdsManager,
		})
	} else if err != nil {
		controllerLog.Errorf("failed to get lazyxds gateway service %v", err)
	} else if !reflect.DeepEqual(gatewaySvc.Spec, old.Spec) {
		controllerLog.Infof("update azyxds gateway service")
		gatewaySvc.ResourceVersion = old.ResourceVersion
		_, err = c.kubeClient.CoreV1().Services("istio-system").Update(context.TODO(), gatewaySvc,
			metav1.UpdateOptions{
				FieldManager: LazyXdsManager,
			})
		if err != nil {
			controllerLog.Errorf("failed to update azyxds gateway service %v", err)
		}
	}
}

// generate a listener on 0.0.0.0 for each service port and forward the request to the real destination service
// lazyxds gateway
func (c *Controller) syncGatewayListener(port uint32, services map[string]*istioconfig.Config) {
	err, envoyfilter := generateEnvoyfilterForGatewayListener(port, services)
	if err != nil {
		controllerLog.Fatalf("failed to generate the gateway listener for port: %v %v", port, err)
	}
	old, err := c.istioClient.NetworkingV1alpha3().EnvoyFilters("istio-system").
		Get(context.TODO(), envoyfilter.Name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		controllerLog.Infof("create envoyfilter for gateway listener for port: : %v, %v", port, envoyfilter)
		_, err = c.istioClient.NetworkingV1alpha3().EnvoyFilters("istio-system").Create(context.TODO(), envoyfilter,
			metav1.CreateOptions{
				FieldManager: LazyXdsManager,
			})
		if err != nil {
			controllerLog.Errorf("failed to create envoyfilter for gateway listener for port: %v, %v", port, err)
		}
	} else if err != nil {
		controllerLog.Errorf("failed to get envoyfilter for gateway listener for port: %v, %v", port, err)
	} else if !reflect.DeepEqual(envoyfilter.Spec, old.Spec) {
		controllerLog.Infof("update envoyfilter for gateway listener for port: : %v, %v", port, envoyfilter)
		envoyfilter.ResourceVersion = old.ResourceVersion
		_, err = c.istioClient.NetworkingV1alpha3().EnvoyFilters("istio-system").Update(context.TODO(), envoyfilter,
			metav1.UpdateOptions{
				FieldManager: LazyXdsManager,
			})
		if err != nil {
			controllerLog.Errorf("failed to update envoyfilter for gateway listener for port: %v, %v", port, err)
		}
	}
}

func generateEnvoyfilterForGatewayListener(port uint32, services map[string]*istioconfig.Config) (error, *istio.EnvoyFilter) {
	listener := generateGatewayForPort(port, services)
	value, err := message2struct(listener)
	if err != nil {
		return err, nil
	}
	listenerPatch := &networking.EnvoyFilter_EnvoyConfigObjectPatch{
		ApplyTo: networking.EnvoyFilter_LISTENER,
		Match: &networking.EnvoyFilter_EnvoyConfigObjectMatch{
			Context: networking.EnvoyFilter_GATEWAY,
		},
		Patch: &networking.EnvoyFilter_Patch{
			Operation: networking.EnvoyFilter_Patch_INSERT_FIRST,
			Value:     value,
		},
	}
	envoyfilter := &istio.EnvoyFilter{
		ObjectMeta: v1.ObjectMeta{
			Name:      "lazyxds-gateway-listener-" + strconv.Itoa(int(port)),
			Namespace: "istio-system",
			Labels: map[string]string{
				ManagedByLabel: LazyXdsManager,
			},
		},
		Spec: networking.EnvoyFilter{
			WorkloadSelector: &networking.WorkloadSelector{
				Labels: map[string]string{
					"app": "lazyxds-gateway",
				},
			},
			ConfigPatches: []*networking.EnvoyFilter_EnvoyConfigObjectPatch{listenerPatch},
		},
	}
	return nil, envoyfilter
}

func generateGatewayForPort(port uint32, services map[string]*istioconfig.Config) *listener.Listener {
	metaProtocolProxy := generateMetaProtocolProxyForGateway(port, services)
	proxy, _ := conversion.MessageToStruct(metaProtocolProxy)
	typedStruct := udpa.TypedStruct{
		TypeUrl: "type.googleapis.com/aeraki.meta_protocol_proxy.v1alpha.MetaProtocolProxy",
		Value:   proxy,
	}

	listener := &listener.Listener{
		Name: "lazyxds_catch_all_" + strconv.Itoa(int(port)),
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: port,
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{
			{
				Name: "lazyxds_catch_all",
				Filters: []*listener.Filter{
					{
						Name: "envoy.filters.network.meta_protocol_proxy",
						ConfigType: &listener.Filter_TypedConfig{
							TypedConfig: util.MessageToAny(&typedStruct),
						},
					},
				},
			},
		},
	}
	return listener
}

func generateMetaProtocolProxyForGateway(port uint32, services map[string]*istioconfig.Config) *metaprotocol.
	MetaProtocolProxy {
	applicationProtocol := "dubbo"
	codec, _ := metaprotocolmodel.GetApplicationProtocolCodec(applicationProtocol)
	routes := make([]*route.Route, 0)
	for host, _ := range services {
		clusterName := model.BuildClusterName(model.TrafficDirectionOutbound, "",
			host, int(port))
		routes = append(routes, &route.Route{
			Match: &route.RouteMatch{
				Metadata: []*v3.HeaderMatcher{
					{
						Name: "host",
						HeaderMatchSpecifier: &v3.HeaderMatcher_ExactMatch{
							ExactMatch: host,
						},
					},
				},
			},
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_Cluster{Cluster: clusterName},
			},
		})
	}
	metaProtocolProxy := &metaprotocol.MetaProtocolProxy{
		StatPrefix: "lazyxds_catch_all",
		RouteSpecifier: &metaprotocol.MetaProtocolProxy_RouteConfig{
			RouteConfig: &route.RouteConfiguration{
				Name:   "lazyxds-gateway",
				Routes: routes,
			},
		},
		ApplicationProtocol: applicationProtocol,
		Codec: &metaprotocol.Codec{
			Name: codec,
		},
		MetaProtocolFilters: []*metaprotocol.MetaProtocolFilter{
			{
				Name: "aeraki.meta_protocol.filters.router",
			},
		},
	}
	return metaProtocolProxy
}
