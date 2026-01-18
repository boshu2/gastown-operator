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

package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// skipIfNoGit skips the test if git is not available
func skipIfNoGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping test")
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient("/tmp/repo", "https://github.com/example/repo.git")

	if client.RepoDir != "/tmp/repo" {
		t.Errorf("RepoDir = %s, want /tmp/repo", client.RepoDir)
	}

	if client.GitURL != "https://github.com/example/repo.git" {
		t.Errorf("GitURL = %s, want https://github.com/example/repo.git", client.GitURL)
	}
}

func TestWithSSHKey(t *testing.T) {
	client := NewClient("/tmp/repo", "git@github.com:example/repo.git")
	client = client.WithSSHKey("/path/to/key")

	if client.SSHKeyPath != "/path/to/key" {
		t.Errorf("SSHKeyPath = %s, want /path/to/key", client.SSHKeyPath)
	}
}

func TestClone(t *testing.T) {
	skipIfNoGit(t)

	// Create a temp "origin" repo
	originDir, err := os.MkdirTemp("", "git-test-origin-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(originDir) }()

	// Initialize the origin repo
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "init", "--bare")
	cmd.Dir = originDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init origin repo: %v", err)
	}

	// Create a clone directory
	cloneDir, err := os.MkdirTemp("", "git-test-clone-*")
	if err != nil {
		t.Fatalf("Failed to create clone dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(cloneDir) }()

	repoPath := filepath.Join(cloneDir, "repo")

	// Clone it
	client := NewClient(repoPath, originDir)
	if err := client.Clone(ctx); err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	// Verify the clone exists
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
		t.Errorf("Clone directory doesn't have .git: %v", err)
	}
}

func TestIsClean(t *testing.T) {
	skipIfNoGit(t)

	// Create a temp repo
	repoDir, err := os.MkdirTemp("", "git-test-clean-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(repoDir) }()

	ctx := context.Background()

	// Initialize repo
	cmd := exec.CommandContext(ctx, "git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	// Configure git user for commits
	cmd = exec.CommandContext(ctx, "git", "config", "user.email", "test@test.com")
	cmd.Dir = repoDir
	_ = cmd.Run()

	cmd = exec.CommandContext(ctx, "git", "config", "user.name", "Test User")
	cmd.Dir = repoDir
	_ = cmd.Run()

	client := NewClient(repoDir, "")

	// Empty repo should be clean
	clean, err := client.IsClean(ctx)
	if err != nil {
		t.Fatalf("IsClean failed: %v", err)
	}
	if !clean {
		t.Error("Expected empty repo to be clean")
	}

	// Add a file
	testFile := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Now should not be clean
	clean, err = client.IsClean(ctx)
	if err != nil {
		t.Fatalf("IsClean failed: %v", err)
	}
	if clean {
		t.Error("Expected repo with untracked file to not be clean")
	}
}
