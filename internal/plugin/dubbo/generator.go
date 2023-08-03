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

package dubbo

import (
	"github.com/aeraki-mesh/client-go/pkg/clientset/versioned"
	dubbov1alpha1 "github.com/aeraki-mesh/client-go/pkg/clientset/versioned/typed/dubbo/v1alpha1"
	"istio.io/pkg/log"
	"k8s.io/client-go/rest"

	"github.com/aeraki-mesh/aeraki/internal/envoyfilter"
	"github.com/aeraki-mesh/aeraki/internal/model"
)

var generatorLog = log.RegisterScope("dubbo-generator", "dubbo generator", 0)

// Generator defines a dubbo envoyfilter Generator
type Generator struct {
	client dubbov1alpha1.DubboV1alpha1Interface
}

// NewGenerator creates an new Dubbo Generator instance
func NewGenerator(cfg *rest.Config) *Generator {
	clientset, err := versioned.NewForConfig(cfg)
	if err != nil {
		generatorLog.Fatalf("Could not create clientset: %e", err)
	}

	return &Generator{
		client: clientset.DubboV1alpha1(),
	}
}

// Generate create EnvoyFilters for Dubbo services
func (g *Generator) Generate(context *model.EnvoyFilterContext) ([]*model.EnvoyFilterWrapper, error) {
	return envoyfilter.GenerateReplaceNetworkFilter(
		context.ServiceEntry,
		context.ServiceEntry.Spec.Ports[0],
		buildOutboundProxy(context),
		buildInboundProxy(context, g.client),
		"envoy.filters.network.dubbo_proxy",
		"type.googleapis.com/envoy.extensions.filters.network.dubbo_proxy.v3.DubboProxy"), nil
}
