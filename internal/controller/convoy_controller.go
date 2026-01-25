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

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
	gterrors "github.com/org/gastown-operator/pkg/errors"
	"github.com/org/gastown-operator/pkg/metrics"
)

const (
	// ConvoySyncInterval is how often we re-sync convoy status.
	// Uses RequeueDefault for normal sync operations.
	ConvoySyncInterval = RequeueDefault

	// Condition types for Convoy
	ConditionConvoyReady    = "Ready"
	ConditionConvoyComplete = "Complete"
)

// ConvoyReconciler reconciles a Convoy object.
// It tracks progress by watching Polecat status.
type ConvoyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=gastown.gastown.io,resources=convoys,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=convoys/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=convoys/finalizers,verbs=update
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=polecats,verbs=get;list;watch

// Reconcile tracks convoy progress by watching Polecat status.
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

	// Initialize convoy if not started
	if convoy.Status.Phase == "" || convoy.Status.Phase == gastownv1alpha1.ConvoyPhasePending {
		now := metav1.Now()
		convoy.Status.StartedAt = &now
		convoy.Status.Phase = gastownv1alpha1.ConvoyPhaseInProgress
		convoy.Status.PendingBeads = convoy.Spec.TrackedBeads
		convoy.Status.CompletedBeads = []string{}

		r.setCondition(&convoy, ConditionConvoyReady, metav1.ConditionTrue, "Started",
			"Convoy started tracking beads")

		if err := r.Status().Update(ctx, &convoy); err != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(err, "failed to update convoy status")
		}

		log.Info("Convoy initialized", "trackedBeads", len(convoy.Spec.TrackedBeads))
	}

	// Get all polecats to check their assigned beads
	var polecatList gastownv1alpha1.PolecatList
	if err := r.List(ctx, &polecatList); err != nil {
		log.Error(err, "Failed to list polecats")
		r.setCondition(&convoy, ConditionConvoyReady, metav1.ConditionFalse, "ListFailed",
			err.Error())

		if updateErr := r.Status().Update(ctx, &convoy); updateErr != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update convoy status")
		}

		timer.RecordResult(metrics.ResultRequeue)
		return ctrl.Result{RequeueAfter: RequeueDefault}, nil
	}

	// Build a map of bead ID -> polecat phase
	beadStatus := make(map[string]gastownv1alpha1.PolecatPhase)
	for _, polecat := range polecatList.Items {
		if polecat.Status.AssignedBead != "" {
			beadStatus[polecat.Status.AssignedBead] = polecat.Status.Phase
		}
	}

	// Categorize tracked beads
	var completed, pending []string
	for _, beadID := range convoy.Spec.TrackedBeads {
		phase, found := beadStatus[beadID]
		if found && phase == gastownv1alpha1.PolecatPhaseDone {
			completed = append(completed, beadID)
		} else {
			pending = append(pending, beadID)
		}
	}

	// Update status
	convoy.Status.CompletedBeads = completed
	convoy.Status.PendingBeads = pending
	convoy.Status.Progress = fmt.Sprintf("%d/%d", len(completed), len(convoy.Spec.TrackedBeads))

	r.setCondition(&convoy, ConditionConvoyReady, metav1.ConditionTrue, "Synced",
		"Convoy status synced from Polecats")

	// Check for completion
	if len(pending) == 0 && len(completed) == len(convoy.Spec.TrackedBeads) {
		now := metav1.Now()
		convoy.Status.CompletedAt = &now
		convoy.Status.Phase = gastownv1alpha1.ConvoyPhaseComplete
		r.setCondition(&convoy, ConditionConvoyComplete, metav1.ConditionTrue, "Complete",
			"All tracked beads completed")

		log.Info("Convoy completed", "completed", len(completed))
	} else {
		r.setCondition(&convoy, ConditionConvoyComplete, metav1.ConditionFalse, "InProgress",
			fmt.Sprintf("Progress: %s", convoy.Status.Progress))
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

// setCondition sets or updates a condition on the Convoy using the standard meta.SetStatusCondition helper.
func (r *ConvoyReconciler) setCondition(convoy *gastownv1alpha1.Convoy, condType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&convoy.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: convoy.Generation,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConvoyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gastownv1alpha1.Convoy{}).
		Named("convoy").
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 3, // Limit concurrent convoy processing
		}).
		Complete(r)
}
