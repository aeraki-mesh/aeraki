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

package ca

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	kubelib "istio.io/istio/pkg/kube"
)

// KeyCertBundle stores the cert, private key and root cert for aeraki.
type KeyCertBundle struct {
	CertPem  *bytes.Buffer
	KeyPem   *bytes.Buffer
	CABundle *bytes.Buffer
}

// GenerateKeyCertBundle generates root ca and server certificate
func GenerateKeyCertBundle(client kubelib.Client, namespace string) (*KeyCertBundle, error) {
	caKeyCertBundle, err := getIstioCA(client.Kube().CoreV1(), namespace)
	if err != nil {
		return nil, err
	}

	caCert, caPrivKey, _, _ := caKeyCertBundle.GetAll()

	dnsNames := []string{"aeraki",
		"aeraki." + namespace,
		"aeraki." + namespace + ".svc"}
	commonName := "aeraki.default.svc"
	// server cert config
	cert := &x509.Certificate{
		SignatureAlgorithm: x509.SHA256WithRSA,
		DNSNames:           dnsNames,
		SerialNumber:       big.NewInt(1658),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"aeraki.net"},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		Version:      3,
	}

	// server private key
	serverPrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	// sign the server cert
	serverCertBytes, err := x509.CreateCertificate(cryptorand.Reader, cert, caCert, &serverPrivKey.PublicKey,
		*caPrivKey)
	if err != nil {
		return nil, err
	}

	// PEM encode the  server cert and key
	serverCertPEM := new(bytes.Buffer)
	_ = pem.Encode(serverCertPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: serverCertBytes,
	})

	serverPrivKeyPEM := new(bytes.Buffer)
	_ = pem.Encode(serverPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(serverPrivKey),
	})
	// PEM encode CA cert
	caPEM := new(bytes.Buffer)
	caPEM.Write(caKeyCertBundle.GetRootCertPem())

	return &KeyCertBundle{
		CertPem:  serverCertPEM,
		KeyPem:   serverPrivKeyPEM,
		CABundle: caPEM,
	}, nil
}
