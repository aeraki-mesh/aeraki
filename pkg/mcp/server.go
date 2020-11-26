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

package mcp

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aeraki-framework/aeraki/pkg/model/protocol"

	"istio.io/istio/pkg/config/schema/collections"

	"github.com/aeraki-framework/aeraki/pkg/envoyfilter"
	"github.com/gogo/protobuf/types"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"istio.io/api/mcp/v1alpha1"
	mcp "istio.io/api/mcp/v1alpha1"
	networking "istio.io/api/networking/v1alpha3"
	istiomodel "istio.io/istio/pilot/pkg/model"
	"istio.io/pkg/log"
)

const (
	// debounceAfter is the delay added to events to wait after a registry event for debouncing.
	// This will delay the push by at least this interval, plus the time getting subsequent events.
	// If no change is detected the push will happen, otherwise we'll keep delaying until things settle.
	debounceAfter = 100 * time.Millisecond

	// debounceMax is the maximum time to wait for events while debouncing.
	// Defaults to 10 seconds. If events keep showing up with no break for this time, we'll trigger a push.
	debounceMax = 10 * time.Second

	// configRootNS is the root config root namespace
	configRootNS = "istio-system"
)

var (
	mcpLog = log.RegisterScope("mcp-server", "mcp debugging", 0)

	// Tracks connections, increment on each new connection.
	connectionNumber = int64(0)
)

type EnvoyFilterWrapper struct {
	service     *networking.ServiceEntry
	envoyfilter *networking.EnvoyFilter
}

// Connection holds information about connected client.
type Connection struct {
	sync.RWMutex

	// ConID is the connection identifier, used as a key in the connection table.
	// Currently based on the sink id, peer addr and a counter.
	ConID string

	// PeerAddr is the address of the client, from network layer.
	PeerAddr string

	// Time of connection, for debugging
	Connect time.Time

	// Sending on this channel results in a push.
	pushChannel chan istiomodel.Event

	// MCP stream implement this interface
	stream mcp.ResourceSource_EstablishResourceStreamServer

	// LastResponse stores the last response nonce to each sink
	LastResponse map[string]string
}

type Server struct {
	listeningAddress string
	grpcServer       *grpc.Server
	mcpClients       map[string]*Connection
	mcpClientsMutex  sync.RWMutex
	configStore      istiomodel.ConfigStore
	generator        envoyfilter.Generator
	instance         protocol.Instance
}

func NewServer(listeningAddress string, store istiomodel.ConfigStore, generator envoyfilter.Generator, instance protocol.Instance) *Server {
	mcpServer := &Server{
		listeningAddress: listeningAddress,
		mcpClients:       make(map[string]*Connection),
		configStore:      store,
		generator:        generator,
		instance:         instance,
	}
	return mcpServer
}

// Start the gRPC MCP server
func (s *Server) Start() error {
	if err := s.startGrpcServer(); err != nil {
		mcpLog.Fatala(err)
		return err
	}
	return nil
}

func (s *Server) Stop() {
	s.grpcServer.Stop()
}

func (s *Server) startGrpcServer() error {
	grpcOptions := s.grpcServerOptions()
	s.grpcServer = grpc.NewServer(grpcOptions...)
	mcp.RegisterResourceSourceServer(s.grpcServer, s)

	listener, err := net.Listen("tcp", s.listeningAddress)
	if err != nil {
		return err
	}

	if err := s.grpcServer.Serve(listener); err != nil {
		mcpLog.Fatala(err)
		return err
	}

	return nil
}

