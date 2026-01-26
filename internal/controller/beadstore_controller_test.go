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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
)

var _ = Describe("BeadStore Controller", func() {
	var (
		ctx           context.Context
		reconciler    *BeadStoreReconciler
		testBeadStore *gastownv1alpha1.BeadStore
		testRig       *gastownv1alpha1.Rig
	)

	BeforeEach(func() {
		ctx = context.Background()
		reconciler = &BeadStoreReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		testBeadStore = &gastownv1alpha1.BeadStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-beadstore",
				Namespace: "default",
			},
			Spec: gastownv1alpha1.BeadStoreSpec{
				RigRef: "test-rig",
				Prefix: "gt-",
			},
		}

		testRig = &gastownv1alpha1.Rig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-rig",
			},
			Spec: gastownv1alpha1.RigSpec{
				GitURL:      "git@github.com:test/repo.git",
				BeadsPrefix: "gt",
			},
		}
	})

	AfterEach(func() {
		// Clean up test beadstore - remove finalizer first
		if testBeadStore != nil {
			var current gastownv1alpha1.BeadStore
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: testBeadStore.Name, Namespace: testBeadStore.Namespace}, &current); err == nil {
				current.Finalizers = nil
				_ = k8sClient.Update(ctx, &current)
			}
			_ = k8sClient.Delete(ctx, testBeadStore)
		}

		// Clean up test rig
		if testRig != nil {
			var currentRig gastownv1alpha1.Rig
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: testRig.Name}, &currentRig); err == nil {
				currentRig.Finalizers = nil
				_ = k8sClient.Update(ctx, &currentRig)
			}
			_ = k8sClient.Delete(ctx, testRig)
		}
	})

	Context("When beadstore does not exist", func() {
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

	Context("When reconciling a new beadstore", func() {
		It("should add finalizer on first reconcile", func() {
			Expect(k8sClient.Create(ctx, testBeadStore)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testBeadStore.Name,
				Namespace: testBeadStore.Namespace,
			}}
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify finalizer was added
			var updated gastownv1alpha1.BeadStore
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      testBeadStore.Name,
				Namespace: testBeadStore.Namespace,
			}, &updated)).To(Succeed())
			Expect(updated.Finalizers).To(ContainElement("gastown.io/beadstore-cleanup"))
		})
	})

	Context("When rig reference is invalid", func() {
		It("should set Ready condition to false with RigNotFound reason", func() {
			Expect(k8sClient.Create(ctx, testBeadStore)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testBeadStore.Name,
				Namespace: testBeadStore.Namespace,
			}}

			// First reconcile adds finalizer
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile validates rig (which doesn't exist)
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			// Verify status reflects rig not found
			var updated gastownv1alpha1.BeadStore
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      testBeadStore.Name,
				Namespace: testBeadStore.Namespace,
			}, &updated)).To(Succeed())

			Expect(updated.Status.Phase).To(Equal(PhasePending))

			// Check Ready condition
			var readyCondition *metav1.Condition
			for i := range updated.Status.Conditions {
				if updated.Status.Conditions[i].Type == ConditionBeadStoreReady {
					readyCondition = &updated.Status.Conditions[i]
					break
				}
			}
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(readyCondition.Reason).To(Equal("RigNotFound"))
		})
	})

	Context("When rig exists", func() {
		BeforeEach(func() {
			// Create the rig first
			Expect(k8sClient.Create(ctx, testRig)).To(Succeed())
		})

		It("should set conditions to true and phase to Synced", func() {
			Expect(k8sClient.Create(ctx, testBeadStore)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testBeadStore.Name,
				Namespace: testBeadStore.Namespace,
			}}

			// First reconcile adds finalizer
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile validates rig and syncs
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(BeadStoreSyncInterval))

			// Verify status is synced
			var updated gastownv1alpha1.BeadStore
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      testBeadStore.Name,
				Namespace: testBeadStore.Namespace,
			}, &updated)).To(Succeed())

			Expect(updated.Status.Phase).To(Equal(PhaseSynced))
			Expect(updated.Status.LastSyncTime).NotTo(BeNil())

			// Check Synced condition
			var syncedCondition *metav1.Condition
			for i := range updated.Status.Conditions {
				if updated.Status.Conditions[i].Type == ConditionBeadStoreSynced {
					syncedCondition = &updated.Status.Conditions[i]
					break
				}
			}
			Expect(syncedCondition).NotTo(BeNil())
			Expect(syncedCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(syncedCondition.Reason).To(Equal("SyncSucceeded"))

			// Check Ready condition
			var readyCondition *metav1.Condition
			for i := range updated.Status.Conditions {
				if updated.Status.Conditions[i].Type == ConditionBeadStoreReady {
					readyCondition = &updated.Status.Conditions[i]
					break
				}
			}
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(readyCondition.Reason).To(Equal("Ready"))
		})

		It("should use custom sync interval when specified", func() {
			customInterval := 10 * time.Minute
			testBeadStore.Spec.SyncInterval = &metav1.Duration{Duration: customInterval}
			Expect(k8sClient.Create(ctx, testBeadStore)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testBeadStore.Name,
				Namespace: testBeadStore.Namespace,
			}}

			// First reconcile adds finalizer
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile syncs
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(customInterval))
		})
	})

	Context("When beadstore is deleted", func() {
		It("should remove finalizer and allow deletion", func() {
			Expect(k8sClient.Create(ctx, testRig)).To(Succeed())
			Expect(k8sClient.Create(ctx, testBeadStore)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testBeadStore.Name,
				Namespace: testBeadStore.Namespace,
			}}

			// Reconcile to add finalizer and sync
			for i := 0; i < 2; i++ {
				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
			}

			// Verify finalizer was added
			var updated gastownv1alpha1.BeadStore
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      testBeadStore.Name,
				Namespace: testBeadStore.Namespace,
			}, &updated)).To(Succeed())
			Expect(controllerutil.ContainsFinalizer(&updated, "gastown.io/beadstore-cleanup")).To(BeTrue())

			// Delete the beadstore
			Expect(k8sClient.Delete(ctx, &updated)).To(Succeed())

			// Reconcile to handle deletion
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify finalizer was removed (beadstore should be gone)
			var deleted gastownv1alpha1.BeadStore
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      testBeadStore.Name,
				Namespace: testBeadStore.Namespace,
			}, &deleted)
			// Either NotFound error or finalizer was removed
			if err == nil {
				Expect(controllerutil.ContainsFinalizer(&deleted, "gastown.io/beadstore-cleanup")).To(BeFalse())
			}
		})
	})

	Context("When reconciliation is idempotent", func() {
		It("should not change status on repeated reconciles", func() {
			Expect(k8sClient.Create(ctx, testRig)).To(Succeed())
			Expect(k8sClient.Create(ctx, testBeadStore)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testBeadStore.Name,
				Namespace: testBeadStore.Namespace,
			}}

			// Reconcile multiple times
			for i := 0; i < 5; i++ {
				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
			}

			// Get final state
			var final gastownv1alpha1.BeadStore
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      testBeadStore.Name,
				Namespace: testBeadStore.Namespace,
			}, &final)).To(Succeed())

			Expect(final.Status.Phase).To(Equal(PhaseSynced))
			Expect(final.Status.LastSyncTime).NotTo(BeNil())

			// Verify we have exactly 2 conditions (Synced and Ready)
			conditionTypes := make(map[string]bool)
			for _, c := range final.Status.Conditions {
				conditionTypes[c.Type] = true
			}
			Expect(conditionTypes).To(HaveLen(2))
			Expect(conditionTypes).To(HaveKey(ConditionBeadStoreSynced))
			Expect(conditionTypes).To(HaveKey(ConditionBeadStoreReady))
		})
	})
})
