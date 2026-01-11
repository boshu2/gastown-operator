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

// ConvoySpec defines the desired state of Convoy
type ConvoySpec struct {
	// Description is a human-readable description of this convoy
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Description string `json:"description"`

	// TrackedBeads is the list of bead IDs to track
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	TrackedBeads []string `json:"trackedBeads"`

	// NotifyOnComplete is the mail address for completion notification
	// +optional
	NotifyOnComplete string `json:"notifyOnComplete,omitempty"`
}

// ConvoyPhase represents the lifecycle phase of a Convoy
// +kubebuilder:validation:Enum=Pending;InProgress;Complete;Failed
type ConvoyPhase string

const (
	ConvoyPhasePending    ConvoyPhase = "Pending"
	ConvoyPhaseInProgress ConvoyPhase = "InProgress"
	ConvoyPhaseComplete   ConvoyPhase = "Complete"
	ConvoyPhaseFailed     ConvoyPhase = "Failed"
)

// ConvoyStatus defines the observed state of Convoy
type ConvoyStatus struct {
	// Phase is the current lifecycle phase
	// +kubebuilder:default=Pending
	Phase ConvoyPhase `json:"phase,omitempty"`

	// Progress is a human-readable progress indicator (e.g., "2/3")
	// +optional
	Progress string `json:"progress,omitempty"`

	// CompletedBeads is the list of beads that have been completed
	// +optional
	CompletedBeads []string `json:"completedBeads,omitempty"`

	// PendingBeads is the list of beads still in progress
	// +optional
	PendingBeads []string `json:"pendingBeads,omitempty"`

	// BeadsConvoyID is the convoy ID in the beads system (e.g., hq-cv-xxx)
	// +optional
	BeadsConvoyID string `json:"beadsConvoyID,omitempty"`

	// StartedAt is when the convoy started
	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`

	// CompletedAt is when the convoy completed
	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`

	// Conditions represent the current state of the Convoy resource
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Progress",type="string",JSONPath=".status.progress"
// +kubebuilder:printcolumn:name="BeadsID",type="string",JSONPath=".status.beadsConvoyID"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Convoy is the Schema for the convoys API.
// A Convoy tracks a batch of beads being worked on.
type Convoy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConvoySpec   `json:"spec,omitempty"`
	Status ConvoyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ConvoyList contains a list of Convoy
type ConvoyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Convoy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Convoy{}, &ConvoyList{})
}
