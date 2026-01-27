//go:build e2e
// +build e2e

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

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/org/gastown-operator/test/utils"
)

// namespace where the project is deployed in
const namespace = "gastown-operator-system"

// serviceAccountName created for the project
const serviceAccountName = "gastown-operator-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "gastown-operator-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "gastown-operator-metrics-binding"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By("labeling the namespace to enforce the restricted security policy")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/enforce=restricted")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

		By("labeling the namespace to allow metrics access (required by NetworkPolicy)")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"metrics=enabled")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with metrics=enabled")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace, "--ignore-not-found")
		_, _ = utils.Run(cmd)

		// Delete all CRs first and wait for them to be fully deleted
		// This ensures finalizers are processed before the controller is removed
		By("cleaning up any remaining Polecats")
		cmd = exec.Command("kubectl", "delete", "polecats", "--all", "-n", namespace, "--ignore-not-found", "--timeout=60s")
		_, _ = utils.Run(cmd)

		By("cleaning up any remaining Rigs")
		cmd = exec.Command("kubectl", "delete", "rigs", "--all", "--ignore-not-found", "--timeout=60s")
		_, _ = utils.Run(cmd)

		By("cleaning up any remaining Witnesses")
		cmd = exec.Command("kubectl", "delete", "witnesses", "--all", "-n", namespace, "--ignore-not-found", "--timeout=60s")
		_, _ = utils.Run(cmd)

		By("cleaning up any remaining Refineries")
		cmd = exec.Command("kubectl", "delete", "refineries", "--all", "-n", namespace, "--ignore-not-found", "--timeout=60s")
		_, _ = utils.Run(cmd)

		By("cleaning up any remaining Convoys")
		cmd = exec.Command("kubectl", "delete", "convoys", "--all", "-n", namespace, "--ignore-not-found", "--timeout=60s")
		_, _ = utils.Run(cmd)

		By("cleaning up any remaining BeadStores")
		cmd = exec.Command("kubectl", "delete", "beadstores", "--all", "-n", namespace, "--ignore-not-found", "--timeout=60s")
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy", "ignore-not-found=true")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall", "ignore-not-found=true")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace, "--ignore-not-found", "--timeout=60s")
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				// Get the name of the controller-manager pod
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				// Validate the pod's status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
				"--clusterrole=gastown-operator-metrics-reader",
				fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
			)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

			By("validating that the metrics service is available")
			cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("ensuring the controller pod is ready")
			verifyControllerPodReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pod", controllerPodName, "-n", namespace,
					"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("True"), "Controller pod not ready")
			}
			Eventually(verifyControllerPodReady, 3*time.Minute, time.Second).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("Serving metrics server"),
					"Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted, 3*time.Minute, time.Second).Should(Succeed())

			// +kubebuilder:scaffold:e2e-metrics-webhooks-readiness

			By("creating the curl-metrics pod to access the metrics endpoint")
			cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
				"--namespace", namespace,
				"--image=curlimages/curl:latest",
				"--overrides",
				fmt.Sprintf(`{
					"spec": {
						"containers": [{
							"name": "curl",
							"image": "curlimages/curl:latest",
							"command": ["/bin/sh", "-c"],
							"args": ["curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics"],
							"securityContext": {
								"readOnlyRootFilesystem": true,
								"allowPrivilegeEscalation": false,
								"capabilities": {
									"drop": ["ALL"]
								},
								"runAsNonRoot": true,
								"runAsUser": 1000,
								"seccompProfile": {
									"type": "RuntimeDefault"
								}
							}
						}],
						"serviceAccountName": "%s"
					}
				}`, token, metricsServiceName, namespace, serviceAccountName))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete.")
			verifyCurlUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
			}
			Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			verifyMetricsAvailable := func(g Gomega) {
				metricsOutput, err := getMetricsOutput()
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
				g.Expect(metricsOutput).NotTo(BeEmpty())
				g.Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
			}
			Eventually(verifyMetricsAvailable, 2*time.Minute).Should(Succeed())
		})

		// +kubebuilder:scaffold:e2e-webhooks-checks

		It("should auto-provision Witness and Refinery when Rig is created", func() {
			rigName := "test-auto-rig"

			By("creating a Rig CR")
			rigYAML := fmt.Sprintf(`
apiVersion: gastown.gastown.io/v1alpha1
kind: Rig
metadata:
  name: %s
spec:
  gitURL: git@github.com:test/auto-rig-repo.git
  beadsPrefix: tr
`, rigName)

			rigFile := filepath.Join("/tmp", "test-auto-rig.yaml")
			err := os.WriteFile(rigFile, []byte(rigYAML), 0644)
			Expect(err).NotTo(HaveOccurred())

			cmd := exec.Command("kubectl", "apply", "-f", rigFile)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create Rig")

			By("verifying the Rig has finalizer added")
			verifyRigFinalizer := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "rig", rigName,
					"-o", "jsonpath={.metadata.finalizers}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("gastown.io/rig-cleanup"))
			}
			Eventually(verifyRigFinalizer, 2*time.Minute, time.Second).Should(Succeed())

			By("verifying Witness CR is auto-created")
			witnessName := rigName + "-witness"
			verifyWitnessCreated := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "witness", witnessName,
					"-n", namespace, "-o", "jsonpath={.metadata.name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Witness should be auto-created")
				g.Expect(output).To(Equal(witnessName))
			}
			Eventually(verifyWitnessCreated, 2*time.Minute, time.Second).Should(Succeed())

			By("verifying Witness has correct rig reference")
			verifyWitnessRigRef := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "witness", witnessName,
					"-n", namespace, "-o", "jsonpath={.spec.rigRef}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal(rigName))
			}
			Eventually(verifyWitnessRigRef, 30*time.Second).Should(Succeed())

			By("verifying Refinery CR is auto-created")
			refineryName := rigName + "-refinery"
			verifyRefineryCreated := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "refinery", refineryName,
					"-n", namespace, "-o", "jsonpath={.metadata.name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Refinery should be auto-created")
				g.Expect(output).To(Equal(refineryName))
			}
			Eventually(verifyRefineryCreated, 2*time.Minute, time.Second).Should(Succeed())

			By("verifying Refinery has correct rig reference and target branch")
			verifyRefinerySpec := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "refinery", refineryName,
					"-n", namespace, "-o", "jsonpath={.spec.rigRef},{.spec.targetBranch}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal(rigName + ",main"))
			}
			Eventually(verifyRefinerySpec, 30*time.Second).Should(Succeed())

			By("verifying Rig status shows children created")
			verifyRigStatus := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "rig", rigName,
					"-o", "jsonpath={.status.witnessCreated},{.status.refineryCreated}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("true,true"))
			}
			Eventually(verifyRigStatus, 2*time.Minute, time.Second).Should(Succeed())

			By("cleaning up the test Rig")
			cmd = exec.Command("kubectl", "delete", "rig", rigName)
			_, _ = utils.Run(cmd)

			By("verifying Witness is deleted with Rig (finalizer cleanup)")
			verifyWitnessDeleted := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "witness", witnessName, "-n", namespace)
				_, err := utils.Run(cmd)
				g.Expect(err).To(HaveOccurred(), "Witness should be deleted")
			}
			Eventually(verifyWitnessDeleted, 2*time.Minute, time.Second).Should(Succeed())

			By("verifying Refinery is deleted with Rig (finalizer cleanup)")
			verifyRefineryDeleted := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "refinery", refineryName, "-n", namespace)
				_, err := utils.Run(cmd)
				g.Expect(err).To(HaveOccurred(), "Refinery should be deleted")
			}
			Eventually(verifyRefineryDeleted, 2*time.Minute, time.Second).Should(Succeed())
		})

		It("should detect merge-ready Polecats via Refinery", func() {
			rigName := "test-merge-rig"
			polecatName := "test-merge-polecat"

			By("creating a Rig CR")
			rigYAML := fmt.Sprintf(`
apiVersion: gastown.gastown.io/v1alpha1
kind: Rig
metadata:
  name: %s
spec:
  gitURL: git@github.com:test/merge-rig-repo.git
  beadsPrefix: tm
`, rigName)

			rigFile := filepath.Join("/tmp", "test-merge-rig.yaml")
			err := os.WriteFile(rigFile, []byte(rigYAML), 0644)
			Expect(err).NotTo(HaveOccurred())

			cmd := exec.Command("kubectl", "apply", "-f", rigFile)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create Rig")

			By("waiting for auto-provisioned Refinery")
			refineryName := rigName + "-refinery"
			verifyRefineryCreated := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "refinery", refineryName,
					"-n", namespace, "-o", "jsonpath={.metadata.name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal(refineryName))
			}
			Eventually(verifyRefineryCreated, 2*time.Minute, time.Second).Should(Succeed())

			By("creating a Polecat with rig label")
			polecatYAML := fmt.Sprintf(`
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: %s
  namespace: %s
  labels:
    gastown.io/rig: %s
spec:
  rig: %s
  desiredState: Working
  beadID: test-merge-1234
status:
  phase: Completed
  branch: polecat/%s
`, polecatName, namespace, rigName, rigName, polecatName)

			polecatFile := filepath.Join("/tmp", "test-merge-polecat.yaml")
			err = os.WriteFile(polecatFile, []byte(polecatYAML), 0644)
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command("kubectl", "apply", "-f", polecatFile)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create Polecat")

			By("adding Available=True condition to Polecat (simulating work completion)")
			// Use kubectl patch to add the Available condition
			patchJSON := `[{"op":"replace","path":"/status/conditions","value":[{"type":"Available","status":"True","reason":"WorkComplete","message":"Work completed successfully","lastTransitionTime":"2026-01-01T00:00:00Z"}]}]`
			cmd = exec.Command("kubectl", "patch", "polecat", polecatName,
				"-n", namespace, "--type=json", "--subresource=status",
				"-p", patchJSON)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to patch Polecat status")

			By("verifying Polecat has Available=True condition")
			verifyPolecatAvailable := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "polecat", polecatName,
					"-n", namespace,
					"-o", "jsonpath={.status.conditions[?(@.type=='Available')].status}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("True"))
			}
			Eventually(verifyPolecatAvailable, 30*time.Second).Should(Succeed())

			By("verifying Refinery detects the merge-ready Polecat (queue length > 0)")
			verifyRefineryQueue := func(g Gomega) {
				// Trigger reconciliation by getting the refinery
				cmd := exec.Command("kubectl", "get", "refinery", refineryName,
					"-n", namespace, "-o", "jsonpath={.status.queueLength}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				// Queue should have at least 1 item
				g.Expect(output).NotTo(Equal("0"), "Refinery should detect merge-ready polecat")
			}
			// Note: This may take time as Refinery reconciles on its interval
			Eventually(verifyRefineryQueue, 3*time.Minute, 5*time.Second).Should(Succeed())

			By("cleaning up test resources")
			cmd = exec.Command("kubectl", "delete", "polecat", polecatName, "-n", namespace)
			_, _ = utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "rig", rigName)
			_, _ = utils.Run(cmd)
		})

		It("should create a Pod for a Polecat in kubernetes execution mode", func() {
			polecatName := "test-k8s-polecat"
			testNamespace := namespace // Use the operator's namespace for testing

			By("creating prerequisite secrets for the Polecat")
			// Create git credentials secret (dummy for testing)
			gitSecretCmd := exec.Command("kubectl", "create", "secret", "generic",
				"test-git-creds", "-n", testNamespace,
				"--from-literal=ssh-privatekey=dummy-key-for-testing")
			_, err := utils.Run(gitSecretCmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create git credentials secret")

			// Create Claude API key secret
			claudeSecretCmd := exec.Command("kubectl", "create", "secret", "generic",
				"test-claude-creds", "-n", testNamespace,
				"--from-literal=api-key=dummy-key-for-testing")
			_, err = utils.Run(claudeSecretCmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create Claude credentials secret")

			By("creating a Polecat with kubernetes execution mode")
			polecatYAML := fmt.Sprintf(`
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: %s
  namespace: %s
spec:
  rig: test-rig
  desiredState: Working
  executionMode: kubernetes
  beadID: test-1234
  kubernetes:
    gitRepository: git@github.com:test/repo.git
    gitBranch: main
    gitSecretRef:
      name: test-git-creds
    apiKeySecretRef:
      name: test-claude-creds
      key: api-key
`, polecatName, testNamespace)

			// Write YAML to temp file
			polecatFile := filepath.Join("/tmp", "test-k8s-polecat.yaml")
			err = os.WriteFile(polecatFile, []byte(polecatYAML), 0644)
			Expect(err).NotTo(HaveOccurred())

			cmd := exec.Command("kubectl", "apply", "-f", polecatFile)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create Polecat")

			By("verifying the controller creates a Pod for the Polecat")
			verifyPodCreated := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pod",
					fmt.Sprintf("polecat-%s", polecatName),
					"-n", testNamespace,
					"-o", "jsonpath={.metadata.name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Pod should be created")
				g.Expect(output).To(Equal(fmt.Sprintf("polecat-%s", polecatName)))
			}
			Eventually(verifyPodCreated, 2*time.Minute, time.Second).Should(Succeed())

			By("verifying the Pod has correct labels")
			verifyPodLabels := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pod",
					fmt.Sprintf("polecat-%s", polecatName),
					"-n", testNamespace,
					"-o", "jsonpath={.metadata.labels}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("gastown.io/polecat"))
				g.Expect(output).To(ContainSubstring("gastown.io/rig"))
				g.Expect(output).To(ContainSubstring("gastown.io/bead"))
			}
			Eventually(verifyPodLabels, 30*time.Second).Should(Succeed())

			By("verifying the Pod has init container and main container")
			verifyPodContainers := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pod",
					fmt.Sprintf("polecat-%s", polecatName),
					"-n", testNamespace,
					"-o", "jsonpath={.spec.initContainers[*].name},{.spec.containers[*].name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("git-init"))
				g.Expect(output).To(ContainSubstring("claude"))
			}
			Eventually(verifyPodContainers, 30*time.Second).Should(Succeed())

			By("verifying the Polecat status reflects Pod creation")
			verifyPolecatStatus := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "polecat",
					polecatName, "-n", testNamespace,
					"-o", "jsonpath={.status.podName}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal(fmt.Sprintf("polecat-%s", polecatName)))
			}
			Eventually(verifyPolecatStatus, 2*time.Minute, time.Second).Should(Succeed())

			By("verifying the Polecat phase is Working")
			verifyPolecatPhase := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "polecat",
					polecatName, "-n", testNamespace,
					"-o", "jsonpath={.status.phase}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Working"))
			}
			Eventually(verifyPolecatPhase, 2*time.Minute, time.Second).Should(Succeed())

			By("cleaning up the test Polecat")
			cmd = exec.Command("kubectl", "delete", "polecat", polecatName, "-n", testNamespace)
			_, _ = utils.Run(cmd)

			By("verifying the Pod is deleted with the Polecat (owner reference)")
			verifyPodDeleted := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pod",
					fmt.Sprintf("polecat-%s", polecatName),
					"-n", testNamespace)
				_, err := utils.Run(cmd)
				g.Expect(err).To(HaveOccurred(), "Pod should be deleted")
			}
			Eventually(verifyPodDeleted, 2*time.Minute, time.Second).Should(Succeed())

			By("cleaning up test secrets")
			cmd = exec.Command("kubectl", "delete", "secret", "test-git-creds", "-n", testNamespace)
			_, _ = utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "secret", "test-claude-creds", "-n", testNamespace)
			_, _ = utils.Run(cmd)
		})
	})
})

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() (string, error) {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	return utils.Run(cmd)
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
