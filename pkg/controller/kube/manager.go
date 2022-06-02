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
	"istio.io/pkg/log"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var controllerLog = log.RegisterScope("controller", "crd controller", 0)

// NewManager create a manager to manager all crd controllers.
func NewManager(kubeConfig *rest.Config, namespace string, leaderElection bool,
	leaderElectionID string) (manager.Manager, error) {
	mgrOpt := manager.Options{
		MetricsBindAddress:      "0",
		LeaderElection:          leaderElection,
		LeaderElectionID:        leaderElectionID,
		LeaderElectionNamespace: namespace,
	}
	m, err := manager.New(kubeConfig, mgrOpt)
	if err != nil {
		return nil, err
	}

	m.Elected()
	return m, nil
}
