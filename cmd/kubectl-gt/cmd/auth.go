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
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync ~/.claude/ to Kubernetes Secret",
		Long: `Sync Claude credentials from local ~/.claude/ directory to a Kubernetes Secret.

This allows polecats running in the cluster to use your Claude account.
The Secret is created in the target namespace (default: gastown).

SECURITY: Only credential files are synced (.credentials.json, settings.json).
Other files (conversation history, cache) are not uploaded for GDPR data minimization.`,
		Example: `  # Sync credentials
  kubectl gt auth sync

  # Preview what would be synced (dry-run)
  kubectl gt auth sync --dry-run

  # Sync from custom location
  kubectl gt auth sync --claude-dir /path/to/.claude

  # Force sync even if up to date
  kubectl gt auth sync --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthSync(claudeDir, force, dryRun)
		},
	}

	homeDir, _ := os.UserHomeDir()
	defaultClaudeDir := filepath.Join(homeDir, ".claude")

	cmd.Flags().StringVar(&claudeDir, "claude-dir", defaultClaudeDir, "Path to Claude config directory")
	cmd.Flags().BoolVar(&force, "force", false, "Force sync even if Secret exists")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview files to sync without uploading")

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

// Allowlist of credential files to sync (GDPR data minimization)
var allowedFiles = []string{
	".credentials.json", // OAuth credentials
	"settings.json",     // User settings (API keys if configured)
}

func runAuthSync(claudeDir string, force bool, dryRun bool) error {
	data, foundFiles, missingFiles, err := loadCredentialFiles(claudeDir)
	if err != nil {
		return err
	}

	warnUnallowedFiles(claudeDir)

	if dryRun {
		printDryRunPreview(data, foundFiles, missingFiles)
		return nil
	}

	operation, err := createOrUpdateSecret(data, force)
	if err != nil {
		return err
	}

	if operation != "" {
		namespace := GetNamespace()
		fmt.Printf("AUDIT: Credential sync - operation=%s namespace=%s secret=%s files=%d timestamp=%s\n",
			operation, namespace, claudeCredsSecretName, len(data), time.Now().UTC().Format(time.RFC3339))
		fmt.Printf("Synced %d files to Secret %s/%s\n", len(data), namespace, claudeCredsSecretName)
		fmt.Println("\nPolecats can now use your Claude credentials.")
	}
	return nil
}

// loadCredentialFiles reads allowlisted credential files from the Claude config directory.
// Returns the file data, lists of found and missing files, or an error.
func loadCredentialFiles(claudeDir string) (map[string][]byte, []string, []string, error) {
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		return nil, nil, nil, fmt.Errorf("claude config directory not found: %s (run 'claude login' first)", claudeDir)
	}

	data := make(map[string][]byte)
	var foundFiles []string
	var missingFiles []string

	for _, fileName := range allowedFiles {
		filePath := filepath.Join(claudeDir, fileName)
		// nolint:gosec // G304: path is constrained to allowlisted files only
		content, err := os.ReadFile(filePath)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, nil, nil, fmt.Errorf("failed to read %s: %w", fileName, err)
			}
			missingFiles = append(missingFiles, fileName)
			continue
		}
		data[fileName] = content
		foundFiles = append(foundFiles, fileName)
	}

	if len(data) == 0 {
		return nil, nil, nil, fmt.Errorf("no credential files found in %s (expected: %v)", claudeDir, allowedFiles)
	}

	return data, foundFiles, missingFiles, nil
}

// warnUnallowedFiles walks the Claude config directory and prints warnings
// for any files not in the credential allowlist.
func warnUnallowedFiles(claudeDir string) {
	err := filepath.Walk(claudeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(claudeDir, path)
		for _, allowed := range allowedFiles {
			if relPath == allowed {
				return nil
			}
		}
		fmt.Printf("  ⚠️  Skipping: %s (not in allowlist)\n", relPath)
		return nil
	})
	if err != nil {
		fmt.Printf("Warning: could not scan for unexpected files: %v\n", err)
	}
}

// printDryRunPreview displays what would be synced without actually uploading.
func printDryRunPreview(data map[string][]byte, foundFiles []string, missingFiles []string) {
	fmt.Println("Dry-run mode: would sync the following files:")
	for _, file := range foundFiles {
		fmt.Printf("  ✓ %s (%d bytes)\n", file, len(data[file]))
	}
	if len(missingFiles) > 0 {
		fmt.Println("\nMissing files (optional):")
		for _, file := range missingFiles {
			fmt.Printf("  ✗ %s\n", file)
		}
	}
	fmt.Printf("\nTotal: %d files, %d bytes\n", len(data), sumBytes(data))
	fmt.Println("\nRe-run without --dry-run to actually sync.")
}

// createOrUpdateSecret syncs credential data to a Kubernetes Secret.
// Returns the operation performed ("create", "update", or "" if skipped) and any error.
func createOrUpdateSecret(data map[string][]byte, force bool) (string, error) {
	config, err := KubeFlags.ToRESTConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}

	namespace := GetNamespace()
	ctx := context.Background()

	existing, err := clientset.CoreV1().Secrets(namespace).Get(ctx, claudeCredsSecretName, metav1.GetOptions{})
	if err == nil && !force {
		lastSync := existing.Annotations[syncTimestampKey]
		fmt.Printf("Secret %s already exists (last sync: %s)\n", claudeCredsSecretName, lastSync)
		fmt.Println("Use --force to overwrite")
		return "", nil
	}

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

	operation := "create"
	if existing != nil && existing.Name != "" {
		_, err = clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
		operation = "update"
	} else {
		_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	}

	if err != nil {
		return "", fmt.Errorf("failed to create/update Secret: %w", err)
	}

	return operation, nil
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

func sumBytes(data map[string][]byte) int {
	total := 0
	for _, content := range data {
		total += len(content)
	}
	return total
}
