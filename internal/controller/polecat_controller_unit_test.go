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
	corev1 "k8s.io/api/core/v1"
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
	// polecatTestNamespace is the default namespace used in polecat unit tests
	polecatTestNamespace = "default"
)

// setupPolecatTestScheme creates a scheme with all required types registered.
func setupPolecatTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, gastownv1alpha1.AddToScheme(scheme))
	return scheme
}

// newTestPolecat creates a test Polecat for unit tests in the default namespace.
func newTestPolecat(name string) *gastownv1alpha1.Polecat {
	return &gastownv1alpha1.Polecat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: polecatTestNamespace,
		},
		Spec: gastownv1alpha1.PolecatSpec{
			Rig:           "test-rig",
			DesiredState:  gastownv1alpha1.PolecatDesiredWorking,
			BeadID:        "test-bead-123",
			ExecutionMode: gastownv1alpha1.ExecutionModeKubernetes,
			Kubernetes: &gastownv1alpha1.KubernetesSpec{
				GitRepository: "git@github.com:example/repo.git",
				GitBranch:     "main",
				GitSecretRef: gastownv1alpha1.SecretReference{
					Name: "git-creds",
				},
				ClaudeCredsSecretRef: &gastownv1alpha1.SecretReference{
					Name: "claude-creds",
				},
			},
		},
	}
}

func TestPolecatReconciler_Unit_HandlesMissingPolecatGracefully(t *testing.T) {
	scheme := setupPolecatTestScheme(t)

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	r := &PolecatReconciler{Client: c, Scheme: scheme}

	// Reconcile a non-existent polecat
	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "nonexistent-polecat",
			Namespace: "default",
		},
	})

	require.NoError(t, err)
	assert.Zero(t, result.RequeueAfter, "should not requeue for missing polecat")
}

func TestPolecatReconciler_Unit_AddsFinalizerOnCreation(t *testing.T) {
	scheme := setupPolecatTestScheme(t)

	polecat := newTestPolecat("test-polecat")

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(polecat).
		WithStatusSubresource(polecat).
		Build()

	r := &PolecatReconciler{Client: c, Scheme: scheme}

	// First reconcile should add finalizer
	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      polecat.Name,
			Namespace: polecat.Namespace,
		},
	})

	require.NoError(t, err)
	assert.NotZero(t, result.RequeueAfter, "should requeue after adding finalizer")

	// Verify finalizer was added
	var updated gastownv1alpha1.Polecat
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      polecat.Name,
		Namespace: polecat.Namespace,
	}, &updated)
	require.NoError(t, err)
	assert.True(t, controllerutil.ContainsFinalizer(&updated, polecatFinalizer),
		"finalizer should be added")
}

func TestPolecatReconciler_Unit_CreatesPodForWorkingState(t *testing.T) {
	scheme := setupPolecatTestScheme(t)

	polecat := newTestPolecat("test-polecat")
	// Pre-add finalizer to skip that step
	polecat.Finalizers = []string{polecatFinalizer}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(polecat).
		WithStatusSubresource(polecat).
		Build()

	r := &PolecatReconciler{Client: c, Scheme: scheme}

	// Reconcile should create pod
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      polecat.Name,
			Namespace: polecat.Namespace,
		},
	})

	require.NoError(t, err)

	// Verify Pod was created
	var pod corev1.Pod
	podName := "polecat-" + polecat.Name
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      podName,
		Namespace: polecat.Namespace,
	}, &pod)
	require.NoError(t, err)
	assert.Equal(t, polecat.Name, pod.Labels["gastown.io/polecat"])
	assert.Equal(t, polecat.Spec.Rig, pod.Labels["gastown.io/rig"])
	assert.Equal(t, polecat.Spec.BeadID, pod.Labels["gastown.io/bead"])
}

func TestPolecatReconciler_Unit_HandlesIdleState(t *testing.T) {
	scheme := setupPolecatTestScheme(t)

	polecat := newTestPolecat("idle-polecat")
	polecat.Spec.DesiredState = gastownv1alpha1.PolecatDesiredIdle
	polecat.Finalizers = []string{polecatFinalizer}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(polecat).
		WithStatusSubresource(polecat).
		Build()

	r := &PolecatReconciler{Client: c, Scheme: scheme}

	// Reconcile should set idle status
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      polecat.Name,
			Namespace: polecat.Namespace,
		},
	})

	require.NoError(t, err)

	// Verify no Pod was created
	var podList corev1.PodList
	err = c.List(context.Background(), &podList, client.InNamespace("default"))
	require.NoError(t, err)
	assert.Empty(t, podList.Items, "no pods should be created for idle polecat")

	// Verify status shows idle
	var updated gastownv1alpha1.Polecat
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      polecat.Name,
		Namespace: polecat.Namespace,
	}, &updated)
	require.NoError(t, err)
	assert.Equal(t, gastownv1alpha1.PolecatPhaseIdle, updated.Status.Phase)
}

