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

// WitnessSpec defines the desired state of Witness (Sentinel in Olympian API).
// A Witness monitors the health of Polecats in a Rig and escalates issues.
type WitnessSpec struct {
	// rigRef references the Rig (Forge) to monitor.
	// +kubebuilder:validation:Required
	RigRef string `json:"rigRef"`

	// healthCheckInterval specifies how often to check polecat health.
	// +kubebuilder:default="30s"
	// +optional
	HealthCheckInterval *metav1.Duration `json:"healthCheckInterval,omitempty"`

	// stuckThreshold specifies how long a polecat can be idle before being considered stuck.
	// +kubebuilder:default="15m"
	// +optional
	StuckThreshold *metav1.Duration `json:"stuckThreshold,omitempty"`

	// escalationTarget specifies where to send alerts (e.g., "mayor", "slack", "email").
	// +kubebuilder:default="mayor"
	// +optional
	EscalationTarget string `json:"escalationTarget,omitempty"`
}

// WitnessStatus defines the observed state of Witness.
type WitnessStatus struct {
	// phase indicates the current phase of the Witness.
	// +kubebuilder:validation:Enum=Pending;Active;Degraded
	// +optional
	Phase string `json:"phase,omitempty"`

	// lastCheckTime is the timestamp of the last health check.
	// +optional
	LastCheckTime *metav1.Time `json:"lastCheckTime,omitempty"`

	// polecatsSummary provides a summary of polecat states in the monitored rig.
	// +optional
	PolecatsSummary PolecatsSummary `json:"polecatsSummary,omitempty"`

	// conditions represent the current state of the Witness resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// PolecatsSummary contains aggregated polecat health information.
type PolecatsSummary struct {
	// total is the total number of polecats in the rig.
	Total int32 `json:"total"`

	// running is the number of actively running polecats.
	Running int32 `json:"running"`

	// succeeded is the number of successfully completed polecats.
	Succeeded int32 `json:"succeeded"`

	// failed is the number of failed polecats.
	Failed int32 `json:"failed"`

	// stuck is the number of polecats that appear stuck (no progress).
	Stuck int32 `json:"stuck"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Rig",type=string,JSONPath=`.spec.rigRef`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Running",type=integer,JSONPath=`.status.polecatsSummary.running`
// +kubebuilder:printcolumn:name="Stuck",type=integer,JSONPath=`.status.polecatsSummary.stuck`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Witness is the Schema for the witnesses API.
// Also known as Sentinel in the Olympian API naming convention.
// A Witness monitors Polecat health in a Rig and escalates anomalies.
type Witness struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of Witness
	// +required
	Spec WitnessSpec `json:"spec"`

	// status defines the observed state of Witness
	// +optional
	Status WitnessStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WitnessList contains a list of Witness
type WitnessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Witness `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Witness{}, &WitnessList{})
}
