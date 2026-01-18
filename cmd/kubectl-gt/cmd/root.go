package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	// Version info (set via ldflags)
	Version   = "dev"
	GitCommit = "none"
	BuildDate = "unknown"
)

// KubeFlags holds the kubernetes configuration flags
var KubeFlags *genericclioptions.ConfigFlags

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kubectl-gt",
	Short: "kubectl plugin for Gas Town operator",
	Long: `kubectl-gt is a kubectl plugin for managing Gas Town resources in Kubernetes.

It provides commands to manage Rigs, Polecats, Convoys, and authentication
for the Gas Town operator. Following the CNPG pattern, this CLI creates
CRDs that the operator reconciles into actual workloads.

Examples:
  # List all rigs
  kubectl gt rig list

  # Dispatch work to a polecat
  kubectl gt sling dm-0001 my-rig

  # Check polecat status
  kubectl gt polecat status my-rig/toast-001

  # Sync Claude credentials
  kubectl gt auth sync`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Initialize kubernetes config flags
	KubeFlags = genericclioptions.NewConfigFlags(true)
	KubeFlags.AddFlags(rootCmd.PersistentFlags())

	// Add subcommands
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newRigCmd())
	rootCmd.AddCommand(newPolecatCmd())
	rootCmd.AddCommand(newSlingCmd())
	rootCmd.AddCommand(newConvoyCmd())
	rootCmd.AddCommand(newAuthCmd())
}

// newVersionCmd creates the version command
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("kubectl-gt %s\n", Version)
			fmt.Printf("  Git commit: %s\n", GitCommit)
			fmt.Printf("  Build date: %s\n", BuildDate)
		},
	}
}

// GetKubeClient returns a kubernetes client from the current flags
func GetKubeClient() error {
	// This will be implemented when we add the client package
	return nil
}

// GetNamespace returns the namespace from flags or default
func GetNamespace() string {
	if KubeFlags.Namespace != nil && *KubeFlags.Namespace != "" {
		return *KubeFlags.Namespace
	}
	return "gastown"
}

// PrintError prints an error message to stderr
func PrintError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}
