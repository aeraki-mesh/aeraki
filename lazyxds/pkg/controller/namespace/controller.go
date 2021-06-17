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

package namespace

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

// Controller is responsible for synchronizing namespace objects.
type Controller struct {
	clusterName     string
	log             logr.Logger
	getter          v1.NamespacesGetter
	lister          corelisters.NamespaceLister
	listerSynced    cache.InformerSynced
	queue           queue.RateLimitingInterface
	syncNamespace   func(context.Context, string, *corev1.Namespace) error
	deleteNamespace func(context.Context, string, string) error
}

// NewController creates a new namespace controller
func NewController(
	clusterName string,
	getter v1.NamespacesGetter,
	informer coreinformers.NamespaceInformer,
	sync func(context.Context, string, *corev1.Namespace) error,
	delete func(context.Context, string, string) error,
) *Controller {
	logger := klogr.New().WithName("NamespaceController")
	c := &Controller{
		clusterName:     clusterName,
		log:             logger,
		getter:          getter,
		lister:          informer.Lister(),
		listerSynced:    informer.Informer().HasSynced,
		queue:           queue.NewNamedRateLimitingQueue(queue.DefaultControllerRateLimiter(), "namespace"),
		syncNamespace:   sync,
		deleteNamespace: delete,
	}

	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.add,
		UpdateFunc: c.update,
		DeleteFunc: c.delete,
	})

	return c
}

func (c *Controller) add(obj interface{}) {
	namespace, _ := obj.(*corev1.Namespace)
	c.log.V(4).Info("Adding Namespace", "name", namespace.Name)
	c.enqueue(namespace)
}

func (c *Controller) update(oldObj, curObj interface{}) {
	old, _ := oldObj.(*corev1.Namespace)
	current, _ := curObj.(*corev1.Namespace)
	if !c.needsUpdate(old, current) {
		return
	}

	c.log.V(4).Info("Updating Namespace", "name", current.Name)
	c.enqueue(current)
}

func (c *Controller) needsUpdate(old *corev1.Namespace, new *corev1.Namespace) bool {
	return !reflect.DeepEqual(old.Annotations, new.Annotations) || new.GetDeletionTimestamp() != nil
}

func (c *Controller) delete(obj interface{}) {
	namespace, ok := obj.(*corev1.Namespace)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			c.log.Info("Couldn't get object from tombstone", "obj", obj)
			return
		}
		namespace, ok = tombstone.Obj.(*corev1.Namespace)
		if !ok {
			c.log.Info("Tombstone contained object that is not a Namespace", "obj", obj)
			return
		}
	}
	c.log.V(4).Info("Deleting Namespace", "name", namespace.Name)
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

	c.log.Info("Starting Namespace controller")
	defer c.log.Info("Shutting down Namespace controller")

	if !cache.WaitForNamedCacheSync("Namespace", stopCh, c.listerSynced) {
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
		logger.V(4).Info("Finished sync namespace", "duration", time.Since(startTime).String())
	}()
	name := key

	namespace, err := c.lister.Get(name)
	if err != nil && apierrors.IsNotFound(err) {
		logger.V(4).Info("Namespace has been deleted")
		return c.deleteNamespace(ctx, c.clusterName, name)
	}
	if err != nil {
		return fmt.Errorf("unable to retrieve namespace from store: error %w", err)
	}
	if !namespace.DeletionTimestamp.IsZero() {
		return c.deleteNamespace(ctx, c.clusterName, name)
	}

	return c.syncNamespace(ctx, c.clusterName, namespace)
}
