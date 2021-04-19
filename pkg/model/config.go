package model

import (
	networking "istio.io/api/networking/v1alpha3"
	istioconfig "istio.io/istio/pkg/config"
)

// ServiceEntryWrapper wraps an Istio ServiceEntry and its metadata, including name, annotations and labels.
// Meta can be used to pass in some additional information that is needed for EnvoyFilter generation. For example,
// we use an "interface" annotation to pass the dubbo interface to the Dubbo generator.
type ServiceEntryWrapper struct {
	istioconfig.Meta
	Spec *networking.ServiceEntry
}

// VirtualServiceWrapper wraps an Istio VirtualService and its metadata, including name, annotations and labels.
type VirtualServiceWrapper struct {
	istioconfig.Meta
	Spec *networking.VirtualService
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

	// ServiceEntry describes the service for which we need to generate the EnvoyFilter.
	ServiceEntry *ServiceEntryWrapper

	// VirtualService is the related VirtualService of the ServiceEntry, which defines the routing rules for the service.
	// Only one VirtualService is allowed for a Service.
	// The value of VirtualService is nil in case that no VirtualService defined for the service.
	VirtualService *VirtualServiceWrapper
}
