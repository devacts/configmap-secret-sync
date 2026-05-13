/*
Copyright 2026.

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

// SecretMirrorSpec defines the desired state of SecretMirror
type SecretMirrorSpec struct {
	SourceConfigMap string `json:"sourceConfigMap"`
}

// SecretMirrorStatus defines the observed state of SecretMirror.
type SecretMirrorStatus struct {
	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// +optional
	SecretName string `json:"secretName,omitempty"`
	// conditions represent the current state of the SecretMirror resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Secret Name",type="string",JSONPath=".status.secretName",description="Secret Name"
// +kubebuilder:printcolumn:name="Secret Status",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status",description="Secret synced or not"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age of SecretMirror"

// SecretMirror is the Schema for the secretmirrors API
type SecretMirror struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of SecretMirror
	// +required
	Spec SecretMirrorSpec `json:"spec"`

	// status defines the observed state of SecretMirror
	// +optional
	Status SecretMirrorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecretMirrorList contains a list of SecretMirror
type SecretMirrorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretMirror `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecretMirror{}, &SecretMirrorList{})
}
