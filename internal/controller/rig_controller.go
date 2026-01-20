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
	"os"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
	gterrors "github.com/org/gastown-operator/pkg/errors"
	"github.com/org/gastown-operator/pkg/gt"
	"github.com/org/gastown-operator/pkg/metrics"
)

const (
	// RigSyncInterval is how often we re-sync with gt CLI.
	// Uses RequeueDefault for normal sync operations.
	RigSyncInterval = RequeueDefault

	// Condition types for Rig.
	// See constants.go for the condition naming convention (unprefixed values).

	// ConditionRigExists indicates the rig's local path exists on the filesystem.
	// This is a prerequisite for the rig to be Ready.
	ConditionRigExists = "Exists"

	// ConditionRigReady uses the standard Ready condition.
	ConditionRigReady = ConditionReady
)

// RigReconciler reconciles a Rig object
type RigReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	GTClient gt.ClientInterface
}

// +kubebuilder:rbac:groups=gastown.gastown.io,resources=rigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=rigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=rigs/finalizers,verbs=update

// Reconcile syncs the Rig CRD with the actual rig state from gt CLI.
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

	// Check if rig path exists on filesystem
	if !r.rigPathExists(rig.Spec.LocalPath) {
		log.Info("Rig path does not exist", "path", rig.Spec.LocalPath)
		r.setCondition(&rig, ConditionRigExists, metav1.ConditionFalse, "PathNotFound",
			"Rig path does not exist on filesystem")
		rig.Status.Phase = gastownv1alpha1.RigPhaseDegraded

		if err := r.Status().Update(ctx, &rig); err != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(err, "failed to update rig status")
		}

		timer.RecordResult(metrics.ResultRequeue)
		return ctrl.Result{RequeueAfter: RequeueLong}, nil
	}

	r.setCondition(&rig, ConditionRigExists, metav1.ConditionTrue, "PathExists",
		"Rig path exists on filesystem")

	// Query gt CLI for rig status with timeout to prevent hung CLI from blocking
	gtCtx, cancel := WithGTClientTimeout(ctx)
	defer cancel()
	status, err := r.GTClient.RigStatus(gtCtx, rig.Name)
	if err != nil {
		log.Error(err, "Failed to get rig status from gt CLI")
		r.setCondition(&rig, ConditionRigReady, metav1.ConditionFalse, "GTCLIError",
			err.Error())
		rig.Status.Phase = gastownv1alpha1.RigPhaseDegraded

		if updateErr := r.Status().Update(ctx, &rig); updateErr != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update rig status")
		}

		timer.RecordResult(metrics.ResultRequeue)
		// Retry sooner for transient errors
		if gterrors.IsRetryable(err) {
			return ctrl.Result{RequeueAfter: RequeueRetryTransient}, nil
		}
		return ctrl.Result{RequeueAfter: RequeueLong}, nil
	}

	// Update status from gt CLI response
	rig.Status.Phase = gastownv1alpha1.RigPhaseReady
	rig.Status.PolecatCount = status.PolecatCount
	rig.Status.ActiveConvoys = status.ActiveConvoys
	now := metav1.Now()
	rig.Status.LastSyncTime = &now

	r.setCondition(&rig, ConditionRigReady, metav1.ConditionTrue, "Synced",
		"Successfully synced with gt CLI")

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

// rigPathExists checks if the rig path exists on the filesystem.
func (r *RigReconciler) rigPathExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
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
	return ctrl.NewControllerManagedBy(mgr).
		For(&gastownv1alpha1.Rig{}).
		Named("rig").
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 3, // Rigs are cluster-scoped, limit concurrency
		}).
		Complete(r)
}