func (s *Server) EstablishResourceStream(stream mcp.ResourceSource_EstablishResourceStreamServer) error {
	var timeChan <-chan time.Time
	var startDebounce time.Time
	var lastResourceUpdateTime time.Time

	pushCounter := 0
	debouncedEvents := 0
	con := s.newConnection(stream)
	s.addConnection(con)
	defer s.removeConnection(con)

	go con.receive()

	for {
		select {
		case e := <-con.pushChannel:
			mcpLog.Debugf("Receive event from push chanel : %v", e)
			lastResourceUpdateTime = time.Now()
			if debouncedEvents == 0 {
				mcpLog.Debugf("This is the first debounced event")
				startDebounce = lastResourceUpdateTime
			}
			timeChan = time.After(debounceAfter)
			debouncedEvents++
		case <-timeChan:
			mcpLog.Debugf("Receive event from time chanel")
			eventDelay := time.Since(startDebounce)
			quietTime := time.Since(lastResourceUpdateTime)
			// it has been too long since the first debounced event or quiet enough since the last debounced event
			if eventDelay >= debounceMax || quietTime >= debounceAfter {
				if debouncedEvents > 0 {
					pushCounter++
					mcpLog.Infof("Push debounce stable[%d] %d: %v since last change, %v since last push",
						pushCounter, debouncedEvents,
						quietTime, eventDelay)
					err := s.pushEnvoyFilters(con)
					if err != nil {
						mcpLog.Errorf("Failed to push EnvoyFilters to Istio: %v", err)
					}
					debouncedEvents = 0
				}
			} else {
				timeChan = time.After(debounceAfter - quietTime)
			}
		}
	}
}

func (s *Server) pushEnvoyFilters(con *Connection) error {
	envoyFilters := make([]*EnvoyFilterWrapper, 0)
	configs, err := s.configStore.List(collections.IstioNetworkingV1Alpha3Serviceentries.Resource().GroupVersionKind(), "")
	if err != nil {
		return fmt.Errorf("failed to list configs: %v", err)
	}
	for _, config := range configs {
		service, ok := config.Spec.(*networking.ServiceEntry)
		if !ok { // should never happen
			return fmt.Errorf("failed in getting a virtual service: %s: %v", config.Labels, err)
		}
		for _, port := range service.Ports {
			if protocol.GetLayer7ProtocolFromPortName(port.Name) == s.instance {
				envoyFilters = append(envoyFilters, &EnvoyFilterWrapper{
					service:     service,
					envoyfilter: s.generator.Generate(service),
				})
				break
			}
		}
	}
	resources, err := constructResources(envoyFilters, s.instance)
	if err != nil {
		return fmt.Errorf("failed to construct resources: %v", err)
	}

	response := &mcp.Resources{
		Collection:  collections.IstioNetworkingV1Alpha3Envoyfilters.Name().String(),
		Resources:   resources,
		Nonce:       time.Now().String(),
		Incremental: false,
	}
	if err = con.send(response); err != nil {
		return fmt.Errorf("failed to send response: %v", err)
	}
	mcpLog.Infof("Pushed %v EnvoyFilters to Istio: %v", len(envoyFilters), response)
	return nil
}

func constructResources(envoyFilters []*EnvoyFilterWrapper, instance protocol.Instance) ([]mcp.Resource, error) {
	resources := make([]mcp.Resource, 0)
	for _, wrapper := range envoyFilters {
		seAny, err := types.MarshalAny(wrapper.envoyfilter)
		if err != nil {
			return resources, err
		}
		resources = append(resources, mcp.Resource{
			Body: seAny,
			Metadata: &v1alpha1.Metadata{
				Name:    configRootNS + "/" + wrapper.service.Hosts[0] + "_" + "aeraki" + "_" + instance.String(),
				Version: "v1",
			},
		})
	}
	return resources, nil
}

func (con *Connection) receive() {
	for {
		req, err := con.stream.Recv()
		if err != nil {
			if isExpectedGRPCError(err) {
				mcpLog.Infof("%s terminated %v", con.ConID, err)
			}
			mcpLog.Errorf("%s terminated with error: %v", con.ConID, err)
			return
		}
		if con.shouldResponse(req) {
			// This MCP server only supports EnvoyFilter
			if req.Collection != collections.IstioNetworkingV1Alpha3Envoyfilters.Name().String() {
				response := &mcp.Resources{
					Incremental: false,
					Collection:  req.GetCollection(),
					Nonce:       time.Now().String(),
				}
				if err = con.send(response); err != nil {
					mcpLog.Errorf("failed to send response: %v", err)
				}
			} else {
				// Send a change event to the connection channel to trigger a push to the client
				con.pushChannel <- istiomodel.EventAdd
			}
		}
	}
}

// isExpectedGRPCError checks a gRPC error code and determines whether it is an expected error when
// things are operating normally. This is basically capturing when the client disconnects.
func isExpectedGRPCError(err error) bool {
	if err == io.EOF {
		return true
	}

	s := status.Convert(err)
	if s.Code() == codes.Canceled || s.Code() == codes.DeadlineExceeded {
		return true
	}
	if s.Code() == codes.Unavailable && s.Message() == "client disconnected" {
		return true
	}
	return false
}

