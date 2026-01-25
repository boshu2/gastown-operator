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

// RigSpec defines the desired state of Rig
type RigSpec struct {
	// GitURL is the remote repository URL
	// +kubebuilder:validation:Required
	GitURL string `json:"gitURL"`

	// BeadsPrefix is the prefix for beads issues (e.g., "ap" for ap-*)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z]{2,10}$`
	BeadsPrefix string `json:"beadsPrefix"`

	// Settings for the rig
	// +optional
	Settings RigSettings `json:"settings,omitempty"`
}

// RigSettings contains optional configuration for a rig
type RigSettings struct {
	// NamepoolTheme is the naming theme for polecats (e.g., "fury-road")
	// +optional
	NamepoolTheme string `json:"namepoolTheme,omitempty"`

	// MaxPolecats is the maximum number of concurrent polecats
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=8
	// +optional
	MaxPolecats int `json:"maxPolecats,omitempty"`
}

// RigPhase represents the current lifecycle phase of a Rig
// +kubebuilder:validation:Enum=Initializing;Ready;Degraded
type RigPhase string

const (
	RigPhaseInitializing RigPhase = "Initializing"
	RigPhaseReady        RigPhase = "Ready"
	RigPhaseDegraded     RigPhase = "Degraded"
)

// RigStatus defines the observed state of Rig
type RigStatus struct {
	// Phase is the current lifecycle phase of the Rig
	// +kubebuilder:default=Initializing
	Phase RigPhase `json:"phase,omitempty"`

	// PolecatCount is the current number of polecats in this rig
	PolecatCount int `json:"polecatCount,omitempty"`

	// ActiveConvoys is the number of convoys currently in progress
	ActiveConvoys int `json:"activeConvoys,omitempty"`

	// WitnessCreated indicates if the Witness CR has been auto-provisioned
	// +optional
	WitnessCreated bool `json:"witnessCreated,omitempty"`

	// RefineryCreated indicates if the Refinery CR has been auto-provisioned
	// +optional
	RefineryCreated bool `json:"refineryCreated,omitempty"`

	// ChildNamespace is the namespace where child resources (Witness, Refinery) are created
	// Defaults to the operator namespace (gastown-system)
	// +optional
	ChildNamespace string `json:"childNamespace,omitempty"`

	// Conditions represent the current state of the Rig resource
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Polecats",type="integer",JSONPath=".status.polecatCount"
// +kubebuilder:printcolumn:name="Convoys",type="integer",JSONPath=".status.activeConvoys"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Rig is the Schema for the rigs API.
// A Rig represents a project workspace containing crew and polecats.
type Rig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RigSpec   `json:"spec,omitempty"`
	Status RigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RigList contains a list of Rig
type RigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Rig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Rig{}, &RigList{})
}
