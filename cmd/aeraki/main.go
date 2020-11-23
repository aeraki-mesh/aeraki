package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/aeraki-framework/aeraki/pkg/envoyfilter"
	"github.com/aeraki-framework/aeraki/pkg/mcp"
	"istio.io/istio/pilot/pkg/model"

	"github.com/aeraki-framework/aeraki/pkg/config"
	"istio.io/pkg/log"
)

const (
	defaultIstiodAddr       = "istiod.istio-system:15010"
	defaultListeningAddress = ":1109"
)

func main() {
	log.Infof("Start server...")
	istiodAddr := flag.String("istiodaddr", defaultIstiodAddr, "Istiod xds server address")
	listeningAddress := flag.String("listeningAddress", defaultListeningAddress, "Listening Address")
	*listeningAddress = "xxx"
	flag.Parse()

	// Create the stop channel for all of the servers.
	stopChan := make(chan struct{}, 1)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChan
		stopChan <- struct{}{}
	}()

	configController := config.NewController(*istiodAddr)
	consulMcpServer := mcp.NewServer(*listeningAddress, *listeningAddress, configController.Store, envoyfilter.NewDubboGenerator())
	configController.RegisterEventHandler("dubbo", func(_, curr model.Config, event model.Event) {
		log.Infof("ServiceEntry: %s", curr.Name)
		//filter := envoyfilter.Generate(curr.Spec.(*networking.ServiceEntry))
		//log.Infof("create fitler: ", filter)
		consulMcpServer.ConfigUpdate(event)
	})

	go consulMcpServer.Start()
	err := configController.Run(stopChan)
	if err != nil {
		log.Errorf("Failed to start configController: %v", err)
		return
	}
}
