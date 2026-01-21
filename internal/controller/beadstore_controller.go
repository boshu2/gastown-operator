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
	"math"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
	gterrors "github.com/org/gastown-operator/pkg/errors"
	"github.com/org/gastown-operator/pkg/gt"
	"github.com/org/gastown-operator/pkg/metrics"
)

const (
	// BeadStoreSyncInterval is how often we sync beads
	BeadStoreSyncInterval = 5 * time.Minute

	// Condition types for BeadStore
	ConditionBeadStoreSynced = "Synced"
	ConditionBeadStoreReady  = "Ready"

	// Phase constants for BeadStore status
	PhaseError   = "Error"
	PhasePending = "Pending"
	PhaseSynced  = "Synced"

	// beadstoreFinalizer ensures cleanup of external resources
	beadstoreFinalizer = "gastown.io/beadstore-cleanup"
)

// BeadStoreReconciler reconciles a BeadStore object
type BeadStoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// GTClient is optional - used in local mode for rig validation
	GTClient gt.ClientInterface
}

// +kubebuilder:rbac:groups=gastown.gastown.io,resources=beadstores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=beadstores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=beadstores/finalizers,verbs=update
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=rigs,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile manages the BeadStore lifecycle.
// It syncs the beads database from git to make them available for work assignment.
func (r *BeadStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	timer := metrics.NewReconcileTimer("beadstore")
	defer timer.ObserveDuration()

	// Fetch the BeadStore instance
	var beadstore gastownv1alpha1.BeadStore
	if err := r.Get(ctx, req.NamespacedName, &beadstore); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Reconciling BeadStore",
		"name", beadstore.Name,
		"rig", beadstore.Spec.RigRef,
		"prefix", beadstore.Spec.Prefix)

	// Handle deletion with finalizer
	if !beadstore.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &beadstore, timer)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&beadstore, beadstoreFinalizer) {
		log.Info("Adding finalizer to BeadStore")
		controllerutil.AddFinalizer(&beadstore, beadstoreFinalizer)
		if err := r.Update(ctx, &beadstore); err != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(err, "failed to add finalizer")
		}
		return ctrl.Result{RequeueAfter: time.Millisecond}, nil
	}

	// Validate RigRef
	rigExists, err := r.validateRig(ctx, beadstore.Spec.RigRef)
	if err != nil {
		log.Error(err, "Failed to validate rig")
		r.setCondition(&beadstore, ConditionBeadStoreReady, metav1.ConditionFalse, "RigValidationFailed",
			err.Error())
		beadstore.Status.Phase = PhaseError
		if updateErr := r.Status().Update(ctx, &beadstore); updateErr != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update status")
		}
		timer.RecordResult(metrics.ResultRequeue)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	if !rigExists {
		log.Info("Rig not found", "rig", beadstore.Spec.RigRef)
		r.setCondition(&beadstore, ConditionBeadStoreReady, metav1.ConditionFalse, "RigNotFound",
			fmt.Sprintf("Rig %q not found", beadstore.Spec.RigRef))
		beadstore.Status.Phase = PhasePending
		if err := r.Status().Update(ctx, &beadstore); err != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(err, "failed to update status")
		}
		timer.RecordResult(metrics.ResultRequeue)
		// Requeue to check again later
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Sync beads from git
	rigStatus, err := r.GTClient.RigStatus(ctx, beadstore.Spec.RigRef)
	if err != nil {
		log.Error(err, "Failed to get rig status")
		r.setCondition(&beadstore, ConditionBeadStoreSynced, metav1.ConditionFalse, "RigStatusFailed",
			err.Error())
		beadstore.Status.Phase = PhaseError
		if updateErr := r.Status().Update(ctx, &beadstore); updateErr != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update status")
		}
		timer.RecordResult(metrics.ResultRequeue)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	// Sync beads from the git repository
	issueCount, err := r.syncBeadsFromGit(ctx, beadstore.Spec.RigRef, rigStatus)
	if err != nil {
		log.Error(err, "Failed to sync beads", "rig", beadstore.Spec.RigRef)
		r.setCondition(&beadstore, ConditionBeadStoreSynced, metav1.ConditionFalse, "SyncFailed",
			err.Error())
		beadstore.Status.Phase = PhaseError
		if updateErr := r.Status().Update(ctx, &beadstore); updateErr != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update status")
		}
		timer.RecordResult(metrics.ResultRequeue)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	// Update status (cap issueCount at MaxInt32 to avoid overflow)
	if issueCount > math.MaxInt32 {
		issueCount = math.MaxInt32
	}
	now := metav1.Now()
	beadstore.Status.Phase = PhaseSynced
	beadstore.Status.LastSyncTime = &now
	beadstore.Status.IssueCount = int32(issueCount) // #nosec G115 -- bounds checked above

	r.setCondition(&beadstore, ConditionBeadStoreSynced, metav1.ConditionTrue, "SyncSucceeded",
		fmt.Sprintf("Successfully synced %d issues", issueCount))
	r.setCondition(&beadstore, ConditionBeadStoreReady, metav1.ConditionTrue, "Ready",
		"BeadStore is ready")

	if err := r.Status().Update(ctx, &beadstore); err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to update status")
	}

	log.Info("BeadStore reconciled successfully",
		"rig", beadstore.Spec.RigRef,
		"issueCount", issueCount)

	timer.RecordResult(metrics.ResultSuccess)

	// Determine sync interval
	syncInterval := BeadStoreSyncInterval
	if beadstore.Spec.SyncInterval != nil {
		syncInterval = beadstore.Spec.SyncInterval.Duration
	}

	return ctrl.Result{RequeueAfter: syncInterval}, nil
}

