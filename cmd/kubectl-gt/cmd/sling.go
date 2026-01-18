package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

func newSlingCmd() *cobra.Command {
	var wait bool
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "sling <bead-id> <rig>",
		Short: "Dispatch work to a polecat",
		Long: `Dispatch a bead to be worked on by a polecat in the specified rig.

This creates a Polecat CR with the given bead ID and desiredState=Working.
The operator will reconcile the Polecat and create a Pod to execute the work.`,
		Args: cobra.ExactArgs(2),
		Example: `  # Sling a bead to a rig
  kubectl gt sling dm-0001 my-rig

  # Sling and wait for scheduling
  kubectl gt sling dm-0001 my-rig --wait

  # Sling with custom timeout
  kubectl gt sling dm-0001 my-rig --wait --timeout=5m`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSling(args[0], args[1], wait, timeout)
		},
	}

	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for polecat to be scheduled")
	cmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "Timeout for --wait")

	return cmd
}

func runSling(beadID, rig string, wait bool, timeout time.Duration) error {
	config, err := KubeFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Verify rig exists
	_, err = client.Resource(rigGVR).Get(context.Background(), rig, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("rig %s not found: %w", rig, err)
	}

	// Generate polecat name
	polecatName := generatePolecatName(rig)
	namespace := GetNamespace()

	// Create Polecat CR
	polecat := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gastown.gastown.io/v1alpha1",
			"kind":       "Polecat",
			"metadata": map[string]interface{}{
				"name":      polecatName,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"rig":           rig,
				"beadID":        beadID,
				"desiredState":  "Working",
				"executionMode": "kubernetes",
				"kubernetes": map[string]interface{}{
					"claudeCredsSecretRef": map[string]interface{}{
						"name": "claude-creds",
					},
				},
			},
		},
	}

	created, err := client.Resource(polecatGVR).Namespace(namespace).Create(context.Background(), polecat, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create polecat: %w", err)
	}

	fmt.Printf("Polecat %s created for bead %s in rig %s\n", created.GetName(), beadID, rig)

	if wait {
		fmt.Printf("Waiting for polecat to be scheduled (timeout: %s)...\n", timeout)
		err = waitForPolecat(client, namespace, polecatName, timeout)
		if err != nil {
			return err
		}
	}

	return nil
}

func generatePolecatName(rig string) string {
	// Simple name generation: rig-<random>
	// In production, this would use a name pool like the gt CLI does
	rand.Seed(time.Now().UnixNano())
	suffix := fmt.Sprintf("%04x", rand.Intn(0xFFFF))
	return fmt.Sprintf("%s-%s", rig, suffix)
}

func waitForPolecat(client dynamic.Interface, namespace, name string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for polecat %s", name)
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
				return nil
			case "Stuck", "Failed":
				conditions, _, _ := unstructured.NestedSlice(polecat.Object, "status", "conditions")
				msg := "unknown reason"
				for _, c := range conditions {
					cond := c.(map[string]interface{})
					if condType, _ := cond["type"].(string); condType == "Ready" {
						if m, _ := cond["message"].(string); m != "" {
							msg = m
						}
					}
				}
				return fmt.Errorf("polecat %s is %s: %s", name, phase, msg)
			default:
				fmt.Printf("  Phase: %s...\n", phase)
			}
		}
	}
}
