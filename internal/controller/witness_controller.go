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
)

const (
	// WitnessConditionHealthy indicates the witness is successfully monitoring.
	WitnessConditionHealthy = "Healthy"

	// WitnessConditionDegraded indicates monitoring issues.
	WitnessConditionDegraded = "Degraded"

	// Default intervals if not specified in spec.
	defaultHealthCheckInterval = 30 * time.Second
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
		r.setCondition(witness, WitnessConditionDegraded, metav1.ConditionTrue,
			"ListFailed", "Failed to list Polecats")
		return ctrl.Result{RequeueAfter: healthCheckInterval}, r.Status().Update(ctx, witness)
	}

	// Calculate summary
	summary := r.calculateSummary(polecatList, stuckThreshold)

	// Update status
	witness.Status.Phase = r.determinePhase(summary)
	witness.Status.LastCheckTime = &metav1.Time{Time: time.Now()}
	witness.Status.PolecatsSummary = summary

	// Set healthy condition
	if summary.Stuck > 0 || summary.Failed > 0 {
		r.setCondition(witness, WitnessConditionHealthy, metav1.ConditionFalse,
			"IssuesDetected", "Stuck or failed polecats detected")

		// Emit event for stuck polecats
		if summary.Stuck > 0 {
			r.Recorder.Event(witness, "Warning", "StuckPolecats",
				"Detected polecats with no progress")

			// Escalate to configured target
			if r.GTClient != nil {
				r.escalateIssues(ctx, witness, summary)
			}
		}

		if summary.Failed > 0 {
			r.Recorder.Event(witness, "Warning", "FailedPolecats",
				"Detected failed polecats")
		}
	} else {
		r.setCondition(witness, WitnessConditionHealthy, metav1.ConditionTrue,
			"AllHealthy", "All polecats are healthy")
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
func (r *WitnessReconciler) calculateSummary(polecats *gastownv1alpha1.PolecatList, stuckThreshold time.Duration) gastownv1alpha1.PolecatsSummary {
	summary := gastownv1alpha1.PolecatsSummary{}

	for _, polecat := range polecats.Items {
		summary.Total++

		// Check conditions for state
		for _, cond := range polecat.Status.Conditions {
			switch cond.Type {
			case "Available":
				if cond.Status == metav1.ConditionTrue {
					summary.Succeeded++
				}
			case "Progressing":
				if cond.Status == metav1.ConditionTrue {
					summary.Running++
					// Check if stuck (no update for too long)
					if time.Since(cond.LastTransitionTime.Time) > stuckThreshold {
						summary.Stuck++
					}
				}
			case "Degraded":
				if cond.Status == metav1.ConditionTrue {
					summary.Failed++
				}
			}
		}
	}

	return summary
}

// determinePhase returns the Witness phase based on summary.
func (r *WitnessReconciler) determinePhase(summary gastownv1alpha1.PolecatsSummary) string {
	if summary.Stuck > 0 || summary.Failed > 0 {
		return "Degraded"
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
func (r *WitnessReconciler) escalateIssues(ctx context.Context, witness *gastownv1alpha1.Witness, summary gastownv1alpha1.PolecatsSummary) {
	log := logf.FromContext(ctx)

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
