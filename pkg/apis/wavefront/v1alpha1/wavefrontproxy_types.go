package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WavefrontProxySpec defines the desired state of WavefrontProxy
// +k8s:openapi-gen=true
type WavefrontProxySpec struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// The WavefrontProxy image to use. Defaults to wavefronthq/proxy:latest
	Image string `json:"image,omitempty"`

	// Wavefront URL (cluster).
	Url string `json:"url,omitempty"`

	// Wavefront API Token.
	Token string `json:"token,omitempty"`

	// The no. of replicas for Wavefront Proxy
	Size int32 `json:"size,omitempty"`

	// Whether proxy is enabled.
	ProxyEnabled bool `json:"proxyEnabled,omitempty"`

	Config ProxyConfig `json:"config,omitempty"`

	//// The port number the proxy will listen on for metrics in Wavefront data format.
	//// This is usually port 2878
	MetricPorts int32 `json:"metricPorts,omitempty"`
}

type ProxyConfig struct {
	MetricsConfig `json:"metrics,omitempty"`

	TraceConfig `json:"trace,omitempty"`

	HistogramConfig `json:"histogram,omitempty"`

	AdvancedConfig corev1.ConfigMap `json:"advanced,omitempty"`

	DataPreProcessConfig `json:"dataPreprocess,omitempty"`
}

type MetricsConfig struct {
	// Comma-separated list of ports to listen on for Wavefront formatted data.(Default: 2878)
	//MetricPorts int32 `json:"metricPorts,omitempty"`
}

type TraceConfig struct {

	// Comma-separated list of ports to listen on for Wavefront trace formatted data. Defaults to none.
	// This is usually 30000
	TracePorts int32 `json:"tracePorts,omitempty"`

	// Comma-separated list of ports on which to listen on for Jaeger Thrift formatted data. Defaults to none.
	// This is usually 30001
	JaegerPorts int32 `json:"jaegerPorts,omitempty"`

	// Comma-separated list of ports on which to listen on for Zipkin Thrift formatted data. Defaults to none.
	// This is usually 9411
	ZipkinPorts int32 `json:"zipkinPorts,omitempty"`

	// Sampling rate to apply to tracing spans sent to the proxy. This rate is applied to all
	// data formats the proxy is listening on.
	// Value should be between 0.0 and 1.0.  Default is 1.0
	TraceSamplingRate float32 `json:"traceSamplingRate,omitempty"`

	// When this is set to a value greater than 0, spans that are greater than or equal to this value will be sampled.
	TraceSamplingDuration float32 `json:"traceSamplingDuration,omitempty"`
}

type HistogramConfig struct {
	// Comma-separated list of ports to listen on for Wavefront histogram distribution formatted data.
	// This is usually 40000
	HistogramDistPorts string `json:"histogramDistPorts,omitempty"`
}

type DataPreProcessConfig struct {
	PreprocessorConfigFile corev1.ConfigMap `json:"preprocessorConfigFile,omitempty"`
	CustomSourceTags       string           `json:"customSourceTags,omitempty"`
	Prefix                 string           `json:"prefix,omitempty"`
	WhitelistRegex         string           `json:"whitelistRegex,omitempty"`
	BlacklistRegex         string           `json:"blacklistRegex,omitempty"`
}

// WavefrontProxyStatus defines the observed state of WavefrontProxy
// +k8s:openapi-gen=true
type WavefrontProxyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Version          string      `json:"version,omitempty"`
	CreatedTimestamp metav1.Time `json:"createdTimestamp,omitempty"`
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WavefrontProxy is the Schema for the wavefrontproxies API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type WavefrontProxy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WavefrontProxySpec   `json:"spec,omitempty"`
	Status WavefrontProxyStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WavefrontProxyList contains a list of WavefrontProxy
type WavefrontProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WavefrontProxy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WavefrontProxy{}, &WavefrontProxyList{})
}
