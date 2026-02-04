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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
)

// setupWitnessTestScheme creates a scheme with all required types registered.
func setupWitnessTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, gastownv1alpha1.AddToScheme(scheme))
	return scheme
}

// newTestWitness creates a test Witness for unit tests.
func newTestWitness(name, namespace, rigRef string) *gastownv1alpha1.Witness {
	return &gastownv1alpha1.Witness{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: gastownv1alpha1.WitnessSpec{
			RigRef:           rigRef,
			EscalationTarget: "mayor",
		},
	}
}

// fakeGTClient is a mock implementation for unit tests.
type fakeGTClient struct {
	mailSendFunc func(ctx context.Context, address, subject, message string) error
	mailSentTo   []string
}

func (f *fakeGTClient) MailSend(ctx context.Context, address, subject, message string) error {
	f.mailSentTo = append(f.mailSentTo, address)
	if f.mailSendFunc != nil {
		return f.mailSendFunc(ctx, address, subject, message)
	}
	return nil
}

func TestWitnessReconciler_Unit_HandlesMissingWitnessGracefully(t *testing.T) {
	scheme := setupWitnessTestScheme(t)

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	r := &WitnessReconciler{
		Client:   c,
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(10),
	}

	// Reconcile a non-existent witness
	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "nonexistent-witness",
			Namespace: "default",
		},
	})

	require.NoError(t, err)
	assert.Zero(t, result.RequeueAfter, "should not requeue for missing witness")
}

func TestWitnessReconciler_Unit_ListsPolecatsCorrectly(t *testing.T) {
	scheme := setupWitnessTestScheme(t)

	witness := newTestWitness("test-witness", "default", "test-rig")

	// Create polecats for the rig
	polecat1 := &gastownv1alpha1.Polecat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "polecat-1",
			Namespace: "default",
			Labels: map[string]string{
				"gastown.io/rig": "test-rig",
			},
		},
		Spec: gastownv1alpha1.PolecatSpec{
			Rig: "test-rig",
		},
		Status: gastownv1alpha1.PolecatStatus{
			Conditions: []metav1.Condition{
				{
					Type:               ConditionProgressing,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}
	polecat2 := &gastownv1alpha1.Polecat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "polecat-2",
			Namespace: "default",
			Labels: map[string]string{
				"gastown.io/rig": "test-rig",
			},
		},
		Spec: gastownv1alpha1.PolecatSpec{
			Rig: "test-rig",
		},
		Status: gastownv1alpha1.PolecatStatus{
			Conditions: []metav1.Condition{
				{
					Type:   ConditionAvailable,
					Status: metav1.ConditionTrue,
				},
			},
		},
	}
	// Polecat from different rig - should not be counted
	polecatOther := &gastownv1alpha1.Polecat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "polecat-other",
			Namespace: "default",
			Labels: map[string]string{
				"gastown.io/rig": "other-rig",
			},
		},
		Spec: gastownv1alpha1.PolecatSpec{
			Rig: "other-rig",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(witness, polecat1, polecat2, polecatOther).
		WithStatusSubresource(witness).
		Build()

	r := &WitnessReconciler{
		Client:   c,
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(10),
	}

	// Reconcile
	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      witness.Name,
			Namespace: witness.Namespace,
		},
	})

	require.NoError(t, err)
	assert.NotZero(t, result.RequeueAfter, "should requeue for health checks")

	// Verify status was updated with correct counts
	var updated gastownv1alpha1.Witness
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      witness.Name,
		Namespace: witness.Namespace,
	}, &updated)
	require.NoError(t, err)

	// Should count 2 polecats for test-rig (not the one from other-rig)
	assert.Equal(t, int32(2), updated.Status.PolecatsSummary.Total)
	assert.Equal(t, int32(1), updated.Status.PolecatsSummary.Running)
	assert.Equal(t, int32(1), updated.Status.PolecatsSummary.Succeeded)
}

