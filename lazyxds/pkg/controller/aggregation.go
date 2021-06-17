/*
 * // Copyright Aeraki Authors
 * //
 * // Licensed under the Apache License, Version 2.0 (the "License");
 * // you may not use this file except in compliance with the License.
 * // You may obtain a copy of the License at
 * //
 * //     http://www.apache.org/licenses/LICENSE-2.0
 * //
 * // Unless required by applicable law or agreed to in writing, software
 * // distributed under the License is distributed on an "AS IS" BASIS,
 * // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * // See the License for the specific language governing permissions and
 * // limitations under the License.
 */

package controller

import (
	"context"
	"fmt"
	"github.com/aeraki-framework/aeraki/lazyxds/cmd/lazyxds/app/config"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/controller/endpoints"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/controller/lazyservice"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/controller/namespace"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/controller/service"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/controller/serviceentry"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/controller/sidecar"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/controller/virtualservice"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/model"
	"github.com/go-logr/logr"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	istioinformer "istio.io/client-go/pkg/informers/externalversions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	"sync"
)

const (
	// ResourcePrefix ...
	ResourcePrefix = "lazyxds-"
	// EgressGatewayFullName ...
	EgressGatewayFullName = config.IstioNamespace + "/" + config.EgressGatewayName
	// ServiceAddressKey ...
	ServiceAddressKey = "lazyxds-service-address"
)

// AggregationController ...
type AggregationController struct {
	log  logr.Logger
	stop <-chan struct{}

	KubeClient    *kubernetes.Clientset // todo need remove
	istioClient   *istioclient.Clientset
	istioInformer istioinformer.SharedInformerFactory
	multiCluster  map[string]*Cluster

	namespaceController *namespace.Controller
	serviceController   *service.Controller
	endpointsController *endpoints.Controller

	virtualServiceController *virtualservice.Controller
	sidecarController        *sidecar.Controller
	serviceEntryController   *serviceentry.Controller
	lazyServiceController    *lazyservice.Controller

	services     sync.Map // {svcID: *model.svc}
	lazyServices map[string]*model.Service

	namespaces sync.Map
	endpoints  sync.Map

	serviceEntries      sync.Map
	httpServicesBinding sync.Map // {vsID: {svcID set}}
	tcpServicesBinding  sync.Map // {vsID: {svcID set}}
}

// NewController ...
func NewController(istioClient *istioclient.Clientset, stop <-chan struct{}) *AggregationController {
	c := &AggregationController{
		log:           klogr.New().WithName("AggregationController"),
		istioClient:   istioClient,
		istioInformer: istioinformer.NewSharedInformerFactory(istioClient, 0),
		stop:          stop,
		multiCluster:  make(map[string]*Cluster),
		lazyServices:  make(map[string]*model.Service),
	}

	c.virtualServiceController = virtualservice.NewController(
		c.istioInformer.Networking().V1alpha3().VirtualServices(),
		c.syncVirtualService,
		c.deleteVirtualService,
	)

	c.sidecarController = sidecar.NewController(
		c.istioInformer.Networking().V1alpha3().Sidecars(),
		c.syncSidecar,
		c.deleteSidecar,
	)

	c.serviceEntryController = serviceentry.NewController(
		c.istioInformer.Networking().V1alpha3().ServiceEntries(),
		c.syncServiceEntry,
		c.deleteServiceEntry,
	)

	c.lazyServiceController = lazyservice.NewController(
		c.syncLazyService,
	)

	return c
}

// AddCluster ...
func (c *AggregationController) AddCluster(name string, client *kubernetes.Clientset) error {
	if _, ok := c.multiCluster[name]; ok {
		return fmt.Errorf("cluster %s already exists", name)
	}
	cluster := NewCluster(name, client)
	c.multiCluster[name] = cluster

	c.namespaceController = namespace.NewController(
		name,
		client.CoreV1(),
		cluster.Informer.Core().V1().Namespaces(),
		c.syncNamespace,
		c.deleteNamespace,
	)

	c.serviceController = service.NewController(
		name,
		client.CoreV1(),
		cluster.Informer.Core().V1().Services(),
		c.syncService,
		c.deleteService,
	)

	c.endpointsController = endpoints.NewController(
		name,
		client.CoreV1(),
		cluster.Informer.Core().V1().Endpoints(),
		c.syncEndpoints,
		c.deleteEndpoints,
	)

	klog.Info("Starting Namespace controller", "cluster", name)
	go c.namespaceController.Run(2, c.stop)

	klog.Info("Starting Service controller", "cluster", name)
	go c.serviceController.Run(4, c.stop)

	klog.Infof("Starting Endpoints controller", "cluster", name)
	go c.endpointsController.Run(4, c.stop)

	cluster.Informer.Start(c.stop)
	return nil
}

// Run ...
func (c *AggregationController) Run() {
	go c.virtualServiceController.Run(2, c.stop)
	go c.sidecarController.Run(2, c.stop)
	go c.serviceEntryController.Run(2, c.stop)
	go c.lazyServiceController.Run(4, c.stop)

	c.istioInformer.Start(c.stop)
}

// ClusterClient ...
func (c *AggregationController) ClusterClient(name string) *kubernetes.Clientset {
	cluster := c.multiCluster[name]
	if cluster == nil {
		return nil
	}
	return cluster.Client
}

// HandleAccess ...
func (c *AggregationController) HandleAccess(fromIP, svcID, toIP string) error {
	c.log.Info("HandleAccess", "fromIP", fromIP, "svcID", svcID, "toIP", toIP)

	fromSvcID := c.IP2ServiceID(fromIP)
	if fromSvcID == "" {
		return nil
	}

	lazySvc, ok := c.lazyServices[fromSvcID]
	if !ok {
		return nil
	}

	if svcID == "" {
		svcID := c.IP2ServiceID(toIP)
		if svcID == "" {
			return nil
		}
	}

	c.log.Info("Add service to egress of lazyservice", "fromService", fromSvcID, "toService", svcID)
	lazySvc.EgressService[svcID] = struct{}{}

	return c.reconcileLazyService(context.TODO(), lazySvc)
}
