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
	"fmt"
	"net/http"
	"time"

	"github.com/aeraki-mesh/aeraki/internal/webhook/validation"
)

// initSSecureWebhookServer handles initialization for the HTTPS webhook server.
func (s *Server) initSecureWebhookServer(args *AerakiArgs) error {
	aerakiLog.Info("initializing secure webhook server for aeraki webhooks")
	// create the https server for hosting the k8s mutationwebhook handlers.
	s.httpsMux = http.NewServeMux()
	s.httpsServer = &http.Server{
		Addr:    args.HTTPSAddr,
		Handler: s.httpsMux,
		TLSConfig: &tls.Config{
			GetCertificate: s.getAerakiCertificate,
			MinVersion:     tls.VersionTLS12,
		},
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Generate Webhook configuration
	if err := validation.GenerateWebhookConfig(s.CABundle, args.RootNamespace); err != nil {
		return fmt.Errorf("failed to generate webhook cofigruation %v", err)
	}
	return nil
}

// getAerakiCertificate returns the aeraki certificate.
func (s *Server) getAerakiCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	s.certMu.RLock()
	defer s.certMu.RUnlock()
	if s.istiodCert != nil {
		return s.istiodCert, nil
	}
	return nil, fmt.Errorf("cert not initialized")
}
