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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
	"github.com/org/gastown-operator/pkg/gt"
)

var _ = Describe("BeadsSync Controller", func() {
	var (
		ctx        context.Context
		reconciler *BeadsSyncReconciler
		mockClient *gt.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockClient = &gt.MockClient{}
		reconciler = &BeadsSyncReconciler{
			Client:   k8sClient,
			Scheme:   k8sClient.Scheme(),
			GTClient: mockClient,
		}
	})

	Context("When syncing polecats", func() {
		It("should update polecat to Done when bead is closed externally", func() {
			// Create a polecat with Working phase
			polecat := &gastownv1alpha1.Polecat{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sync-test-polecat",
					Namespace: "default",
				},
				Spec: gastownv1alpha1.PolecatSpec{
					Rig:          "test-rig",
					DesiredState: gastownv1alpha1.PolecatDesiredWorking,
					BeadID:       "bead-123",
				},
			}
			Expect(k8sClient.Create(ctx, polecat)).To(Succeed())

			// Update status to Working with assigned bead
			polecat.Status.Phase = gastownv1alpha1.PolecatPhaseWorking
			polecat.Status.AssignedBead = "bead-123"
			Expect(k8sClient.Status().Update(ctx, polecat)).To(Succeed())

			// Mock bead status as closed
			mockClient.BeadStatusFunc = func(ctx context.Context, beadID string) (*gt.BeadStatus, error) {
				return &gt.BeadStatus{
					ID:     beadID,
					Status: "closed",
				}, nil
			}

			// Reconcile
			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      polecat.Name,
				Namespace: polecat.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(BeadsSyncInterval))

			// Verify polecat phase is now Done
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseDone))

			// Cleanup
			Expect(k8sClient.Delete(ctx, polecat)).To(Succeed())
		})

		It("should skip polecats without assigned bead", func() {
			// Create a polecat without assigned bead
			polecat := &gastownv1alpha1.Polecat{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "no-bead-polecat",
					Namespace: "default",
				},
				Spec: gastownv1alpha1.PolecatSpec{
					Rig:          "test-rig",
					DesiredState: gastownv1alpha1.PolecatDesiredIdle,
				},
			}
			Expect(k8sClient.Create(ctx, polecat)).To(Succeed())

			// BeadStatus should not be called
			mockClient.BeadStatusFunc = func(ctx context.Context, beadID string) (*gt.BeadStatus, error) {
				Fail("BeadStatus should not be called for polecat without assigned bead")
				return nil, nil
			}

			// Reconcile
			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      polecat.Name,
				Namespace: polecat.Namespace,
			}}
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Cleanup
			Expect(k8sClient.Delete(ctx, polecat)).To(Succeed())
		})

		It("should handle bead status errors gracefully", func() {
			// Create a polecat with Working phase
			polecat := &gastownv1alpha1.Polecat{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "error-test-polecat",
					Namespace: "default",
				},
				Spec: gastownv1alpha1.PolecatSpec{
					Rig:          "test-rig",
					DesiredState: gastownv1alpha1.PolecatDesiredWorking,
					BeadID:       "bead-error",
				},
			}
			Expect(k8sClient.Create(ctx, polecat)).To(Succeed())

			// Update status
			polecat.Status.Phase = gastownv1alpha1.PolecatPhaseWorking
			polecat.Status.AssignedBead = "bead-error"
			Expect(k8sClient.Status().Update(ctx, polecat)).To(Succeed())

			// Mock bead status error
			mockClient.BeadStatusFunc = func(ctx context.Context, beadID string) (*gt.BeadStatus, error) {
				return nil, errors.New("bead not found")
			}

			// Reconcile should not fail
			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      polecat.Name,
				Namespace: polecat.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(BeadsSyncInterval))

			// Polecat should remain Working
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseWorking))

			// Cleanup
			Expect(k8sClient.Delete(ctx, polecat)).To(Succeed())
		})

		It("should not update polecat if bead is still open", func() {
			// Create a polecat with Working phase
			polecat := &gastownv1alpha1.Polecat{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "open-bead-polecat",
					Namespace: "default",
				},
				Spec: gastownv1alpha1.PolecatSpec{
					Rig:          "test-rig",
					DesiredState: gastownv1alpha1.PolecatDesiredWorking,
					BeadID:       "bead-open",
				},
			}
			Expect(k8sClient.Create(ctx, polecat)).To(Succeed())

			// Update status
			polecat.Status.Phase = gastownv1alpha1.PolecatPhaseWorking
			polecat.Status.AssignedBead = "bead-open"
			Expect(k8sClient.Status().Update(ctx, polecat)).To(Succeed())

			// Mock bead status as open
			mockClient.BeadStatusFunc = func(ctx context.Context, beadID string) (*gt.BeadStatus, error) {
				return &gt.BeadStatus{
					ID:     beadID,
					Status: "in_progress",
				}, nil
			}

			// Reconcile
			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      polecat.Name,
				Namespace: polecat.Namespace,
			}}
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Polecat should remain Working
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseWorking))

			// Cleanup
			Expect(k8sClient.Delete(ctx, polecat)).To(Succeed())
		})
	})

	Context("When syncing convoys", func() {
		It("should update convoy phase when changed externally", func() {
			// Create a convoy with InProgress phase
			convoy := &gastownv1alpha1.Convoy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sync-test-convoy",
					Namespace: "default",
				},
				Spec: gastownv1alpha1.ConvoySpec{
					Description:  "Test Convoy",
					TrackedBeads: []string{"bead-1", "bead-2"},
				},
			}
			Expect(k8sClient.Create(ctx, convoy)).To(Succeed())

			// Update status
			convoy.Status.Phase = gastownv1alpha1.ConvoyPhaseInProgress
			convoy.Status.BeadsConvoyID = "convoy-123"
			Expect(k8sClient.Status().Update(ctx, convoy)).To(Succeed())

			// Mock convoy status as complete
			mockClient.ConvoyStatusFunc = func(ctx context.Context, convoyID string) (*gt.ConvoyStatus, error) {
				return &gt.ConvoyStatus{
					ID:        convoyID,
					Phase:     string(gastownv1alpha1.ConvoyPhaseComplete),
					Progress:  "5/5",
					Completed: []string{"bead-1", "bead-2", "bead-3", "bead-4", "bead-5"},
					Pending:   []string{},
				}, nil
			}

			// Reconcile - need a polecat to trigger
			polecat := &gastownv1alpha1.Polecat{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "trigger-polecat",
					Namespace: "default",
				},
				Spec: gastownv1alpha1.PolecatSpec{
					Rig:          "test-rig",
					DesiredState: gastownv1alpha1.PolecatDesiredIdle,
				},
			}
			Expect(k8sClient.Create(ctx, polecat)).To(Succeed())

			mockClient.BeadStatusFunc = func(ctx context.Context, beadID string) (*gt.BeadStatus, error) {
				return nil, errors.New("no bead")
			}

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      polecat.Name,
				Namespace: polecat.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(BeadsSyncInterval))

			// Verify convoy is updated
			var updated gastownv1alpha1.Convoy
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      convoy.Name,
				Namespace: convoy.Namespace,
			}, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.ConvoyPhaseComplete))
			Expect(updated.Status.Progress).To(Equal("5/5"))
			Expect(updated.Status.CompletedBeads).To(HaveLen(5))

			// Cleanup
			Expect(k8sClient.Delete(ctx, convoy)).To(Succeed())
			Expect(k8sClient.Delete(ctx, polecat)).To(Succeed())
		})

		It("should skip convoys that are already complete", func() {
			// Create a completed convoy
			convoy := &gastownv1alpha1.Convoy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "complete-convoy",
					Namespace: "default",
				},
				Spec: gastownv1alpha1.ConvoySpec{
					Description:  "Complete Convoy",
					TrackedBeads: []string{"bead-1"},
				},
			}
			Expect(k8sClient.Create(ctx, convoy)).To(Succeed())

			// Update status to complete
			convoy.Status.Phase = gastownv1alpha1.ConvoyPhaseComplete
			convoy.Status.BeadsConvoyID = "convoy-done"
			Expect(k8sClient.Status().Update(ctx, convoy)).To(Succeed())

			// ConvoyStatus should not be called
			mockClient.ConvoyStatusFunc = func(ctx context.Context, convoyID string) (*gt.ConvoyStatus, error) {
				if convoyID == "convoy-done" {
					Fail("ConvoyStatus should not be called for completed convoy")
				}
				return nil, errors.New("not found")
			}

			// Trigger reconcile
			polecat := &gastownv1alpha1.Polecat{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "trigger-polecat-2",
					Namespace: "default",
				},
				Spec: gastownv1alpha1.PolecatSpec{
					Rig:          "test-rig",
					DesiredState: gastownv1alpha1.PolecatDesiredIdle,
				},
			}
			Expect(k8sClient.Create(ctx, polecat)).To(Succeed())

			mockClient.BeadStatusFunc = func(ctx context.Context, beadID string) (*gt.BeadStatus, error) {
				return nil, errors.New("no bead")
			}

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      polecat.Name,
				Namespace: polecat.Namespace,
			}}
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Cleanup
			Expect(k8sClient.Delete(ctx, convoy)).To(Succeed())
			Expect(k8sClient.Delete(ctx, polecat)).To(Succeed())
		})

		It("should skip convoys without BeadsConvoyID", func() {
			// Create a convoy without BeadsConvoyID
			convoy := &gastownv1alpha1.Convoy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "no-id-convoy",
					Namespace: "default",
				},
				Spec: gastownv1alpha1.ConvoySpec{
					Description:  "No ID Convoy",
					TrackedBeads: []string{"bead-1"},
				},
			}
			Expect(k8sClient.Create(ctx, convoy)).To(Succeed())

			convoy.Status.Phase = gastownv1alpha1.ConvoyPhaseInProgress
			// No BeadsConvoyID set
			Expect(k8sClient.Status().Update(ctx, convoy)).To(Succeed())

			// ConvoyStatus should not be called
			mockClient.ConvoyStatusFunc = func(ctx context.Context, convoyID string) (*gt.ConvoyStatus, error) {
				Fail("ConvoyStatus should not be called for convoy without ID")
				return nil, nil
			}

			// Trigger reconcile
			polecat := &gastownv1alpha1.Polecat{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "trigger-polecat-3",
					Namespace: "default",
				},
				Spec: gastownv1alpha1.PolecatSpec{
					Rig:          "test-rig",
					DesiredState: gastownv1alpha1.PolecatDesiredIdle,
				},
			}
			Expect(k8sClient.Create(ctx, polecat)).To(Succeed())

			mockClient.BeadStatusFunc = func(ctx context.Context, beadID string) (*gt.BeadStatus, error) {
				return nil, errors.New("no bead")
			}

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      polecat.Name,
				Namespace: polecat.Namespace,
			}}
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Cleanup
			Expect(k8sClient.Delete(ctx, convoy)).To(Succeed())
			Expect(k8sClient.Delete(ctx, polecat)).To(Succeed())
		})

		It("should handle convoy status errors gracefully", func() {
			// Create a convoy
			convoy := &gastownv1alpha1.Convoy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "error-convoy",
					Namespace: "default",
				},
				Spec: gastownv1alpha1.ConvoySpec{
					Description:  "Error Convoy",
					TrackedBeads: []string{"bead-1"},
				},
			}
			Expect(k8sClient.Create(ctx, convoy)).To(Succeed())

			convoy.Status.Phase = gastownv1alpha1.ConvoyPhaseInProgress
			convoy.Status.BeadsConvoyID = "convoy-error"
			Expect(k8sClient.Status().Update(ctx, convoy)).To(Succeed())

			// Mock error
			mockClient.ConvoyStatusFunc = func(ctx context.Context, convoyID string) (*gt.ConvoyStatus, error) {
				return nil, errors.New("network error")
			}

			// Trigger reconcile
			polecat := &gastownv1alpha1.Polecat{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "trigger-polecat-4",
					Namespace: "default",
				},
				Spec: gastownv1alpha1.PolecatSpec{
					Rig:          "test-rig",
					DesiredState: gastownv1alpha1.PolecatDesiredIdle,
				},
			}
			Expect(k8sClient.Create(ctx, polecat)).To(Succeed())

			mockClient.BeadStatusFunc = func(ctx context.Context, beadID string) (*gt.BeadStatus, error) {
				return nil, errors.New("no bead")
			}

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      polecat.Name,
				Namespace: polecat.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			// Should not fail, just continue
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(BeadsSyncInterval))

			// Convoy should remain unchanged
			var updated gastownv1alpha1.Convoy
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      convoy.Name,
				Namespace: convoy.Namespace,
			}, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.ConvoyPhaseInProgress))

			// Cleanup
			Expect(k8sClient.Delete(ctx, convoy)).To(Succeed())
			Expect(k8sClient.Delete(ctx, polecat)).To(Succeed())
		})
	})
})
