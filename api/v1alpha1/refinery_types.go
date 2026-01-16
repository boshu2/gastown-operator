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

// RefinerySpec defines the desired state of Refinery (Crucible in Olympian API).
// A Refinery processes merge queues for a Rig, sequentially rebasing and merging
// polecat branches after validation.
type RefinerySpec struct {
	// rigRef references the Rig (Forge) to process merges for.
	// +kubebuilder:validation:Required
	RigRef string `json:"rigRef"`

	// targetBranch is the branch to merge into (e.g., "main").
	// +kubebuilder:default="main"
	// +optional
	TargetBranch string `json:"targetBranch,omitempty"`

	// testCommand is the command to run after rebase to validate the branch.
	// If empty, no tests are run.
	// +optional
	TestCommand string `json:"testCommand,omitempty"`

	// parallelism controls how many merges can be processed concurrently.
	// Default is 1 (sequential processing).
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// +optional
	Parallelism int32 `json:"parallelism,omitempty"`

	// gitSecretRef references the Secret containing git credentials.
	// +optional
	GitSecretRef *SecretReference `json:"gitSecretRef,omitempty"`
}

// SecretReference contains information to locate a secret.
type SecretReference struct {
	// name is the name of the secret.
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// RefineryStatus defines the observed state of Refinery.
type RefineryStatus struct {
	// phase indicates the current phase of the Refinery.
	// +kubebuilder:validation:Enum=Idle;Processing;Error
	// +optional
	Phase string `json:"phase,omitempty"`

	// queueLength is the number of branches waiting to be merged.
	// +optional
	QueueLength int32 `json:"queueLength"`

	// currentMerge is the branch currently being processed.
	// +optional
	CurrentMerge string `json:"currentMerge,omitempty"`

	// lastMergeTime is the timestamp of the last successful merge.
	// +optional
	LastMergeTime *metav1.Time `json:"lastMergeTime,omitempty"`

	// mergesSummary provides aggregate merge statistics.
	// +optional
	MergesSummary MergesSummary `json:"mergesSummary,omitempty"`

	// conditions represent the current state of the Refinery resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// MergesSummary contains aggregate merge statistics.
type MergesSummary struct {
	// total is the total number of merges attempted.
	Total int32 `json:"total"`

	// succeeded is the number of successful merges.
	Succeeded int32 `json:"succeeded"`

	// failed is the number of failed merges (conflicts, test failures).
	Failed int32 `json:"failed"`

	// pending is the number of branches waiting in queue.
	Pending int32 `json:"pending"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Rig",type=string,JSONPath=`.spec.rigRef`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Queue",type=integer,JSONPath=`.status.queueLength`
// +kubebuilder:printcolumn:name="Current",type=string,JSONPath=`.status.currentMerge`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Refinery is the Schema for the refineries API.
// Also known as Crucible in the Olympian API naming convention.
// A Refinery processes merge queues, sequentially rebasing and merging polecat branches.
type Refinery struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of Refinery
	// +required
	Spec RefinerySpec `json:"spec"`

	// status defines the observed state of Refinery
	// +optional
	Status RefineryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RefineryList contains a list of Refinery
type RefineryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Refinery `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Refinery{}, &RefineryList{})
}
