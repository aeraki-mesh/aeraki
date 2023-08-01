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
	"istio.io/pkg/log"

	"github.com/aeraki-mesh/aeraki/internal/webhook/validation/scheme"
	"github.com/aeraki-mesh/aeraki/internal/webhook/validation/server"
)

func (s *Server) initConfigValidation(args *AerakiArgs) error {
	if s.kubeClient == nil {
		return nil
	}

	log.Info("initializing config validator")
	// always start the validation server
	params := server.Options{
		Schemas:      scheme.Aeraki,
		DomainSuffix: args.KubeDomainSuffix,
		Mux:          s.httpsMux,
	}
	_, err := server.New(params)
	if err != nil {
		return err
	}

	return nil
}
