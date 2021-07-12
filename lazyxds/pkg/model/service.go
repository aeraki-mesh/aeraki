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
	"fmt"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"reflect"
	"sort"
	"sync"
)

// Service represent one service cross multi-cluster
type Service struct {
	mu            sync.Mutex // todo
	Name          string
	Namespace     string
	Distribution  map[string]*clusterServiceStatus
	EgressService map[string]struct{} // http service which reported from als
	NSLazy        NSLazyStatus

	Spec   serviceStatus // desired status
	Status serviceStatus
}

// serviceStatus is the status of lazyxds service
type serviceStatus struct {
	ClusterIPs []string
	HTTPPorts  map[string]struct{}
	TCPPorts   map[string]struct{}

	LazyEnabled  bool
	LazySelector map[string]string
}

// clusterServiceStatus is the service status of one cluster
type clusterServiceStatus struct {
	ClusterIP string
	HTTPPorts map[string]struct{}
	TCPPorts  map[string]struct{}

	LazyEnabled bool
	Selector    map[string]string
}

// NewService creates new Service
func NewService(service *corev1.Service) *Service {
	return &Service{
		Name:          service.Name,
		Namespace:     service.Namespace,
		Distribution:  make(map[string]*clusterServiceStatus),
		EgressService: make(map[string]struct{}),
	}
}

// ID use FQDN as service id
func (svc *Service) ID() string {
	return utils.FQDN(svc.Name, svc.Namespace)
}

// NeedReconcileService check if service need reconcile
func (svc *Service) NeedReconcileService() bool {
	return !reflect.DeepEqual(svc.Status.HTTPPorts, svc.Spec.HTTPPorts) ||
		!reflect.DeepEqual(svc.Status.TCPPorts, svc.Spec.TCPPorts) ||
		!reflect.DeepEqual(svc.Status.ClusterIPs, svc.Spec.ClusterIPs)
}

// FinishReconcileService update status using spec info
func (svc *Service) FinishReconcileService() {
	svc.Status.HTTPPorts = svc.Spec.HTTPPorts
	svc.Status.TCPPorts = svc.Spec.TCPPorts
	svc.Status.ClusterIPs = svc.Spec.ClusterIPs
}

// NeedReconcileLazy check if lazy info is equal in status and spec
func (svc *Service) NeedReconcileLazy() bool {
	return !reflect.DeepEqual(svc.Status.LazyEnabled, svc.Spec.LazyEnabled) ||
		!reflect.DeepEqual(svc.Status.LazySelector, svc.Spec.LazySelector)
}

// FinishReconcileLazy update lazy status using spec info
func (svc *Service) FinishReconcileLazy() {
	svc.Status.LazyEnabled = svc.Spec.LazyEnabled
	svc.Status.LazySelector = svc.Spec.LazySelector
}

// UpdateClusterService update the service of one cluster
func (svc *Service) UpdateClusterService(clusterName string, service *corev1.Service) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	cs := &clusterServiceStatus{
		ClusterIP: service.Spec.ClusterIP,
		HTTPPorts: make(map[string]struct{}),
		TCPPorts:  make(map[string]struct{}),

		LazyEnabled: utils.IsLazyEnabled(service.Annotations),
		Selector:    service.Spec.Selector,
	}

	// https://github.com/aeraki-framework/aeraki/issues/83
	// if a service without selector, the lazy loading is disabled
	// we always disable lazy loading feature on a service with ExternalName
	if service.Spec.Type == corev1.ServiceTypeExternalName {
		cs.Selector = nil
	}

	for _, servicePort := range service.Spec.Ports {
		if utils.IsHTTP(servicePort) {
			cs.HTTPPorts[fmt.Sprint(servicePort.Port)] = struct{}{}
		} else {
			cs.TCPPorts[fmt.Sprint(servicePort.Port)] = struct{}{}
		}
	}
	svc.Distribution[clusterName] = cs

	svc.updateSpec()
}

