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
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/controller/discoveryselector"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/controller/multicluster"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"sync"

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

	KubeClient    *kubernetes.Clientset
	istioClient   *istioclient.Clientset
	istioInformer istioinformer.SharedInformerFactory
	kubeInformer  informers.SharedInformerFactory
	multiCluster  map[string]*Cluster

	namespaceController map[string]*namespace.Controller
	serviceController   map[string]*service.Controller
	endpointsController map[string]*endpoints.Controller

	virtualServiceController *virtualservice.Controller
	sidecarController        *sidecar.Controller
	serviceEntryController   *serviceentry.Controller
	lazyServiceController    *lazyservice.Controller
	configMapController      *discoveryselector.Controller
	multiClusterController   *multicluster.Controller

	// all services of all k8s clusters
	services sync.Map // format: {svcID: *model.svc}
	// all lazy services
	lazyServices map[string]*model.Service
	// istio discovery namespace selector
	discoverySelectors []labels.Selector

	namespaces sync.Map
	endpoints  sync.Map

	serviceEntries      sync.Map
	httpServicesBinding sync.Map // {vsID: {svcID set}}
	tcpServicesBinding  sync.Map // {vsID: {svcID set}}
}

// NewController ...
func NewController(istioClient *istioclient.Clientset, kubeClient *kubernetes.Clientset, stop <-chan struct{}) *AggregationController {
	c := &AggregationController{
		log:                 klogr.New().WithName("AggregationController"),
		istioClient:         istioClient,
		KubeClient:          kubeClient,
		istioInformer:       istioinformer.NewSharedInformerFactory(istioClient, 0),
		kubeInformer:        informers.NewSharedInformerFactory(kubeClient, 0),
		stop:                stop,
		multiCluster:        make(map[string]*Cluster),
		lazyServices:        make(map[string]*model.Service),
		namespaceController: make(map[string]*namespace.Controller),
		serviceController:   make(map[string]*service.Controller),
		endpointsController: make(map[string]*endpoints.Controller),
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

	c.configMapController = discoveryselector.NewController(
		c.KubeClient,
		c.updateDiscoverySelector,
		c.reconcileAllNamespaces,
	)

	c.multiClusterController = multicluster.NewController(
		c.kubeInformer.Core().V1().Secrets(),
		c.syncCluster,
		c.deleteCluster,
	)

	return c
}

// AddCluster ...
func (c *AggregationController) AddCluster(name string, client *kubernetes.Clientset) error {
	if _, ok := c.multiCluster[name]; ok {
		return fmt.Errorf("cluster %s already exists", name)
	}
	stopCh := make(chan struct{})
	cluster := NewCluster(name, client, stopCh)
	c.multiCluster[name] = cluster

	namespaceController := namespace.NewController(
		name,
		client.CoreV1(),
		cluster.Informer.Core().V1().Namespaces(),
		c.syncNamespace,
		c.deleteNamespace,
	)

	serviceController := service.NewController(
		name,
		client.CoreV1(),
		cluster.Informer.Core().V1().Services(),
		c.syncService,
		c.deleteService,
	)

	endpointsController := endpoints.NewController(
		name,
		client.CoreV1(),
		cluster.Informer.Core().V1().Endpoints(),
		c.syncEndpoints,
		c.deleteEndpoints,
	)

	c.namespaceController[name] = namespaceController
	c.serviceController[name] = serviceController
	c.endpointsController[name] = endpointsController

	klog.Info("Starting Namespace controller", "cluster", name)
	go namespaceController.Run(2, stopCh)

	klog.Info("Starting Service controller", "cluster", name)
	go serviceController.Run(4, stopCh)

	klog.Infof("Starting Endpoints controller", "cluster", name)
	go endpointsController.Run(4, stopCh)

	cluster.Informer.Start(stopCh)
	return nil
}

// DeleteCluster ...
func (c *AggregationController) DeleteCluster(name string) error {
	cluster, ok := c.multiCluster[name]
	if !ok {
		klog.Infof("DeleteCluster: cluster not exists, clusterName=%s", name)
		return nil
	}
	close(cluster.stopCh)

	c.services.Range(func(key, value interface{}) bool {
		svc := value.(*model.Service)
		if _, ok := svc.Distribution[name]; !ok {
			return true
		}
		err := c.deleteService(context.TODO(), name, utils.FQDN(svc.Name, svc.Namespace))
		if err != nil {
			klog.Errorf("Delete service error:%v", err)
		}
		return true
	})

	c.namespaces.Range(func(key, value interface{}) bool {
		ns := value.(*model.Namespace)
		if _, ok := ns.Distribution[name]; !ok {
			return true
		}
		err := c.deleteNamespace(context.TODO(), name, ns.Name)
		if err != nil {
			klog.Errorf("Delete namespace error:%v", err)
		}
		return true
	})

	c.endpoints.Range(func(key, value interface{}) bool {
		ep := value.(*model.Endpoints)
		if ep.Cluster == name {
			err := c.deleteEndpoints(context.TODO(), ep.Name, ep.Namespace)
			if err != nil {
				klog.Errorf("Delete endpoint error:%v", err)
			}
		}
		return true
	})

	delete(c.multiCluster, name)
	return nil
}

// Run ...
func (c *AggregationController) Run() {
	go c.virtualServiceController.Run(2, c.stop)
	go c.sidecarController.Run(2, c.stop)
	go c.serviceEntryController.Run(2, c.stop)
	go c.lazyServiceController.Run(4, c.stop)
	go c.configMapController.Run(c.stop)
	go c.multiClusterController.Run(2, c.stop)

	c.kubeInformer.Start(c.stop)
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
