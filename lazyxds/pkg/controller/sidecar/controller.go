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

package sidecar

import (
	"context"
	"fmt"
	"github.com/aeraki-framework/aeraki/lazyxds/cmd/lazyxds/app/config"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils"
	istio "istio.io/client-go/pkg/apis/networking/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"time"

	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils/log"
	"github.com/go-logr/logr"
	istioinformers "istio.io/client-go/pkg/informers/externalversions/networking/v1alpha3"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	queue "k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2/klogr"

	networklister "istio.io/client-go/pkg/listers/networking/v1alpha3"
)

// Controller is responsible for synchronizing Istio sidecar objects.
type Controller struct {
	log                 logr.Logger
	lister              networklister.SidecarLister
	listerSynced        cache.InformerSynced
	queue               queue.RateLimitingInterface
	syncSidecarConfig   func(context.Context, *istio.Sidecar) error
	deleteSidecarConfig func(context.Context, string) error
}

// NewController creates a new sidecar controller
func NewController(
	informer istioinformers.SidecarInformer,
	syncSidecarConfig func(context.Context, *istio.Sidecar) error,
	deleteSidecarConfig func(context.Context, string) error,
) *Controller {
	logger := klogr.New().WithName("SidecarController")
	c := &Controller{
		log:                 logger,
		lister:              informer.Lister(),
		listerSynced:        informer.Informer().HasSynced,
		queue:               queue.NewNamedRateLimitingQueue(queue.DefaultControllerRateLimiter(), "Sidecar"),
		syncSidecarConfig:   syncSidecarConfig,
		deleteSidecarConfig: deleteSidecarConfig,
	}

	// only handle AddFunc, we don't automatically turn on lazyxds for namespaces or services
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.add,
		DeleteFunc: c.delete,
	})

	return c
}

func (c *Controller) add(obj interface{}) {
	sidecar, _ := obj.(*istio.Sidecar)

	// We only handle Sidecar configs which are not managed by LazyXds controller
	if c.isSidecarConfigManagedByLazyXds(sidecar) {
		return
	}

	c.log.V(4).Info("Adding Sidecar", "name", sidecar.Name)
	c.enqueue(sidecar)
}

func (c *Controller) delete(obj interface{}) {
	sidecar, ok := obj.(*istio.Sidecar)

	if c.isSidecarConfigManagedByLazyXds(sidecar) {
		return
	}

	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			c.log.Info("Couldn't get object from tombstone", "obj", obj)
			return
		}
		sidecar, ok = tombstone.Obj.(*istio.Sidecar)
		if !ok {
			c.log.Info("Tombstone contained object that is not a Service", "obj", obj)
			return
		}
	}
	c.log.V(4).Info("Deleting Sidecar", "name", sidecar.Name)
	c.enqueue(obj)
}

func (c *Controller) isSidecarConfigManagedByLazyXds(sidecar *istio.Sidecar) bool {
	for _, mangedFiled := range sidecar.ManagedFields {
		if mangedFiled.Manager == config.LazyXdsManager {
			return true
		}
	}
	return false
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

	c.log.Info("Starting Sidecar controller")
	defer c.log.Info("Shutting down Sidecar controller")

	if !cache.WaitForNamedCacheSync("Sidecar", stopCh, c.listerSynced) {
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
		logger.V(4).Info("Finished sync Sidecar", "duration", time.Since(startTime).String())
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	sidecar, err := c.lister.Sidecars(ns).Get(name)
	if err != nil && apierrors.IsNotFound(err) {
		logger.V(4).Info("Sidecar has been deleted")
		return c.deleteSidecarConfig(ctx, utils.ObjectID(name, ns))
	}
	if err != nil {
		return fmt.Errorf("unable to retrieve sidecar from store: error %w", err)
	}

	if !sidecar.DeletionTimestamp.IsZero() {
		return c.deleteSidecarConfig(ctx, utils.ObjectID(name, ns))
	}

	return c.syncSidecarConfig(ctx, sidecar)
}
