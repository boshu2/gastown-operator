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
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
	"github.com/org/gastown-operator/internal/git"
)

const (
	// RefineryConditionReady indicates the refinery is ready to process merges.
	RefineryConditionReady = "Ready"

	// RefineryConditionProcessing indicates a merge is in progress.
	RefineryConditionProcessing = "Processing"

	// Default requeue interval for idle refinery.
	// Uses RequeueDefault for normal idle monitoring.
	refineryIdleRequeueInterval = RequeueDefault

	// Requeue interval during active processing.
	// Uses a shorter interval for active merge monitoring.
	refineryProcessingRequeueInterval = 5 * time.Second
)

// RefineryReconciler reconciles a Refinery object
type RefineryReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	// GitClientFactory creates git clients. If nil, uses git.DefaultGitClientFactory.
	GitClientFactory git.GitClientFactory
}

// +kubebuilder:rbac:groups=gastown.gastown.io,resources=refineries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=refineries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=refineries/finalizers,verbs=update
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=polecats,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=gastown.gastown.io,resources=rigs,verbs=get;list;watch
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
// Workflow:
//  1. Get the Rig to find the git URL
//  2. Get git credentials from GitSecretRef
//  3. Clone/fetch the repository
//  4. Rebase the polecat branch onto target
//  5. Run tests if TestCommand is configured
//  6. Push to target branch
//  7. Clean up polecat branch
func (r *RefineryReconciler) processMerge(
	ctx context.Context, refinery *gastownv1alpha1.Refinery, polecat *gastownv1alpha1.Polecat,
) error {
	log := logf.FromContext(ctx)

	// Get the source branch from polecat status
	sourceBranch := polecat.Status.Branch
	if sourceBranch == "" {
		return fmt.Errorf("polecat %s has no branch in status", polecat.Name)
	}

	targetBranch := refinery.Spec.TargetBranch
	if targetBranch == "" {
		targetBranch = "main"
	}

	log.Info("Processing merge",
		"polecat", polecat.Name,
		"sourceBranch", sourceBranch,
		"targetBranch", targetBranch,
		"testCommand", refinery.Spec.TestCommand)

	// Get the Rig to find the git URL
	rig := &gastownv1alpha1.Rig{}
	if err := r.Get(ctx, types.NamespacedName{Name: refinery.Spec.RigRef}, rig); err != nil {
		return fmt.Errorf("failed to get rig %s: %w", refinery.Spec.RigRef, err)
	}

	gitURL := rig.Spec.GitURL
	if gitURL == "" {
		return fmt.Errorf("rig %s has no gitURL", refinery.Spec.RigRef)
	}

	// Set up git credentials if specified
	var sshKeyPath string
	if refinery.Spec.GitSecretRef != nil {
		keyPath, cleanup, err := r.setupGitCredentials(ctx, refinery)
		if err != nil {
			return fmt.Errorf("failed to setup git credentials: %w", err)
		}
		defer cleanup()
		sshKeyPath = keyPath
	}

	// Create a temp directory for the clone
	workDir, err := os.MkdirTemp("", "refinery-merge-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	repoDir := filepath.Join(workDir, "repo")

	// Create git client using factory (allows test injection)
	factory := r.GitClientFactory
	if factory == nil {
		factory = git.DefaultGitClientFactory
	}
	gitClient := factory(repoDir, gitURL, sshKeyPath)

	// Clone the repository
	log.Info("Cloning repository", "url", gitURL)
	if err := gitClient.Clone(ctx); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Perform the merge
	mergeOpts := git.MergeOptions{
		SourceBranch:       sourceBranch,
		TargetBranch:       targetBranch,
		TestCommand:        refinery.Spec.TestCommand,
		DeleteSourceBranch: true,
	}

	log.Info("Executing merge workflow",
		"sourceBranch", sourceBranch,
		"targetBranch", targetBranch)

	result, err := gitClient.MergeBranch(ctx, mergeOpts)
	if err != nil {
		return fmt.Errorf("merge failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("merge failed: %s", result.Error)
	}

	log.Info("Merge completed successfully",
		"mergedCommit", result.MergedCommit,
		"sourceBranch", sourceBranch,
		"targetBranch", targetBranch)

	// Update polecat status to indicate merge complete
	meta.SetStatusCondition(&polecat.Status.Conditions, metav1.Condition{
		Type:               "Merged",
		Status:             metav1.ConditionTrue,
		Reason:             "MergeComplete",
		Message:            fmt.Sprintf("Branch %s merged to %s (commit: %s)", sourceBranch, targetBranch, result.MergedCommit),
		LastTransitionTime: metav1.Now(),
	})

	if err := r.Status().Update(ctx, polecat); err != nil {
		return err
	}

	return nil
}

// setupGitCredentials extracts SSH key from secret and writes to temp file.
// Returns the path to the key file and a cleanup function.
func (r *RefineryReconciler) setupGitCredentials(
	ctx context.Context, refinery *gastownv1alpha1.Refinery,
) (string, func(), error) {
	if refinery.Spec.GitSecretRef == nil {
		return "", func() {}, nil
	}

	secret := &corev1.Secret{}
	secretKey := types.NamespacedName{
		Name:      refinery.Spec.GitSecretRef.Name,
		Namespace: refinery.Namespace,
	}

	if err := r.Get(ctx, secretKey, secret); err != nil {
		return "", nil, fmt.Errorf("failed to get git secret %s: %w", secretKey, err)
	}

	// Look for SSH key in common key names
	var sshKey []byte
	for _, keyName := range []string{"ssh-privatekey", "id_rsa", "id_ed25519", "identity"} {
		if key, ok := secret.Data[keyName]; ok {
			sshKey = key
			break
		}
	}

	if sshKey == nil {
		return "", nil, fmt.Errorf("no SSH key found in secret %s", secretKey)
	}

	// Write key to temp file
	keyFile, err := os.CreateTemp("", "git-ssh-key-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp key file: %w", err)
	}

	if _, err := keyFile.Write(sshKey); err != nil {
		_ = os.Remove(keyFile.Name())
		return "", nil, fmt.Errorf("failed to write SSH key: %w", err)
	}

	if err := keyFile.Chmod(0o600); err != nil {
		_ = os.Remove(keyFile.Name())
		return "", nil, fmt.Errorf("failed to chmod SSH key: %w", err)
	}

	if err := keyFile.Close(); err != nil {
		_ = os.Remove(keyFile.Name())
		return "", nil, fmt.Errorf("failed to close SSH key file: %w", err)
	}

	cleanup := func() {
		_ = os.Remove(keyFile.Name())
	}

	return keyFile.Name(), cleanup, nil
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
