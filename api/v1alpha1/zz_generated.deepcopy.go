//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Collector) DeepCopyInto(out *Collector) {
	*out = *in
	out.Resources = in.Resources
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Collector.
func (in *Collector) DeepCopy() *Collector {
	if in == nil {
		return nil
	}
	out := new(Collector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataExport) DeepCopyInto(out *DataExport) {
	*out = *in
	out.Proxy = in.Proxy
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataExport.
func (in *DataExport) DeepCopy() *DataExport {
	if in == nil {
		return nil
	}
	out := new(DataExport)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Histogram) DeepCopyInto(out *Histogram) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Histogram.
func (in *Histogram) DeepCopy() *Histogram {
	if in == nil {
		return nil
	}
	out := new(Histogram)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HttpProxy) DeepCopyInto(out *HttpProxy) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HttpProxy.
func (in *HttpProxy) DeepCopy() *HttpProxy {
	if in == nil {
		return nil
	}
	out := new(HttpProxy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JaegerTracing) DeepCopyInto(out *JaegerTracing) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JaegerTracing.
func (in *JaegerTracing) DeepCopy() *JaegerTracing {
	if in == nil {
		return nil
	}
	out := new(JaegerTracing)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Metrics) DeepCopyInto(out *Metrics) {
	*out = *in
	out.Cluster = in.Cluster
	out.Node = in.Node
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Metrics.
func (in *Metrics) DeepCopy() *Metrics {
	if in == nil {
		return nil
	}
	out := new(Metrics)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Proxy) DeepCopyInto(out *Proxy) {
	*out = *in
	out.HttpProxy = in.HttpProxy
	out.Tracing = in.Tracing
	out.Histogram = in.Histogram
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Proxy.
func (in *Proxy) DeepCopy() *Proxy {
	if in == nil {
		return nil
	}
	out := new(Proxy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Resource) DeepCopyInto(out *Resource) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Resource.
func (in *Resource) DeepCopy() *Resource {
	if in == nil {
		return nil
	}
	out := new(Resource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Resources) DeepCopyInto(out *Resources) {
	*out = *in
	out.Requests = in.Requests
	out.Limits = in.Limits
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Resources.
func (in *Resources) DeepCopy() *Resources {
	if in == nil {
		return nil
	}
	out := new(Resources)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Tracing) DeepCopyInto(out *Tracing) {
	*out = *in
	out.Wavefront = in.Wavefront
	out.Jaeger = in.Jaeger
	out.Zipkin = in.Zipkin
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Tracing.
func (in *Tracing) DeepCopy() *Tracing {
	if in == nil {
		return nil
	}
	out := new(Tracing)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Wavefront) DeepCopyInto(out *Wavefront) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Wavefront.
func (in *Wavefront) DeepCopy() *Wavefront {
	if in == nil {
		return nil
	}
	out := new(Wavefront)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Wavefront) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WavefrontList) DeepCopyInto(out *WavefrontList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Wavefront, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WavefrontList.
func (in *WavefrontList) DeepCopy() *WavefrontList {
	if in == nil {
		return nil
	}
	out := new(WavefrontList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *WavefrontList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WavefrontSpec) DeepCopyInto(out *WavefrontSpec) {
	*out = *in
	out.Metrics = in.Metrics
	out.DataExport = in.DataExport
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WavefrontSpec.
func (in *WavefrontSpec) DeepCopy() *WavefrontSpec {
	if in == nil {
		return nil
	}
	out := new(WavefrontSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WavefrontStatus) DeepCopyInto(out *WavefrontStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WavefrontStatus.
func (in *WavefrontStatus) DeepCopy() *WavefrontStatus {
	if in == nil {
		return nil
	}
	out := new(WavefrontStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WavefrontTracing) DeepCopyInto(out *WavefrontTracing) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WavefrontTracing.
func (in *WavefrontTracing) DeepCopy() *WavefrontTracing {
	if in == nil {
		return nil
	}
	out := new(WavefrontTracing)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ZipkinTracing) DeepCopyInto(out *ZipkinTracing) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ZipkinTracing.
func (in *ZipkinTracing) DeepCopy() *ZipkinTracing {
	if in == nil {
		return nil
	}
	out := new(ZipkinTracing)
	in.DeepCopyInto(out)
	return out
}
