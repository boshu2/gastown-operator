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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
)

var _ = Describe("Rig Controller", func() {
	var (
		ctx        context.Context
		reconciler *RigReconciler
		testRig    *gastownv1alpha1.Rig
	)

	BeforeEach(func() {
		ctx = context.Background()

		reconciler = &RigReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		testRig = &gastownv1alpha1.Rig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-rig",
			},
			Spec: gastownv1alpha1.RigSpec{
				GitURL:      "git@github.com:test/repo.git",
				BeadsPrefix: "test",
			},
		}
	})

	AfterEach(func() {
		// Clean up test rig - remove finalizer first
		if testRig != nil {
			var current gastownv1alpha1.Rig
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: testRig.Name}, &current); err == nil {
				current.Finalizers = nil
				_ = k8sClient.Update(ctx, &current)
			}
			_ = k8sClient.Delete(ctx, testRig)

			// Clean up any created Witness/Refinery
			witness := &gastownv1alpha1.Witness{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: testRig.Name + "-witness", Namespace: "default"}, witness); err == nil {
				_ = k8sClient.Delete(ctx, witness)
			}
			refinery := &gastownv1alpha1.Refinery{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: testRig.Name + "-refinery", Namespace: "default"}, refinery); err == nil {
				_ = k8sClient.Delete(ctx, refinery)
			}
		}
	})

	Context("When rig does not exist", func() {
		It("should return without error", func() {
			// Don't create the rig, just try to reconcile
			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "non-existent-rig"}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeZero())
		})
	})

	Context("When auto-provisioning children", func() {
		BeforeEach(func() {
			// Set the namespace env var for tests
			GinkgoT().Setenv("GASTOWN_NAMESPACE", "default")
		})

		It("should add finalizer on first reconcile", func() {
			Expect(k8sClient.Create(ctx, testRig)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRig.Name}}
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify finalizer was added
			var updated gastownv1alpha1.Rig
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testRig.Name}, &updated)).To(Succeed())
			Expect(updated.Finalizers).To(ContainElement("gastown.io/rig-cleanup"))
		})

		It("should create Witness CR for rig", func() {
			Expect(k8sClient.Create(ctx, testRig)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRig.Name}}

			// First reconcile adds finalizer
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile creates children
			_, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify Witness was created
			witness := &gastownv1alpha1.Witness{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      testRig.Name + "-witness",
					Namespace: "default",
				}, witness)
			}).Should(Succeed())

			Expect(witness.Spec.RigRef).To(Equal(testRig.Name))
			Expect(witness.Labels["gastown.io/rig-owner"]).To(Equal(testRig.Name))
		})

		It("should create Refinery CR for rig", func() {
			Expect(k8sClient.Create(ctx, testRig)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRig.Name}}

			// First reconcile adds finalizer
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile creates children
			_, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify Refinery was created
			refinery := &gastownv1alpha1.Refinery{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      testRig.Name + "-refinery",
					Namespace: "default",
				}, refinery)
			}).Should(Succeed())

			Expect(refinery.Spec.RigRef).To(Equal(testRig.Name))
			Expect(refinery.Spec.TargetBranch).To(Equal("main"))
			Expect(refinery.Labels["gastown.io/rig-owner"]).To(Equal(testRig.Name))
		})

		It("should update status after creating children", func() {
			Expect(k8sClient.Create(ctx, testRig)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRig.Name}}

			// Reconcile until children are created
			for i := 0; i < 3; i++ {
				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
			}

			// Verify status was updated
			var updated gastownv1alpha1.Rig
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testRig.Name}, &updated)).To(Succeed())
			Expect(updated.Status.WitnessCreated).To(BeTrue())
			Expect(updated.Status.RefineryCreated).To(BeTrue())
			Expect(updated.Status.ChildNamespace).To(Equal("default"))
		})

		It("should not recreate children if already created", func() {
			Expect(k8sClient.Create(ctx, testRig)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRig.Name}}

			// Reconcile multiple times
			for i := 0; i < 5; i++ {
				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
			}

			// Verify only one Witness exists
			witnessList := &gastownv1alpha1.WitnessList{}
			Expect(k8sClient.List(ctx, witnessList)).To(Succeed())
			witnessCount := 0
			for _, w := range witnessList.Items {
				if w.Spec.RigRef == testRig.Name {
					witnessCount++
				}
			}
			Expect(witnessCount).To(Equal(1))
		})

		// Note: Tests for Ready condition and Phase=Ready are skipped in envtest because
		// they require field indexers which are only set up when using a full manager.
		// When the polecat list fails due to missing indexer, the rig goes to Degraded.
		// These are tested in integration tests with a real controller manager.

		It("should set Ready condition after children are created (Degraded in envtest due to missing indexer)", func() {
			Expect(k8sClient.Create(ctx, testRig)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRig.Name}}

			// Reconcile until stable
			for i := 0; i < 3; i++ {
				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
			}

			// Verify Ready condition exists (status depends on field indexer availability)
			var updated gastownv1alpha1.Rig
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testRig.Name}, &updated)).To(Succeed())

			var readyCondition *metav1.Condition
			for i := range updated.Status.Conditions {
				if updated.Status.Conditions[i].Type == ConditionRigReady {
					readyCondition = &updated.Status.Conditions[i]
					break
				}
			}
			Expect(readyCondition).NotTo(BeNil())
			// In envtest without field indexers, this will be False/ListFailed
			// In real deployment with manager, this would be True/Ready
		})
	})

	Context("When deleting a rig with children", func() {
		BeforeEach(func() {
			GinkgoT().Setenv("GASTOWN_NAMESPACE", "default")
		})

		It("should delete Witness and Refinery when rig is deleted", func() {
			Expect(k8sClient.Create(ctx, testRig)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRig.Name}}

			// Reconcile to create children
			for i := 0; i < 3; i++ {
				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
			}

			// Verify children were created
			witness := &gastownv1alpha1.Witness{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      testRig.Name + "-witness",
				Namespace: "default",
			}, witness)).To(Succeed())

			refinery := &gastownv1alpha1.Refinery{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      testRig.Name + "-refinery",
				Namespace: "default",
			}, refinery)).To(Succeed())

			// Delete the rig
			var current gastownv1alpha1.Rig
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testRig.Name}, &current)).To(Succeed())
			Expect(k8sClient.Delete(ctx, &current)).To(Succeed())

			// Reconcile to handle deletion
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify children were deleted
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      testRig.Name + "-witness",
				Namespace: "default",
			}, witness)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())

			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      testRig.Name + "-refinery",
				Namespace: "default",
			}, refinery)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})

		It("should remove finalizer after cleanup", func() {
			Expect(k8sClient.Create(ctx, testRig)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRig.Name}}

			// Reconcile to add finalizer and create children
			for i := 0; i < 3; i++ {
				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
			}

			// Verify finalizer was added
			var current gastownv1alpha1.Rig
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testRig.Name}, &current)).To(Succeed())
			Expect(controllerutil.ContainsFinalizer(&current, "gastown.io/rig-cleanup")).To(BeTrue())

			// Delete the rig
			Expect(k8sClient.Delete(ctx, &current)).To(Succeed())

			// Reconcile to handle deletion
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify finalizer was removed (rig should be gone or have no finalizer)
			var deleted gastownv1alpha1.Rig
			err = k8sClient.Get(ctx, types.NamespacedName{Name: testRig.Name}, &deleted)
			// Either NotFound or finalizer removed
			if err == nil {
				Expect(controllerutil.ContainsFinalizer(&deleted, "gastown.io/rig-cleanup")).To(BeFalse())
			} else {
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}
		})

		It("should handle deletion when children are already gone", func() {
			Expect(k8sClient.Create(ctx, testRig)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRig.Name}}

			// Reconcile to add finalizer and create children
			for i := 0; i < 3; i++ {
				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
			}

			// Manually delete the children before rig deletion
			witness := &gastownv1alpha1.Witness{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      testRig.Name + "-witness",
				Namespace: "default",
			}, witness)).To(Succeed())
			Expect(k8sClient.Delete(ctx, witness)).To(Succeed())

			refinery := &gastownv1alpha1.Refinery{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      testRig.Name + "-refinery",
				Namespace: "default",
			}, refinery)).To(Succeed())
			Expect(k8sClient.Delete(ctx, refinery)).To(Succeed())

			// Delete the rig
			var current gastownv1alpha1.Rig
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: testRig.Name}, &current)).To(Succeed())
			Expect(k8sClient.Delete(ctx, &current)).To(Succeed())

			// Reconcile should succeed even without children
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify finalizer was removed
			var deleted gastownv1alpha1.Rig
			err = k8sClient.Get(ctx, types.NamespacedName{Name: testRig.Name}, &deleted)
			if err == nil {
				Expect(controllerutil.ContainsFinalizer(&deleted, "gastown.io/rig-cleanup")).To(BeFalse())
			}
		})
	})

	// Note: Tests for counting polecats/convoys are skipped in envtest because they
	// require field indexers which are only set up when using a full manager.
	// These are tested in integration tests with a real controller manager.
})
