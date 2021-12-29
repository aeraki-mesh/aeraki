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

	userapi "github.com/aeraki-framework/aeraki/api/metaprotocol/v1alpha1"
	mpclient "github.com/aeraki-framework/aeraki/client-go/pkg/apis/metaprotocol/v1alpha1"
	"github.com/aeraki-framework/aeraki/pkg/xds"
	metaroute "github.com/aeraki-framework/meta-protocol-control-plane-api/meta_protocol_proxy/config/route/v1alpha"
	lrldataplane "github.com/aeraki-framework/meta-protocol-control-plane-api/meta_protocol_proxy/filters/local_ratelimit/v1alpha"
	mpdataplane "github.com/aeraki-framework/meta-protocol-control-plane-api/meta_protocol_proxy/v1alpha"
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

func buildInboundFilters(metaRouter *mpclient.MetaRouter) ([]*mpdataplane.MetaProtocolFilter, error) {
	var filters []*mpdataplane.MetaProtocolFilter
	var err error
	if metaRouter != nil {
		if metaRouter.Spec.LocalRateLimit != nil {
			if filters, err = appendLocalRateLimitFilter(metaRouter, filters); err != nil {
				return filters, err
			}
		}
		filters = appendGlobalRateLimitFilter(metaRouter, filters)
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
		lrt.TokenBucket = crd2kenBucket(localRateLimit.TokenBucket)
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
	filters []*mpdataplane.MetaProtocolFilter) []*mpdataplane.MetaProtocolFilter {
	return filters
}

func crd2Conditions(conditions []*userapi.LocalRateLimitCondition) []*lrldataplane.LocalRateLimitCondition {
	var localConditions []*lrldataplane.LocalRateLimitCondition
	for _, condition := range conditions {
		tokenBucket := crd2kenBucket(condition.TokenBucket)
		localConditions = append(localConditions, &lrldataplane.LocalRateLimitCondition{
			TokenBucket: tokenBucket,
			Match: &metaroute.RouteMatch{
				Metadata: xds.MetaMatch2HttpHeaderMatch(condition.Match),
			},
		})
	}
	return localConditions
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
