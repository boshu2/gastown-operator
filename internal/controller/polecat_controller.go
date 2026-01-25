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

	corev1 "k8s.io/api/core/v1"
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
	"github.com/org/gastown-operator/pkg/pod"
)

const (
	// PolecatSyncInterval is how often we re-sync Pod status.
	// Uses RequeueShort for active polecat monitoring.
	PolecatSyncInterval = RequeueShort

	// Condition types for Polecat
	ConditionPolecatReady   = "Ready"
	ConditionPolecatWorking = "Working"

	// polecatFinalizer ensures cleanup of Pod resources
	polecatFinalizer = "gastown.io/polecat-cleanup"
)

// PolecatReconciler reconciles a Polecat object.
// Polecats run as Pods in the cluster.
type PolecatReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=gastown.gastown.io,resources=polecats,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=polecats/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=polecats/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

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

	// Handle deletion with finalizer
	if !polecat.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &polecat, timer)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&polecat, polecatFinalizer) {
		log.Info("Adding finalizer to Polecat")
		controllerutil.AddFinalizer(&polecat, polecatFinalizer)
		if err := r.Update(ctx, &polecat); err != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(err, "failed to add finalizer")
		}
		// Requeue immediately to continue reconciliation
		return ctrl.Result{RequeueAfter: time.Millisecond}, nil
	}

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

// ensureWorking ensures the polecat Pod is running.
func (r *PolecatReconciler) ensureWorking(ctx context.Context, polecat *gastownv1alpha1.Polecat, timer *metrics.ReconcileTimer) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Validate kubernetes spec is present
	if polecat.Spec.Kubernetes == nil {
		r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionFalse, "MissingKubernetesSpec",
			"kubernetes spec is required")
		polecat.Status.Phase = gastownv1alpha1.PolecatPhaseStuck
		if err := r.Status().Update(ctx, polecat); err != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(err, "failed to update status")
		}
		timer.RecordResult(metrics.ResultRequeue)
		return ctrl.Result{RequeueAfter: RequeueLong}, nil
	}

	podName := fmt.Sprintf("polecat-%s", polecat.Name)

	// Check if Pod already exists
	var existingPod corev1.Pod
	err := r.Get(ctx, client.ObjectKey{Name: podName, Namespace: polecat.Namespace}, &existingPod)
	if err == nil {
		// Pod exists, sync status from it
		return r.syncStatusFromPod(ctx, polecat, &existingPod, timer)
	}

	if !apierrors.IsNotFound(err) {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to get existing pod")
	}

	// Pod doesn't exist, create it
	log.Info("Creating Pod for Polecat",
		"podName", podName,
		"beadID", polecat.Spec.BeadID,
		"gitRepo", polecat.Spec.Kubernetes.GitRepository)

	builder := pod.NewBuilder(polecat)
	newPod, err := builder.Build()
	if err != nil {
		r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionFalse, "PodBuildFailed",
			err.Error())
		polecat.Status.Phase = gastownv1alpha1.PolecatPhaseStuck
		if updateErr := r.Status().Update(ctx, polecat); updateErr != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update status")
		}
		timer.RecordResult(metrics.ResultRequeue)
		return ctrl.Result{RequeueAfter: RequeueDefault}, nil
	}

	// Set owner reference for garbage collection
	if err := controllerutil.SetControllerReference(polecat, newPod, r.Scheme); err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to set owner reference")
	}

	if err := r.Create(ctx, newPod); err != nil {
		log.Error(err, "Failed to create Pod")
		r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionFalse, "PodCreateFailed",
			err.Error())
		polecat.Status.Phase = gastownv1alpha1.PolecatPhaseStuck
		if updateErr := r.Status().Update(ctx, polecat); updateErr != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update status")
		}
		timer.RecordResult(metrics.ResultRequeue)
		return ctrl.Result{RequeueAfter: RequeueDefault}, nil
	}

	// Update status with pod info
	polecat.Status.PodName = podName
	polecat.Status.Phase = gastownv1alpha1.PolecatPhaseWorking
	polecat.Status.AssignedBead = polecat.Spec.BeadID
	// Old conditions (backward compatibility)
	r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionTrue, "PodCreated",
		"Pod created successfully")
	r.setCondition(polecat, ConditionPolecatWorking, metav1.ConditionTrue, "Working",
		"Polecat is working on assigned bead")
	// New standard conditions
	r.setCondition(polecat, ConditionProgressing, metav1.ConditionTrue, "PodCreated",
		"Pod created, work starting")
	r.setCondition(polecat, ConditionAvailable, metav1.ConditionFalse, "NotReady",
		"Work in progress")
	r.setCondition(polecat, ConditionDegraded, metav1.ConditionFalse, "Healthy",
		"No issues detected")

	if err := r.Status().Update(ctx, polecat); err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to update status")
	}

	log.Info("Pod created for Polecat", "podName", podName)
	timer.RecordResult(metrics.ResultSuccess)
	return ctrl.Result{RequeueAfter: PolecatSyncInterval}, nil
}

