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
	"crypto/tls"

	"github.com/aeraki-mesh/aeraki/internal/ca"
)

func (s *Server) initRootCA() error {
	s.certMu.Lock()
	defer s.certMu.Unlock()

	if s.istiodCert == nil {
		bundle, err := ca.GenerateKeyCertBundle(s.kubeClient, s.args.RootNamespace)
		if err != nil {
			return err
		}
		x509Cert, err := tls.X509KeyPair(bundle.CertPem.Bytes(), bundle.KeyPem.Bytes())
		if err != nil {
			return err
		}
		s.istiodCert = &x509Cert
		s.CABundle = bundle.CABundle
	}
	return nil
}
