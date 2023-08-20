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

package redis

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/aeraki-mesh/client-go/pkg/clientset/versioned"
	redisv1alpha1 "github.com/aeraki-mesh/client-go/pkg/clientset/versioned/typed/redis/v1alpha1"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	networking "istio.io/api/networking/v1alpha3"
	istiomodel "istio.io/istio/pilot/pkg/model"
	"istio.io/pkg/log"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	"github.com/aeraki-mesh/aeraki/internal/envoyfilter"
	"github.com/aeraki-mesh/aeraki/internal/model"
)

var generatorLog = log.RegisterScope("redis-generator", "redis generator", 0)

// New Generator
func New(cfg *rest.Config, store istiomodel.ConfigStore) *Generator {
	clientset, err := versioned.NewForConfig(cfg)
	if err != nil {
		generatorLog.Fatalf("Could not create clientset: %e", err)
	}

	k8scli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		generatorLog.Fatalf("Could not create clientset: %e", err)
	}

	g := &Generator{
		secretsGetter: k8scli.CoreV1(),
		redis:         clientset.RedisV1alpha1(),
		store:         store,
	}
	generatorLog.Infof("redis generator created")
	return g
}

// Generator generate redis proxy filter configuration for redis service
type Generator struct {
	secretsGetter corev1.SecretsGetter
	redis         redisv1alpha1.RedisV1alpha1Interface
	store         istiomodel.ConfigStore
}

// Timeout is the default timeout for listing object from apiserver
const Timeout = time.Second * 10

// Generate redis envoy filter
func (g *Generator) Generate(filterContext *model.EnvoyFilterContext) (filters []*model.EnvoyFilterWrapper, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	se := filterContext.ServiceEntry.Spec
	for _, port := range se.Ports {
		if strings.HasPrefix(port.Name, "tcp-redis") {
			portFilters := g.generate(ctx, filterContext, port)
			if portFilters != nil {
				filters = append(filters, portFilters...)
			}
		}
	}
	return filters, nil
}

func (g *Generator) generate(ctx context.Context, filterContext *model.EnvoyFilterContext,
	targetPort *networking.ServicePort) []*model.EnvoyFilterWrapper {
	port := targetPort.Number
	portName := targetPort.Name
	generatorLog.Debugf("generate %s/%s/%s", filterContext.ServiceEntry.Namespace,
		filterContext.ServiceEntry.Name, portName)
	// copy and replace ports
	spec := *filterContext.ServiceEntry.Spec.DeepCopy()
	spec.Ports = []*networking.ServicePort{targetPort}
	filters := envoyfilter.GenerateReplaceNetworkFilter(
		filterContext.ServiceEntry,
		filterContext.ServiceEntry.Spec.Ports[0],
		g.buildOutboundProxyWithFallback(ctx, filterContext, port, portName),
		g.buildInboundProxy(filterContext),
		"envoy.filters.network.redis_proxy",
		"type.googleapis.com/envoy.extensions.filters.network.redis_proxy.v3.RedisProxy")

	cluster := g.buildOutboundCluster(ctx, filterContext, port)
	if cluster != nil {
		for _, filter := range filters {
			if filter.Envoyfilter.WorkloadSelector == nil {
				filter.Envoyfilter.ConfigPatches = append(filter.Envoyfilter.ConfigPatches,
					ReplaceClusterPatches(cluster)...)
			}
		}
	}
	if generatorLog.DebugEnabled() {
		fdata, _ := json.Marshal(filters)
		generatorLog.Infof("%s", string(fdata))
	}
	return filters
}

// ReplaceClusterPatches create a `replace` operation patch on `cluster`
func ReplaceClusterPatches(cluster *clusterv3.Cluster) []*networking.EnvoyFilter_EnvoyConfigObjectPatch {
	clusterStruct, err := valueOf(cluster)
	if err != nil {
		generatorLog.Errorf("convert cluster to struct failed: %e", err)
		return nil
	}
	addCluster := clusterPatch(cluster.Name)
	addCluster.Match = nil
	addCluster.Patch = &networking.EnvoyFilter_Patch{
		Operation: networking.EnvoyFilter_Patch_ADD,
		Value:     clusterStruct,
	}
	removeCluster := clusterPatch(cluster.Name)
	removeCluster.Patch = &networking.EnvoyFilter_Patch{
		Operation: networking.EnvoyFilter_Patch_REMOVE,
	}
	return []*networking.EnvoyFilter_EnvoyConfigObjectPatch{removeCluster, addCluster}
}

func clusterPatch(name string) *networking.EnvoyFilter_EnvoyConfigObjectPatch {
	return &networking.EnvoyFilter_EnvoyConfigObjectPatch{
		ApplyTo: networking.EnvoyFilter_CLUSTER,
		Match: &networking.EnvoyFilter_EnvoyConfigObjectMatch{
			ObjectTypes: &networking.EnvoyFilter_EnvoyConfigObjectMatch_Cluster{
				Cluster: &networking.EnvoyFilter_ClusterMatch{
					Name: name,
				}},
		},
	}
}

func valueOf(message proto.Message) (out *_struct.Struct, err error) {
	var buf []byte

	if buf, err = protojson.Marshal(message); err != nil {
		return nil, err
	}

	out = &_struct.Struct{}
	if err := protojson.Unmarshal(buf, out); err != nil {
		return nil, err
	}
	return out, nil
}
