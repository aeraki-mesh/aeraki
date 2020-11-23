module github.com/aeraki-framework/aeraki

go 1.14

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

replace istio.io/istio => /Users/huabingzhao/go/src/istio.io/istio

require (
	github.com/Jeffail/gabs/v2 v2.6.0
	github.com/envoyproxy/go-control-plane v0.9.8-0.20201019204000-12785f608982
	github.com/gogo/protobuf v1.3.1
	github.com/google/uuid v1.1.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/istio-ecosystem/consul-mcp v0.0.0-20201014012513-6b638508c74b // indirect
	github.com/sirupsen/logrus v1.6.0
	google.golang.org/grpc v1.33.1
	istio.io/api v0.0.0-20201019135039-64b3eaad773f
	istio.io/istio v0.0.0-20201118224433-c87a4c874df2
	istio.io/pkg v0.0.0-20200831193257-fe7110296cbc
)
