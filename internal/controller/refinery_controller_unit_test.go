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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
	"github.com/org/gastown-operator/internal/git"
	gterrors "github.com/org/gastown-operator/pkg/errors"
)

// setupRefineryTestScheme creates a scheme with all required types registered.
func setupRefineryTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, gastownv1alpha1.AddToScheme(scheme))
	return scheme
}

// failingMockGitClient always fails on Clone to simulate git server unavailability.
type failingMockGitClient struct {
	cloneErr error
}

func (m *failingMockGitClient) Clone(_ context.Context) error {
	return m.cloneErr
}

func (m *failingMockGitClient) MergeBranch(_ context.Context, _ git.MergeOptions) (*git.MergeResult, error) {
	return nil, fmt.Errorf("should not be called when clone fails")
}

// successMockGitClient succeeds on all operations.
type successMockGitClient struct{}

func (m *successMockGitClient) Clone(_ context.Context) error {
	return nil
}

func (m *successMockGitClient) MergeBranch(_ context.Context, _ git.MergeOptions) (*git.MergeResult, error) {
	return &git.MergeResult{
		Success:      true,
		MergedCommit: "abc123",
	}, nil
}

func TestRefineryReconciler_Unit_CircuitBreakerBlocksAfterMaxRetries(t *testing.T) {
	scheme := setupRefineryTestScheme(t)

	rig := &gastownv1alpha1.Rig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cb-rig",
		},
		Spec: gastownv1alpha1.RigSpec{
			GitURL:      "git@github.com:test/repo.git",
			BeadsPrefix: "cb",
		},
	}

	refinery := &gastownv1alpha1.Refinery{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cb-refinery",
			Namespace: "default",
		},
		Spec: gastownv1alpha1.RefinerySpec{
			RigRef:       "cb-rig",
			TargetBranch: "main",
			Parallelism:  1,
		},
	}

	// Create a merge-ready polecat
	polecat := &gastownv1alpha1.Polecat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cb-polecat",
			Namespace: "default",
			Labels: map[string]string{
				"gastown.io/rig": "cb-rig",
			},
		},
		Spec: gastownv1alpha1.PolecatSpec{
			Rig: "cb-rig",
		},
		Status: gastownv1alpha1.PolecatStatus{
			Branch: "feature/test-branch",
			Conditions: []metav1.Condition{
				{
					Type:               ConditionAvailable,
					Status:             metav1.ConditionTrue,
					Reason:             "Ready",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(rig, refinery, polecat).
		WithStatusSubresource(refinery, polecat).
		Build()

	// Use a small maxRetries for testing
	backoff := gterrors.NewBackoffCalculatorWithConfig(
		5*time.Millisecond, 100*time.Millisecond, 3,
	)

	failFactory := func(_, _, _ string) git.GitClient {
		return &failingMockGitClient{cloneErr: fmt.Errorf("connection refused")}
	}

	recorder := record.NewFakeRecorder(20)
	r := &RefineryReconciler{
		Client:           c,
		Scheme:           scheme,
		Recorder:         recorder,
		GitClientFactory: failFactory,
		Backoff:          backoff,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      refinery.Name,
			Namespace: refinery.Namespace,
		},
	}

	// Reconcile 3 times to exhaust retries (maxRetries=3)
	for i := 0; i < 3; i++ {
		_, err := r.Reconcile(context.Background(), req)
		require.NoError(t, err)
	}

	// Verify circuit breaker is now open
	assert.True(t, backoff.ShouldGiveUp("default/cb-refinery"),
		"circuit breaker should be open after max retries")

	// Next reconcile should skip git operations entirely
	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	// Verify the refinery status reflects circuit breaker state
	var updated gastownv1alpha1.Refinery
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      refinery.Name,
		Namespace: refinery.Namespace,
	}, &updated)
	require.NoError(t, err)

	var hasCircuitBreakerCondition bool
	for _, cond := range updated.Status.Conditions {
		if cond.Type == RefineryConditionReady &&
			cond.Status == metav1.ConditionFalse &&
			cond.Reason == "CircuitBreakerOpen" {
			hasCircuitBreakerCondition = true
			break
		}
	}
	assert.True(t, hasCircuitBreakerCondition,
		"should have CircuitBreakerOpen condition when circuit breaker is open")
}

