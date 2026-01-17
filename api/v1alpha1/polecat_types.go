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
	corev1 "k8s.io/api/core/v1"
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

// ExecutionMode determines where the polecat runs
// +kubebuilder:validation:Enum=local;kubernetes
type ExecutionMode string

const (
	// ExecutionModeLocal runs via gt CLI and tmux (default)
	ExecutionModeLocal ExecutionMode = "local"
	// ExecutionModeKubernetes runs as a Pod in the cluster
	ExecutionModeKubernetes ExecutionMode = "kubernetes"
)

// AgentType represents the coding agent to use
// +kubebuilder:validation:Enum=opencode;claude-code;aider;custom
type AgentType string

const (
	AgentTypeOpenCode   AgentType = "opencode"
	AgentTypeClaudeCode AgentType = "claude-code"
	AgentTypeAider      AgentType = "aider"
	AgentTypeCustom     AgentType = "custom"
)

// LLMProvider represents the LLM provider to use
// +kubebuilder:validation:Enum=litellm;anthropic;openai;ollama
type LLMProvider string

const (
	LLMProviderLiteLLM   LLMProvider = "litellm"
	LLMProviderAnthropic LLMProvider = "anthropic"
	LLMProviderOpenAI    LLMProvider = "openai"
	LLMProviderOllama    LLMProvider = "ollama"
)

// SecretKeyRef references a key in a Secret
type SecretKeyRef struct {
	// Name is the name of the secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Key is the key in the secret
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

// ModelProviderConfig configures the LLM provider endpoint
type ModelProviderConfig struct {
	// Endpoint is the API base URL (e.g., https://ai-gateway.example.com/v1)
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// APIKeySecretRef references the secret containing the API key
	// +optional
	APIKeySecretRef *SecretKeyRef `json:"apiKeySecretRef,omitempty"`
}

// AgentConfig configures the coding agent
type AgentConfig struct {
	// Provider is the LLM provider to use
	// +kubebuilder:default=litellm
	// +optional
	Provider LLMProvider `json:"provider,omitempty"`

	// Model is the model name/ID to use (e.g., "claude-sonnet-4", "devstral-123b")
	// +optional
	Model string `json:"model,omitempty"`

	// ModelProvider configures the LLM endpoint and credentials
	// +optional
	ModelProvider *ModelProviderConfig `json:"modelProvider,omitempty"`

	// Image overrides the default container image for the agent
	// +optional
	Image string `json:"image,omitempty"`

	// Command overrides the default entrypoint command
	// +optional
	Command []string `json:"command,omitempty"`

	// Args provides additional arguments to the agent command
	// +optional
	Args []string `json:"args,omitempty"`

	// ConfigMapRef references a ConfigMap containing agent configuration (e.g., opencode.json)
	// +optional
	ConfigMapRef *corev1.LocalObjectReference `json:"configMapRef,omitempty"`

	// Env provides additional environment variables
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`
}

// KubernetesSpec defines configuration for kubernetes execution mode
type KubernetesSpec struct {
	// GitRepository is the git repo URL to clone (SSH or HTTPS format)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^(git@[a-zA-Z0-9._-]+:|https?://[a-zA-Z0-9._-]+/)[a-zA-Z0-9._/-]+(\.git)?$`
	GitRepository string `json:"gitRepository"`

	// GitBranch is the branch to checkout
	// +kubebuilder:default=main
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9._/-]+$`
	// +optional
	GitBranch string `json:"gitBranch,omitempty"`

	// WorkBranch is the branch name to create for work (defaults to feature/<beadID>)
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9._/-]+$`
	// +optional
	WorkBranch string `json:"workBranch,omitempty"`

	// GitSecretRef references a Secret containing SSH key for git
	// +kubebuilder:validation:Required
	GitSecretRef SecretReference `json:"gitSecretRef"`

	// ClaudeCredsSecretRef references a Secret containing ~/.claude/ contents
	// +kubebuilder:validation:Required
	ClaudeCredsSecretRef SecretReference `json:"claudeCredsSecretRef"`

	// Image overrides the default agent container image
	// +optional
	Image string `json:"image,omitempty"`

	// Resources for the agent container
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// ActiveDeadlineSeconds is the max runtime before Pod is terminated
	// +kubebuilder:default=3600
	// +optional
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`
}

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

	// ExecutionMode determines where the polecat runs
	// +kubebuilder:default=local
	// +optional
	ExecutionMode ExecutionMode `json:"executionMode,omitempty"`

	// Kubernetes contains configuration for kubernetes execution mode
	// Required when executionMode is "kubernetes"
	// +optional
	Kubernetes *KubernetesSpec `json:"kubernetes,omitempty"`

	// Agent is the coding agent type to use
	// +kubebuilder:default=opencode
	// +optional
	Agent AgentType `json:"agent,omitempty"`

	// AgentConfig provides configuration for the coding agent
	// +optional
	AgentConfig *AgentConfig `json:"agentConfig,omitempty"`

	// Resources defines compute resources for the polecat pod
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// TTLSecondsAfterFinished limits how long a completed polecat persists
	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`

	// MaxIdleSeconds terminates polecat if idle for this duration
	// +optional
	MaxIdleSeconds *int32 `json:"maxIdleSeconds,omitempty"`
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

	// TmuxSession is the tmux session name (local mode)
	// +optional
	TmuxSession string `json:"tmuxSession,omitempty"`

	// SessionActive indicates if the tmux session is running (local mode)
	SessionActive bool `json:"sessionActive,omitempty"`

	// PodName is the name of the Pod running the agent (kubernetes mode)
	// +optional
	PodName string `json:"podName,omitempty"`

	// LastActivity is when the polecat last showed activity
	// +optional
	LastActivity *metav1.Time `json:"lastActivity,omitempty"`

	// CleanupStatus indicates the git workspace state
	// +optional
	CleanupStatus CleanupStatus `json:"cleanupStatus,omitempty"`

	// Agent is the agent type currently running
	// +optional
	Agent AgentType `json:"agent,omitempty"`

	// AgentImage is the container image being used
	// +optional
	AgentImage string `json:"agentImage,omitempty"`

	// AgentModel is the LLM model being used
	// +optional
	AgentModel string `json:"agentModel,omitempty"`

	// Conditions represent the current state of the Polecat resource
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Rig",type="string",JSONPath=".spec.rig"
// +kubebuilder:printcolumn:name="Mode",type="string",JSONPath=".spec.executionMode"
// +kubebuilder:printcolumn:name="Agent",type="string",JSONPath=".spec.agent"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Bead",type="string",JSONPath=".status.assignedBead"
// +kubebuilder:printcolumn:name="Model",type="string",JSONPath=".status.agentModel",priority=1
// +kubebuilder:printcolumn:name="Pod",type="string",JSONPath=".status.podName",priority=1
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
