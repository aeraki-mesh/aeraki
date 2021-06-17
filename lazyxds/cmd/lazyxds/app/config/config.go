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

package config

import (
	"github.com/aeraki-framework/aeraki/lazyxds/cmd/lazyxds/app/options"
	componentbaseconfig "k8s.io/component-base/config"
	"os"
	"strings"
)

const (
	// AutoCreateEgress ...
	AutoCreateEgress = "LAZYXDS_AUTO_CREATE_EGRESS"
)

// New creates a new Config from Options
func New(options *options.Options) (*Config, error) {
	config := &Config{
		LeaderElection: options.LeaderElection,
	}

	config.KubeConfig = options.KubeConfig
	config.AutoCreateEgress = strings.ToLower(os.Getenv(AutoCreateEgress)) == "true"
	config.IstiodAddress = options.IstiodAddress
	config.ProxyImage = options.ProxyImage

	return config, nil
}

// Config is the lazyxds manager configuration
type Config struct {
	KubeConfig       string
	AutoCreateEgress bool
	IstiodAddress    string
	ProxyImage       string
	LeaderElection   *componentbaseconfig.LeaderElectionConfiguration
}
