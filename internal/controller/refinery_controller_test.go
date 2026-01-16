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
)

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
	})
})
