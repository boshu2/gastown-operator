package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

var polecatGVR = schema.GroupVersionResource{
	Group:    "gastown.gastown.io",
	Version:  "v1alpha1",
	Resource: "polecats",
}

func newPolecatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "polecat",
		Short: "Manage polecat workers",
		Long:  `Commands for listing, viewing, and managing polecat workers.`,
	}

	cmd.AddCommand(newPolecatListCmd())
	cmd.AddCommand(newPolecatStatusCmd())
	cmd.AddCommand(newPolecatLogsCmd())
	cmd.AddCommand(newPolecatNukeCmd())

	return cmd
}

func newPolecatListCmd() *cobra.Command {
	var rig string
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "list [rig]",
		Short: "List polecats",
		Example: `  # List all polecats
  kubectl gt polecat list

  # List polecats for a specific rig
  kubectl gt polecat list my-rig

  # Output as JSON
  kubectl gt polecat list -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				rig = args[0]
			}
			return runPolecatList(rig, outputFormat)
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json, yaml)")

	return cmd
}

func newPolecatStatusCmd() *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "status <rig>/<name>",
		Short: "Show polecat details",
		Args:  cobra.ExactArgs(1),
		Example: `  # Show polecat status
  kubectl gt polecat status my-rig/toast-001

  # Output as JSON
  kubectl gt polecat status my-rig/toast-001 -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			parts := strings.SplitN(args[0], "/", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid format: use <rig>/<name>")
			}
			return runPolecatStatus(parts[0], parts[1], outputFormat)
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json, yaml)")

	return cmd
}

