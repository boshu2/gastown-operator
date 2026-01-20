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
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
	"github.com/org/gastown-operator/pkg/gt"
)

var _ = Describe("Polecat Controller", func() {
	var (
		ctx         context.Context
		reconciler  *PolecatReconciler
		mockClient  *gt.MockClient
		testPolecat *gastownv1alpha1.Polecat
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockClient = &gt.MockClient{}
		reconciler = &PolecatReconciler{
			Client:   k8sClient,
			Scheme:   k8sClient.Scheme(),
			GTClient: mockClient,
		}

		testPolecat = &gastownv1alpha1.Polecat{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "test-polecat",
				Namespace:  "default",
				Finalizers: []string{"gastown.io/polecat-cleanup"},
			},
			Spec: gastownv1alpha1.PolecatSpec{
				Rig:          "test-rig",
				DesiredState: gastownv1alpha1.PolecatDesiredWorking,
				BeadID:       "test-bead-123",
			},
		}
	})

	AfterEach(func() {
		// Clean up test polecat - remove finalizer first to allow deletion
		if testPolecat != nil {
			var current gastownv1alpha1.Polecat
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: testPolecat.Name, Namespace: testPolecat.Namespace}, &current); err == nil {
				current.Finalizers = nil
				_ = k8sClient.Update(ctx, &current)
			}
			_ = k8sClient.Delete(ctx, testPolecat)
		}
	})

	Context("When ensuring Working state", func() {
		It("should create polecat via sling when it doesn't exist", func() {
			var slingCalled bool
			var slingBeadID, slingRig string

			mockClient.PolecatExistsFunc = func(ctx context.Context, rig, name string) (bool, error) {
				return false, nil
			}
			mockClient.SlingFunc = func(ctx context.Context, beadID, rig string) error {
				slingCalled = true
				slingBeadID = beadID
				slingRig = rig
				return nil
			}
			mockClient.PolecatStatusFunc = func(ctx context.Context, rig, name string) (*gt.PolecatStatus, error) {
				return &gt.PolecatStatus{
					Name:          name,
					Phase:         "Working",
					AssignedBead:  "test-bead-123",
					Branch:        "polecat/test-polecat",
					TmuxSession:   "gt-test-rig-test-polecat",
					SessionActive: true,
				}, nil
			}

			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(PolecatSyncInterval))
			Expect(slingCalled).To(BeTrue())
			Expect(slingBeadID).To(Equal("test-bead-123"))
			Expect(slingRig).To(Equal("test-rig"))

			// Verify status was updated
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseWorking))
			Expect(updated.Status.AssignedBead).To(Equal("test-bead-123"))
			Expect(updated.Status.SessionActive).To(BeTrue())
		})

		It("should handle sling failure gracefully", func() {
			mockClient.PolecatExistsFunc = func(ctx context.Context, rig, name string) (bool, error) {
				return false, nil
			}
			mockClient.SlingFunc = func(ctx context.Context, beadID, rig string) error {
				return errors.New("sling failed: no available slots")
			}

			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			// Verify status shows stuck
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseStuck))

			// Verify condition was set
			var foundCondition bool
			for _, cond := range updated.Status.Conditions {
				if cond.Type == ConditionPolecatReady && cond.Status == metav1.ConditionFalse {
					foundCondition = true
					Expect(cond.Reason).To(Equal("SlingFailed"))
					break
				}
			}
			Expect(foundCondition).To(BeTrue())
		})
	})

	Context("When ensuring Idle state", func() {
		It("should handle idle polecat that does not exist locally", func() {
			testPolecat.Spec.DesiredState = gastownv1alpha1.PolecatDesiredIdle

			mockClient.PolecatExistsFunc = func(ctx context.Context, rig, name string) (bool, error) {
				return false, nil // Polecat doesn't exist locally
			}

			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(PolecatSyncInterval))

			// Verify status shows idle
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseIdle))
		})

		It("should handle status error during idle check", func() {
			testPolecat.Spec.DesiredState = gastownv1alpha1.PolecatDesiredIdle

			mockClient.PolecatExistsFunc = func(ctx context.Context, rig, name string) (bool, error) {
				return true, nil
			}
			mockClient.PolecatStatusFunc = func(ctx context.Context, rig, name string) (*gt.PolecatStatus, error) {
				return nil, errors.New("status unavailable")
			}

			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))
		})

		It("should reset a working polecat to idle", func() {
			var resetCalled bool

			testPolecat.Spec.DesiredState = gastownv1alpha1.PolecatDesiredIdle

			mockClient.PolecatExistsFunc = func(ctx context.Context, rig, name string) (bool, error) {
				return true, nil
			}
			mockClient.PolecatStatusFunc = func(ctx context.Context, rig, name string) (*gt.PolecatStatus, error) {
				return &gt.PolecatStatus{
					Name:          name,
					Phase:         "Working",
					AssignedBead:  "some-bead",
					SessionActive: true,
				}, nil
			}
			mockClient.PolecatResetFunc = func(ctx context.Context, rig, name string) error {
				resetCalled = true
				return nil
			}

			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(PolecatSyncInterval))
			Expect(resetCalled).To(BeTrue())

			// Verify status shows idle
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseIdle))
		})

		It("should handle reset failure with proper error condition (P8 fix)", func() {
			testPolecat.Spec.DesiredState = gastownv1alpha1.PolecatDesiredIdle

			mockClient.PolecatExistsFunc = func(ctx context.Context, rig, name string) (bool, error) {
				return true, nil
			}
			mockClient.PolecatStatusFunc = func(ctx context.Context, rig, name string) (*gt.PolecatStatus, error) {
				return &gt.PolecatStatus{
					Name:          name,
					Phase:         "Working",
					AssignedBead:  "some-bead",
					SessionActive: true,
				}, nil
			}
			mockClient.PolecatResetFunc = func(ctx context.Context, rig, name string) error {
				return errors.New("reset failed: tmux session locked")
			}

			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			// Verify condition shows reset failure (P8 fix verification)
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())

			var foundCondition bool
			for _, cond := range updated.Status.Conditions {
				if cond.Type == ConditionPolecatReady && cond.Status == metav1.ConditionFalse {
					foundCondition = true
					Expect(cond.Reason).To(Equal("ResetFailed"))
					break
				}
			}
			Expect(foundCondition).To(BeTrue(), "Expected ResetFailed condition (P8 fix)")
		})
	})

	Context("When ensuring Terminated state", func() {
		It("should refuse to terminate polecat with uncommitted work", func() {
			testPolecat.Spec.DesiredState = gastownv1alpha1.PolecatDesiredTerminated

			mockClient.PolecatExistsFunc = func(ctx context.Context, rig, name string) (bool, error) {
				return true, nil
			}
			mockClient.PolecatStatusFunc = func(ctx context.Context, rig, name string) (*gt.PolecatStatus, error) {
				return &gt.PolecatStatus{
					Name:          name,
					Phase:         "Working",
					CleanupStatus: "dirty",
				}, nil
			}

			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute))

			// Verify condition shows uncommitted work
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())

			var foundCondition bool
			for _, cond := range updated.Status.Conditions {
				if cond.Type == ConditionPolecatReady && cond.Reason == "UncommittedWork" {
					foundCondition = true
					break
				}
			}
			Expect(foundCondition).To(BeTrue())
		})

		It("should terminate clean polecat successfully", func() {
			var nukeCalled bool

			testPolecat.Spec.DesiredState = gastownv1alpha1.PolecatDesiredTerminated

			mockClient.PolecatExistsFunc = func(ctx context.Context, rig, name string) (bool, error) {
				return true, nil
			}
			mockClient.PolecatStatusFunc = func(ctx context.Context, rig, name string) (*gt.PolecatStatus, error) {
				return &gt.PolecatStatus{
					Name:          name,
					Phase:         "Idle",
					CleanupStatus: "clean",
				}, nil
			}
			mockClient.PolecatNukeFunc = func(ctx context.Context, rig, name string, force bool) error {
				nukeCalled = true
				Expect(force).To(BeFalse())
				return nil
			}

			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeZero())
			Expect(nukeCalled).To(BeTrue())

			// Verify status shows terminated
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseTerminated))
			Expect(updated.Status.SessionActive).To(BeFalse())
		})
	})

	Context("When polecat does not exist", func() {
		It("should return without error", func() {
			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      "non-existent",
				Namespace: "default",
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeZero())
		})
	})

	Context("When using kubernetes execution mode", func() {
		var k8sPolecat *gastownv1alpha1.Polecat

		BeforeEach(func() {
			k8sPolecat = &gastownv1alpha1.Polecat{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "k8s-polecat",
					Namespace:  "default",
					Finalizers: []string{"gastown.io/polecat-cleanup"},
				},
				Spec: gastownv1alpha1.PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  gastownv1alpha1.PolecatDesiredWorking,
					BeadID:        "test-bead-k8s",
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
		})

		It("should fail gracefully when kubernetes spec is missing", func() {
			// Create polecat with kubernetes mode but no kubernetes spec
			badPolecat := &gastownv1alpha1.Polecat{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bad-k8s-polecat",
					Namespace:  "default",
					Finalizers: []string{"gastown.io/polecat-cleanup"},
				},
				Spec: gastownv1alpha1.PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  gastownv1alpha1.PolecatDesiredWorking,
					BeadID:        "test-bead",
					ExecutionMode: gastownv1alpha1.ExecutionModeKubernetes,
					// No Kubernetes spec!
				},
			}

			Expect(k8sClient.Create(ctx, badPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      badPolecat.Name,
				Namespace: badPolecat.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute))

			// Verify status shows stuck with proper condition
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseStuck))

			var foundCondition bool
			for _, cond := range updated.Status.Conditions {
				if cond.Type == ConditionPolecatReady && cond.Reason == "MissingKubernetesSpec" {
					foundCondition = true
					break
				}
			}
			Expect(foundCondition).To(BeTrue())

			// Cleanup
			updated.Finalizers = nil
			Expect(k8sClient.Update(ctx, &updated)).To(Succeed())
			Expect(k8sClient.Delete(ctx, &updated)).To(Succeed())
		})

		AfterEach(func() {
			if k8sPolecat != nil {
				// Remove finalizer to allow deletion
				var current gastownv1alpha1.Polecat
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: k8sPolecat.Name, Namespace: k8sPolecat.Namespace}, &current); err == nil {
					current.Finalizers = nil
					_ = k8sClient.Update(ctx, &current)
				}
				_ = k8sClient.Delete(ctx, k8sPolecat)
			}
		})

		It("should create a Pod when polecat has kubernetes execution mode", func() {
			Expect(k8sClient.Create(ctx, k8sPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      k8sPolecat.Name,
				Namespace: k8sPolecat.Namespace,
			}}

			// Reconcile should create the Pod
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify Pod was created with correct spec
			var pod corev1.Pod
			podName := "polecat-" + k8sPolecat.Name
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      podName,
					Namespace: k8sPolecat.Namespace,
				}, &pod)
			}).Should(Succeed())

			// Verify Pod labels
			Expect(pod.Labels["gastown.io/polecat"]).To(Equal(k8sPolecat.Name))
			Expect(pod.Labels["gastown.io/rig"]).To(Equal(k8sPolecat.Spec.Rig))
			Expect(pod.Labels["gastown.io/bead"]).To(Equal(k8sPolecat.Spec.BeadID))

			// Verify init container exists
			Expect(pod.Spec.InitContainers).To(HaveLen(1))
			Expect(pod.Spec.InitContainers[0].Name).To(Equal("git-init"))

			// Verify main container exists
			Expect(pod.Spec.Containers).To(HaveLen(1))
			Expect(pod.Spec.Containers[0].Name).To(Equal("claude"))

			// Verify volumes
			volumeNames := make([]string, len(pod.Spec.Volumes))
			for i, v := range pod.Spec.Volumes {
				volumeNames[i] = v.Name
			}
			Expect(volumeNames).To(ContainElements("workspace", "git-creds", "claude-creds"))

			// Cleanup
			Expect(k8sClient.Delete(ctx, &pod)).To(Succeed())
		})

		It("should sync polecat status from Pod status", func() {
			Expect(k8sClient.Create(ctx, k8sPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      k8sPolecat.Name,
				Namespace: k8sPolecat.Namespace,
			}}

			// First reconcile creates the Pod
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Get the created Pod and update its status to Running
			var pod corev1.Pod
			podName := "polecat-" + k8sPolecat.Name
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      podName,
					Namespace: k8sPolecat.Namespace,
				}, &pod)
			}).Should(Succeed())

			pod.Status.Phase = corev1.PodRunning
			Expect(k8sClient.Status().Update(ctx, &pod)).To(Succeed())

			// Reconcile again to sync status
			_, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify Polecat status reflects Pod status
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseWorking))
			Expect(updated.Status.PodName).To(Equal(podName))

			// Cleanup
			Expect(k8sClient.Delete(ctx, &pod)).To(Succeed())
		})

		It("should mark polecat as Done when Pod succeeds", func() {
			Expect(k8sClient.Create(ctx, k8sPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      k8sPolecat.Name,
				Namespace: k8sPolecat.Namespace,
			}}

			// First reconcile creates the Pod
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Get the created Pod and update its status to Succeeded
			var pod corev1.Pod
			podName := "polecat-" + k8sPolecat.Name
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      podName,
					Namespace: k8sPolecat.Namespace,
				}, &pod)
			}).Should(Succeed())

			pod.Status.Phase = corev1.PodSucceeded
			Expect(k8sClient.Status().Update(ctx, &pod)).To(Succeed())

			// Reconcile again to sync status
			_, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify Polecat status shows Done
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseDone))

			// Cleanup
			Expect(k8sClient.Delete(ctx, &pod)).To(Succeed())
		})

		It("should mark polecat as Stuck when Pod fails", func() {
			Expect(k8sClient.Create(ctx, k8sPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      k8sPolecat.Name,
				Namespace: k8sPolecat.Namespace,
			}}

			// First reconcile creates the Pod
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Get the created Pod and update its status to Failed
			var pod corev1.Pod
			podName := "polecat-" + k8sPolecat.Name
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      podName,
					Namespace: k8sPolecat.Namespace,
				}, &pod)
			}).Should(Succeed())

			pod.Status.Phase = corev1.PodFailed
			Expect(k8sClient.Status().Update(ctx, &pod)).To(Succeed())

			// Reconcile again to sync status
			_, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify Polecat status shows Stuck
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseStuck))

			// Cleanup
			Expect(k8sClient.Delete(ctx, &pod)).To(Succeed())
		})

		It("should delete Pod when polecat is terminated", func() {
			k8sPolecat.Spec.DesiredState = gastownv1alpha1.PolecatDesiredTerminated

			Expect(k8sClient.Create(ctx, k8sPolecat)).To(Succeed())

			// First create the Pod by temporarily setting to Working
			k8sPolecat.Spec.DesiredState = gastownv1alpha1.PolecatDesiredWorking
			Expect(k8sClient.Update(ctx, k8sPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      k8sPolecat.Name,
				Namespace: k8sPolecat.Namespace,
			}}

			// Create the Pod
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify Pod exists
			var pod corev1.Pod
			podName := "polecat-" + k8sPolecat.Name
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      podName,
					Namespace: k8sPolecat.Namespace,
				}, &pod)
			}).Should(Succeed())

			// Now set to terminated
			Expect(k8sClient.Get(ctx, req.NamespacedName, k8sPolecat)).To(Succeed())
			k8sPolecat.Spec.DesiredState = gastownv1alpha1.PolecatDesiredTerminated
			Expect(k8sClient.Update(ctx, k8sPolecat)).To(Succeed())

			// Reconcile to delete Pod
			_, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify Pod was deleted
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      podName,
					Namespace: k8sPolecat.Namespace,
				}, &pod)
				return apierrors.IsNotFound(err)
			}).Should(BeTrue())

			// Verify Polecat status shows Terminated
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseTerminated))
		})
	})

	Context("When handling nuke failures", func() {
		It("should handle nuke failure during termination", func() {
			testPolecat.Spec.DesiredState = gastownv1alpha1.PolecatDesiredTerminated

			mockClient.PolecatExistsFunc = func(ctx context.Context, rig, name string) (bool, error) {
				return true, nil
			}
			mockClient.PolecatStatusFunc = func(ctx context.Context, rig, name string) (*gt.PolecatStatus, error) {
				return &gt.PolecatStatus{
					Name:          name,
					Phase:         "Idle",
					CleanupStatus: "clean",
				}, nil
			}
			mockClient.PolecatNukeFunc = func(ctx context.Context, rig, name string, force bool) error {
				return errors.New("nuke failed: tmux not responding")
			}

			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			// Verify condition shows nuke failed
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())

			var foundCondition bool
			for _, cond := range updated.Status.Conditions {
				if cond.Type == ConditionPolecatReady && cond.Reason == "NukeFailed" {
					foundCondition = true
					break
				}
			}
			Expect(foundCondition).To(BeTrue())
		})
	})

	Context("When handling polecat status errors", func() {
		It("should handle gt CLI status errors in working state", func() {
			mockClient.PolecatExistsFunc = func(ctx context.Context, rig, name string) (bool, error) {
				return true, nil // Already exists
			}
			mockClient.PolecatStatusFunc = func(ctx context.Context, rig, name string) (*gt.PolecatStatus, error) {
				return nil, errors.New("gt CLI not responding")
			}

			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			// Verify condition shows error
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())

			var foundCondition bool
			for _, cond := range updated.Status.Conditions {
				if cond.Type == ConditionPolecatReady && cond.Reason == "GTCLIError" {
					foundCondition = true
					break
				}
			}
			Expect(foundCondition).To(BeTrue())
		})

		It("should handle polecat existence check errors", func() {
			mockClient.PolecatExistsFunc = func(ctx context.Context, rig, name string) (bool, error) {
				return false, errors.New("network error")
			}

			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}
			_, err := reconciler.Reconcile(ctx, req)

			// Should return error since existence check failed
			Expect(err).To(HaveOccurred())
		})
	})

	Context("When polecat already terminated", func() {
		It("should handle termination of already-nuked polecat", func() {
			testPolecat.Spec.DesiredState = gastownv1alpha1.PolecatDesiredTerminated

			mockClient.PolecatExistsFunc = func(ctx context.Context, rig, name string) (bool, error) {
				return false, nil // Already gone
			}

			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeZero()) // Should not requeue

			// Verify status shows terminated
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseTerminated))
		})
	})

	Context("When syncing status from gt CLI", func() {
		It("should sync LastActivity when present", func() {
			now := time.Now()
			testPolecat.Spec.DesiredState = gastownv1alpha1.PolecatDesiredWorking

			mockClient.PolecatExistsFunc = func(ctx context.Context, rig, name string) (bool, error) {
				return true, nil
			}
			mockClient.PolecatStatusFunc = func(ctx context.Context, rig, name string) (*gt.PolecatStatus, error) {
				return &gt.PolecatStatus{
					Name:          name,
					Phase:         "Working",
					AssignedBead:  "bead-123",
					Branch:        "polecat/test-branch",
					WorktreePath:  "/path/to/worktree",
					TmuxSession:   "gt-test-rig-test-polecat",
					SessionActive: true,
					LastActivity:  now,
					CleanupStatus: "clean",
				}, nil
			}

			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify all status fields synced
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Branch).To(Equal("polecat/test-branch"))
			Expect(updated.Status.WorktreePath).To(Equal("/path/to/worktree"))
			Expect(updated.Status.TmuxSession).To(Equal("gt-test-rig-test-polecat"))
			Expect(updated.Status.CleanupStatus).To(Equal(gastownv1alpha1.CleanupStatusClean))
			Expect(updated.Status.LastActivity).NotTo(BeNil())
		})
	})

	Context("When handling finalizers", func() {
		It("should add finalizer on first reconcile", func() {
			// Create polecat without finalizer
			polecat := &gastownv1alpha1.Polecat{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "finalizer-test",
					Namespace: "default",
				},
				Spec: gastownv1alpha1.PolecatSpec{
					Rig:          "test-rig",
					DesiredState: gastownv1alpha1.PolecatDesiredIdle,
				},
			}

			mockClient.PolecatExistsFunc = func(ctx context.Context, rig, name string) (bool, error) {
				return false, nil
			}

			Expect(k8sClient.Create(ctx, polecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      polecat.Name,
				Namespace: polecat.Namespace,
			}}

			// First reconcile should add finalizer and request requeue
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Millisecond))

			// Verify finalizer was added
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Finalizers).To(ContainElement("gastown.io/polecat-cleanup"))

			// Cleanup
			updated.Finalizers = nil
			Expect(k8sClient.Update(ctx, &updated)).To(Succeed())
			Expect(k8sClient.Delete(ctx, &updated)).To(Succeed())
		})

		It("should cleanup local polecat on deletion", func() {
			var nukeCalled bool
			var nukeForce bool

			polecat := &gastownv1alpha1.Polecat{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "local-delete-test",
					Namespace:  "default",
					Finalizers: []string{"gastown.io/polecat-cleanup"},
				},
				Spec: gastownv1alpha1.PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  gastownv1alpha1.PolecatDesiredIdle,
					ExecutionMode: gastownv1alpha1.ExecutionModeLocal,
				},
			}

			mockClient.PolecatExistsFunc = func(ctx context.Context, rig, name string) (bool, error) {
				return true, nil
			}
			mockClient.PolecatNukeFunc = func(ctx context.Context, rig, name string, force bool) error {
				nukeCalled = true
				nukeForce = force
				return nil
			}

			Expect(k8sClient.Create(ctx, polecat)).To(Succeed())

			// Mark for deletion
			Expect(k8sClient.Delete(ctx, polecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      polecat.Name,
				Namespace: polecat.Namespace,
			}}

			// Reconcile should trigger cleanup
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(nukeCalled).To(BeTrue())
			Expect(nukeForce).To(BeTrue()) // Force on deletion

			// Verify polecat is deleted (finalizer removed)
			Eventually(func() bool {
				var p gastownv1alpha1.Polecat
				err := k8sClient.Get(ctx, req.NamespacedName, &p)
				return apierrors.IsNotFound(err)
			}).Should(BeTrue())
		})

		It("should cleanup kubernetes polecat Pod on deletion", func() {
			k8sPolecat := &gastownv1alpha1.Polecat{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "k8s-delete-test",
					Namespace:  "default",
					Finalizers: []string{"gastown.io/polecat-cleanup"},
				},
				Spec: gastownv1alpha1.PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  gastownv1alpha1.PolecatDesiredIdle,
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

			Expect(k8sClient.Create(ctx, k8sPolecat)).To(Succeed())

			// Create a Pod that would be cleaned up
			podName := "polecat-" + k8sPolecat.Name
			testPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podName,
					Namespace: k8sPolecat.Namespace,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "test", Image: "busybox"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, testPod)).To(Succeed())

			// Mark for deletion
			Expect(k8sClient.Delete(ctx, k8sPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      k8sPolecat.Name,
				Namespace: k8sPolecat.Namespace,
			}}

			// Reconcile should trigger cleanup
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify Pod was deleted
			Eventually(func() bool {
				var p corev1.Pod
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      podName,
					Namespace: k8sPolecat.Namespace,
				}, &p)
				return apierrors.IsNotFound(err)
			}).Should(BeTrue())

			// Verify polecat is deleted (finalizer removed)
			Eventually(func() bool {
				var p gastownv1alpha1.Polecat
				err := k8sClient.Get(ctx, req.NamespacedName, &p)
				return apierrors.IsNotFound(err)
			}).Should(BeTrue())
		})
	})
})