func TestWitnessReconciler_Unit_CalculatesSummaryCorrectly(t *testing.T) {
	r := &WitnessReconciler{}
	stuckThreshold := 15 * time.Minute

	tests := []struct {
		name     string
		polecats *gastownv1alpha1.PolecatList
		want     gastownv1alpha1.PolecatsSummary
	}{
		{
			name:     "empty list",
			polecats: &gastownv1alpha1.PolecatList{Items: []gastownv1alpha1.Polecat{}},
			want: gastownv1alpha1.PolecatsSummary{
				Total:     0,
				Running:   0,
				Succeeded: 0,
				Failed:    0,
				Stuck:     0,
			},
		},
		{
			name: "running polecats",
			polecats: &gastownv1alpha1.PolecatList{
				Items: []gastownv1alpha1.Polecat{
					{
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:               ConditionProgressing,
									Status:             metav1.ConditionTrue,
									LastTransitionTime: metav1.Now(),
								},
							},
						},
					},
				},
			},
			want: gastownv1alpha1.PolecatsSummary{
				Total:   1,
				Running: 1,
			},
		},
		{
			name: "succeeded polecats",
			polecats: &gastownv1alpha1.PolecatList{
				Items: []gastownv1alpha1.Polecat{
					{
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:   ConditionAvailable,
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			want: gastownv1alpha1.PolecatsSummary{
				Total:     1,
				Succeeded: 1,
			},
		},
		{
			name: "failed polecats",
			polecats: &gastownv1alpha1.PolecatList{
				Items: []gastownv1alpha1.Polecat{
					{
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:   ConditionDegraded,
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			want: gastownv1alpha1.PolecatsSummary{
				Total:  1,
				Failed: 1,
			},
		},
		{
			name: "stuck polecats (old progressing condition)",
			polecats: &gastownv1alpha1.PolecatList{
				Items: []gastownv1alpha1.Polecat{
					{
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:               ConditionProgressing,
									Status:             metav1.ConditionTrue,
									LastTransitionTime: metav1.NewTime(time.Now().Add(-20 * time.Minute)),
								},
							},
						},
					},
				},
			},
			want: gastownv1alpha1.PolecatsSummary{
				Total:   1,
				Running: 1,
				Stuck:   1,
			},
		},
		{
			name: "mixed polecats",
			polecats: &gastownv1alpha1.PolecatList{
				Items: []gastownv1alpha1.Polecat{
					{
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:               ConditionProgressing,
									Status:             metav1.ConditionTrue,
									LastTransitionTime: metav1.Now(),
								},
							},
						},
					},
					{
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:   ConditionAvailable,
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
					{
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:   ConditionDegraded,
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			want: gastownv1alpha1.PolecatsSummary{
				Total:     3,
				Running:   1,
				Succeeded: 1,
				Failed:    1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.calculateSummary(tt.polecats, stuckThreshold)
			assert.Equal(t, tt.want.Total, got.Total, "Total mismatch")
			assert.Equal(t, tt.want.Running, got.Running, "Running mismatch")
			assert.Equal(t, tt.want.Succeeded, got.Succeeded, "Succeeded mismatch")
			assert.Equal(t, tt.want.Failed, got.Failed, "Failed mismatch")
			assert.Equal(t, tt.want.Stuck, got.Stuck, "Stuck mismatch")
		})
	}
}

func TestWitnessReconciler_Unit_DeterminePhase(t *testing.T) {
	r := &WitnessReconciler{}

	tests := []struct {
		name    string
		summary gastownv1alpha1.PolecatsSummary
		want    string
	}{
		{
			name: "pending when no polecats",
			summary: gastownv1alpha1.PolecatsSummary{
				Total:     0,
				Running:   0,
				Succeeded: 0,
				Failed:    0,
				Stuck:     0,
			},
			want: "Pending",
		},
		{
			name: "active when running polecats",
			summary: gastownv1alpha1.PolecatsSummary{
				Total:     2,
				Running:   2,
				Succeeded: 0,
				Failed:    0,
				Stuck:     0,
			},
			want: "Active",
		},
		{
			name: "degraded when stuck polecats",
			summary: gastownv1alpha1.PolecatsSummary{
				Total:     3,
				Running:   2,
				Succeeded: 0,
				Failed:    0,
				Stuck:     1,
			},
			want: "Degraded",
		},
		{
			name: "degraded when failed polecats",
			summary: gastownv1alpha1.PolecatsSummary{
				Total:     3,
				Running:   1,
				Succeeded: 1,
				Failed:    1,
				Stuck:     0,
			},
			want: "Degraded",
		},
		{
			name: "pending when only succeeded",
			summary: gastownv1alpha1.PolecatsSummary{
				Total:     2,
				Running:   0,
				Succeeded: 2,
				Failed:    0,
				Stuck:     0,
			},
			want: "Pending",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.determinePhase(tt.summary)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWitnessReconciler_Unit_EscalationLogic(t *testing.T) {
	scheme := setupWitnessTestScheme(t)

	witness := newTestWitness("test-witness", "default", "test-rig")

	// Create stuck polecat
	stuckPolecat := &gastownv1alpha1.Polecat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stuck-polecat",
			Namespace: "default",
			Labels: map[string]string{
				"gastown.io/rig": "test-rig",
			},
		},
		Spec: gastownv1alpha1.PolecatSpec{
			Rig: "test-rig",
		},
		Status: gastownv1alpha1.PolecatStatus{
			Conditions: []metav1.Condition{
				{
					Type:               ConditionProgressing,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(time.Now().Add(-20 * time.Minute)),
				},
			},
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(witness, stuckPolecat).
		WithStatusSubresource(witness).
		Build()

	gtClient := &fakeGTClient{}
	recorder := record.NewFakeRecorder(10)

	r := &WitnessReconciler{
		Client:   c,
		Scheme:   scheme,
		Recorder: recorder,
		GTClient: gtClient,
	}

	// Reconcile
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      witness.Name,
			Namespace: witness.Namespace,
		},
	})

	require.NoError(t, err)

	// Verify mail was sent to mayor
	assert.Contains(t, gtClient.mailSentTo, "mayor", "should send mail to mayor")

	// Verify Ready condition is False when stuck polecats exist
	var updated gastownv1alpha1.Witness
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      witness.Name,
		Namespace: witness.Namespace,
	}, &updated)
	require.NoError(t, err)

	var readyConditionFalse bool
	for _, cond := range updated.Status.Conditions {
		if cond.Type == ConditionWitnessReady && cond.Status == metav1.ConditionFalse {
			readyConditionFalse = true
			assert.Equal(t, "IssuesDetected", cond.Reason)
			break
		}
	}
	assert.True(t, readyConditionFalse, "Ready condition should be False when stuck")
}