func newPolecatLogsCmd() *cobra.Command {
	var follow bool
	var container string

	cmd := &cobra.Command{
		Use:   "logs <rig>/<name>",
		Short: "Stream polecat pod logs",
		Args:  cobra.ExactArgs(1),
		Example: `  # Stream logs
  kubectl gt polecat logs my-rig/toast-001

  # Follow logs
  kubectl gt polecat logs my-rig/toast-001 -f`,
		RunE: func(cmd *cobra.Command, args []string) error {
			parts := strings.SplitN(args[0], "/", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid format: use <rig>/<name>")
			}
			return runPolecatLogs(parts[0], parts[1], follow, container)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	cmd.Flags().StringVarP(&container, "container", "c", "claude", "Container name")

	return cmd
}

func newPolecatNukeCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "nuke <rig>/<name>",
		Short: "Terminate a polecat",
		Args:  cobra.ExactArgs(1),
		Example: `  # Terminate polecat
  kubectl gt polecat nuke my-rig/toast-001

  # Force terminate
  kubectl gt polecat nuke my-rig/toast-001 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			parts := strings.SplitN(args[0], "/", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid format: use <rig>/<name>")
			}
			return runPolecatNuke(parts[0], parts[1], force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force termination without cleanup")

	return cmd
}

func runPolecatList(rig, outputFormat string) error {
	config, err := KubeFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	namespace := GetNamespace()
	list, err := client.Resource(polecatGVR).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list polecats: %w", err)
	}

	// Filter by rig if specified
	items := make([]unstructured.Unstructured, 0, len(list.Items))
	for _, item := range list.Items {
		if rig != "" {
			itemRig, _, _ := unstructured.NestedString(item.Object, "spec", "rig")
			if itemRig != rig {
				continue
			}
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		if outputFormat == OutputFormatJSON {
			fmt.Println("[]")
			return nil
		}
		if outputFormat == OutputFormatYAML {
			fmt.Println("items: []")
			return nil
		}
		if rig != "" {
			fmt.Printf("No polecats found for rig %s\n", rig)
		} else {
			fmt.Println("No polecats found")
		}
		return nil
	}

	switch outputFormat {
	case OutputFormatYAML:
		for i, item := range items {
			if i > 0 {
				fmt.Println("---")
			}
			data, err := yaml.Marshal(item.Object)
			if err != nil {
				return fmt.Errorf("failed to marshal polecat: %w", err)
			}
			fmt.Print(string(data))
		}
	case OutputFormatJSON:
		// Output as JSON array
		objects := make([]map[string]any, len(items))
		for i, item := range items {
			objects[i] = item.Object
		}
		data, err := json.MarshalIndent(objects, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal polecats: %w", err)
		}
		fmt.Println(string(data))
	default:
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "NAME\tRIG\tBEAD\tPHASE\tPOD\tAGE")
		for _, item := range items {
			name := item.GetName()
			itemRig, _, _ := unstructured.NestedString(item.Object, "spec", "rig")
			beadID, _, _ := unstructured.NestedString(item.Object, "spec", "beadID")
			phase, _, _ := unstructured.NestedString(item.Object, "status", "phase")
			podName, _, _ := unstructured.NestedString(item.Object, "status", "podName")
			age := item.GetCreationTimestamp().Time

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				name, itemRig, beadID, phase, podName, formatAge(age))
		}
		_ = w.Flush()
	}

	return nil
}

//nolint:gocyclo // Complexity from exhaustive status field printing; linear and readable
func runPolecatStatus(rig, name, outputFormat string) error {
	config, err := KubeFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	namespace := GetNamespace()
	polecat, err := client.Resource(polecatGVR).Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get polecat %s: %w", name, err)
	}

	// Verify rig matches
	actualRig, _, _ := unstructured.NestedString(polecat.Object, "spec", "rig")
	if actualRig != rig {
		return fmt.Errorf("polecat %s belongs to rig %s, not %s", name, actualRig, rig)
	}

	switch outputFormat {
	case OutputFormatYAML:
		data, err := yaml.Marshal(polecat.Object)
		if err != nil {
			return fmt.Errorf("failed to marshal polecat: %w", err)
		}
		fmt.Print(string(data))
	case OutputFormatJSON:
		data, err := json.MarshalIndent(polecat.Object, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal polecat: %w", err)
		}
		fmt.Println(string(data))
	default:
		// Print polecat details
		fmt.Printf("Name:           %s\n", polecat.GetName())
		fmt.Printf("Rig:            %s\n", actualRig)

		if beadID, ok, _ := unstructured.NestedString(polecat.Object, "spec", "beadID"); ok {
			fmt.Printf("Bead ID:        %s\n", beadID)
		}
		if desiredState, ok, _ := unstructured.NestedString(polecat.Object, "spec", "desiredState"); ok {
			fmt.Printf("Desired State:  %s\n", desiredState)
		}
		if execMode, ok, _ := unstructured.NestedString(polecat.Object, "spec", "executionMode"); ok {
			fmt.Printf("Execution Mode: %s\n", execMode)
		}

		fmt.Println()

		// Status
		if phase, ok, _ := unstructured.NestedString(polecat.Object, "status", "phase"); ok {
			fmt.Printf("Phase:          %s\n", phase)
		}
		if podName, ok, _ := unstructured.NestedString(polecat.Object, "status", "podName"); ok && podName != "" {
			fmt.Printf("Pod:            %s\n", podName)
		}
		if branch, ok, _ := unstructured.NestedString(polecat.Object, "status", "branch"); ok && branch != "" {
			fmt.Printf("Branch:         %s\n", branch)
		}

		// Conditions
		if conditions, ok, _ := unstructured.NestedSlice(polecat.Object, "status", "conditions"); ok && len(conditions) > 0 {
			fmt.Println("\nConditions:")
			for _, c := range conditions {
				cond, _ := c.(map[string]any)
				condType, _ := cond["type"].(string)
				status, _ := cond["status"].(string)
				reason, _ := cond["reason"].(string)
				message, _ := cond["message"].(string)
				fmt.Printf("  %s: %s (%s)\n", condType, status, reason)
				if message != "" {
					fmt.Printf("    %s\n", message)
				}
			}
		}
	}

	return nil
}

func runPolecatLogs(_, name string, follow bool, container string) error {
	// First get the polecat to find its pod name
	config, err := KubeFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	namespace := GetNamespace()
	ctx := context.Background()
	polecat, err := dynClient.Resource(polecatGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get polecat %s: %w", name, err)
	}

	podName, ok, _ := unstructured.NestedString(polecat.Object, "status", "podName")
	if !ok || podName == "" {
		return fmt.Errorf("polecat %s has no associated pod", name)
	}

	// Create kubernetes clientset for log streaming
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Streaming logs from pod %s, container %s...\n\n", podName, container)

	return streamPodLogs(clientset, namespace, podName, container, follow)
}

func streamPodLogs(clientset *kubernetes.Clientset, namespace, podName, container string, follow bool) error {
	logOptions := &corev1.PodLogOptions{
		Container: container,
		Follow:    follow,
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, logOptions)
	stream, err := req.Stream(context.Background())
	if err != nil {
		return fmt.Errorf("failed to open log stream: %w", err)
	}
	defer func() { _ = stream.Close() }()

	_, err = io.Copy(os.Stdout, stream)
	if err != nil && err != io.EOF {
		return fmt.Errorf("error streaming logs: %w", err)
	}

	return nil
}

func runPolecatNuke(rig, name string, force bool) error {
	config, err := KubeFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	namespace := GetNamespace()

	// Get the polecat
	polecat, err := client.Resource(polecatGVR).Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get polecat %s: %w", name, err)
	}

	// Verify rig matches
	actualRig, _, _ := unstructured.NestedString(polecat.Object, "spec", "rig")
	if actualRig != rig {
		return fmt.Errorf("polecat %s belongs to rig %s, not %s", name, actualRig, rig)
	}

	// Update desiredState to Terminated
	_ = unstructured.SetNestedField(polecat.Object, "Terminated", "spec", "desiredState")

	_, err = client.Resource(polecatGVR).Namespace(namespace).Update(context.Background(), polecat, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update polecat: %w", err)
	}

	if force {
		fmt.Printf("Polecat %s/%s marked for forced termination\n", rig, name)
	} else {
		fmt.Printf("Polecat %s/%s marked for termination (will cleanup first)\n", rig, name)
	}

	return nil
}
