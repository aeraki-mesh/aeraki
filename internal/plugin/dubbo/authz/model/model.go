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

package model

import (
	"fmt"

	rbacpb "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	"istio.io/istio/pilot/pkg/security/trustdomain"

	dubbopb "github.com/aeraki-mesh/api/dubbo/v1alpha1"
)

const (
	// RBACDUBBOFilterName is the name of the dubbo rbac filter
	RBACDUBBOFilterName = "envoy.filters.dubbo.rbac"
	// RBACDubboFilterStatPrefix is the stat prefix for dubbo rbac filter
	RBACDubboFilterStatPrefix = "dubbo."

	attrSrcNamespace = "source.namespace" // e.g. "default".
	attrSrcPrincipal = "source.principal" // source identity, e,g, "cluster.local/ns/default/sa/productpage".

	// Internal names used to generate corresponding Envoy matcher.
	dubboInterface = "dubboInterface"
	dubboMethod    = "dubboMethod"
)

type rule struct {
	key       string
	values    []string
	notValues []string
	g         generator
}

type ruleList struct {
	rules []*rule
}

// Model represents a single rule from an authorization policy. The conditions of the rule are consolidated into
// permission or principal to align with the Envoy RBAC filter API.
type Model struct {
	permissions []ruleList
	principals  []ruleList
}

// New returns a model representing a single authorization policy.
func New(r *dubbopb.Rule) (*Model, error) {
	m := Model{}

	basePermission := ruleList{}
	basePrincipal := ruleList{}

	for _, from := range r.From {
		merged := basePrincipal.copy()
		if s := from.Source; s != nil {
			merged.insertFront(srcNamespaceGenerator{}, attrSrcNamespace, s.Namespaces, s.NotNamespaces)
			merged.insertFront(srcPrincipalGenerator{}, attrSrcPrincipal, s.Principals, s.NotPrincipals)
		}
		m.principals = append(m.principals, merged)
	}
	if len(r.From) == 0 {
		m.principals = append(m.principals, basePrincipal)
	}

	for _, to := range r.To {
		merged := basePermission.copy()
		if o := to.Operation; o != nil {
			merged.insertFront(interfaceGenerator{}, dubboInterface, o.Interfaces, o.NotInterfaces)
			merged.insertFront(methodGenerator{}, dubboMethod, o.Methods, o.NotMethods)
		}
		m.permissions = append(m.permissions, merged)
	}
	if len(r.To) == 0 {
		m.permissions = append(m.permissions, basePermission)
	}

	return &m, nil
}

// MigrateTrustDomain replaces the trust domain in source principal based on the trust domain aliases information.
func (m *Model) MigrateTrustDomain(tdBundle trustdomain.Bundle) {
	for _, p := range m.principals {
		for _, r := range p.rules {
			if r.key == attrSrcPrincipal {
				if len(r.values) != 0 {
					r.values = tdBundle.ReplaceTrustDomainAliases(r.values)
				}
				if len(r.notValues) != 0 {
					r.notValues = tdBundle.ReplaceTrustDomainAliases(r.notValues)
				}
			}
		}
	}
}

// Generate generates the Envoy RBAC config from the model.
func (m *Model) Generate(action rbacpb.RBAC_Action) (*rbacpb.Policy, error) {
	var permissions []*rbacpb.Permission
	for _, rl := range m.permissions {
		permission, err := generatePermission(rl, action)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}
	if len(permissions) == 0 {
		return nil, fmt.Errorf("must have at least 1 permission")
	}

	var principals []*rbacpb.Principal
	for _, rl := range m.principals {
		principal, err := generatePrincipal(rl, action)
		if err != nil {
			return nil, err
		}
		principals = append(principals, principal)
	}
	if len(principals) == 0 {
		return nil, fmt.Errorf("must have at least 1 principal")
	}

	return &rbacpb.Policy{
		Permissions: permissions,
		Principals:  principals,
	}, nil
}

func generatePermission(rl ruleList, action rbacpb.RBAC_Action) (*rbacpb.Permission, error) {
	var and []*rbacpb.Permission
	for _, r := range rl.rules {
		ret, err := r.permission(action)
		if err != nil {
			return nil, err
		}
		and = append(and, ret...)
	}
	if len(and) == 0 {
		and = append(and, permissionAny())
	}
	return permissionAnd(and), nil
}

func generatePrincipal(rl ruleList, action rbacpb.RBAC_Action) (*rbacpb.Principal, error) {
	var and []*rbacpb.Principal
	for _, r := range rl.rules {
		ret, err := r.principal(action)
		if err != nil {
			return nil, err
		}
		and = append(and, ret...)
	}
	if len(and) == 0 {
		and = append(and, principalAny())
	}
	return principalAnd(and), nil
}

// nolint: dupl
func (r *rule) permission(action rbacpb.RBAC_Action) ([]*rbacpb.Permission, error) {
	var permissions []*rbacpb.Permission
	var or []*rbacpb.Permission
	for _, value := range r.values {
		p, err := r.g.permission(r.key, value)
		if err := r.checkError(action, err); err != nil {
			return nil, err
		}
		if p != nil {
			or = append(or, p)
		}
	}
	if len(or) > 0 {
		permissions = append(permissions, permissionOr(or))
	}

	or = nil
	for _, notValue := range r.notValues {
		p, err := r.g.permission(r.key, notValue)
		if err := r.checkError(action, err); err != nil {
			return nil, err
		}
		if p != nil {
			or = append(or, p)
		}
	}
	if len(or) > 0 {
		permissions = append(permissions, permissionNot(permissionOr(or)))
	}
	return permissions, nil
}

// nolint: dupl
func (r *rule) principal(action rbacpb.RBAC_Action) ([]*rbacpb.Principal, error) {
	var principals []*rbacpb.Principal
	var or []*rbacpb.Principal
	for _, value := range r.values {
		p, err := r.g.principal(r.key, value)
		if err := r.checkError(action, err); err != nil {
			return nil, err
		}
		if p != nil {
			or = append(or, p)
		}
	}
	if len(or) > 0 {
		principals = append(principals, principalOr(or))
	}

	or = nil
	for _, notValue := range r.notValues {
		p, err := r.g.principal(r.key, notValue)
		if err := r.checkError(action, err); err != nil {
			return nil, err
		}
		if p != nil {
			or = append(or, p)
		}
	}
	if len(or) > 0 {
		principals = append(principals, principalNot(principalOr(or)))
	}
	return principals, nil
}

func (r *rule) checkError(action rbacpb.RBAC_Action, err error) error {
	if action == rbacpb.RBAC_ALLOW {
		// Return the error as-is for allow policy. This will make all rules in the current permission ignored, effectively
		// result in a smaller allow policy (i.e. less likely to allow a request).
		return err
	}

	// Ignore the error for a deny or audit policy. This will make the current rule ignored and continue the generation of
	// the next rule, effectively resulting in a wider deny or audit policy (i.e. more likely to deny or audit a request).
	return nil
}

func (p *ruleList) copy() ruleList {
	r := ruleList{}
	r.rules = append([]*rule{}, p.rules...)
	return r
}

func (p *ruleList) insertFront(g generator, key string, values, notValues []string) {
	if len(values) == 0 && len(notValues) == 0 {
		return
	}
	r := &rule{
		key:       key,
		values:    values,
		notValues: notValues,
		g:         g,
	}

	p.rules = append([]*rule{r}, p.rules...)
}
