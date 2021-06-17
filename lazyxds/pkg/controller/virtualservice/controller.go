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

package virtualservice

import (
	"context"
	"fmt"
	"github.com/aeraki-framework/aeraki/lazyxds/cmd/lazyxds/app/config"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils"
	istio "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"reflect"
	"time"

	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils/log"
	"github.com/go-logr/logr"
	istioinformers "istio.io/client-go/pkg/informers/externalversions/networking/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	queue "k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2/klogr"

	networklister "istio.io/client-go/pkg/listers/networking/v1alpha3"
)

// Controller is responsible for synchronizing virtualService objects.
type Controller struct {
	log                  logr.Logger
	lister               networklister.VirtualServiceLister
	listerSynced         cache.InformerSynced
	queue                queue.RateLimitingInterface
	syncVirtualService   func(context.Context, *istio.VirtualService) error
	deleteVirtualService func(context.Context, string) error
}

// NewController creates a new virtualService controller
func NewController(
	informer istioinformers.VirtualServiceInformer,
	syncVirtualService func(context.Context, *istio.VirtualService) error,
	deleteVirtualService func(context.Context, string) error,
) *Controller {
	logger := klogr.New().WithName("VirtualServiceController")
	c := &Controller{
		log:                  logger,
		lister:               informer.Lister(),
		listerSynced:         informer.Informer().HasSynced,
		queue:                queue.NewNamedRateLimitingQueue(queue.DefaultControllerRateLimiter(), "virtualService"),
		syncVirtualService:   syncVirtualService,
		deleteVirtualService: deleteVirtualService,
	}

	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.add,
		UpdateFunc: c.update,
		DeleteFunc: c.delete,
	})

	return c
}

func (c *Controller) add(obj interface{}) {
	virtualService, _ := obj.(*istio.VirtualService)
	c.log.V(4).Info("Adding VirtualService", "name", virtualService.Name)
	c.enqueue(virtualService)
}

func (c *Controller) update(oldObj, curObj interface{}) {
	old, _ := oldObj.(*istio.VirtualService)
	current, _ := curObj.(*istio.VirtualService)
	if !c.needsUpdate(old, current) {
		return
	}

	c.log.V(4).Info("Updating VirtualService", "name", current.Name)
	c.enqueue(current)
}

func (c *Controller) needsUpdate(old *istio.VirtualService, new *istio.VirtualService) bool {
	return !reflect.DeepEqual(old.Spec, new.Spec) || new.GetDeletionTimestamp() != nil
}

func (c *Controller) delete(obj interface{}) {
	virtualService, ok := obj.(*istio.VirtualService)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			c.log.Info("Couldn't get object from tombstone", "obj", obj)
			return
		}
		virtualService, ok = tombstone.Obj.(*istio.VirtualService)
		if !ok {
			c.log.Info("Tombstone contained object that is not a VirtualService", "obj", obj)
			return
		}
	}
	c.log.V(4).Info("Deleting VirtualService", "name", virtualService.Name)
	c.enqueue(obj)
}

func (c *Controller) enqueue(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.queue.Add(key)
}

// Run begins watching and syncing.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	c.log.Info("Starting VirtualService controller")
	defer c.log.Info("Shutting down VirtualService controller")

	if !cache.WaitForNamedCacheSync("VirtualService", stopCh, c.listerSynced) {
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(c.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (c *Controller) worker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	logger := c.log.WithValues("key", key)
	ctx := log.WithContext(context.Background(), logger)
	err := c.syncFromKey(ctx, key.(string))
	if err != nil {
		c.queue.AddRateLimited(key)
		logger.Error(err, "Sync error")
		return true
	}

	c.queue.Forget(key)
	logger.Info("Successfully synced")

	return true
}

func (c *Controller) syncFromKey(ctx context.Context, key string) error {
	startTime := time.Now()
	logger := log.FromContext(ctx)
	logger.V(4).Info("Starting sync")
	defer func() {
		logger.V(4).Info("Finished sync virtualService", "duration", time.Since(startTime).String())
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if ns == config.IstioNamespace && name == config.EgressVirtualServiceName {
		return nil
	}
	if err != nil {
		return err
	}

	virtualService, err := c.lister.VirtualServices(ns).Get(name)
	if err != nil && apierrors.IsNotFound(err) {
		logger.V(4).Info("VirtualService has been deleted")
		return c.deleteVirtualService(ctx, utils.FQDN(name, ns))
	}
	if err != nil {
		return fmt.Errorf("unable to retrieve virtualService from store: error %w", err)
	}
	if !virtualService.DeletionTimestamp.IsZero() {
		return c.deleteVirtualService(ctx, utils.FQDN(name, ns))
	}

	return c.syncVirtualService(ctx, virtualService)
}
