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

package metaprotocol

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aeraki-mesh/aeraki/test/e2e/util"
	"istio.io/pkg/log"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	//shutdown()
	os.Exit(code)
}

func setup() {
	util.CreateNamespace("metaprotocol", "")
	util.LabelNamespace("metaprotocol", "istio-injection=enabled", "")
	util.KubeApply("metaprotocol", "../../../k8s/aeraki-bootstrap-config.yaml", "")
	util.KubeApply("metaprotocol", "testdata/metaprotocol-sample.yaml", "")
	util.KubeApply("metaprotocol", "testdata/serviceentry.yaml", "")
	util.KubeApply("metaprotocol", "testdata/destinationrule.yaml", "")
	util.KubeApply("metaprotocol", "testdata/rate-limit-server/", "")
}

func shutdown() {
	util.KubeDelete("metaprotocol", "testdata/metaprotocol-sample.yaml", "")
	util.KubeDelete("metaprotocol", "testdata/serviceentry.yaml", "")
	util.KubeDelete("metaprotocol", "testdata/destinationrule.yaml", "")
	util.DeleteNamespace("metaprotocol", "")
}

func TestSidecarOutboundConfig(t *testing.T) {
	util.WaitForDeploymentsReady("metaprotocol", 10*time.Minute, "")
	time.Sleep(10 * time.Second) //wait for serviceentry vip allocation
	consumerPod, _ := util.GetPodName("metaprotocol", "app=dubbo-sample-consumer", "")
	config, _ := util.PodExec("metaprotocol", consumerPod, "istio-proxy", "curl -s 127.0.0.1:15000/config_dump", false, "")
	config = strings.Join(strings.Fields(config), "")
	want := "{\n\"name\":\"envoy.filters.network.meta_protocol_proxy\",\n\"typed_config\":{\n\"@type\":\"type.googleapis.com/udpa.type.v1.TypedStruct\",\n\"type_url\":\"type.googleapis.com/aeraki.meta_protocol_proxy.v1alpha.MetaProtocolProxy\",\n\"value\":{\n\"stat_prefix\":\"outbound|20880||org.apache.dubbo.samples.basic.api.demoservice\",\n\"application_protocol\":\"dubbo\",\n\"rds\":{\n\"config_source\":{\n\"api_config_source\":{\n\"api_type\":\"GRPC\",\n\"grpc_services\":[\n{\n\"envoy_grpc\":{\n\"cluster_name\":\"aeraki-xds\"\n}\n}\n],\n\"transport_api_version\":\"V3\"\n},\n\"resource_api_version\":\"V3\"\n},\n\"route_config_name\":\"org.apache.dubbo.samples.basic.api.demoservice_20880\"\n},\n\"codec\":{\n\"name\":\"aeraki.meta_protocol.codec.dubbo\"\n},\n\"meta_protocol_filters\":[\n{\n\"name\":\"aeraki.meta_protocol.filters.router\"\n}\n]\n}\n}\n}"
	want = strings.Join(strings.Fields(want), "")
	if !strings.Contains(config, want) {
		t.Errorf("cant't find metaprotocol proxy in the outbound listener of the envoy sidecar: conf \n %s, want \n %s", config, want)
	}
}

func TestSidecarInboundConfig(t *testing.T) {
	util.WaitForDeploymentsReady("metaprotocol", 10*time.Minute, "")
	time.Sleep(1 * time.Minute) //wait for serviceentry vip allocation
	providerPod, _ := util.GetPodName("metaprotocol", "app=dubbo-sample-provider", "")
	config, _ := util.PodExec("metaprotocol", providerPod, "istio-proxy", "curl -s 127.0.0.1:15000/config_dump", false, "")
	config = strings.Join(strings.Fields(config), "")
	want := "{\n\"name\":\"envoy.filters.network.meta_protocol_proxy\",\n\"typed_config\":{\n\"@type\":\"type.googleapis.com/udpa.type.v1.TypedStruct\",\n\"type_url\":\"type.googleapis.com/aeraki.meta_protocol_proxy.v1alpha.MetaProtocolProxy\",\n\"value\":{\n\"stat_prefix\":\"inbound|20880||\",\n\"application_protocol\":\"dubbo\",\n\"route_config\":{\n\"name\":\"inbound|20880||\",\n\"routes\":[\n{\n\"route\":{\n\"cluster\":\"inbound|20880||\"\n}\n}\n]\n},\n\"codec\":{\n\"name\":\"aeraki.meta_protocol.codec.dubbo\"\n},\n\"meta_protocol_filters\":[\n{\n\"name\":\"aeraki.meta_protocol.filters.router\"\n}\n]\n}\n}\n}"
	want = strings.Join(strings.Fields(want), "")
	if !strings.Contains(config, want) {
		t.Errorf("cant't find metaprotocol proxy in the inbound listener of the envoy sidecar: conf \n %s, want \n %s", config, want)
	}
}

func TestVersionRouting(t *testing.T) {
	util.WaitForDeploymentsReady("metaprotocol", 10*time.Minute, "")
	testVersion("v1", t)
	testVersion("v2", t)
}

