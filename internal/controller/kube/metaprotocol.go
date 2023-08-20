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

	"github.com/aeraki-mesh/client-go/pkg/apis/metaprotocol/v1alpha1"
	"istio.io/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	metaprotocolmodel "github.com/aeraki-mesh/aeraki/internal/model/metaprotocol"
)

var metaProtocolLog = log.RegisterScope("meta-protocol-controller", "meta-protocol-controller debugging", 0)

// nolint: dupl
var (
	metaProtocolPredicates = predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			switch old := e.ObjectOld.(type) {
			case *v1alpha1.ApplicationProtocol:
				newAP, ok := e.ObjectNew.(*v1alpha1.ApplicationProtocol)
				if !ok {
					return false
				}
				if old.GetDeletionTimestamp() != newAP.GetDeletionTimestamp() ||
					old.GetGeneration() != newAP.GetGeneration() {
					return true
				}
			default:
				return false
			}
			return false
		},
	}
)

// MetaProtocolController control ApplicationProtocol
type MetaProtocolController struct {
	client.Client
	triggerPush func() error
}

// Reconcile will try to trigger once mcp push.
func (r *MetaProtocolController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	metaProtocolLog.Infof("reconcile: %s/%s", request.Namespace, request.Name)
	protocol := &v1alpha1.ApplicationProtocol{}
	err := r.Get(ctx, request.NamespacedName, protocol)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	metaProtocolLog.Debugf("register application protocol : %s, codec: %s", protocol.Spec.Protocol, protocol.Spec.Codec)
	metaprotocolmodel.SetApplicationProtocolCodec(protocol.Spec.Protocol, protocol.Spec.Codec)

	if r.triggerPush != nil {
		err := r.triggerPush()
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
	}
	return reconcile.Result{}, nil
}

// AddApplicationProtocolController adds ApplicationProtocolController
func AddApplicationProtocolController(mgr manager.Manager, triggerPush func() error) error {
	metaProtocolCtrl := &MetaProtocolController{Client: mgr.GetClient(), triggerPush: triggerPush}
	c, err := controller.New("aeraki-meta-protocol-application-protocol-controller", mgr,
		controller.Options{Reconciler: metaProtocolCtrl})
	if err != nil {
		return err
	}
	// Watch for changes to primary resource IstioFilter
	err = c.Watch(source.Kind(mgr.GetCache(), &v1alpha1.ApplicationProtocol{}), &handler.EnqueueRequestForObject{},
		metaProtocolPredicates)
	if err != nil {
		return err
	}
	controllerLog.Infof("MetaProtocolApplicationProtocolController registered")
	return nil
}
