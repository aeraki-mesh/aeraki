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

package thrift

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aeraki-framework/aeraki/test/e2e/util"
	"istio.io/pkg/log"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() {
	util.CreateNamespace("thrift", "")
	util.LabelNamespace("thrift", "istio-injection=enabled", "")
	util.KubeApply("thrift", "testdata/thrift-sample.yaml", "")
	util.KubeApply("thrift", "testdata/destinationrule.yaml", "")
}

func shutdown() {
	util.KubeDelete("thrift", "testdata/thrift-sample.yaml", "")
	util.KubeDelete("thrift", "testdata/serviceentry.yaml", "")
	util.KubeDelete("thrift", "testdata/destinationrule.yaml", "")
	util.DeleteNamespace("thrift", "")
}

func TestSidecarOutboundConfig(t *testing.T) {
	util.WaitForDeploymentsReady("thrift", 10*time.Minute, "")
	consumerPod, _ := util.GetPodName("thrift", "app=thrift-sample-client", "")
	config, _ := util.PodExec("thrift", consumerPod, "istio-proxy", "curl 127.0.0.1:15000/config_dump", false, "")
	config = strings.Join(strings.Fields(config), "")
	want := "{\n\"name\":\"envoy.filters.network.thrift_proxy\",\n\"typed_config\":{\n\"@type\":\"type.googleapis.com/envoy.extensions.filters.network.thrift_proxy.v3.ThriftProxy\",\n\"stat_prefix\":\"outbound|9090||thrift-sample-server.thrift.svc.cluster.local\",\n\"route_config\":{\n\"name\":\"outbound|9090||thrift-sample-server.thrift.svc.cluster.local\",\n\"routes\":[\n{\n\"match\":{\n\"method_name\":\"\"\n},\n\"route\":{\n\"cluster\":\"outbound|9090||thrift-sample-server.thrift.svc.cluster.local\"\n}\n}\n]\n},\n\"thrift_filters\":[\n{\n\"name\":\"envoy.filters.thrift.router\"\n}\n]\n}\n}"
	want = strings.Join(strings.Fields(want), "")
	log.Info(config)
	if !strings.Contains(config, want) {
		t.Error("cant't find thrift proxy in the outbound listener of the envoy sidecar")
	}
}

func TestVersionRouting(t *testing.T) {
	util.WaitForDeploymentsReady("thrift", 10*time.Minute, "")
	testVersion("v1", t)
	testVersion("v2", t)
}

func testVersion(version string, t *testing.T) {
	util.KubeApply("thrift", "testdata/virtualservice-"+version+".yaml", "")
	log.Info("Waiting for rules to propagate ...")
	time.Sleep(1 * time.Minute)
	consumerPod, _ := util.GetPodName("thrift", "app=thrift-sample-client", "")
	for i := 0; i < 5; i++ {
		thriftResponse, _ := util.PodExec("thrift", consumerPod, "thrift-sample-client", "curl 127.0.0.1:9009/hello", false, "")
		want := "response from thrift-sample-server-" + version
		log.Info(thriftResponse)
		if !strings.Contains(thriftResponse, want) {
			t.Error("")
		}
	}
}

func TestPercentageRouting(t *testing.T) {
	util.WaitForDeploymentsReady("thrift", 10*time.Minute, "")
	util.KubeApply("thrift", "testdata/virtualservice-traffic-splitting.yaml", "")
	log.Info("Waiting for rules to propagate ...")
	time.Sleep(1 * time.Minute)
	consumerPod, _ := util.GetPodName("thrift", "app=thrift-sample-client", "")
	v1 := 0
	for i := 0; i < 20; i++ {
		thriftResponse, _ := util.PodExec("thrift", consumerPod, "thrift-sample-client", "curl 127.0.0.1:9009/hello", false, "")
		responseV1 := "response from thrift-sample-server-v1"
		log.Info(thriftResponse)
		if strings.Contains(thriftResponse, responseV1) {
			v1++
		}
	}
	// The most accurate number should be 6, but the number may fall into a range around 6 since the sample is not big enough
	if v1 > 8 || v1 < 4 {
		t.Errorf("percentage traffic routing failed, want: %v got:%v ", 3, v1)
	}
}
