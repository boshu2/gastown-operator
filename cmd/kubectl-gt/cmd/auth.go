package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	claudeCredsSecretName = "claude-creds"
	syncTimestampKey      = "gastown.io/last-sync"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Claude authentication",
		Long:  `Commands for syncing and checking Claude credentials.`,
	}

	cmd.AddCommand(newAuthSyncCmd())
	cmd.AddCommand(newAuthStatusCmd())

	return cmd
}

func newAuthSyncCmd() *cobra.Command {
	var claudeDir string
	var force bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync ~/.claude/ to Kubernetes Secret",
		Long: `Sync Claude credentials from local ~/.claude/ directory to a Kubernetes Secret.

This allows polecats running in the cluster to use your Claude account.
The Secret is created in the target namespace (default: gastown).`,
		Example: `  # Sync credentials
  kubectl gt auth sync

  # Sync from custom location
  kubectl gt auth sync --claude-dir /path/to/.claude

  # Force sync even if up to date
  kubectl gt auth sync --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthSync(claudeDir, force)
		},
	}

	homeDir, _ := os.UserHomeDir()
	defaultClaudeDir := filepath.Join(homeDir, ".claude")

	cmd.Flags().StringVar(&claudeDir, "claude-dir", defaultClaudeDir, "Path to Claude config directory")
	cmd.Flags().BoolVar(&force, "force", false, "Force sync even if Secret exists")

	return cmd
}

func newAuthStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check credential status",
		Example: `  # Check status
  kubectl gt auth status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthStatus()
		},
	}

	return cmd
}

func runAuthSync(claudeDir string, force bool) error {
	// Check Claude directory exists
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		return fmt.Errorf("claude config directory not found: %s (run 'claude login' first)", claudeDir)
	}

	// Read all files in Claude directory
	data := make(map[string][]byte)
	err := filepath.Walk(claudeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(claudeDir, path)
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		data[relPath] = content
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to read Claude directory: %w", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("no files found in %s", claudeDir)
	}

	// Get kubernetes client
	config, err := KubeFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	namespace := GetNamespace()

	// Check if Secret exists
	ctx := context.Background()
	existing, err := clientset.CoreV1().Secrets(namespace).Get(ctx, claudeCredsSecretName, metav1.GetOptions{})
	if err == nil && !force {
		lastSync := existing.Annotations[syncTimestampKey]
		fmt.Printf("Secret %s already exists (last sync: %s)\n", claudeCredsSecretName, lastSync)
		fmt.Println("Use --force to overwrite")
		return nil
	}

	// Create or update Secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claudeCredsSecretName,
			Namespace: namespace,
			Annotations: map[string]string{
				syncTimestampKey: time.Now().UTC().Format(time.RFC3339),
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}

	if existing != nil && existing.Name != "" {
		_, err = clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	} else {
		_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	}

	if err != nil {
		return fmt.Errorf("failed to create/update Secret: %w", err)
	}

	fmt.Printf("Synced %d files to Secret %s/%s\n", len(data), namespace, claudeCredsSecretName)
	fmt.Println("\nPolecats can now use your Claude credentials.")
	return nil
}

func runAuthStatus() error {
	config, err := KubeFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	namespace := GetNamespace()
	ctx := context.Background()

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, claudeCredsSecretName, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("Secret %s not found in namespace %s\n", claudeCredsSecretName, namespace)
		fmt.Println("\nRun 'kubectl gt auth sync' to create it.")
		return nil
	}

	fmt.Printf("Secret:     %s/%s\n", namespace, claudeCredsSecretName)
	fmt.Printf("Files:      %d\n", len(secret.Data))

	if lastSync, ok := secret.Annotations[syncTimestampKey]; ok {
		syncTime, _ := time.Parse(time.RFC3339, lastSync)
		age := time.Since(syncTime)

		fmt.Printf("Last Sync:  %s (%s ago)\n", lastSync, formatDuration(age))

		// Warn if stale
		if age > 24*time.Hour {
			fmt.Println("\n⚠️  Credentials may be stale. Consider running 'kubectl gt auth sync --force'")
		}
	}

	// List files (masked)
	fmt.Println("\nFiles:")
	for name := range secret.Data {
		fmt.Printf("  - %s\n", name)
	}

	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
