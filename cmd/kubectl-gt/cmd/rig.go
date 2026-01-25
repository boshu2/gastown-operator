package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/yaml"
)

var rigGVR = schema.GroupVersionResource{
	Group:    "gastown.gastown.io",
	Version:  "v1alpha1",
	Resource: "rigs",
}

func newRigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rig",
		Short: "Manage Gas Town rigs",
		Long:  `Commands for listing, viewing, and creating Gas Town rigs.`,
	}

	cmd.AddCommand(newRigListCmd())
	cmd.AddCommand(newRigStatusCmd())
	cmd.AddCommand(newRigCreateCmd())

	return cmd
}

func newRigListCmd() *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all rigs",
		Example: `  # List all rigs
  kubectl gt rig list

  # List rigs in a specific namespace
  kubectl gt rig list -n my-namespace

  # Output as YAML
  kubectl gt rig list -o yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRigList(outputFormat)
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, yaml, json)")

	return cmd
}

func newRigStatusCmd() *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "status <name>",
		Short: "Show rig details",
		Args:  cobra.ExactArgs(1),
		Example: `  # Show rig status
  kubectl gt rig status my-rig

  # Output as JSON
  kubectl gt rig status my-rig -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRigStatus(args[0], outputFormat)
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json, yaml)")

	return cmd
}

func newRigCreateCmd() *cobra.Command {
	var gitURL, prefix, localPath string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new rig",
		Args:  cobra.ExactArgs(1),
		Example: `  # Create a rig
  kubectl gt rig create my-rig --git-url https://github.com/org/repo --prefix mr`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRigCreate(args[0], gitURL, prefix, localPath)
		},
	}

	cmd.Flags().StringVar(&gitURL, "git-url", "", "Git repository URL (required)")
	cmd.Flags().StringVar(&prefix, "prefix", "", "Beads prefix (required)")
	cmd.Flags().StringVar(&localPath, "local-path", "", "Local filesystem path (required)")
	_ = cmd.MarkFlagRequired("git-url")
	_ = cmd.MarkFlagRequired("prefix")
	_ = cmd.MarkFlagRequired("local-path")

	return cmd
}

func runRigList(outputFormat string) error {
	config, err := KubeFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Rigs are cluster-scoped
	list, err := client.Resource(rigGVR).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list rigs: %w", err)
	}

	if len(list.Items) == 0 {
		fmt.Println("No rigs found")
		return nil
	}

	switch outputFormat {
	case OutputFormatYAML:
		for i, item := range list.Items {
			if i > 0 {
				fmt.Println("---")
			}
			data, err := yaml.Marshal(item.Object)
			if err != nil {
				return fmt.Errorf("failed to marshal rig: %w", err)
			}
			fmt.Print(string(data))
		}
	case OutputFormatJSON:
		for _, item := range list.Items {
			data, err := json.MarshalIndent(item.Object, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal rig: %w", err)
			}
			fmt.Println(string(data))
		}
	default:
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "NAME\tPREFIX\tGIT-URL\tPHASE\tAGE")
		for _, item := range list.Items {
			name := item.GetName()
			prefix, _, _ := unstructured.NestedString(item.Object, "spec", "beadsPrefix")
			gitURL, _, _ := unstructured.NestedString(item.Object, "spec", "gitURL")
			phase, _, _ := unstructured.NestedString(item.Object, "status", "phase")
			age := item.GetCreationTimestamp().Time

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				name, prefix, truncate(gitURL, 40), phase, formatAge(age))
		}
		_ = w.Flush()
	}

	return nil
}

func runRigStatus(name, outputFormat string) error {
	config, err := KubeFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	rig, err := client.Resource(rigGVR).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get rig %s: %w", name, err)
	}

	switch outputFormat {
	case OutputFormatYAML:
		data, err := yaml.Marshal(rig.Object)
		if err != nil {
			return fmt.Errorf("failed to marshal rig: %w", err)
		}
		fmt.Print(string(data))
	case OutputFormatJSON:
		data, err := json.MarshalIndent(rig.Object, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal rig: %w", err)
		}
		fmt.Println(string(data))
	default:
		// Print rig details
		fmt.Printf("Name:        %s\n", rig.GetName())

		if prefix, ok, _ := unstructured.NestedString(rig.Object, "spec", "beadsPrefix"); ok {
			fmt.Printf("Prefix:      %s\n", prefix)
		}
		if gitURL, ok, _ := unstructured.NestedString(rig.Object, "spec", "gitURL"); ok {
			fmt.Printf("Git URL:     %s\n", gitURL)
		}
		if localPath, ok, _ := unstructured.NestedString(rig.Object, "spec", "localPath"); ok {
			fmt.Printf("Local Path:  %s\n", localPath)
		}

		fmt.Println()

		// Status
		if phase, ok, _ := unstructured.NestedString(rig.Object, "status", "phase"); ok {
			fmt.Printf("Phase:       %s\n", phase)
		}

		// Conditions
		if conditions, ok, _ := unstructured.NestedSlice(rig.Object, "status", "conditions"); ok && len(conditions) > 0 {
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

func runRigCreate(name, gitURL, prefix, localPath string) error {
	config, err := KubeFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	rig := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "gastown.gastown.io/v1alpha1",
			"kind":       "Rig",
			"metadata": map[string]any{
				"name": name,
			},
			"spec": map[string]any{
				"gitURL":      gitURL,
				"beadsPrefix": prefix,
				"localPath":   localPath,
			},
		},
	}

	_, err = client.Resource(rigGVR).Create(context.Background(), rig, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create rig: %w", err)
	}

	fmt.Printf("Rig %s created\n", name)
	return nil
}
