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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
)

const (
	// testNamespace is the default namespace used in unit tests
	testNamespace = "default"
)

// setupRigTestScheme creates a scheme with all required types registered.
func setupRigTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, gastownv1alpha1.AddToScheme(scheme))
	return scheme
}

// newTestRig creates a test Rig for unit tests with a unique name.
func newTestRig(name string) *gastownv1alpha1.Rig {
	return &gastownv1alpha1.Rig{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: gastownv1alpha1.RigSpec{
			GitURL:      "git@github.com:test/repo.git",
			BeadsPrefix: "test",
		},
	}
}

func TestRigReconciler_Unit_HandlesMissingRigGracefully(t *testing.T) {
	scheme := setupRigTestScheme(t)

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	r := &RigReconciler{Client: c, Scheme: scheme}

	// Reconcile a non-existent rig
	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "nonexistent-rig"},
	})

	require.NoError(t, err)
	assert.Zero(t, result.RequeueAfter, "should not requeue for missing rig")
}

func TestRigReconciler_Unit_AddsFinalizerOnCreation(t *testing.T) {
	scheme := setupRigTestScheme(t)

	rig := newTestRig("finalizer-rig")

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(rig).
		WithStatusSubresource(rig).
		Build()

	r := &RigReconciler{Client: c, Scheme: scheme}

	// First reconcile should add finalizer
	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: rig.Name},
	})

	require.NoError(t, err)
	assert.NotZero(t, result.RequeueAfter, "should requeue after adding finalizer")

	// Verify finalizer was added
	var updated gastownv1alpha1.Rig
	err = c.Get(context.Background(), types.NamespacedName{Name: rig.Name}, &updated)
	require.NoError(t, err)
	assert.True(t, controllerutil.ContainsFinalizer(&updated, rigFinalizer),
		"finalizer should be added")
}

func TestRigReconciler_Unit_CreatesWitnessForNewRig(t *testing.T) {
	// Set namespace env var for test
	t.Setenv("GASTOWN_NAMESPACE", testNamespace)

	scheme := setupRigTestScheme(t)

	rig := newTestRig("witness-rig")
	// Pre-add finalizer to skip that step
	rig.Finalizers = []string{rigFinalizer}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(rig).
		WithStatusSubresource(rig).
		Build()

	r := &RigReconciler{Client: c, Scheme: scheme}

	// Reconcile should create children
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: rig.Name},
	})

	require.NoError(t, err)

	// Verify Witness was created
	var witness gastownv1alpha1.Witness
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      rig.Name + "-witness",
		Namespace: testNamespace,
	}, &witness)
	require.NoError(t, err)
	assert.Equal(t, rig.Name, witness.Spec.RigRef)
	assert.Equal(t, rig.Name, witness.Labels["gastown.io/rig-owner"])
	assert.Equal(t, "rig-controller", witness.Labels["app.kubernetes.io/managed-by"])
}

func TestRigReconciler_Unit_CreatesRefineryForNewRig(t *testing.T) {
	// Set namespace env var for test
	t.Setenv("GASTOWN_NAMESPACE", testNamespace)

	scheme := setupRigTestScheme(t)

	rig := newTestRig("refinery-rig")
	// Pre-add finalizer to skip that step
	rig.Finalizers = []string{rigFinalizer}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(rig).
		WithStatusSubresource(rig).
		Build()

	r := &RigReconciler{Client: c, Scheme: scheme}

	// Reconcile should create children
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: rig.Name},
	})

	require.NoError(t, err)

	// Verify Refinery was created
	var refinery gastownv1alpha1.Refinery
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      rig.Name + "-refinery",
		Namespace: testNamespace,
	}, &refinery)
	require.NoError(t, err)
	assert.Equal(t, rig.Name, refinery.Spec.RigRef)
	assert.Equal(t, "main", refinery.Spec.TargetBranch)
	assert.Equal(t, rig.Name, refinery.Labels["gastown.io/rig-owner"])
}

func TestRigReconciler_Unit_UpdatesStatusAfterChildrenCreated(t *testing.T) {
	// Set namespace env var for test
	t.Setenv("GASTOWN_NAMESPACE", testNamespace)

	scheme := setupRigTestScheme(t)

	rig := newTestRig("status-rig")
	rig.Finalizers = []string{rigFinalizer}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(rig).
		WithStatusSubresource(rig).
		Build()

	r := &RigReconciler{Client: c, Scheme: scheme}

	// Reconcile to create children
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: rig.Name},
	})
	require.NoError(t, err)

	// Verify status was updated
	var updated gastownv1alpha1.Rig
	err = c.Get(context.Background(), types.NamespacedName{Name: rig.Name}, &updated)
	require.NoError(t, err)
	assert.True(t, updated.Status.WitnessCreated)
	assert.True(t, updated.Status.RefineryCreated)
	assert.Equal(t, testNamespace, updated.Status.ChildNamespace)
}

