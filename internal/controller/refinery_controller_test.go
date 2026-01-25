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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
	"github.com/org/gastown-operator/internal/git"
)

// mockGitClient implements git.GitClient for testing.
type mockGitClient struct {
	cloneErr    error
	mergeErr    error
	mergeResult *git.MergeResult
}

func (m *mockGitClient) Clone(ctx context.Context) error {
	return m.cloneErr
}

func (m *mockGitClient) MergeBranch(ctx context.Context, opts git.MergeOptions) (*git.MergeResult, error) {
	if m.mergeErr != nil {
		return nil, m.mergeErr
	}
	if m.mergeResult != nil {
		return m.mergeResult, nil
	}
	return &git.MergeResult{
		Success:      true,
		MergedCommit: "abc123",
	}, nil
}

var _ = Describe("Refinery Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-refinery"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		refinery := &gastownv1alpha1.Refinery{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Refinery")
			err := k8sClient.Get(ctx, typeNamespacedName, refinery)
			if err != nil && errors.IsNotFound(err) {
				resource := &gastownv1alpha1.Refinery{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: gastownv1alpha1.RefinerySpec{
						RigRef:       "test-rig",
						TargetBranch: "main",
						TestCommand:  "make test",
						Parallelism:  1,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &gastownv1alpha1.Refinery{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Refinery")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &RefineryReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(10),
			}

			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))
		})

		It("should mark as Idle when no merge-ready polecats", func() {
			By("Reconciling the created resource")
			controllerReconciler := &RefineryReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(10),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the updated status")
			updatedRefinery := &gastownv1alpha1.Refinery{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedRefinery)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedRefinery.Status.Phase).To(Equal("Idle"))
			Expect(updatedRefinery.Status.QueueLength).To(Equal(int32(0)))
		})
	})

	Context("When finding merge-ready polecats", func() {
		It("should find polecats with Available condition", func() {
			r := &RefineryReconciler{}

			polecats := &gastownv1alpha1.PolecatList{
				Items: []gastownv1alpha1.Polecat{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "polecat-1"},
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:   "Available",
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{Name: "polecat-2"},
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:   "Progressing",
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{Name: "polecat-3"},
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:   "Available",
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			}

			ready := r.findMergeReadyPolecats(polecats)
			Expect(ready).To(HaveLen(2))
			Expect(ready[0].Name).To(Equal("polecat-1"))
			Expect(ready[1].Name).To(Equal("polecat-3"))
		})

		It("should return empty list when no polecats are ready", func() {
			r := &RefineryReconciler{}

			polecats := &gastownv1alpha1.PolecatList{
				Items: []gastownv1alpha1.Polecat{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "polecat-1"},
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:   "Progressing",
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			}

			ready := r.findMergeReadyPolecats(polecats)
			Expect(ready).To(BeEmpty())
		})

		It("should fallback to old Ready condition with PodSucceeded reason", func() {
			r := &RefineryReconciler{}

			polecats := &gastownv1alpha1.PolecatList{
				Items: []gastownv1alpha1.Polecat{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "old-style-polecat"},
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:   "Ready",
									Status: metav1.ConditionTrue,
									Reason: "PodSucceeded",
								},
							},
						},
					},
				},
			}

			ready := r.findMergeReadyPolecats(polecats)
			Expect(ready).To(HaveLen(1))
			Expect(ready[0].Name).To(Equal("old-style-polecat"))
		})

		It("should not use Ready condition if Available is present", func() {
			r := &RefineryReconciler{}

			polecats := &gastownv1alpha1.PolecatList{
				Items: []gastownv1alpha1.Polecat{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "transition-polecat"},
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:   "Available",
									Status: metav1.ConditionFalse, // Not ready
								},
								{
									Type:   "Ready",
									Status: metav1.ConditionTrue,
									Reason: "PodSucceeded",
								},
							},
						},
					},
				},
			}

			// Should NOT be in ready list because Available is present (even though false)
			// and we prefer new conditions over old ones
			ready := r.findMergeReadyPolecats(polecats)
			Expect(ready).To(BeEmpty())
		})

		It("should prefer Available over Ready when both are true", func() {
			r := &RefineryReconciler{}

			polecats := &gastownv1alpha1.PolecatList{
				Items: []gastownv1alpha1.Polecat{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "both-conditions-polecat"},
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:   "Available",
									Status: metav1.ConditionTrue,
								},
								{
									Type:   "Ready",
									Status: metav1.ConditionTrue,
									Reason: "PodSucceeded",
								},
							},
						},
					},
				},
			}

			// Should be in ready list (counted once, not twice)
			ready := r.findMergeReadyPolecats(polecats)
			Expect(ready).To(HaveLen(1))
		})

		It("should not use Ready condition with other reasons", func() {
			r := &RefineryReconciler{}

			polecats := &gastownv1alpha1.PolecatList{
				Items: []gastownv1alpha1.Polecat{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "non-succeeded-polecat"},
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:   "Ready",
									Status: metav1.ConditionTrue,
									Reason: "PodRunning", // Not PodSucceeded
								},
							},
						},
					},
				},
			}

			// Should NOT be in ready list because Ready reason is not PodSucceeded
			ready := r.findMergeReadyPolecats(polecats)
			Expect(ready).To(BeEmpty())
		})
	})

	Context("When processing merges", func() {
		It("should process merge-ready polecats and update status", func() {
			ctx := context.Background()

			// Create the Rig that the refinery references
			rig := &gastownv1alpha1.Rig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "merge-test-rig",
				},
				Spec: gastownv1alpha1.RigSpec{
					GitURL:      "git@github.com:test/repo.git",
					BeadsPrefix: "test",
				},
			}
			Expect(k8sClient.Create(ctx, rig)).To(Succeed())

			// Create a refinery
			refinery := &gastownv1alpha1.Refinery{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "merge-test-refinery",
					Namespace: "default",
				},
				Spec: gastownv1alpha1.RefinerySpec{
					RigRef:       "merge-test-rig",
					TargetBranch: "main",
					TestCommand:  "make test",
					Parallelism:  1,
				},
			}
			Expect(k8sClient.Create(ctx, refinery)).To(Succeed())

			// Create a merge-ready polecat (with Available condition and matching rig label)
			polecat := &gastownv1alpha1.Polecat{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "merge-ready-polecat",
					Namespace: "default",
					Labels: map[string]string{
						"gastown.io/rig": "merge-test-rig",
					},
				},
				Spec: gastownv1alpha1.PolecatSpec{
					Rig:          "merge-test-rig",
					DesiredState: gastownv1alpha1.PolecatDesiredWorking,
					BeadID:       "merge-bead-123",
				},
			}
			Expect(k8sClient.Create(ctx, polecat)).To(Succeed())

			// Set Available condition and Branch on polecat
			polecat.Status.Phase = gastownv1alpha1.PolecatPhaseDone
			polecat.Status.Branch = "feature/merge-bead-123"
			polecat.Status.Conditions = []metav1.Condition{
				{
					Type:               "Available",
					Status:             metav1.ConditionTrue,
					Reason:             "Ready",
					Message:            "Polecat completed work",
					LastTransitionTime: metav1.Now(),
				},
			}
			Expect(k8sClient.Status().Update(ctx, polecat)).To(Succeed())

			// Create mock git client factory
			mockClient := &mockGitClient{}
			mockFactory := func(repoDir, gitURL, sshKeyPath string) git.GitClient {
				return mockClient
			}

			// Create reconciler with fake recorder and mock git factory
			controllerReconciler := &RefineryReconciler{
				Client:           k8sClient,
				Scheme:           k8sClient.Scheme(),
				Recorder:         record.NewFakeRecorder(10),
				GitClientFactory: mockFactory,
			}

			// Reconcile
			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      refinery.Name,
					Namespace: refinery.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			// Verify refinery status was updated
			var updatedRefinery gastownv1alpha1.Refinery
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      refinery.Name,
				Namespace: refinery.Namespace,
			}, &updatedRefinery)).To(Succeed())

			Expect(updatedRefinery.Status.MergesSummary.Succeeded).To(Equal(int32(1)))
			Expect(updatedRefinery.Status.CurrentMerge).To(Equal("merge-ready-polecat"))
			Expect(updatedRefinery.Status.LastMergeTime).NotTo(BeNil())

			// Verify polecat got Merged condition
			var updatedPolecat gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      polecat.Name,
				Namespace: polecat.Namespace,
			}, &updatedPolecat)).To(Succeed())

			var hasMergedCondition bool
			for _, cond := range updatedPolecat.Status.Conditions {
				if cond.Type == "Merged" && cond.Status == metav1.ConditionTrue {
					hasMergedCondition = true
					break
				}
			}
			Expect(hasMergedCondition).To(BeTrue())

			// Cleanup
			Expect(k8sClient.Delete(ctx, rig)).To(Succeed())
			Expect(k8sClient.Delete(ctx, refinery)).To(Succeed())
			Expect(k8sClient.Delete(ctx, polecat)).To(Succeed())
		})

		It("should handle non-existent refinery gracefully", func() {
			ctx := context.Background()

			controllerReconciler := &RefineryReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(10),
			}

			// Reconcile non-existent refinery
			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "non-existent-refinery",
					Namespace: "default",
				},
			})

			// Should not error, just return empty result
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})

		It("should requeue quickly when queue has pending items", func() {
			ctx := context.Background()

			// Create a refinery
			refinery := &gastownv1alpha1.Refinery{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "queue-test-refinery",
					Namespace: "default",
				},
				Spec: gastownv1alpha1.RefinerySpec{
					RigRef:       "queue-rig",
					TargetBranch: "main",
					Parallelism:  1,
				},
			}
			Expect(k8sClient.Create(ctx, refinery)).To(Succeed())

			// Create multiple merge-ready polecats
			for i := 1; i <= 3; i++ {
				polecat := &gastownv1alpha1.Polecat{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "queue-polecat-" + string(rune('0'+i)),
						Namespace: "default",
						Labels: map[string]string{
							"gastown.io/rig": "queue-rig",
						},
					},
					Spec: gastownv1alpha1.PolecatSpec{
						Rig:          "queue-rig",
						DesiredState: gastownv1alpha1.PolecatDesiredWorking,
					},
				}
				Expect(k8sClient.Create(ctx, polecat)).To(Succeed())

				polecat.Status.Conditions = []metav1.Condition{
					{
						Type:               "Available",
						Status:             metav1.ConditionTrue,
						Reason:             "Ready",
						LastTransitionTime: metav1.Now(),
					},
				}
				Expect(k8sClient.Status().Update(ctx, polecat)).To(Succeed())
			}

			controllerReconciler := &RefineryReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(10),
			}

			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      refinery.Name,
					Namespace: refinery.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Should requeue at processing interval since there's more work
			Expect(result.RequeueAfter).To(Equal(refineryProcessingRequeueInterval))

			// Cleanup
			Expect(k8sClient.Delete(ctx, refinery)).To(Succeed())
			for i := 1; i <= 3; i++ {
				polecat := &gastownv1alpha1.Polecat{}
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      "queue-polecat-" + string(rune('0'+i)),
					Namespace: "default",
				}, polecat)
				_ = k8sClient.Delete(ctx, polecat)
			}
		})
	})
})
