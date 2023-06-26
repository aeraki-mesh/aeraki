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
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"istio.io/istio/pkg/security"
	"istio.io/istio/security/pkg/cmd"
	"istio.io/istio/security/pkg/pki/ca"
	"istio.io/istio/security/pkg/pki/ra"
	"istio.io/istio/security/pkg/pki/util"
	"istio.io/pkg/env"
	"istio.io/pkg/log"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	// LocalCertDir replaces the "cert-chain", "signing-cert" and "signing-key" flags in citadel - Istio installer is
	// requires a secret named "cacerts" with specific files inside.
	localCertDir = env.RegisterStringVar("ROOT_CA_DIR", "./etc/cacerts",
		"Location of a local or mounted CA root")

	workloadCertTTL = env.RegisterDurationVar("DEFAULT_WORKLOAD_CERT_TTL",
		cmd.DefaultWorkloadCertTTL,
		"The default TTL of issued workload certificates. Applied when the client sets a "+
			"non-positive TTL in the CSR.")

	maxWorkloadCertTTL = env.RegisterDurationVar("MAX_WORKLOAD_CERT_TTL",
		cmd.DefaultMaxWorkloadCertTTL,
		"The max TTL of issued workload certificates.")

	selfSignedCACertTTL = env.RegisterDurationVar("CITADEL_SELF_SIGNED_CA_CERT_TTL",
		cmd.DefaultSelfSignedCACertTTL,
		"The TTL of self-signed CA root certificate.")

	selfSignedRootCertCheckInterval = env.RegisterDurationVar("CITADEL_SELF_SIGNED_ROOT_CERT_CHECK_INTERVAL",
		cmd.DefaultSelfSignedRootCertCheckInterval,
		"The interval that self-signed CA checks its root certificate "+
			"expiration time and rotates root certificate. Setting this interval "+
			"to zero or a negative value disables automated root cert check and "+
			"rotation. This interval is suggested to be larger than 10 minutes.")

	selfSignedRootCertGracePeriodPercentile = env.RegisterIntVar("CITADEL_SELF_SIGNED_ROOT_CERT_GRACE_PERIOD_PERCENTILE",
		cmd.DefaultRootCertGracePeriodPercentile,
		"Grace period percentile for self-signed root cert.")

	enableJitterForRootCertRotator = env.RegisterBoolVar("CITADEL_ENABLE_JITTER_FOR_ROOT_CERT_ROTATOR",
		true,
		"If true, set up a jitter to start root cert rotator. "+
			"Jitter selects a backoff time in seconds to start root cert rotator, "+
			"and the back off time is below root cert check interval.")

	caRSAKeySize = env.RegisterIntVar("CITADEL_SELF_SIGNED_CA_RSA_KEY_SIZE", 2048,
		"Specify the RSA key size to use for self-signed Istio CA certificates.")

	// TODO: Likely to be removed and added to mesh config
	externalCaType = env.RegisterStringVar("EXTERNAL_CA", "",
		"External CA Integration Type. Permitted Values are ISTIOD_RA_KUBERNETES_API or "+
			"ISTIOD_RA_ISTIO_API").Get()
)

type caOptions struct {
	// Either extCAK8s or extCAGrpc
	ExternalCAType   ra.CaExternalType
	ExternalCASigner string
	// domain to use in SPIFFE identity URLs
	TrustDomain      string
	Namespace        string
	Authenticators   []security.Authenticator
	CertSignerDomain string
}

