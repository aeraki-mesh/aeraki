// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: api/metaprotocol/v1alpha1/metaprotocol_metarouter.proto

// $schema: metaprotocol.aeraki.io.v1alpha1.MetaRouter
// $title: MetaRouter
// $description: MetaRouter defines route policies for MetaProtocol proxy.
//
// MetaRouter defines route policies for MetaProtocol proxy.
//
// ```yaml
// apiVersion: metaprotocol.aeraki.io/v1alpha1
// kind: MetaRouter
// metadata:
//   name: attribute-based-route
//   namespace: istio-system
// spec:
//   hosts:
//   - org.apache.dubbo.samples.basic.api.demoservice
//   routes:
//   - name: v1
//     match:
//       attributes:
//         interface:
//           exact: org.apache.dubbo.samples.basic.api.DemoService
//         method:
//           exact: sayHello
//     route:
//     - destination:
//         host: org.apache.dubbo.samples.basic.api.demoservice
//         subset: v1
//
// ```
//
// ```yaml
// apiVersion: metaprotocol.aeraki.io/v1alpha1
// kind: MetaRouter
// metadata:
//   name: traffic-splitting
// spec:
//   hosts:
//     - org.apache.dubbo.samples.basic.api.demoservice
//   routes:
//     - name: traffic-spilt
//       route:
//         - destination:
//             host: org.apache.dubbo.samples.basic.api.demoservice
//             subset: v1
//           weight: 20
//         - destination:
//             host: org.apache.dubbo.samples.basic.api.demoservice
//             subset: v2
//           weight: 80

package v1alpha1

import (
	bytes "bytes"
	fmt "fmt"
	github_com_gogo_protobuf_jsonpb "github.com/gogo/protobuf/jsonpb"
	proto "github.com/gogo/protobuf/proto"
	_ "github.com/gogo/protobuf/types"
	_ "istio.io/gogo-genproto/googleapis/google/api"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// MarshalJSON is a custom marshaler for MetaRouter
func (this *MetaRouter) MarshalJSON() ([]byte, error) {
	str, err := MetaprotocolMetarouterMarshaler.MarshalToString(this)
	return []byte(str), err
}

// UnmarshalJSON is a custom unmarshaler for MetaRouter
func (this *MetaRouter) UnmarshalJSON(b []byte) error {
	return MetaprotocolMetarouterUnmarshaler.Unmarshal(bytes.NewReader(b), this)
}

// MarshalJSON is a custom marshaler for MetaRoute
func (this *MetaRoute) MarshalJSON() ([]byte, error) {
	str, err := MetaprotocolMetarouterMarshaler.MarshalToString(this)
	return []byte(str), err
}

// UnmarshalJSON is a custom unmarshaler for MetaRoute
func (this *MetaRoute) UnmarshalJSON(b []byte) error {
	return MetaprotocolMetarouterUnmarshaler.Unmarshal(bytes.NewReader(b), this)
}

// MarshalJSON is a custom marshaler for MetaRouteMatch
func (this *MetaRouteMatch) MarshalJSON() ([]byte, error) {
	str, err := MetaprotocolMetarouterMarshaler.MarshalToString(this)
	return []byte(str), err
}

// UnmarshalJSON is a custom unmarshaler for MetaRouteMatch
func (this *MetaRouteMatch) UnmarshalJSON(b []byte) error {
	return MetaprotocolMetarouterUnmarshaler.Unmarshal(bytes.NewReader(b), this)
}

// MarshalJSON is a custom marshaler for StringMatch
func (this *StringMatch) MarshalJSON() ([]byte, error) {
	str, err := MetaprotocolMetarouterMarshaler.MarshalToString(this)
	return []byte(str), err
}

// UnmarshalJSON is a custom unmarshaler for StringMatch
func (this *StringMatch) UnmarshalJSON(b []byte) error {
	return MetaprotocolMetarouterUnmarshaler.Unmarshal(bytes.NewReader(b), this)
}

// MarshalJSON is a custom marshaler for MetaRouteDestination
func (this *MetaRouteDestination) MarshalJSON() ([]byte, error) {
	str, err := MetaprotocolMetarouterMarshaler.MarshalToString(this)
	return []byte(str), err
}

// UnmarshalJSON is a custom unmarshaler for MetaRouteDestination
func (this *MetaRouteDestination) UnmarshalJSON(b []byte) error {
	return MetaprotocolMetarouterUnmarshaler.Unmarshal(bytes.NewReader(b), this)
}

// MarshalJSON is a custom marshaler for Destination
func (this *Destination) MarshalJSON() ([]byte, error) {
	str, err := MetaprotocolMetarouterMarshaler.MarshalToString(this)
	return []byte(str), err
}

// UnmarshalJSON is a custom unmarshaler for Destination
func (this *Destination) UnmarshalJSON(b []byte) error {
	return MetaprotocolMetarouterUnmarshaler.Unmarshal(bytes.NewReader(b), this)
}

// MarshalJSON is a custom marshaler for PortSelector
func (this *PortSelector) MarshalJSON() ([]byte, error) {
	str, err := MetaprotocolMetarouterMarshaler.MarshalToString(this)
	return []byte(str), err
}

// UnmarshalJSON is a custom unmarshaler for PortSelector
func (this *PortSelector) UnmarshalJSON(b []byte) error {
	return MetaprotocolMetarouterUnmarshaler.Unmarshal(bytes.NewReader(b), this)
}

// MarshalJSON is a custom marshaler for LocalRateLimit
func (this *LocalRateLimit) MarshalJSON() ([]byte, error) {
	str, err := MetaprotocolMetarouterMarshaler.MarshalToString(this)
	return []byte(str), err
}

// UnmarshalJSON is a custom unmarshaler for LocalRateLimit
func (this *LocalRateLimit) UnmarshalJSON(b []byte) error {
	return MetaprotocolMetarouterUnmarshaler.Unmarshal(bytes.NewReader(b), this)
}

// MarshalJSON is a custom marshaler for TokenBucket
func (this *TokenBucket) MarshalJSON() ([]byte, error) {
	str, err := MetaprotocolMetarouterMarshaler.MarshalToString(this)
	return []byte(str), err
}

// UnmarshalJSON is a custom unmarshaler for TokenBucket
func (this *TokenBucket) UnmarshalJSON(b []byte) error {
	return MetaprotocolMetarouterUnmarshaler.Unmarshal(bytes.NewReader(b), this)
}

// MarshalJSON is a custom marshaler for LocalRateLimitCondition
func (this *LocalRateLimitCondition) MarshalJSON() ([]byte, error) {
	str, err := MetaprotocolMetarouterMarshaler.MarshalToString(this)
	return []byte(str), err
}

// UnmarshalJSON is a custom unmarshaler for LocalRateLimitCondition
func (this *LocalRateLimitCondition) UnmarshalJSON(b []byte) error {
	return MetaprotocolMetarouterUnmarshaler.Unmarshal(bytes.NewReader(b), this)
}

var (
	MetaprotocolMetarouterMarshaler   = &github_com_gogo_protobuf_jsonpb.Marshaler{}
	MetaprotocolMetarouterUnmarshaler = &github_com_gogo_protobuf_jsonpb.Unmarshaler{AllowUnknownFields: true}
)
