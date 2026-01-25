package cmd

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func newSlingCmd() *cobra.Command {
	var wait bool
	var waitReady bool
	var timeout time.Duration
	var polecatName string
	var nameTheme string
	var gitSecret string

	cmd := &cobra.Command{
		Use:   "sling <bead-id> <rig>",
		Short: "Dispatch work to a polecat",
		Long: `Dispatch a bead to be worked on by a polecat in the specified rig.

This creates a Polecat CR with the given bead ID and desiredState=Working.
The operator will reconcile the Polecat and create a Pod to execute the work.

The git repository URL is automatically fetched from the Rig's gitURL field.`,
		Args: cobra.ExactArgs(2),
		Example: `  # Sling a bead to a rig
  kubectl gt sling dm-0001 my-rig

  # Sling and wait for scheduling
  kubectl gt sling dm-0001 my-rig --wait

  # Sling with a specific name
  kubectl gt sling dm-0001 my-rig --name furiosa

  # Sling with themed name
  kubectl gt sling dm-0001 my-rig --theme mad-max

  # Sling and wait for pod to be ready
  kubectl gt sling dm-0001 my-rig --wait-ready --timeout=5m

  # Sling with custom git secret
  kubectl gt sling dm-0001 my-rig --git-secret my-git-creds`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSling(args[0], args[1], wait, waitReady, timeout, polecatName, nameTheme, gitSecret)
		},
	}

	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for polecat to be scheduled")
	cmd.Flags().BoolVar(&waitReady, "wait-ready", false, "Wait for pod to be running and ready")
	cmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "Timeout for --wait or --wait-ready")
	cmd.Flags().StringVar(&polecatName, "name", "", "Explicit polecat name (e.g., furiosa)")
	cmd.Flags().StringVar(&nameTheme, "theme", "", "Naming theme (mad-max, minerals, wasteland)")
	cmd.Flags().StringVar(&gitSecret, "git-secret", "git-creds", "Name of Secret containing git credentials")

	return cmd
}

func runSling(beadID, rigName string, wait, waitReady bool, timeout time.Duration,
	explicitName, theme, gitSecret string) error {
	config, err := KubeFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Fetch rig to get gitURL
	rig, err := client.Resource(rigGVR).Get(context.Background(), rigName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("rig %s not found: %w", rigName, err)
	}

	gitURL, _, err := unstructured.NestedString(rig.Object, "spec", "gitURL")
	if err != nil || gitURL == "" {
		return fmt.Errorf("rig %s has no gitURL configured", rigName)
	}

	// Generate polecat name based on flags
	var polecatName string
	if explicitName != "" {
		polecatName = explicitName
	} else if theme != "" {
		polecatName = generateThemedName(rigName, theme)
	} else {
		polecatName = generatePolecatName(rigName)
	}
	namespace := GetNamespace()

	// Create Polecat CR
	polecat := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "gastown.gastown.io/v1alpha1",
			"kind":       "Polecat",
			"metadata": map[string]any{
				"name":      polecatName,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"rig":           rigName,
				"beadID":        beadID,
				"desiredState":  "Working",
				"executionMode": "kubernetes",
				"kubernetes": map[string]any{
					"gitRepository": gitURL,
					"gitSecretRef": map[string]any{
						"name": gitSecret,
					},
					"claudeCredsSecretRef": map[string]any{
						"name": "claude-creds",
					},
				},
			},
		},
	}

	ctx := context.Background()
	created, err := client.Resource(polecatGVR).Namespace(namespace).Create(ctx, polecat, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create polecat: %w", err)
	}

	fmt.Printf("Polecat %s created for bead %s in rig %s\n", created.GetName(), beadID, rigName)

	if wait || waitReady {
		fmt.Printf("Waiting for polecat to be scheduled (timeout: %s)...\n", timeout)
		podName, err := waitForPolecatScheduled(client, namespace, polecatName, timeout)
		if err != nil {
			return err
		}

		if waitReady && podName != "" {
			fmt.Printf("Waiting for pod %s to be ready...\n", podName)
			err = waitForPodReady(config, namespace, podName, timeout)
			if err != nil {
				return err
			}
			fmt.Printf("Pod %s is ready\n", podName)
		}
	}

	return nil
}

func generatePolecatName(rig string) string {
	// Simple name generation: rig-<random>
	b := make([]byte, 2)
	_, _ = rand.Read(b)
	suffix := fmt.Sprintf("%04x", b)
	return fmt.Sprintf("%s-%s", rig, suffix)
}

// Themed name pools for polecat naming
var nameThemes = map[string][]string{
	"mad-max": {
		"furiosa", "nux", "slit", "rictus", "capable", "toast", "dag", "cheedo",
		"angharad", "immortan", "keeper", "valkyrie", "ace", "morsov", "corpus",
		"war-boy", "doof", "coma", "organic", "prime", "scrotus", "hope", "glory",
	},
	"minerals": {
		"obsidian", "quartz", "jasper", "onyx", "opal", "topaz", "amber", "jade",
		"ruby", "sapphire", "emerald", "diamond", "garnet", "pearl", "coral",
		"crystal", "flint", "granite", "marble", "slate", "basalt", "pumice",
	},
	"wasteland": {
		"rust", "chrome", "nitro", "guzzle", "witness", "shiny", "fury", "road",
		"thunder", "dust", "storm", "blade", "spike", "chain", "gear", "bolt",
		"piston", "diesel", "vapor", "scrap", "iron", "steel",
	},
}

func generateThemedName(rig, theme string) string {
	names, ok := nameThemes[theme]
	if !ok {
		// Fall back to random if theme not found
		return generatePolecatName(rig)
	}

	// Pick a random name from the theme
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	idx := int(b[0]) % len(names)
	return names[idx]
}

func waitForPolecatScheduled(client dynamic.Interface, namespace, name string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("timeout waiting for polecat %s", name)
		case <-ticker.C:
			polecat, err := client.Resource(polecatGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				continue
			}

			phase, _, _ := unstructured.NestedString(polecat.Object, "status", "phase")
			switch phase {
			case "Working":
				podName, _, _ := unstructured.NestedString(polecat.Object, "status", "podName")
				fmt.Printf("Polecat %s is working (pod: %s)\n", name, podName)
				return podName, nil
			case "Stuck", "Failed":
				conditions, _, _ := unstructured.NestedSlice(polecat.Object, "status", "conditions")
				msg := "unknown reason"
				for _, c := range conditions {
					cond, _ := c.(map[string]any)
					if condType, _ := cond["type"].(string); condType == "Ready" {
						if m, _ := cond["message"].(string); m != "" {
							msg = m
						}
					}
				}
				return "", fmt.Errorf("polecat %s is %s: %s", name, phase, msg)
			default:
				fmt.Printf("  Phase: %s...\n", phase)
			}
		}
	}
}

func waitForPodReady(config *rest.Config, namespace, podName string, timeout time.Duration) error {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for pod %s to be ready", podName)
		case <-ticker.C:
			pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
			if err != nil {
				continue
			}

			// Check if pod is running
			if pod.Status.Phase != corev1.PodRunning {
				fmt.Printf("  Pod phase: %s...\n", pod.Status.Phase)
				continue
			}

			// Check if all containers are ready
			allReady := true
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status != corev1.ConditionTrue {
					allReady = false
					break
				}
			}

			if allReady {
				return nil
			}
			fmt.Printf("  Pod running, waiting for Ready condition...\n")
		}
	}
}
