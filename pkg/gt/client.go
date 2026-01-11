/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gt

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Client wraps the gt CLI tool for operator use
type Client struct {
	// GTPath is the path to the gt binary
	GTPath string

	// TownRoot is the GT_TOWN_ROOT directory
	TownRoot string
}

// NewClient creates a new gt client with default settings
func NewClient(townRoot string) *Client {
	gtPath := "gt"
	if p := os.Getenv("GT_PATH"); p != "" {
		gtPath = p
	}
	return &Client{
		GTPath:   gtPath,
		TownRoot: townRoot,
	}
}

// run executes a gt command and returns stdout
func (c *Client) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, c.GTPath, args...)
	cmd.Env = append(os.Environ(), "GT_TOWN_ROOT="+c.TownRoot)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gt %s: %w: %s", strings.Join(args, " "), err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gt %s: %w", strings.Join(args, " "), err)
	}
	return output, nil
}

// runJSON executes a gt command with --json and unmarshals the result
func (c *Client) runJSON(ctx context.Context, result interface{}, args ...string) error {
	args = append(args, "--json")
	output, err := c.run(ctx, args...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(output, result); err != nil {
		return fmt.Errorf("parse gt output: %w", err)
	}
	return nil
}

// --- Rig Operations ---

// RigList returns all rigs in the town
func (c *Client) RigList(ctx context.Context) ([]RigInfo, error) {
	var result []RigInfo
	if err := c.runJSON(ctx, &result, "rig", "list"); err != nil {
		return nil, err
	}
	return result, nil
}

// RigStatus returns detailed status for a specific rig
func (c *Client) RigStatus(ctx context.Context, name string) (*RigStatus, error) {
	var result RigStatus
	if err := c.runJSON(ctx, &result, "rig", "status", name); err != nil {
		return nil, err
	}
	return &result, nil
}

// RigExists checks if a rig exists
func (c *Client) RigExists(ctx context.Context, name string) (bool, error) {
	rigs, err := c.RigList(ctx)
	if err != nil {
		return false, err
	}
	for _, r := range rigs {
		if r.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// --- Polecat Operations ---

// Sling dispatches work to a polecat (creates one if needed)
func (c *Client) Sling(ctx context.Context, beadID, rig string) error {
	_, err := c.run(ctx, "sling", beadID, rig)
	return err
}

// PolecatList returns all polecats in a rig
func (c *Client) PolecatList(ctx context.Context, rig string) ([]PolecatInfo, error) {
	var result []PolecatInfo
	if err := c.runJSON(ctx, &result, "polecat", "list", "--rig", rig); err != nil {
		return nil, err
	}
	return result, nil
}

// PolecatStatus returns detailed status for a specific polecat
func (c *Client) PolecatStatus(ctx context.Context, rig, name string) (*PolecatStatus, error) {
	var result PolecatStatus
	if err := c.runJSON(ctx, &result, "polecat", "status", fmt.Sprintf("%s/%s", rig, name)); err != nil {
		return nil, err
	}
	return &result, nil
}

// PolecatNuke removes a polecat
func (c *Client) PolecatNuke(ctx context.Context, rig, name string, force bool) error {
	args := []string{"polecat", "nuke", fmt.Sprintf("%s/%s", rig, name)}
	if force {
		args = append(args, "--force")
	}
	_, err := c.run(ctx, args...)
	return err
}

// PolecatReset resets a polecat to idle state
func (c *Client) PolecatReset(ctx context.Context, rig, name string) error {
	_, err := c.run(ctx, "polecat", "reset", fmt.Sprintf("%s/%s", rig, name))
	return err
}

// PolecatExists checks if a polecat exists
func (c *Client) PolecatExists(ctx context.Context, rig, name string) (bool, error) {
	polecats, err := c.PolecatList(ctx, rig)
	if err != nil {
		return false, err
	}
	for _, p := range polecats {
		if p.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// --- Convoy Operations ---

// ConvoyCreate creates a new convoy for tracking beads
func (c *Client) ConvoyCreate(ctx context.Context, description string, beadIDs []string) (string, error) {
	args := []string{"convoy", "create", description}
	args = append(args, beadIDs...)
	output, err := c.run(ctx, args...)
	if err != nil {
		return "", err
	}
	// Parse convoy ID from output
	return strings.TrimSpace(string(output)), nil
}

// ConvoyStatus returns detailed status for a convoy
func (c *Client) ConvoyStatus(ctx context.Context, id string) (*ConvoyStatus, error) {
	var result ConvoyStatus
	if err := c.runJSON(ctx, &result, "convoy", "status", id); err != nil {
		return nil, err
	}
	return &result, nil
}

// ConvoyList returns all convoys
func (c *Client) ConvoyList(ctx context.Context) ([]ConvoyInfo, error) {
	var result []ConvoyInfo
	if err := c.runJSON(ctx, &result, "convoy", "list"); err != nil {
		return nil, err
	}
	return result, nil
}

// --- Hook Operations ---

// Hook assigns a bead to an assignee
func (c *Client) Hook(ctx context.Context, beadID, assignee string) error {
	_, err := c.run(ctx, "hook", beadID, assignee)
	return err
}

// HookStatus returns what's hooked to an assignee
func (c *Client) HookStatus(ctx context.Context, assignee string) (*HookInfo, error) {
	var result HookInfo
	if err := c.runJSON(ctx, &result, "hook", "--status", assignee); err != nil {
		return nil, err
	}
	return &result, nil
}

// --- Beads Operations ---

// BeadStatus returns the status of a bead
func (c *Client) BeadStatus(ctx context.Context, beadID string) (*BeadStatus, error) {
	var result BeadStatus
	if err := c.runJSON(ctx, &result, "bd", "show", beadID); err != nil {
		return nil, err
	}
	return &result, nil
}

// --- Mail Operations ---

// MailSend sends a mail message
func (c *Client) MailSend(ctx context.Context, address, subject, message string) error {
	_, err := c.run(ctx, "mail", "send", address, "-s", subject, "-m", message)
	return err
}
