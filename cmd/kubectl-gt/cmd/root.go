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

// Banner is the Gas Town ASCII art logo
const Banner = `
   ___   _   ___   _____ _____      ___  _
  / __| /_\ / __| |_   _/ _ \ \    / / \| |
 | (_ |/ _ \\__ \   | || (_) \ \/\/ /| .' |
  \___/_/ \_\___/   |_| \___/ \_/\_/ |_|\_|
  ═══════════════════════════════════════════
       WITNESS ME! Shiny and Chrome.
  ═══════════════════════════════════════════
`

// KubeFlags holds the kubernetes configuration flags
var KubeFlags *genericclioptions.ConfigFlags

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kubectl-gt",
	Short: "kubectl plugin for Gas Town operator",
	Long: Banner + `
  AI agent orchestration for Kubernetes.
  Dispatch work. Scale your agent army.

  ` + "\033[1mCOMMANDS\033[0m" + `
    rig       Manage project rigs (workspaces)
    polecat   Manage worker pods
    sling     Dispatch work to a polecat
    convoy    Track batch operations
    auth      Manage Claude credentials

  ` + "\033[1mQUICK START\033[0m" + `
    # Create a rig for your project
    kubectl gt rig create my-project --git-url git@github.com:org/repo --prefix mp

    # Dispatch work (spawns a polecat)
    kubectl gt sling issue-123 my-project --theme mad-max

    # Watch it work
    kubectl gt polecat logs my-project/furiosa -f

  ` + "\033[1mTHEMES\033[0m" + `
    --theme mad-max    Furiosa, Nux, Toast, Capable...
    --theme minerals   Obsidian, Quartz, Jasper...
    --theme wasteland  Rust, Chrome, Nitro...

  Ride eternal, shiny and chrome.`,
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
