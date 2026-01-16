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
				Name:      "test-polecat",
				Namespace: "default",
			},
			Spec: gastownv1alpha1.PolecatSpec{
				Rig:          "test-rig",
				DesiredState: gastownv1alpha1.PolecatDesiredWorking,
				BeadID:       "test-bead-123",
			},
		}
	})

	AfterEach(func() {
		// Clean up test polecat
		if testPolecat != nil {
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
})
