package cmd

import (
	"testing"
)

func TestNewSlingCmd(t *testing.T) {
	cmd := newSlingCmd()

	if cmd.Use != "sling <bead-id> <rig>" {
		t.Errorf("expected Use to be 'sling <bead-id> <rig>', got %s", cmd.Use)
	}

	if cmd.Short != "Dispatch work to a polecat" {
		t.Errorf("expected Short to be 'Dispatch work to a polecat', got %s", cmd.Short)
	}

	// Check flags exist
	flags := []string{"wait", "wait-ready", "timeout", "name", "theme"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag --%s to exist", flag)
		}
	}
}

func TestGeneratePolecatName(t *testing.T) {
	rig := "test-rig"
	name := generatePolecatName(rig)

	if name == "" {
		t.Error("expected non-empty name")
	}

	// Should start with rig name
	if len(name) < len(rig)+1 {
		t.Errorf("expected name to be longer than rig name, got %s", name)
	}
}

func TestGenerateThemedName(t *testing.T) {
	tests := []struct {
		theme    string
		wantPool bool
	}{
		{"mad-max", true},
		{"minerals", true},
		{"wasteland", true},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.theme, func(t *testing.T) {
			name := generateThemedName("test-rig", tt.theme)
			if name == "" {
				t.Error("expected non-empty name")
			}

			if tt.wantPool {
				// Should be from the theme pool (no rig prefix)
				found := false
				for _, n := range nameThemes[tt.theme] {
					if n == name {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected name %s to be in theme pool %s", name, tt.theme)
				}
			}
		})
	}
}
