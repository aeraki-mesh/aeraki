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

package manager

import (
	"context"
	"fmt"
	"github.com/aeraki-framework/aeraki/lazyxds/cmd/lazyxds/app/config"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/accesslog"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/bootstrap"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/controller"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// LazyXdsManager ...
type LazyXdsManager interface {
	AddCluster(name string, client *kubernetes.Clientset) error
	DeleteCluster(name string) error
	Run() error
	//Run(stopCh <-chan struct{}) error
}

// Manager contains the main lazy xds controller
type Manager struct {
	conf            *config.Config
	stop            <-chan struct{} // todo move to args of run function
	masterClient    *kubernetes.Clientset
	istioClient     *istioclient.Clientset
	controller      *controller.AggregationController
	accessLogServer *accesslog.Server
}

var singleton *Manager

// NewManager ...
func NewManager(conf *config.Config, stop <-chan struct{}) (*Manager, error) {
	if singleton != nil {
		klog.Error("LazyXds Manager already exist")
		return singleton, nil
	}

	kubeClient, err := utils.NewKubeClient(conf.KubeConfig)
	if err != nil {
		return nil, err
	}
	istioClient, err := utils.NewIstioClient(conf.KubeConfig)
	if err != nil {
		return nil, err
	}

	singleton = &Manager{
		conf:         conf,
		stop:         stop,
		masterClient: kubeClient,
		istioClient:  istioClient,
		controller:   controller.NewController(istioClient, stop),
	}
	singleton.accessLogServer = accesslog.NewAccessLogServer(singleton.controller)

	return singleton, nil
}

// Run ...
func (m *Manager) Run() error {
	klog.Info("Starting access log server...")
	if err := m.accessLogServer.Serve(); err != nil {
		return fmt.Errorf("start access log server failed: %w", err)
	}

	m.controller.Run()

	// todo we need support multiple cluster
	if err := m.AddCluster("Kubernetes", m.masterClient); err != nil {
		return err
	}
	m.controller.KubeClient = m.masterClient

	return nil
}

// AddCluster ...
func (m *Manager) AddCluster(name string, client *kubernetes.Clientset) error {
	if m.conf.AutoCreateEgress {
		klog.Info("Starting create lazyxds egress", "cluster", name)
		if err := bootstrap.InitEgress(context.TODO(), name, client, m.istioClient, m.conf.IstiodAddress, m.conf.ProxyImage); err != nil {
			return fmt.Errorf("init egress of cluster %s failed: %w", name, err)
		}
	}

	return m.controller.AddCluster(name, client)
}

// DeleteCluster ...
func (m *Manager) DeleteCluster(name string) error {
	return nil
}