func TestWitnessReconciler_Unit_NoEscalationWhenHealthy(t *testing.T) {
	scheme := setupWitnessTestScheme(t)

	witness := newTestWitness("healthy-witness", "default", "healthy-rig")

	// Create healthy polecat
	healthyPolecat := &gastownv1alpha1.Polecat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "healthy-polecat",
			Namespace: "default",
			Labels: map[string]string{
				"gastown.io/rig": "healthy-rig",
			},
		},
		Spec: gastownv1alpha1.PolecatSpec{
			Rig: "healthy-rig",
		},
		Status: gastownv1alpha1.PolecatStatus{
			Conditions: []metav1.Condition{
				{
					Type:               ConditionProgressing,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(witness, healthyPolecat).
		WithStatusSubresource(witness).
		Build()

	gtClient := &fakeGTClient{}

	r := &WitnessReconciler{
		Client:   c,
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(10),
		GTClient: gtClient,
	}

	// Reconcile
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      witness.Name,
			Namespace: witness.Namespace,
		},
	})

	require.NoError(t, err)

	// Verify no mail was sent
	assert.Empty(t, gtClient.mailSentTo, "should not send mail when healthy")

	// Verify Ready condition is True
	var updated gastownv1alpha1.Witness
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      witness.Name,
		Namespace: witness.Namespace,
	}, &updated)
	require.NoError(t, err)

	var readyConditionTrue bool
	for _, cond := range updated.Status.Conditions {
		if cond.Type == ConditionWitnessReady && cond.Status == metav1.ConditionTrue {
			readyConditionTrue = true
			break
		}
	}
	assert.True(t, readyConditionTrue, "Ready condition should be True when healthy")
}

func TestWitnessReconciler_Unit_FallbackToOldConditions(t *testing.T) {
	r := &WitnessReconciler{}
	stuckThreshold := 15 * time.Minute

	t.Run("fallback to Ready condition for succeeded", func(t *testing.T) {
		polecats := &gastownv1alpha1.PolecatList{
			Items: []gastownv1alpha1.Polecat{
				{
					Status: gastownv1alpha1.PolecatStatus{
						Conditions: []metav1.Condition{
							{
								Type:   "Ready",
								Status: metav1.ConditionTrue,
								Reason: "PodSucceeded",
							},
						},
					},
				},
			},
		}

		summary := r.calculateSummary(polecats, stuckThreshold)
		assert.Equal(t, int32(1), summary.Total)
		assert.Equal(t, int32(1), summary.Succeeded)
	})

	t.Run("fallback to Working condition for running", func(t *testing.T) {
		polecats := &gastownv1alpha1.PolecatList{
			Items: []gastownv1alpha1.Polecat{
				{
					Status: gastownv1alpha1.PolecatStatus{
						Conditions: []metav1.Condition{
							{
								Type:               "Working",
								Status:             metav1.ConditionTrue,
								LastTransitionTime: metav1.Now(),
							},
						},
					},
				},
			},
		}

		summary := r.calculateSummary(polecats, stuckThreshold)
		assert.Equal(t, int32(1), summary.Total)
		assert.Equal(t, int32(1), summary.Running)
	})

	t.Run("detect stuck with old Working condition", func(t *testing.T) {
		polecats := &gastownv1alpha1.PolecatList{
			Items: []gastownv1alpha1.Polecat{
				{
					Status: gastownv1alpha1.PolecatStatus{
						Conditions: []metav1.Condition{
							{
								Type:               "Working",
								Status:             metav1.ConditionTrue,
								LastTransitionTime: metav1.NewTime(time.Now().Add(-20 * time.Minute)),
							},
						},
					},
				},
			},
		}

		summary := r.calculateSummary(polecats, stuckThreshold)
		assert.Equal(t, int32(1), summary.Total)
		assert.Equal(t, int32(1), summary.Running)
		assert.Equal(t, int32(1), summary.Stuck)
	})

	t.Run("prefers new conditions when both present", func(t *testing.T) {
		polecats := &gastownv1alpha1.PolecatList{
			Items: []gastownv1alpha1.Polecat{
				{
					Status: gastownv1alpha1.PolecatStatus{
						Conditions: []metav1.Condition{
							{
								Type:   ConditionAvailable,
								Status: metav1.ConditionTrue,
							},
							{
								Type:   "Ready",
								Status: metav1.ConditionTrue,
								Reason: "PodSucceeded",
							},
						},
					},
				},
			},
		}

		summary := r.calculateSummary(polecats, stuckThreshold)
		assert.Equal(t, int32(1), summary.Total)
		// Should only count once, not double-count
		assert.Equal(t, int32(1), summary.Succeeded)
	})
}

