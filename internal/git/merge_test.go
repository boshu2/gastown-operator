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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMergeBranch_Integration tests the full MergeBranch workflow with real git repos.
// This simulates the refinery merging a polecat's feature branch into main.
func TestMergeBranch_Integration(t *testing.T) {
	skipIfNoGit(t)

	ctx := context.Background()

	// Create a temp directory structure
	tempDir, err := os.MkdirTemp("", "merge-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create "origin" bare repo (simulates remote like GitHub)
	originDir := filepath.Join(tempDir, "origin.git")
	require.NoError(t, runGitCmd(t, "", "init", "--bare", originDir))

	// Create "refinery" working repo (where refinery processes merges)
	refineryDir := filepath.Join(tempDir, "refinery-repo")
	require.NoError(t, runGitCmd(t, "", "clone", originDir, refineryDir))

	// Configure git user in refinery repo
	require.NoError(t, runGitCmd(t, refineryDir, "config", "user.email", "test@test.com"))
	require.NoError(t, runGitCmd(t, refineryDir, "config", "user.name", "Test User"))

	// Create initial commit on main
	initialFile := filepath.Join(refineryDir, "README.md")
	require.NoError(t, os.WriteFile(initialFile, []byte("# Test Repo\n"), 0o600))
	require.NoError(t, runGitCmd(t, refineryDir, "add", "README.md"))
	require.NoError(t, runGitCmd(t, refineryDir, "commit", "-m", "Initial commit"))
	require.NoError(t, runGitCmd(t, refineryDir, "push", "-u", "origin", "main"))

	// Create "polecat" working repo (simulates a polecat's worktree)
	polecatDir := filepath.Join(tempDir, "polecat-repo")
	require.NoError(t, runGitCmd(t, "", "clone", originDir, polecatDir))

	// Configure git user in polecat repo
	require.NoError(t, runGitCmd(t, polecatDir, "config", "user.email", "polecat@test.com"))
	require.NoError(t, runGitCmd(t, polecatDir, "config", "user.name", "Polecat Worker"))

	// Create feature branch with polecat's work
	require.NoError(t, runGitCmd(t, polecatDir, "checkout", "-b", "feature/bead-1234"))

	// Add feature work (simulates what polecat did)
	featureFile := filepath.Join(polecatDir, "feature.go")
	require.NoError(t, os.WriteFile(featureFile, []byte("package feature\n\nfunc Hello() string { return \"world\" }\n"), 0o600))
	require.NoError(t, runGitCmd(t, polecatDir, "add", "feature.go"))
	require.NoError(t, runGitCmd(t, polecatDir, "commit", "-m", "feat: add hello function"))

	// Push feature branch to origin
	require.NoError(t, runGitCmd(t, polecatDir, "push", "-u", "origin", "feature/bead-1234"))

	// Now test the merge from refinery's perspective
	client := NewClient(refineryDir, originDir)

	t.Run("successful merge without test command", func(t *testing.T) {
		result, err := client.MergeBranch(ctx, MergeOptions{
			SourceBranch:       "feature/bead-1234",
			TargetBranch:       "main",
			DeleteSourceBranch: false, // Keep branch for inspection
		})

		require.NoError(t, err)
		assert.True(t, result.Success, "merge should succeed")
		assert.NotEmpty(t, result.MergedCommit, "should have merged commit SHA")
		assert.Empty(t, result.Error, "should have no error")

		// Verify the feature file exists on main
		_, err = os.Stat(filepath.Join(refineryDir, "feature.go"))
		assert.NoError(t, err, "feature.go should exist on main after merge")

		// Verify we're on main
		branch, err := client.CurrentBranch(ctx)
		require.NoError(t, err)
		assert.Equal(t, "main", branch)
	})

	t.Run("successful merge with test command", func(t *testing.T) {
		// Add another commit to feature branch
		require.NoError(t, runGitCmd(t, polecatDir, "checkout", "feature/bead-1234"))

		anotherFile := filepath.Join(polecatDir, "another.txt")
		require.NoError(t, os.WriteFile(anotherFile, []byte("another file\n"), 0o600))
		require.NoError(t, runGitCmd(t, polecatDir, "add", "another.txt"))
		require.NoError(t, runGitCmd(t, polecatDir, "commit", "-m", "feat: add another file"))
		require.NoError(t, runGitCmd(t, polecatDir, "push", "origin", "feature/bead-1234"))

		// Create a Makefile with a simple test target
		makefilePath := filepath.Join(refineryDir, "Makefile")
		require.NoError(t, os.WriteFile(makefilePath, []byte("test:\n\t@echo 'Tests passed'\n"), 0o600))
		require.NoError(t, runGitCmd(t, refineryDir, "add", "Makefile"))
		require.NoError(t, runGitCmd(t, refineryDir, "commit", "-m", "Add Makefile for tests"))
		require.NoError(t, runGitCmd(t, refineryDir, "push", "origin", "main"))

		// Merge with a test command that passes
		result, err := client.MergeBranch(ctx, MergeOptions{
			SourceBranch: "feature/bead-1234",
			TargetBranch: "main",
			TestCommand:  "make test",
		})

		require.NoError(t, err)
		assert.True(t, result.Success, "merge should succeed with passing tests")
	})
}

// TestMergeBranch_ConflictHandling tests merge failure scenarios.
func TestMergeBranch_ConflictHandling(t *testing.T) {
	skipIfNoGit(t)

	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "merge-conflict-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Setup origin
	originDir := filepath.Join(tempDir, "origin.git")
	require.NoError(t, runGitCmd(t, "", "init", "--bare", originDir))

	// Setup refinery
	refineryDir := filepath.Join(tempDir, "refinery-repo")
	require.NoError(t, runGitCmd(t, "", "clone", originDir, refineryDir))
	require.NoError(t, runGitCmd(t, refineryDir, "config", "user.email", "test@test.com"))
	require.NoError(t, runGitCmd(t, refineryDir, "config", "user.name", "Test User"))

	// Initial commit
	testFile := filepath.Join(refineryDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("line1\n"), 0o600))
	require.NoError(t, runGitCmd(t, refineryDir, "add", "test.txt"))
	require.NoError(t, runGitCmd(t, refineryDir, "commit", "-m", "Initial"))
	require.NoError(t, runGitCmd(t, refineryDir, "push", "-u", "origin", "main"))

	// Create feature branch
	require.NoError(t, runGitCmd(t, refineryDir, "checkout", "-b", "feature/conflict"))
	require.NoError(t, os.WriteFile(testFile, []byte("feature change\n"), 0o600))
	require.NoError(t, runGitCmd(t, refineryDir, "add", "test.txt"))
	require.NoError(t, runGitCmd(t, refineryDir, "commit", "-m", "Feature change"))
	require.NoError(t, runGitCmd(t, refineryDir, "push", "-u", "origin", "feature/conflict"))

	// Make conflicting change on main
	require.NoError(t, runGitCmd(t, refineryDir, "checkout", "main"))
	require.NoError(t, os.WriteFile(testFile, []byte("main change\n"), 0o600))
	require.NoError(t, runGitCmd(t, refineryDir, "add", "test.txt"))
	require.NoError(t, runGitCmd(t, refineryDir, "commit", "-m", "Main change"))
	require.NoError(t, runGitCmd(t, refineryDir, "push", "origin", "main"))

	client := NewClient(refineryDir, originDir)

	t.Run("handles rebase conflict gracefully", func(t *testing.T) {
		result, err := client.MergeBranch(ctx, MergeOptions{
			SourceBranch: "feature/conflict",
			TargetBranch: "main",
		})

		// Should fail with rebase error
		require.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "rebase failed")

		// Verify repo is in clean state (rebase aborted)
		clean, err := client.IsClean(ctx)
		require.NoError(t, err)
		assert.True(t, clean, "repo should be clean after failed merge (rebase aborted)")
	})
}

