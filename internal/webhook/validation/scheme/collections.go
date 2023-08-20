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

package scheme

import (
	"reflect"

	metaprotocolv1alpha1 "github.com/aeraki-mesh/api/metaprotocol/v1alpha1"
	"istio.io/istio/pkg/config/schema/collection"
	"istio.io/istio/pkg/config/schema/resource"
)

var (

	// AerakiMetaprotocolV1Alpha1Applicationprotocols describes the collection
	// aeraki/metaprotocol/v1alpha1/applicationprotocols
	AerakiMetaprotocolV1Alpha1Applicationprotocols = resource.Builder{
		Identifier: "ApplicationProtocol",
		Group:      "metaprotocol.aeraki.io",
		Kind:       "ApplicationProtocol",
		Plural:     "applicationprotocols",
		Version:    "v1alpha1",
		VersionAliases: []string{
			"v1",
		},
		Proto: "aeraki.io.v1alpha1.ApplicationProtocol",
		//StatusProto: "istio.meta.v1alpha1.IstioStatus",
		ReflectType: reflect.TypeOf(&metaprotocolv1alpha1.ApplicationProtocol{}).Elem(),
		//StatusType: reflect.TypeOf(&istioioapimetav1alpha1.IstioStatus{}).Elem(),
		ProtoPackage: "github.com/aeraki-mesh/api/metaprotocol/v1alpha1",
		// StatusPackage: "istio.io/api/meta/v1alpha1",
		ClusterScoped: false,
		Synthetic:     false,
		Builtin:       false,
		ValidateProto: ValidateApplicationProtocol,
	}.MustBuild()

	// AerakiMetaprotocolV1Alpha1Metarouters describes the collection
	// aeraki/metaprotocol/v1alpha1/metarouters
	AerakiMetaprotocolV1Alpha1Metarouters = resource.Builder{
		Identifier: "MetaRouter",
		Group:      "metaprotocol.aeraki.io",
		Kind:       "MetaRouter",
		Plural:     "metarouters",
		Version:    "v1alpha1",
		VersionAliases: []string{
			"v1",
		},
		Proto: "aeraki.io.v1alpha1.MetaRouter",
		//StatusProto: "istio.meta.v1alpha1.IstioStatus",
		ReflectType: reflect.TypeOf(&metaprotocolv1alpha1.MetaRouter{}).Elem(),
		//StatusType: reflect.TypeOf(&istioioapimetav1alpha1.IstioStatus{}).Elem(),
		ProtoPackage: "github.com/aeraki-mesh/api/metaprotocol/v1alpha1",
		// StatusPackage: "istio.io/api/meta/v1alpha1",
		ClusterScoped: false,
		Synthetic:     false,
		Builtin:       false,
		ValidateProto: ValidateMetaRouter,
	}.MustBuild()

	// Aeraki contains Aeraki collections in the system.
	Aeraki = collection.NewSchemasBuilder().
		MustAdd(AerakiMetaprotocolV1Alpha1Applicationprotocols).
		MustAdd(AerakiMetaprotocolV1Alpha1Metarouters).
		Build()
)
