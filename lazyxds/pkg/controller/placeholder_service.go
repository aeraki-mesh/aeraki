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

package controller

import (
	"context"
	"fmt"
	"github.com/aeraki-framework/aeraki/lazyxds/cmd/lazyxds/app/config"
	"strconv"

	"github.com/aeraki-framework/aeraki/lazyxds/pkg/model"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PlaceHolderService ...
const PlaceHolderService = "lazyxds-placeholder-service"

// buildPlaceHolderService creates a fake service to make Istio create all the needed HTTP listeners at the envoy
// proxy. A route will be added to all the created HTTP listeners to redirect traffic to the lazyxds egress gateway.
func (c *AggregationController) buildPlaceHolderService(ctx context.Context) error {
	existingGlobalService, err := c.KubeClient.CoreV1().Services("istio-system").Get(ctx,
		PlaceHolderService, metaV1.GetOptions{})
	firstCreate := false
	if err != nil {
		if errors.IsNotFound(err) {
			firstCreate = true
		} else {
			return err
		}
	}

	currentHTTPPorts := make(map[string]struct{})
	handler := func(key, value interface{}) bool {
		svc := value.(*model.Service)
		for servicePort := range svc.Spec.HTTPPorts {
			currentHTTPPorts[servicePort] = struct{}{}
		}
		return true
	}
	c.services.Range(handler)
	if len(currentHTTPPorts) == 0 {
		// todo delete empty svc
		return nil
	}

	var servicePorts []coreV1.ServicePort
	for port := range currentHTTPPorts {
		portNum, _ := strconv.Atoi(port)
		servicePorts = append(servicePorts, coreV1.ServicePort{
			Name: "http-" + port,
			Port: int32(portNum),
		})
	}

	if firstCreate {
		if err := c.createPlaceholderService(ctx, servicePorts); err != nil {
			return err
		}
	}

	if !firstCreate && c.isServicePortsChanged(existingGlobalService, currentHTTPPorts) {
		if err := c.updatePlaceholderService(ctx, existingGlobalService, servicePorts); err != nil {
			return err
		}
	}

	return nil
}

func (c *AggregationController) isServicePortsChanged(existingGlobalService *coreV1.Service,
	currentHTTPPorts map[string]struct{}) bool {
	globalServicePorts := make(map[string]struct{})
	for port := range existingGlobalService.Spec.Ports {
		portStr := fmt.Sprint(port)
		globalServicePorts[portStr] = struct{}{}
	}

	for httpPort := range currentHTTPPorts {
		if _, ok := globalServicePorts[httpPort]; !ok {
			return true
		}
	}

	for httpPort := range globalServicePorts {
		if _, ok := currentHTTPPorts[httpPort]; !ok {
			return true
		}
	}
	return false
}

func (c *AggregationController) updatePlaceholderService(ctx context.Context, existingGlobalService *coreV1.Service,
	servicePorts []coreV1.ServicePort) error {
	existingGlobalService.Spec.Ports = servicePorts
	_, err := c.KubeClient.CoreV1().Services("istio-system").Update(ctx, existingGlobalService,
		metaV1.UpdateOptions{
			FieldManager: config.LazyXdsManager,
		})
	if err != nil {
		return err
	}
	return nil
}

func (c *AggregationController) createPlaceholderService(ctx context.Context, servicePorts []coreV1.ServicePort) error {
	newGlobalService := &coreV1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name: PlaceHolderService,
		},
		Spec: coreV1.ServiceSpec{
			Selector: map[string]string{
				"app": PlaceHolderService,
			},
			Ports: servicePorts,
		},
	}

	_, err := c.KubeClient.CoreV1().Services("istio-system").Create(ctx, newGlobalService,
		metaV1.CreateOptions{
			FieldManager: config.LazyXdsManager,
		})
	return err
}
