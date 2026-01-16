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
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
	"github.com/org/gastown-operator/pkg/gt"
)

var _ = Describe("Rig Controller", func() {
	var (
		ctx        context.Context
		reconciler *RigReconciler
		mockClient *gt.MockClient
		testRig    *gastownv1alpha1.Rig
		tempDir    string
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create temp directory for rig path tests
		var err error
		tempDir, err = os.MkdirTemp("", "rig-test-*")
		Expect(err).NotTo(HaveOccurred())

		mockClient = &gt.MockClient{}
		reconciler = &RigReconciler{
			Client:   k8sClient,
			Scheme:   k8sClient.Scheme(),
			GTClient: mockClient,
		}

		testRig = &gastownv1alpha1.Rig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-rig",
			},
			Spec: gastownv1alpha1.RigSpec{
				GitURL:      "git@github.com:test/repo.git",
				BeadsPrefix: "test",
				LocalPath:   tempDir,
			},
		}
	})

	AfterEach(func() {
		// Clean up temp directory
		if tempDir != "" {
			_ = os.RemoveAll(tempDir)
		}

		// Clean up test rig
		if testRig != nil {
			_ = k8sClient.Delete(ctx, testRig)
		}
	})

	Context("When reconciling a rig with existing path", func() {
		It("should update status to Ready", func() {
			// Setup mock to return successful status
			mockClient.RigStatusFunc = func(ctx context.Context, name string) (*gt.RigStatus, error) {
				return &gt.RigStatus{
					Name:          name,
					PolecatCount:  2,
					ActiveConvoys: 1,
				}, nil
			}

			// Create the rig
			Expect(k8sClient.Create(ctx, testRig)).To(Succeed())

			// Reconcile
			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRig.Name}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(RigSyncInterval))

			// Verify status was updated
			var updatedRig gastownv1alpha1.Rig
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updatedRig)).To(Succeed())
			Expect(updatedRig.Status.Phase).To(Equal(gastownv1alpha1.RigPhaseReady))
			Expect(updatedRig.Status.PolecatCount).To(Equal(2))
			Expect(updatedRig.Status.ActiveConvoys).To(Equal(1))
		})
	})

	Context("When reconciling a rig with missing path", func() {
		It("should update status to Degraded", func() {
			// Set path to non-existent directory
			testRig.Spec.LocalPath = filepath.Join(tempDir, "does-not-exist")

			// Create the rig
			Expect(k8sClient.Create(ctx, testRig)).To(Succeed())

			// Reconcile
			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRig.Name}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter.Minutes()).To(BeNumerically(">=", 1))

			// Verify status was updated to Degraded
			var updatedRig gastownv1alpha1.Rig
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updatedRig)).To(Succeed())
			Expect(updatedRig.Status.Phase).To(Equal(gastownv1alpha1.RigPhaseDegraded))

			// Verify condition was set
			var foundCondition bool
			for _, cond := range updatedRig.Status.Conditions {
				if cond.Type == ConditionRigExists && cond.Status == metav1.ConditionFalse {
					foundCondition = true
					break
				}
			}
			Expect(foundCondition).To(BeTrue(), "Expected RigExists condition to be False")
		})
	})

	Context("When gt CLI returns an error", func() {
		It("should update status to Degraded and retry", func() {
			// Setup mock to return error
			mockClient.RigStatusFunc = func(ctx context.Context, name string) (*gt.RigStatus, error) {
				return nil, errors.New("gt CLI not available")
			}

			// Create the rig
			Expect(k8sClient.Create(ctx, testRig)).To(Succeed())

			// Reconcile
			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRig.Name}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter.Seconds()).To(BeNumerically(">", 0))

			// Verify status was updated to Degraded
			var updatedRig gastownv1alpha1.Rig
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updatedRig)).To(Succeed())
			Expect(updatedRig.Status.Phase).To(Equal(gastownv1alpha1.RigPhaseDegraded))

			// Verify Ready condition was set to False
			var foundCondition bool
			for _, cond := range updatedRig.Status.Conditions {
				if cond.Type == ConditionRigReady && cond.Status == metav1.ConditionFalse {
					foundCondition = true
					Expect(cond.Reason).To(Equal("GTCLIError"))
					break
				}
			}
			Expect(foundCondition).To(BeTrue(), "Expected Ready condition to be False")
		})
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
})
