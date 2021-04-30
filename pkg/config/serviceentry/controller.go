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

package serviceentry

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/aeraki-framework/aeraki/pkg/model"
	istionapi "istio.io/api/networking/v1alpha3"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	"istio.io/istio/pkg/kube"
	"istio.io/pkg/log"
)

const (
	maxRetries         = 5
	aerakiFieldManager = "Aeraki"
)

var controllerLog = log.RegisterScope("serviceEntry-controller", "serviceEntry-controller debugging", 0)

// Controller is the controller implementation for serviceEntry resources
type Controller struct {
	istioClientset *istioclient.Clientset
	queue          workqueue.RateLimitingInterface
	informer       cache.SharedIndexInformer
	serviceIPs     map[string]string
	maxIP          int
}

// NewController returns a new serviceEntry controller
func NewController(
	istioClientset *istioclient.Clientset) *Controller {
	serviceEntryInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(opts meta_v1.ListOptions) (runtime.Object, error) {
				return istioClientset.NetworkingV1alpha3().ServiceEntries("").
					List(context.
						TODO(), opts)
			},
			WatchFunc: func(opts meta_v1.ListOptions) (watch.Interface, error) {
				return istioClientset.NetworkingV1alpha3().ServiceEntries("").Watch(context.TODO(), opts)
			},
		},
		&istionetworking.ServiceEntry{}, 0, cache.Indexers{},
	)

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	controller := &Controller{
		istioClientset: istioClientset,
		informer:       serviceEntryInformer,
		queue:          queue,
		serviceIPs:     make(map[string]string),
	}

	controllerLog.Info("Setting up event handlers")
	serviceEntryInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			controllerLog.Infof("Processing add: %s", key)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if oldObj == newObj || reflect.DeepEqual(oldObj, newObj) {
				return
			}

			key, err := cache.MetaNamespaceKeyFunc(newObj)
			controllerLog.Infof("Processing update: %s", key)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			controllerLog.Infof("Processing delete: %s", key)
			if err == nil {
				queue.Add(key)
			}
		},
	})

	return controller
}

// Run starts the controller until it receives a message over stopCh
func (c *Controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	go c.informer.Run(stopCh)

	// Wait for the caches to be synced before starting workers
	if !kube.WaitForCacheSyncInterval(stopCh, time.Millisecond*100, c.informer.HasSynced) {
		return
	}

	go wait.Until(c.runWorker, 5*time.Second, stopCh)
	<-stopCh
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.processItem(key.(string))
	if err == nil {
		// No error, reset the ratelimit counters
		c.queue.Forget(key)
	} else if c.queue.NumRequeues(key) < maxRetries {
		controllerLog.Errorf("Error processing %s (will retry): %v", key, err)
		c.queue.AddRateLimited(key)
	} else {
		controllerLog.Errorf("Error processing %s (giving up): %v", key, err)
		c.queue.Forget(key)
		utilruntime.HandleError(err)
	}

	return true
}

func (c *Controller) processItem(key string) error {
	obj, exists, err := c.informer.GetIndexer().GetByKey(key)
	if err != nil {
		return fmt.Errorf("error fetching object %s error: %v", key, err)
	}
	if exists {
		c.autoAllocateIP(key, obj.(*istionetworking.ServiceEntry))
	}

	return nil
}

// Automatically allocates IPs for service entry services WITHOUT an address field when resolution is not NONE.
// The IPs are allocated from the reserved Class E subnet (240.240.0.0/16) that is not reachable outside the pod.
// When DNS capture is enabled, Envoy will resolve the DNS to these IPs.
// The listeners for TCP services will also be set up on these IPs.
func (c *Controller) autoAllocateIP(key string, s *istionetworking.ServiceEntry) {
	if s.Spec.Resolution == istionapi.ServiceEntry_NONE {
		return
	}
	if len(s.Spec.Addresses) > 0 {
		// leave it as it is if the VIP is not in the CIDR range "240.240.0.0/16"
		if !strings.HasPrefix(s.Spec.Addresses[0], "240.240") {
			return
		}

		if name, ok := c.serviceIPs[s.Spec.Addresses[0]]; ok {
			//update the vip if it's conflicted with an existing ServiceEntry
			if name != key {
				_, exists, err := c.informer.GetStore().GetByKey(name)
				if err != nil {
					controllerLog.Errorf("failed to get serviceEntry from informer local store: %v", err)
				} else if !exists {
					c.serviceIPs[s.Spec.Addresses[0]] = key
				} else {
					controllerLog.Infof("update conflicting vip for serviceEntry %s", s)
					s.Spec.Addresses[0] = c.nextAvailableIP()
					c.updateServiceEntry(s, key)
				}
			}
		} else {
			c.serviceIPs[s.Spec.Addresses[0]] = key
		}
	} else {
		s.Spec.Addresses = []string{c.nextAvailableIP()}
		c.updateServiceEntry(s, key)
	}
}

func (c *Controller) updateServiceEntry(s *istionetworking.ServiceEntry, key string) {
	_, err := c.istioClientset.NetworkingV1alpha3().ServiceEntries(s.Namespace).Update(context.TODO(), s,
		meta_v1.UpdateOptions{
			FieldManager: aerakiFieldManager,
		})
	if err != nil {
		controllerLog.Errorf("failed to update serviceEntry %s, error: %v", s.Name, err)
	} else {
		c.serviceIPs[s.Spec.Addresses[0]] = key
		controllerLog.Infof("allocate vip for serviceEntry %v", model.Struct2JSON(s))
	}
}

func (c *Controller) nextAvailableIP() string {
	for {
		nextIP := c.getNextIP()
		key, exists := c.serviceIPs[nextIP]
		if exists {
			_, exists, err := c.informer.GetStore().GetByKey(key)
			if err != nil {
				controllerLog.Errorf("failed to get serviceEntry from informer local store: %v", err)
			} else if !exists { //Release this IP if the serviceEntry has already been deleted
				delete(c.serviceIPs, key)
				return nextIP
			}
		} else {
			return nextIP
		}
	}
}

func (c *Controller) getNextIP() string {
	if c.maxIP%255 == 0 {
		c.maxIP++
	}
	if c.maxIP >= 255*255 {
		controllerLog.Errorf("out of IPs to allocate for service entries, restart from 0")
		c.maxIP = 0
	}
	thirdOctet := c.maxIP / 255
	fourthOctet := c.maxIP % 255
	c.maxIP++
	return fmt.Sprintf("240.240.%d.%d", thirdOctet, fourthOctet)
}
