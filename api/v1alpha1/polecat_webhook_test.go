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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPolecatCustomValidator_ValidateCreate(t *testing.T) {
	validator := &PolecatCustomValidator{}
	ctx := context.Background()

	tests := []struct {
		name        string
		polecat     *Polecat
		wantErr     bool
		errContains string
		wantWarning bool
	}{
		{
			name: "valid local polecat",
			polecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  PolecatDesiredIdle,
					ExecutionMode: ExecutionModeLocal,
				},
			},
			wantErr: false,
		},
		{
			name: "valid kubernetes polecat",
			polecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  PolecatDesiredWorking,
					ExecutionMode: ExecutionModeKubernetes,
					Kubernetes: &KubernetesSpec{
						GitRepository:        "git@github.com:org/repo.git",
						GitBranch:            "main",
						GitSecretRef:         SecretReference{Name: "git-secret"},
						ClaudeCredsSecretRef: &SecretReference{Name: "claude-creds"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing rig",
			polecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					DesiredState:  PolecatDesiredIdle,
					ExecutionMode: ExecutionModeLocal,
				},
			},
			wantErr:     true,
			errContains: "spec.rig: is required",
		},
		{
			name: "kubernetes mode without spec",
			polecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  PolecatDesiredWorking,
					ExecutionMode: ExecutionModeKubernetes,
				},
			},
			wantErr:     true,
			errContains: "spec.kubernetes: is required when executionMode is 'kubernetes'",
		},
		{
			name: "kubernetes mode missing git secret",
			polecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  PolecatDesiredWorking,
					ExecutionMode: ExecutionModeKubernetes,
					Kubernetes: &KubernetesSpec{
						GitRepository:        "git@github.com:org/repo.git",
						ClaudeCredsSecretRef: &SecretReference{Name: "claude-creds"},
					},
				},
			},
			wantErr:     true,
			errContains: "spec.kubernetes.gitSecretRef.name: is required",
		},
		{
			name: "kubernetes mode missing claude creds",
			polecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  PolecatDesiredWorking,
					ExecutionMode: ExecutionModeKubernetes,
					Kubernetes: &KubernetesSpec{
						GitRepository: "git@github.com:org/repo.git",
						GitSecretRef:  SecretReference{Name: "git-secret"},
					},
				},
			},
			wantErr:     true,
			errContains: "spec.kubernetes: either claudeCredsSecretRef or apiKeySecretRef is required",
		},
		{
			name: "high resource usage warning",
			polecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  PolecatDesiredWorking,
					ExecutionMode: ExecutionModeLocal,
					Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("8"),
							corev1.ResourceMemory: resource.MustParse("16Gi"),
						},
					},
				},
			},
			wantErr:     false,
			wantWarning: true, // Should warn about high resources
		},
		{
			name: "requests exceed limits",
			polecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  PolecatDesiredWorking,
					ExecutionMode: ExecutionModeLocal,
					Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("4"),
							corev1.ResourceMemory: resource.MustParse("8Gi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("2"),
							corev1.ResourceMemory: resource.MustParse("4Gi"),
						},
					},
				},
			},
			wantErr:     true,
			errContains: "exceeds limit",
		},
		{
			name: "long active deadline warning",
			polecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  PolecatDesiredWorking,
					ExecutionMode: ExecutionModeKubernetes,
					Kubernetes: &KubernetesSpec{
						GitRepository:         "git@github.com:org/repo.git",
						GitSecretRef:          SecretReference{Name: "git-secret"},
						ClaudeCredsSecretRef:  &SecretReference{Name: "claude-creds"},
						ActiveDeadlineSeconds: int64Ptr(14400), // 4 hours
					},
				},
			},
			wantErr:     false,
			wantWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, err := validator.ValidateCreate(ctx, tt.polecat)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
			if tt.wantWarning {
				assert.NotEmpty(t, warnings)
			}
		})
	}
}

