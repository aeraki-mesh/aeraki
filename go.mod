module github.com/aeraki-mesh/aeraki

go 1.16

replace github.com/spf13/viper => github.com/istio/viper v1.3.3-0.20190515210538-2789fed3109c

// Old version had no license
replace github.com/chzyer/logex => github.com/chzyer/logex v1.1.11-0.20170329064859-445be9e134b2

// Avoid pulling in incompatible libraries
replace github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d

replace github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible

// Client-go does not handle different versions of mergo due to some breaking changes - use the matching version
replace github.com/imdario/mergo => github.com/imdario/mergo v0.3.5

//replace github.com/envoyproxy/go-control-plane => /Users/huabingzhao/workspace/go-control-plane

//replace github.com/aeraki-mesh/meta-protocol-control-plane-api => github.com/aeraki-mesh/meta-protocol-control-plane-api v0.0.0-20220325074604-63adf119a7bc

require (
	github.com/aeraki-mesh/meta-protocol-control-plane-api v0.0.0-20220515142731-39ec5b3fe065
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/envoyproxy/go-control-plane v0.10.2-0.20211130161932-f62def555c97
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.6
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/uuid v1.3.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/pkg/errors v0.9.1
	github.com/zhaohuabing/debounce v1.0.0
	go.uber.org/atomic v1.9.0
	golang.org/x/net v0.0.0-20211020060615-d418f374d309
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/grpc v1.42.0
	google.golang.org/protobuf v1.27.2-0.20220217170731-3992ea83a23c
	istio.io/api v0.0.0-20220413220906-0d07ea5cbef8
	istio.io/client-go v1.12.7-0.20220413221605-4b21f100d914
	istio.io/gogo-genproto v0.0.0-20220413221206-c6177de3a4de
	istio.io/istio v0.0.0-20220502132137-56f057aaaf2a
	istio.io/pkg v0.0.0-20220413221105-d9bc5148f7a7
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	sigs.k8s.io/controller-runtime v0.10.2
)
