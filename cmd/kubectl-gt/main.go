/*
kubectl-gt is a kubectl plugin for managing Gas Town resources in Kubernetes.

It provides a CLI interface to create and manage Gas Town CRDs (Rig, Polecat, Convoy)
following the CNPG pattern: CLI creates intent via CRDs, operator handles execution.

Usage:

	kubectl gt <command> [flags]

Available Commands:

	rig        Manage Gas Town rigs
	polecat    Manage polecat workers
	sling      Dispatch work to a polecat
	convoy     Manage convoy (batch) tracking
	auth       Manage Claude authentication
	version    Print version information
*/
package main

import (
	"os"

	"github.com/org/gastown-operator/cmd/kubectl-gt/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
