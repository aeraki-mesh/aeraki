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

package metaprotocol

import (
	userapi "github.com/aeraki-framework/aeraki/api/metaprotocol/v1alpha1"
	mpclient "github.com/aeraki-framework/aeraki/client-go/pkg/apis/metaprotocol/v1alpha1"
	"github.com/aeraki-framework/aeraki/pkg/xds"
	lrldataplane "github.com/aeraki-framework/meta-protocol-control-plane-api/meta_protocol_proxy/filters/local_ratelimit/v1alpha"
	mpdataplane "github.com/aeraki-framework/meta-protocol-control-plane-api/meta_protocol_proxy/v1alpha"
	ratelimit "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	commondataplane "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/protobuf/types/known/anypb"
)

func buildOutboundFilters(metaRouter *mpclient.MetaRouter) []*mpdataplane.MetaProtocolFilter {
	var filters []*mpdataplane.MetaProtocolFilter
	filters = appendRouter(filters)
	return filters
}

func buildInboundFilters(metaRouter *mpclient.MetaRouter) []*mpdataplane.MetaProtocolFilter {
	var filters []*mpdataplane.MetaProtocolFilter
	if metaRouter != nil {
		filters = appendLocalRateLimitFilter(metaRouter, filters)
		filters = appendGlobalRateLimitFilter(metaRouter, filters)
	}
	filters = appendRouter(filters)
	return filters
}

func appendRouter(filters []*mpdataplane.MetaProtocolFilter) []*mpdataplane.MetaProtocolFilter {
	router := mpdataplane.MetaProtocolFilter{
		Name: "aeraki.meta_protocol.filters.router",
	}
	return append(filters, &router)
}

func appendLocalRateLimitFilter(metaRouter *mpclient.MetaRouter,
	filters []*mpdataplane.MetaProtocolFilter) []*mpdataplane.MetaProtocolFilter {
	if metaRouter.Spec.LocalRateLimit == nil {
		return filters
	}

	localRateLimit := metaRouter.Spec.LocalRateLimit
	lrt := &lrldataplane.LocalRateLimit{
		StatPrefix: metaRouter.Spec.Hosts[0],
		Match: &lrldataplane.LocalRatelimitMatch{
			Metadata: xds.MetaMatch2HttpHeaderMatch(localRateLimit.Match),
		},
		TokenBucket: crd2kenBucket(localRateLimit.TokenBucket),
		Descriptors: crd2Descriptors(localRateLimit.Descriptors),
	}

	config, err := anypb.New(lrt)
	if err != nil {
		generatorLog.Errorf("local ratelimit create failed: %e", err)
	}

	localRateLimitFilter := mpdataplane.MetaProtocolFilter{
		Name:   "aeraki.meta_protocol.filters.local_ratelimit",
		Config: config,
	}
	return append(filters, &localRateLimitFilter)
}

func appendGlobalRateLimitFilter(metaRouter *mpclient.MetaRouter,
	filters []*mpdataplane.MetaProtocolFilter) []*mpdataplane.MetaProtocolFilter {
	return filters
}

func crd2Descriptors(descriptorCrds []*userapi.RateLimitDescriptor) []*ratelimit.LocalRateLimitDescriptor {
	var localDescriptors []*ratelimit.LocalRateLimitDescriptor
	for _, descriptor := range descriptorCrds {
		tokenBucket := crd2kenBucket(descriptor.TokenBucket)

		var entries []*ratelimit.RateLimitDescriptor_Entry
		for _, entry := range descriptor.Entries {
			entries = append(entries,
				&ratelimit.RateLimitDescriptor_Entry{
					Key:   entry.Key,
					Value: entry.Value,
				})
		}
		localDescriptors = append(localDescriptors, &ratelimit.LocalRateLimitDescriptor{
			Entries:     entries,
			TokenBucket: tokenBucket,
		})
	}
	return localDescriptors
}

func crd2kenBucket(tbCrd *userapi.TokenBucket) *commondataplane.TokenBucket {
	tokenBucket := &commondataplane.TokenBucket{
		MaxTokens: tbCrd.MaxTokens,
		FillInterval: &duration.Duration{
			Seconds: tbCrd.FillInterval.Seconds,
			Nanos:   tbCrd.FillInterval.Nanos,
		},
	}

	tokensPerFill := tbCrd.TokensPerFill.Value
	if tokensPerFill != 0 {
		tokenBucket.TokensPerFill = &wrappers.UInt32Value{
			Value: tokensPerFill,
		}
	}
	return tokenBucket
}