func TestRigReconciler_Unit_DoesNotRecreateExistingChildren(t *testing.T) {
	// Set namespace env var for test
	t.Setenv("GASTOWN_NAMESPACE", testNamespace)

	scheme := setupRigTestScheme(t)

	rig := newTestRig("existing-children-rig")
	rig.Finalizers = []string{rigFinalizer}
	rig.Status.WitnessCreated = true
	rig.Status.RefineryCreated = true
	rig.Status.ChildNamespace = testNamespace

	// Pre-create children
	witness := &gastownv1alpha1.Witness{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rig.Name + "-witness",
			Namespace: testNamespace,
			Labels: map[string]string{
				"gastown.io/rig-owner": rig.Name,
			},
		},
		Spec: gastownv1alpha1.WitnessSpec{
			RigRef: rig.Name,
		},
	}
	refinery := &gastownv1alpha1.Refinery{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rig.Name + "-refinery",
			Namespace: testNamespace,
			Labels: map[string]string{
				"gastown.io/rig-owner": rig.Name,
			},
		},
		Spec: gastownv1alpha1.RefinerySpec{
			RigRef:       rig.Name,
			TargetBranch: "main",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(rig, witness, refinery).
		WithStatusSubresource(rig).
		Build()

	r := &RigReconciler{Client: c, Scheme: scheme}

	// Multiple reconciles should not error or create duplicates
	for i := 0; i < 3; i++ {
		_, err := r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{Name: rig.Name},
		})
		require.NoError(t, err)
	}

	// Verify only one Witness exists
	var witnessList gastownv1alpha1.WitnessList
	err := c.List(context.Background(), &witnessList, client.InNamespace(testNamespace))
	require.NoError(t, err)
	assert.Len(t, witnessList.Items, 1)
}

func TestRigReconciler_Unit_HandlesDeletedRig(t *testing.T) {
	// Set namespace env var for test
	t.Setenv("GASTOWN_NAMESPACE", testNamespace)

	scheme := setupRigTestScheme(t)

	// Create the rig without deletion timestamp first
	rig := newTestRig("deleted-rig")
	rig.Finalizers = []string{rigFinalizer}
	rig.Status.ChildNamespace = testNamespace

	// Pre-create children
	witness := &gastownv1alpha1.Witness{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rig.Name + "-witness",
			Namespace: testNamespace,
		},
		Spec: gastownv1alpha1.WitnessSpec{
			RigRef: rig.Name,
		},
	}
	refinery := &gastownv1alpha1.Refinery{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rig.Name + "-refinery",
			Namespace: testNamespace,
		},
		Spec: gastownv1alpha1.RefinerySpec{
			RigRef: rig.Name,
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(rig, witness, refinery).
		WithStatusSubresource(rig).
		Build()

	r := &RigReconciler{Client: c, Scheme: scheme}

	// Simulate deletion by calling Delete (which sets DeletionTimestamp)
	var current gastownv1alpha1.Rig
	err := c.Get(context.Background(), types.NamespacedName{Name: rig.Name}, &current)
	require.NoError(t, err)
	err = c.Delete(context.Background(), &current)
	require.NoError(t, err)

	// Re-fetch after deletion to get the object with DeletionTimestamp
	err = c.Get(context.Background(), types.NamespacedName{Name: rig.Name}, &current)
	require.NoError(t, err)
	assert.NotNil(t, current.DeletionTimestamp, "deletion timestamp should be set")

	// Reconcile should handle deletion
	_, err = r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: rig.Name},
	})
	require.NoError(t, err)

	// Verify children were deleted
	var w gastownv1alpha1.Witness
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      rig.Name + "-witness",
		Namespace: testNamespace,
	}, &w)
	assert.Error(t, err, "witness should be deleted")

	var rf gastownv1alpha1.Refinery
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      rig.Name + "-refinery",
		Namespace: testNamespace,
	}, &rf)
	assert.Error(t, err, "refinery should be deleted")

	// Note: After removing the finalizer, the fake client deletes the object
	// So we check that either the object is gone or has no finalizer
	var updated gastownv1alpha1.Rig
	err = c.Get(context.Background(), types.NamespacedName{Name: rig.Name}, &updated)
	if err == nil {
		assert.False(t, controllerutil.ContainsFinalizer(&updated, rigFinalizer),
			"finalizer should be removed")
	}
	// If err != nil (not found), the object was correctly deleted
}

