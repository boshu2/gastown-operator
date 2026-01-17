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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
)

// BeadStoreReconciler reconciles a BeadStore object
type BeadStoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=gastown.gastown.io,resources=beadstores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=beadstores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=beadstores/finalizers,verbs=update

// Reconcile manages the BeadStore lifecycle.
// STUB: This is a placeholder implementation. Beads sync is not yet implemented.
// TODO(he-xxxx): Implement beadstore synchronization:
//  1. Fetch BeadStore resource
//  2. Validate RigRef points to existing Rig
//  3. Initialize beads database if needed
//  4. Sync with git backend using GitSecretRef credentials
//  5. Update status fields (Phase, LastSyncTime, IssueCount)
//  6. Requeue after SyncInterval
func (r *BeadStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.V(1).Info("Reconciling BeadStore (STUB - not implemented)", "name", req.NamespacedName)

	// STUB: No-op until beadstore sync logic is implemented
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BeadStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gastownv1alpha1.BeadStore{}).
		Named("beadstore").
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1, // BeadStore is a singleton config
		}).
		Complete(r)
}
