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
	"fmt"
	"github.com/aeraki-framework/aeraki/lazyxds/cmd/lazyxds/app/config"
	"strings"
)

const (
	// DefaultDNS is the suffix of default fqdn in kubernetes
	DefaultDNS = "svc.cluster.local"
)

// FQDN returns the fully qualified domain name of service
func FQDN(name, namespace string) string {
	// return fmt.Sprintf("%s/%s", name, namespace)
	return fmt.Sprintf("%s.%s.%s", name, namespace, DefaultDNS)
}

// ObjectID use name.namespace as object id
func ObjectID(name, namespace string) string {
	return fmt.Sprintf("%s.%s", name, namespace)
}

// PortID use svcID:port as port id
func PortID(svcID, port string) string {
	return fmt.Sprintf("%s:%s", svcID, port)
}

// ParseID parse the object id to name and namespace
func ParseID(id string) (string, string) {
	parts := strings.Split(id, ".")
	if len(parts) < 2 {
		// todo panic
		return "", ""
	}
	name := parts[0]
	namespace := parts[1]
	return name, namespace
}

// ServiceID2EgressString turn the service id to egress string of sidecar crd
func ServiceID2EgressString(id string) string {
	_, namespace := ParseID(id)

	return fmt.Sprintf("%s/%s", namespace, id)
}

// UpstreamCluster2ServiceID extract the service id from xds cluster id
func UpstreamCluster2ServiceID(cluster string) string {
	parts := strings.Split(cluster, "|")
	if len(parts) != 4 {
		return ""
	}
	if parts[0] != "outbound" {
		return ""
	}
	return parts[3]
}

// IsLazyEnabled check if lazyxds is enabled
func IsLazyEnabled(annotations map[string]string) bool {
	return strings.ToLower(annotations[config.LazyLoadingAnnotation]) == "true"
}