func TestRigReconciler_Unit_HandlesDeletedRigWithMissingChildren(t *testing.T) {
	// Set namespace env var for test
	t.Setenv("GASTOWN_NAMESPACE", testNamespace)

	scheme := setupRigTestScheme(t)

	rig := newTestRig("orphaned-rig")
	rig.Finalizers = []string{rigFinalizer}
	rig.Status.ChildNamespace = testNamespace

	// No children exist - they were already deleted

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(rig).
		WithStatusSubresource(rig).
		Build()

	r := &RigReconciler{Client: c, Scheme: scheme}

	// Simulate deletion by calling Delete (which sets DeletionTimestamp)
	var current gastownv1alpha1.Rig
	err := c.Get(context.Background(), types.NamespacedName{Name: rig.Name}, &current)
	require.NoError(t, err)
	err = c.Delete(context.Background(), &current)
	require.NoError(t, err)

	// Reconcile should succeed even without children
	_, err = r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: rig.Name},
	})
	require.NoError(t, err)

	// After removing finalizer, the object should be deleted by the fake client
	// or have no finalizer if still present
	var updated gastownv1alpha1.Rig
	err = c.Get(context.Background(), types.NamespacedName{Name: rig.Name}, &updated)
	if err == nil {
		assert.False(t, controllerutil.ContainsFinalizer(&updated, rigFinalizer))
	}
	// If err != nil (not found), the object was correctly deleted
}

func TestRigReconciler_Unit_UsesEnvVarForNamespace(t *testing.T) {
	customNamespace := "custom-gastown"
	t.Setenv("GASTOWN_NAMESPACE", customNamespace)

	scheme := setupRigTestScheme(t)

	rig := newTestRig("custom-ns-rig")
	rig.Finalizers = []string{rigFinalizer}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(rig).
		WithStatusSubresource(rig).
		Build()

	r := &RigReconciler{Client: c, Scheme: scheme}

	// Reconcile to create children
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: rig.Name},
	})
	require.NoError(t, err)

	// Verify children were created in custom namespace
	var witness gastownv1alpha1.Witness
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      rig.Name + "-witness",
		Namespace: customNamespace,
	}, &witness)
	require.NoError(t, err)

	var refinery gastownv1alpha1.Refinery
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      rig.Name + "-refinery",
		Namespace: customNamespace,
	}, &refinery)
	require.NoError(t, err)
}

func TestRigReconciler_Unit_DefaultsToGastownSystemNamespace(t *testing.T) {
	// Ensure env var is not set (t.Setenv to empty string effectively unsets it for the test)
	t.Setenv("GASTOWN_NAMESPACE", "")

	r := &RigReconciler{}
	ns := r.getChildNamespace()
	assert.Equal(t, "gastown-system", ns)
}

func TestRigReconciler_Unit_TableDrivenTests(t *testing.T) {
	tests := []struct {
		name     string
		objects  []client.Object
		req      reconcile.Request
		wantErr  bool
		validate func(t *testing.T, c client.Client)
	}{
		{
			name:    "handles missing rig gracefully",
			objects: []client.Object{},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "nonexistent"},
			},
			wantErr: false,
		},
		{
			name: "adds finalizer to new rig",
			objects: []client.Object{
				&gastownv1alpha1.Rig{
					ObjectMeta: metav1.ObjectMeta{Name: "new-rig"},
					Spec: gastownv1alpha1.RigSpec{
						GitURL:      "git@example.com:test/repo.git",
						BeadsPrefix: "nr",
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "new-rig"},
			},
			wantErr: false,
			validate: func(t *testing.T, c client.Client) {
				var rig gastownv1alpha1.Rig
				err := c.Get(context.Background(), types.NamespacedName{Name: "new-rig"}, &rig)
				require.NoError(t, err)
				assert.Contains(t, rig.Finalizers, rigFinalizer)
			},
		},
		{
			name: "creates witness and refinery",
			objects: []client.Object{
				&gastownv1alpha1.Rig{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "full-rig",
						Finalizers: []string{rigFinalizer},
					},
					Spec: gastownv1alpha1.RigSpec{
						GitURL:      "git@example.com:test/repo.git",
						BeadsPrefix: "fr",
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "full-rig"},
			},
			wantErr: false,
			validate: func(t *testing.T, c client.Client) {
				var witness gastownv1alpha1.Witness
				err := c.Get(context.Background(), types.NamespacedName{
					Name:      "full-rig-witness",
					Namespace: testNamespace,
				}, &witness)
				require.NoError(t, err)
				assert.Equal(t, "full-rig", witness.Spec.RigRef)

				var refinery gastownv1alpha1.Refinery
				err = c.Get(context.Background(), types.NamespacedName{
					Name:      "full-rig-refinery",
					Namespace: testNamespace,
				}, &refinery)
				require.NoError(t, err)
				assert.Equal(t, "full-rig", refinery.Spec.RigRef)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set namespace env var for this test
			t.Setenv("GASTOWN_NAMESPACE", testNamespace)

			scheme := setupRigTestScheme(t)

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithStatusSubresource(&gastownv1alpha1.Rig{}).
				Build()

			r := &RigReconciler{Client: c, Scheme: scheme}
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
