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
	"syscall"

	"github.com/aeraki-framework/aeraki/pkg/bootstrap"
	"github.com/aeraki-framework/aeraki/pkg/model/protocol"
	"github.com/aeraki-framework/aeraki/plugin/dubbo"
)

const (
	defaultIstiodAddr    = "istiod.istio-system:15010"
	defaultListeningAddr = ":1109"
)

func main() {
	args := bootstrap.NewAerakiArgs()
	args.IstiodAddr = *flag.String("istiodaddr", defaultIstiodAddr, "Istiod xds server address")
	args.ListenAddr = *flag.String("listeningAddr", defaultListeningAddr, "MCP listening Address")
	flag.Parse()
	args.Protocol = protocol.Parse("dubbo")
	args.Generator = dubbo.NewGenerator()
	server := bootstrap.NewServer(args)

	// Create the stop channel for all of the servers.
	stopChan := make(chan struct{}, 1)
	server.Start(stopChan)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	stopChan <- struct{}{}
}
