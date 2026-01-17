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
	// RefineryConditionReady indicates the refinery is ready to process merges.
	RefineryConditionReady = "Ready"

	// RefineryConditionProcessing indicates a merge is in progress.
	RefineryConditionProcessing = "Processing"

	// Default requeue interval for idle refinery.
	refineryIdleRequeueInterval = 30 * time.Second

	// Requeue interval during active processing.
	refineryProcessingRequeueInterval = 5 * time.Second
)

// RefineryReconciler reconciles a Refinery object
type RefineryReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=gastown.gastown.io,resources=refineries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=refineries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=refineries/finalizers,verbs=update
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=polecats,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile processes the merge queue for the Refinery's Rig.
func (r *RefineryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Refinery instance
	refinery := &gastownv1alpha1.Refinery{}
	if err := r.Get(ctx, req.NamespacedName, refinery); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Reconciling Refinery", "rigRef", refinery.Spec.RigRef)

	// List Polecats in the namespace that belong to this rig and are ready for merge
	polecatList := &gastownv1alpha1.PolecatList{}
	listOpts := []client.ListOption{
		client.InNamespace(req.Namespace),
		client.MatchingLabels{"gastown.io/rig": refinery.Spec.RigRef},
	}
	if err := r.List(ctx, polecatList, listOpts...); err != nil {
		log.Error(err, "Failed to list Polecats")
		r.setCondition(refinery, RefineryConditionReady, metav1.ConditionFalse,
			"ListFailed", "Failed to list Polecats")
		return ctrl.Result{RequeueAfter: refineryIdleRequeueInterval}, r.Status().Update(ctx, refinery)
	}

	// Find polecats that are ready for merge
	mergeQueue := r.findMergeReadyPolecats(polecatList)

	// Update queue statistics
	refinery.Status.QueueLength = int32(len(mergeQueue))
	refinery.Status.MergesSummary.Pending = int32(len(mergeQueue))

	// If no work, mark as Idle
	if len(mergeQueue) == 0 {
		refinery.Status.Phase = "Idle"
		refinery.Status.CurrentMerge = ""
		r.setCondition(refinery, RefineryConditionReady, metav1.ConditionTrue,
			"Idle", "No merges pending")

		if err := r.Status().Update(ctx, refinery); err != nil {
			log.Error(err, "Failed to update Refinery status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: refineryIdleRequeueInterval}, nil
	}

	// Process the first item in queue (sequential processing)
	if refinery.Spec.Parallelism <= 1 && len(mergeQueue) > 0 {
		targetPolecat := mergeQueue[0]
		refinery.Status.Phase = "Processing"
		refinery.Status.CurrentMerge = targetPolecat.Name

		r.setCondition(refinery, RefineryConditionProcessing, metav1.ConditionTrue,
			"Processing", "Processing merge for "+targetPolecat.Name)

		// Process the merge
		if err := r.processMerge(ctx, refinery, &targetPolecat); err != nil {
			log.Error(err, "Failed to process merge", "polecat", targetPolecat.Name)
			refinery.Status.MergesSummary.Failed++
			r.Recorder.Event(refinery, "Warning", "MergeFailed",
				"Merge failed for "+targetPolecat.Name+": "+err.Error())
		} else {
			refinery.Status.MergesSummary.Succeeded++
			refinery.Status.MergesSummary.Total++
			refinery.Status.LastMergeTime = &metav1.Time{Time: time.Now()}
			r.Recorder.Event(refinery, "Normal", "MergeSucceeded",
				"Successfully merged "+targetPolecat.Name)
		}
	}

	// Update status
	if err := r.Status().Update(ctx, refinery); err != nil {
		log.Error(err, "Failed to update Refinery status")
		return ctrl.Result{}, err
	}

	log.Info("Refinery reconciliation complete",
		"phase", refinery.Status.Phase,
		"queueLength", refinery.Status.QueueLength,
		"succeeded", refinery.Status.MergesSummary.Succeeded,
		"failed", refinery.Status.MergesSummary.Failed)

	// Requeue quickly if there's work to do
	if refinery.Status.QueueLength > 0 {
		return ctrl.Result{RequeueAfter: refineryProcessingRequeueInterval}, nil
	}
	return ctrl.Result{RequeueAfter: refineryIdleRequeueInterval}, nil
}

// findMergeReadyPolecats finds polecats that have completed successfully and are ready for merge.
func (r *RefineryReconciler) findMergeReadyPolecats(polecats *gastownv1alpha1.PolecatList) []gastownv1alpha1.Polecat {
	var ready []gastownv1alpha1.Polecat

	for _, polecat := range polecats.Items {
		// Check for Available condition (succeeded)
		for _, cond := range polecat.Status.Conditions {
			if cond.Type == "Available" && cond.Status == metav1.ConditionTrue {
				ready = append(ready, polecat)
				break
			}
		}
	}

	return ready
}

// processMerge handles the merge workflow for a single polecat branch.
// STUB: This is a simulated implementation. Real git operations are not performed.
// TODO(he-xxxx): Implement actual git merge workflow:
//  1. Clone/fetch the repository using GitSecretRef credentials
//  2. Checkout the target branch
//  3. Rebase the polecat branch onto target
//  4. Run tests if TestCommand is configured
//  5. Push to target branch
//  6. Clean up polecat branch
func (r *RefineryReconciler) processMerge(ctx context.Context, refinery *gastownv1alpha1.Refinery, polecat *gastownv1alpha1.Polecat) error {
	log := logf.FromContext(ctx)

	// STUB: Simulate success without actual git operations
	log.Info("Processing merge (STUB - simulated, no actual git operations)",
		"polecat", polecat.Name,
		"targetBranch", refinery.Spec.TargetBranch,
		"testCommand", refinery.Spec.TestCommand)

	// Update polecat status to indicate merge complete
	// In production, this would be done after actual git operations
	meta.SetStatusCondition(&polecat.Status.Conditions, metav1.Condition{
		Type:               "Merged",
		Status:             metav1.ConditionTrue,
		Reason:             "MergeComplete",
		Message:            "Branch merged to " + refinery.Spec.TargetBranch,
		LastTransitionTime: metav1.Now(),
	})

	if err := r.Status().Update(ctx, polecat); err != nil {
		return err
	}

	return nil
}

// setCondition updates or adds a condition to the Refinery status.
func (r *RefineryReconciler) setCondition(refinery *gastownv1alpha1.Refinery, condType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&refinery.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *RefineryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gastownv1alpha1.Refinery{}).
		Named("refinery").
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 2, // Merges should be serialized per rig anyway
		}).
		Complete(r)
}
