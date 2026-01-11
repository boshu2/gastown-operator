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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
	gterrors "github.com/org/gastown-operator/pkg/errors"
	"github.com/org/gastown-operator/pkg/gt"
	"github.com/org/gastown-operator/pkg/metrics"
)

const (
	// ConvoySyncInterval is how often we re-sync with gt CLI
	ConvoySyncInterval = 30 * time.Second

	// Condition types for Convoy
	ConditionConvoyReady    = "Ready"
	ConditionConvoyComplete = "Complete"
)

// ConvoyReconciler reconciles a Convoy object
type ConvoyReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	GTClient *gt.Client
}

// +kubebuilder:rbac:groups=gastown.gastown.io,resources=convoys,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=convoys/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=convoys/finalizers,verbs=update

// Reconcile tracks convoy progress and sends completion notifications.
func (r *ConvoyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	timer := metrics.NewReconcileTimer("convoy")
	defer timer.ObserveDuration()

	// Fetch the Convoy instance
	var convoy gastownv1alpha1.Convoy
	if err := r.Get(ctx, req.NamespacedName, &convoy); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Reconciling Convoy",
		"name", convoy.Name,
		"trackedBeads", len(convoy.Spec.TrackedBeads))

	// If convoy is complete, don't requeue
	if convoy.Status.Phase == gastownv1alpha1.ConvoyPhaseComplete {
		return ctrl.Result{}, nil
	}

	// Ensure convoy exists in beads system
	if convoy.Status.BeadsConvoyID == "" {
		log.Info("Creating convoy in beads system")
		id, err := r.GTClient.ConvoyCreate(ctx, convoy.Spec.Description, convoy.Spec.TrackedBeads)
		if err != nil {
			log.Error(err, "Failed to create convoy in beads")
			r.setCondition(&convoy, ConditionConvoyReady, metav1.ConditionFalse, "CreateFailed",
				err.Error())

			if updateErr := r.Status().Update(ctx, &convoy); updateErr != nil {
				timer.RecordResult(metrics.ResultError)
				return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update convoy status")
			}

			timer.RecordResult(metrics.ResultRequeue)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		convoy.Status.BeadsConvoyID = id
		now := metav1.Now()
		convoy.Status.StartedAt = &now
		convoy.Status.Phase = gastownv1alpha1.ConvoyPhaseInProgress
		convoy.Status.PendingBeads = convoy.Spec.TrackedBeads

		r.setCondition(&convoy, ConditionConvoyReady, metav1.ConditionTrue, "Created",
			"Convoy created in beads system")

		if err := r.Status().Update(ctx, &convoy); err != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(err, "failed to update convoy status")
		}

		log.Info("Convoy created", "beadsID", id)
	}

	// Sync status from gt CLI
	status, err := r.GTClient.ConvoyStatus(ctx, convoy.Status.BeadsConvoyID)
	if err != nil {
		log.Error(err, "Failed to get convoy status from gt CLI")
		r.setCondition(&convoy, ConditionConvoyReady, metav1.ConditionFalse, "GTCLIError",
			err.Error())

		if updateErr := r.Status().Update(ctx, &convoy); updateErr != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update convoy status")
		}

		timer.RecordResult(metrics.ResultRequeue)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Update status from gt CLI response
	convoy.Status.Phase = gastownv1alpha1.ConvoyPhase(status.Phase)
	convoy.Status.Progress = status.Progress
	convoy.Status.CompletedBeads = status.Completed
	convoy.Status.PendingBeads = status.Pending

	r.setCondition(&convoy, ConditionConvoyReady, metav1.ConditionTrue, "Synced",
		"Successfully synced with gt CLI")

	// Check for completion
	if status.Phase == "Complete" && convoy.Status.CompletedAt == nil {
		now := metav1.Now()
		convoy.Status.CompletedAt = &now
		r.setCondition(&convoy, ConditionConvoyComplete, metav1.ConditionTrue, "Complete",
			"All tracked beads completed")

		// Send notification if configured
		if convoy.Spec.NotifyOnComplete != "" {
			r.sendCompletionNotification(ctx, &convoy)
		}

		log.Info("Convoy completed",
			"completed", len(convoy.Status.CompletedBeads),
			"notified", convoy.Spec.NotifyOnComplete != "")
	} else if status.Phase != "Complete" {
		r.setCondition(&convoy, ConditionConvoyComplete, metav1.ConditionFalse, "InProgress",
			fmt.Sprintf("Progress: %s", status.Progress))
	}

	if err := r.Status().Update(ctx, &convoy); err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to update convoy status")
	}

	log.Info("Convoy reconciled",
		"phase", convoy.Status.Phase,
		"progress", convoy.Status.Progress)

	timer.RecordResult(metrics.ResultSuccess)

	// Don't requeue if complete
	if convoy.Status.Phase == gastownv1alpha1.ConvoyPhaseComplete {
		return ctrl.Result{}, nil
	}

	return ctrl.Result{RequeueAfter: ConvoySyncInterval}, nil
}

// sendCompletionNotification sends a mail notification on convoy completion.
func (r *ConvoyReconciler) sendCompletionNotification(ctx context.Context, convoy *gastownv1alpha1.Convoy) {
	log := logf.FromContext(ctx)

	subject := fmt.Sprintf("Convoy Complete: %s", convoy.Spec.Description)
	message := fmt.Sprintf("Convoy %s has completed.\n\nCompleted beads: %d\n\n%v",
		convoy.Name,
		len(convoy.Status.CompletedBeads),
		convoy.Status.CompletedBeads)

	if err := r.GTClient.MailSend(ctx, convoy.Spec.NotifyOnComplete, subject, message); err != nil {
		log.Error(err, "Failed to send completion notification",
			"address", convoy.Spec.NotifyOnComplete)
	} else {
		log.Info("Sent completion notification", "address", convoy.Spec.NotifyOnComplete)
	}
}

// setCondition sets or updates a condition on the Convoy.
func (r *ConvoyReconciler) setCondition(convoy *gastownv1alpha1.Convoy, condType string, status metav1.ConditionStatus, reason, message string) {
	condition := metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: convoy.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}

	for i, existing := range convoy.Status.Conditions {
		if existing.Type == condType {
			if existing.Status != status {
				convoy.Status.Conditions[i] = condition
			}
			return
		}
	}
	convoy.Status.Conditions = append(convoy.Status.Conditions, condition)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConvoyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gastownv1alpha1.Convoy{}).
		Named("convoy").
		Complete(r)
}
