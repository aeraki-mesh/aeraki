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

package model

import (
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils"
	corev1 "k8s.io/api/core/v1"
)

// NSLazyStatus represents the status of lazy xds
type NSLazyStatus int

const (
	// NSLazyStatusNone ...
	NSLazyStatusNone NSLazyStatus = 0
	// NSLazyStatusEnabled ...
	NSLazyStatusEnabled NSLazyStatus = 1
	// NSLazyStatusDisabled ...
	NSLazyStatusDisabled NSLazyStatus = 2
)

// Namespace represent the namespace which contains multi-cluster info
type Namespace struct {
	Name string

	Distribution map[string]bool
	UserSidecar  map[string]struct{}

	LazyStatus NSLazyStatus
}

// NewNamespace creates new Namespace struct from kubernetes namespace
func NewNamespace(namespace *corev1.Namespace) *Namespace {
	return &Namespace{
		Name:         namespace.Name,
		Distribution: make(map[string]bool),
		UserSidecar:  make(map[string]struct{}),
	}
}

// ID use name of namespace as id
func (ns *Namespace) ID() string {
	return ns.Name
}

// LazyEnabled check if the namespace enable lazyxds
// Currently, if there is any user created sidecar CRD, the lazyxds will be disabled
func (ns *Namespace) LazyEnabled(clusterName string) bool {
	if len(ns.UserSidecar) > 0 {
		return false
	}

	return ns.Distribution[clusterName]
}

// Update update one namespace of the multiCluster
func (ns *Namespace) Update(clusterName string, namespace *corev1.Namespace) {
	ns.Distribution[clusterName] = utils.IsLazyEnabled(namespace.Annotations)
	ns.updateLazyStatus()
}

// AddSidecar record user-created sidecar crd
func (ns *Namespace) AddSidecar(sidecarName string) {
	ns.UserSidecar[sidecarName] = struct{}{}
	ns.updateLazyStatus()
}

// DeleteSidecar delete use-created sidecar crd
func (ns *Namespace) DeleteSidecar(sidecarName string) {
	delete(ns.UserSidecar, sidecarName)
	ns.updateLazyStatus()
}

// Delete delete a namespace of one cluster
func (ns *Namespace) Delete(clusterName string) {
	delete(ns.Distribution, clusterName)
	ns.updateLazyStatus()
}

func (ns *Namespace) updateLazyStatus() {
	lazyStatus := NSLazyStatusNone

	if len(ns.UserSidecar) > 0 {
		lazyStatus = NSLazyStatusDisabled
	} else {
		for _, lazy := range ns.Distribution {
			if lazy { // if one is true, then true
				lazyStatus = NSLazyStatusEnabled
				break
			}
		}
	}

	ns.LazyStatus = lazyStatus
}
