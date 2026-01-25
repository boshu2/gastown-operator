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
	// RigSyncInterval is how often we re-sync rig status.
	// Uses RequeueDefault for normal sync operations.
	RigSyncInterval = RequeueDefault

	// Condition types for Rig.
	// See constants.go for the condition naming convention (unprefixed values).

	// ConditionRigReady uses the standard Ready condition.
	ConditionRigReady = ConditionReady
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

// Reconcile aggregates status from Polecats and Convoys in the Rig.
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
