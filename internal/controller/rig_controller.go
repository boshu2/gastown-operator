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
	"os"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
	gterrors "github.com/org/gastown-operator/pkg/errors"
	"github.com/org/gastown-operator/pkg/metrics"
)

const (
	// RigSyncInterval is how often we re-sync rig status.
	// Uses RequeueDefault for normal sync operations.
	RigSyncInterval = RequeueDefault

	// Condition types for Rig.
	// See constants.go for the condition naming convention (unprefixed values).

	// ConditionRigReady uses the standard Ready condition.
	ConditionRigReady = ConditionReady

	// rigFinalizer ensures cleanup of child resources (Witness, Refinery)
	rigFinalizer = "gastown.io/rig-cleanup"

	// defaultChildNamespace is where Witness/Refinery CRs are created
	// Can be overridden by GASTOWN_NAMESPACE env var
	defaultChildNamespace = "gastown-system"
)

// RigReconciler reconciles a Rig object.
// It aggregates status from child Polecats and Convoys.
type RigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=gastown.gastown.io,resources=rigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=rigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=rigs/finalizers,verbs=update
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=polecats,verbs=get;list;watch
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=convoys,verbs=get;list;watch
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=witnesses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=refineries,verbs=get;list;watch;create;update;patch;delete

// Reconcile aggregates status from Polecats and Convoys in the Rig.
// It also auto-provisions Witness and Refinery CRs when a Rig is created.
func (r *RigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	timer := metrics.NewReconcileTimer("rig")
	defer timer.ObserveDuration()

	// Fetch the Rig instance
	var rig gastownv1alpha1.Rig
	if err := r.Get(ctx, req.NamespacedName, &rig); err != nil {
		// Ignore not-found errors (deleted before reconcile)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Reconciling Rig", "name", rig.Name)

	// Handle deletion with finalizer
	if !rig.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &rig, timer)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&rig, rigFinalizer) {
		log.Info("Adding finalizer to Rig")
		controllerutil.AddFinalizer(&rig, rigFinalizer)
		if err := r.Update(ctx, &rig); err != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(err, "failed to add finalizer")
		}
		// Requeue to continue reconciliation
		return ctrl.Result{Requeue: true}, nil
	}

	// Ensure Witness and Refinery children exist
	if err := r.ensureChildren(ctx, &rig); err != nil {
		log.Error(err, "Failed to ensure child resources")
		r.setCondition(&rig, ConditionRigReady, metav1.ConditionFalse, "ChildCreationFailed",
			err.Error())
		rig.Status.Phase = gastownv1alpha1.RigPhaseDegraded
		if updateErr := r.Status().Update(ctx, &rig); updateErr != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update rig status")
		}
		timer.RecordResult(metrics.ResultRequeue)
		return ctrl.Result{RequeueAfter: RequeueDefault}, nil
	}

	// Count polecats for this rig
	var polecatList gastownv1alpha1.PolecatList
	if err := r.List(ctx, &polecatList, client.MatchingFields{"spec.rig": rig.Name}); err != nil {
		log.Error(err, "Failed to list polecats for rig")
		r.setCondition(&rig, ConditionRigReady, metav1.ConditionFalse, "ListFailed",
			err.Error())
		rig.Status.Phase = gastownv1alpha1.RigPhaseDegraded

		if updateErr := r.Status().Update(ctx, &rig); updateErr != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update rig status")
		}

		timer.RecordResult(metrics.ResultRequeue)
		return ctrl.Result{RequeueAfter: RequeueDefault}, nil
	}

	// Count convoys for this rig
	var convoyList gastownv1alpha1.ConvoyList
	activeConvoys := 0
	if err := r.List(ctx, &convoyList); err != nil {
		log.Error(err, "Failed to list convoys")
	} else {
		for _, convoy := range convoyList.Items {
			if convoy.Status.Phase == gastownv1alpha1.ConvoyPhaseInProgress {
				activeConvoys++
			}
		}
	}

	// Update status
	rig.Status.Phase = gastownv1alpha1.RigPhaseReady
	rig.Status.PolecatCount = len(polecatList.Items)
	rig.Status.ActiveConvoys = activeConvoys

	r.setCondition(&rig, ConditionRigReady, metav1.ConditionTrue, "Ready",
		"Rig is ready")

	if err := r.Status().Update(ctx, &rig); err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to update rig status")
	}

	log.Info("Rig reconciled successfully",
		"polecats", rig.Status.PolecatCount,
		"convoys", rig.Status.ActiveConvoys)

	timer.RecordResult(metrics.ResultSuccess)
	return ctrl.Result{RequeueAfter: RigSyncInterval}, nil
}

