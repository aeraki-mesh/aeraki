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
	"fmt"

	metaroute "github.com/aeraki-mesh/meta-protocol-control-plane-api/meta_protocol_proxy/config/route/v1alpha"
	grldataplane "github.com/aeraki-mesh/meta-protocol-control-plane-api/meta_protocol_proxy/filters/global_ratelimit/v1alpha"
	lrldataplane "github.com/aeraki-mesh/meta-protocol-control-plane-api/meta_protocol_proxy/filters/local_ratelimit/v1alpha"
	mpdataplane "github.com/aeraki-mesh/meta-protocol-control-plane-api/meta_protocol_proxy/v1alpha"
	envoycore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyrl "github.com/envoyproxy/go-control-plane/envoy/config/ratelimit/v3"
	commondataplane "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/protobuf/types/known/anypb"

	userapi "github.com/aeraki-mesh/aeraki/api/metaprotocol/v1alpha1"
	mpclient "github.com/aeraki-mesh/aeraki/client-go/pkg/apis/metaprotocol/v1alpha1"
	"github.com/aeraki-mesh/aeraki/pkg/xds"
)

func buildOutboundFilters(metaRouter *mpclient.MetaRouter) []*mpdataplane.MetaProtocolFilter {
	var filters []*mpdataplane.MetaProtocolFilter
	filters = appendRouter(filters)
	return filters
}

func buildInboundFilters(metaRouter *mpclient.MetaRouter) ([]*mpdataplane.MetaProtocolFilter, error) {
	var filters []*mpdataplane.MetaProtocolFilter
	var err error
	if metaRouter != nil {
		if metaRouter.Spec.LocalRateLimit != nil {
			if filters, err = appendLocalRateLimitFilter(metaRouter, filters); err != nil {
				return filters, err
			}
		}
		if metaRouter.Spec.GlobalRateLimit != nil {
			if filters, err = appendGlobalRateLimitFilter(metaRouter, filters); err != nil {
				return filters, err
			}
		}
	}

	return appendRouter(filters), nil
}

func appendRouter(filters []*mpdataplane.MetaProtocolFilter) []*mpdataplane.MetaProtocolFilter {
	router := mpdataplane.MetaProtocolFilter{
		Name: "aeraki.meta_protocol.filters.router",
	}
	return append(filters, &router)
}

func appendLocalRateLimitFilter(metaRouter *mpclient.MetaRouter,
	filters []*mpdataplane.MetaProtocolFilter) ([]*mpdataplane.MetaProtocolFilter, error) {
	localRateLimit := metaRouter.Spec.LocalRateLimit

	if localRateLimit.TokenBucket == nil && len(localRateLimit.Conditions) == 0 {
		return nil, fmt.Errorf("either tokenBucket or conditions should be specified")
	}
	lrt := &lrldataplane.LocalRateLimit{
		StatPrefix: metaRouter.Spec.Hosts[0],
	}
	if localRateLimit.TokenBucket != nil {
		lrt.TokenBucket = crd2tokenBucket(localRateLimit.TokenBucket)
	}
	if len(localRateLimit.Conditions) > 0 {
		lrt.Conditions = crd2Conditions(localRateLimit.Conditions)
	}

	config, err := anypb.New(lrt)
	if err != nil {
		generatorLog.Errorf("local ratelimit create failed: %e", err)
	}

	localRateLimitFilter := mpdataplane.MetaProtocolFilter{
		Name:   "aeraki.meta_protocol.filters.local_ratelimit",
		Config: config,
	}
	filters = append(filters, &localRateLimitFilter)
	return filters, nil
}

func appendGlobalRateLimitFilter(metaRouter *mpclient.MetaRouter,
	filters []*mpdataplane.MetaProtocolFilter) ([]*mpdataplane.MetaProtocolFilter, error) {
	globalRateLimit := metaRouter.Spec.GlobalRateLimit

	if len(globalRateLimit.Descriptors) == 0 {
		return nil, fmt.Errorf("then length of global rate [lmit actions should not be zero")
	}

	var descriptors []*grldataplane.Descriptor
	for _, action := range globalRateLimit.Descriptors {
		descriptors = append(descriptors, &grldataplane.Descriptor{
			Property:      action.Property,
			DescriptorKey: action.DescriptorKey,
		})
	}

	grt := &grldataplane.RateLimit{
		Match: &metaroute.RouteMatch{
			Metadata: xds.MetaMatch2HttpHeaderMatch(globalRateLimit.Match),
		},
		Domain: globalRateLimit.Domain,
		Timeout: &duration.Duration{
			Seconds: globalRateLimit.RequestTimeout.Seconds,
			Nanos:   globalRateLimit.RequestTimeout.Nanos,
		},
		FailureModeDeny: globalRateLimit.DenyOnFail,
		RateLimitService: &envoyrl.RateLimitServiceConfig{
			GrpcService: &envoycore.GrpcService{
				TargetSpecifier: &envoycore.GrpcService_EnvoyGrpc_{
					EnvoyGrpc: &envoycore.GrpcService_EnvoyGrpc{
						ClusterName: globalRateLimit.RateLimitService,
					},
				},
			},
		},
		Descriptors: descriptors,
	}
	config, err := anypb.New(grt)
	if err != nil {
		generatorLog.Errorf("global ratelimit create failed: %e", err)
	}

	globalRateLimitFilter := mpdataplane.MetaProtocolFilter{
		Name:   "aeraki.meta_protocol.filters.ratelimit",
		Config: config,
	}
	filters = append(filters, &globalRateLimitFilter)
	return filters, nil
}

func crd2Conditions(conditions []*userapi.LocalRateLimit_Condition) []*lrldataplane.LocalRateLimitCondition {
	var localConditions []*lrldataplane.LocalRateLimitCondition
	for _, condition := range conditions {
		if condition.TokenBucket != nil {
			tokenBucket := crd2tokenBucket(condition.TokenBucket)
			localConditions = append(localConditions, &lrldataplane.LocalRateLimitCondition{
				TokenBucket: tokenBucket,
				Match: &metaroute.RouteMatch{
					Metadata: xds.MetaMatch2HttpHeaderMatch(condition.Match),
				},
			})
		}
	}
	return localConditions
}

func crd2tokenBucket(tbCrd *userapi.LocalRateLimit_TokenBucket) *commondataplane.TokenBucket {
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