// syncStatusFromPod updates Polecat status based on Pod status.
// Sets both old conditions (Ready, Working) and new standard conditions (Available, Progressing, Degraded)
// during the transition period. Witness and Refinery look for the new conditions.
func (r *PolecatReconciler) syncStatusFromPod(ctx context.Context, polecat *gastownv1alpha1.Polecat, p *corev1.Pod, timer *metrics.ReconcileTimer) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	polecat.Status.PodName = p.Name
	polecat.Status.AssignedBead = polecat.Spec.BeadID

	// Map Pod phase to Polecat phase and set conditions.
	// We set BOTH old conditions (Ready, Working) and new standard conditions
	// (Available, Progressing, Degraded) for backward compatibility during transition.
	switch p.Status.Phase {
	case corev1.PodPending:
		polecat.Status.Phase = gastownv1alpha1.PolecatPhaseWorking
		polecat.Status.PodActive = false
		// Old conditions (backward compatibility)
		r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionTrue, "PodPending",
			"Pod is pending")
		// New standard conditions
		r.setCondition(polecat, ConditionProgressing, metav1.ConditionTrue, "PodPending",
			"Pod is pending")
		r.setCondition(polecat, ConditionAvailable, metav1.ConditionFalse, "NotReady",
			"Work in progress")
		r.setCondition(polecat, ConditionDegraded, metav1.ConditionFalse, "Healthy",
			"No issues detected")
	case corev1.PodRunning:
		polecat.Status.Phase = gastownv1alpha1.PolecatPhaseWorking
		polecat.Status.PodActive = true
		// Old conditions (backward compatibility)
		r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionTrue, "PodRunning",
			"Pod is running")
		r.setCondition(polecat, ConditionPolecatWorking, metav1.ConditionTrue, "Working",
			"Agent is working")
		// New standard conditions
		r.setCondition(polecat, ConditionProgressing, metav1.ConditionTrue, "PodRunning",
			"Pod is running")
		r.setCondition(polecat, ConditionAvailable, metav1.ConditionFalse, "NotReady",
			"Work in progress")
		r.setCondition(polecat, ConditionDegraded, metav1.ConditionFalse, "Healthy",
			"No issues detected")
	case corev1.PodSucceeded:
		polecat.Status.Phase = gastownv1alpha1.PolecatPhaseDone
		polecat.Status.PodActive = false
		// Old conditions (backward compatibility)
		r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionTrue, "PodSucceeded",
			"Pod completed successfully")
		r.setCondition(polecat, ConditionPolecatWorking, metav1.ConditionFalse, "Completed",
			"Work completed")
		// New standard conditions - Available=True signals merge readiness
		r.setCondition(polecat, ConditionProgressing, metav1.ConditionFalse, "Completed",
			"Work completed")
		r.setCondition(polecat, ConditionAvailable, metav1.ConditionTrue, "WorkComplete",
			"Work complete, ready for merge")
		r.setCondition(polecat, ConditionDegraded, metav1.ConditionFalse, "Healthy",
			"No issues detected")
	case corev1.PodFailed:
		polecat.Status.Phase = gastownv1alpha1.PolecatPhaseStuck
		polecat.Status.PodActive = false
		// Old conditions (backward compatibility)
		r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionFalse, "PodFailed",
			"Pod failed")
		r.setCondition(polecat, ConditionPolecatWorking, metav1.ConditionFalse, "Failed",
			"Work failed")
		// New standard conditions - Degraded=True signals failure
		r.setCondition(polecat, ConditionProgressing, metav1.ConditionFalse, "Failed",
			"Work failed")
		r.setCondition(polecat, ConditionAvailable, metav1.ConditionFalse, "Failed",
			"Work failed")
		r.setCondition(polecat, ConditionDegraded, metav1.ConditionTrue, "PodFailed",
			"Pod failed")
	}

	// Update last activity from Pod start time
	if p.Status.StartTime != nil {
		polecat.Status.LastActivity = p.Status.StartTime
	}

	if err := r.Status().Update(ctx, polecat); err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to update status")
	}

	log.Info("Synced status from Pod",
		"podName", p.Name,
		"podPhase", p.Status.Phase,
		"polecatPhase", polecat.Status.Phase)

	timer.RecordResult(metrics.ResultSuccess)

	// Don't requeue if pod is done
	if p.Status.Phase == corev1.PodSucceeded || p.Status.Phase == corev1.PodFailed {
		return ctrl.Result{}, nil
	}

	return ctrl.Result{RequeueAfter: PolecatSyncInterval}, nil
}

