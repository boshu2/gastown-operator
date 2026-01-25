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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

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
		// Clean up test rig
		if testRig != nil {
			_ = k8sClient.Delete(ctx, testRig)
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

	// Note: Tests for counting polecats/convoys are skipped in envtest because they
	// require field indexers which are only set up when using a full manager.
	// These are tested in integration tests with a real controller manager.
})
