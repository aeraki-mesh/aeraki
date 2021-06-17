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

package endpoints

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils/log"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	queue "k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2/klogr"
)

// Controller is responsible for synchronizing endpoints objects.
type Controller struct {
	clusterName     string
	log             logr.Logger
	getter          v1.EndpointsGetter
	lister          corelisters.EndpointsLister
	listerSynced    cache.InformerSynced
	queue           queue.RateLimitingInterface
	syncEndpoints   func(context.Context, *corev1.Endpoints) error
	deleteEndpoints func(context.Context, string, string) error
}

// NewController creates a new endpoints controller
func NewController(
	clusterName string,
	getter v1.EndpointsGetter,
	informer coreinformers.EndpointsInformer,
	sync func(context.Context, *corev1.Endpoints) error,
	delete func(context.Context, string, string) error,
) *Controller {
	logger := klogr.New().WithName("EndpointsController")
	c := &Controller{
		clusterName:     clusterName,
		log:             logger,
		getter:          getter,
		lister:          informer.Lister(),
		listerSynced:    informer.Informer().HasSynced,
		queue:           queue.NewNamedRateLimitingQueue(queue.DefaultControllerRateLimiter(), "endpoints"),
		syncEndpoints:   sync,
		deleteEndpoints: delete,
	}

	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.add,
		UpdateFunc: c.update,
		DeleteFunc: c.delete,
	})

	return c
}

func (c *Controller) add(obj interface{}) {
	endpoints, _ := obj.(*corev1.Endpoints)
	c.log.V(4).Info("Adding Endpoints", "name", endpoints.Name)
	c.enqueue(endpoints)
}

func (c *Controller) update(oldObj, curObj interface{}) {
	old, _ := oldObj.(*corev1.Endpoints)
	current, _ := curObj.(*corev1.Endpoints)
	if !c.needsUpdate(old, current) {
		return
	}

	c.log.V(4).Info("Updating Endpoints", "name", current.Name)
	c.enqueue(current)
}

func (c *Controller) needsUpdate(old *corev1.Endpoints, new *corev1.Endpoints) bool {
	return !reflect.DeepEqual(old.Subsets, new.Subsets) || new.GetDeletionTimestamp() != nil
}

func (c *Controller) delete(obj interface{}) {
	endpoints, ok := obj.(*corev1.Endpoints)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			c.log.Info("Couldn't get object from tombstone", "obj", obj)
			return
		}
		endpoints, ok = tombstone.Obj.(*corev1.Endpoints)
		if !ok {
			c.log.Info("Tombstone contained object that is not a Endpoints", "obj", obj)
			return
		}
	}
	c.log.V(4).Info("Deleting Endpoints", "name", endpoints.Name)
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

	c.log.Info("Starting Endpoints controller")
	defer c.log.Info("Shutting down Endpoints controller")

	if !cache.WaitForNamedCacheSync("Endpoints", stopCh, c.listerSynced) {
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
		logger.V(4).Info("Finished sync endpoints", "duration", time.Since(startTime).String())
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	endpoints, err := c.lister.Endpoints(ns).Get(name)
	if err != nil && apierrors.IsNotFound(err) {
		logger.V(4).Info("Endpoints has been deleted")
		// todo may need delay for a while, because access log may be late
		return c.deleteEndpoints(ctx, name, ns)
	}
	if err != nil {
		return fmt.Errorf("unable to retrieve endpoints from store: error %w", err)
	}

	return c.syncEndpoints(ctx, endpoints)
}
