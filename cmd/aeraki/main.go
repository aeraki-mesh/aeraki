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

package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/google/uuid"

	"github.com/aeraki-mesh/aeraki/pkg/bootstrap"
	"github.com/aeraki-mesh/aeraki/pkg/config/constants"
	"github.com/aeraki-mesh/aeraki/pkg/envoyfilter"
	"github.com/aeraki-mesh/aeraki/pkg/model/protocol"
	"github.com/aeraki-mesh/aeraki/plugin/kafka"
	"github.com/aeraki-mesh/aeraki/plugin/metaprotocol"
	"github.com/aeraki-mesh/aeraki/plugin/thrift"
	"github.com/aeraki-mesh/aeraki/plugin/zookeeper"

	"istio.io/pkg/env"
	"istio.io/pkg/log"
)

const (
	defaultIstiodAddr        = "istiod.istio-system:15010"
	defaultRootNamespace     = "istio-system"
	defaultElectionID        = "aeraki-controller"
	defaultLogLevel          = "all:info"
	defaultConfigStoreSecret = ""
	defaultKubernetesDomain  = "cluster.local"
	defaultMeshConfigMapName = "istio"
)

func main() {
	args := bootstrap.NewAerakiArgs()
	flag.BoolVar(&args.Master, "master", true, "Istiod xds server address")
	flag.StringVar(&args.AerakiXdsAddr, "aeraki-xds-address", constants.DefaultAerakiXdsAddr, "Aeraki xds server address")
	flag.StringVar(&args.AerakiXdsPort, "aeraki-xds-port", constants.DefaultAerakiXdsPort, "Aeraki xds server port")
	flag.StringVar(&args.IstiodAddr, "istiod-address", defaultIstiodAddr, "Istiod xds server address")
	flag.StringVar(&args.IstioConfigMapName, "istiod-configMap-name", defaultMeshConfigMapName, "Istiod configMap name")
	flag.StringVar(&args.RootNamespace, "root-namespace", defaultRootNamespace, "The Root Namespace of Aeraki")
	flag.StringVar(&args.ClusterID, "cluster-id", "", "The cluster where Aeraki is deployed")
	flag.StringVar(&args.ConfigStoreSecret, "config-store-secret", defaultConfigStoreSecret,
		"The secret to store the Istio kube config store, use the in cluster API server if it's not specified")
	flag.StringVar(&args.ElectionID, "election-id", defaultElectionID, "ElectionID to elect master controller")
	flag.StringVar(&args.ServerID, "server-id", "", "Aeraki server id")
	flag.StringVar(&args.LogLevel, "log-level", defaultLogLevel, "Component log level")
	flag.BoolVar(&args.EnableEnvoyFilterNSScope, "enable-envoy-filter-namespace-scope", false,
		"Generate Envoy Filters in the service namespace")
	flag.StringVar(&args.KubeDomainSuffix, "domain", defaultKubernetesDomain, "Kubernetes DNS domain suffix")
	flag.StringVar(&args.HTTPSAddr, "httpsAddr", ":15017", "validation service HTTPS address")

	flag.Parse()
	if args.ServerID == "" {
		args.ServerID = "Aeraki-" + uuid.New().String()
	}

	args.PodName = env.RegisterStringVar("POD_NAME", args.ServerID, "").Get()
	args.RootNamespace = env.RegisterStringVar("AERAKI_NAMESPACE", args.RootNamespace, "").Get()
	args.EnableEnvoyFilterNSScope = env.RegisterBoolVar(constants.DefaultAerakiEnableEnvoyFilterNsScope,
		args.EnableEnvoyFilterNSScope, "").Get()
	args.IstiodAddr = env.RegisterStringVar("AERAKI_ISTIOD_ADDR", args.IstiodAddr, "").Get()
	args.AerakiXdsAddr = env.RegisterStringVar("AERAKI_XDS_ADDR", constants.DefaultAerakiXdsAddr, "").Get()
	args.AerakiXdsPort = env.RegisterStringVar("AERAKI_XDS_PORT", constants.DefaultAerakiXdsPort, "").Get()

	flag.VisitAll(func(flag *flag.Flag) {
		log.Infof("Aeraki parameter: %s: %v", flag.Name, flag.Value)
	})

	setLogLevels(args.LogLevel)
	// Create the stop channel for all of the servers.
	stopChan := make(chan struct{}, 1)
	args.Protocols = initGenerators()
	server, err := bootstrap.NewServer(args)
	if err != nil {
		log.Fatalf("Failed to init Aeraki :%v", err)
		os.Exit(1)
	}
	server.Start(stopChan)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	stopChan <- struct{}{}
}

func initGenerators() map[protocol.Instance]envoyfilter.Generator {
	return map[protocol.Instance]envoyfilter.Generator{
		protocol.Thrift:       thrift.NewGenerator(),
		protocol.Kafka:        kafka.NewGenerator(),
		protocol.Zookeeper:    zookeeper.NewGenerator(),
		protocol.MetaProtocol: metaprotocol.NewGenerator(),
	}
}

func setLogLevels(level string) {
	logOpts := log.DefaultOptions()
	levels := strings.Split(level, ",")
	for _, l := range levels {
		cl := strings.Split(l, ":")
		if len(cl) != 2 {
			continue
		}
		logOpts.SetOutputLevel(cl[0], stringToLevel[cl[1]])
	}
	_ = log.Configure(logOpts)
}

// this is the same as istio.io/pkg/log.stringToLevel
var stringToLevel = map[string]log.Level{
	"debug": log.DebugLevel,
	"info":  log.InfoLevel,
	"warn":  log.WarnLevel,
	"error": log.ErrorLevel,
	"fatal": log.FatalLevel,
	"none":  log.NoneLevel,
}
