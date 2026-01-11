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

// PolecatDesiredState represents the desired lifecycle state
// +kubebuilder:validation:Enum=Idle;Working;Terminated
type PolecatDesiredState string

const (
	PolecatDesiredIdle       PolecatDesiredState = "Idle"
	PolecatDesiredWorking    PolecatDesiredState = "Working"
	PolecatDesiredTerminated PolecatDesiredState = "Terminated"
)

// PolecatSpec defines the desired state of Polecat
type PolecatSpec struct {
	// Rig is the name of the rig this polecat belongs to
	// +kubebuilder:validation:Required
	Rig string `json:"rig"`

	// DesiredState is the target lifecycle state
	// +kubebuilder:validation:Required
	// +kubebuilder:default=Idle
	DesiredState PolecatDesiredState `json:"desiredState"`

	// BeadID is the bead to hook (triggers gt sling if set)
	// +optional
	BeadID string `json:"beadID,omitempty"`
}

// PolecatPhase represents the observed lifecycle phase
// +kubebuilder:validation:Enum=Idle;Working;Done;Stuck;Terminated
type PolecatPhase string

const (
	PolecatPhaseIdle       PolecatPhase = "Idle"
	PolecatPhaseWorking    PolecatPhase = "Working"
	PolecatPhaseDone       PolecatPhase = "Done"
	PolecatPhaseStuck      PolecatPhase = "Stuck"
	PolecatPhaseTerminated PolecatPhase = "Terminated"
)

// CleanupStatus represents the git workspace state
// +kubebuilder:validation:Enum=clean;has_uncommitted;has_unpushed;unknown
type CleanupStatus string

const (
	CleanupStatusClean       CleanupStatus = "clean"
	CleanupStatusUncommitted CleanupStatus = "has_uncommitted"
	CleanupStatusUnpushed    CleanupStatus = "has_unpushed"
	CleanupStatusUnknown     CleanupStatus = "unknown"
)

// PolecatStatus defines the observed state of Polecat
type PolecatStatus struct {
	// Phase is the current lifecycle phase
	// +kubebuilder:default=Idle
	Phase PolecatPhase `json:"phase,omitempty"`

	// AssignedBead is the bead currently hooked to this polecat
	// +optional
	AssignedBead string `json:"assignedBead,omitempty"`

	// Branch is the git branch the polecat is working on
	// +optional
	Branch string `json:"branch,omitempty"`

	// WorktreePath is the filesystem path to the worktree
	// +optional
	WorktreePath string `json:"worktreePath,omitempty"`

	// TmuxSession is the tmux session name
	// +optional
	TmuxSession string `json:"tmuxSession,omitempty"`

	// SessionActive indicates if the tmux session is running
	SessionActive bool `json:"sessionActive,omitempty"`

	// LastActivity is when the polecat last showed activity
	// +optional
	LastActivity *metav1.Time `json:"lastActivity,omitempty"`

	// CleanupStatus indicates the git workspace state
	// +optional
	CleanupStatus CleanupStatus `json:"cleanupStatus,omitempty"`

	// Conditions represent the current state of the Polecat resource
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Rig",type="string",JSONPath=".spec.rig"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Bead",type="string",JSONPath=".status.assignedBead"
// +kubebuilder:printcolumn:name="Session",type="boolean",JSONPath=".status.sessionActive"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Polecat is the Schema for the polecats API.
// A Polecat is an autonomous worker agent within a Rig.
type Polecat struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PolecatSpec   `json:"spec,omitempty"`
	Status PolecatStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PolecatList contains a list of Polecat
type PolecatList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Polecat `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Polecat{}, &PolecatList{})
}
