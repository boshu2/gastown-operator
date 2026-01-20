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
	"github.com/org/gastown-operator/pkg/gt"
	"github.com/org/gastown-operator/pkg/metrics"
	"github.com/org/gastown-operator/pkg/pod"
)

const (
	// PolecatSyncInterval is how often we re-sync with gt CLI.
	// Uses RequeueShort for active polecat monitoring (faster than other controllers).
	PolecatSyncInterval = RequeueShort

	// Condition types for Polecat
	ConditionPolecatReady   = "Ready"
	ConditionPolecatWorking = "Working"

	// polecatFinalizer ensures cleanup of external resources
	polecatFinalizer = "gastown.io/polecat-cleanup"
)

// PolecatReconciler reconciles a Polecat object
type PolecatReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	GTClient gt.ClientInterface
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

// ensureWorking ensures the polecat is working on a bead.
func (r *PolecatReconciler) ensureWorking(ctx context.Context, polecat *gastownv1alpha1.Polecat, timer *metrics.ReconcileTimer) (ctrl.Result, error) {
	// Route based on execution mode
	if polecat.Spec.ExecutionMode == gastownv1alpha1.ExecutionModeKubernetes {
		return r.ensureWorkingKubernetes(ctx, polecat, timer)
	}
	return r.ensureWorkingLocal(ctx, polecat, timer)
}

// ensureWorkingKubernetes handles kubernetes execution mode
func (r *PolecatReconciler) ensureWorkingKubernetes(ctx context.Context, polecat *gastownv1alpha1.Polecat, timer *metrics.ReconcileTimer) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Validate kubernetes spec is present
	if polecat.Spec.Kubernetes == nil {
		r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionFalse, "MissingKubernetesSpec",
			"kubernetes spec is required when executionMode is kubernetes")
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
	r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionTrue, "PodCreated",
		"Pod created successfully")
	r.setCondition(polecat, ConditionPolecatWorking, metav1.ConditionTrue, "Working",
		"Polecat is working on assigned bead")

	if err := r.Status().Update(ctx, polecat); err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to update status")
	}

	log.Info("Pod created for Polecat", "podName", podName)
	timer.RecordResult(metrics.ResultSuccess)
	return ctrl.Result{RequeueAfter: PolecatSyncInterval}, nil
}

// syncStatusFromPod updates Polecat status based on Pod status
func (r *PolecatReconciler) syncStatusFromPod(ctx context.Context, polecat *gastownv1alpha1.Polecat, p *corev1.Pod, timer *metrics.ReconcileTimer) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	polecat.Status.PodName = p.Name
	polecat.Status.AssignedBead = polecat.Spec.BeadID

	// Map Pod phase to Polecat phase
	switch p.Status.Phase {
	case corev1.PodPending:
		polecat.Status.Phase = gastownv1alpha1.PolecatPhaseWorking
		r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionTrue, "PodPending",
			"Pod is pending")
	case corev1.PodRunning:
		polecat.Status.Phase = gastownv1alpha1.PolecatPhaseWorking
		polecat.Status.SessionActive = true
		r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionTrue, "PodRunning",
			"Pod is running")
		r.setCondition(polecat, ConditionPolecatWorking, metav1.ConditionTrue, "Working",
			"Agent is working")
	case corev1.PodSucceeded:
		polecat.Status.Phase = gastownv1alpha1.PolecatPhaseDone
		polecat.Status.SessionActive = false
		r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionTrue, "PodSucceeded",
			"Pod completed successfully")
		r.setCondition(polecat, ConditionPolecatWorking, metav1.ConditionFalse, "Completed",
			"Work completed")
	case corev1.PodFailed:
		polecat.Status.Phase = gastownv1alpha1.PolecatPhaseStuck
		polecat.Status.SessionActive = false
		r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionFalse, "PodFailed",
			"Pod failed")
		r.setCondition(polecat, ConditionPolecatWorking, metav1.ConditionFalse, "Failed",
			"Work failed")
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

