package envoyfilter

import (
	"testing"

	"github.com/sirupsen/logrus"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pkg/config/xds"
)

func Test_buildDubboProxy(t *testing.T) {

	//buf := &bytes.Buffer{}
	//
	//_ = (&jsonpb.Marshaler{OrigName: true}).Marshal(buf, buildDubboProxy("xxx",100))
	//
	////fmt.Printf("%v",buf)
	//var out = &types.Struct{}
	//(&jsonpb.Unmarshaler{AllowUnknownFields: false}).Unmarshal(buf, out)
	//
	////fmt.Printf("%v",out)
	//
	//for k,v := range out.Fields {
	//	fmt.Println()
	//	fmt.Printf("%v",k)
	//	fmt.Println()
	//	fmt.Printf("%v",v)
	//}
	service := &networking.ServiceEntry{
		Hosts:     []string{"test.service"},
		Addresses: []string{"10.10.10.0"},
		Ports:     []*networking.Port{&networking.Port{Number: 8888, Protocol: "tcp-dubbo-test"}},
	}

	filter := Generate(service)

	v, _ := xds.BuildXDSObjectFromStruct(networking.EnvoyFilter_NETWORK_FILTER, filter.ConfigPatches[0].Patch.Value, false)

	logrus.Infof("%v", v)
}
