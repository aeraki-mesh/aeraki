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

package lazyservice

import (
	"context"
	"github.com/go-logr/logr"
	"k8s.io/klog/v2/klogr"
	"time"

	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils/log"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	queue "k8s.io/client-go/util/workqueue"
)

// Controller is responsible for synchronizing lazy service.
type Controller struct {
	log         logr.Logger
	queue       queue.RateLimitingInterface
	syncService func(context.Context, string) error
}

// NewController creates a new lazy service controller
func NewController(syncService func(context.Context, string) error) *Controller {
	logger := klogr.New().WithName("LazyServiceController")
	c := &Controller{
		log:         logger,
		queue:       queue.NewNamedRateLimitingQueue(queue.DefaultControllerRateLimiter(), "lazyservice"),
		syncService: syncService,
	}

	return c
}

// Add ...
func (c *Controller) Add(id string) {
	c.log.V(4).Info("Adding LazyService", "name", id)
	c.queue.Add(id)
}

// Run begins watching and syncing.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	c.log.Info("Starting LazyService controller")
	defer c.log.Info("Shutting down LazyService controller")

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
		logger.V(4).Info("Finished sync LazyService", "duration", time.Since(startTime).String())
	}()

	return c.syncService(ctx, key)
}
