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

package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
	gterrors "github.com/org/gastown-operator/pkg/errors"
)

const (
	// ConditionWitnessReady indicates the witness is successfully monitoring.
	// Changed from "Healthy" to "Ready" for consistency with other controllers.
	// See constants.go for the condition naming convention.
	ConditionWitnessReady = ConditionReady

	// ConditionWitnessDegraded indicates monitoring issues.
	// "Degraded" is a standard Kubernetes condition type.
	ConditionWitnessDegraded = ConditionDegraded

	// Default intervals if not specified in spec.
	// Uses RequeueDefault for health check frequency.
	defaultHealthCheckInterval = RequeueDefault
	defaultStuckThreshold      = 15 * time.Minute
)

// WitnessReconciler reconciles a Witness object
type WitnessReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	GTClient interface {
		MailSend(ctx context.Context, address, subject, message string) error
	}
	// Backoff provides circuit breaker functionality for escalation.
	// If nil, escalation always proceeds. When configured, prevents
	// excessive escalation when issues persist.
	Backoff *gterrors.BackoffCalculator
}

// +kubebuilder:rbac:groups=gastown.gastown.io,resources=witnesses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=witnesses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=witnesses/finalizers,verbs=update
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=polecats,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile monitors Polecat health for the Witness's Rig and updates status.
func (r *WitnessReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Witness instance
	witness := &gastownv1alpha1.Witness{}
	if err := r.Get(ctx, req.NamespacedName, witness); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Reconciling Witness", "rigRef", witness.Spec.RigRef)

	// Get health check interval from spec or use default
	healthCheckInterval := defaultHealthCheckInterval
	if witness.Spec.HealthCheckInterval != nil {
		healthCheckInterval = witness.Spec.HealthCheckInterval.Duration
	}

	// Get stuck threshold from spec or use default
	stuckThreshold := defaultStuckThreshold
	if witness.Spec.StuckThreshold != nil {
		stuckThreshold = witness.Spec.StuckThreshold.Duration
	}

	// List Polecats in the namespace that belong to this rig
	polecatList := &gastownv1alpha1.PolecatList{}
	listOpts := []client.ListOption{
		client.InNamespace(req.Namespace),
		client.MatchingLabels{"gastown.io/rig": witness.Spec.RigRef},
	}
	if err := r.List(ctx, polecatList, listOpts...); err != nil {
		log.Error(err, "Failed to list Polecats")
		r.setCondition(witness, ConditionWitnessDegraded, metav1.ConditionTrue,
			"ListFailed", "Failed to list Polecats")
		return ctrl.Result{RequeueAfter: healthCheckInterval}, r.Status().Update(ctx, witness)
	}

	// Calculate summary
	summary := r.calculateSummary(polecatList, stuckThreshold)

	// Update status
	witness.Status.Phase = r.determinePhase(summary)
	witness.Status.LastCheckTime = &metav1.Time{Time: time.Now()}
	witness.Status.PolecatsSummary = summary

	// Key for backoff tracking
	backoffKey := fmt.Sprintf("%s/%s", witness.Namespace, witness.Name)

	// Set healthy condition
	if summary.Stuck > 0 || summary.Failed > 0 {
		r.setCondition(witness, ConditionWitnessReady, metav1.ConditionFalse,
			"IssuesDetected", "Stuck or failed polecats detected")

		// Emit event for stuck polecats
		if summary.Stuck > 0 {
			r.Recorder.Event(witness, "Warning", "StuckPolecats",
				"Detected polecats with no progress")

			// Escalate to configured target (with circuit breaker)
			if r.GTClient != nil {
				shouldEscalate := true
				if r.Backoff != nil {
					if r.Backoff.ShouldGiveUp(backoffKey) {
						log.Info("Circuit breaker open, skipping escalation",
							"witness", witness.Name,
							"retries", r.Backoff.GetRetryCount(backoffKey))
						r.Recorder.Event(witness, "Warning", "EscalationCircuitBreaker",
							"Too many escalation attempts, circuit breaker open")
						shouldEscalate = false
					}
				}
				if shouldEscalate {
					r.escalateIssues(ctx, witness, summary, backoffKey)
				}
			}
		}

		if summary.Failed > 0 {
			r.Recorder.Event(witness, "Warning", "FailedPolecats",
				"Detected failed polecats")
		}
	} else {
		r.setCondition(witness, ConditionWitnessReady, metav1.ConditionTrue,
			"AllHealthy", "All polecats are healthy")

		// Reset circuit breaker when healthy
		if r.Backoff != nil {
			r.Backoff.ResetRetries(backoffKey)
		}
	}

	// Update status
	if err := r.Status().Update(ctx, witness); err != nil {
		log.Error(err, "Failed to update Witness status")
		return ctrl.Result{}, err
	}

	log.Info("Witness health check complete",
		"total", summary.Total,
		"running", summary.Running,
		"succeeded", summary.Succeeded,
		"failed", summary.Failed,
		"stuck", summary.Stuck)

	// Requeue after health check interval
	return ctrl.Result{RequeueAfter: healthCheckInterval}, nil
}

