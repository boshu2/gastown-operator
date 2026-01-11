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
	// PolecatSyncInterval is how often we re-sync with gt CLI
	PolecatSyncInterval = 10 * time.Second

	// Condition types for Polecat
	ConditionPolecatReady   = "Ready"
	ConditionPolecatWorking = "Working"
)

// PolecatReconciler reconciles a Polecat object
type PolecatReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	GTClient *gt.Client
}

// +kubebuilder:rbac:groups=gastown.gastown.io,resources=polecats,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=polecats/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=polecats/finalizers,verbs=update

// Reconcile implements the state machine for Polecat lifecycle.
func (r *PolecatReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	timer := metrics.NewReconcileTimer("polecat")
	defer timer.ObserveDuration()

	// Fetch the Polecat instance
	var polecat gastownv1alpha1.Polecat
	if err := r.Get(ctx, req.NamespacedName, &polecat); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Reconciling Polecat",
		"name", polecat.Name,
		"rig", polecat.Spec.Rig,
		"desiredState", polecat.Spec.DesiredState)

	// Handle based on desired state
	switch polecat.Spec.DesiredState {
	case gastownv1alpha1.PolecatDesiredWorking:
		return r.ensureWorking(ctx, &polecat, timer)
	case gastownv1alpha1.PolecatDesiredIdle:
		return r.ensureIdle(ctx, &polecat, timer)
	case gastownv1alpha1.PolecatDesiredTerminated:
		return r.ensureTerminated(ctx, &polecat, timer)
	default:
		log.Info("Unknown desired state, defaulting to idle")
		return r.ensureIdle(ctx, &polecat, timer)
	}
}

// ensureWorking ensures the polecat is working on a bead.
func (r *PolecatReconciler) ensureWorking(ctx context.Context, polecat *gastownv1alpha1.Polecat, timer *metrics.ReconcileTimer) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Check if polecat exists
	exists, err := r.GTClient.PolecatExists(ctx, polecat.Spec.Rig, polecat.Name)
	if err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to check polecat existence")
	}

	// If polecat doesn't exist and we have a bead to work on, create via sling
	if !exists && polecat.Spec.BeadID != "" {
		log.Info("Creating polecat via gt sling",
			"beadID", polecat.Spec.BeadID,
			"rig", polecat.Spec.Rig)

		if err := r.GTClient.Sling(ctx, polecat.Spec.BeadID, polecat.Spec.Rig); err != nil {
			r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionFalse, "SlingFailed",
				err.Error())
			polecat.Status.Phase = gastownv1alpha1.PolecatPhaseStuck

			if updateErr := r.Status().Update(ctx, polecat); updateErr != nil {
				timer.RecordResult(metrics.ResultError)
				return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update polecat status")
			}

			timer.RecordResult(metrics.ResultRequeue)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Sync status from gt CLI
	status, err := r.GTClient.PolecatStatus(ctx, polecat.Spec.Rig, polecat.Name)
	if err != nil {
		log.Error(err, "Failed to get polecat status from gt CLI")
		r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionFalse, "GTCLIError",
			err.Error())

		if updateErr := r.Status().Update(ctx, polecat); updateErr != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update polecat status")
		}

		timer.RecordResult(metrics.ResultRequeue)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Update status from gt CLI response
	r.syncStatusFromGT(polecat, status)
	r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionTrue, "Synced",
		"Successfully synced with gt CLI")
	r.setCondition(polecat, ConditionPolecatWorking, metav1.ConditionTrue, "Working",
		"Polecat is working on assigned bead")

	if err := r.Status().Update(ctx, polecat); err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to update polecat status")
	}

	log.Info("Polecat reconciled (working)",
		"phase", polecat.Status.Phase,
		"bead", polecat.Status.AssignedBead,
		"sessionActive", polecat.Status.SessionActive)

	timer.RecordResult(metrics.ResultSuccess)
	return ctrl.Result{RequeueAfter: PolecatSyncInterval}, nil
}