// ensureWorkingLocal handles local execution mode via gt CLI
func (r *PolecatReconciler) ensureWorkingLocal(ctx context.Context, polecat *gastownv1alpha1.Polecat, timer *metrics.ReconcileTimer) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Check if polecat exists with timeout
	gtCtx, cancel := WithGTClientTimeout(ctx)
	exists, err := r.GTClient.PolecatExists(gtCtx, polecat.Spec.Rig, polecat.Name)
	cancel()
	if err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to check polecat existence")
	}

	// If polecat doesn't exist and we have a bead to work on, create via sling
	if !exists && polecat.Spec.BeadID != "" {
		log.Info("Creating polecat via gt sling",
			"beadID", polecat.Spec.BeadID,
			"rig", polecat.Spec.Rig)

		gtCtx, cancel := WithGTClientTimeout(ctx)
		err := r.GTClient.Sling(gtCtx, polecat.Spec.BeadID, polecat.Spec.Rig)
		cancel()
		if err != nil {
			r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionFalse, "SlingFailed",
				err.Error())
			polecat.Status.Phase = gastownv1alpha1.PolecatPhaseStuck

			if updateErr := r.Status().Update(ctx, polecat); updateErr != nil {
				timer.RecordResult(metrics.ResultError)
				return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update polecat status")
			}

			timer.RecordResult(metrics.ResultRequeue)
			return ctrl.Result{RequeueAfter: RequeueDefault}, nil
		}
	}

	// Sync status from gt CLI with timeout
	gtCtx, cancel = WithGTClientTimeout(ctx)
	status, err := r.GTClient.PolecatStatus(gtCtx, polecat.Spec.Rig, polecat.Name)
	cancel()
	if err != nil {
		log.Error(err, "Failed to get polecat status from gt CLI")
		r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionFalse, "GTCLIError",
			err.Error())

		if updateErr := r.Status().Update(ctx, polecat); updateErr != nil {
			timer.RecordResult(metrics.ResultError)
			return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update polecat status")
		}

		timer.RecordResult(metrics.ResultRequeue)
		return ctrl.Result{RequeueAfter: RequeueDefault}, nil
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

	// Check if polecat exists and has work with timeout
	gtCtx, cancel := WithGTClientTimeout(ctx)
	exists, err := r.GTClient.PolecatExists(gtCtx, polecat.Spec.Rig, polecat.Name)
	cancel()
	if err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to check polecat existence")
	}

	if exists {
		// Get current status with timeout
		gtCtx, cancel := WithGTClientTimeout(ctx)
		status, err := r.GTClient.PolecatStatus(gtCtx, polecat.Spec.Rig, polecat.Name)
		cancel()
		if err != nil {
			timer.RecordResult(metrics.ResultRequeue)
			return ctrl.Result{RequeueAfter: RequeueDefault}, nil
		}

		// If polecat is working, reset it to idle
		if status.Phase == "Working" || status.AssignedBead != "" {
			log.Info("Resetting polecat to idle")
			gtCtx, cancel := WithGTClientTimeout(ctx)
			err := r.GTClient.PolecatReset(gtCtx, polecat.Spec.Rig, polecat.Name)
			cancel()
			if err != nil {
				log.Error(err, "Failed to reset polecat")
				r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionFalse, "ResetFailed",
					err.Error())

				if updateErr := r.Status().Update(ctx, polecat); updateErr != nil {
					timer.RecordResult(metrics.ResultError)
					return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update polecat status")
				}

				timer.RecordResult(metrics.ResultRequeue)
				return ctrl.Result{RequeueAfter: RequeueDefault}, nil
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
	// Route based on execution mode
	if polecat.Spec.ExecutionMode == gastownv1alpha1.ExecutionModeKubernetes {
		return r.ensureTerminatedKubernetes(ctx, polecat, timer)
	}
	return r.ensureTerminatedLocal(ctx, polecat, timer)
}

// ensureTerminatedKubernetes handles kubernetes termination
func (r *PolecatReconciler) ensureTerminatedKubernetes(ctx context.Context, polecat *gastownv1alpha1.Polecat, timer *metrics.ReconcileTimer) (ctrl.Result, error) {
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
	polecat.Status.SessionActive = false
	polecat.Status.PodName = ""
	r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionTrue, "Terminated",
		"Polecat has been terminated")

	if err := r.Status().Update(ctx, polecat); err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to update status")
	}

	log.Info("Polecat terminated (kubernetes mode)")
	timer.RecordResult(metrics.ResultSuccess)
	return ctrl.Result{}, nil
}