// UpdateNSLazy update the enabled status of the namespace
// If the ns lazy status changed, we need update service spec
func (svc *Service) UpdateNSLazy(status NSLazyStatus) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	if svc.NSLazy != status {
		svc.NSLazy = status
		svc.updateSpec()
	}
}

// DeleteFromCluster delete the service of one cluster
func (svc *Service) DeleteFromCluster(clusterName string) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	delete(svc.Distribution, clusterName)
	svc.updateSpec()
}

func (svc *Service) updateSpec() {
	spec := serviceStatus{
		HTTPPorts:    make(map[string]struct{}),
		TCPPorts:     make(map[string]struct{}),
		LazySelector: make(map[string]string),
	}

	ports := make(map[string]bool)
	clusterIPSet := make(map[string]bool)
	svcMustDisableLazy := false
	for _, cs := range svc.Distribution {
		for p := range cs.HTTPPorts {
			if old, ok := ports[p]; ok {
				ports[p] = old // only if the port of all clusters are http, it's http
			} else {
				ports[p] = true
			}
		}
		for p := range cs.TCPPorts {
			ports[p] = false
		}

		if len(cs.Selector) == 0 {
			svcMustDisableLazy = true
		}

		if svcMustDisableLazy {
			spec.LazyEnabled = false
		} else {
			if svc.NSLazy == NSLazyStatusDisabled {
				spec.LazyEnabled = false
			} else if svc.NSLazy == NSLazyStatusEnabled {
				spec.LazyEnabled = true
			} else {
				spec.LazyEnabled = spec.LazyEnabled || cs.LazyEnabled
			}
		}

		// if cs.LazyEnabled {
		spec.LazySelector = cs.Selector // random now, need doc this
		// }

		ip := cs.ClusterIP
		if clusterIPSet[ip] {
			continue
		}
		clusterIPSet[ip] = true
		spec.ClusterIPs = append(spec.ClusterIPs, ip)
	}
	sort.Slice(spec.ClusterIPs, func(i, j int) bool {
		return spec.ClusterIPs[i] > spec.ClusterIPs[j]
	})

	for p, isHTTP := range ports {
		if isHTTP {
			spec.HTTPPorts[p] = struct{}{}
		} else {
			spec.TCPPorts[p] = struct{}{}
		}
	}

	svc.Spec = spec
}

// DomainListOfPort return the whole list of domains related with this port
func (svc *Service) DomainListOfPort(num, sourceNS string) []string {
	fqdn := svc.ID()
	list := []string{
		fqdn,
		fmt.Sprintf("%s:%s", fqdn, num),

		fmt.Sprintf("%s.%s.%s", svc.Name, svc.Namespace, "svc.cluster"),
		fmt.Sprintf("%s.%s.%s:%s", svc.Name, svc.Namespace, "svc.cluster", num),

		fmt.Sprintf("%s.%s.%s", svc.Name, svc.Namespace, "svc"),
		fmt.Sprintf("%s.%s.%s:%s", svc.Name, svc.Namespace, "svc", num),

		fmt.Sprintf("%s.%s", svc.Name, svc.Namespace),
		fmt.Sprintf("%s.%s:%s", svc.Name, svc.Namespace, num),
	}
	if svc.Namespace == sourceNS { // in case 2 services with same name are in 2 different ns
		list = append(list, svc.Name)
		list = append(list, fmt.Sprintf("%s:%s", svc.Name, num))
	}

	for _, ip := range svc.Spec.ClusterIPs {
		if ip == "None" { // todo: for headless service, but the sub domain is miss in outbound of egreee, need fix
			l := len(list)
			for i := 0; i < l; i++ {
				list = append(list, fmt.Sprintf("*.%s", list[i]))
			}
		} else if ip != "" {
			list = append(list, ip)
			list = append(list, fmt.Sprintf("%s:%s", ip, num))
		}
	}

	return list
}
