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

	"github.com/aeraki-mesh/client-go/pkg/apis/redis/v1alpha1"
	"istio.io/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var redisLog = log.RegisterScope("redis-controller", "redis-controller debugging", 0)

// RedisController control RedisService or RedisDestination
type RedisController struct {
	triggerPush func() error
}

// Reconcile will try to trigger once mcp push.
func (r *RedisController) Reconcile(_ context.Context, request reconcile.Request) (reconcile.Result, error) {
	redisLog.Infof("reconcile: %s/%s", request.Namespace, request.Name)
	if r.triggerPush != nil {
		err := r.triggerPush()
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
	}
	return reconcile.Result{}, nil
}

// AddRedisServiceController adds RedisServiceController
func AddRedisServiceController(mgr manager.Manager, triggerPush func() error) error {
	redisCtrl := &RedisController{triggerPush: triggerPush}
	c, err := controller.New("aeraki-redis-service-controller", mgr, controller.Options{Reconciler: redisCtrl})
	if err != nil {
		return err
	}
	// Watch for changes to primary resource IstioFilter
	err = c.Watch(source.Kind(mgr.GetCache(), &v1alpha1.RedisService{}),
		&handler.EnqueueRequestForObject{}, redisPredicates)
	if err != nil {
		return err
	}
	controllerLog.Infof("RedisServiceController registered")
	return nil
}

// AddRedisDestinationController adds RedisDestinationControlle
func AddRedisDestinationController(mgr manager.Manager, triggerPush func() error) error {
	redisCtrl := &RedisController{triggerPush: triggerPush}
	c, err := controller.New("aeraki-redis-destination-controller", mgr, controller.Options{Reconciler: redisCtrl})
	if err != nil {
		return err
	}
	// Watch for changes to primary resource IstioFilter
	err = c.Watch(source.Kind(mgr.GetCache(), &v1alpha1.RedisDestination{}),
		&handler.EnqueueRequestForObject{}, redisPredicates)
	if err != nil {
		return err
	}
	controllerLog.Infof("RedisDestinationController registered")
	return nil
}

var (
	redisPredicates = predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			switch old := e.ObjectOld.(type) {
			case *v1alpha1.RedisService:
				newRS, ok := e.ObjectNew.(*v1alpha1.RedisService)
				if !ok {
					return false
				}
				if old.GetDeletionTimestamp() != newRS.GetDeletionTimestamp() ||
					old.GetGeneration() != newRS.GetGeneration() {
					return true
				}
			case *v1alpha1.RedisDestination:
				newRD, ok := e.ObjectNew.(*v1alpha1.RedisDestination)
				if !ok {
					return false
				}
				if old.GetDeletionTimestamp() != newRD.GetDeletionTimestamp() ||
					old.GetGeneration() != newRD.GetGeneration() {
					return true
				}
			default:
				return false
			}
			return false
		},
	}
)
