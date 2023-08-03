// Copyright Istio Authors
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

package leaderelection

import (
	"context"
	"time"

	"go.uber.org/atomic"
	"istio.io/istio/pilot/pkg/leaderelection/k8sleaderelection"
	"istio.io/istio/pilot/pkg/leaderelection/k8sleaderelection/k8sresourcelock"
	"istio.io/pkg/log"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Various locks used throughout the code
const (
	EnvoyFilterController = "aeraki-envoyfilter-controller-election"
	AllocateVIPController = "aeraki-service-entry-vip-allocation-controller-election"
)

// LeaderElection is used for controller leader election
type LeaderElection struct {
	namespace string
	name      string
	runFns    []func(stop <-chan struct{})
	client    kubernetes.Interface
	ttl       time.Duration

	// Records which "cycle" the election is on. This is incremented each time an election is won and then lost
	// This is mostly just for testing
	cycle      *atomic.Int32
	electionID string
}

// Run will start leader election, calling all runFns when we become the leader.
func (l *LeaderElection) Run(stop <-chan struct{}) {
	for {
		le, err := l.create()
		if err != nil {
			// This should never happen; errors are only from invalid input and the input is not user modifiable
			panic("LeaderElection creation failed: " + err.Error())
		}
		l.cycle.Inc()
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-stop
			cancel()
		}()
		le.Run(ctx)
		select {
		case <-stop:
			// We were told to stop explicitly. Exit now
			return
		default:
			cancel()
			// Otherwise, we may have lost our lock. In practice, this is extremely rare; we need to have the lock, then lose it
			// Typically this means something went wrong, such as API server downtime, etc
			// If this does happen, we will start the cycle over again
			log.Errorf("Leader election cycle %v lost. Trying again", l.cycle.Load())
		}
	}
}

func (l *LeaderElection) create() (*k8sleaderelection.LeaderElector, error) {
	callbacks := k8sleaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			log.Infof("leader election lock acquired: %v", l.electionID)
			for _, f := range l.runFns {
				go f(ctx.Done())
			}
		},
		OnStoppedLeading: func() {
			log.Infof("leader election lock lost: %v", l.electionID)
		},
	}
	lock := k8sresourcelock.ConfigMapLock{
		ConfigMapMeta: metaV1.ObjectMeta{Namespace: l.namespace, Name: l.electionID},
		Client:        l.client.CoreV1(),
		LockConfig: k8sresourcelock.ResourceLockConfig{
			Identity: l.name,
		},
	}
	return k8sleaderelection.NewLeaderElector(k8sleaderelection.LeaderElectionConfig{
		Lock:          &lock,
		LeaseDuration: l.ttl,
		RenewDeadline: l.ttl / 2,
		RetryPeriod:   l.ttl / 4,
		Callbacks:     callbacks,
		// When Pilot exits, the lease will be dropped. This is more likely to lead to a case where
		// to instances are both considered the leaders. As such, if this is intended to be use for mission-critical
		// usages (rather than avoiding duplication of work), this may need to be re-evaluated.
		ReleaseOnCancel: true,
	})
}

// AddRunFunction registers a function to run when we are the leader. These will be run asynchronously.
// To avoid running when not a leader, functions should respect the stop channel.
func (l *LeaderElection) AddRunFunction(f func(stop <-chan struct{})) *LeaderElection {
	l.runFns = append(l.runFns, f)
	return l
}

// NewLeaderElection creates a new LeaderElection
func NewLeaderElection(namespace, name, electionID string, client kubernetes.Interface) *LeaderElection {
	if name == "" {
		name = "unknown"
	}
	return &LeaderElection{
		namespace:  namespace,
		name:       name,
		electionID: electionID,
		client:     client,
		// Default to a 30s ttl. Overridable for tests
		ttl:   time.Second * 30,
		cycle: atomic.NewInt32(0),
	}
}
