package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/aeraki-framework/aeraki/pkg/mcp"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/model"

	"github.com/aeraki-framework/aeraki/pkg/config"
	"github.com/aeraki-framework/aeraki/pkg/envoyfilter"
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

	consulMcpServer := mcp.NewServer(*listeningAddress, *listeningAddress)
	go consulMcpServer.Start()

	configController := config.NewController(*istiodAddr)
	configController.RegisterEventHandler("dubbo", func(_, curr model.Config, event model.Event) {
		log.Infof("ServiceEntry: %s", curr.Name)
		filter := envoyfilter.Generate(curr.Spec.(*networking.ServiceEntry))
		log.Infof("create fitler: ", filter)
	})
	err := configController.Run(stopChan)
	if err != nil {
		log.Errorf("Failed to start configController: %v", err)
		return
	}
}
