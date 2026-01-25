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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
)

var _ = Describe("Polecat Controller", func() {
	var (
		ctx         context.Context
		reconciler  *PolecatReconciler
		testPolecat *gastownv1alpha1.Polecat
	)

	BeforeEach(func() {
		ctx = context.Background()
		reconciler = &PolecatReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		testPolecat = &gastownv1alpha1.Polecat{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "test-polecat",
				Namespace:  "default",
				Finalizers: []string{"gastown.io/polecat-cleanup"},
			},
			Spec: gastownv1alpha1.PolecatSpec{
				Rig:           "test-rig",
				DesiredState:  gastownv1alpha1.PolecatDesiredWorking,
				BeadID:        "test-bead-123",
				ExecutionMode: gastownv1alpha1.ExecutionModeKubernetes,
				Kubernetes: &gastownv1alpha1.KubernetesSpec{
					GitRepository: "git@github.com:example/repo.git",
					GitBranch:     "main",
					GitSecretRef: gastownv1alpha1.SecretReference{
						Name: "git-creds",
					},
					ClaudeCredsSecretRef: &gastownv1alpha1.SecretReference{
						Name: "claude-creds",
					},
				},
			},
		}
	})

	AfterEach(func() {
		// Clean up test polecat - remove finalizer first to allow deletion
		if testPolecat != nil {
			var current gastownv1alpha1.Polecat
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: testPolecat.Name, Namespace: testPolecat.Namespace}, &current); err == nil {
				current.Finalizers = nil
				_ = k8sClient.Update(ctx, &current)
			}
			_ = k8sClient.Delete(ctx, testPolecat)
		}
	})

	Context("When polecat does not exist", func() {
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

	Context("When creating a polecat with kubernetes execution mode", func() {
		It("should fail gracefully when kubernetes spec is missing", func() {
			// Create polecat with kubernetes mode but no kubernetes spec
			badPolecat := &gastownv1alpha1.Polecat{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bad-k8s-polecat",
					Namespace:  "default",
					Finalizers: []string{"gastown.io/polecat-cleanup"},
				},
				Spec: gastownv1alpha1.PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  gastownv1alpha1.PolecatDesiredWorking,
					BeadID:        "test-bead",
					ExecutionMode: gastownv1alpha1.ExecutionModeKubernetes,
					// No Kubernetes spec!
				},
			}

			Expect(k8sClient.Create(ctx, badPolecat)).To(Succeed())
			defer func() {
				var p gastownv1alpha1.Polecat
				if k8sClient.Get(ctx, types.NamespacedName{Name: badPolecat.Name, Namespace: badPolecat.Namespace}, &p) == nil {
					p.Finalizers = nil
					_ = k8sClient.Update(ctx, &p)
					_ = k8sClient.Delete(ctx, &p)
				}
			}()

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      badPolecat.Name,
				Namespace: badPolecat.Namespace,
			}}
			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			// Should requeue after error interval
			Expect(result.RequeueAfter).NotTo(BeZero())

			// Verify status shows stuck with proper condition
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseStuck))

			var foundCondition bool
			for _, cond := range updated.Status.Conditions {
				if cond.Type == ConditionPolecatReady && cond.Reason == "MissingKubernetesSpec" {
					foundCondition = true
					break
				}
			}
			Expect(foundCondition).To(BeTrue())
		})

		It("should create a Pod when polecat has kubernetes execution mode", func() {
			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}

			// Reconcile should create the Pod
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify Pod was created with correct spec
			var pod corev1.Pod
			podName := "polecat-" + testPolecat.Name
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      podName,
					Namespace: testPolecat.Namespace,
				}, &pod)
			}).Should(Succeed())

			// Verify Pod labels
			Expect(pod.Labels["gastown.io/polecat"]).To(Equal(testPolecat.Name))
			Expect(pod.Labels["gastown.io/rig"]).To(Equal(testPolecat.Spec.Rig))
			Expect(pod.Labels["gastown.io/bead"]).To(Equal(testPolecat.Spec.BeadID))

			// Verify init container exists
			Expect(pod.Spec.InitContainers).To(HaveLen(1))
			Expect(pod.Spec.InitContainers[0].Name).To(Equal("git-init"))

			// Verify main container and telemetry sidecar exist
			Expect(pod.Spec.Containers).To(HaveLen(2))
			Expect(pod.Spec.Containers[0].Name).To(Equal("claude"))
			Expect(pod.Spec.Containers[1].Name).To(Equal("telemetry"))

			// Verify volumes
			volumeNames := make([]string, len(pod.Spec.Volumes))
			for i, v := range pod.Spec.Volumes {
				volumeNames[i] = v.Name
			}
			Expect(volumeNames).To(ContainElements("workspace", "git-creds", "claude-creds"))

			// Cleanup
			Expect(k8sClient.Delete(ctx, &pod)).To(Succeed())
		})

		It("should sync polecat status from Pod status", func() {
			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}

			// First reconcile creates the Pod
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Get the created Pod and update its status to Running
			var pod corev1.Pod
			podName := "polecat-" + testPolecat.Name
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      podName,
					Namespace: testPolecat.Namespace,
				}, &pod)
			}).Should(Succeed())

			pod.Status.Phase = corev1.PodRunning
			Expect(k8sClient.Status().Update(ctx, &pod)).To(Succeed())

			// Reconcile again to sync status
			_, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify Polecat status reflects Pod status
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseWorking))
			Expect(updated.Status.PodName).To(Equal(podName))

			// Cleanup
			Expect(k8sClient.Delete(ctx, &pod)).To(Succeed())
		})

		It("should mark polecat as Done when Pod succeeds", func() {
			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}

			// First reconcile creates the Pod
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Get the created Pod and update its status to Succeeded
			var pod corev1.Pod
			podName := "polecat-" + testPolecat.Name
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      podName,
					Namespace: testPolecat.Namespace,
				}, &pod)
			}).Should(Succeed())

			pod.Status.Phase = corev1.PodSucceeded
			Expect(k8sClient.Status().Update(ctx, &pod)).To(Succeed())

			// Reconcile again to sync status
			_, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify Polecat status shows Done
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseDone))

			// Cleanup
			Expect(k8sClient.Delete(ctx, &pod)).To(Succeed())
		})

		It("should set Available condition when Pod succeeds", func() {
			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}

			// First reconcile creates the Pod
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Get the created Pod and update its status to Succeeded
			var pod corev1.Pod
			podName := "polecat-" + testPolecat.Name
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      podName,
					Namespace: testPolecat.Namespace,
				}, &pod)
			}).Should(Succeed())

			pod.Status.Phase = corev1.PodSucceeded
			Expect(k8sClient.Status().Update(ctx, &pod)).To(Succeed())

			// Reconcile again to sync status
			_, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify standard conditions are set correctly
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())

			// Available=True signals merge readiness (this is what Witness/Refinery look for)
			var availableCondition, progressingCondition, degradedCondition *metav1.Condition
			for i := range updated.Status.Conditions {
				switch updated.Status.Conditions[i].Type {
				case ConditionAvailable:
					availableCondition = &updated.Status.Conditions[i]
				case ConditionProgressing:
					progressingCondition = &updated.Status.Conditions[i]
				case ConditionDegraded:
					degradedCondition = &updated.Status.Conditions[i]
				}
			}

			Expect(availableCondition).NotTo(BeNil(), "Available condition should exist")
			Expect(availableCondition.Status).To(Equal(metav1.ConditionTrue), "Available should be True")
			Expect(availableCondition.Reason).To(Equal("WorkComplete"))

			Expect(progressingCondition).NotTo(BeNil(), "Progressing condition should exist")
			Expect(progressingCondition.Status).To(Equal(metav1.ConditionFalse), "Progressing should be False")

			Expect(degradedCondition).NotTo(BeNil(), "Degraded condition should exist")
			Expect(degradedCondition.Status).To(Equal(metav1.ConditionFalse), "Degraded should be False")

			// Cleanup
			Expect(k8sClient.Delete(ctx, &pod)).To(Succeed())
		})

		It("should set Degraded condition when Pod fails", func() {
			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}

			// First reconcile creates the Pod
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Get the created Pod and update its status to Failed
			var pod corev1.Pod
			podName := "polecat-" + testPolecat.Name
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      podName,
					Namespace: testPolecat.Namespace,
				}, &pod)
			}).Should(Succeed())

			pod.Status.Phase = corev1.PodFailed
			Expect(k8sClient.Status().Update(ctx, &pod)).To(Succeed())

			// Reconcile again to sync status
			_, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify standard conditions are set correctly
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())

			var availableCondition, degradedCondition *metav1.Condition
			for i := range updated.Status.Conditions {
				switch updated.Status.Conditions[i].Type {
				case ConditionAvailable:
					availableCondition = &updated.Status.Conditions[i]
				case ConditionDegraded:
					degradedCondition = &updated.Status.Conditions[i]
				}
			}

			Expect(availableCondition).NotTo(BeNil(), "Available condition should exist")
			Expect(availableCondition.Status).To(Equal(metav1.ConditionFalse), "Available should be False")

			Expect(degradedCondition).NotTo(BeNil(), "Degraded condition should exist")
			Expect(degradedCondition.Status).To(Equal(metav1.ConditionTrue), "Degraded should be True")
			Expect(degradedCondition.Reason).To(Equal("PodFailed"))

			// Cleanup
			Expect(k8sClient.Delete(ctx, &pod)).To(Succeed())
		})

		It("should set Progressing condition when Pod is running", func() {
			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}

			// First reconcile creates the Pod
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Get the created Pod and update its status to Running
			var pod corev1.Pod
			podName := "polecat-" + testPolecat.Name
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      podName,
					Namespace: testPolecat.Namespace,
				}, &pod)
			}).Should(Succeed())

			pod.Status.Phase = corev1.PodRunning
			Expect(k8sClient.Status().Update(ctx, &pod)).To(Succeed())

			// Reconcile again to sync status
			_, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify standard conditions are set correctly
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())

			var availableCondition, progressingCondition, degradedCondition *metav1.Condition
			for i := range updated.Status.Conditions {
				switch updated.Status.Conditions[i].Type {
				case ConditionAvailable:
					availableCondition = &updated.Status.Conditions[i]
				case ConditionProgressing:
					progressingCondition = &updated.Status.Conditions[i]
				case ConditionDegraded:
					degradedCondition = &updated.Status.Conditions[i]
				}
			}

			Expect(progressingCondition).NotTo(BeNil(), "Progressing condition should exist")
			Expect(progressingCondition.Status).To(Equal(metav1.ConditionTrue), "Progressing should be True")

			Expect(availableCondition).NotTo(BeNil(), "Available condition should exist")
			Expect(availableCondition.Status).To(Equal(metav1.ConditionFalse), "Available should be False")

			Expect(degradedCondition).NotTo(BeNil(), "Degraded condition should exist")
			Expect(degradedCondition.Status).To(Equal(metav1.ConditionFalse), "Degraded should be False")

			// Cleanup
			Expect(k8sClient.Delete(ctx, &pod)).To(Succeed())
		})

		It("should mark polecat as Stuck when Pod fails", func() {
			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}

			// First reconcile creates the Pod
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Get the created Pod and update its status to Failed
			var pod corev1.Pod
			podName := "polecat-" + testPolecat.Name
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      podName,
					Namespace: testPolecat.Namespace,
				}, &pod)
			}).Should(Succeed())

			pod.Status.Phase = corev1.PodFailed
			Expect(k8sClient.Status().Update(ctx, &pod)).To(Succeed())

			// Reconcile again to sync status
			_, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify Polecat status shows Stuck
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseStuck))

			// Cleanup
			Expect(k8sClient.Delete(ctx, &pod)).To(Succeed())
		})

		It("should delete Pod when polecat is terminated", func() {
			testPolecat.Spec.DesiredState = gastownv1alpha1.PolecatDesiredWorking
			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}

			// Create the Pod
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify Pod exists
			var pod corev1.Pod
			podName := "polecat-" + testPolecat.Name
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      podName,
					Namespace: testPolecat.Namespace,
				}, &pod)
			}).Should(Succeed())

			// Now set to terminated
			Expect(k8sClient.Get(ctx, req.NamespacedName, testPolecat)).To(Succeed())
			testPolecat.Spec.DesiredState = gastownv1alpha1.PolecatDesiredTerminated
			Expect(k8sClient.Update(ctx, testPolecat)).To(Succeed())

			// Reconcile to delete Pod
			_, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify Pod was deleted
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      podName,
					Namespace: testPolecat.Namespace,
				}, &pod)
				return apierrors.IsNotFound(err)
			}).Should(BeTrue())

			// Verify Polecat status shows Terminated
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseTerminated))
		})
	})

	Context("When handling finalizers", func() {
		It("should add finalizer on first reconcile", func() {
			// Create polecat without finalizer
			polecat := &gastownv1alpha1.Polecat{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "finalizer-test",
					Namespace: "default",
				},
				Spec: gastownv1alpha1.PolecatSpec{
					Rig:           "test-rig",
					DesiredState:  gastownv1alpha1.PolecatDesiredIdle,
					ExecutionMode: gastownv1alpha1.ExecutionModeKubernetes,
					Kubernetes: &gastownv1alpha1.KubernetesSpec{
						GitRepository: "git@github.com:example/repo.git",
						GitSecretRef: gastownv1alpha1.SecretReference{
							Name: "git-creds",
						},
						ClaudeCredsSecretRef: &gastownv1alpha1.SecretReference{
							Name: "claude-creds",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, polecat)).To(Succeed())
			defer func() {
				var p gastownv1alpha1.Polecat
				if k8sClient.Get(ctx, types.NamespacedName{Name: polecat.Name, Namespace: polecat.Namespace}, &p) == nil {
					p.Finalizers = nil
					_ = k8sClient.Update(ctx, &p)
					_ = k8sClient.Delete(ctx, &p)
				}
			}()

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      polecat.Name,
				Namespace: polecat.Namespace,
			}}

			// First reconcile should add finalizer
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify finalizer was added
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Finalizers).To(ContainElement("gastown.io/polecat-cleanup"))
		})

		It("should cleanup Pod on deletion", func() {
			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}

			// First reconcile creates the Pod
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			podName := "polecat-" + testPolecat.Name

			// Verify Pod was created
			var pod corev1.Pod
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      podName,
					Namespace: testPolecat.Namespace,
				}, &pod)
			}).Should(Succeed())

			// Mark polecat for deletion
			Expect(k8sClient.Delete(ctx, testPolecat)).To(Succeed())

			// Reconcile should trigger cleanup
			_, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify Pod was deleted
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      podName,
					Namespace: testPolecat.Namespace,
				}, &pod)
				return apierrors.IsNotFound(err)
			}).Should(BeTrue())

			// Verify polecat is deleted (finalizer removed)
			Eventually(func() bool {
				var p gastownv1alpha1.Polecat
				err := k8sClient.Get(ctx, req.NamespacedName, &p)
				return apierrors.IsNotFound(err)
			}).Should(BeTrue())
		})
	})

	Context("When handling idle state", func() {
		It("should not create Pod when desired state is Idle", func() {
			testPolecat.Spec.DesiredState = gastownv1alpha1.PolecatDesiredIdle
			Expect(k8sClient.Create(ctx, testPolecat)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      testPolecat.Name,
				Namespace: testPolecat.Namespace,
			}}

			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify no Pod was created
			var podList corev1.PodList
			Expect(k8sClient.List(ctx, &podList)).To(Succeed())
			for _, pod := range podList.Items {
				Expect(pod.Labels["gastown.io/polecat"]).NotTo(Equal(testPolecat.Name))
			}

			// Verify status shows idle
			var updated gastownv1alpha1.Polecat
			Expect(k8sClient.Get(ctx, req.NamespacedName, &updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(gastownv1alpha1.PolecatPhaseIdle))
		})
	})
})