func (con *Connection) send(response *mcp.Resources) error {
	con.Lock()
	defer con.Unlock()
	err := con.stream.Send(response)
	con.LastResponse[response.Collection] = response.Nonce
	return err
}

func (con *Connection) shouldResponse(req *mcp.RequestResources) bool {
	// This is the first request, we should response.
	if req.ResponseNonce == "" {
		mcpLog.Debugf("RESOURCE:%s: REQ %s initial request", req.Collection, con.ConID)
		return true
	}

	// The presence of ErrorDetail means that this is a NACK and the previous response is invalid. We can't resent the
	// same resources. It's perhaps caused by an error in the source code, we should check the error log and fix the code
	// in that case.
	if req.ErrorDetail != nil {
		errCode := codes.Code(req.ErrorDetail.Code)
		mcpLog.Errorf("RESOURCE:%s: ACK ERROR %s %s:%s", req.Collection, con.ConID, errCode.String(), req.ErrorDetail.GetMessage())
		return false
	}

	previousRespone, ok := con.LastResponse[req.Collection]
	// MCP Server does not have information about this collection, but MCP Sink client sends response nonce - either
	// because MCP Server is restarted or MCP Sink client disconnects and reconnects.
	// We should always respond with the current resource.
	if !ok {
		mcpLog.Warnf("RESOURCE:%s: RECONNECT %s %s", req.Collection, con.ConID, req.ResponseNonce)
		return true
	}

	// If there is mismatch in the nonce, that is a case of expired/stale nonce.
	// A nonce becomes stale following a newer nonce being sent to Envoy.
	if req.ResponseNonce != previousRespone {
		mcpLog.Warnf("RESOURCE:%s: REQ %s Expired nonce received %s, sent %s", req.Collection,
			con.ConID, req.ResponseNonce, previousRespone)
		return false
	}

	// If it comes here, that means nonce match. This an ACK. we should not response unless there is a change in MCP Server side.
	mcpLog.Debugf("RESOURCE:%s: ACK %s %s", req.Collection, con.ConID, req.ResponseNonce)
	return false
}

func (s *Server) grpcServerOptions() []grpc.ServerOption {
	interceptors := []grpc.UnaryServerInterceptor{
		// setup server prometheus monitoring (as final interceptor in chain)
		prometheus.UnaryServerInterceptor,
	}

	grpcOptions := []grpc.ServerOption{
		grpc.UnaryInterceptor(middleware.ChainUnaryServer(interceptors...)),
	}

	return grpcOptions
}

func (s *Server) newConnection(stream mcp.ResourceSource_EstablishResourceStreamServer) *Connection {
	ctx := stream.Context()
	peerAddr := "0.0.0.0"
	if peerInfo, ok := peer.FromContext(ctx); ok {
		peerAddr = peerInfo.Addr.String()
	}
	id := atomic.AddInt64(&connectionNumber, 1)
	conId := peerAddr + "-" + strconv.FormatInt(id, 10)
	return &Connection{
		PeerAddr:     peerAddr,
		Connect:      time.Now(),
		ConID:        conId,
		pushChannel:  make(chan istiomodel.Event, 100),
		stream:       stream,
		LastResponse: make(map[string]string),
	}
}

func (s *Server) addConnection(con *Connection) {
	s.mcpClientsMutex.Lock()
	defer s.mcpClientsMutex.Unlock()
	s.mcpClients[con.ConID] = con

	mcpLog.Infof("Receive connection from client: %s", con.ConID)
}

func (s *Server) removeConnection(con *Connection) {
	s.mcpClientsMutex.Lock()
	defer s.mcpClientsMutex.Unlock()
	delete(s.mcpClients, con.ConID)

	mcpLog.Infof("Remove connection from client: %s", con.ConID)
}

func (s *Server) ConfigUpdate(event istiomodel.Event) {
	s.mcpClientsMutex.Lock()
	defer s.mcpClientsMutex.Unlock()

	for _, con := range s.mcpClients {
		con.pushChannel <- event
	}
}
