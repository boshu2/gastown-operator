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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
)

// mockGTClient is a mock implementation of the GT client for testing.
type mockGTClient struct {
	mailSendFunc func(ctx context.Context, address, subject, message string) error
}

func (m *mockGTClient) MailSend(ctx context.Context, address, subject, message string) error {
	if m.mailSendFunc != nil {
		return m.mailSendFunc(ctx, address, subject, message)
	}
	return nil
}

var _ = Describe("Witness Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-witness"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		witness := &gastownv1alpha1.Witness{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Witness")
			err := k8sClient.Get(ctx, typeNamespacedName, witness)
			if err != nil && errors.IsNotFound(err) {
				resource := &gastownv1alpha1.Witness{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: gastownv1alpha1.WitnessSpec{
						RigRef:           "test-rig",
						EscalationTarget: "mayor",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &gastownv1alpha1.Witness{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Witness")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &WitnessReconciler{
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

		It("should update status with polecat summary", func() {
			By("Reconciling the created resource")
			controllerReconciler := &WitnessReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(10),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the updated status")
			updatedWitness := &gastownv1alpha1.Witness{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedWitness)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedWitness.Status.LastCheckTime).NotTo(BeNil())
		})
	})

	Context("When determining phase", func() {
		It("should return Pending when no polecats", func() {
			r := &WitnessReconciler{}
			summary := gastownv1alpha1.PolecatsSummary{
				Total:     0,
				Running:   0,
				Succeeded: 0,
				Failed:    0,
				Stuck:     0,
			}
			Expect(r.determinePhase(summary)).To(Equal("Pending"))
		})

		It("should return Active when polecats are running", func() {
			r := &WitnessReconciler{}
			summary := gastownv1alpha1.PolecatsSummary{
				Total:     2,
				Running:   2,
				Succeeded: 0,
				Failed:    0,
				Stuck:     0,
			}
			Expect(r.determinePhase(summary)).To(Equal("Active"))
		})

		It("should return Degraded when polecats are stuck", func() {
			r := &WitnessReconciler{}
			summary := gastownv1alpha1.PolecatsSummary{
				Total:     3,
				Running:   2,
				Succeeded: 0,
				Failed:    0,
				Stuck:     1,
			}
			Expect(r.determinePhase(summary)).To(Equal("Degraded"))
		})

		It("should return Degraded when polecats have failed", func() {
			r := &WitnessReconciler{}
			summary := gastownv1alpha1.PolecatsSummary{
				Total:     3,
				Running:   1,
				Succeeded: 1,
				Failed:    1,
				Stuck:     0,
			}
			Expect(r.determinePhase(summary)).To(Equal("Degraded"))
		})
	})

	Context("When calculating summary", func() {
		It("should detect stuck polecats", func() {
			r := &WitnessReconciler{}
			stuckThreshold := 15 * time.Minute
			oldTime := metav1.NewTime(time.Now().Add(-20 * time.Minute))

			polecats := &gastownv1alpha1.PolecatList{
				Items: []gastownv1alpha1.Polecat{
					{
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:               "Progressing",
									Status:             metav1.ConditionTrue,
									LastTransitionTime: oldTime,
								},
							},
						},
					},
				},
			}

			summary := r.calculateSummary(polecats, stuckThreshold)
			Expect(summary.Total).To(Equal(int32(1)))
			Expect(summary.Running).To(Equal(int32(1)))
			Expect(summary.Stuck).To(Equal(int32(1)))
		})

		It("should count succeeded polecats", func() {
			r := &WitnessReconciler{}
			stuckThreshold := 15 * time.Minute

			polecats := &gastownv1alpha1.PolecatList{
				Items: []gastownv1alpha1.Polecat{
					{
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

			summary := r.calculateSummary(polecats, stuckThreshold)
			Expect(summary.Total).To(Equal(int32(1)))
			Expect(summary.Succeeded).To(Equal(int32(1)))
		})

		It("should count failed polecats", func() {
			r := &WitnessReconciler{}
			stuckThreshold := 15 * time.Minute

			polecats := &gastownv1alpha1.PolecatList{
				Items: []gastownv1alpha1.Polecat{
					{
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:   "Degraded",
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			}

			summary := r.calculateSummary(polecats, stuckThreshold)
			Expect(summary.Total).To(Equal(int32(1)))
			Expect(summary.Failed).To(Equal(int32(1)))
		})

		It("should fallback to old Ready condition for succeeded polecats", func() {
			r := &WitnessReconciler{}
			stuckThreshold := 15 * time.Minute

			// Polecat using old condition style (Ready with PodSucceeded reason)
			polecats := &gastownv1alpha1.PolecatList{
				Items: []gastownv1alpha1.Polecat{
					{
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

			summary := r.calculateSummary(polecats, stuckThreshold)
			Expect(summary.Total).To(Equal(int32(1)))
			Expect(summary.Succeeded).To(Equal(int32(1)))
		})

		It("should fallback to old Working condition for running polecats", func() {
			r := &WitnessReconciler{}
			stuckThreshold := 15 * time.Minute
			recentTime := metav1.NewTime(time.Now())

			// Polecat using old condition style (Working)
			polecats := &gastownv1alpha1.PolecatList{
				Items: []gastownv1alpha1.Polecat{
					{
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:               "Working",
									Status:             metav1.ConditionTrue,
									LastTransitionTime: recentTime,
								},
							},
						},
					},
				},
			}

			summary := r.calculateSummary(polecats, stuckThreshold)
			Expect(summary.Total).To(Equal(int32(1)))
			Expect(summary.Running).To(Equal(int32(1)))
		})

		It("should prefer new conditions over old when both present", func() {
			r := &WitnessReconciler{}
			stuckThreshold := 15 * time.Minute

			// Polecat with BOTH new and old conditions (transition state)
			polecats := &gastownv1alpha1.PolecatList{
				Items: []gastownv1alpha1.Polecat{
					{
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

			summary := r.calculateSummary(polecats, stuckThreshold)
			Expect(summary.Total).To(Equal(int32(1)))
			// Should only count once, not double-count
			Expect(summary.Succeeded).To(Equal(int32(1)))
		})

		It("should detect stuck polecats using old Working condition", func() {
			r := &WitnessReconciler{}
			stuckThreshold := 15 * time.Minute
			oldTime := metav1.NewTime(time.Now().Add(-20 * time.Minute))

			// Polecat using old condition style (Working) that's stuck
			polecats := &gastownv1alpha1.PolecatList{
				Items: []gastownv1alpha1.Polecat{
					{
						Status: gastownv1alpha1.PolecatStatus{
							Conditions: []metav1.Condition{
								{
									Type:               "Working",
									Status:             metav1.ConditionTrue,
									LastTransitionTime: oldTime,
								},
							},
						},
					},
				},
			}

			summary := r.calculateSummary(polecats, stuckThreshold)
			Expect(summary.Total).To(Equal(int32(1)))
			Expect(summary.Running).To(Equal(int32(1)))
			Expect(summary.Stuck).To(Equal(int32(1)))
		})
	})

	Context("When escalating issues", func() {
		It("should send mail to mayor when escalation target is mayor", func() {
			r := &WitnessReconciler{
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(10),
			}

			mockMailSent := false
			r.GTClient = &mockGTClient{
				mailSendFunc: func(ctx context.Context, address, subject, message string) error {
					mockMailSent = true
					Expect(address).To(Equal("mayor"))
					Expect(subject).To(ContainSubstring("Health Alert"))
					Expect(message).To(ContainSubstring("Stuck"))
					return nil
				},
			}

			ctx := context.Background()
			witness := &gastownv1alpha1.Witness{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-witness",
					Namespace: "default",
				},
				Spec: gastownv1alpha1.WitnessSpec{
					RigRef:           "test-rig",
					EscalationTarget: "mayor",
				},
				Status: gastownv1alpha1.WitnessStatus{
					Phase: "Degraded",
				},
			}

			summary := gastownv1alpha1.PolecatsSummary{
				Total:     5,
				Running:   3,
				Succeeded: 1,
				Failed:    0,
				Stuck:     1,
			}

			r.escalateIssues(ctx, witness, summary)
			Expect(mockMailSent).To(BeTrue())
		})
	})
})