func TestRefineryReconciler_Unit_CircuitBreakerResetsOnSuccess(t *testing.T) {
	scheme := setupRefineryTestScheme(t)

	rig := &gastownv1alpha1.Rig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "reset-rig",
		},
		Spec: gastownv1alpha1.RigSpec{
			GitURL:      "git@github.com:test/repo.git",
			BeadsPrefix: "rs",
		},
	}

	refinery := &gastownv1alpha1.Refinery{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "reset-refinery",
			Namespace: "default",
		},
		Spec: gastownv1alpha1.RefinerySpec{
			RigRef:       "reset-rig",
			TargetBranch: "main",
			Parallelism:  1,
		},
	}

	polecat := &gastownv1alpha1.Polecat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "reset-polecat",
			Namespace: "default",
			Labels: map[string]string{
				"gastown.io/rig": "reset-rig",
			},
		},
		Spec: gastownv1alpha1.PolecatSpec{
			Rig: "reset-rig",
		},
		Status: gastownv1alpha1.PolecatStatus{
			Branch: "feature/reset-branch",
			Conditions: []metav1.Condition{
				{
					Type:               ConditionAvailable,
					Status:             metav1.ConditionTrue,
					Reason:             "Ready",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(rig, refinery, polecat).
		WithStatusSubresource(refinery, polecat).
		Build()

	backoff := gterrors.NewBackoffCalculatorWithConfig(
		5*time.Millisecond, 100*time.Millisecond, 5,
	)

	// Pre-load some failures
	backoffKey := "default/reset-refinery"
	_ = backoff.GetBackoffResult(backoffKey) // failure 1
	_ = backoff.GetBackoffResult(backoffKey) // failure 2
	assert.Equal(t, 2, backoff.GetRetryCount(backoffKey))

	successFactory := func(_, _, _ string) git.GitClient {
		return &successMockGitClient{}
	}

	recorder := record.NewFakeRecorder(20)
	r := &RefineryReconciler{
		Client:           c,
		Scheme:           scheme,
		Recorder:         recorder,
		GitClientFactory: successFactory,
		Backoff:          backoff,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      refinery.Name,
			Namespace: refinery.Namespace,
		},
	}

	// Successful reconcile should reset circuit breaker
	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, 0, backoff.GetRetryCount(backoffKey),
		"circuit breaker should be reset after successful merge")
}

func TestRefineryReconciler_Unit_NoCircuitBreakerWhenNilBackoff(t *testing.T) {
	scheme := setupRefineryTestScheme(t)

	rig := &gastownv1alpha1.Rig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nil-backoff-rig",
		},
		Spec: gastownv1alpha1.RigSpec{
			GitURL:      "git@github.com:test/repo.git",
			BeadsPrefix: "nb",
		},
	}

	refinery := &gastownv1alpha1.Refinery{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nil-backoff-refinery",
			Namespace: "default",
		},
		Spec: gastownv1alpha1.RefinerySpec{
			RigRef:       "nil-backoff-rig",
			TargetBranch: "main",
			Parallelism:  1,
		},
	}

	polecat := &gastownv1alpha1.Polecat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nil-backoff-polecat",
			Namespace: "default",
			Labels: map[string]string{
				"gastown.io/rig": "nil-backoff-rig",
			},
		},
		Spec: gastownv1alpha1.PolecatSpec{
			Rig: "nil-backoff-rig",
		},
		Status: gastownv1alpha1.PolecatStatus{
			Branch: "feature/nil-backoff-branch",
			Conditions: []metav1.Condition{
				{
					Type:               ConditionAvailable,
					Status:             metav1.ConditionTrue,
					Reason:             "Ready",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(rig, refinery, polecat).
		WithStatusSubresource(refinery, polecat).
		Build()

	failFactory := func(_, _, _ string) git.GitClient {
		return &failingMockGitClient{cloneErr: fmt.Errorf("connection refused")}
	}

	recorder := record.NewFakeRecorder(20)
	r := &RefineryReconciler{
		Client:           c,
		Scheme:           scheme,
		Recorder:         recorder,
		GitClientFactory: failFactory,
		Backoff:          nil, // No circuit breaker
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      refinery.Name,
			Namespace: refinery.Namespace,
		},
	}

	// Should still proceed with merge attempts (and fail) without panic
	for i := 0; i < 5; i++ {
		_, err := r.Reconcile(context.Background(), req)
		require.NoError(t, err)
	}

	// Verify merge failures were recorded (no circuit breaker blocking)
	var updated gastownv1alpha1.Refinery
	err := c.Get(context.Background(), types.NamespacedName{
		Name:      refinery.Name,
		Namespace: refinery.Namespace,
	}, &updated)
	require.NoError(t, err)
	assert.Equal(t, int32(5), updated.Status.MergesSummary.Failed,
		"all 5 attempts should have been made without circuit breaker")
}