func TestPolecatReconciler_Unit_DeletesPodWhenIdleStateRequested(t *testing.T) {
	scheme := setupPolecatTestScheme(t)

	polecat := newTestPolecat("working-to-idle")
	polecat.Spec.DesiredState = gastownv1alpha1.PolecatDesiredIdle
	polecat.Finalizers = []string{polecatFinalizer}

	// Pre-create a pod
	existingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "polecat-" + polecat.Name,
			Namespace: polecat.Namespace,
			Labels: map[string]string{
				"gastown.io/polecat": polecat.Name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "test", Image: "test:latest"},
			},
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(polecat, existingPod).
		WithStatusSubresource(polecat).
		Build()

	r := &PolecatReconciler{Client: c, Scheme: scheme}

	// Reconcile should delete pod for idle state
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      polecat.Name,
			Namespace: polecat.Namespace,
		},
	})

	require.NoError(t, err)

	// Verify Pod was deleted
	var pod corev1.Pod
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      "polecat-" + polecat.Name,
		Namespace: polecat.Namespace,
	}, &pod)
	assert.Error(t, err, "pod should be deleted")

	// Verify status shows idle
	var updated gastownv1alpha1.Polecat
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      polecat.Name,
		Namespace: polecat.Namespace,
	}, &updated)
	require.NoError(t, err)
	assert.Equal(t, gastownv1alpha1.PolecatPhaseIdle, updated.Status.Phase)
}

func TestPolecatReconciler_Unit_SyncsStatusFromRunningPod(t *testing.T) {
	scheme := setupPolecatTestScheme(t)

	polecat := newTestPolecat("running-polecat")
	polecat.Finalizers = []string{polecatFinalizer}

	// Pre-create a running pod
	existingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "polecat-" + polecat.Name,
			Namespace: polecat.Namespace,
			Labels: map[string]string{
				"gastown.io/polecat": polecat.Name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "claude", Image: "claude:latest"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(polecat, existingPod).
		WithStatusSubresource(polecat, existingPod).
		Build()

	r := &PolecatReconciler{Client: c, Scheme: scheme}

	// Reconcile should sync status from pod
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      polecat.Name,
			Namespace: polecat.Namespace,
		},
	})

	require.NoError(t, err)

	// Verify status reflects running pod
	var updated gastownv1alpha1.Polecat
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      polecat.Name,
		Namespace: polecat.Namespace,
	}, &updated)
	require.NoError(t, err)
	assert.Equal(t, gastownv1alpha1.PolecatPhaseWorking, updated.Status.Phase)
	assert.True(t, updated.Status.PodActive)
	assert.Equal(t, "polecat-"+polecat.Name, updated.Status.PodName)
}

func TestPolecatReconciler_Unit_SyncsStatusFromSucceededPod(t *testing.T) {
	scheme := setupPolecatTestScheme(t)

	polecat := newTestPolecat("succeeded-polecat")
	polecat.Finalizers = []string{polecatFinalizer}

	// Pre-create a succeeded pod
	existingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "polecat-" + polecat.Name,
			Namespace: polecat.Namespace,
			Labels: map[string]string{
				"gastown.io/polecat": polecat.Name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "claude", Image: "claude:latest"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodSucceeded,
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(polecat, existingPod).
		WithStatusSubresource(polecat, existingPod).
		Build()

	r := &PolecatReconciler{Client: c, Scheme: scheme}

	// Reconcile should sync status from pod
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      polecat.Name,
			Namespace: polecat.Namespace,
		},
	})

	require.NoError(t, err)

	// Verify status reflects succeeded pod
	var updated gastownv1alpha1.Polecat
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      polecat.Name,
		Namespace: polecat.Namespace,
	}, &updated)
	require.NoError(t, err)
	assert.Equal(t, gastownv1alpha1.PolecatPhaseDone, updated.Status.Phase)
	assert.False(t, updated.Status.PodActive)

	// Verify Available condition is True (signals merge readiness)
	var availableFound bool
	for _, cond := range updated.Status.Conditions {
		if cond.Type == ConditionAvailable && cond.Status == metav1.ConditionTrue {
			availableFound = true
			assert.Equal(t, "WorkComplete", cond.Reason)
			break
		}
	}
	assert.True(t, availableFound, "Available condition should be True")
}

