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

// Package git provides helpers for running git operations in the operator.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Client provides git operations for a repository.
type Client struct {
	// RepoDir is the path to the git repository
	RepoDir string

	// SSHKeyPath is the path to the SSH key for authentication (optional)
	SSHKeyPath string

	// GitURL is the remote repository URL
	GitURL string
}

// NewClient creates a new git client for the given repository directory.
func NewClient(repoDir, gitURL string) *Client {
	return &Client{
		RepoDir: repoDir,
		GitURL:  gitURL,
	}
}

// WithSSHKey sets the SSH key path for authentication.
func (c *Client) WithSSHKey(keyPath string) *Client {
	c.SSHKeyPath = keyPath
	return c
}

// runGit executes a git command in the repository directory.
func (c *Client) runGit(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = c.RepoDir

	// Set up SSH authentication if key is provided
	if c.SSHKeyPath != "" {
		sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
			c.SSHKeyPath)
		cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCmd)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w\nstderr: %s",
			strings.Join(args, " "), err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// Clone clones a repository to the client's RepoDir.
func (c *Client) Clone(ctx context.Context) error {
	// Ensure parent directory exists
	parentDir := filepath.Dir(c.RepoDir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", "clone", c.GitURL, c.RepoDir)

	// Set up SSH authentication if key is provided
	if c.SSHKeyPath != "" {
		sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
			c.SSHKeyPath)
		cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCmd)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w\nstderr: %s", err, stderr.String())
	}

	return nil
}

// Fetch fetches updates from the remote.
func (c *Client) Fetch(ctx context.Context) error {
	_, err := c.runGit(ctx, "fetch", "--all", "--prune")
	return err
}

// Checkout checks out a branch or commit.
func (c *Client) Checkout(ctx context.Context, ref string) error {
	_, err := c.runGit(ctx, "checkout", ref)
	return err
}

// Pull pulls changes from the remote.
func (c *Client) Pull(ctx context.Context) error {
	_, err := c.runGit(ctx, "pull", "--ff-only")
	return err
}

// RebaseOnto rebases the current branch onto another branch.
func (c *Client) RebaseOnto(ctx context.Context, ontoBranch string) error {
	_, err := c.runGit(ctx, "rebase", ontoBranch)
	return err
}

// Push pushes the current branch to the remote.
func (c *Client) Push(ctx context.Context) error {
	_, err := c.runGit(ctx, "push")
	return err
}

// PushForce force pushes the current branch to the remote.
func (c *Client) PushForce(ctx context.Context) error {
	_, err := c.runGit(ctx, "push", "--force-with-lease")
	return err
}

// Merge merges a branch into the current branch (fast-forward if possible).
func (c *Client) Merge(ctx context.Context, branch string) error {
	_, err := c.runGit(ctx, "merge", "--ff-only", branch)
	return err
}

// MergeNoFF merges a branch with a merge commit.
func (c *Client) MergeNoFF(ctx context.Context, branch, message string) error {
	_, err := c.runGit(ctx, "merge", "--no-ff", "-m", message, branch)
	return err
}

// DeleteRemoteBranch deletes a branch on the remote.
func (c *Client) DeleteRemoteBranch(ctx context.Context, branch string) error {
	_, err := c.runGit(ctx, "push", "origin", "--delete", branch)
	return err
}

// DeleteLocalBranch deletes a local branch.
func (c *Client) DeleteLocalBranch(ctx context.Context, branch string) error {
	_, err := c.runGit(ctx, "branch", "-D", branch)
	return err
}

// CurrentBranch returns the name of the current branch.
func (c *Client) CurrentBranch(ctx context.Context) (string, error) {
	return c.runGit(ctx, "rev-parse", "--abbrev-ref", "HEAD")
}

// BranchExists checks if a branch exists locally or remotely.
func (c *Client) BranchExists(ctx context.Context, branch string) (bool, error) {
	// Check local
	_, err := c.runGit(ctx, "rev-parse", "--verify", branch)
	if err == nil {
		return true, nil
	}

	// Check remote
	_, err = c.runGit(ctx, "rev-parse", "--verify", "origin/"+branch)
	return err == nil, nil
}

// GetCommitSHA returns the SHA of the current HEAD.
func (c *Client) GetCommitSHA(ctx context.Context) (string, error) {
	return c.runGit(ctx, "rev-parse", "HEAD")
}

// IsClean returns true if the working directory has no uncommitted changes.
func (c *Client) IsClean(ctx context.Context) (bool, error) {
	output, err := c.runGit(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return output == "", nil
}

// AbortRebase aborts an in-progress rebase.
func (c *Client) AbortRebase(ctx context.Context) error {
	_, err := c.runGit(ctx, "rebase", "--abort")
	return err
}

// ResetHard resets the working directory to the given ref.
func (c *Client) ResetHard(ctx context.Context, ref string) error {
	_, err := c.runGit(ctx, "reset", "--hard", ref)
	return err
}
