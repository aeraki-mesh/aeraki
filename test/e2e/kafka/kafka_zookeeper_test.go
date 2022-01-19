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

	"github.com/aeraki-mesh/aeraki/test/e2e/util"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() {
	util.CreateNamespace("kafka", "")
	util.LabelNamespace("kafka", "istio-injection=enabled", "")
	util.Shell("helm repo add zhaohuabing https://zhaohuabing.github.io/helm-repo")
	util.Shell("helm repo update")
	util.Shell("helm install my-release --set persistence.enabled=false --set zookeeper.persistence.enabled=false zhaohuabing/kafka -n kafka")
	util.KubeApply("kafka", "testdata/kafka-sample.yaml", "")
}

func shutdown() {
	//util.Shell("helm delete my-release -n kafka")
	//util.KubeDelete("kafka", "testdata/kafka-sample.yaml", "")
	//util.DeleteNamespace("kafka", "")
}

func TestKafkaSidecarOutboundConfig(t *testing.T) {
	util.WaitForStatefulsetReady("kafka", 20*time.Minute, "")
	util.WaitForDeploymentsReady("kafka", 20*time.Minute, "")
	consumerPod, _ := util.GetPodName("kafka", "app.kubernetes.io/name=kafka", "")
	config, _ := util.PodExec("kafka", consumerPod, "istio-proxy", "curl -s 127.0.0.1:15000/config_dump", true, "")
	config = strings.Join(strings.Fields(config), "")
	want := "{\n\"name\":\"envoy.filters.network.kafka_broker\",\n\"typed_config\":{\n\"@type\":\"type.googleapis.com/udpa.type.v1.TypedStruct\",\n\"type_url\":\"type.googleapis.com/envoy.extensions.filters.network.kafka_broker.v3.KafkaBroker\",\n\"value\":{\n\"stat_prefix\":\"outbound|9092||my-release-kafka.kafka.svc.cluster.local\"\n}\n}\n}"
	want = strings.Join(strings.Fields(want), "")
	if !strings.Contains(config, want) {
		t.Errorf("cant't find kafaka filter in the outbound listener of the envoy sidecar: conf \n %s, want \n %s", config, want)
	}
}

func TestKafkaSidecarInboundConfig(t *testing.T) {
	util.WaitForStatefulsetReady("kafka", 20*time.Minute, "")
	util.WaitForDeploymentsReady("kafka", 20*time.Minute, "")
	kafkaPod, _ := util.GetPodName("kafka", "app.kubernetes.io/name=kafka", "")
	config, _ := util.PodExec("kafka", kafkaPod, "istio-proxy", "curl -s 127.0.0.1:15000/config_dump", true, "")
	config = strings.Join(strings.Fields(config), "")
	want := "{\n\"name\":\"envoy.filters.network.kafka_broker\",\n\"typed_config\":{\n\"@type\":\"type.googleapis.com/udpa.type.v1.TypedStruct\",\n\"type_url\":\"type.googleapis.com/envoy.extensions.filters.network.kafka_broker.v3.KafkaBroker\",\n\"value\":{\n\"stat_prefix\":\"inbound|9092||\"\n}\n}\n}"
	want = strings.Join(strings.Fields(want), "")
	if !strings.Contains(config, want) {
		t.Errorf("cant't find kafka filter in the inbound listener of the envoy sidecar: conf \n %s, want \n %s", config, want)
	}
}

func TestZookeeperSidecarOutboundConfig(t *testing.T) {
	util.WaitForDeploymentsReady("kafka", 20*time.Minute, "")
	kafkaPod, _ := util.GetPodName("kafka", "app.kubernetes.io/name=kafka", "")
	config, _ := util.PodExec("kafka", kafkaPod, "istio-proxy", "curl -s 127.0.0.1:15000/config_dump", true, "")
	config = strings.Join(strings.Fields(config), "")
	want := "{\n\"name\":\"envoy.filters.network.zookeeper_proxy\",\n\"typed_config\":{\n\"@type\":\"type.googleapis.com/udpa.type.v1.TypedStruct\",\n\"type_url\":\"type.googleapis.com/envoy.extensions.filters.network.zookeeper_proxy.v3.ZooKeeperProxy\",\n\"value\":{\n\"stat_prefix\":\"outbound|2181||my-release-zookeeper.kafka.svc.cluster.local\"\n}\n}\n}"
	want = strings.Join(strings.Fields(want), "")
	if !strings.Contains(config, want) {
		t.Errorf("cant't find zookeeper filter in the outbound listener of the envoy sidecar: conf \n %s, want \n %s", config, want)
	}
}

func TestZookeeperSidecarInboundConfig(t *testing.T) {
	util.WaitForDeploymentsReady("kafka", 20*time.Minute, "")
	zookeeperPod, _ := util.GetPodName("kafka", "app.kubernetes.io/name=zookeeper", "")
	config, _ := util.PodExec("kafka", zookeeperPod, "istio-proxy", "curl -s 127.0.0.1:15000/config_dump", true, "")
	config = strings.Join(strings.Fields(config), "")
	want := "{\n\"name\":\"envoy.filters.network.zookeeper_proxy\",\n\"typed_config\":{\n\"@type\":\"type.googleapis.com/udpa.type.v1.TypedStruct\",\n\"type_url\":\"type.googleapis.com/envoy.extensions.filters.network.zookeeper_proxy.v3.ZooKeeperProxy\",\n\"value\":{\n\"stat_prefix\":\"inbound|2181||\"\n}\n}\n}"
	want = strings.Join(strings.Fields(want), "")
	if !strings.Contains(config, want) {
		t.Errorf("cant't find zookeeper filter in the inbound listener of the envoy sidecar: conf \n %s, want \n %s", config, want)
	}
}