// handleDeletion handles cleanup when BeadStore is deleted
//
//nolint:unparam // Result is always empty but signature matches controller pattern
func (r *BeadStoreReconciler) handleDeletion(ctx context.Context, beadstore *gastownv1alpha1.BeadStore, timer *metrics.ReconcileTimer) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if controllerutil.ContainsFinalizer(beadstore, beadstoreFinalizer) {
		log.Info("Deleting BeadStore")
		// No external resources to clean up, just remove finalizer
		controllerutil.RemoveFinalizer(beadstore, beadstoreFinalizer)
		if err := r.Update(ctx, beadstore); err != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(err, "failed to remove finalizer")
		}
		timer.RecordResult(metrics.ResultSuccess)
	}

	return ctrl.Result{}, nil
}

// validateRig checks if the referenced Rig exists
func (r *BeadStoreReconciler) validateRig(ctx context.Context, rigRef string) (bool, error) {
	// If GTClient is available, use it
	if r.GTClient != nil {
		exists, err := r.GTClient.RigExists(ctx, rigRef)
		if err != nil {
			return false, gterrors.Wrap(err, "failed to check if rig exists via GT client")
		}
		return exists, nil
	}

	// Otherwise, try to fetch the Rig CRD from Kubernetes
	var rig gastownv1alpha1.Rig
	err := r.Get(ctx, client.ObjectKey{Name: rigRef}, &rig)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, gterrors.Wrap(err, "failed to fetch Rig CRD")
	}
	return true, nil
}

// syncBeadsFromGit reads beads from the git repository
func (r *BeadStoreReconciler) syncBeadsFromGit(ctx context.Context, rigRef string, rigStatus *gt.RigStatus) (int, error) {
	log := logf.FromContext(ctx)

	// For now, return the OpenBeads count from rig status if available
	// In a full implementation, this would:
	// 1. Clone/fetch the git repository
	// 2. Parse the .beads/issues.jsonl file
	// 3. Count and validate issues with the correct prefix
	// 4. Update internal caches as needed

	if rigStatus == nil {
		return 0, gterrors.New("rig status is nil")
	}

	issueCount := rigStatus.OpenBeads
	log.V(1).Info("Synced beads from git",
		"rig", rigRef,
		"issueCount", issueCount)

	return issueCount, nil
}

// setCondition updates a condition in the BeadStore status
func (r *BeadStoreReconciler) setCondition(beadstore *gastownv1alpha1.BeadStore, condType string, status metav1.ConditionStatus, reason, message string) {
	condition := metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: beadstore.Generation,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	// Find existing condition
	for i, c := range beadstore.Status.Conditions {
		if c.Type == condType {
			if c.Status != status || c.Reason != reason {
				beadstore.Status.Conditions[i] = condition
			}
			return
		}
	}

	// Add new condition if not found
	beadstore.Status.Conditions = append(beadstore.Status.Conditions, condition)
}

// SetupWithManager sets up the controller with the Manager.
func (r *BeadStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gastownv1alpha1.BeadStore{}).
		Owns(&corev1.Secret{}).
		Named("beadstore").
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1, // BeadStore is a singleton config
		}).
		Complete(r)
}
