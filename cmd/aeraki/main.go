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

	"github.com/aeraki-framework/aeraki/pkg/envoyfilter"
	"github.com/aeraki-framework/aeraki/plugin/kafka"
	"github.com/aeraki-framework/aeraki/plugin/thrift"
	"github.com/aeraki-framework/aeraki/plugin/zookeeper"
	"istio.io/pkg/log"

	"github.com/aeraki-framework/aeraki/pkg/bootstrap"
	"github.com/aeraki-framework/aeraki/pkg/model/protocol"
)

const (
	defaultIstiodAddr = "istiod.istio-system:15010"
	defaultNamespace  = "istio-system"
	defaultElectionID = "aeraki-controller"
	defaultLogLevel   = "default:info"
)

func main() {
	args := bootstrap.NewAerakiArgs()
	flag.StringVar(&args.IstiodAddr, "istiod-address", defaultIstiodAddr, "Istiod xds server address")
	flag.StringVar(&args.Namespace, "namespace", defaultNamespace, "The current namespace where Aeraki is deployed")
	flag.StringVar(&args.ConfigStoreSecret, "config-store-secret", defaultNamespace,
		"The secret to store the Istio kube config store, use the in cluster API server if it's not specified")
	flag.StringVar(&args.ElectionID, "electionID", defaultElectionID, "ElectionID to elect master controller")
	flag.StringVar(&args.LogLevel, "log-level", defaultLogLevel, "Component log level")
	flag.Parse()

	setLogLevels(args.LogLevel)
	// Create the stop channel for all of the servers.
	stopChan := make(chan struct{}, 1)
	args.Protocols = initGenerators()
	server, err := bootstrap.NewServer(args)
	if err != nil {
		log.Fatalf("Failed to start Aeraki :%v", err)
	}
	server.Start(stopChan)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	stopChan <- struct{}{}
}

func initGenerators() map[protocol.Instance]envoyfilter.Generator {
	return map[protocol.Instance]envoyfilter.Generator{
		protocol.Thrift:    thrift.NewGenerator(),
		protocol.Kafka:     kafka.NewGenerator(),
		protocol.Zookeeper: zookeeper.NewGenerator(),
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
