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
	util.CreateNamespace("dubbo", "")
	util.LabelNamespace("dubbo", "istio-injection=enabled", "")
	util.KubeApply("dubbo", "testdata/dubbo-sample.yaml", "")
	util.KubeApply("dubbo", "testdata/serviceentry.yaml", "")
	util.KubeApply("dubbo", "testdata/destinationrule.yaml", "")
}

func shutdown() {
	util.KubeDelete("dubbo", "testdata/dubbo-sample.yaml", "")
	util.KubeDelete("dubbo", "testdata/serviceentry.yaml", "")
	util.KubeDelete("dubbo", "testdata/destinationrule.yaml", "")
	util.DeleteNamespace("dubbo", "")
}

func TestSidecarConfig(t *testing.T) {
	util.WaitForDeploymentsReady("dubbo", 10*time.Minute, "")
	consumerPod, _ := util.GetPodName("dubbo", "app=dubbo-sample-consumer", "")
	config, _ := util.PodExec("dubbo", consumerPod, "dubbo-sample-consumer", "curl 127.0.0.1:15000/config_dump", false, "")
	config = strings.Join(strings.Fields(config), "")
	want := "{\n\"name\":\"envoy.filters.network.dubbo_proxy\",\n\"typed_config\":{\n\"@type\":\"type.googleapis.com/envoy.extensions.filters.network.dubbo_proxy.v3.DubboProxy\",\n\"stat_prefix\":\"outbound|20880||org.apache.dubbo.samples.basic.api.demoservice\",\n\"route_config\":[\n{\n\"name\":\"outbound|20880||org.apache.dubbo.samples.basic.api.demoservice\",\n\"interface\":\"org.apache.dubbo.samples.basic.api.DemoService\",\n\"routes\":[\n{\n\"match\":{\n\"method\":{\n\"name\":{\n\"safe_regex\":{\n\"google_re2\":{},\n\"regex\":\".*\"\n}\n}\n}\n},\n\"route\":{\n\"cluster\":\"outbound|20880||org.apache.dubbo.samples.basic.api.demoservice\"\n}\n}\n]\n}\n]\n}\n}\n]\n}"
	want = strings.Join(strings.Fields(want), "")
	log.Info(config)
	if !strings.Contains(config, want) {
		t.Error("cant't find dubbo proxy in the outbound listener of the envoy sidecar")
	}
}
