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
	"context"
	"github.com/aeraki-framework/aeraki/lazyxds/cmd/lazyxds/app/config"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/manager"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
	"os"
	"time"
)

// Run start leader election and run main process
func Run(conf *config.Config, stopCh <-chan struct{}) error {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-stopCh
		cancel()
	}()

	if !conf.LeaderElection.LeaderElect {
		doRun(ctx, conf)
		return nil
	}

	client, err := utils.NewKubeClient(conf.KubeConfig)
	if err != nil {
		klog.Fatalf("build kube client failed: %v", err)
	}

	id, err := os.Hostname()
	if err != nil {
		return err
	}

	rl, err := resourcelock.New(conf.LeaderElection.ResourceLock,
		conf.LeaderElection.ResourceNamespace,
		conf.LeaderElection.ResourceName,
		client.CoreV1(),
		client.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity: id,
		})
	if err != nil {
		klog.Fatalf("error creating lock: %v", err)
	}

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            rl,
		ReleaseOnCancel: true,
		LeaseDuration:   conf.LeaderElection.LeaseDuration.Duration,
		RenewDeadline:   conf.LeaderElection.RenewDeadline.Duration,
		RetryPeriod:     conf.LeaderElection.RetryPeriod.Duration,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(c context.Context) {
				klog.Infof("%s: leading", id)
				doRun(c, conf)
			},
			OnStoppedLeading: func() {
				klog.Infof("%s: stop leading", id)
				time.Sleep(3 * time.Second)
				klog.Infof("%s: stopped leading", id)
			},
			OnNewLeader: func(identity string) {
				if identity == id {
					return
				}
				klog.Infof("new leader elected: %v", identity)
			},
		},
	})

	klog.Info("app exiting")
	return nil
}

func doRun(ctx context.Context, conf *config.Config) {
	var err error
	var m manager.LazyXdsManager
	m, err = manager.NewManager(conf, ctx.Done())
	if err != nil {
		klog.Fatalf("new lazyxds manager failed: %v", err)
	}
	if err = m.Run(); err != nil {
		klog.Fatalf("run lazyxds manager failed: %v", err)
	}

	<-ctx.Done()
}