func getIstioCA(client corev1.CoreV1Interface, namespace string) (*util.KeyCertBundle, error) {
	opts := &caOptions{
		TrustDomain:      "aeraki.net",
		Namespace:        namespace,
		ExternalCAType:   ra.CaExternalType(externalCaType),
		CertSignerDomain: "aeraki.net",
	}

	var caOpts *ca.IstioCAOptions
	var err error

	fileBundle, err := detectSigningCABundle()
	if err != nil {
		return nil, fmt.Errorf("unable to determine signing file format %v", err)
	}
	if _, err := os.Stat(fileBundle.RootCertFile); err != nil {
		// In Citadel, normal self-signed doesn't use a root-cert.pem file for additional roots.
		// In Istiod, it is possible to provide one via "cacerts" secret in both cases, for consistency.
		fileBundle.RootCertFile = ""
	}
	if _, err := os.Stat(fileBundle.SigningKeyFile); err != nil {
		// The user-provided certs are missing - create a self-signed cert.
		if client != nil {
			log.Info("Use self-signed certificate as the CA certificate")

			// Abort after 20 minutes.
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute*20)
			defer cancel()
			// rootCertFile will be added to "ca-cert.pem".
			// readSigningCertOnly set to false - it doesn't seem to be used in Citadel, nor do we have a way
			// to set it only for one job.
			caOpts, err = ca.NewSelfSignedIstioCAOptions(ctx,
				selfSignedRootCertGracePeriodPercentile.Get(), selfSignedCACertTTL.Get(),
				selfSignedRootCertCheckInterval.Get(), workloadCertTTL.Get(),
				maxWorkloadCertTTL.Get(), opts.TrustDomain, true,
				opts.Namespace, -1, client, fileBundle.RootCertFile,
				enableJitterForRootCertRotator.Get(), caRSAKeySize.Get())
		} else {
			log.Warnf(
				"Use local self-signed CA certificate for testing. Will use in-memory root CA, no K8S access and no ca key file %s",
				fileBundle.SigningKeyFile)

			caOpts, err = ca.NewSelfSignedDebugIstioCAOptions(fileBundle.RootCertFile, selfSignedCACertTTL.Get(),
				workloadCertTTL.Get(), maxWorkloadCertTTL.Get(), opts.TrustDomain, caRSAKeySize.Get())
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create a self-signed istiod CA: %v", err)
		}
	} else {
		log.Info("Use local CA certificate")

		caOpts, err = ca.NewPluggedCertIstioCAOptions(fileBundle, workloadCertTTL.Get(), maxWorkloadCertTTL.Get(), caRSAKeySize.Get())
		if err != nil {
			return nil, fmt.Errorf("failed to create an istiod CA: %v", err)
		}

		//if features.AutoReloadPluginCerts {
		//	s.initCACertsWatcher()
		//}
	}
	istioCA, err := ca.NewIstioCA(caOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create an istiod CA: %v", err)
	}

	// TODO: provide an endpoint returning all the roots. SDS can only pull a single root in current impl.
	// ca.go saves or uses the secret, but also writes to the configmap "istio-security", under caTLSRootCert
	// rootCertRotatorChan channel accepts signals to stop root cert rotator for
	// self-signed CA.
	// Start root cert rotator in a separate goroutine.
	// istioCA.Run(s.internalStop)
	return istioCA.GetCAKeyCertBundle(), nil
}

// detectSigningCABundle determines in which format the signing ca files are created.
// kubernetes tls secrets mount files as tls.crt,tls.key,ca.crt
// istiod secret is ca-cert.pem ca-key.pem cert-chain.pem root-cert.pem
func detectSigningCABundle() (ca.SigningCAFileBundle, error) {
	tlsSigningFile := path.Join(localCertDir.Get(), ca.TLSSecretCACertFile)

	// looking for tls file format (tls.crt)
	if _, err := os.Stat(tlsSigningFile); !os.IsNotExist(err) {
		log.Info("Using kubernetes.io/tls secret type for signing ca files")
		return ca.SigningCAFileBundle{
			RootCertFile: path.Join(localCertDir.Get(), ca.TLSSecretRootCertFile),
			CertChainFiles: []string{
				tlsSigningFile,
				path.Join(localCertDir.Get(), ca.TLSSecretRootCertFile),
			},
			SigningCertFile: tlsSigningFile,
			SigningKeyFile:  path.Join(localCertDir.Get(), ca.TLSSecretCAPrivateKeyFile),
		}, nil
	} else if os.IsNotExist(err) {
		// noop, file does not exist, move on
	} else if err != nil {
		return ca.SigningCAFileBundle{}, err
	}
	log.Info("Using istiod file format for signing ca files")
	// default ca file format
	return ca.SigningCAFileBundle{
		RootCertFile:    path.Join(localCertDir.Get(), ca.RootCertFile),
		CertChainFiles:  []string{path.Join(localCertDir.Get(), ca.CertChainFile)},
		SigningCertFile: path.Join(localCertDir.Get(), ca.CACertFile),
		SigningKeyFile:  path.Join(localCertDir.Get(), ca.CAPrivateKeyFile),
	}, nil
}
