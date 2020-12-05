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

package protocol_test

import (
	"testing"

	"github.com/aeraki-framework/aeraki/pkg/model/protocol"
)

func Test_getLayer7ProtocolFromPortName(t *testing.T) {
	tests := []struct {
		testName string
		portName string
		want     protocol.Instance
	}{
		{testName: "tcp-dubbo", portName: "tcp-dubbo", want: protocol.Dubbo},
		{testName: "tcp-Dubbo", portName: "tcp-Dubbo", want: protocol.Dubbo},
		{testName: "tcp-Dubbo-28001", portName: "tcp-Dubbo-28001", want: protocol.Dubbo},
		{testName: "Dubbo", portName: "Dubbo", want: protocol.Unsupported},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			if got := protocol.GetLayer7ProtocolFromPortName(tt.portName); got != tt.want {
				t.Errorf("getLayer7ProtocolFromPortName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDubbo(t *testing.T) {
	if !protocol.Dubbo.IsDubbo() {
		t.Errorf("Dubbo should be Dubbo")
	}
	if protocol.Dubbo.IsThrift() {
		t.Errorf("Dubbo is not Thrift")
	}
}
