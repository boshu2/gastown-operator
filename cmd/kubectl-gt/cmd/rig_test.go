package cmd

import (
	"testing"
)

func TestNewRigCmd(t *testing.T) {
	cmd := newRigCmd()

	if cmd.Use != "rig" {
		t.Errorf("expected Use to be 'rig', got %s", cmd.Use)
	}

	// Check subcommands
	expectedSubs := []string{"list", "status", "create"}
	for _, sub := range expectedSubs {
		found := false
		for _, c := range cmd.Commands() {
			if c.Name() == sub {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %s to exist", sub)
		}
	}
}

func TestNewRigListCmd(t *testing.T) {
	cmd := newRigListCmd()

	if cmd.Use != "list" {
		t.Errorf("expected Use to be 'list', got %s", cmd.Use)
	}

	// Check output flag
	if cmd.Flags().Lookup("output") == nil {
		t.Error("expected --output flag to exist")
	}
}

func TestNewRigStatusCmd(t *testing.T) {
	cmd := newRigStatusCmd()

	if cmd.Use != "status <name>" {
		t.Errorf("expected Use to be 'status <name>', got %s", cmd.Use)
	}

	// Check output flag
	if cmd.Flags().Lookup("output") == nil {
		t.Error("expected --output flag to exist")
	}
}

func TestNewRigCreateCmd(t *testing.T) {
	cmd := newRigCreateCmd()

	if cmd.Use != "create <name>" {
		t.Errorf("expected Use to be 'create <name>', got %s", cmd.Use)
	}

	// Check required flags
	flags := []string{"git-url", "prefix"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag --%s to exist", flag)
		}
	}
}
