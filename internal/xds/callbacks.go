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

package xds

import (
	"context"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
)

type callbacks struct {
	cacheMgr cacheMgr
}

func newCallbacks(cacheMgr cacheMgr) serverv3.Callbacks {
	return &callbacks{
		cacheMgr: cacheMgr,
	}
}

func (cb *callbacks) OnStreamRequest(_ int64, request *discovery.DiscoveryRequest) error {
	xdsLog.Infof("receive rds request from: %s", request.Node.Id)
	if !cb.cacheMgr.hasNode(request.Node.Id) {
		xdsLog.Infof("init rds cache for node: %s", request.Node.Id)
		cb.cacheMgr.initNode(request.Node.Id)
	}
	return nil
}

func (cb *callbacks) OnStreamOpen(_ context.Context, id int64, typ string) error {
	xdsLog.Infof("stream %d open for %s\n", id, typ)
	return nil
}
func (cb *callbacks) OnStreamClosed(id int64, node *core.Node) {
	xdsLog.Infof("node %s stream %d closed\n", node.Id, id)
	cb.cacheMgr.clearNode(node.Id)
}

func (cb *callbacks) OnDeltaStreamOpen(_ context.Context, id int64, typ string) error {
	xdsLog.Infof("delta stream %d open for %s\n", id, typ)
	return nil
}
func (cb *callbacks) OnDeltaStreamClosed(id int64, node *core.Node) {
	xdsLog.Infof("node %s delta stream %d closed\n", node.Id, id)
}

func (cb *callbacks) OnStreamResponse(_ context.Context, _ int64, request *discovery.DiscoveryRequest,
	response *discovery.DiscoveryResponse) {
	xdsLog.Debugf("send rds response to: %s :%v", request.Node.Id, response.Resources)
}
func (cb *callbacks) OnStreamDeltaResponse(_ int64, _ *discovery.DeltaDiscoveryRequest,
	_ *discovery.DeltaDiscoveryResponse) {
}
func (cb *callbacks) OnStreamDeltaRequest(_ int64, _ *discovery.DeltaDiscoveryRequest) error {
	return nil
}
func (cb *callbacks) OnFetchRequest(_ context.Context, _ *discovery.DiscoveryRequest) error {
	return nil
}
func (cb *callbacks) OnFetchResponse(*discovery.DiscoveryRequest, *discovery.DiscoveryResponse) {}
