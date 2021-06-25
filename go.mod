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

// Pending https://github.com/kubernetes/kube-openapi/pull/220
replace k8s.io/kube-openapi => github.com/howardjohn/kube-openapi v0.0.0-20210104181841-c0b40d2cb1c8

replace istio.io/api => istio.io/api v0.0.0-20210218044411-561dc276d04d

replace istio.io/client-go => istio.io/client-go v1.9.1-0.20210224044613-d50a7c1b358b

replace istio.io/gogo-genproto => istio.io/gogo-genproto v0.0.0-20210204223132-432f642bc065

replace istio.io/pkg => istio.io/pkg v0.0.0-20201230223204-2d0a1c8bd9e5

replace istio.io/istio => istio.io/istio v0.0.0-20210226235243-2dd7b6207f02

require (
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/envoyproxy/go-control-plane v0.9.9-0.20210115003313-31f9241a16e6
	github.com/fatih/color v1.10.0
	github.com/go-logr/logr v0.3.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.4
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/gosuri/uitable v0.0.4
	github.com/hashicorp/go-multierror v1.1.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.4
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/zhaohuabing/debounce v1.0.0
	golang.org/x/net v0.0.0-20210226172049-e18ecbb05110
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/grpc v1.33.2
	google.golang.org/protobuf v1.25.0
	istio.io/api v0.0.0-20210302211031-2e1e4d7e6f4b
	istio.io/client-go v1.9.1
	istio.io/gogo-genproto v0.0.0-20210302011020-ae262edaabe3
	istio.io/istio v0.0.0-20210304052440-b811231b14cf
	istio.io/pkg v0.0.0-20210302010922-525eaee65cc5
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/client-go v0.20.4
	k8s.io/component-base v0.20.2
	k8s.io/klog/v2 v2.4.0
	sigs.k8s.io/controller-runtime v0.8.2
)
