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

package utils

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// IsHTTP check if the port is using http protocol
func IsHTTP(port corev1.ServicePort) bool {
	// If application protocol is set, we will use that
	// If not, use the port name
	var name string
	if port.AppProtocol != nil {
		name = *port.AppProtocol
	}
	if name == "" {
		name = port.Name
	}
	name = strings.ToLower(name)

	if strings.HasPrefix(name, "https") {
		return false
	}

	return strings.HasPrefix(name, "http") ||
		strings.HasPrefix(name, "http2") ||
		strings.HasPrefix(name, "grpc")
}
