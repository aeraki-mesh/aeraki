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

package protocol

import "strings"

var cache map[string]Instance = make(map[string]Instance)

type Instance string

// Unsupported - value to signify that the protocol is unsupported.
const Unsupported Instance = "UnsupportedProtocol"

func Parse(s string) Instance {
	name := strings.ToLower(s)
	if protocol, ok := cache[name]; ok {
		return protocol
	}

	var protocol Instance = Instance(name)
	cache[name] = protocol
	return protocol
}

func GetLayer7ProtocolFromPortName(name string) Instance {
	s := strings.Split(name, "-")
	if len(s) > 1 {
		return Parse(s[1])
	}
	return Unsupported
}

func (i Instance) String() string {
	return string(i)
}
