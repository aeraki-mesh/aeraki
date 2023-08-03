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

package metaprotocolgateway

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	"istio.io/pkg/log"

	"github.com/aeraki-mesh/aeraki/test/e2e/metaprotocolgateway/gen-go/hello"
	"github.com/aeraki-mesh/aeraki/test/e2e/util"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	//shutdown()
	os.Exit(code)
}
func setup() {
	util.CreateNamespace("meta-thrift", "")
	util.LabelNamespace("meta-thrift", "istio-injection=disabled", "")
	util.KubeApply("istio-system", "../../../k8s/aeraki-bootstrap-config.yaml", "")
	util.KubeApply("meta-thrift", "testdata/thrift-sample.yaml", "")    // thrift server/svc
	util.KubeApply("istio-system", "testdata/ingress-gateway.yaml", "") // ingress gateway
	util.KubeApply("meta-thrift", "testdata/destinationrule.yaml", "")
	util.KubeApply("meta-thrift", "testdata/metarouter.yaml", "")
}

func shutdown() {
	util.KubeDelete("meta-thrift", "testdata/metarouter.yaml", "")
	util.KubeDelete("meta-thrift", "testdata/destinationrule.yaml", "")
	util.KubeDelete("meta-thrift", "testdata/virtualservice.yaml", "")
	util.KubeDelete("istio-system", "testdata/ingress-gateway.yaml", "")
	util.KubeDelete("meta-thrift", "testdata/thrift-sample.yaml", "")
	util.DeleteNamespace("meta-thrift", "")
}

func TestThriftRouter(t *testing.T) {
	util.WaitForDeploymentsReady("meta-thrift", 10*time.Minute, "")
	util.WaitForDeploymentsReady("istio-system", 10*time.Minute, "")

	// waiting for gateway listener ready
	time.Sleep(1 * time.Minute)
	svcIP, err := util.GetServiceIP("istio-ingressgateway", "istio-system", "")
	if err != nil {
		t.Errorf("failed to get istio-ingressgateway svcIP")
	}
	var transport thrift.TTransport
	transport, err = thrift.NewTSocket(fmt.Sprintf("%s:9090", svcIP))
	if err != nil {
		fmt.Println("Error opening socket:", err)
	}

	//protocol
	var protocolFactory thrift.TProtocolFactory
	protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()

	//no buffered
	var transportFactory thrift.TTransportFactory
	transportFactory = thrift.NewTTransportFactory()

	transport, err = transportFactory.GetTransport(transport)
	if err != nil {
		fmt.Println("error running client:", err)
	}

	if err := transport.Open(); err != nil {
		fmt.Println("error running client:", err)
	}

	iprot := protocolFactory.GetProtocol(transport)
	oprot := protocolFactory.GetProtocol(transport)
	client := hello.NewHelloServiceClient(thrift.NewTStandardClient(iprot, oprot))

	// v1 route
	util.KubeApply("meta-thrift", "testdata/metarouter-v1.yaml", "")
	time.Sleep(10 * time.Second)
	for i := 0; i < 5; i++ {
		resp, err := client.SayHello(context.TODO(), "AerakiClient")
		if err != nil {
			log.Errorf("err is %v", err)
			t.Errorf("failed to call thrift server")
		}
		log.Info(resp)
		if !strings.Contains(resp, "thrift-sample-server-v1") {
			t.Errorf("Version routing failed, want: v1, got %s", resp)
		}
	}

	// v2 route
	util.KubeApply("meta-thrift", "testdata/metarouter-v2.yaml", "")
	time.Sleep(10 * time.Second)
	for i := 0; i < 5; i++ {
		resp, err := client.SayHello(context.TODO(), "AerakiClient")
		if err != nil {
			log.Errorf("err is %v", err)
			t.Errorf("failed to call thrift server")
		}
		log.Info(resp)
		if !strings.Contains(resp, "thrift-sample-server-v2") {
			t.Errorf("Version routing failed, want: v2, got %s", resp)
		}
	}

}
