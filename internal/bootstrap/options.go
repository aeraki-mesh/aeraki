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

package bootstrap

import (
	"github.com/aeraki-mesh/aeraki/internal/envoyfilter"
	"github.com/aeraki-mesh/aeraki/internal/model/protocol"
)

// AerakiArgs provides all of the configuration parameters for the Aeraki service.
type AerakiArgs struct {
	Master                   bool
	IstiodAddr               string
	AerakiXdsAddr            string
	AerakiXdsPort            string
	PodName                  string
	IstioConfigMapName       string
	HTTPSAddr                string // The listening address for HTTPS (webhooks).
	HTTPAddr                 string // The listening address for HTTP (health).
	RootNamespace            string
	ClusterID                string
	ConfigStoreSecret        string
	ElectionID               string
	ServerID                 string
	LogLevel                 string
	KubeDomainSuffix         string
	EnableEnvoyFilterNSScope bool
	Protocols                map[protocol.Instance]envoyfilter.Generator
}

// NewAerakiArgs constructs AerakiArgs with default value.
func NewAerakiArgs() *AerakiArgs {
	return &AerakiArgs{}
}
