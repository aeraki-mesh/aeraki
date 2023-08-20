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

package kube

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	istionapi "istio.io/api/networking/v1alpha3"
	networking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"istio.io/pkg/log"
	"k8s.io/apimachinery/pkg/api/errors"
	controllerclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/aeraki-mesh/aeraki/internal/config/constants"
	"github.com/aeraki-mesh/aeraki/internal/model"
)

var serviceEntryLog = log.RegisterScope("service-entry-controller", "service-entry-controller debugging", 0)

// nolint: dupl
var (
	serviceEntryPredicates = predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			switch old := e.ObjectOld.(type) {
			case *networking.ServiceEntry:
				newSE, ok := e.ObjectNew.(*networking.ServiceEntry)
				if !ok {
					return false
				}
				if old.GetDeletionTimestamp() != newSE.GetDeletionTimestamp() ||
					old.GetGeneration() != newSE.GetGeneration() {
					return true
				}
			default:
				return false
			}
			return false
		},
	}
)

// serviceEntryController allocate VIPs to service entries
type serviceEntryController struct {
	controllerclient.Client
	serviceIPs map[string]controllerclient.ObjectKey
	maxIP      int
}

// Reconcile will try to trigger once mcp push.
func (c *serviceEntryController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	serviceEntryLog.Infof("reconcile: %s/%s", request.Namespace, request.Name)

	// Fetch the ReplicaSet from the cache
	se := &networking.ServiceEntry{}
	err := c.Get(ctx, request.NamespacedName, se)
	if errors.IsNotFound(err) {
		// The service entry has been deleted, the IP will be recycled when we reach 255*255
		return reconcile.Result{}, nil
	}

	if err != nil {
		return reconcile.Result{}, fmt.Errorf("could not fetch ServiceEntry: %+v", err)
	}

	c.autoAllocateIP(request.NamespacedName, se)
	return reconcile.Result{}, nil
}

// AddServiceEntryController adds serviceEntryController
func AddServiceEntryController(mgr manager.Manager) error {
	serviceEntryCtrl := &serviceEntryController{
		Client:     mgr.GetClient(),
		serviceIPs: make(map[string]controllerclient.ObjectKey),
	}
	c, err := controller.New("aeraki-service-entry-controller", mgr,
		controller.Options{Reconciler: serviceEntryCtrl})
	if err != nil {
		return err
	}
	// Watch for changes on ServiceEntry CRD
	err = c.Watch(source.Kind(mgr.GetCache(), &networking.ServiceEntry{}), &handler.EnqueueRequestForObject{},
		serviceEntryPredicates)
	if err != nil {
		return err
	}

	serviceEntryLog.Infof("ServiceEntryController (used to allocate VIP for Service Entry) registered")
	return nil
}

// Automatically allocates IPs for service entry services WITHOUT an address field when resolution is not NONE.
// The IPs are allocated from the reserved Class E subnet (240.240.0.0/16) that is not reachable outside the pod.
// When DNS capture is enabled, Envoy will resolve the DNS to these IPs.
// The listeners for TCP services will also be set up on these IPs.
func (c *serviceEntryController) autoAllocateIP(key controllerclient.ObjectKey, s *networking.ServiceEntry) {
	if s.Spec.Resolution == istionapi.ServiceEntry_NONE {
		return
	}

	// Check whether the VIP conflicts with existing SEs if this service entry already has one
	if len(s.Spec.Addresses) > 0 {
		// leave it as it is if the VIP is not in the CIDR range "240.240.0.0/16"
		if !strings.HasPrefix(s.Spec.Addresses[0], "240.240") {
			return
		}

		if existingKey, ok := c.serviceIPs[s.Spec.Addresses[0]]; ok {
			// update the vip if it's conflicted with an existing ServiceEntry
			if !reflect.DeepEqual(existingKey, key) {
				err := c.Get(context.TODO(), existingKey, &networking.ServiceEntry{})
				if err == nil {
					serviceEntryLog.Infof("update conflicting vip for serviceEntry %s", s)
					s.Spec.Addresses[0] = c.nextAvailableIP()
					c.updateServiceEntry(s, key)
				} else if errors.IsNotFound(err) {
					c.serviceIPs[s.Spec.Addresses[0]] = key
				} else {
					serviceEntryLog.Errorf("failed to get serviceEntry: %v", err)
				}
			}
		} else {
			// store the vip and serviceEntry in the map
			c.serviceIPs[s.Spec.Addresses[0]] = key
		}
	} else {
		s.Spec.Addresses = []string{c.nextAvailableIP()}
		c.updateServiceEntry(s, key)
	}
}

func (c *serviceEntryController) updateServiceEntry(s *networking.ServiceEntry, key controllerclient.ObjectKey) {
	err := c.Client.Update(context.TODO(), s,
		&controllerclient.UpdateOptions{
			FieldManager: constants.AerakiFieldManager,
		})
	if err == nil {
		c.serviceIPs[s.Spec.Addresses[0]] = key
		serviceEntryLog.Infof("allocate vip for serviceEntry %v", model.Struct2JSON(s))
	} else {
		serviceEntryLog.Errorf("failed to update serviceEntry %s, error: %v", s.Name, err)
	}
}

func (c *serviceEntryController) nextAvailableIP() string {
	for {
		nextIP := c.getNextIP()
		key, exists := c.serviceIPs[nextIP]
		if exists {
			err := c.Get(context.TODO(), key, &networking.ServiceEntry{})
			if err != nil {
				if errors.IsNotFound(err) {
					delete(c.serviceIPs, nextIP)
					return nextIP
				}
				serviceEntryLog.Errorf("failed to get serviceEntry: %v", err)
			}
		} else {
			return nextIP
		}
	}
}

func (c *serviceEntryController) getNextIP() string {
	if c.maxIP%255 == 0 {
		c.maxIP++
	}
	if c.maxIP >= 255*255 {
		serviceEntryLog.Errorf("out of IPs to allocate for service entries, restart from 0")
		c.maxIP = 0
	}
	thirdOctet := c.maxIP / 255
	fourthOctet := c.maxIP % 255
	c.maxIP++
	return fmt.Sprintf("240.240.%d.%d", thirdOctet, fourthOctet)
}
