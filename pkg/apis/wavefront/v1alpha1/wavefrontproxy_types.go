package v1alpha1

import (
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
	Url string `json:"url"`

	// Wavefront API Token.
	Token string `json:"token"`

	// The no. of replicas for Wavefront Proxy. Defaults to 1
	Size *int32 `json:"size,omitempty"`

	// The port number the proxy will listen on for metrics in Wavefront data format.
	// This is usually port 2878 by default.
	MetricPort int32 `json:"metricPort,omitempty"`

	// The port to listen on for Wavefront trace formatted data. Defaults to none.
	// This is usually 30000
	TracePort int32 `json:"tracePort,omitempty"`

	// The port to listen on for Jaeger Thrift formatted data. Defaults to none.
	// This is usually 30001
	JaegerPort int32 `json:"jaegerPort,omitempty"`

	// The port to listen on for Zipkin formatted data. Defaults to none.
	// This is usually 9411
	ZipkinPort int32 `json:"zipkinPort,omitempty"`

	// Sampling rate to apply to tracing spans sent to the proxy. This rate is applied to all
	// data formats the proxy is listening on.
	// Value should be between 0.0 and 1.0.  Default is 1.0
	TraceSamplingRate float64 `json:"traceSamplingRate,omitempty"`

	// When this is set to a value greater than 0, spans that are greater than or equal to this value will be sampled.
	TraceSamplingDuration float64 `json:"traceSamplingDuration,omitempty"`

	// The port to listen on for Wavefront histogram distribution formatted data.
	// This is usually 40000
	HistogramDistPort int32 `json:"histogramDistPort,omitempty"`

	// The name of the config map providing the preprocessor rules for the Wavefront proxy.
	Preprocessor string `json:"preprocessor,omitempty"`

	// The name of the config map providing the advanced configurations for the Wavefront proxy.
	Advanced string `json:"advanced,omitempty"`

	// The comma separated list of ports that need to be opened on Proxy Pod and Services.
	// Needs to be explicitly specified when using "Advanced" configuration.
	AdditionalPorts string `json:"additionalPorts,omitempty"`

	// If set to true, Proxy pods will be upgraded automatically in case new minor upgrade version is available.
	// For pinning Proxy to a specific version, you will need to set this option to false.
	// We support only minor version Auto Upgrades.
	EnableAutoUpgrade bool `json:"enableAutoUpgrade,omitempty"`

	// Openshift Specific configurations starts from here

	// Set to true when running proxy in Openshift platform.
	Openshift bool `json:"openshift,omitempty"`

	// The name of the storage claim to be used for creating proxy buffers directory.
	// This is applicable only in an Openshift environment."
	StorageClaimName string `json:"storageClaimName,omitempty"`
}

// WavefrontProxyStatus defines the observed state of WavefrontProxy
// +k8s:openapi-gen=true
type WavefrontProxyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	Version          string      `json:"version,omitempty"`
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
