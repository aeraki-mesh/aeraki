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

import (
	"strings"
)

// Instance defines network protocols for ports
type Instance string

const (
	// Dubbo declares that the port carries dubbo traffic.
	Dubbo Instance = "Dubbo"
	// Thrift declares that the port carries Thrift traffic.
	Thrift Instance = "Thrift"
	// Mongo declares that the port carries MongoDB traffic.
	Mongo Instance = "Mongo"
	// Redis declares that the port carries Redis traffic.
	Redis Instance = "Redis"
	// MySQL declares that the port carries MySQL traffic.
	MySQL Instance = "MySQL"
	// Kafka declares that the port carries Kafka traffic.
	Kafka Instance = "Kafka"
	// Zookeeper declares that the port carries Zookeeper traffic.
	Zookeeper Instance = "Zookeeper"
	// MetaProtocol declares that the port carries MetaProtocol traffic.
	MetaProtocol Instance = "MetaProtocol"
	// Unsupported - value to signify that the protocol is unsupported.
	Unsupported Instance = "UnsupportedProtocol"
)

var protocolMap = make(map[string]Instance)

// nolint: gochecknoinits
func init() {
	protocolMap["dubbo"] = Dubbo
	protocolMap["thrift"] = Thrift
	protocolMap["mongo"] = Mongo
	protocolMap["redis"] = Redis
	protocolMap["mysql"] = MySQL
	protocolMap["kafka"] = Kafka
	protocolMap["zookeeper"] = Zookeeper
	protocolMap["metaprotocol"] = MetaProtocol
}

// RegisterProtocol register custom protocol
func RegisterProtocol(name string, protocol Instance) {
	protocolMap[name] = protocol
}

// Parse from string ignoring case
func Parse(s string) Instance {
	if instance, ok := protocolMap[strings.ToLower(s)]; ok {
		return instance
	}
	return Unsupported
}

// IsDubbo is true for protocols that use Dubbo as transport protocol
func (i Instance) IsDubbo() bool {
	switch i {
	case Dubbo:
		return true
	default:
		return false
	}
}

// IsThrift is true for protocols that use Thrift as transport protocol
func (i Instance) IsThrift() bool {
	switch i {
	case Thrift:
		return true
	default:
		return false
	}
}

// IsMetaProtocol is true for protocols that use MetaProtocol as transport protocol
func (i Instance) IsMetaProtocol() bool {
	switch i {
	case MetaProtocol:
		return true
	default:
		return false
	}
}

// IsUnsupported is true for protocols that are not supported
func (i Instance) IsUnsupported() bool {
	return i == Unsupported
}

// ToString converts an Instance to a string
func (i Instance) ToString() string {
	return string(i)
}

// GetLayer7ProtocolFromPortName extracts the layer-7 protocol name from the port name of a service
func GetLayer7ProtocolFromPortName(name string) Instance {
	s := strings.Split(name, "-")
	if len(s) > 1 {
		return Parse(s[1])
	}
	return Unsupported
}

// IsAerakiSupportedProtocols return true if the protocol is supported by Aeraki, false if not
func IsAerakiSupportedProtocols(name string) bool {
	protocol := GetLayer7ProtocolFromPortName(name)
	return protocol != Unsupported
}
