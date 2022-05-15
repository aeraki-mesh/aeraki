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
	fmt "fmt"
	proto "github.com/gogo/protobuf/proto"
	_ "github.com/gogo/protobuf/types"
	_ "istio.io/gogo-genproto/googleapis/google/api"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// DeepCopyInto supports using MetaRouter within kubernetes types, where deepcopy-gen is used.
func (in *MetaRouter) DeepCopyInto(out *MetaRouter) {
	p := proto.Clone(in).(*MetaRouter)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetaRouter. Required by controller-gen.
func (in *MetaRouter) DeepCopy() *MetaRouter {
	if in == nil {
		return nil
	}
	out := new(MetaRouter)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new MetaRouter. Required by controller-gen.
func (in *MetaRouter) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using MetaRoute within kubernetes types, where deepcopy-gen is used.
func (in *MetaRoute) DeepCopyInto(out *MetaRoute) {
	p := proto.Clone(in).(*MetaRoute)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetaRoute. Required by controller-gen.
func (in *MetaRoute) DeepCopy() *MetaRoute {
	if in == nil {
		return nil
	}
	out := new(MetaRoute)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new MetaRoute. Required by controller-gen.
func (in *MetaRoute) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using KeyValue within kubernetes types, where deepcopy-gen is used.
func (in *KeyValue) DeepCopyInto(out *KeyValue) {
	p := proto.Clone(in).(*KeyValue)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KeyValue. Required by controller-gen.
func (in *KeyValue) DeepCopy() *KeyValue {
	if in == nil {
		return nil
	}
	out := new(KeyValue)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new KeyValue. Required by controller-gen.
func (in *KeyValue) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using MetaRouteMatch within kubernetes types, where deepcopy-gen is used.
func (in *MetaRouteMatch) DeepCopyInto(out *MetaRouteMatch) {
	p := proto.Clone(in).(*MetaRouteMatch)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetaRouteMatch. Required by controller-gen.
func (in *MetaRouteMatch) DeepCopy() *MetaRouteMatch {
	if in == nil {
		return nil
	}
	out := new(MetaRouteMatch)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new MetaRouteMatch. Required by controller-gen.
func (in *MetaRouteMatch) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using StringMatch within kubernetes types, where deepcopy-gen is used.
func (in *StringMatch) DeepCopyInto(out *StringMatch) {
	p := proto.Clone(in).(*StringMatch)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StringMatch. Required by controller-gen.
func (in *StringMatch) DeepCopy() *StringMatch {
	if in == nil {
		return nil
	}
	out := new(StringMatch)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new StringMatch. Required by controller-gen.
func (in *StringMatch) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using MetaRouteDestination within kubernetes types, where deepcopy-gen is used.
func (in *MetaRouteDestination) DeepCopyInto(out *MetaRouteDestination) {
	p := proto.Clone(in).(*MetaRouteDestination)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetaRouteDestination. Required by controller-gen.
func (in *MetaRouteDestination) DeepCopy() *MetaRouteDestination {
	if in == nil {
		return nil
	}
	out := new(MetaRouteDestination)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new MetaRouteDestination. Required by controller-gen.
func (in *MetaRouteDestination) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using Destination within kubernetes types, where deepcopy-gen is used.
func (in *Destination) DeepCopyInto(out *Destination) {
	p := proto.Clone(in).(*Destination)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Destination. Required by controller-gen.
func (in *Destination) DeepCopy() *Destination {
	if in == nil {
		return nil
	}
	out := new(Destination)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new Destination. Required by controller-gen.
func (in *Destination) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using PortSelector within kubernetes types, where deepcopy-gen is used.
func (in *PortSelector) DeepCopyInto(out *PortSelector) {
	p := proto.Clone(in).(*PortSelector)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PortSelector. Required by controller-gen.
func (in *PortSelector) DeepCopy() *PortSelector {
	if in == nil {
		return nil
	}
	out := new(PortSelector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new PortSelector. Required by controller-gen.
func (in *PortSelector) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using LocalRateLimit within kubernetes types, where deepcopy-gen is used.
func (in *LocalRateLimit) DeepCopyInto(out *LocalRateLimit) {
	p := proto.Clone(in).(*LocalRateLimit)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LocalRateLimit. Required by controller-gen.
func (in *LocalRateLimit) DeepCopy() *LocalRateLimit {
	if in == nil {
		return nil
	}
	out := new(LocalRateLimit)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new LocalRateLimit. Required by controller-gen.
func (in *LocalRateLimit) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using LocalRateLimit_TokenBucket within kubernetes types, where deepcopy-gen is used.
func (in *LocalRateLimit_TokenBucket) DeepCopyInto(out *LocalRateLimit_TokenBucket) {
	p := proto.Clone(in).(*LocalRateLimit_TokenBucket)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LocalRateLimit_TokenBucket. Required by controller-gen.
func (in *LocalRateLimit_TokenBucket) DeepCopy() *LocalRateLimit_TokenBucket {
	if in == nil {
		return nil
	}
	out := new(LocalRateLimit_TokenBucket)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new LocalRateLimit_TokenBucket. Required by controller-gen.
func (in *LocalRateLimit_TokenBucket) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using LocalRateLimit_Condition within kubernetes types, where deepcopy-gen is used.
func (in *LocalRateLimit_Condition) DeepCopyInto(out *LocalRateLimit_Condition) {
	p := proto.Clone(in).(*LocalRateLimit_Condition)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LocalRateLimit_Condition. Required by controller-gen.
func (in *LocalRateLimit_Condition) DeepCopy() *LocalRateLimit_Condition {
	if in == nil {
		return nil
	}
	out := new(LocalRateLimit_Condition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new LocalRateLimit_Condition. Required by controller-gen.
func (in *LocalRateLimit_Condition) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using GlobalRateLimit within kubernetes types, where deepcopy-gen is used.
func (in *GlobalRateLimit) DeepCopyInto(out *GlobalRateLimit) {
	p := proto.Clone(in).(*GlobalRateLimit)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GlobalRateLimit. Required by controller-gen.
func (in *GlobalRateLimit) DeepCopy() *GlobalRateLimit {
	if in == nil {
		return nil
	}
	out := new(GlobalRateLimit)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new GlobalRateLimit. Required by controller-gen.
func (in *GlobalRateLimit) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using GlobalRateLimit_Descriptor within kubernetes types, where deepcopy-gen is used.
func (in *GlobalRateLimit_Descriptor) DeepCopyInto(out *GlobalRateLimit_Descriptor) {
	p := proto.Clone(in).(*GlobalRateLimit_Descriptor)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GlobalRateLimit_Descriptor. Required by controller-gen.
func (in *GlobalRateLimit_Descriptor) DeepCopy() *GlobalRateLimit_Descriptor {
	if in == nil {
		return nil
	}
	out := new(GlobalRateLimit_Descriptor)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new GlobalRateLimit_Descriptor. Required by controller-gen.
func (in *GlobalRateLimit_Descriptor) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using Percent within kubernetes types, where deepcopy-gen is used.
func (in *Percent) DeepCopyInto(out *Percent) {
	p := proto.Clone(in).(*Percent)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Percent. Required by controller-gen.
func (in *Percent) DeepCopy() *Percent {
	if in == nil {
		return nil
	}
	out := new(Percent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new Percent. Required by controller-gen.
func (in *Percent) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}