func TestPolecatCustomValidator_ValidateUpdate(t *testing.T) {
	validator := &PolecatCustomValidator{}
	ctx := context.Background()

	oldPolecat := &Polecat{
		ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
		Spec: PolecatSpec{
			Rig:           "test-rig",
			DesiredState:  PolecatDesiredIdle,
			ExecutionMode: ExecutionModeLocal,
		},
	}

	tests := []struct {
		name        string
		newPolecat  *Polecat
		wantErr     bool
		errContains string
	}{
		{
			name: "valid update - change desired state",
			newPolecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  PolecatDesiredWorking,
					ExecutionMode: ExecutionModeLocal,
				},
			},
			wantErr: false,
		},
		{
			name: "valid update - add bead ID",
			newPolecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  PolecatDesiredWorking,
					ExecutionMode: ExecutionModeLocal,
					BeadID:        "ap-1234",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid update - change rig (immutable)",
			newPolecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					Rig:           "different-rig",
					DesiredState:  PolecatDesiredIdle,
					ExecutionMode: ExecutionModeLocal,
				},
			},
			wantErr:     true,
			errContains: "spec.rig is immutable",
		},
		{
			name: "invalid update - change execution mode (immutable)",
			newPolecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  PolecatDesiredIdle,
					ExecutionMode: ExecutionModeKubernetes,
					Kubernetes: &KubernetesSpec{
						GitRepository:        "git@github.com:org/repo.git",
						GitSecretRef:         SecretReference{Name: "git-secret"},
						ClaudeCredsSecretRef: &SecretReference{Name: "claude-creds"},
					},
				},
			},
			wantErr:     true,
			errContains: "spec.executionMode is immutable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validator.ValidateUpdate(ctx, oldPolecat, tt.newPolecat)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPolecatCustomValidator_ValidateDelete(t *testing.T) {
	validator := &PolecatCustomValidator{}
	ctx := context.Background()

	polecat := &Polecat{
		ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
		Spec: PolecatSpec{
			Rig:           "test-rig",
			DesiredState:  PolecatDesiredIdle,
			ExecutionMode: ExecutionModeLocal,
		},
	}

	warnings, err := validator.ValidateDelete(ctx, polecat)
	require.NoError(t, err)
	assert.Nil(t, warnings)
}

// Note: WrongType tests removed - generics enforce type safety at compile time

func TestPolecatCustomDefaulter_Default(t *testing.T) {
	defaulter := &PolecatCustomDefaulter{}
	ctx := context.Background()

	tests := []struct {
		name          string
		polecat       *Polecat
		checkDefaults func(t *testing.T, p *Polecat)
	}{
		{
			name: "sets default execution mode",
			polecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					Rig: "test-rig",
				},
			},
			checkDefaults: func(t *testing.T, p *Polecat) {
				assert.Equal(t, ExecutionModeLocal, p.Spec.ExecutionMode)
			},
		},
		{
			name: "sets default agent type",
			polecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					Rig: "test-rig",
				},
			},
			checkDefaults: func(t *testing.T, p *Polecat) {
				assert.Equal(t, AgentTypeOpenCode, p.Spec.Agent)
			},
		},
		{
			name: "sets default desired state",
			polecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					Rig: "test-rig",
				},
			},
			checkDefaults: func(t *testing.T, p *Polecat) {
				assert.Equal(t, PolecatDesiredIdle, p.Spec.DesiredState)
			},
		},
		{
			name: "sets kubernetes defaults",
			polecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					Rig:           "test-rig",
					ExecutionMode: ExecutionModeKubernetes,
					Kubernetes: &KubernetesSpec{
						GitRepository:        "git@github.com:org/repo.git",
						GitSecretRef:         SecretReference{Name: "git-secret"},
						ClaudeCredsSecretRef: &SecretReference{Name: "claude-creds"},
					},
				},
			},
			checkDefaults: func(t *testing.T, p *Polecat) {
				assert.Equal(t, "main", p.Spec.Kubernetes.GitBranch)
				assert.NotNil(t, p.Spec.Kubernetes.ActiveDeadlineSeconds)
				assert.Equal(t, int64(3600), *p.Spec.Kubernetes.ActiveDeadlineSeconds)
			},
		},
		{
			name: "sets agent config defaults",
			polecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					Rig: "test-rig",
				},
			},
			checkDefaults: func(t *testing.T, p *Polecat) {
				require.NotNil(t, p.Spec.AgentConfig)
				assert.Equal(t, LLMProviderLiteLLM, p.Spec.AgentConfig.Provider)
			},
		},
		{
			name: "preserves existing values",
			polecat: &Polecat{
				ObjectMeta: metav1.ObjectMeta{Name: "test-polecat"},
				Spec: PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  PolecatDesiredWorking,
					ExecutionMode: ExecutionModeKubernetes,
					Agent:         AgentTypeClaudeCode,
					Kubernetes: &KubernetesSpec{
						GitRepository:        "git@github.com:org/repo.git",
						GitBranch:            "develop",
						GitSecretRef:         SecretReference{Name: "git-secret"},
						ClaudeCredsSecretRef: &SecretReference{Name: "claude-creds"},
					},
				},
			},
			checkDefaults: func(t *testing.T, p *Polecat) {
				assert.Equal(t, PolecatDesiredWorking, p.Spec.DesiredState)
				assert.Equal(t, AgentTypeClaudeCode, p.Spec.Agent)
				assert.Equal(t, "develop", p.Spec.Kubernetes.GitBranch)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := defaulter.Default(ctx, tt.polecat)
			require.NoError(t, err)
			tt.checkDefaults(t, tt.polecat)
		})
	}
}

