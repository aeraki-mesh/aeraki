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

package kube

import (
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestController_nextAvailableIP(t *testing.T) {

	c := &serviceEntryController{
		serviceIPs: make(map[string]client.ObjectKey),
		maxIP:      0,
	}

	if got := c.nextAvailableIP(); got != "240.240.0.1" {
		t.Errorf("nextAvailableIP() = %v, want %v", got, "240.240.0.1")
	}
	c.serviceIPs["240.240.0.1"] = client.ObjectKey{
		Namespace: "test",
		Name:      "service1",
	}

	if got := c.nextAvailableIP(); got != "240.240.0.2" {
		t.Errorf("nextAvailableIP() = %v, want %v", got, "240.240.0.2")
	}
	c.serviceIPs["240.240.0.2"] = client.ObjectKey{
		Namespace: "test",
		Name:      "service2",
	}

	if got := c.nextAvailableIP(); got != "240.240.0.3" {
		t.Errorf("nextAvailableIP() = %v, want %v", got, "240.240.0.3")
	}
	c.serviceIPs["240.240.0.3"] = client.ObjectKey{
		Namespace: "test",
		Name:      "service3",
	}

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
