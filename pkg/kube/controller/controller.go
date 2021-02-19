package controller

import (
	"github.com/aeraki-framework/aeraki/client-go/pkg/clientset/versioned/scheme"
	"istio.io/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var controllerLog = log.RegisterScope("controller", "crd controller", 0)

// NewManager create a manager to manager all crd controllers.
func NewManager(namespace string, electionID string, triggerPush func() error) manager.Manager {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		controllerLog.Fatalf("Could not get apiserver config: %v\n", err)
		return nil
	}
	mgrOpt := manager.Options{
		MetricsBindAddress:      "0",
		LeaderElection:          true,
		LeaderElectionNamespace: namespace,
		LeaderElectionID:        electionID,
	}
	m, err := manager.New(cfg, mgrOpt)
	if err != nil {
		controllerLog.Fatalf("Could not create a controller manager: %v", err)
		return nil
	}

	err = addRedisServiceController(m, triggerPush)
	if err != nil {
		controllerLog.Fatalf("Could not add RedisServiceController: %e", err)
		return nil
	}
	err = addRedisDestinationController(m, triggerPush)
	if err != nil {
		controllerLog.Fatalf("Could not add RedisDestinationController: %e", err)
		return nil
	}
	err = scheme.AddToScheme(m.GetScheme())
	if err != nil {
		controllerLog.Fatalf("Could not add schema: %e", err)
	}
	return m
}