// ensureChildren creates Witness and Refinery CRs for the Rig if they don't exist.
// This is the key auto-provisioning logic: creating a Rig gives you full Gas Town functionality.
func (r *RigReconciler) ensureChildren(ctx context.Context, rig *gastownv1alpha1.Rig) error {
	log := logf.FromContext(ctx)
	ns := r.getChildNamespace()

	// Track if we made changes
	statusChanged := false

	// Set child namespace in status if not set
	if rig.Status.ChildNamespace == "" {
		rig.Status.ChildNamespace = ns
		statusChanged = true
	}

	// Ensure Witness
	if !rig.Status.WitnessCreated {
		witnessName := rig.Name + "-witness"
		witness := &gastownv1alpha1.Witness{
			ObjectMeta: metav1.ObjectMeta{
				Name:      witnessName,
				Namespace: ns,
				Labels: map[string]string{
					"gastown.io/rig-owner":         rig.Name,
					"app.kubernetes.io/managed-by": "rig-controller",
				},
			},
			Spec: gastownv1alpha1.WitnessSpec{
				RigRef: rig.Name,
			},
		}

		// Note: Can't set cross-namespace owner reference for cluster-scoped Rig to namespace-scoped Witness
		// Use label gastown.io/rig-owner for GC lookup instead

		if err := r.Create(ctx, witness); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create Witness %s: %w", witnessName, err)
			}
			log.Info("Witness already exists", "name", witnessName)
		} else {
			log.Info("Created Witness for Rig", "witness", witnessName, "rig", rig.Name)
		}
		rig.Status.WitnessCreated = true
		statusChanged = true
	}

	// Ensure Refinery
	if !rig.Status.RefineryCreated {
		refineryName := rig.Name + "-refinery"
		refinery := &gastownv1alpha1.Refinery{
			ObjectMeta: metav1.ObjectMeta{
				Name:      refineryName,
				Namespace: ns,
				Labels: map[string]string{
					"gastown.io/rig-owner":         rig.Name,
					"app.kubernetes.io/managed-by": "rig-controller",
				},
			},
			Spec: gastownv1alpha1.RefinerySpec{
				RigRef:       rig.Name,
				TargetBranch: "main", // Default target branch
			},
		}

		if err := r.Create(ctx, refinery); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create Refinery %s: %w", refineryName, err)
			}
			log.Info("Refinery already exists", "name", refineryName)
		} else {
			log.Info("Created Refinery for Rig", "refinery", refineryName, "rig", rig.Name)
		}
		rig.Status.RefineryCreated = true
		statusChanged = true
	}

	// Update status if changed
	if statusChanged {
		if err := r.Status().Update(ctx, rig); err != nil {
			return fmt.Errorf("failed to update rig status after child creation: %w", err)
		}
	}

	return nil
}

// handleDeletion handles cleanup when a Rig is being deleted.
// It deletes the auto-provisioned Witness and Refinery CRs.
func (r *RigReconciler) handleDeletion(ctx context.Context, rig *gastownv1alpha1.Rig, timer *metrics.ReconcileTimer) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(rig, rigFinalizer) {
		// Finalizer already removed, nothing to do
		return ctrl.Result{}, nil
	}

	log.Info("Handling Rig deletion, cleaning up child resources", "rig", rig.Name)

	ns := rig.Status.ChildNamespace
	if ns == "" {
		ns = r.getChildNamespace()
	}

	// Delete Witness
	witnessName := rig.Name + "-witness"
	witness := &gastownv1alpha1.Witness{}
	if err := r.Get(ctx, client.ObjectKey{Name: witnessName, Namespace: ns}, witness); err == nil {
		log.Info("Deleting Witness", "name", witnessName)
		if err := r.Delete(ctx, witness); err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, "Failed to delete Witness", "name", witnessName)
			timer.RecordResult(metrics.ResultRequeue)
			return ctrl.Result{RequeueAfter: RequeueDefault}, nil
		}
	} else if !apierrors.IsNotFound(err) {
		log.Error(err, "Failed to get Witness for deletion", "name", witnessName)
	}

	// Delete Refinery
	refineryName := rig.Name + "-refinery"
	refinery := &gastownv1alpha1.Refinery{}
	if err := r.Get(ctx, client.ObjectKey{Name: refineryName, Namespace: ns}, refinery); err == nil {
		log.Info("Deleting Refinery", "name", refineryName)
		if err := r.Delete(ctx, refinery); err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, "Failed to delete Refinery", "name", refineryName)
			timer.RecordResult(metrics.ResultRequeue)
			return ctrl.Result{RequeueAfter: RequeueDefault}, nil
		}
	} else if !apierrors.IsNotFound(err) {
		log.Error(err, "Failed to get Refinery for deletion", "name", refineryName)
	}

	// Remove finalizer after successful cleanup
	log.Info("Cleanup complete, removing finalizer", "rig", rig.Name)
	controllerutil.RemoveFinalizer(rig, rigFinalizer)
	if err := r.Update(ctx, rig); err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to remove finalizer")
	}

	timer.RecordResult(metrics.ResultSuccess)
	return ctrl.Result{}, nil
}

// getChildNamespace returns the namespace where child resources should be created.
// Uses GASTOWN_NAMESPACE env var if set, otherwise defaults to gastown-system.
func (r *RigReconciler) getChildNamespace() string {
	if ns := os.Getenv("GASTOWN_NAMESPACE"); ns != "" {
		return ns
	}
	return defaultChildNamespace
}

// setCondition sets or updates a condition on the Rig using the standard meta.SetStatusCondition helper.
func (r *RigReconciler) setCondition(rig *gastownv1alpha1.Rig, condType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&rig.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: rig.Generation,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *RigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add index for looking up polecats by rig name
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &gastownv1alpha1.Polecat{}, "spec.rig", func(rawObj client.Object) []string {
		polecat := rawObj.(*gastownv1alpha1.Polecat)
		return []string{polecat.Spec.Rig}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&gastownv1alpha1.Rig{}).
		Named("rig").
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 3, // Rigs are cluster-scoped, limit concurrency
		}).
		Complete(r)
}
