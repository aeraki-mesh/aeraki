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
	"github.com/aeraki-framework/aeraki/client-go/pkg/clientset/versioned/scheme"
	"istio.io/pkg/log"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var controllerLog = log.RegisterScope("controller", "crd controller", 0)

// NewManager create a manager to manager all crd controllers.
func NewManager(kubeConfig *rest.Config, namespace string, electionID string,
	triggerPush func() error) manager.Manager {
	mgrOpt := manager.Options{
		MetricsBindAddress:      "0",
		LeaderElection:          true,
		LeaderElectionNamespace: namespace,
		LeaderElectionID:        electionID,
	}
	m, err := manager.New(kubeConfig, mgrOpt)
	if err != nil {
		controllerLog.Fatalf("could not create a controller manager: %v", err)
	}

	err = addRedisServiceController(m, triggerPush)
	if err != nil {
		controllerLog.Fatalf("could not add RedisServiceController: %e", err)
	}
	err = addRedisDestinationController(m, triggerPush)
	if err != nil {
		controllerLog.Fatalf("could not add RedisDestinationController: %e", err)
	}
	err = addDubboAuthorizationPolicyController(m, triggerPush)
	if err != nil {
		controllerLog.Fatalf("could not add DubboAuthorizationPolicyController: %e", err)
	}
	err = scheme.AddToScheme(m.GetScheme())
	if err != nil {
		controllerLog.Fatalf("could not add schema: %e", err)
	}
	return m
}
