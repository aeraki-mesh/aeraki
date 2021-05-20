package controller

import (
	"github.com/aeraki-framework/aeraki/client-go/pkg/clientset/versioned/scheme"
	"istio.io/pkg/log"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var controllerLog = log.RegisterScope("controller", "crd controller", 0)

// NewManager create a manager to manager all crd controllers.
func NewManager(kubeConfig *rest.Config, namespace string, electionID string,
	triggerPush func() error) manager.Manager {
	mgrOpt := manager.Options{
		MetricsBindAddress:      "0",
		LeaderElection:          true,
		LeaderElectionNamespace: namespace,
		LeaderElectionID:        electionID,
	}
	m, err := manager.New(kubeConfig, mgrOpt)
	if err != nil {
		controllerLog.Fatalf("Could not create a controller manager: %v", err)
	}

	err = addRedisServiceController(m, triggerPush)
	if err != nil {
		controllerLog.Fatalf("Could not add RedisServiceController: %e", err)
	}
	err = addRedisDestinationController(m, triggerPush)
	if err != nil {
		controllerLog.Fatalf("Could not add RedisDestinationController: %e", err)
	}
	err = addDubboAuthorizationPolicyController(m, triggerPush)
	if err != nil {
		controllerLog.Fatalf("Could not add DubboAuthorizationPolicyController: %e", err)
	}
	err = scheme.AddToScheme(m.GetScheme())
	if err != nil {
		controllerLog.Fatalf("Could not add schema: %e", err)
	}
	return m
}
