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
	"net/url"
	"regexp"
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var riglog = logf.Log.WithName("rig-resource")

// SetupRigWebhookWithManager registers the Rig webhooks with the manager.
func SetupRigWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &Rig{}).
		WithValidator(&RigCustomValidator{}).
		WithDefaulter(&RigCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-gastown-gastown-io-v1alpha1-rig,mutating=true,failurePolicy=fail,sideEffects=None,groups=gastown.gastown.io,resources=rigs,verbs=create;update,versions=v1alpha1,name=mrig.kb.io,admissionReviewVersions=v1

// RigCustomDefaulter implements admission.Defaulter[*Rig] for Rig.
type RigCustomDefaulter struct{}

var _ admission.Defaulter[*Rig] = &RigCustomDefaulter{}

// Default implements admission.Defaulter so a webhook will be registered for the type.
func (d *RigCustomDefaulter) Default(ctx context.Context, rig *Rig) error {
	riglog.Info("defaulting", "name", rig.Name)

	// Default MaxPolecats to 8 if not set
	if rig.Spec.Settings.MaxPolecats == 0 {
		rig.Spec.Settings.MaxPolecats = 8
	}

	// Default NamepoolTheme to "mad-max" if not set
	if rig.Spec.Settings.NamepoolTheme == "" {
		rig.Spec.Settings.NamepoolTheme = "mad-max"
	}

	return nil
}

// +kubebuilder:webhook:path=/validate-gastown-gastown-io-v1alpha1-rig,mutating=false,failurePolicy=fail,sideEffects=None,groups=gastown.gastown.io,resources=rigs,verbs=create;update,versions=v1alpha1,name=vrig.kb.io,admissionReviewVersions=v1

// RigCustomValidator implements admission.Validator[*Rig] for Rig.
type RigCustomValidator struct{}

var _ admission.Validator[*Rig] = &RigCustomValidator{}

// ValidateCreate implements admission.Validator so a webhook will be registered for the type.
func (v *RigCustomValidator) ValidateCreate(ctx context.Context, rig *Rig) (admission.Warnings, error) {
	riglog.Info("validate create", "name", rig.Name)

	return v.validateRig(rig)
}

// ValidateUpdate implements admission.Validator so a webhook will be registered for the type.
func (v *RigCustomValidator) ValidateUpdate(ctx context.Context, oldRig, rig *Rig) (admission.Warnings, error) {
	riglog.Info("validate update", "name", rig.Name)

	// Validate immutable fields
	if oldRig.Spec.BeadsPrefix != rig.Spec.BeadsPrefix {
		return nil, fmt.Errorf("spec.beadsPrefix is immutable: cannot change from %q to %q",
			oldRig.Spec.BeadsPrefix, rig.Spec.BeadsPrefix)
	}

	return v.validateRig(rig)
}

// ValidateDelete implements admission.Validator so a webhook will be registered for the type.
func (v *RigCustomValidator) ValidateDelete(ctx context.Context, rig *Rig) (admission.Warnings, error) {
	riglog.Info("validate delete", "name", rig.Name)

	// No validation on delete
	return nil, nil
}

// validateRig performs validation common to create and update.
func (v *RigCustomValidator) validateRig(rig *Rig) (admission.Warnings, error) {
	var allErrs []string
	var warnings admission.Warnings

	// Validate GitURL
	if err := validateGitURL(rig.Spec.GitURL); err != nil {
		allErrs = append(allErrs, fmt.Sprintf("spec.gitURL: %v", err))
	}

	// Validate BeadsPrefix
	if err := validateBeadsPrefix(rig.Spec.BeadsPrefix); err != nil {
		allErrs = append(allErrs, fmt.Sprintf("spec.beadsPrefix: %v", err))
	}

	// Validate Settings if present
	if rig.Spec.Settings.MaxPolecats > 50 {
		warnings = append(warnings, "spec.settings.maxPolecats > 50 may cause resource contention")
	}

	if rig.Spec.Settings.NamepoolTheme != "" {
		if err := validateNamepoolTheme(rig.Spec.Settings.NamepoolTheme); err != nil {
			allErrs = append(allErrs, fmt.Sprintf("spec.settings.namepoolTheme: %v", err))
		}
	}

	if len(allErrs) > 0 {
		return warnings, fmt.Errorf("validation failed: %s", strings.Join(allErrs, "; "))
	}

	return warnings, nil
}

// validateGitURL checks that the Git URL is valid.
func validateGitURL(gitURL string) error {
	if gitURL == "" {
		return fmt.Errorf("gitURL is required")
	}

	// Check for SSH format: git@host:path.git
	sshPattern := regexp.MustCompile(`^git@[\w.-]+:[\w./-]+(?:\.git)?$`)
	if sshPattern.MatchString(gitURL) {
		return nil
	}

	// Check for HTTPS format
	if strings.HasPrefix(gitURL, "https://") || strings.HasPrefix(gitURL, "http://") {
		parsed, err := url.Parse(gitURL)
		if err != nil {
			return fmt.Errorf("invalid URL: %w", err)
		}
		if parsed.Host == "" {
			return fmt.Errorf("URL must have a host")
		}
		if parsed.Path == "" || parsed.Path == "/" {
			return fmt.Errorf("URL must include a repository path")
		}
		return nil
	}

	// Check for file:// protocol (local repos)
	if strings.HasPrefix(gitURL, "file://") {
		return nil
	}

	return fmt.Errorf("gitURL must be SSH (git@host:path), HTTPS, HTTP, or file:// format")
}

// validateBeadsPrefix checks that the beads prefix is valid.
func validateBeadsPrefix(prefix string) error {
	if prefix == "" {
		return fmt.Errorf("beadsPrefix is required")
	}

	// Pattern: 2-10 lowercase letters
	pattern := regexp.MustCompile(`^[a-z]{2,10}$`)
	if !pattern.MatchString(prefix) {
		return fmt.Errorf("must be 2-10 lowercase letters, got %q", prefix)
	}

	// Reserved prefixes
	reserved := map[string]bool{
		"hq":      true, // Town-level beads
		"system":  true,
		"default": true,
	}
	if reserved[prefix] {
		return fmt.Errorf("prefix %q is reserved", prefix)
	}

	return nil
}

// validateNamepoolTheme checks that the namepool theme is valid.
func validateNamepoolTheme(theme string) error {
	validThemes := map[string]bool{
		"mad-max":   true,
		"minerals":  true,
		"wasteland": true,
	}
	if !validThemes[theme] {
		return fmt.Errorf("unknown theme %q, valid themes: mad-max, minerals, wasteland", theme)
	}
	return nil
}
