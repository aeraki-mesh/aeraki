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
		{testName: "tcp-dubbo", portName: "tcp-dubbo", want: "dubbo"},
		{testName: "tcp-Dubbo", portName: "tcp-Dubbo", want: "dubbo"},
		{testName: "tcp-Dubbo-28001", portName: "tcp-Dubbo-28001", want: "dubbo"},
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
