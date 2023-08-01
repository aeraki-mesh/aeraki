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

package metaprotocol

import (
	"fmt"
	"strings"
	"sync"
)

var applicationProtocols sync.Map

// nolint: gochecknoinits
func init() {
	applicationProtocols.Store("dubbo", "aeraki.meta_protocol.codec.dubbo")
	applicationProtocols.Store("thrift", "aeraki.meta_protocol.codec.thrift")
}

// SetApplicationProtocolCodec sets the codec for a specific protocol
func SetApplicationProtocolCodec(protocol, codec string) {
	applicationProtocols.Store(protocol, codec)
}

// GetApplicationProtocolCodec get the codec for a specific protocol
func GetApplicationProtocolCodec(protocol string) (string, error) {
	codec, ok := applicationProtocols.Load(protocol)
	if !ok {
		return "", fmt.Errorf("can't find codec for protocol: %s", protocol)
	}
	return fmt.Sprintf("%s", codec), nil
}

// GetApplicationProtocolFromPortName extracts the application protocol name from metaprotocol port name
func GetApplicationProtocolFromPortName(portName string) (string, error) {
	s := strings.Split(portName, "-")
	if len(s) > 1 {
		return s[2], nil
	}
	return "", fmt.Errorf("can't find application protocol in port name: %s", portName)
}
