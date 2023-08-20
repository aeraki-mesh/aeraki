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

	"istio.io/pkg/log"
	v1 "k8s.io/api/core/v1"
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
)

var namespaceLog = log.RegisterScope("namespace-controller", "namespace-controller debugging", 0)

var (
	namespacePredicates = predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return true
		},
	}
)

// namespaceController creates bootstrap configMap for sidecar proxies
type namespaceController struct {
	controllerclient.Client
	AerakiAddr string
	AerakiPort string
}

// Reconcile watch namespace change and create bootstrap configmap for sidecar proxies
func (c *namespaceController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	namespaceLog.Infof("reconcile: %s", request.Name)

	ns := &v1.Namespace{}
	err := c.Get(ctx, request.NamespacedName, ns)
	if errors.IsNotFound(err) {
		return reconcile.Result{}, nil
	}

	if err != nil {
		return reconcile.Result{}, fmt.Errorf("could not fetch Namespace: %+v", err)
	}

	if c.shouldHandle(ns) {
		c.createBootstrapConfigMap(ns.Name)
	}
	return reconcile.Result{}, nil
}

// AddNamespaceController adds namespaceController
func AddNamespaceController(mgr manager.Manager, aerakiAddr, aerakiPort string) error {
	namespaceCtrl := &namespaceController{
		Client:     mgr.GetClient(),
		AerakiAddr: aerakiAddr,
		AerakiPort: aerakiPort,
	}
	c, err := controller.New("aeraki-namespace-controller", mgr,
		controller.Options{Reconciler: namespaceCtrl})
	if err != nil {
		return err
	}
	// Watch for changes on Namespace CRD
	err = c.Watch(source.Kind(mgr.GetCache(), &v1.Namespace{}), &handler.EnqueueRequestForObject{},
		namespacePredicates)
	if err != nil {
		return err
	}

	namespaceLog.Infof("NamespaceController (used to create bootstrap configMap for sidecar proxies) registered")
	return nil
}

func (c *namespaceController) createBootstrapConfigMap(ns string) {
	cm := &v1.ConfigMap{}
	cm.Name = "aeraki-bootstrap-config"
	cm.Namespace = ns
	cm.Data = map[string]string{
		"custom_bootstrap.json": GetBootstrapConfig(c.AerakiAddr, c.AerakiPort),
	}
	if err := c.Client.Create(context.TODO(), cm, &controllerclient.CreateOptions{
		FieldManager: constants.AerakiFieldManager,
	}); err != nil {
		if !errors.IsAlreadyExists(err) {
			namespaceLog.Errorf("failed to create configMap: %v", err)
		}
	}
}
func (c *namespaceController) shouldHandle(ns *v1.Namespace) bool {
	if ns.Labels["istio-injection"] == "enabled" || ns.Labels["istio.io/rev"] != "" {
		return true
	}
	return false
}
