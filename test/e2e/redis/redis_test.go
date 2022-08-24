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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/util/jsonpath"

	"github.com/aeraki-mesh/aeraki/test/e2e/util"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() {
	util.CreateNamespace("redis", "")
	util.CreateNamespace("redis-client", "")
	util.LabelNamespace("redis-client", "istio-injection=enabled", "")
	util.KubeApply("redis", "testdata/redis-single.yaml", "")
	util.KubeApply("redis", "testdata/redis-cluster.yaml", "")
	util.KubeApply("redis-client", "testdata/redis-client.yaml", "")
	util.WaitForDeploymentsReady("redis", 10*time.Minute, "")
	util.WaitForStatefulsetReady("redis", 10*time.Minute, "")
	podsInfo, _ := util.GetPodsInfo("redis", "", "app=redis-cluster")
	var redisClusterAddrs string
	for _, info := range podsInfo {
		redisClusterAddrs += info.IPAddr + ":6379 "
	}
	util.PodExec("redis", podsInfo[0].Name, "redis",
		"redis-cli --cluster create --cluster-yes --cluster-replicas 1 "+redisClusterAddrs,
		false, "")
	util.KubeApply("redis", "testdata/redisservice.yaml", "")
	util.KubeApply("redis", "testdata/redisdestination.yaml", "")
	util.WaitForDeploymentsReady("redis-client", 10*time.Minute, "")
	// wait for sidecar sync xds from pilot
	time.Sleep(10 * time.Second)
}

func shutdown() {
	util.DeleteNamespace("redis", "")
	util.DeleteNamespace("redis-client", "")
}

func TestAutoAuth(t *testing.T) {
	clientPod, _ := util.GetPodName("redis-client", "app=redis-client", "")
	singleIP, _ := util.GetServiceIP("redis-single", "redis", "")

	out, _ := util.PodExec("redis-client", clientPod, "redis-client",
		fmt.Sprintf("redis-cli -h %s set a 1", singleIP),
		false, "")
	if strings.TrimSpace(out) != "OK" {
		t.Fatalf("redis-cli set command execute failed: %s", out)
	}
	answer, _ := util.PodExec("redis-client", clientPod, "redis-client",
		fmt.Sprintf("redis-cli -h %s get a", singleIP),
		false, "")
	if strings.TrimSpace(answer) != "1" {
		t.Fatalf("redis-cli get value failed: %s", answer)
	}
}

func getRedisProxyListener(dump map[string]interface{}, vip string) string {
	p := jsonpath.New("redis-listener-filter")
	err := p.Parse("{ .configs[?(@.dynamic_listeners)].dynamic_listeners[?(@.name=='" + vip + "_6379')].active_state.listener.filter_chains[*].filters[?(@.name=='envoy.filters.network.redis_proxy')] }")
	if err != nil {
		panic(err)
	}
	results, _ := p.FindResults(dump)

	for _, result := range results {
		for _, value := range result {
			d, _ := json.MarshalIndent(value.Interface(), "  ", "  ")
			return string(d)
		}
	}
	return ""
}

func TestKeyPrefixRoute(t *testing.T) {
	const prefix = "cluster"
	const auth = "testredis"
	singleIP, _ := util.GetServiceIP("redis-single", "redis", "")
	clusterIP, _ := util.GetServiceIP("redis-cluster", "redis", "")
	clientPod, _ := util.GetPodName("redis-client", "app=redis-client", "")
	config, _ := util.PodExec("redis-client", clientPod, "istio-proxy", "curl -s 127.0.0.1:15000/config_dump", true, "")
	dumpConf := map[string]interface{}{}
	json.Unmarshal(bytes.TrimSpace([]byte(config)), &dumpConf)
	t.Log("cluster:", getRedisProxyListener(dumpConf, clusterIP))
	t.Log("single:", getRedisProxyListener(dumpConf, singleIP))

	out, _ := util.PodExec("redis-client", clientPod, "redis-client",
		fmt.Sprintf("redis-cli -h %s set %s-a 1", singleIP, prefix),
		false, "")
	if strings.TrimSpace(out) != "OK" {
		t.Fatalf("redis-cli set command execute failed: %s", out)
	}
	answer, _ := util.PodExec("redis-client", clientPod, "redis-client",
		fmt.Sprintf("sh -c 'REDISCLI_AUTH=%s redis-cli -h %s get %s-a'", auth, clusterIP, prefix),
		false, "")
	if strings.TrimSpace(answer) != "" {
		t.Fatalf("redis-cli get value should be empty but got: %s", answer)
	}

	out, _ = util.PodExec("redis-client", clientPod, "redis-client",
		fmt.Sprintf("sh -c 'REDISCLI_AUTH=%s redis-cli -h %s set %s-b 9'", auth, clusterIP, prefix),
		false, "")
	if strings.TrimSpace(out) != "OK" {
		t.Fatalf("redis-cli set command execute failed: %s", out)
	}
	answer, _ = util.PodExec("redis-client", clientPod, "redis-client",
		fmt.Sprintf("redis-cli -h %s get %s-b", singleIP, prefix),
		false, "")
	if strings.TrimSpace(answer) != "" {
		t.Fatalf("redis-cli get value should be empty but got: %s", answer)
	}

	out, _ = util.PodExec("redis-client", clientPod, "redis-client",
		fmt.Sprintf("sh -c 'REDISCLI_AUTH=%s redis-cli -h %s set b 9'", auth, clusterIP),
		false, "")
	if strings.TrimSpace(out) != "OK" {
		t.Fatalf("redis-cli set command execute failed: %s", out)
	}

	answer, _ = util.PodExec("redis-client", clientPod, "redis-client",
		fmt.Sprintf("redis-cli -h %s get b", singleIP),
		false, "")
	if strings.TrimSpace(answer) != "9" {
		t.Fatalf("redis-cli get value should be 9 but got: %s", answer)
	}
}