func testVersion(version string, t *testing.T) {
	util.KubeApply("metaprotocol", "testdata/metarouter-"+version+".yaml", "")
	defer util.KubeDelete("metaprotocol", "testdata/metarouter-"+version+".yaml", "")

	log.Info("Waiting for rules to propagate ...")
	time.Sleep(1 * time.Minute)
	consumerPod, _ := util.GetPodName("metaprotocol", "app=dubbo-sample-consumer", "")
	for i := 0; i < 5; i++ {
		dubboResponse, _ := util.PodExec("metaprotocol", consumerPod, "dubbo-sample-consumer",
			"curl -s 127.0.0.1:9009/hello", false, "")
		want := "response from dubbo-sample-provider-" + version
		log.Info(dubboResponse)
		if !strings.Contains(dubboResponse, want) {
			t.Errorf("Version routing failed, want: %s, got %s", want, dubboResponse)
		}
	}
}

func TestPercentageRouting(t *testing.T) {
	util.WaitForDeploymentsReady("metaprotocol", 10*time.Minute, "")
	util.KubeApply("metaprotocol", "testdata/metarouter-traffic-splitting.yaml", "")
	defer util.KubeDelete("metaprotocol", "testdata/metarouter-traffic-splitting.yaml", "")

	log.Info("Waiting for rules to propagate ...")
	time.Sleep(1 * time.Minute)
	consumerPod, _ := util.GetPodName("metaprotocol", "app=dubbo-sample-consumer", "")
	v1 := 0
	for i := 0; i < 40; i++ {
		dubboResponse, _ := util.PodExec("metaprotocol", consumerPod, "dubbo-sample-consumer",
			"curl -s 127.0.0.1:9009/hello", false, "")
		responseV1 := "response from dubbo-sample-provider-v1"
		log.Info(dubboResponse)
		if strings.Contains(dubboResponse, responseV1) {
			v1++
		}
	}
	// The most accurate number should be 8, but the number may fall into a range around 8 since the sample is not big enough
	if v1 > 12 || v1 < 4 {
		t.Errorf("percentage traffic routing failed, want: %s got:%v ", "between 4 and 12", v1)
	} else {
		t.Logf("%v requests have been sent to v1", v1)
	}
}

func TestAttributeRouting(t *testing.T) {
	util.WaitForDeploymentsReady("metaprotocol", 10*time.Minute, "")
	testAttributeMatch("exact", t)
	testAttributeMatch("prefix", t)
	testAttributeMatch("regex", t)
}

func testAttributeMatch(matchPattern string, t *testing.T) {
	util.KubeApply("metaprotocol", "testdata/metarouter-attribute-"+matchPattern+".yaml", "")
	log.Info("Waiting for rules to propagate ...")
	time.Sleep(1 * time.Minute)
	consumerPod, _ := util.GetPodName("metaprotocol", "app=dubbo-sample-consumer", "")
	for i := 0; i < 5; i++ {
		dubboResponse, _ := util.PodExec("metaprotocol", consumerPod, "dubbo-sample-consumer",
			"curl -s 127.0.0.1:9009/hello", false, "")
		want := "response from dubbo-sample-provider-v2"
		log.Info(dubboResponse)
		if !strings.Contains(dubboResponse, want) {
			t.Errorf("attribute routing failed, want: %s, got %s", want, dubboResponse)
		}
	}
}

func TestLocalRateLimit(t *testing.T) {
	util.WaitForDeploymentsReady("metaprotocol", 10*time.Minute, "")
	util.KubeApply("metaprotocol", "testdata/metarouter-local-ratelimit.yaml", "")
	defer util.KubeDelete("metaprotocol", "testdata/metarouter-local-ratelimit.yaml", "")

	log.Info("Waiting for rules to propagate ...")
	time.Sleep(1 * time.Minute)
	consumerPod, _ := util.GetPodName("metaprotocol", "app=dubbo-sample-consumer", "")
	request := 0
	for i := 0; i < 10; i++ {
		dubboResponse, _ := util.PodExec("metaprotocol", consumerPod, "dubbo-sample-consumer",
			"curl -s 127.0.0.1:9009/hello", false, "")
		responseV1 := "response from dubbo-sample-provider-v1"
		log.Info(dubboResponse)
		if strings.Contains(dubboResponse, responseV1) {
			request++
		}
	}
	if request != 2 {
		t.Errorf("local rate limit failed, want: %v got:%v ", 2, request)
	} else {
		t.Logf("%v requests have been sent to server", request)
	}
}

func TestGlobalRateLimit(t *testing.T) {
	util.WaitForDeploymentsReady("metaprotocol", 10*time.Minute, "")
	util.KubeApply("metaprotocol", "testdata/metarouter-global-ratelimit.yaml", "")
	defer util.KubeDelete("metaprotocol", "testdata/metarouter-global-ratelimit.yaml", "")

	log.Info("Waiting for rules to propagate ...")
	time.Sleep(1 * time.Minute)
	consumerPod, _ := util.GetPodName("metaprotocol", "app=dubbo-sample-consumer", "")
	request := 0
	for i := 0; i < 20; i++ {
		dubboResponse, _ := util.PodExec("metaprotocol", consumerPod, "dubbo-sample-consumer",
			"curl -s 127.0.0.1:9009/hello", false, "")
		responseV1 := "response from dubbo-sample-provider-v1"
		log.Info(dubboResponse)
		if strings.Contains(dubboResponse, responseV1) {
			request++
		}
	}

	if request != 10 {
		t.Errorf("global rate limit failed, want: %v got:%v ", 10, request)
	} else {
		t.Logf("%v requests have been sent to server", request)
	}
}
