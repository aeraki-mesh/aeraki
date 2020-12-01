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

// See https://github.com/istio/istio/pull/29209/files, there is a bug in the Istio 1.8.0
replace istio.io/istio => github.com/zhaohuabing/istio v0.0.0-20201126073354-c3d313b335ea

require (
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/docker/cli v0.0.0-20200130152716-5d0cf8839492
	github.com/docker/go v1.5.1-1 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/envoyproxy/go-control-plane v0.9.8-0.20201019204000-12785f608982
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.3
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/theupdateframework/notary v0.6.1 // indirect
	google.golang.org/grpc v1.33.1
	istio.io/api v0.0.0-20201112235759-fa4ee46c5dc2
	istio.io/istio v0.0.0-20201118224433-c87a4c874df2
	istio.io/pkg v0.0.0-20201112235759-c861803834b2
)
