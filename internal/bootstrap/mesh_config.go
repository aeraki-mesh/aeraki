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

package bootstrap

import (
	"istio.io/istio/pkg/config/mesh/kubemesh"
)

const (
	// defaultMeshConfigMapName is the default name of the ConfigMap with the mesh config
	// The actual name can be different - use getMeshConfigMapName
	// defaultMeshConfigMapName = "istio"
	// configMapKey should match the expected MeshConfig file name
	configMapKey = "mesh"
)

// initSSecureWebhookServer handles initialization for the HTTPS webhook server.
func (s *Server) initConfigMapWatcher(args *AerakiArgs, handler func()) {
	// Watch the istio ConfigMap for mesh config changes.
	// This may be necessary for external Istiod.
	s.configMapWatcher = kubemesh.NewConfigMapWatcher(
		s.kubeClient, args.RootNamespace, args.IstioConfigMapName, configMapKey, false, s.internalStop)
	s.configMapWatcher.AddMeshHandler(handler)
}