// Note: WrongType tests removed - generics enforce type safety at compile time

func TestValidateKubernetesSpec(t *testing.T) {
	tests := []struct {
		name        string
		spec        *KubernetesSpec
		wantErrs    int
		errContains []string
	}{
		{
			name: "valid spec",
			spec: &KubernetesSpec{
				GitRepository:        "git@github.com:org/repo.git",
				GitSecretRef:         SecretReference{Name: "git-secret"},
				ClaudeCredsSecretRef: &SecretReference{Name: "claude-creds"},
			},
			wantErrs: 0,
		},
		{
			name: "missing git repository",
			spec: &KubernetesSpec{
				GitSecretRef:         SecretReference{Name: "git-secret"},
				ClaudeCredsSecretRef: &SecretReference{Name: "claude-creds"},
			},
			wantErrs:    1,
			errContains: []string{"spec.kubernetes.gitRepository: is required"},
		},
		{
			name:     "missing all required fields",
			spec:     &KubernetesSpec{},
			wantErrs: 3,
			errContains: []string{
				"spec.kubernetes.gitRepository: is required",
				"spec.kubernetes.gitSecretRef.name: is required",
				"spec.kubernetes: either claudeCredsSecretRef or apiKeySecretRef is required",
			},
		},
		{
			name: "invalid active deadline",
			spec: &KubernetesSpec{
				GitRepository:         "git@github.com:org/repo.git",
				GitSecretRef:          SecretReference{Name: "git-secret"},
				ClaudeCredsSecretRef:  &SecretReference{Name: "claude-creds"},
				ActiveDeadlineSeconds: int64Ptr(-1),
			},
			wantErrs:    1,
			errContains: []string{"spec.kubernetes.activeDeadlineSeconds: must be positive"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateKubernetesSpec(tt.spec)
			assert.Len(t, errs, tt.wantErrs)
			for _, expected := range tt.errContains {
				found := false
				for _, err := range errs {
					if err == expected {
						found = true
						break
					}
				}
				assert.True(t, found, "expected error %q not found in %v", expected, errs)
			}
		})
	}
}

func TestValidateResources(t *testing.T) {
	tests := []struct {
		name         string
		resources    *corev1.ResourceRequirements
		wantErrs     int
		wantWarnings int
	}{
		{
			name: "valid resources - requests below limits",
			resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
			},
			wantErrs:     0,
			wantWarnings: 0,
		},
		{
			name: "cpu request exceeds limit",
			resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("4"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("2"),
				},
			},
			wantErrs:     1,
			wantWarnings: 0,
		},
		{
			name: "memory request exceeds limit",
			resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("8Gi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
			},
			wantErrs:     1,
			wantWarnings: 0,
		},
		{
			name: "high cpu warning",
			resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("8"),
				},
			},
			wantErrs:     0,
			wantWarnings: 1,
		},
		{
			name: "high memory warning",
			resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("16Gi"),
				},
			},
			wantErrs:     0,
			wantWarnings: 1,
		},
		{
			name: "both cpu and memory warnings",
			resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("8"),
					corev1.ResourceMemory: resource.MustParse("16Gi"),
				},
			},
			wantErrs:     0,
			wantWarnings: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs, warnings := validateResources(tt.resources)
			assert.Len(t, errs, tt.wantErrs)
			assert.Len(t, warnings, tt.wantWarnings)
		})
	}
}

// Helper function for int64 pointers
func int64Ptr(i int64) *int64 {
	return &i
}
