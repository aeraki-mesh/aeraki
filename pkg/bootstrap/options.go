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
	"github.com/aeraki-framework/aeraki/pkg/envoyfilter"
	"github.com/aeraki-framework/aeraki/pkg/model/protocol"
)

// AerakiArgs provides all of the configuration parameters for the Aeraki service.
type AerakiArgs struct {
	IstiodAddr string
	ListenAddr string
	Protocols  map[protocol.Instance]envoyfilter.Generator
}

func NewAerakiArgs() *AerakiArgs {
	return &AerakiArgs{}
}