func TestPolecatReconciler_Unit_SyncsStatusFromFailedPod(t *testing.T) {
	scheme := setupPolecatTestScheme(t)

	polecat := newTestPolecat("failed-polecat")
	polecat.Finalizers = []string{polecatFinalizer}

	// Pre-create a failed pod
	existingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "polecat-" + polecat.Name,
			Namespace: polecat.Namespace,
			Labels: map[string]string{
				"gastown.io/polecat": polecat.Name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "claude", Image: "claude:latest"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(polecat, existingPod).
		WithStatusSubresource(polecat, existingPod).
		Build()

	r := &PolecatReconciler{Client: c, Scheme: scheme}

	// Reconcile should sync status from pod
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      polecat.Name,
			Namespace: polecat.Namespace,
		},
	})

	require.NoError(t, err)

	// Verify status reflects failed pod
	var updated gastownv1alpha1.Polecat
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      polecat.Name,
		Namespace: polecat.Namespace,
	}, &updated)
	require.NoError(t, err)
	assert.Equal(t, gastownv1alpha1.PolecatPhaseStuck, updated.Status.Phase)
	assert.False(t, updated.Status.PodActive)

	// Verify Degraded condition is True
	var degradedFound bool
	for _, cond := range updated.Status.Conditions {
		if cond.Type == ConditionDegraded && cond.Status == metav1.ConditionTrue {
			degradedFound = true
			assert.Equal(t, "PodFailed", cond.Reason)
			break
		}
	}
	assert.True(t, degradedFound, "Degraded condition should be True")
}

func TestPolecatReconciler_Unit_HandlesTerminatedState(t *testing.T) {
	scheme := setupPolecatTestScheme(t)

	polecat := newTestPolecat("terminated-polecat")
	polecat.Spec.DesiredState = gastownv1alpha1.PolecatDesiredTerminated
	polecat.Finalizers = []string{polecatFinalizer}

	// Pre-create a pod
	existingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "polecat-" + polecat.Name,
			Namespace: polecat.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "test", Image: "test:latest"},
			},
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(polecat, existingPod).
		WithStatusSubresource(polecat).
		Build()

	r := &PolecatReconciler{Client: c, Scheme: scheme}

	// Reconcile should delete pod and set terminated status
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      polecat.Name,
			Namespace: polecat.Namespace,
		},
	})

	require.NoError(t, err)

	// Verify Pod was deleted
	var pod corev1.Pod
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      "polecat-" + polecat.Name,
		Namespace: polecat.Namespace,
	}, &pod)
	assert.Error(t, err, "pod should be deleted")

	// Verify status shows terminated
	var updated gastownv1alpha1.Polecat
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      polecat.Name,
		Namespace: polecat.Namespace,
	}, &updated)
	require.NoError(t, err)
	assert.Equal(t, gastownv1alpha1.PolecatPhaseTerminated, updated.Status.Phase)
}

func TestPolecatReconciler_Unit_HandlesMissingKubernetesSpec(t *testing.T) {
	scheme := setupPolecatTestScheme(t)

	polecat := &gastownv1alpha1.Polecat{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "bad-polecat",
			Namespace:  "default",
			Finalizers: []string{polecatFinalizer},
		},
		Spec: gastownv1alpha1.PolecatSpec{
			Rig:           "test-rig",
			DesiredState:  gastownv1alpha1.PolecatDesiredWorking,
			ExecutionMode: gastownv1alpha1.ExecutionModeKubernetes,
			// No Kubernetes spec!
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(polecat).
		WithStatusSubresource(polecat).
		Build()

	r := &PolecatReconciler{Client: c, Scheme: scheme}

	// Reconcile should not error but set stuck status
	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      polecat.Name,
			Namespace: polecat.Namespace,
		},
	})

	require.NoError(t, err)
	assert.NotZero(t, result.RequeueAfter, "should requeue")

	// Verify status shows stuck with MissingKubernetesSpec reason
	var updated gastownv1alpha1.Polecat
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      polecat.Name,
		Namespace: polecat.Namespace,
	}, &updated)
	require.NoError(t, err)
	assert.Equal(t, gastownv1alpha1.PolecatPhaseStuck, updated.Status.Phase)

	// Check for condition
	var foundCondition bool
	for _, cond := range updated.Status.Conditions {
		if cond.Type == ConditionPolecatReady && cond.Reason == "MissingKubernetesSpec" {
			foundCondition = true
			break
		}
	}
	assert.True(t, foundCondition, "should have MissingKubernetesSpec condition")
}

