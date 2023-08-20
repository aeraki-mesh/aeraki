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

	"github.com/aeraki-mesh/client-go/pkg/apis/dubbo/v1alpha1"
	"istio.io/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var dubboLog = log.RegisterScope("dubbo-controller", "dubbo-controller debugging", 0)

// DubboController control DubboAuthorizationPolicy
type DubboController struct {
	triggerPush func() error
}

// Reconcile will try to trigger once mcp push.
func (r *DubboController) Reconcile(_ context.Context, request reconcile.Request) (reconcile.Result, error) {
	dubboLog.Infof("reconcile: %s/%s", request.Namespace, request.Name)
	if r.triggerPush != nil {
		err := r.triggerPush()
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
	}
	return reconcile.Result{}, nil
}

// AddDubboAuthorizationPolicyController adds DubboAuthorizationPolicyController
func AddDubboAuthorizationPolicyController(mgr manager.Manager, triggerPush func() error) error {
	dubboCtrl := &DubboController{triggerPush: triggerPush}
	c, err := controller.New("aeraki-dubbo-authorization-policy-controller", mgr,
		controller.Options{Reconciler: dubboCtrl})
	if err != nil {
		return err
	}
	// Watch for changes to primary resource IstioFilter
	err = c.Watch(source.Kind(mgr.GetCache(), &v1alpha1.DubboAuthorizationPolicy{}),
		&handler.EnqueueRequestForObject{}, dubboPredicates)
	if err != nil {
		return err
	}
	controllerLog.Infof("DubboAuthorizationPolicyController registered")
	return nil
}

// nolint: dupl
var (
	dubboPredicates = predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			switch old := e.ObjectOld.(type) {
			case *v1alpha1.DubboAuthorizationPolicy:
				newDA, ok := e.ObjectNew.(*v1alpha1.DubboAuthorizationPolicy)
				if !ok {
					return false
				}
				if old.GetDeletionTimestamp() != newDA.GetDeletionTimestamp() ||
					old.GetGeneration() != newDA.GetGeneration() {
					return true
				}
			default:
				return false
			}
			return false
		},
	}
)
