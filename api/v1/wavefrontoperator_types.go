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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WavefrontOperatorSpec defines the desired state of WavefrontOperator
type WavefrontOperatorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ClusterName is a unique name for the Kubernetes cluster to be
	// identified via a metric tag on Wavefront.
	WavefrontUrl string `json:"wavefrontUrl,required"`

	// Wavefront API Token.
	WavefrontToken string `json:"wavefrontToken,required"`
}

// WavefrontOperatorStatus defines the observed state of WavefrontOperator
type WavefrontOperatorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// WavefrontOperator is the Schema for the wavefrontoperators API
type WavefrontOperator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WavefrontOperatorSpec   `json:"spec,omitempty"`
	Status WavefrontOperatorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WavefrontOperatorList contains a list of WavefrontOperator
type WavefrontOperatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WavefrontOperator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WavefrontOperator{}, &WavefrontOperatorList{})
}
