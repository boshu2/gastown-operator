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

var _ = Describe("Convoy Controller", func() {
	var (
		ctx        context.Context
		reconciler *ConvoyReconciler
		testConvoy *gastownv1alpha1.Convoy
	)

	BeforeEach(func() {
		ctx = context.Background()
		reconciler = &ConvoyReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		testConvoy = &gastownv1alpha1.Convoy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-convoy",
				Namespace: "default",
			},
			Spec: gastownv1alpha1.ConvoySpec{
				Description: "Test wave 1",
				TrackedBeads: []string{
					"test-bead-1",
					"test-bead-2",
					"test-bead-3",
				},
			},
		}
	})

	AfterEach(func() {
		// Clean up test convoy
		if testConvoy != nil {
			_ = k8sClient.Delete(ctx, testConvoy)
		}
	})

	Context("When creating a new convoy", func() {
		It("should initialize convoy status", func() {
			Expect(k8sClient.Create(ctx, testConvoy)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testConvoy.Name,
				Namespace: testConvoy.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(ConvoySyncInterval))

			// Verify status was updated
			var updated gastownv1alpha1.Convoy
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.ConvoyPhaseInProgress))
			Expect(updated.Status.StartedAt).NotTo(BeNil())
			Expect(updated.Status.PendingBeads).To(Equal([]string{"test-bead-1", "test-bead-2", "test-bead-3"}))
		})
	})

	Context("When syncing convoy progress", func() {
		It("should track completed beads from polecat status", func() {
			// Create a polecat with Done status for one of the beads
			donePolecat := &gastownv1alpha1.Polecat{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "done-polecat",
					Namespace: "default",
				},
				Spec: gastownv1alpha1.PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  gastownv1alpha1.PolecatDesiredWorking,
					ExecutionMode: gastownv1alpha1.ExecutionModeKubernetes,
					Kubernetes: &gastownv1alpha1.KubernetesSpec{
						GitRepository:        "git@github.com:org/repo.git",
						GitSecretRef:         gastownv1alpha1.SecretReference{Name: "git-secret"},
						ClaudeCredsSecretRef: &gastownv1alpha1.SecretReference{Name: "claude-creds"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, donePolecat)).To(Succeed())
			defer func() { _ = k8sClient.Delete(ctx, donePolecat) }()

			// Update polecat status to show assigned bead and Done phase
			var createdPolecat gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      donePolecat.Name,
				Namespace: donePolecat.Namespace,
			}, &createdPolecat)).To(Succeed())
			createdPolecat.Status.AssignedBead = "test-bead-1"
			createdPolecat.Status.Phase = gastownv1alpha1.PolecatPhaseDone
			Expect(k8sClient.Status().Update(ctx, &createdPolecat)).To(Succeed())

			// Pre-set convoy as in progress
			testConvoy.Status.Phase = gastownv1alpha1.ConvoyPhaseInProgress
			Expect(k8sClient.Create(ctx, testConvoy)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testConvoy.Name,
				Namespace: testConvoy.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(ConvoySyncInterval))

			// Verify progress was updated
			var updated gastownv1alpha1.Convoy
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Progress).To(Equal("1/3"))
			Expect(updated.Status.CompletedBeads).To(ContainElement("test-bead-1"))
			Expect(updated.Status.PendingBeads).To(ContainElements("test-bead-2", "test-bead-3"))
		})
	})

	Context("When convoy completes", func() {
		It("should mark as complete and not requeue", func() {
			// Create polecats with Done status for all beads
			for i, beadID := range []string{"test-bead-1", "test-bead-2", "test-bead-3"} {
				polecat := &gastownv1alpha1.Polecat{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "done-polecat-" + string(rune('a'+i)),
						Namespace: "default",
					},
					Spec: gastownv1alpha1.PolecatSpec{
						Rig:           "test-rig",
						DesiredState:  gastownv1alpha1.PolecatDesiredWorking,
						ExecutionMode: gastownv1alpha1.ExecutionModeKubernetes,
						Kubernetes: &gastownv1alpha1.KubernetesSpec{
							GitRepository:        "git@github.com:org/repo.git",
							GitSecretRef:         gastownv1alpha1.SecretReference{Name: "git-secret"},
							ClaudeCredsSecretRef: &gastownv1alpha1.SecretReference{Name: "claude-creds"},
						},
					},
				}
				Expect(k8sClient.Create(ctx, polecat)).To(Succeed())
				defer func(p *gastownv1alpha1.Polecat) { _ = k8sClient.Delete(ctx, p) }(polecat)

				var created gastownv1alpha1.Polecat
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      polecat.Name,
					Namespace: polecat.Namespace,
				}, &created)).To(Succeed())
				created.Status.AssignedBead = beadID
				created.Status.Phase = gastownv1alpha1.PolecatPhaseDone
				Expect(k8sClient.Status().Update(ctx, &created)).To(Succeed())
			}

			// Pre-set convoy as in progress
			testConvoy.Status.Phase = gastownv1alpha1.ConvoyPhaseInProgress
			Expect(k8sClient.Create(ctx, testConvoy)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testConvoy.Name,
				Namespace: testConvoy.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			// Should NOT requeue when complete
			Expect(result.RequeueAfter).To(BeZero())

			// Verify status shows complete
			var updated gastownv1alpha1.Convoy
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.ConvoyPhaseComplete))
			Expect(updated.Status.CompletedAt).NotTo(BeNil())

			// Verify Complete condition was set
			var foundCondition bool
			for _, cond := range updated.Status.Conditions {
				if cond.Type == ConditionConvoyComplete && cond.Status == metav1.ConditionTrue {
					foundCondition = true
					break
				}
			}
			Expect(foundCondition).To(BeTrue())
		})
	})

	Context("When convoy is already complete", func() {
		It("should not requeue", func() {
			// Create the convoy first
			Expect(k8sClient.Create(ctx, testConvoy)).To(Succeed())

			// Then update status separately (envtest ignores status on Create)
			var created gastownv1alpha1.Convoy
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      testConvoy.Name,
				Namespace: testConvoy.Namespace,
			}, &created)).To(Succeed())

			created.Status.Phase = gastownv1alpha1.ConvoyPhaseComplete
			Expect(k8sClient.Status().Update(ctx, &created)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testConvoy.Name,
				Namespace: testConvoy.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeZero())
		})
	})

	Context("When convoy does not exist", func() {
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