// ensureTerminatedLocal handles local termination via gt CLI
func (r *PolecatReconciler) ensureTerminatedLocal(ctx context.Context, polecat *gastownv1alpha1.Polecat, timer *metrics.ReconcileTimer) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Check if polecat exists with timeout
	gtCtx, cancel := WithGTClientTimeout(ctx)
	exists, err := r.GTClient.PolecatExists(gtCtx, polecat.Spec.Rig, polecat.Name)
	cancel()
	if err != nil {
		timer.RecordResult(metrics.ResultError)
		return ctrl.Result{}, gterrors.Wrap(err, "failed to check polecat existence")
	}

	if exists {
		// Get current status to check for uncommitted work with timeout
		gtCtx, cancel := WithGTClientTimeout(ctx)
		status, err := r.GTClient.PolecatStatus(gtCtx, polecat.Spec.Rig, polecat.Name)
		cancel()
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
			return ctrl.Result{RequeueAfter: RequeueLong}, nil
		}

		// Nuke the polecat with timeout
		log.Info("Terminating polecat")
		gtCtx, cancel = WithGTClientTimeout(ctx)
		err = r.GTClient.PolecatNuke(gtCtx, polecat.Spec.Rig, polecat.Name, false)
		cancel()
		if err != nil {
			log.Error(err, "Failed to nuke polecat")
			r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionFalse, "NukeFailed",
				err.Error())

			if updateErr := r.Status().Update(ctx, polecat); updateErr != nil {
				timer.RecordResult(metrics.ResultError)
				return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update polecat status")
			}

			timer.RecordResult(metrics.ResultRequeue)
			return ctrl.Result{RequeueAfter: RequeueDefault}, nil
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

	// Route cleanup based on execution mode
	var cleanupErr error
	if polecat.Spec.ExecutionMode == gastownv1alpha1.ExecutionModeKubernetes {
		cleanupErr = r.cleanupKubernetes(ctx, polecat)
	} else {
		cleanupErr = r.cleanupLocal(ctx, polecat)
	}

	if cleanupErr != nil {
		log.Error(cleanupErr, "Failed to cleanup Polecat resources")
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

// cleanupKubernetes deletes the Pod for a kubernetes-mode Polecat.
func (r *PolecatReconciler) cleanupKubernetes(ctx context.Context, polecat *gastownv1alpha1.Polecat) error {
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

// cleanupLocal nukes the polecat via gt CLI.
func (r *PolecatReconciler) cleanupLocal(ctx context.Context, polecat *gastownv1alpha1.Polecat) error {
	log := logf.FromContext(ctx)

	gtCtx, cancel := WithGTClientTimeout(ctx)
	exists, err := r.GTClient.PolecatExists(gtCtx, polecat.Spec.Rig, polecat.Name)
	cancel()
	if err != nil {
		return gterrors.Wrap(err, "failed to check polecat existence")
	}

	if !exists {
		log.Info("Polecat already cleaned up")
		return nil
	}

	log.Info("Nuking polecat via gt CLI", "rig", polecat.Spec.Rig, "name", polecat.Name)
	// Force nuke on deletion - we're deleting the resource anyway
	gtCtx, cancel = WithGTClientTimeout(ctx)
	err = r.GTClient.PolecatNuke(gtCtx, polecat.Spec.Rig, polecat.Name, true)
	cancel()
	if err != nil {
		return gterrors.Wrap(err, "failed to nuke polecat")
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