// ensureIdle ensures the polecat is in idle state (no Pod running).
func (r *PolecatReconciler) ensureIdle(ctx context.Context, polecat *gastownv1alpha1.Polecat, timer *metrics.ReconcileTimer) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	podName := fmt.Sprintf("polecat-%s", polecat.Name)

	// Check if Pod exists and delete it if so
	var existingPod corev1.Pod
	err := r.Get(ctx, client.ObjectKey{Name: podName, Namespace: polecat.Namespace}, &existingPod)
	if err == nil {
		// Pod exists, delete it
		log.Info("Deleting Pod to transition to idle", "podName", podName)
		if err := r.Delete(ctx, &existingPod); err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, "Failed to delete Pod")
			timer.RecordResult(metrics.ResultRequeue)
			return ctrl.Result{RequeueAfter: RequeueDefault}, nil
		}
	} else if !apierrors.IsNotFound(err) {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to check pod existence")
	}

	// Update status to idle
	polecat.Status.Phase = gastownv1alpha1.PolecatPhaseIdle
	polecat.Status.PodActive = false
	polecat.Status.PodName = ""
	polecat.Status.AssignedBead = ""
	// Old conditions (backward compatibility)
	r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionTrue, "Idle",
		"Polecat is idle and ready for work")
	r.setCondition(polecat, ConditionPolecatWorking, metav1.ConditionFalse, "Idle",
		"No work assigned")
	// New standard conditions
	r.setCondition(polecat, ConditionProgressing, metav1.ConditionFalse, "Idle",
		"No work assigned")
	r.setCondition(polecat, ConditionAvailable, metav1.ConditionFalse, "Idle",
		"Idle, no work to merge")
	r.setCondition(polecat, ConditionDegraded, metav1.ConditionFalse, "Healthy",
		"No issues detected")

	if err := r.Status().Update(ctx, polecat); err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to update polecat status")
	}

	log.Info("Polecat is idle")
	timer.RecordResult(metrics.ResultSuccess)
	return ctrl.Result{}, nil
}

