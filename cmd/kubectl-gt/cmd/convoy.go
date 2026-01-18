package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var convoyGVR = schema.GroupVersionResource{
	Group:    "gastown.gastown.io",
	Version:  "v1alpha1",
	Resource: "convoys",
}

func newConvoyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convoy",
		Short: "Manage convoy (batch) tracking",
		Long:  `Commands for creating and tracking convoys (batches of beads).`,
	}

	cmd.AddCommand(newConvoyListCmd())
	cmd.AddCommand(newConvoyStatusCmd())
	cmd.AddCommand(newConvoyCreateCmd())

	return cmd
}

func newConvoyListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all convoys",
		Example: `  # List all convoys
  kubectl gt convoy list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConvoyList()
		},
	}

	return cmd
}

func newConvoyStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <id>",
		Short: "Show convoy details",
		Args:  cobra.ExactArgs(1),
		Example: `  # Show convoy status
  kubectl gt convoy status cv-abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConvoyStatus(args[0])
		},
	}

	return cmd
}

func newConvoyCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <description> <bead1> [bead2] ...",
		Short: "Create a convoy to track beads",
		Args:  cobra.MinimumNArgs(2),
		Example: `  # Create a convoy
  kubectl gt convoy create "Wave 1" dm-0001 dm-0002 dm-0003`,
		RunE: func(cmd *cobra.Command, args []string) error {
			description := args[0]
			beads := args[1:]
			return runConvoyCreate(description, beads)
		},
	}

	return cmd
}

func runConvoyList() error {
	config, err := KubeFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	namespace := GetNamespace()
	list, err := client.Resource(convoyGVR).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list convoys: %w", err)
	}

	if len(list.Items) == 0 {
		fmt.Println("No convoys found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tDESCRIPTION\tCOMPLETED\tPENDING\tPHASE\tAGE")
	for _, item := range list.Items {
		name := item.GetName()
		description, _, _ := unstructured.NestedString(item.Object, "spec", "description")
		phase, _, _ := unstructured.NestedString(item.Object, "status", "phase")
		age := item.GetCreationTimestamp().Time

		// Count beads
		completed, _, _ := unstructured.NestedSlice(item.Object, "status", "completedBeads")
		pending, _, _ := unstructured.NestedSlice(item.Object, "status", "pendingBeads")

		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\t%s\n",
			name, truncate(description, 30), len(completed), len(pending), phase, formatAge(age))
	}
	_ = w.Flush()

	return nil
}

func runConvoyStatus(id string) error {
	config, err := KubeFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	namespace := GetNamespace()
	convoy, err := client.Resource(convoyGVR).Namespace(namespace).Get(context.Background(), id, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get convoy %s: %w", id, err)
	}

	// Print convoy details
	fmt.Printf("ID:          %s\n", convoy.GetName())

	if desc, ok, _ := unstructured.NestedString(convoy.Object, "spec", "description"); ok {
		fmt.Printf("Description: %s\n", desc)
	}

	// Tracked beads from spec
	if beads, ok, _ := unstructured.NestedSlice(convoy.Object, "spec", "trackedBeads"); ok {
		beadIDs := make([]string, 0, len(beads))
		for _, b := range beads {
			if id, ok := b.(string); ok {
				beadIDs = append(beadIDs, id)
			}
		}
		fmt.Printf("Beads:       %s\n", strings.Join(beadIDs, ", "))
	}

	fmt.Println()

	// Status
	if phase, ok, _ := unstructured.NestedString(convoy.Object, "status", "phase"); ok {
		fmt.Printf("Phase:       %s\n", phase)
	}

	// Progress
	completed, _, _ := unstructured.NestedSlice(convoy.Object, "status", "completedBeads")
	pending, _, _ := unstructured.NestedSlice(convoy.Object, "status", "pendingBeads")
	total := len(completed) + len(pending)
	if total > 0 {
		pct := float64(len(completed)) / float64(total) * 100
		fmt.Printf("Progress:    %d/%d (%.0f%%)\n", len(completed), total, pct)
	}

	if len(completed) > 0 {
		fmt.Println("\nCompleted:")
		for _, b := range completed {
			fmt.Printf("  ✓ %v\n", b)
		}
	}

	if len(pending) > 0 {
		fmt.Println("\nPending:")
		for _, b := range pending {
			fmt.Printf("  ○ %v\n", b)
		}
	}

	return nil
}

func runConvoyCreate(description string, beads []string) error {
	config, err := KubeFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	namespace := GetNamespace()

	// Generate convoy name
	convoyName := fmt.Sprintf("cv-%s", generatePolecatName(""))

	// Convert beads to interface slice
	beadSlice := make([]any, len(beads))
	for i, b := range beads {
		beadSlice[i] = b
	}

	convoy := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "gastown.gastown.io/v1alpha1",
			"kind":       "Convoy",
			"metadata": map[string]any{
				"name":      convoyName,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"description":  description,
				"trackedBeads": beadSlice,
			},
		},
	}

	ctx := context.Background()
	created, err := client.Resource(convoyGVR).Namespace(namespace).Create(ctx, convoy, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create convoy: %w", err)
	}

	fmt.Printf("Convoy %s created tracking %d beads\n", created.GetName(), len(beads))
	return nil
}
