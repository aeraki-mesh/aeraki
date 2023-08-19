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
	"strings"

	rbacpb "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	"istio.io/istio/pkg/spiffe"

	"github.com/aeraki-mesh/aeraki/internal/plugin/dubbo/authz/matcher"
)

type generator interface {
	permission(key, value string) (*rbacpb.Permission, error)
	principal(key, value string) (*rbacpb.Principal, error)
}

type interfaceGenerator struct {
}

func (interfaceGenerator) permission(_, value string) (*rbacpb.Permission, error) {
	m := matcher.MetadataStringMatcher("envoy.filters.dubbo.rbac", "service", matcher.StringMatcher(value))
	return permissionMetadata(m), nil
}

func (interfaceGenerator) principal(_, _ string) (*rbacpb.Principal, error) {
	return nil, fmt.Errorf("unimplemented")
}

type methodGenerator struct {
}

func (methodGenerator) permission(_, value string) (*rbacpb.Permission, error) {
	m := matcher.MetadataStringMatcher("envoy.filters.dubbo.rbac", "method", matcher.StringMatcher(value))
	return permissionMetadata(m), nil
}

func (methodGenerator) principal(_, _ string) (*rbacpb.Principal, error) {
	return nil, fmt.Errorf("unimplemented")
}

type srcNamespaceGenerator struct {
}

func (srcNamespaceGenerator) permission(_, _ string) (*rbacpb.Permission, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (srcNamespaceGenerator) principal(_, value string) (*rbacpb.Principal, error) {
	v := strings.Replace(value, "*", ".*", -1)
	m := matcher.StringMatcherRegex(fmt.Sprintf(".*/ns/%s/.*", v))
	return principalAuthenticated(m), nil
}

type srcPrincipalGenerator struct {
}

func (srcPrincipalGenerator) permission(_, _ string) (*rbacpb.Permission, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (srcPrincipalGenerator) principal(_, value string) (*rbacpb.Principal, error) {
	m := matcher.StringMatcherWithPrefix(value, spiffe.URIPrefix)
	return principalAuthenticated(m), nil
}
