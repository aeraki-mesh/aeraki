package model

import (
	networking "istio.io/api/networking/v1alpha3"
	istioconfig "istio.io/istio/pkg/config"
)

type ServiceEntryWrapper struct {
	istioconfig.Meta
	Spec *networking.ServiceEntry
}