func TestWitnessReconciler_Unit_UsesSpecIntervals(t *testing.T) {
	scheme := setupWitnessTestScheme(t)

	customInterval := metav1.Duration{Duration: 1 * time.Minute}
	witness := &gastownv1alpha1.Witness{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom-witness",
			Namespace: "default",
		},
		Spec: gastownv1alpha1.WitnessSpec{
			RigRef:              "test-rig",
			HealthCheckInterval: &customInterval,
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(witness).
		WithStatusSubresource(witness).
		Build()

	r := &WitnessReconciler{
		Client:   c,
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(10),
	}

	// Reconcile
	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      witness.Name,
			Namespace: witness.Namespace,
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 1*time.Minute, result.RequeueAfter, "should use custom interval")
}

func TestWitnessReconciler_Unit_TableDrivenTests(t *testing.T) {
	tests := []struct {
		name     string
		objects  []client.Object
		req      reconcile.Request
		wantErr  bool
		validate func(t *testing.T, c client.Client)
	}{
		{
			name:    "handles missing witness gracefully",
			objects: []client.Object{},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "nonexistent",
					Namespace: "default",
				},
			},
			wantErr: false,
		},
		{
			name: "updates last check time",
			objects: []client.Object{
				&gastownv1alpha1.Witness{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "check-witness",
						Namespace: "default",
					},
					Spec: gastownv1alpha1.WitnessSpec{
						RigRef: "test-rig",
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "check-witness",
					Namespace: "default",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, c client.Client) {
				var witness gastownv1alpha1.Witness
				err := c.Get(context.Background(), types.NamespacedName{
					Name:      "check-witness",
					Namespace: "default",
				}, &witness)
				require.NoError(t, err)
				assert.NotNil(t, witness.Status.LastCheckTime)
			},
		},
		{
			name: "sets phase based on polecats",
			objects: []client.Object{
				&gastownv1alpha1.Witness{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "phase-witness",
						Namespace: "default",
					},
					Spec: gastownv1alpha1.WitnessSpec{
						RigRef: "test-rig",
					},
				},
				&gastownv1alpha1.Polecat{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "running-pc",
						Namespace: "default",
						Labels: map[string]string{
							"gastown.io/rig": "test-rig",
						},
					},
					Spec: gastownv1alpha1.PolecatSpec{
						Rig: "test-rig",
					},
					Status: gastownv1alpha1.PolecatStatus{
						Conditions: []metav1.Condition{
							{
								Type:               ConditionProgressing,
								Status:             metav1.ConditionTrue,
								LastTransitionTime: metav1.Now(),
							},
						},
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "phase-witness",
					Namespace: "default",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, c client.Client) {
				var witness gastownv1alpha1.Witness
				err := c.Get(context.Background(), types.NamespacedName{
					Name:      "phase-witness",
					Namespace: "default",
				}, &witness)
				require.NoError(t, err)
				assert.Equal(t, "Active", witness.Status.Phase)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := setupWitnessTestScheme(t)

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithStatusSubresource(&gastownv1alpha1.Witness{}).
				Build()

			r := &WitnessReconciler{
				Client:   c,
				Scheme:   scheme,
				Recorder: record.NewFakeRecorder(10),
			}
			_, err := r.Reconcile(context.Background(), tt.req)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.validate != nil {
				tt.validate(t, c)
			}
		})
	}
}
