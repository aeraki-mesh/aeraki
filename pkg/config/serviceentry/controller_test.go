package serviceentry

import (
	"testing"
	"time"

	k8sinformers "k8s.io/client-go/informers"
	fakek8sclientset "k8s.io/client-go/kubernetes/fake"
)

func TestController_nextAvailableIP(t *testing.T) {
	k8sClient := fakek8sclientset.NewSimpleClientset()
	sharedK8sInformerFactory := k8sinformers.NewSharedInformerFactory(k8sClient, time.Duration(time.Hour))
	serviceInformer := sharedK8sInformerFactory.Core().V1().Services().Informer()

	c := &Controller{
		serviceIPs: make(map[string]string),
		maxIP:      0,
		informer:   serviceInformer,
	}

	if got := c.nextAvailableIP(); got != "240.240.0.1" {
		t.Errorf("nextAvailableIP() = %v, want %v", got, "240.240.0.1")
	}
	c.serviceIPs["240.240.0.1"] = "service1"

	if got := c.nextAvailableIP(); got != "240.240.0.2" {
		t.Errorf("nextAvailableIP() = %v, want %v", got, "240.240.0.2")
	}
	c.serviceIPs["240.240.0.2"] = "service2"

	if got := c.nextAvailableIP(); got != "240.240.0.3" {
		t.Errorf("nextAvailableIP() = %v, want %v", got, "240.240.0.3")
	}
	c.serviceIPs["240.240.0.3"] = "service3"

	c.maxIP = 255
	if got := c.nextAvailableIP(); got != "240.240.1.1" {
		t.Errorf("nextAvailableIP() = %v, want %v", got, "240.240.1.1")
	}
	c.maxIP = 256
	if got := c.nextAvailableIP(); got != "240.240.1.1" {
		t.Errorf("nextAvailableIP() = %v, want %v", got, "240.240.1.1")
	}
	c.maxIP = 255*254 + 100
	if got := c.nextAvailableIP(); got != "240.240.254.100" {
		t.Errorf("nextAvailableIP() = %v, want %v", got, "240.240.254.100")
	}
}
