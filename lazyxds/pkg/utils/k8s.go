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

package utils

import (
	"context"
	"fmt"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils/log"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"time"
)

// NewKubeClient creates new kube client
func NewKubeClient(kubeconfigPath string) (*kubernetes.Clientset, error) {
	var err error
	var kubeConf *rest.Config

	if kubeconfigPath == "" {
		// creates the in-cluster config
		kubeConf, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("build default in cluster kube config failed: %w", err)
		}
	} else {
		kubeConf, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("build kube client config from config file failed: %w", err)
		}
	}
	return kubernetes.NewForConfig(kubeConf)
}

// NewIstioClient creates new istio client which use to handle istio CRD
func NewIstioClient(kubeconfigPath string) (*istioclient.Clientset, error) {
	var err error
	var istioConf *rest.Config

	if kubeconfigPath == "" {
		istioConf, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("build default in cluster istio config failed: %w", err)
		}
	} else {
		istioConf, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("build istio client config from config file failed: %w", err)
		}
	}

	return istioclient.NewForConfig(istioConf)
}

// WaitDeployment wait the deployment ready
func WaitDeployment(ctx context.Context, client *kubernetes.Clientset, ns, name string) (err error) {
	logger := log.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
			logger.Info(fmt.Sprintf("Waiting deployment: %s/%s", ns, name))
			time.Sleep(5 * time.Second)
			deployment, e := client.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
			if e != nil {
				logger.Error(e, fmt.Sprintf("failed to get deployment %s/%s", ns, name))
				if errors.IsNotFound(e) {
					continue
				} else {
					err = e
					return
				}
			}
			desired := *deployment.Spec.Replicas
			current := deployment.Status.Replicas
			available := deployment.Status.AvailableReplicas
			updated := deployment.Status.UpdatedReplicas
			if current != desired {
				logger.Info("replicas of currently pods is not in line with the desire",
					"current", current, "desired", desired)
				continue
			}
			if available != desired {
				logger.Info("replicas of available pods is not in line with the desire",
					"available", available, "desired", desired)
				continue
			}
			if updated != desired {
				logger.Info("replicas of updated pods if not in line with the desire",
					"updated", updated, "desired", desired)
				continue
			}

			return
		}
	}
}
