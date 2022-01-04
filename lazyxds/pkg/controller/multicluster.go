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

package controller

import (
	"context"
	"fmt"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func (c *AggregationController) syncCluster(ctx context.Context, secret *corev1.Secret) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting add new cluster to mesh", "clusterName", secret.Name)
	var rawConfig []byte
	for _, kubeConfig := range secret.Data {
		rawConfig = kubeConfig
		break
	}

	kubeConfig, err := getRestConfig(rawConfig)
	if err != nil {
		err = fmt.Errorf("failed to get Istio config store secret: %v", err)
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	return c.AddCluster(secret.Name, kubeClient)
}

func (c *AggregationController) deleteCluster(ctx context.Context, clusterName string) error {
	return c.DeleteCluster(clusterName)
}

func getRestConfig(kubeConfig []byte) (*rest.Config, error) {
	if len(kubeConfig) == 0 {
		return nil, fmt.Errorf("kubeconfig is empty")
	}

	rawConfig, err := clientcmd.Load(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("kubeconfig cannot be loaded: %v", err)
	}

	if err := clientcmd.Validate(*rawConfig); err != nil {
		return nil, fmt.Errorf("kubeconfig is not valid: %v", err)
	}

	clientConfig := clientcmd.NewDefaultClientConfig(*rawConfig, &clientcmd.ConfigOverrides{})
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create kube clients: %v", err)
	}
	return restConfig, nil
}
