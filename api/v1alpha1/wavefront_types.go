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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WavefrontSpec defines the desired state of Wavefront
type WavefrontSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// CollectorEnabled is whether to enable the collector.
	CollectorEnabled bool `json:"collectorEnabled,required"`

	// ProxyUrl is the proxy URL that the collector sends metrics to.
	ProxyUrl string `json:"proxyUrl,omitempty"`

	// WavefrontUrl is the wavefront instance.
	WavefrontUrl string `json:"wavefrontUrl,required"`

	// WavefrontTokenSecret is the name of the secret that contains a wavefront API Token.
	WavefrontTokenSecret string `json:"wavefrontTokenSecret,required"`

	// ClusterName is a unique name for the Kubernetes cluster to be identified via a metric tag on Wavefront.
	ClusterName string `json:"clusterName,omitempty"`

	// ControllerManagerUID is for internal use of deletion delegation
	ControllerManagerUID string `json:"-"`

	// Metrics has resource configuration for node- and cluster-deployed collectors
	Metrics Metrics `json:"metrics,omitempty"`

	// DataExport options
	DataExport DataExport `json:"dataExport,omitempty"`
}

type Metrics struct {
	// Collector ConfigMap name. Leave blank to use defaults
	CollectorConfig string `json:"collectorConfig,omitempty"`

	// Cluster is for resource configuration for the cluster collector
	Cluster Collector `json:"cluster,omitempty"`

	// Node is for resource configuration for the node collector
	Node Collector `json:"node,omitempty"`
}

type DataExport struct {
	// Proxy configuration options
	Proxy Proxy `json:"proxy,omitempty"`
}

type Proxy struct {
	// Enabled is whether to enable the wavefront proxy.
	Enabled bool `json:"enabled,omitempty"`

	// MetricPort is the primary port for Wavefront data format metrics. Defaults to 2878.
	MetricPort int `json:"metricPort,omitempty"`

	// DeltaCounterPort accumulates 1-minute delta counters on Wavefront data format (usually 50000)
	DeltaCounterPort int `json:"deltaCounterPort,omitempty"`

	// Args is additional Wavefront proxy properties to be passed as command line arguments in the
	// --<property_name> <value> format. Multiple properties can be specified.
	Args string `json:"args,omitempty"`

	// HttpProxy is used in configurations when direct HTTP connections to Wavefront servers are not possible.
	HttpProxy HttpProxy `json:"httpProxy,omitempty"`

	// Distributed tracing configuration
	Tracing Tracing `json:"tracing,omitempty"`

	// Histogram distribution configuration
	Histogram Histogram `json:"histogram,omitempty"`
}

type HttpProxy struct {

	// Host must be used with httpProxy.port.
	Host string `json:"host,omitempty"`

	// Port must be used with httpProxy.host.
	Port int `json:"port,omitempty"`

	// Enable HTTP proxy with CA cert. Must be used with httpProxy.host and httpProxy.port if set to true.
	CAcert bool `json:"cacert,omitempty"`

	// When used with httpProxy.password, sets credentials to use with the HTTP proxy if the proxy requires authentication.
	User string `json:"user,omitempty"`

	// When used with httpProxy.user, sets credentials to use with the HTTP proxy if the proxy requires authentication.
	Password string `json:"password,omitempty"`
}

type Tracing struct {

	// Wavefront distributed tracing configurations
	Wavefront WavefrontTracing `json:"wavefront,omitempty"`

	// Jaeger distributed tracing configurations
	Jaeger JaegerTracing `json:"jaeger,omitempty"`

	// Zipkin distributed tracing configurations
	Zipkin ZipkinTracing `json:"zipkin,omitempty"`
}

type WavefrontTracing struct {
	// Port for distributed tracing data (usually 30000)
	Port int `json:"port,omitempty"`

	// Distributed tracing data sampling rate (0 to 1)
	SamplingRate int `json:"samplingRate,omitempty"`

	// When set to greater than 0, spans that exceed this duration will force trace to be sampled (ms)
	SamplingDuration int `json:"samplingDuration,omitempty"`
}

type JaegerTracing struct {
	// Port for Jaeger format distributed tracing data (usually 30001)
	Port int `json:"port,omitempty"`

	// Port for Jaeger Thrift format data (usually 30080)
	HttpPort int `json:"httpPort,omitempty"`

	// Port for Jaeger GRPC format data (usually 14250)
	GprcPort int `json:"gprcPort,omitempty"`

	// Custom application name for traces received on Jaeger's Http or Gprc port.
	ApplicationName int `json:"applicationName,omitempty"`
}

type ZipkinTracing struct {
	// Port for Zipkin format distributed tracing data (usually 9411)
	Port int `json:"port,omitempty"`

	// Custom application name for traces received on Zipkin's port.
	ApplicationName int `json:"applicationName,omitempty"`
}

type Histogram struct {
	// Port for histogram distribution format data (usually 40000)
	Port int `json:"port,omitempty"`

	// Port to accumulate 1-minute based histograms on Wavefront data format (usually 40001)
	MinutePort int `json:"minutePort,omitempty"`

	// Port to accumulate 1-hour based histograms on Wavefront data format (usually 40002)
	HourPort int `json:"hourPort,omitempty"`

	// Port to accumulate 1-day based histograms on Wavefront data format (usually 40003)
	DayPort int `json:"dayPort,omitempty"`
}

type Resource struct {
	// CPU is for specifying CPU requirements
	CPU string `json:"cpu,omitempty" yaml:"cpu,omitempty"`

	// Memory is for specifying Memory requirements
	Memory string `json:"memory,omitempty" yaml:"memory,omitempty"`
}

type Resources struct {
	// Request CPU and Memory requirements
	Requests Resource `json:"requests,omitempty" yaml:"requests,omitempty"`

	// Limit CPU and Memory requirements
	Limits Resource `json:"limits,omitempty" yaml:"limits,omitempty"`
}

type Collector struct {
	// Compute resources required by the Collector containers.
	Resources Resources `json:"resources,omitempty"`
}

// WavefrontStatus defines the observed state of Wavefront
type WavefrontStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Wavefront is the Schema for the wavefronts API
type Wavefront struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WavefrontSpec   `json:"spec,omitempty"`
	Status WavefrontStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WavefrontList contains a list of Wavefront
type WavefrontList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Wavefront `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Wavefront{}, &WavefrontList{})
}
