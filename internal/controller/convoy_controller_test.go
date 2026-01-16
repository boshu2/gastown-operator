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

const (
	testConvoyID = "existing-convoy"
)

var _ = Describe("Convoy Controller", func() {
	var (
		ctx        context.Context
		reconciler *ConvoyReconciler
		mockClient *gt.MockClient
		testConvoy *gastownv1alpha1.Convoy
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockClient = &gt.MockClient{}
		reconciler = &ConvoyReconciler{
			Client:   k8sClient,
			Scheme:   k8sClient.Scheme(),
			GTClient: mockClient,
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
		It("should create convoy in beads system", func() {
			var createCalled bool
			var createDesc string
			var createBeads []string

			mockClient.ConvoyCreateFunc = func(ctx context.Context, description string, beadIDs []string) (string, error) {
				createCalled = true
				createDesc = description
				createBeads = beadIDs
				return "convoy-abc123", nil
			}
			mockClient.ConvoyStatusFunc = func(ctx context.Context, id string) (*gt.ConvoyStatus, error) {
				return &gt.ConvoyStatus{
					ID:        id,
					Phase:     "InProgress",
					Progress:  "0/3 complete",
					Pending:   []string{"test-bead-1", "test-bead-2", "test-bead-3"},
					Completed: []string{},
				}, nil
			}

			Expect(k8sClient.Create(ctx, testConvoy)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testConvoy.Name,
				Namespace: testConvoy.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(ConvoySyncInterval))
			Expect(createCalled).To(BeTrue())
			Expect(createDesc).To(Equal("Test wave 1"))
			Expect(createBeads).To(Equal([]string{"test-bead-1", "test-bead-2", "test-bead-3"}))

			// Verify status was updated
			var updated gastownv1alpha1.Convoy
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.BeadsConvoyID).To(Equal("convoy-abc123"))
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.ConvoyPhaseInProgress))
			Expect(updated.Status.StartedAt).NotTo(BeNil())
		})

		It("should handle convoy create failure", func() {
			mockClient.ConvoyCreateFunc = func(ctx context.Context, description string, beadIDs []string) (string, error) {
				return "", errors.New("beads database locked")
			}

			Expect(k8sClient.Create(ctx, testConvoy)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testConvoy.Name,
				Namespace: testConvoy.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			// Verify condition was set
			var updated gastownv1alpha1.Convoy
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())

			var foundCondition bool
			for _, cond := range updated.Status.Conditions {
				if cond.Type == ConditionConvoyReady && cond.Status == metav1.ConditionFalse {
					foundCondition = true
					Expect(cond.Reason).To(Equal("CreateFailed"))
					break
				}
			}
			Expect(foundCondition).To(BeTrue())
		})
	})

	Context("When syncing convoy progress", func() {
		It("should update progress from gt CLI", func() {
			// Pre-set the convoy with an existing beads ID
			testConvoy.Status.BeadsConvoyID = testConvoyID
			testConvoy.Status.Phase = gastownv1alpha1.ConvoyPhaseInProgress

			mockClient.ConvoyStatusFunc = func(ctx context.Context, id string) (*gt.ConvoyStatus, error) {
				return &gt.ConvoyStatus{
					ID:        id,
					Phase:     "InProgress",
					Progress:  "2/3 complete",
					Pending:   []string{"test-bead-3"},
					Completed: []string{"test-bead-1", "test-bead-2"},
				}, nil
			}

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
			Expect(updated.Status.Progress).To(Equal("2/3 complete"))
			Expect(updated.Status.CompletedBeads).To(Equal([]string{"test-bead-1", "test-bead-2"}))
			Expect(updated.Status.PendingBeads).To(Equal([]string{"test-bead-3"}))
		})
	})

	Context("When convoy completes", func() {
		It("should mark as complete and not requeue", func() {
			// Pre-set the convoy with an existing beads ID
			testConvoy.Status.BeadsConvoyID = testConvoyID
			testConvoy.Status.Phase = gastownv1alpha1.ConvoyPhaseInProgress

			mockClient.ConvoyStatusFunc = func(ctx context.Context, id string) (*gt.ConvoyStatus, error) {
				return &gt.ConvoyStatus{
					ID:        id,
					Phase:     "Complete",
					Progress:  "3/3 complete",
					Pending:   []string{},
					Completed: []string{"test-bead-1", "test-bead-2", "test-bead-3"},
				}, nil
			}

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

		It("should send notification when configured", func() {
			var mailSent bool
			var mailAddress, mailSubject string

			testConvoy.Spec.NotifyOnComplete = "mayor@gastown.io"
			testConvoy.Status.BeadsConvoyID = testConvoyID
			testConvoy.Status.Phase = gastownv1alpha1.ConvoyPhaseInProgress

			mockClient.ConvoyStatusFunc = func(ctx context.Context, id string) (*gt.ConvoyStatus, error) {
				return &gt.ConvoyStatus{
					ID:        id,
					Phase:     "Complete",
					Progress:  "3/3 complete",
					Pending:   []string{},
					Completed: []string{"test-bead-1", "test-bead-2", "test-bead-3"},
				}, nil
			}
			mockClient.MailSendFunc = func(ctx context.Context, address, subject, message string) error {
				mailSent = true
				mailAddress = address
				mailSubject = subject
				return nil
			}

			Expect(k8sClient.Create(ctx, testConvoy)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testConvoy.Name,
				Namespace: testConvoy.Namespace,
			}}
			_, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(mailSent).To(BeTrue())
			Expect(mailAddress).To(Equal("mayor@gastown.io"))
			Expect(mailSubject).To(ContainSubstring("Test wave 1"))
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
			created.Status.BeadsConvoyID = "completed-convoy"
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