// ensureIdle ensures the polecat is in idle state.
func (r *PolecatReconciler) ensureIdle(ctx context.Context, polecat *gastownv1alpha1.Polecat, timer *metrics.ReconcileTimer) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Check if polecat exists and has work
	exists, err := r.GTClient.PolecatExists(ctx, polecat.Spec.Rig, polecat.Name)
	if err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to check polecat existence")
	}

	if exists {
		// Get current status
		status, err := r.GTClient.PolecatStatus(ctx, polecat.Spec.Rig, polecat.Name)
		if err != nil {
			timer.RecordResult(metrics.ResultRequeue)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		// If polecat is working, reset it to idle
		if status.Phase == "Working" || status.AssignedBead != "" {
			log.Info("Resetting polecat to idle")
			if err := r.GTClient.PolecatReset(ctx, polecat.Spec.Rig, polecat.Name); err != nil {
				log.Error(err, "Failed to reset polecat")
			}
		}

		r.syncStatusFromGT(polecat, status)
	}

	polecat.Status.Phase = gastownv1alpha1.PolecatPhaseIdle
	r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionTrue, "Idle",
		"Polecat is idle and ready for work")
	r.setCondition(polecat, ConditionPolecatWorking, metav1.ConditionFalse, "Idle",
		"No work assigned")

	if err := r.Status().Update(ctx, polecat); err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to update polecat status")
	}

	timer.RecordResult(metrics.ResultSuccess)
	return ctrl.Result{RequeueAfter: PolecatSyncInterval}, nil
}

// ensureTerminated ensures the polecat is terminated.
func (r *PolecatReconciler) ensureTerminated(ctx context.Context, polecat *gastownv1alpha1.Polecat, timer *metrics.ReconcileTimer) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Check if polecat exists
	exists, err := r.GTClient.PolecatExists(ctx, polecat.Spec.Rig, polecat.Name)
	if err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to check polecat existence")
	}

	if exists {
		// Get current status to check for uncommitted work
		status, err := r.GTClient.PolecatStatus(ctx, polecat.Spec.Rig, polecat.Name)
		if err == nil && status.CleanupStatus != "clean" {
			// Don't nuke if there's uncommitted work
			log.Info("Polecat has uncommitted work, refusing to terminate",
				"cleanupStatus", status.CleanupStatus)
			r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionFalse, "UncommittedWork",
				"Polecat has uncommitted work, cannot terminate")

			if updateErr := r.Status().Update(ctx, polecat); updateErr != nil {
				timer.RecordResult(metrics.ResultError)
				return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update polecat status")
			}

			timer.RecordResult(metrics.ResultRequeue)
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}

		// Nuke the polecat
		log.Info("Terminating polecat")
		if err := r.GTClient.PolecatNuke(ctx, polecat.Spec.Rig, polecat.Name, false); err != nil {
			log.Error(err, "Failed to nuke polecat")
			r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionFalse, "NukeFailed",
				err.Error())

			if updateErr := r.Status().Update(ctx, polecat); updateErr != nil {
				timer.RecordResult(metrics.ResultError)
				return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update polecat status")
			}

			timer.RecordResult(metrics.ResultRequeue)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Update status to terminated
	polecat.Status.Phase = gastownv1alpha1.PolecatPhaseTerminated
	polecat.Status.SessionActive = false
	polecat.Status.AssignedBead = ""
	r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionTrue, "Terminated",
		"Polecat has been terminated")

	if err := r.Status().Update(ctx, polecat); err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to update polecat status")
	}

	log.Info("Polecat terminated successfully")
	timer.RecordResult(metrics.ResultSuccess)
	// Don't requeue - polecat is terminated
	return ctrl.Result{}, nil
}

// syncStatusFromGT updates the polecat status from gt CLI response.
func (r *PolecatReconciler) syncStatusFromGT(polecat *gastownv1alpha1.Polecat, status *gt.PolecatStatus) {
	polecat.Status.Phase = gastownv1alpha1.PolecatPhase(status.Phase)
	polecat.Status.AssignedBead = status.AssignedBead
	polecat.Status.Branch = status.Branch
	polecat.Status.WorktreePath = status.WorktreePath
	polecat.Status.TmuxSession = status.TmuxSession
	polecat.Status.SessionActive = status.SessionActive
	polecat.Status.CleanupStatus = gastownv1alpha1.CleanupStatus(status.CleanupStatus)

	if !status.LastActivity.IsZero() {
		t := metav1.NewTime(status.LastActivity)
		polecat.Status.LastActivity = &t
	}
}

// setCondition sets or updates a condition on the Polecat.
func (r *PolecatReconciler) setCondition(polecat *gastownv1alpha1.Polecat, condType string, status metav1.ConditionStatus, reason, message string) {
	condition := metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: polecat.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}

	for i, existing := range polecat.Status.Conditions {
		if existing.Type == condType {
			if existing.Status != status {
				polecat.Status.Conditions[i] = condition
			}
			return
		}
	}
	polecat.Status.Conditions = append(polecat.Status.Conditions, condition)
}

// SetupWithManager sets up the controller with the Manager.
func (r *PolecatReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gastownv1alpha1.Polecat{}).
		Named("polecat").
		Complete(r)
}
