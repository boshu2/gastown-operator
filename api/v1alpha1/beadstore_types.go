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

// BeadStoreSpec defines the desired state of BeadStore.
// A BeadStore manages the configuration for a beads issue tracking database.
type BeadStoreSpec struct {
	// rigRef references the Rig this BeadStore is associated with.
	// +kubebuilder:validation:Required
	RigRef string `json:"rigRef"`

	// prefix is the issue ID prefix for this beadstore (e.g., "gt-", "he-").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z]+-$`
	Prefix string `json:"prefix"`

	// gitSecretRef references the Secret containing git credentials for syncing.
	// +optional
	GitSecretRef *SecretReference `json:"gitSecretRef,omitempty"`

	// syncInterval specifies how often to sync the beadstore with git.
	// +kubebuilder:default="5m"
	// +optional
	SyncInterval *metav1.Duration `json:"syncInterval,omitempty"`
}

// BeadStoreStatus defines the observed state of BeadStore.
type BeadStoreStatus struct {
	// phase indicates the current phase of the BeadStore.
	// +kubebuilder:validation:Enum=Pending;Synced;Error
	// +optional
	Phase string `json:"phase,omitempty"`

	// lastSyncTime is the timestamp of the last successful sync.
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// issueCount is the number of issues in this beadstore.
	// +optional
	IssueCount int32 `json:"issueCount"`

	// conditions represent the current state of the BeadStore resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Rig",type=string,JSONPath=`.spec.rigRef`
// +kubebuilder:printcolumn:name="Prefix",type=string,JSONPath=`.spec.prefix`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Issues",type=integer,JSONPath=`.status.issueCount`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// BeadStore is the Schema for the beadstores API.
// A BeadStore manages the configuration for a beads issue tracking database.
type BeadStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of BeadStore
	// +required
	Spec BeadStoreSpec `json:"spec"`

	// status defines the observed state of BeadStore
	// +optional
	Status BeadStoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BeadStoreList contains a list of BeadStore
type BeadStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BeadStore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BeadStore{}, &BeadStoreList{})
}
