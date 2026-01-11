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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
	"github.com/org/gastown-operator/pkg/gt"
	"github.com/org/gastown-operator/pkg/metrics"
)

const (
	// BeadsSyncInterval is how often we poll for beads changes
	BeadsSyncInterval = 30 * time.Second
)

// BeadsSyncReconciler watches for beads changes and updates CRDs accordingly.
// This controller polls the beads system for status changes that occur
// outside of the K8s operator (e.g., direct gt CLI usage).
type BeadsSyncReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	GTClient *gt.Client
}

// +kubebuilder:rbac:groups=gastown.gastown.io,resources=polecats,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=convoys,verbs=get;list;watch;update;patch

// Reconcile syncs Polecat and Convoy CRDs with the beads system.
// This controller uses a periodic requeue rather than watching specific resources.
func (r *BeadsSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	timer := metrics.NewReconcileTimer("beadssync")
	defer timer.ObserveDuration()

	log.V(1).Info("Running beads sync")

	// Sync polecats
	if err := r.syncPolecats(ctx); err != nil {
		log.Error(err, "Failed to sync polecats")
		timer.RecordResult(metrics.ResultError)
	}

	// Sync convoys
	if err := r.syncConvoys(ctx); err != nil {
		log.Error(err, "Failed to sync convoys")
		timer.RecordResult(metrics.ResultError)
	}

	timer.RecordResult(metrics.ResultSuccess)
	return ctrl.Result{RequeueAfter: BeadsSyncInterval}, nil
}

// syncPolecats checks all polecats for status changes.
func (r *BeadsSyncReconciler) syncPolecats(ctx context.Context) error {
	log := logf.FromContext(ctx)

	var polecats gastownv1alpha1.PolecatList
	if err := r.List(ctx, &polecats); err != nil {
		return err
	}

	for _, polecat := range polecats.Items {
		// Skip if no assigned bead
		if polecat.Status.AssignedBead == "" {
			continue
		}

		// Check bead status
		beadStatus, err := r.GTClient.BeadStatus(ctx, polecat.Status.AssignedBead)
		if err != nil {
			log.V(1).Info("Failed to get bead status",
				"polecat", polecat.Name,
				"bead", polecat.Status.AssignedBead,
				"error", err)
			continue
		}

		// If bead is closed but polecat shows Working, update to Done
		if beadStatus.Status == "closed" && polecat.Status.Phase == gastownv1alpha1.PolecatPhaseWorking {
			log.Info("Bead closed externally, updating polecat to Done",
				"polecat", polecat.Name,
				"bead", polecat.Status.AssignedBead)

			polecat.Status.Phase = gastownv1alpha1.PolecatPhaseDone
			if err := r.Status().Update(ctx, &polecat); err != nil {
				log.Error(err, "Failed to update polecat status",
					"polecat", polecat.Name)
			}
		}
	}

	return nil
}

// syncConvoys checks all convoys for completion.
func (r *BeadsSyncReconciler) syncConvoys(ctx context.Context) error {
	log := logf.FromContext(ctx)

	var convoys gastownv1alpha1.ConvoyList
	if err := r.List(ctx, &convoys); err != nil {
		return err
	}

	for _, convoy := range convoys.Items {
		// Skip if already complete or no beads convoy ID
		if convoy.Status.Phase == gastownv1alpha1.ConvoyPhaseComplete {
			continue
		}
		if convoy.Status.BeadsConvoyID == "" {
			continue
		}

		// Check convoy status
		status, err := r.GTClient.ConvoyStatus(ctx, convoy.Status.BeadsConvoyID)
		if err != nil {
			log.V(1).Info("Failed to get convoy status",
				"convoy", convoy.Name,
				"beadsID", convoy.Status.BeadsConvoyID,
				"error", err)
			continue
		}

		// Update if phase changed
		newPhase := gastownv1alpha1.ConvoyPhase(status.Phase)
		if newPhase != convoy.Status.Phase {
			log.Info("Convoy phase changed externally",
				"convoy", convoy.Name,
				"oldPhase", convoy.Status.Phase,
				"newPhase", newPhase)

			convoy.Status.Phase = newPhase
			convoy.Status.Progress = status.Progress
			convoy.Status.CompletedBeads = status.Completed
			convoy.Status.PendingBeads = status.Pending

			if err := r.Status().Update(ctx, &convoy); err != nil {
				log.Error(err, "Failed to update convoy status",
					"convoy", convoy.Name)
			}
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BeadsSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Use a source that triggers periodic reconciliation
	return ctrl.NewControllerManagedBy(mgr).
		Named("beadssync").
		// We don't watch any specific resource - we just poll periodically
		// The controller will be triggered by the initial reconcile and then requeue
		For(&gastownv1alpha1.Polecat{}).
		Complete(reconcile.Func(func(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
			// Ignore the actual request - we sync all resources
			return r.Reconcile(ctx, req)
		}))
}
