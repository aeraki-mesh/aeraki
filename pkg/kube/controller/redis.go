package controller

import (
	"context"
	"reflect"

	"github.com/aeraki-framework/aeraki/client-go/pkg/apis/redis/v1alpha1"
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
func (r *RedisController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	redisLog.Infof("reconcile: %s/%s", request.Namespace, request.Name)
	if r.triggerPush != nil {
		err := r.triggerPush()
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
	}
	return reconcile.Result{}, nil
}

func addRedisServiceController(mgr manager.Manager, triggerPush func() error) error {
	redisCtrl := &RedisController{triggerPush: triggerPush}
	c, err := controller.New("aeraki-redis-service-controller", mgr, controller.Options{Reconciler: redisCtrl})
	if err != nil {
		return err
	}
	// Watch for changes to primary resource IstioFilter
	err = c.Watch(&source.Kind{Type: &v1alpha1.RedisService{}}, &handler.EnqueueRequestForObject{}, redisPredicates)
	if err != nil {
		return err
	}
	controllerLog.Infof("RedisServiceController registered")
	return nil
}

func addRedisDestinationController(mgr manager.Manager, triggerPush func() error) error {
	redisCtrl := &RedisController{triggerPush: triggerPush}
	c, err := controller.New("aeraki-redis-destination-controller", mgr, controller.Options{Reconciler: redisCtrl})
	if err != nil {
		return err
	}
	// Watch for changes to primary resource IstioFilter
	err = c.Watch(&source.Kind{Type: &v1alpha1.RedisDestination{}}, &handler.EnqueueRequestForObject{}, redisPredicates)
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
			switch oldFilter := e.ObjectOld.(type) {
			case *v1alpha1.RedisService:
				newFilter, ok := e.ObjectNew.(*v1alpha1.RedisService)
				if !ok {
					return false
				}
				if !reflect.DeepEqual(oldFilter.Spec, newFilter.Spec) ||
					oldFilter.GetDeletionTimestamp() != newFilter.GetDeletionTimestamp() ||
					oldFilter.GetGeneration() != newFilter.GetGeneration() {
					return true
				}
			case *v1alpha1.RedisDestination:
				newFilter, ok := e.ObjectNew.(*v1alpha1.RedisDestination)
				if !ok {
					return false
				}
				if !reflect.DeepEqual(oldFilter.Spec, newFilter.Spec) ||
					oldFilter.GetDeletionTimestamp() != newFilter.GetDeletionTimestamp() ||
					oldFilter.GetGeneration() != newFilter.GetGeneration() {
					return true
				}
			default:
				return false
			}
			return false
		},
	}
)
