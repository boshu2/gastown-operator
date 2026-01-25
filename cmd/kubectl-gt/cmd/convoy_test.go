package cmd

import (
	"testing"
)

func TestNewConvoyCmd(t *testing.T) {
	cmd := newConvoyCmd()

	if cmd.Use != "convoy" {
		t.Errorf("expected Use to be 'convoy', got %s", cmd.Use)
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

func TestNewConvoyListCmd(t *testing.T) {
	cmd := newConvoyListCmd()

	if cmd.Use != "list" {
		t.Errorf("expected Use to be 'list', got %s", cmd.Use)
	}

	// Check output flag
	if cmd.Flags().Lookup("output") == nil {
		t.Error("expected --output flag to exist")
	}
}

func TestNewConvoyStatusCmd(t *testing.T) {
	cmd := newConvoyStatusCmd()

	if cmd.Use != "status <id>" {
		t.Errorf("expected Use to be 'status <id>', got %s", cmd.Use)
	}

	// Check output flag
	if cmd.Flags().Lookup("output") == nil {
		t.Error("expected --output flag to exist")
	}
}

func TestNewConvoyCreateCmd(t *testing.T) {
	cmd := newConvoyCreateCmd()

	if cmd.Use != "create <description> <bead1> [bead2] ..." {
		t.Errorf("expected Use to be 'create <description> <bead1> [bead2] ...', got %s", cmd.Use)
	}
}
