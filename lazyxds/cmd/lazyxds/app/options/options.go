/*
 * // Copyright Aeraki Authors
 * //
 * // Licensed under the Apache License, Version 2.0 (the "License");
 * // you may not use this file except in compliance with the License.
 * // You may obtain a copy of the License at
 * //
 * //     http://www.apache.org/licenses/LICENSE-2.0
 * //
 * // Unless required by applicable law or agreed to in writing, software
 * // distributed under the License is distributed on an "AS IS" BASIS,
 * // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * // See the License for the specific language governing permissions and
 * // limitations under the License.
 */

package options

import (
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils/leaderelectionconfig"
	"github.com/spf13/pflag"
	"k8s.io/component-base/config"
)

const (
	// DefaultIstiodAddress is the default istiod address
	DefaultIstiodAddress = "istiod.istio-system.svc:15012"
	// DefaultProxyImage is the default sidecar image of istio
	DefaultProxyImage = "docker.io/istio/proxyv2:1.10.0"
)

// Options for lazyxds
type Options struct {
	KubeConfig     string
	IstiodAddress  string
	ProxyImage     string
	LeaderElection *config.LeaderElectionConfiguration
}

// New creates an Options
func New(basename string) *Options {
	return &Options{
		LeaderElection: leaderelectionconfig.New(basename),
	}
}

// AddFlags add several flags of lazyxds
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.KubeConfig, "kube-config", "",
		"service discovery kube config file")
	fs.StringVar(&o.IstiodAddress, "istiod-address", DefaultIstiodAddress,
		"istiod address, use to create lazyxds egress")
	fs.StringVar(&o.ProxyImage, "proxy-image", DefaultProxyImage,
		"proxy image, use to create lazyxds egress")
	leaderelectionconfig.AddFlags(o.LeaderElection, fs)
}

// Validate will check the requirements of options
func (o *Options) Validate() []error {
	return nil
}
