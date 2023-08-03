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

package model

import (
	metaprotocol "github.com/aeraki-mesh/client-go/pkg/apis/metaprotocol/v1alpha1"
	networking "istio.io/api/networking/v1alpha3"
	istioconfig "istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/mesh"
)

// ServiceEntryWrapper wraps an Istio ServiceEntry and its metadata, including name, annotations and labels.
// Meta can be used to pass in some additional information that is needed for EnvoyFilter generation. For example,
// we use an "interface" annotation to pass the dubbo interface to the Dubbo generator.
type ServiceEntryWrapper struct {
	istioconfig.Meta
	Spec *networking.ServiceEntry
}

// GatewayWrapper wraps an Istio Gateway and its metadata, including name, annotations and labels.
type GatewayWrapper struct {
	istioconfig.Meta
	Spec *networking.Gateway
}

// VirtualServiceWrapper wraps an Istio VirtualService and its metadata, including name, annotations and labels.
type VirtualServiceWrapper struct {
	istioconfig.Meta
	Spec *networking.VirtualService
}

// DestinationRuleWrapper wraps an Istio DestinationRule and its metadata, including name, annotations and labels.
type DestinationRuleWrapper struct {
	istioconfig.Meta
	Spec *networking.DestinationRule
}

// EnvoyFilterWrapper wraps an Istio EnvoyFilterWrapper and its name, which is used as an unique identifier in Istio.
// If two Envoyfilters with the same name have been created, the previous one sill be replaced by the latter one
type EnvoyFilterWrapper struct {
	Name        string
	Namespace   string
	Envoyfilter *networking.EnvoyFilter
}

// EnvoyFilterContext provides an aggregate API for EnvoyFilter generator
type EnvoyFilterContext struct {

	// Global Mesh config
	MeshConfig mesh.Holder

	// Gateway describes the gateway for which we need to generate the EnvoyFilter.
	// ServiceEntry will be ignored when this field specified.
	Gateway *GatewayWrapper

	// ServiceEntry describes the service for which we need to generate the EnvoyFilter.
	ServiceEntry *ServiceEntryWrapper

	// VirtualService is the related VirtualService of the ServiceEntry, which defines the routing rules for the service.
	// Only one VirtualService is allowed for a Service.
	// The value of VirtualService is nil in case that no VirtualService defined for the service.
	VirtualService *VirtualServiceWrapper

	// VirtualService is the related VirtualService of the ServiceEntry, which defines the routing rules for the service.
	// Only one VirtualService is allowed for a Service.
	// The value of VirtualService is nil in case that no VirtualService defined for the service.
	MetaRouter *metaprotocol.MetaRouter
}
