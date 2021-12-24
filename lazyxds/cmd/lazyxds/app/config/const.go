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

package config

import "fmt"

const (
	// IstioNamespace is the default root namespace of istio
	IstioNamespace = "istio-system"
	// LazyXdsManager is the controller name which will put into fieldManager
	LazyXdsManager = "lazyxds-manager"
	// EgressName is name of egress deployment and service
	EgressName = "istio-egressgateway-lazyxds"
	// EgressServicePort is default port of egress service
	EgressServicePort = 8080
	// EgressGatewayName is the istio gateway name of lazyxds egress
	EgressGatewayName = "lazyxds-egress"
	// AccessLogServicePort is the default port of access log service
	AccessLogServicePort = 8080
	// EgressVirtualServiceName the vs name of lazyxds egress
	EgressVirtualServiceName = "lazyxds-egress"
	// LazyLoadingAnnotation is the annotation name which use to enable/disable lazy xds feature
	LazyLoadingAnnotation = "lazy-xds"
	// ManagedByLabel is the common label indicate the component is managed by which controller
	ManagedByLabel = "app.kubernetes.io/managed-by"
)

// GetEgressCluster returns the egress xds cluster string
// default is "outbound|8080||istio-egressgateway-lazyxds.istio-system.svc.cluster.local"
func GetEgressCluster() string {
	return fmt.Sprintf("outbound|%d||%s.%s.svc.cluster.local",
		EgressServicePort,
		EgressName,
		IstioNamespace,
	)
}
