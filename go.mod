module github.com/aeraki-framework/aeraki

go 1.15

replace github.com/spf13/viper => github.com/istio/viper v1.3.3-0.20190515210538-2789fed3109c

// Old version had no license
replace github.com/chzyer/logex => github.com/chzyer/logex v1.1.11-0.20170329064859-445be9e134b2

// Avoid pulling in incompatible libraries
replace github.com/docker/distribution => github.com/docker/distribution v2.7.1+incompatible

// Avoid pulling in kubernetes/kubernetes
replace github.com/Microsoft/hcsshim => github.com/Microsoft/hcsshim v0.8.8-0.20200421182805-c3e488f0d815

// Client-go does not handle different versions of mergo due to some breaking changes - use the matching version
replace github.com/imdario/mergo => github.com/imdario/mergo v0.3.5

// See https://github.com/kubernetes/kubernetes/issues/92867, there is a bug in the library
replace github.com/evanphx/json-patch => github.com/evanphx/json-patch v0.0.0-20190815234213-e83c0a1c26c8

// There are some bugs in the Istio 1.8.0
// https://github.com/istio/istio/pull/29209
// https://github.com/istio/istio/pull/29296
replace istio.io/istio => github.com/zhaohuabing/istio v0.0.0-20210111232828-ecea0bbe3312

// https://github.com/istio/api/pull/1774 add destination port support for envoyfilter
replace istio.io/api => github.com/istio/api v0.0.0-20201217155105-21c3bd1ba1d3

require (
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/envoyproxy/go-control-plane v0.9.8
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.3
	github.com/golang/sync v0.0.0-20180314180146-1d60e4601c6f
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.0
	github.com/klauspost/compress v1.11.0 // indirect
	github.com/onsi/ginkgo v1.14.2 // indirect
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.3 // indirect
	github.com/stretchr/testify v1.7.0 // indirect
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b
	golang.org/x/tools v0.1.0 // indirect
	google.golang.org/protobuf v1.25.0
	honnef.co/go/tools v0.0.1-2020.1.6 // indirect
	istio.io/api v0.0.0-20210109163259-0575f65cd5df
	istio.io/client-go v0.0.0-20200908160912-f99162621a1a
	istio.io/gogo-genproto v0.0.0-20201015184601-1e80d26d6249
	istio.io/istio v0.0.0-20201118224433-c87a4c874df2
	istio.io/pkg v0.0.0-20201230223204-2d0a1c8bd9e5
	k8s.io/apimachinery v0.20.1
	k8s.io/client-go v0.20.1
	sigs.k8s.io/controller-runtime v0.7.0
)
