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

package serviceentry

import (
	"context"
	"fmt"
	istio "istio.io/client-go/pkg/apis/networking/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"reflect"
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

// LazyXdsManager ...
const LazyXdsManager = "lazyxds"

// Controller is responsible for synchronizing Istio serviceEntry objects.
type Controller struct {
	log                logr.Logger
	lister             networklister.ServiceEntryLister
	listerSynced       cache.InformerSynced
	queue              queue.RateLimitingInterface
	syncServiceEntry   func(context.Context, *istio.ServiceEntry) error
	deleteServiceEntry func(context.Context, string, string) error
}

// NewController creates a new serviceEntry controller
func NewController(
	informer istioinformers.ServiceEntryInformer,
	syncServiceEntryConfig func(context.Context, *istio.ServiceEntry) error,
	deleteServiceEntryConfig func(context.Context, string, string) error,
) *Controller {
	logger := klogr.New().WithName("ServiceEntryController")
	c := &Controller{
		log:                logger,
		lister:             informer.Lister(),
		listerSynced:       informer.Informer().HasSynced,
		queue:              queue.NewNamedRateLimitingQueue(queue.DefaultControllerRateLimiter(), "ServiceEntry"),
		syncServiceEntry:   syncServiceEntryConfig,
		deleteServiceEntry: deleteServiceEntryConfig,
	}

	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.add,
		UpdateFunc: c.update,
		DeleteFunc: c.delete,
	})

	return c
}

func (c *Controller) add(obj interface{}) {
	serviceEntry, _ := obj.(*istio.ServiceEntry)
	c.log.V(4).Info("Adding ServiceEntry", "name", serviceEntry.Name)
	c.enqueue(serviceEntry)
}

func (c *Controller) update(oldObj, curObj interface{}) {
	old, _ := oldObj.(*istio.ServiceEntry)
	current, _ := curObj.(*istio.ServiceEntry)
	if !c.needsUpdate(old, current) {
		return
	}

	c.log.V(4).Info("Updating ServiceEntry", "name", current.Name)
	c.enqueue(current)
}

func (c *Controller) needsUpdate(old *istio.ServiceEntry, new *istio.ServiceEntry) bool {
	return !reflect.DeepEqual(old.Spec, new.Spec) || new.GetDeletionTimestamp() != nil
}

func (c *Controller) delete(obj interface{}) {
	serviceEntry, ok := obj.(*istio.ServiceEntry)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			c.log.Info("Couldn't get object from tombstone", "obj", obj)
			return
		}
		serviceEntry, ok = tombstone.Obj.(*istio.ServiceEntry)
		if !ok {
			c.log.Info("Tombstone contained object that is not a Service", "obj", obj)
			return
		}
	}
	c.log.V(4).Info("Deleting ServiceEntry", "name", serviceEntry.Name)
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

	c.log.Info("Starting ServiceEntry controller")
	defer c.log.Info("Shutting down ServiceEntry controller")

	if !cache.WaitForNamedCacheSync("ServiceEntry", stopCh, c.listerSynced) {
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
		logger.V(4).Info("Finished sync ServiceEntry", "duration", time.Since(startTime).String())
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	serviceEntry, err := c.lister.ServiceEntries(ns).Get(name)
	if err != nil && apierrors.IsNotFound(err) {
		logger.V(4).Info("ServiceEntry has been deleted")
		return c.deleteServiceEntry(ctx, name, ns)
	}
	if err != nil {
		return fmt.Errorf("unable to retrieve serviceentry from store: error %w", err)
	}

	if !serviceEntry.DeletionTimestamp.IsZero() {
		logger.V(4).Info("ServiceEntry has been deleted, should not be here")
		return c.deleteServiceEntry(ctx, name, ns)
	}

	return c.syncServiceEntry(ctx, serviceEntry)
}
