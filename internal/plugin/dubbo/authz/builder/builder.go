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

package builder

import (
	"context"
	"fmt"

	dubborulepb "github.com/aeraki-mesh/api/dubbo/v1alpha1"
	dubboapi "github.com/aeraki-mesh/client-go/pkg/apis/dubbo/v1alpha1"
	dubboclient "github.com/aeraki-mesh/client-go/pkg/clientset/versioned/typed/dubbo/v1alpha1"
	rbacpb "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	dubbopb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/dubbo_proxy/v3"
	rbacdubbopb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/rbac/v3"
	"google.golang.org/protobuf/types/known/anypb"
	"istio.io/istio/pilot/pkg/security/trustdomain"
	"istio.io/pkg/log"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	authzmodel "github.com/aeraki-mesh/aeraki/internal/plugin/dubbo/authz/model"
)

var (
	authzLog = log.RegisterScope("authorization", "Aeraki Dubbo Authorization Policy", 0)
)

// Builder builds Istio authorization policy to Envoy RBAC filter.
type Builder struct {
	trustDomainBundle trustdomain.Bundle
	denyPolicies      []*dubboapi.DubboAuthorizationPolicy
	allowPolicies     []*dubboapi.DubboAuthorizationPolicy
}

// New returns a new builder for the given workload with the authorization policy.
// Returns nil if none of the authorization policies are enabled for the workload.
func New(trustDomainBundle trustdomain.Bundle, namespace string,
	client dubboclient.DubboV1alpha1Interface) *Builder {
	allowPolicies := make([]*dubboapi.DubboAuthorizationPolicy, 0)
	denyPolicies := make([]*dubboapi.DubboAuthorizationPolicy, 0)

	dubboAuthorizationPolicyList, err := client.DubboAuthorizationPolicies(namespace).List(context.TODO(),
		v1.ListOptions{})
	if err != nil {
		authzLog.Errorf("failed to list DubboAuthorizationPolicy: %v", err)
	} else {
		for i := range dubboAuthorizationPolicyList.Items {
			config := dubboAuthorizationPolicyList.Items[i]
			switch config.Spec.GetAction() {
			case dubborulepb.DubboAuthorizationPolicy_ALLOW:
				allowPolicies = append(allowPolicies, config)
			case dubborulepb.DubboAuthorizationPolicy_DENY:
				denyPolicies = append(denyPolicies, config)
			default:
				log.Errorf("ignored authorization policy %s.%s with unsupported action: %s",
					config.Namespace, config.Name, config.Spec.GetAction())
			}
		}
	}

	return &Builder{
		trustDomainBundle: trustDomainBundle,
		denyPolicies:      denyPolicies,
		allowPolicies:     allowPolicies,
	}
}

// BuildDubboFilter returns the RBAC TCP filters built from the authorization policy.
func (b Builder) BuildDubboFilter() []*dubbopb.DubboFilter {
	filters := make([]*dubbopb.DubboFilter, 0)

	if denyConfig := build(b.denyPolicies, b.trustDomainBundle, rbacpb.RBAC_DENY); denyConfig != nil {
		filters = append(filters, createDubboRBACFilter(denyConfig))
	}
	if allowConfig := build(b.allowPolicies, b.trustDomainBundle, rbacpb.RBAC_ALLOW); allowConfig != nil {
		filters = append(filters, createDubboRBACFilter(allowConfig))
	}

	return filters
}

func build(policies []*dubboapi.DubboAuthorizationPolicy, tdBundle trustdomain.Bundle,
	action rbacpb.RBAC_Action) *rbacpb.RBAC {
	if len(policies) == 0 {
		return nil
	}

	rules := &rbacpb.RBAC{
		Action:   action,
		Policies: map[string]*rbacpb.Policy{},
	}

	for i := range policies {
		for i, rule := range policies[i].Spec.Rules {
			name := fmt.Sprintf("ns[%s]-policy[%s]-rule[%d]", policies[i].Namespace, policies[i].Name, i)
			if rule == nil {
				authzLog.Errorf("skipped nil rule %s", name)
				continue
			}
			m, err := authzmodel.New(rule)
			if err != nil {
				authzLog.Errorf("skipped rule %s: %v", name, err)
				continue
			}
			m.MigrateTrustDomain(tdBundle)
			generated, err := m.Generate(action)
			if err != nil {
				authzLog.Errorf("skipped rule %s: %v", name, err)
				continue
			}
			if generated != nil {
				rules.Policies[name] = generated
				authzLog.Debugf("rule %s generated policy: %+v", name, generated)
			}
		}
	}

	return rules
}

func createDubboRBACFilter(config *rbacpb.RBAC) *dubbopb.DubboFilter {
	if config == nil {
		return nil
	}

	rbacConfig := &rbacdubbopb.RBAC{
		Rules:      config,
		StatPrefix: authzmodel.RBACDubboFilterStatPrefix,
	}
	rbacPolicyInAny, _ := anypb.New(rbacConfig)
	return &dubbopb.DubboFilter{
		Name:   authzmodel.RBACDUBBOFilterName,
		Config: rbacPolicyInAny,
	}
}