func TestPolecatReconciler_Unit_HandlesDeletion(t *testing.T) {
	scheme := setupPolecatTestScheme(t)

	polecat := newTestPolecat("deleting-polecat")
	polecat.Finalizers = []string{polecatFinalizer}

	// Pre-create a pod
	existingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "polecat-" + polecat.Name,
			Namespace: polecat.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "test", Image: "test:latest"},
			},
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(polecat, existingPod).
		WithStatusSubresource(polecat).
		Build()

	r := &PolecatReconciler{Client: c, Scheme: scheme}

	// Simulate deletion by calling Delete (which sets DeletionTimestamp)
	var current gastownv1alpha1.Polecat
	err := c.Get(context.Background(), types.NamespacedName{
		Name:      polecat.Name,
		Namespace: polecat.Namespace,
	}, &current)
	require.NoError(t, err)
	err = c.Delete(context.Background(), &current)
	require.NoError(t, err)

	// Reconcile should cleanup and remove finalizer
	_, err = r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      polecat.Name,
			Namespace: polecat.Namespace,
		},
	})

	require.NoError(t, err)

	// Verify Pod was deleted
	var pod corev1.Pod
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      "polecat-" + polecat.Name,
		Namespace: polecat.Namespace,
	}, &pod)
	assert.Error(t, err, "pod should be deleted")

	// After removing finalizer, the object should be deleted by the fake client
	// or have no finalizer if still present
	var updated gastownv1alpha1.Polecat
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      polecat.Name,
		Namespace: polecat.Namespace,
	}, &updated)
	if err == nil {
		assert.False(t, controllerutil.ContainsFinalizer(&updated, polecatFinalizer))
	}
	// If err != nil (not found), the object was correctly deleted
}

func TestPolecatReconciler_Unit_TableDrivenTests(t *testing.T) {
	tests := []struct {
		name     string
		objects  []client.Object
		req      reconcile.Request
		wantErr  bool
		validate func(t *testing.T, c client.Client)
	}{
		{
			name:    "handles missing polecat gracefully",
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
			name: "adds finalizer to new polecat",
			objects: []client.Object{
				&gastownv1alpha1.Polecat{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "new-polecat",
						Namespace: "default",
					},
					Spec: gastownv1alpha1.PolecatSpec{
						Rig:          "test-rig",
						DesiredState: gastownv1alpha1.PolecatDesiredIdle,
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "new-polecat",
					Namespace: "default",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, c client.Client) {
				var polecat gastownv1alpha1.Polecat
				err := c.Get(context.Background(), types.NamespacedName{
					Name:      "new-polecat",
					Namespace: "default",
				}, &polecat)
				require.NoError(t, err)
				assert.Contains(t, polecat.Finalizers, polecatFinalizer)
			},
		},
		{
			name: "sets idle phase for idle state",
			objects: []client.Object{
				&gastownv1alpha1.Polecat{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "idle-polecat",
						Namespace:  "default",
						Finalizers: []string{polecatFinalizer},
					},
					Spec: gastownv1alpha1.PolecatSpec{
						Rig:          "test-rig",
						DesiredState: gastownv1alpha1.PolecatDesiredIdle,
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "idle-polecat",
					Namespace: "default",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, c client.Client) {
				var polecat gastownv1alpha1.Polecat
				err := c.Get(context.Background(), types.NamespacedName{
					Name:      "idle-polecat",
					Namespace: "default",
				}, &polecat)
				require.NoError(t, err)
				assert.Equal(t, gastownv1alpha1.PolecatPhaseIdle, polecat.Status.Phase)
			},
		},
		{
			name: "creates pod for working state with kubernetes spec",
			objects: []client.Object{
				&gastownv1alpha1.Polecat{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "working-polecat",
						Namespace:  "default",
						Finalizers: []string{polecatFinalizer},
					},
					Spec: gastownv1alpha1.PolecatSpec{
						Rig:           "test-rig",
						DesiredState:  gastownv1alpha1.PolecatDesiredWorking,
						BeadID:        "test-bead",
						ExecutionMode: gastownv1alpha1.ExecutionModeKubernetes,
						Kubernetes: &gastownv1alpha1.KubernetesSpec{
							GitRepository: "git@github.com:test/repo.git",
							GitSecretRef: gastownv1alpha1.SecretReference{
								Name: "git-creds",
							},
							ClaudeCredsSecretRef: &gastownv1alpha1.SecretReference{
								Name: "claude-creds",
							},
						},
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "working-polecat",
					Namespace: "default",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, c client.Client) {
				var pod corev1.Pod
				err := c.Get(context.Background(), types.NamespacedName{
					Name:      "polecat-working-polecat",
					Namespace: "default",
				}, &pod)
				require.NoError(t, err)
				assert.Equal(t, "working-polecat", pod.Labels["gastown.io/polecat"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := setupPolecatTestScheme(t)

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithStatusSubresource(&gastownv1alpha1.Polecat{}).
				Build()

			r := &PolecatReconciler{Client: c, Scheme: scheme}
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