// calculateSummary aggregates Polecat states.
// Checks both new standard conditions (Available, Progressing, Degraded) and
// falls back to old conditions (Ready, Working) for backward compatibility.
func (r *WitnessReconciler) calculateSummary(polecats *gastownv1alpha1.PolecatList, stuckThreshold time.Duration) gastownv1alpha1.PolecatsSummary {
	summary := gastownv1alpha1.PolecatsSummary{}

	for _, polecat := range polecats.Items {
		summary.Total++

		// Track what we find for this polecat
		var hasAvailable, hasProgressing bool
		var hasOldWorking bool
		var workingCond metav1.Condition

		// Check all conditions
		for _, cond := range polecat.Status.Conditions {
			switch cond.Type {
			// New standard conditions
			case ConditionAvailable:
				hasAvailable = true
				if cond.Status == metav1.ConditionTrue {
					summary.Succeeded++
				}
			case ConditionProgressing:
				hasProgressing = true
				if cond.Status == metav1.ConditionTrue {
					summary.Running++
					// Check if stuck (no update for too long)
					if time.Since(cond.LastTransitionTime.Time) > stuckThreshold {
						summary.Stuck++
					}
				}
			case ConditionDegraded:
				if cond.Status == metav1.ConditionTrue {
					summary.Failed++
				}
			// Old conditions (backward compatibility)
			case "Ready":
				// Only use Ready as fallback for Available if Available isn't present
				if cond.Status == metav1.ConditionTrue && cond.Reason == "PodSucceeded" {
					// This is a completed polecat using old conditions
					if !hasAvailable {
						summary.Succeeded++
					}
				}
			case "Working":
				hasOldWorking = true
				workingCond = cond
			}
		}

		// Fallback: if new conditions aren't present, use old ones
		if !hasProgressing && hasOldWorking && workingCond.Status == metav1.ConditionTrue {
			summary.Running++
			// Check if stuck using old Working condition
			if time.Since(workingCond.LastTransitionTime.Time) > stuckThreshold {
				summary.Stuck++
			}
		}

		// Note: If neither new nor old conditions are present, polecat is not counted
		// in running/succeeded/failed - this is expected for newly created polecats
	}

	return summary
}

// determinePhase returns the Witness phase based on summary.
func (r *WitnessReconciler) determinePhase(summary gastownv1alpha1.PolecatsSummary) string {
	if summary.Stuck > 0 || summary.Failed > 0 {
		return ConditionDegraded // Phase matches condition name
	}
	if summary.Running > 0 {
		return "Active"
	}
	return "Pending"
}

// setCondition updates or adds a condition to the Witness status.
func (r *WitnessReconciler) setCondition(witness *gastownv1alpha1.Witness, condType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&witness.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

// escalateIssues sends escalation alerts based on the configured escalation target.
// The backoffKey is used to track escalation attempts for the circuit breaker.
func (r *WitnessReconciler) escalateIssues(ctx context.Context, witness *gastownv1alpha1.Witness, summary gastownv1alpha1.PolecatsSummary, backoffKey string) {
	log := logf.FromContext(ctx)

	// Increment backoff counter for each escalation attempt
	if r.Backoff != nil {
		_ = r.Backoff.GetBackoffResult(backoffKey) // Increments counter, we ignore the Result
		log.Info("Escalation attempt recorded",
			"witness", witness.Name,
			"attempts", r.Backoff.GetRetryCount(backoffKey))
	}

	target := witness.Spec.EscalationTarget
	if target == "" {
		target = "mayor"
	}

	subject := fmt.Sprintf("Health Alert: Witness %s.%s detected issues", witness.Namespace, witness.Name)
	message := fmt.Sprintf("Rig: %s\nPhase: %s\nStuck Polecats: %d\nFailed Polecats: %d\nRunning: %d/%d",
		witness.Spec.RigRef, witness.Status.Phase, summary.Stuck, summary.Failed, summary.Running, summary.Total)

	switch target {
	case "mayor":
		if r.GTClient != nil {
			if err := r.GTClient.MailSend(ctx, target, subject, message); err != nil {
				log.Error(err, "Failed to send escalation mail to mayor", "witness", witness.Name)
				r.Recorder.Event(witness, "Warning", "EscalationFailed",
					fmt.Sprintf("Failed to send alert to mayor: %v", err))
			} else {
				log.Info("Escalation mail sent to mayor", "witness", witness.Name)
			}
		}
	case "slack":
		log.Info("Slack escalation configured but not yet implemented", "witness", witness.Name)
		r.Recorder.Event(witness, "Warning", "SlackNotConfigured",
			"Slack escalation is not yet implemented")
	case "email":
		log.Info("Email escalation configured but not yet implemented", "witness", witness.Name)
		r.Recorder.Event(witness, "Warning", "EmailNotConfigured",
			"Email escalation is not yet implemented")
	default:
		log.Info("Unknown escalation target", "witness", witness.Name, "target", target)
		r.Recorder.Event(witness, "Warning", "UnknownEscalationTarget",
			fmt.Sprintf("Unknown escalation target: %s", target))
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *WitnessReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gastownv1alpha1.Witness{}).
		Named("witness").
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 2, // Witnesses are lightweight monitors
		}).
		Complete(r)
}
