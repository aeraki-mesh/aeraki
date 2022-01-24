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

package multicluster

import (
	"context"
	"fmt"
	"github.com/aeraki-mesh/aeraki/lazyxds/cmd/lazyxds/app/config"
	"github.com/aeraki-mesh/aeraki/lazyxds/pkg/utils/log"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	queue "k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2/klogr"
	"reflect"
	"time"
)

// Controller is responsible for synchronizing multiCluster secrets.
type Controller struct {
	log           logr.Logger
	informer      cache.SharedIndexInformer
	listerSynced  cache.InformerSynced
	queue         queue.RateLimitingInterface
	syncCluster   func(context.Context, *corev1.Secret) error
	deleteCluster func(context.Context, string) error
}

// NewController creates a new multiCluster controller
func NewController(
	kubeClient *kubernetes.Clientset,
	syncSecret func(context.Context, *corev1.Secret) error,
	deleteSecret func(context.Context, string) error,
) *Controller {
	logger := klogr.New().WithName("MultiClusterController")

	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(opts meta_v1.ListOptions) (runtime.Object, error) {
				opts.LabelSelector = config.MultiClusterSecretLabel + "=true"
				obj, err := kubeClient.CoreV1().Secrets(config.IstioNamespace).List(context.TODO(), opts)
				return obj, err
			},
			WatchFunc: func(opts meta_v1.ListOptions) (watch.Interface, error) {
				opts.LabelSelector = config.MultiClusterSecretLabel + "=true"
				return kubeClient.CoreV1().Secrets(config.IstioNamespace).Watch(context.TODO(), opts)
			},
		},
		&corev1.Secret{}, 0, cache.Indexers{},
	)

	c := &Controller{
		log:           logger,
		informer:      informer,
		listerSynced:  informer.HasSynced,
		queue:         queue.NewNamedRateLimitingQueue(queue.DefaultControllerRateLimiter(), "secret"),
		syncCluster:   syncSecret,
		deleteCluster: deleteSecret,
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.add,
		UpdateFunc: c.update,
		DeleteFunc: c.delete,
	})

	return c
}

func (c *Controller) add(obj interface{}) {
	secret, _ := obj.(*corev1.Secret)
	c.log.V(4).Info("Adding Secret", "name", secret.Name)
	c.enqueue(secret)
}

func (c *Controller) update(oldObj, curObj interface{}) {
	old, _ := oldObj.(*corev1.Secret)
	current, _ := curObj.(*corev1.Secret)
	if !c.needsUpdate(old, current) {
		return
	}

	c.log.V(4).Info("Updating Secret", "name", current.Name)
	c.enqueue(current)
}

func (c *Controller) needsUpdate(old *corev1.Secret, new *corev1.Secret) bool {
	return !reflect.DeepEqual(old.Data, new.Data) || !reflect.DeepEqual(old.Labels, new.Labels) || new.GetDeletionTimestamp() != nil
}

func (c *Controller) delete(obj interface{}) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			c.log.Info("Couldn't get object from tombstone", "obj", obj)
			return
		}
		secret, ok = tombstone.Obj.(*corev1.Secret)
		if !ok {
			c.log.Info("Tombstone contained object that is not a Service", "obj", obj)
			return
		}
	}
	c.log.V(4).Info("Deleting Secret", "name", secret.Name)
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

	c.log.Info("Starting secret controller")
	go c.informer.Run(stopCh)
	log.Info("Waiting for informer caches to sync")
	defer c.log.Info("Shutting down Secret controller")

	if !cache.WaitForNamedCacheSync("Secret", stopCh, c.listerSynced) {
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
		logger.V(4).Info("Finished sync Secret", "duration", time.Since(startTime).String())
	}()

	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	//secret, err := c.lister.Secrets(ns).Get(name)
	obj, exists, err := c.informer.GetIndexer().GetByKey(key)
	if err != nil && apierrors.IsNotFound(err) {
		logger.V(4).Info("Secret has been deleted")
		return c.deleteCluster(ctx, name)
	}
	if err != nil {
		return fmt.Errorf("unable to retrieve secret from store: error %w", err)
	}
	if exists {
		secret := obj.(*corev1.Secret)
		return c.syncCluster(ctx, secret)
	} else {
		return c.deleteCluster(ctx, name)
	}

}
