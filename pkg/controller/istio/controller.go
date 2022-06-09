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

package istio

import (
	"io/ioutil"
	"reflect"
	"strings"
	"time"

	"github.com/aeraki-mesh/aeraki/pkg/model"

	"istio.io/istio/pkg/security"

	"istio.io/istio/security/pkg/nodeagent/cache"

	"github.com/aeraki-mesh/aeraki/pkg/model/protocol"
	"github.com/cenkalti/backoff"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/config/memory"
	istiomodel "istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/adsc"
	istioconfig "istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/collection"
	"istio.io/istio/pkg/config/schema/collections"
	citadel "istio.io/istio/security/pkg/nodeagent/caclient/providers/citadel"
	"istio.io/pkg/log"
)

const (
	// istiodCACertPath is the ca volume mount file name for istio root ca.
	istiodCACertPath = "/var/run/secrets/istio/root-cert.pem"

	// K8sSAJwtFileName is the token volume mount file name for k8s jwt token.
	K8sSAJwtFileName = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

var (
	controllerLog = log.RegisterScope("config-controller", "config-controller debugging", 0)
	// We need serviceentry and virtualservice to generate the envoyfiters
	configCollection = collection.NewSchemasBuilder().MustAdd(collections.IstioNetworkingV1Alpha3Serviceentries).
				MustAdd(collections.IstioNetworkingV1Alpha3Virtualservices).
				MustAdd(collections.IstioNetworkingV1Alpha3Destinationrules).
				MustAdd(collections.IstioNetworkingV1Alpha3Envoyfilters).Build()
)

// Options for config controller
type Options struct {
	ClusterID  string
	NameSpace  string
	IstiodAddr string
}

// Controller watches Istio config xDS server and notifies the listeners when config changes.
type Controller struct {
	options     *Options
	xdsMCP      *adsc.ADSC
	Store       istiomodel.ConfigStore
	configCache istiomodel.ConfigStoreCache
}

// NewController creates a new Controller instance based on the provided arguments.
func NewController(options *Options) *Controller {
	store := memory.Make(configCollection)
	return &Controller{
		options:     options,
		Store:       store,
		configCache: memory.NewController(store),
	}
}

// Run until a signal is received, this function won't block
func (c *Controller) Run(stop <-chan struct{}) {
	go c.configCache.Run(stop)
	go func() {
		c.connectIstio()
		for {
			time.Sleep(30 * time.Minute)
			c.reconnectIstio()
		}
	}()
}

func (c *Controller) reconnectIstio() {
	controllerLog.Info("Close connection to Istio MCP over xDS server")
	c.closeConnection()
	c.connectIstio()
	controllerLog.Info("Reconnect to Istio MCP over xDS server")
}

func (c *Controller) closeConnection() {
	c.xdsMCP.Close()
}

func (c *Controller) connectIstio() {
	var err error

	config := adsc.Config{
		Namespace: c.options.NameSpace,
		Meta: istiomodel.NodeMetadata{
			Generator: "api",
			// Currently we use clusterId="" to indicates that the result should not be filtered by the proxy
			//location.
			// For example, all the VIPs of the clusters should be included in the addresses of a service entry.
			// https://github.com/istio/istio/pull/36820
			ClusterID: "",
		}.ToStruct(),
		InitialDiscoveryRequests: c.configInitialRequests(),
		BackoffPolicy:            backoff.NewConstantBackOff(time.Second),
	}

	for {
		// we assume it's tls if istiod port is not 15010
		if !strings.HasSuffix(c.options.IstiodAddr, ":15010") {
			sm, err := c.newSecretManager()
			if err != nil {
				controllerLog.Errorf("failed to create SecretManager %s %v", c.options.IstiodAddr, err)
			} else {
				config.SecretManager = sm
			}
		}
		c.xdsMCP, err = adsc.New(c.options.IstiodAddr, &config)
		if err != nil {
			controllerLog.Errorf("failed to dial XDS %s %v", c.options.IstiodAddr, err)
			time.Sleep(5 * time.Second)
			continue
		}
		c.xdsMCP.Store = istiomodel.MakeIstioStore(c.configCache)
		if err = c.xdsMCP.Run(); err != nil {
			controllerLog.Errorf("adsc: failed running %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		return
	}
}

func (c *Controller) configInitialRequests() []*discovery.DiscoveryRequest {
	schemas := configCollection.All()
	requests := make([]*discovery.DiscoveryRequest, len(schemas))
	for i, schema := range schemas {
		requests[i] = &discovery.DiscoveryRequest{
			TypeUrl: schema.Resource().GroupVersionKind().String(),
		}
	}
	return requests
}

// RegisterEventHandler adds a handler to receive config update events for a configuration type
func (c *Controller) RegisterEventHandler(handler func(*istioconfig.Config, *istioconfig.Config, istiomodel.Event)) {
	handlerWrapper := func(prev istioconfig.Config, curr istioconfig.Config, event istiomodel.Event) {
		if event == istiomodel.EventUpdate && reflect.DeepEqual(prev.Spec, curr.Spec) {
			return
		}
		// We care about these resources:
		// * ServiceEntry: Services handled by Aeraki: tcp-metaprotocol, tcp-dubbo, tcp-thrift, tcp-redis
		// * VirtualService: Route rules for dubbo and thrift
		// * DestinationRule: the Load balancing policy in set in the dr,
		//   httpHeaderName is used to convey the metadata key for generating hash
		if curr.GroupVersionKind == collections.IstioNetworkingV1Alpha3Serviceentries.Resource().GroupVersionKind() {
			//controllerLog.Infof("Service Entry changed: %s %s", event.String(), curr.Name)
			if c.shouldHandleSeChange(curr) {
				handler(&prev, &curr, event)
			} else if c.shouldHandleSeChange(prev) {
				handler(&prev, &curr, event)
			}
		} else if curr.GroupVersionKind == collections.IstioNetworkingV1Alpha3Virtualservices.Resource().GroupVersionKind() {
			controllerLog.Infof("virtual service changed: %s %s", event.String(), curr.Name)
			if c.shouldHandleVsChange(curr) {
				handler(&prev, &curr, event)
			} else if c.shouldHandleVsChange(prev) {
				handler(&prev, &curr, event)
			}
		} else if curr.GroupVersionKind == collections.IstioNetworkingV1Alpha3Destinationrules.Resource().GroupVersionKind() {
			controllerLog.Infof("Destination rules changed: %s %s", event.String(), curr.Name)
			if c.shouldHandleDrChange(curr) {
				handler(&prev, &curr, event)
			} else if c.shouldHandleDrChange(prev) {
				handler(&prev, &curr, event)
			}
		}
	}

	schemas := configCollection.All()
	for _, schema := range schemas {
		c.configCache.RegisterEventHandler(schema.Resource().GroupVersionKind(), handlerWrapper)
	}
}

func (c *Controller) shouldHandleSeChange(seConfig istioconfig.Config) bool {
	service, ok := seConfig.Spec.(*networking.ServiceEntry)
	if !ok {
		// This should never happen
		controllerLog.Errorf("failed in getting a service entry: %v", seConfig.Name)
		return false
	}
	for _, port := range service.Ports {
		if protocol.IsAerakiSupportedProtocols(port.Name) {
			return true
		}
	}
	return false
}

func (c *Controller) shouldHandleVsChange(vsConfig istioconfig.Config) bool {
	vs, ok := vsConfig.Spec.(*networking.VirtualService)
	if !ok {
		// This should never happen
		controllerLog.Errorf("failed in getting a virtual service: %v", vsConfig.Name)
		return false
	}
	serviceEntries, err := c.Store.List(collections.IstioNetworkingV1Alpha3Serviceentries.Resource().GroupVersionKind(), "")
	if err != nil {
		controllerLog.Errorf("failed to list configs: %v", err)
		return false
	}
	for _, se := range serviceEntries {
		service, ok := se.Spec.(*networking.ServiceEntry)
		if !ok { // should never happen
			controllerLog.Errorf("failed in getting a service entry: %s: %v", se.Name, err)
			return false
		}
		if len(vs.Hosts) > 0 {
			for _, host := range service.Hosts {
				if model.IsFQDNEquals(host, se.Namespace, vs.Hosts[0], vsConfig.Namespace) {
					for _, port := range service.Ports {
						if protocol.IsAerakiSupportedProtocols(port.Name) {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func (c *Controller) shouldHandleDrChange(drConfig istioconfig.Config) bool {
	// We only care about the Load balancing policy in the dr,
	// httpHeaderName is used to convey the metadata key for generating hash
	dr, ok := drConfig.Spec.(*networking.DestinationRule)
	if !ok {
		// This should never happen
		controllerLog.Errorf("failed in getting a destination rule: %s", drConfig.Name)
		return false
	}
	serviceEntries, err := c.Store.List(collections.IstioNetworkingV1Alpha3Serviceentries.Resource().GroupVersionKind(), "")
	if err != nil {
		controllerLog.Errorf("failed to list configs: %v", err)
		return false
	}
	for _, se := range serviceEntries {
		service, ok := se.Spec.(*networking.ServiceEntry)
		if !ok { // should never happen
			controllerLog.Errorf("failed in getting a service entry: %s: %v", se.Name, err)
			return false
		}

		for _, host := range service.Hosts {
			if model.IsFQDNEquals(host, se.Namespace, dr.Host, drConfig.Namespace) {
				for _, port := range service.Ports {
					if protocol.IsAerakiSupportedProtocols(port.Name) {
						return true
					}
				}
			}
		}
	}
	return false
}

func (c *Controller) newSecretManager() (*cache.SecretManagerClient, error) {
	var rootCert []byte
	var err error

	if rootCert, err = ioutil.ReadFile(istiodCACertPath); err != nil {
		log.Fatalf("invalid config -  missing a root certificate %s", istiodCACertPath)
	}

	// Will use TLS unless the reserved 15010 port is used ( istiod on an ipsec/secure VPC)
	// rootCert may be nil - in which case the system roots are used, and the CA is expected to have public key
	// Otherwise assume the injection has mounted /etc/certs/root-cert.pem
	o := &security.Options{
		CAEndpoint:        c.options.IstiodAddr,
		ClusterID:         c.options.ClusterID,
		JWTPath:           K8sSAJwtFileName,
		WorkloadNamespace: c.options.NameSpace,
		TrustDomain:       "cluster.local",
		ServiceAccount:    "aeraki",
	}

	caClient, err := citadel.NewCitadelClient(o, true, rootCert)
	if err != nil {
		return nil, err
	}

	return cache.NewSecretManagerClient(caClient, o)
}
