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
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"istio.io/pkg/log"

	flag "github.com/spf13/pflag"

	"github.com/aeraki-mesh/aeraki/internal/bootstrap"
	"github.com/aeraki-mesh/aeraki/internal/config/constants"
	"github.com/aeraki-mesh/aeraki/internal/envoyfilter"
	"github.com/aeraki-mesh/aeraki/internal/model/protocol"
	"github.com/aeraki-mesh/aeraki/internal/plugin/kafka"
	"github.com/aeraki-mesh/aeraki/internal/plugin/metaprotocol"
	"github.com/aeraki-mesh/aeraki/internal/plugin/thrift"
	"github.com/aeraki-mesh/aeraki/internal/plugin/zookeeper"
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
	flag.BoolVar(&args.Master, "master", true, "Run as master")
	flag.BoolVar(&args.EnableEnvoyFilterNSScope, "enable-envoy-filter-namespace-scope", false,
		"Generate Envoy Filters in the service namespace")
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
	flag.StringVar(&args.KubeDomainSuffix, "domain", defaultKubernetesDomain, "Kubernetes DNS domain suffix")
	flag.StringVar(&args.HTTPSAddr, "httpsAddr", ":15017", "validation service HTTPS address")
	flag.StringVar(&args.HTTPAddr, "httpAddr", ":8080", "Aeraki readiness service HTTP address")
	loggingOptions := log.DefaultOptions()
	loggingOptions.AttachFlags(flag.StringArrayVar, flag.StringVar, flag.IntVar, flag.BoolVar)
	flag.Parse()

	flag.VisitAll(func(flag *flag.Flag) {
		log.Infof("Aeraki parameter: %s: %v", flag.Name, flag.Value)
	})

	if args.ServerID == "" {
		args.ServerID = "Aeraki-" + uuid.New().String()
	}

	initArgsWithEnv(args)
	log.Infof("Aeraki bootstrap parameter: %v", args)

	setLogLevels(args.LogLevel)
	if err := log.Configure(loggingOptions); err != nil {
		log.Error("Failed to init Aeraki log: %v", err)
	}
	// Create the stop channel for all of the servers.
	stopChan := make(chan struct{}, 1)
	args.Protocols = initGenerators()
	server, err := bootstrap.NewServer(args)
	if err != nil {
		log.Fatalf("Failed to init Aeraki :%v", err)
	}
	server.Start(stopChan)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	stopChan <- struct{}{}
}

func initArgsWithEnv(args *bootstrap.AerakiArgs) {
	xdsAddr := os.Getenv("AERAKI_XDS_ADDR")
	if xdsAddr != "" {
		args.AerakiXdsAddr = xdsAddr
	}

	xdsPort := os.Getenv("AERAKI_XDS_PORT")
	if xdsPort != "" {
		args.AerakiXdsPort = xdsPort
	}

	istiodAddr := os.Getenv("AERAKI_ISTIOD_ADDR")
	if istiodAddr != "" {
		args.IstiodAddr = istiodAddr
	}

	namespace := os.Getenv("AERAKI_NAMESPACE")
	if namespace != "" {
		args.RootNamespace = namespace
	}

	clusterID := os.Getenv("AERAKI_CLUSTER_ID")
	if clusterID != "" {
		args.ClusterID = clusterID
	}

	secret := os.Getenv("AERAKI_ISTIO_CONFIG_STORE_SECRET")
	if secret != "" {
		args.ConfigStoreSecret = secret
	}

	logLevel := os.Getenv("AERAKI_LOG_LEVEL")
	if logLevel != "" {
		args.LogLevel = logLevel
	}

	podName := os.Getenv("POD_NAME")
	if podName != "" {
		args.PodName = os.Getenv("POD_NAME")
	} else {
		args.PodName = args.ServerID
	}
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
