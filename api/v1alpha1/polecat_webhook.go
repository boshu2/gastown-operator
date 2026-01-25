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
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var polecatlog = logf.Log.WithName("polecat-resource")

// SetupPolecatWebhookWithManager registers the Polecat webhooks with the manager.
func SetupPolecatWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &Polecat{}).
		WithValidator(&PolecatCustomValidator{}).
		WithDefaulter(&PolecatCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-gastown-gastown-io-v1alpha1-polecat,mutating=false,failurePolicy=fail,sideEffects=None,groups=gastown.gastown.io,resources=polecats,verbs=create;update,versions=v1alpha1,name=vpolecat.kb.io,admissionReviewVersions=v1

// PolecatCustomValidator implements admission.Validator[*Polecat] for Polecat.
type PolecatCustomValidator struct{}

var _ admission.Validator[*Polecat] = &PolecatCustomValidator{}

// ValidateCreate implements admission.Validator.
func (v *PolecatCustomValidator) ValidateCreate(ctx context.Context, polecat *Polecat) (admission.Warnings, error) {
	polecatlog.Info("validate create", "name", polecat.Name)

	return v.validatePolecat(polecat)
}

// ValidateUpdate implements admission.Validator.
func (v *PolecatCustomValidator) ValidateUpdate(ctx context.Context, oldPolecat, polecat *Polecat) (admission.Warnings, error) {
	polecatlog.Info("validate update", "name", polecat.Name)

	// Validate immutable fields
	if oldPolecat.Spec.Rig != polecat.Spec.Rig {
		return nil, fmt.Errorf("spec.rig is immutable: cannot change from %q to %q",
			oldPolecat.Spec.Rig, polecat.Spec.Rig)
	}

	if oldPolecat.Spec.ExecutionMode != polecat.Spec.ExecutionMode {
		return nil, fmt.Errorf("spec.executionMode is immutable: cannot change from %q to %q",
			oldPolecat.Spec.ExecutionMode, polecat.Spec.ExecutionMode)
	}

	return v.validatePolecat(polecat)
}

// ValidateDelete implements admission.Validator.
func (v *PolecatCustomValidator) ValidateDelete(ctx context.Context, polecat *Polecat) (admission.Warnings, error) {
	polecatlog.Info("validate delete", "name", polecat.Name)

	// No validation on delete
	return nil, nil
}

// validatePolecat performs validation common to create and update.
func (v *PolecatCustomValidator) validatePolecat(polecat *Polecat) (admission.Warnings, error) {
	var allErrs []string
	var warnings admission.Warnings

	// Validate rig name
	if polecat.Spec.Rig == "" {
		allErrs = append(allErrs, "spec.rig: is required")
	}

	// Validate kubernetes spec when executionMode is kubernetes
	if polecat.Spec.ExecutionMode == ExecutionModeKubernetes {
		if polecat.Spec.Kubernetes == nil {
			allErrs = append(allErrs, "spec.kubernetes: is required when executionMode is 'kubernetes'")
		} else {
			errs := validateKubernetesSpec(polecat.Spec.Kubernetes)
			allErrs = append(allErrs, errs...)
		}
	}

	// Validate resources if specified
	if polecat.Spec.Resources != nil {
		errs, warns := validateResources(polecat.Spec.Resources)
		allErrs = append(allErrs, errs...)
		warnings = append(warnings, warns...)
	}

	// Validate kubernetes.resources if specified
	if polecat.Spec.Kubernetes != nil && polecat.Spec.Kubernetes.Resources != nil {
		errs, warns := validateResources(polecat.Spec.Kubernetes.Resources)
		allErrs = append(allErrs, errs...)
		warnings = append(warnings, warns...)
	}

	// Validate TTL
	if polecat.Spec.TTLSecondsAfterFinished != nil && *polecat.Spec.TTLSecondsAfterFinished < 0 {
		allErrs = append(allErrs, "spec.ttlSecondsAfterFinished: must be non-negative")
	}

	// Validate MaxIdleSeconds
	if polecat.Spec.MaxIdleSeconds != nil && *polecat.Spec.MaxIdleSeconds < 0 {
		allErrs = append(allErrs, "spec.maxIdleSeconds: must be non-negative")
	}

	// Warning for long-running configurations
	if polecat.Spec.Kubernetes != nil &&
		polecat.Spec.Kubernetes.ActiveDeadlineSeconds != nil &&
		*polecat.Spec.Kubernetes.ActiveDeadlineSeconds > 7200 {
		warnings = append(warnings, "spec.kubernetes.activeDeadlineSeconds > 7200 (2 hours) may be expensive")
	}

	if len(allErrs) > 0 {
		return warnings, fmt.Errorf("validation failed: %s", strings.Join(allErrs, "; "))
	}

	return warnings, nil
}

// validateKubernetesSpec validates the kubernetes execution spec.
func validateKubernetesSpec(k *KubernetesSpec) []string {
	var errs []string

	// GitRepository is required (validated by CRD, but double-check)
	if k.GitRepository == "" {
		errs = append(errs, "spec.kubernetes.gitRepository: is required")
	}

	// GitSecretRef is required
	if k.GitSecretRef.Name == "" {
		errs = append(errs, "spec.kubernetes.gitSecretRef.name: is required")
	}

	// Either ClaudeCredsSecretRef or ApiKeySecretRef is required for authentication
	hasOAuth := k.ClaudeCredsSecretRef != nil && k.ClaudeCredsSecretRef.Name != ""
	hasAPIKey := k.ApiKeySecretRef != nil && k.ApiKeySecretRef.Name != ""
	if !hasOAuth && !hasAPIKey {
		errs = append(errs, "spec.kubernetes: either claudeCredsSecretRef or apiKeySecretRef is required")
	}

	// Validate ActiveDeadlineSeconds
	if k.ActiveDeadlineSeconds != nil && *k.ActiveDeadlineSeconds <= 0 {
		errs = append(errs, "spec.kubernetes.activeDeadlineSeconds: must be positive")
	}

	return errs
}

// validateResources validates that resource requests don't exceed limits.
//
//nolint:gocyclo // Complexity from parallel CPU/memory validation paths; extracting would reduce clarity
func validateResources(resources *corev1.ResourceRequirements) ([]string, admission.Warnings) {
	var errs []string
	var warnings admission.Warnings

	// Check CPU: requests <= limits
	if resources.Requests != nil && resources.Limits != nil {
		reqCPU := resources.Requests.Cpu()
		limCPU := resources.Limits.Cpu()
		if reqCPU != nil && limCPU != nil && !reqCPU.IsZero() && !limCPU.IsZero() {
			if reqCPU.Cmp(*limCPU) > 0 {
				errs = append(errs, fmt.Sprintf("cpu request (%s) exceeds limit (%s)",
					reqCPU.String(), limCPU.String()))
			}
		}

		// Check Memory: requests <= limits
		reqMem := resources.Requests.Memory()
		limMem := resources.Limits.Memory()
		if reqMem != nil && limMem != nil && !reqMem.IsZero() && !limMem.IsZero() {
			if reqMem.Cmp(*limMem) > 0 {
				errs = append(errs, fmt.Sprintf("memory request (%s) exceeds limit (%s)",
					reqMem.String(), limMem.String()))
			}
		}
	}

	// Warn about high resource usage
	if resources.Requests != nil {
		if cpu := resources.Requests.Cpu(); cpu != nil {
			fourCores := resource.MustParse("4")
			if cpu.Cmp(fourCores) > 0 {
				warnings = append(warnings, fmt.Sprintf("cpu request (%s) is high, consider reducing", cpu.String()))
			}
		}
		if mem := resources.Requests.Memory(); mem != nil {
			eightGi := resource.MustParse("8Gi")
			if mem.Cmp(eightGi) > 0 {
				warnings = append(warnings, fmt.Sprintf("memory request (%s) is high, consider reducing", mem.String()))
			}
		}
	}

	return errs, warnings
}

// +kubebuilder:webhook:path=/mutate-gastown-gastown-io-v1alpha1-polecat,mutating=true,failurePolicy=fail,sideEffects=None,groups=gastown.gastown.io,resources=polecats,verbs=create;update,versions=v1alpha1,name=mpolecat.kb.io,admissionReviewVersions=v1

// PolecatCustomDefaulter implements admission.Defaulter[*Polecat] for Polecat.
type PolecatCustomDefaulter struct{}

var _ admission.Defaulter[*Polecat] = &PolecatCustomDefaulter{}

// Default implements admission.Defaulter.
func (d *PolecatCustomDefaulter) Default(ctx context.Context, polecat *Polecat) error {
	polecatlog.Info("default", "name", polecat.Name)

	// Ensure labels map exists
	if polecat.Labels == nil {
		polecat.Labels = make(map[string]string)
	}

	// Set gastown.io labels for discovery by other controllers (e.g., Refinery)
	if polecat.Spec.Rig != "" {
		polecat.Labels["gastown.io/rig"] = polecat.Spec.Rig
	}
	if polecat.Spec.BeadID != "" {
		polecat.Labels["gastown.io/bead"] = polecat.Spec.BeadID
	}
	polecat.Labels["gastown.io/polecat"] = polecat.Name

	// Set default execution mode
	if polecat.Spec.ExecutionMode == "" {
		polecat.Spec.ExecutionMode = ExecutionModeKubernetes
	}

	// Set default agent type
	if polecat.Spec.Agent == "" {
		polecat.Spec.Agent = AgentTypeClaudeCode
	}

	// Set default desired state
	if polecat.Spec.DesiredState == "" {
		polecat.Spec.DesiredState = PolecatDesiredIdle
	}

	// Set kubernetes defaults
	if polecat.Spec.ExecutionMode == ExecutionModeKubernetes && polecat.Spec.Kubernetes != nil {
		// Set default git branch
		if polecat.Spec.Kubernetes.GitBranch == "" {
			polecat.Spec.Kubernetes.GitBranch = "main"
		}

		// Set default active deadline (1 hour)
		if polecat.Spec.Kubernetes.ActiveDeadlineSeconds == nil {
			defaultDeadline := int64(3600)
			polecat.Spec.Kubernetes.ActiveDeadlineSeconds = &defaultDeadline
		}
	}

	// Set agent config defaults
	if polecat.Spec.AgentConfig == nil {
		polecat.Spec.AgentConfig = &AgentConfig{}
	}
	if polecat.Spec.AgentConfig.Provider == "" {
		polecat.Spec.AgentConfig.Provider = LLMProviderLiteLLM
	}

	return nil
}
