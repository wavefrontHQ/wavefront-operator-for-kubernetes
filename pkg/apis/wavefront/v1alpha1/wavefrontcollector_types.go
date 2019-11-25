package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WavefrontCollectorSpec defines the desired state of WavefrontCollector
// +k8s:openapi-gen=true
type WavefrontCollectorSpec struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// Defaults to wavefronthq/wavefront-kubernetes-collector:latest
	Image string `json:"image,omitempty"`

	// Whether to deploy the collector as a daemonset. False will roll out as a deployment.
	Daemon bool `json:"daemon,omitempty"`

	// Whether to enable debug logging and profiling
	EnableDebug bool `json:"enableDebug,omitempty"`

	// List of environment variables to set for the Collector containers.
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Compute resources required by the Collector containers.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Tolerations for the collector pods
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// The name of the config map providing the configuration for the collector instance.
	// If empty, a default name of "collectorName-config" is assumed.
	ConfigName string `json:"configName,omitempty"`

	// If set to true, Collector pods will be upgraded automatically in case new minor upgrade version is available.
	// For pinning Collector to a specific version, you will need to set this option to false.
	// We support only minor version Auto Upgrades.
	EnableAutoUpgrade bool `json:"enableAutoUpgrade,omitempty"`

	// Openshift Specific configurations starts from here.

	// Set to true when running collector in Openshift platform.
	Openshift bool `json:"openshift,omitempty"`

	// If set to true, Collector will use default config bundled in the image
	// else it will use the config from ConfigName
	UseOpenshiftDefaultConfig bool `json:"useOpenshiftDefaultConfig,omitempty"`
}

// WavefrontCollectorStatus defines the observed state of WavefrontCollector
// +k8s:openapi-gen=true
type WavefrontCollectorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	Version string `json:"version,omitempty"`

	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WavefrontCollector is the Schema for the wavefrontcollectors API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type WavefrontCollector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WavefrontCollectorSpec   `json:"spec,omitempty"`
	Status WavefrontCollectorStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WavefrontCollectorList contains a list of WavefrontCollector
type WavefrontCollectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WavefrontCollector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WavefrontCollector{}, &WavefrontCollectorList{})
}
