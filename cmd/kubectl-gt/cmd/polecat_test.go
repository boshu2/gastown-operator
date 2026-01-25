package cmd

import (
	"testing"
)

func TestNewPolecatCmd(t *testing.T) {
	cmd := newPolecatCmd()

	if cmd.Use != "polecat" {
		t.Errorf("expected Use to be 'polecat', got %s", cmd.Use)
	}

	// Check subcommands
	expectedSubs := []string{"list", "status", "logs", "nuke"}
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

func TestNewPolecatListCmd(t *testing.T) {
	cmd := newPolecatListCmd()

	if cmd.Use != "list [rig]" {
		t.Errorf("expected Use to be 'list [rig]', got %s", cmd.Use)
	}

	// Check output flag
	if cmd.Flags().Lookup("output") == nil {
		t.Error("expected --output flag to exist")
	}
}

func TestNewPolecatStatusCmd(t *testing.T) {
	cmd := newPolecatStatusCmd()

	if cmd.Use != "status <rig>/<name>" {
		t.Errorf("expected Use to be 'status <rig>/<name>', got %s", cmd.Use)
	}

	// Check output flag
	if cmd.Flags().Lookup("output") == nil {
		t.Error("expected --output flag to exist")
	}
}

func TestNewPolecatLogsCmd(t *testing.T) {
	cmd := newPolecatLogsCmd()

	if cmd.Use != "logs <rig>/<name>" {
		t.Errorf("expected Use to be 'logs <rig>/<name>', got %s", cmd.Use)
	}

	// Check flags
	flags := []string{"follow", "container"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag --%s to exist", flag)
		}
	}
}

func TestNewPolecatNukeCmd(t *testing.T) {
	cmd := newPolecatNukeCmd()

	if cmd.Use != "nuke <rig>/<name>" {
		t.Errorf("expected Use to be 'nuke <rig>/<name>', got %s", cmd.Use)
	}

	// Check force flag
	if cmd.Flags().Lookup("force") == nil {
		t.Error("expected --force flag to exist")
	}
}