// ensureTerminated ensures the polecat is terminated.
func (r *PolecatReconciler) ensureTerminated(ctx context.Context, polecat *gastownv1alpha1.Polecat, timer *metrics.ReconcileTimer) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	podName := fmt.Sprintf("polecat-%s", polecat.Name)

	// Check if Pod exists
	var existingPod corev1.Pod
	err := r.Get(ctx, client.ObjectKey{Name: podName, Namespace: polecat.Namespace}, &existingPod)
	if err == nil {
		// Pod exists, delete it
		log.Info("Deleting Pod for terminated Polecat", "podName", podName)
		if err := r.Delete(ctx, &existingPod); err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, "Failed to delete Pod")
			r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionFalse, "PodDeleteFailed",
				err.Error())
			if updateErr := r.Status().Update(ctx, polecat); updateErr != nil {
				timer.RecordResult(metrics.ResultError)
				return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update status")
			}
			timer.RecordResult(metrics.ResultRequeue)
			return ctrl.Result{RequeueAfter: RequeueDefault}, nil
		}
	} else if !apierrors.IsNotFound(err) {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to check pod existence")
	}

	// Update status to terminated
	polecat.Status.Phase = gastownv1alpha1.PolecatPhaseTerminated
	polecat.Status.PodActive = false
	polecat.Status.PodName = ""
	// Old conditions (backward compatibility)
	r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionTrue, "Terminated",
		"Polecat has been terminated")
	// New standard conditions
	r.setCondition(polecat, ConditionProgressing, metav1.ConditionFalse, "Terminated",
		"Polecat terminated")
	r.setCondition(polecat, ConditionAvailable, metav1.ConditionFalse, "Terminated",
		"Polecat terminated")
	r.setCondition(polecat, ConditionDegraded, metav1.ConditionFalse, "Terminated",
		"Polecat terminated gracefully")

	if err := r.Status().Update(ctx, polecat); err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to update status")
	}

	log.Info("Polecat terminated")
	timer.RecordResult(metrics.ResultSuccess)
	return ctrl.Result{}, nil
}

// setCondition sets or updates a condition on the Polecat using the standard meta.SetStatusCondition helper.
func (r *PolecatReconciler) setCondition(polecat *gastownv1alpha1.Polecat, condType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&polecat.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: polecat.Generation,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

// handleDeletion handles cleanup when a Polecat is being deleted.
func (r *PolecatReconciler) handleDeletion(ctx context.Context, polecat *gastownv1alpha1.Polecat, timer *metrics.ReconcileTimer) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(polecat, polecatFinalizer) {
		// Finalizer already removed, nothing to do
		return ctrl.Result{}, nil
	}

	log.Info("Handling Polecat deletion, cleaning up resources")

	// Cleanup Pod
	if err := r.cleanupPod(ctx, polecat); err != nil {
		log.Error(err, "Failed to cleanup Polecat Pod")
		timer.RecordResult(metrics.ResultRequeue)
		return ctrl.Result{RequeueAfter: RequeueDefault}, nil
	}

	// Remove finalizer after successful cleanup
	log.Info("Cleanup complete, removing finalizer")
	controllerutil.RemoveFinalizer(polecat, polecatFinalizer)
	if err := r.Update(ctx, polecat); err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to remove finalizer")
	}

	timer.RecordResult(metrics.ResultSuccess)
	return ctrl.Result{}, nil
}

// cleanupPod deletes the Pod for a Polecat.
func (r *PolecatReconciler) cleanupPod(ctx context.Context, polecat *gastownv1alpha1.Polecat) error {
	log := logf.FromContext(ctx)
	podName := fmt.Sprintf("polecat-%s", polecat.Name)

	var existingPod corev1.Pod
	err := r.Get(ctx, client.ObjectKey{Name: podName, Namespace: polecat.Namespace}, &existingPod)
	if apierrors.IsNotFound(err) {
		log.Info("Pod already deleted", "podName", podName)
		return nil
	}
	if err != nil {
		return gterrors.Wrap(err, "failed to get pod")
	}

	log.Info("Deleting Pod for Polecat cleanup", "podName", podName)
	if err := r.Delete(ctx, &existingPod); err != nil && !apierrors.IsNotFound(err) {
		return gterrors.Wrap(err, "failed to delete pod")
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PolecatReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gastownv1alpha1.Polecat{}).
		Named("polecat").
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 5, // Limit concurrent pod creations
		}).
		Complete(r)
}
