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
	"bytes"
	"context"
	"reflect"
	"strconv"

	route "github.com/aeraki-mesh/meta-protocol-control-plane-api/aeraki/meta_protocol_proxy/config/route/v1alpha"
	metaprotocol "github.com/aeraki-mesh/meta-protocol-control-plane-api/aeraki/meta_protocol_proxy/v1alpha"
	"istio.io/istio/pilot/pkg/networking/util"

	metaprotocolmodel "github.com/aeraki-mesh/aeraki/pkg/model/metaprotocol"
	udpa "github.com/cncf/xds/go/udpa/type/v1"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/conversion"
	gogojsonpb "github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/types"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	networking "istio.io/api/networking/v1alpha3"
	istio "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// generate a listener on 0.0.0.0 for each service port to catch all outbound traffic and redirect it to the
// lazyxds gateway
func (c *Controller) syncCatchAllListenersForSidecars() {
	for port := range c.cache {
		err, envoyfilter := generateEnvoyfilterForCatchAllListener(port)
		if err != nil {
			controllerLog.Fatalf("failed to generate the catchall listener for port: %v %v", port, err)
		}
		old, err := c.istioClient.NetworkingV1alpha3().EnvoyFilters("istio-system").
			Get(context.TODO(), envoyfilter.Name, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			controllerLog.Infof("create envoyfilter for catch-all listener for port: : %v, %v", port, envoyfilter)
			_, err = c.istioClient.NetworkingV1alpha3().EnvoyFilters("istio-system").Create(context.TODO(), envoyfilter,
				metav1.CreateOptions{
					FieldManager: LazyXdsManager,
				})
			if err != nil {
				controllerLog.Errorf("failed to create envoyfilter for catch-all listener for port: %v, %v", port, err)
			}
		} else if err != nil {
			controllerLog.Errorf("failed to get envoyfilter for catch-all listener for port: %v, %v", port, err)
		} else if !reflect.DeepEqual(envoyfilter.Spec, old.Spec) {
			controllerLog.Infof("update envoyfilter for catch-all listener for port: : %v, %v", port, envoyfilter)
			envoyfilter.ResourceVersion = old.ResourceVersion
			_, err = c.istioClient.NetworkingV1alpha3().EnvoyFilters("istio-system").Update(context.TODO(), envoyfilter,
				metav1.UpdateOptions{
					FieldManager: LazyXdsManager,
				})
			if err != nil {
				controllerLog.Errorf("failed to update envoyfilter for catch-all listener for port: %v, %v", port, err)
			}
		}
	}
}

func generateEnvoyfilterForCatchAllListener(port uint32) (error, *istio.EnvoyFilter) {
	listener := generateCatchAllListenerForPort(port)
	value, err := message2struct(listener)
	if err != nil {
		return err, nil
	}
	listenerPatch := &networking.EnvoyFilter_EnvoyConfigObjectPatch{
		ApplyTo: networking.EnvoyFilter_LISTENER,
		Match: &networking.EnvoyFilter_EnvoyConfigObjectMatch{
			Context: networking.EnvoyFilter_SIDECAR_OUTBOUND,
		},
		Patch: &networking.EnvoyFilter_Patch{
			Operation: networking.EnvoyFilter_Patch_INSERT_FIRST,
			Value:     value,
		},
	}
	envoyfilter := &istio.EnvoyFilter{
		ObjectMeta: v1.ObjectMeta{
			Name:      "lazyxds-catch-all-listener-" + strconv.Itoa(int(port)),
			Namespace: "istio-system",
			Labels: map[string]string{
				ManagedByLabel: LazyXdsManager,
			},
		},
		Spec: networking.EnvoyFilter{
			ConfigPatches: []*networking.EnvoyFilter_EnvoyConfigObjectPatch{listenerPatch},
		},
	}
	return nil, envoyfilter
}

func generateCatchAllListenerForPort(port uint32) *listener.Listener {
	metaProtocolProxy := generateMetaProtocolProxy()
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

func generateMetaProtocolProxy() *metaprotocol.MetaProtocolProxy {
	applicationProtocol := "dubbo"
	codec, _ := metaprotocolmodel.GetApplicationProtocolCodec(applicationProtocol)
	metaProtocolProxy := &metaprotocol.MetaProtocolProxy{
		StatPrefix: "lazyxds_catch_all",
		RouteSpecifier: &metaprotocol.MetaProtocolProxy_RouteConfig{
			RouteConfig: &route.RouteConfiguration{
				Name: "lazyxds-gateway",
				Routes: []*route.Route{
					{
						Route: &route.RouteAction{
							ClusterSpecifier: &route.RouteAction_Cluster{Cluster: "istio-system.egressgateway"},
						},
					},
				},
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

func message2struct(listener proto.Message) (*types.Struct, error) {
	var buf []byte
	var err error
	if buf, err = protojson.Marshal(listener); err != nil {
		return nil, err
	}
	var value = &types.Struct{}
	if err := (&gogojsonpb.Unmarshaler{AllowUnknownFields: false}).Unmarshal(bytes.NewBuffer(buf), value); err != nil {
		return nil, err
	}
	return value, nil
}