// TestMergeBranch_NonExistentBranch tests handling of missing branches.
func TestMergeBranch_NonExistentBranch(t *testing.T) {
	skipIfNoGit(t)

	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "merge-noexist-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Setup origin
	originDir := filepath.Join(tempDir, "origin.git")
	require.NoError(t, runGitCmd(t, "", "init", "--bare", originDir))

	// Setup repo
	repoDir := filepath.Join(tempDir, "repo")
	require.NoError(t, runGitCmd(t, "", "clone", originDir, repoDir))
	require.NoError(t, runGitCmd(t, repoDir, "config", "user.email", "test@test.com"))
	require.NoError(t, runGitCmd(t, repoDir, "config", "user.name", "Test User"))

	// Initial commit
	testFile := filepath.Join(repoDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test\n"), 0o600))
	require.NoError(t, runGitCmd(t, repoDir, "add", "test.txt"))
	require.NoError(t, runGitCmd(t, repoDir, "commit", "-m", "Initial"))
	require.NoError(t, runGitCmd(t, repoDir, "push", "-u", "origin", "main"))

	client := NewClient(repoDir, originDir)

	t.Run("fails gracefully for non-existent source branch", func(t *testing.T) {
		result, err := client.MergeBranch(ctx, MergeOptions{
			SourceBranch: "feature/does-not-exist",
			TargetBranch: "main",
		})

		require.Error(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "checkout source branch failed")
	})
}

// TestMergeBranch_WithBranchDeletion tests branch cleanup after merge.
func TestMergeBranch_WithBranchDeletion(t *testing.T) {
	skipIfNoGit(t)

	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "merge-delete-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Setup origin
	originDir := filepath.Join(tempDir, "origin.git")
	require.NoError(t, runGitCmd(t, "", "init", "--bare", originDir))

	// Setup repo
	repoDir := filepath.Join(tempDir, "repo")
	require.NoError(t, runGitCmd(t, "", "clone", originDir, repoDir))
	require.NoError(t, runGitCmd(t, repoDir, "config", "user.email", "test@test.com"))
	require.NoError(t, runGitCmd(t, repoDir, "config", "user.name", "Test User"))

	// Initial commit
	testFile := filepath.Join(repoDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test\n"), 0o600))
	require.NoError(t, runGitCmd(t, repoDir, "add", "test.txt"))
	require.NoError(t, runGitCmd(t, repoDir, "commit", "-m", "Initial"))
	require.NoError(t, runGitCmd(t, repoDir, "push", "-u", "origin", "main"))

	// Create and push feature branch
	require.NoError(t, runGitCmd(t, repoDir, "checkout", "-b", "feature/to-delete"))
	testFile2 := filepath.Join(repoDir, "feature.txt")
	require.NoError(t, os.WriteFile(testFile2, []byte("feature\n"), 0o600))
	require.NoError(t, runGitCmd(t, repoDir, "add", "feature.txt"))
	require.NoError(t, runGitCmd(t, repoDir, "commit", "-m", "Feature"))
	require.NoError(t, runGitCmd(t, repoDir, "push", "-u", "origin", "feature/to-delete"))
	require.NoError(t, runGitCmd(t, repoDir, "checkout", "main"))

	client := NewClient(repoDir, originDir)

	t.Run("deletes source branch after successful merge", func(t *testing.T) {
		result, err := client.MergeBranch(ctx, MergeOptions{
			SourceBranch:       "feature/to-delete",
			TargetBranch:       "main",
			DeleteSourceBranch: true,
		})

		require.NoError(t, err)
		assert.True(t, result.Success)

		// Verify local branch is deleted
		exists, err := client.BranchExists(ctx, "feature/to-delete")
		require.NoError(t, err)
		assert.False(t, exists, "local branch should be deleted")

		// Fetch to update remote refs
		require.NoError(t, client.Fetch(ctx))

		// Verify remote branch is deleted
		_, err = runGitCmdOutput(t, repoDir, "rev-parse", "--verify", "origin/feature/to-delete")
		assert.Error(t, err, "remote branch should be deleted")
	})
}

// runGitCmd is a test helper to run git commands.
func runGitCmd(t *testing.T, dir string, args ...string) error {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("git %v failed:\n%s", args, string(output))
	}
	return err
}

// runGitCmdOutput runs a git command and returns its output.
func runGitCmdOutput(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	return string(output), err
}
