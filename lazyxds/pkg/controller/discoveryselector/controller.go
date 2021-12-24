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

package discoveryselector

import (
	"context"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils/log"
	"github.com/go-logr/logr"
	meshv1alpha1 "istio.io/api/mesh/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/yaml"
	"time"
)

// Controller is responsible for watching discoverySelectors
type Controller struct {
	log                     logr.Logger
	clientset               *kubernetes.Clientset
	updateDiscoverySelector func([]*metav1.LabelSelector) error
	reconcileAllNamespaces  func(context.Context) error
}

// NewController creates a new service controller
func NewController(
	clientset *kubernetes.Clientset,
	updateDiscoverySelector func([]*metav1.LabelSelector) error,
	reconcileAllNamespaces func(context.Context) error,
) *Controller {
	logger := klogr.New().WithName("ConfigMapController")
	c := &Controller{
		log:                     logger,
		clientset:               clientset,
		updateDiscoverySelector: updateDiscoverySelector,
		reconcileAllNamespaces:  reconcileAllNamespaces,
	}

	return c
}

// Run begins watching and syncing.
func (c *Controller) Run(stopCh <-chan struct{}) {
	c.log.Info("starting discoveryselector controller...")
	ctx := log.WithContext(context.Background(), c.log)
	configMapWatcher, err := c.clientset.CoreV1().ConfigMaps("istio-system").Watch(ctx, metav1.ListOptions{
		LabelSelector: "release=istio",
	})
	if err != nil {
		c.log.Error(err, "watch configMap <istio> failed")
		return
	}
	for {
		select {
		case e, ok := <-configMapWatcher.ResultChan():
			if !ok {
				c.log.Info("configMapWatcher chan has been close!")
				c.log.Info("clean chan over!")
				time.Sleep(time.Second * 5)
			}
			if e.Object != nil {
				c.log.Info("configMapWatcher chan is ok")
				dataMap := e.Object.DeepCopyObject().(*corev1.ConfigMap).Data
				if _, ok := dataMap["mesh"]; !ok {
					break
				}
				meshconfig := meshv1alpha1.MeshConfig{}
				err := yaml.Unmarshal([]byte(dataMap["mesh"]), &meshconfig)
				if err != nil {
					c.log.Error(err, "deserialize meshconfig failed")
					break
				}
				c.log.Info("meshconfig.DiscoverySelectors modified", "matchLabels", meshconfig.DiscoverySelectors)
				err = c.updateDiscoverySelector(meshconfig.DiscoverySelectors)
				if err != nil {
					c.log.Error(err, "update DiscoverySelector error")
					break
				}
				err = c.reconcileAllNamespaces(ctx)
				if err != nil {
					c.log.Error(err, "reconcileAllNamespaces error")
				}
			}
		case <-stopCh:
			c.log.Info("close configMapWatcher")
			return
		}
	}
}
