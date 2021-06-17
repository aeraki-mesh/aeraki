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

package app

import (
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils/app"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils/signal"
	"k8s.io/klog/v2"

	"github.com/aeraki-framework/aeraki/lazyxds/cmd/lazyxds/app/config"
	"github.com/aeraki-framework/aeraki/lazyxds/cmd/lazyxds/app/options"
)

const commandDesc = `lazyxds enables istio only push needed xds to sidecars`

// New creates a App object with default parameters.
func New(basename string) *app.App {
	opts := options.New(basename)
	application := app.NewApp("lazyxds",
		basename,
		app.WithOptions(opts),
		app.WithDescription(commandDesc),
		app.WithRunFunc(run(opts)),
	)
	return application
}

func run(opts *options.Options) app.RunFunc {
	return func(basename string) error {
		defer klog.Flush()

		cfg, err := config.New(opts)
		if err != nil {
			return err
		}

		stopCh := signal.SetupSignalHandler()
		return Run(cfg, stopCh)
	}
}
