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

	// WavefrontProxyEnabled is whether to enable the wavefront proxy.
	WavefrontProxyEnabled bool `json:"wavefrontProxyEnabled,required"`

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

	// DataExport has configuration for proxy to export metric data
	DataExport DataExport `json:"dataExport,omitempty"`
}

type DataExport struct {
	// Proxy ConfigMap name. Leave blank to use defaults
	ProxyConfig string `json:"proxyConfig,omitempty"`
}

type Metrics struct {
	// Collector ConfigMap name. Leave blank to use defaults
	CollectorConfig string `json:"collectorConfig,omitempty"`

	// Cluster is for resource configuration for the cluster collector
	Cluster Collector `json:"cluster,omitempty"`

	// Node is for resource configuration for the node collector
	Node Collector `json:"node,omitempty"`
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
