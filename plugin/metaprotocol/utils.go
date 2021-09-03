// Copyright 2020 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package metaprotocol

import "strings"

var codec = map[string]string{ //todo make this configuration via crd
	"dubbo":  "aeraki.meta_protocol.codec.dubbo",
	"thrift": "aeraki.meta_protocol.codec.thrift",
}

// GetApplicationProtocolFromMetaProtocol extracts the application protocol name from metaprotocol port name
func getApplicationProtocol(portName string) string {
	s := strings.Split(portName, "-")
	if len(s) > 1 {
		return s[2]
	}
	return ""
}

func getCodec(applicationProtocol string) string {
	return codec[applicationProtocol]
}
