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
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// MergeOptions configures the merge workflow.
type MergeOptions struct {
	// SourceBranch is the branch to merge (e.g., feature/ap-1234)
	SourceBranch string

	// TargetBranch is the branch to merge into (e.g., main)
	TargetBranch string

	// TestCommand is an optional command to run after rebase to validate
	TestCommand string

	// DeleteSourceBranch deletes the source branch after successful merge
	DeleteSourceBranch bool
}

// MergeResult contains the result of a merge operation.
type MergeResult struct {
	// Success indicates if the merge completed successfully
	Success bool

	// MergedCommit is the SHA of the merged commit on target branch
	MergedCommit string

	// Error contains the error message if merge failed
	Error string
}

// MergeBranch performs the full merge workflow:
// 1. Fetch latest
// 2. Checkout target branch
// 3. Pull target to ensure up-to-date
// 4. Checkout source branch
// 5. Rebase onto target
// 6. Run tests if configured
// 7. Checkout target and merge (fast-forward)
// 8. Push target
// 9. Delete source branch if configured
func (c *Client) MergeBranch(ctx context.Context, opts MergeOptions) (*MergeResult, error) {
	result := &MergeResult{}

	// Step 1: Fetch latest
	if err := c.Fetch(ctx); err != nil {
		result.Error = fmt.Sprintf("fetch failed: %v", err)
		return result, err
	}

	// Step 2: Checkout target branch
	if err := c.Checkout(ctx, opts.TargetBranch); err != nil {
		result.Error = fmt.Sprintf("checkout target failed: %v", err)
		return result, err
	}

	// Step 3: Pull target to ensure up-to-date
	if err := c.Pull(ctx); err != nil {
		result.Error = fmt.Sprintf("pull target failed: %v", err)
		return result, err
	}

	// Step 4: Checkout source branch
	if err := c.Checkout(ctx, opts.SourceBranch); err != nil {
		// Try remote branch
		if err := c.Checkout(ctx, "origin/"+opts.SourceBranch); err != nil {
			result.Error = fmt.Sprintf("checkout source branch failed: %v", err)
			return result, err
		}
		// Create local tracking branch
		if _, err := c.runGit(ctx, "checkout", "-b", opts.SourceBranch); err != nil {
			result.Error = fmt.Sprintf("create tracking branch failed: %v", err)
			return result, err
		}
	}

	// Step 5: Rebase onto target
	if err := c.RebaseOnto(ctx, opts.TargetBranch); err != nil {
		// Abort the rebase if it failed
		_ = c.AbortRebase(ctx)
		result.Error = fmt.Sprintf("rebase failed: %v", err)
		return result, err
	}

	// Step 6: Run tests if configured
	if opts.TestCommand != "" {
		if err := c.runTests(ctx, opts.TestCommand); err != nil {
			result.Error = fmt.Sprintf("tests failed: %v", err)
			return result, err
		}
	}

	// Step 7: Checkout target and merge (fast-forward)
	if err := c.Checkout(ctx, opts.TargetBranch); err != nil {
		result.Error = fmt.Sprintf("checkout target for merge failed: %v", err)
		return result, err
	}

	if err := c.Merge(ctx, opts.SourceBranch); err != nil {
		result.Error = fmt.Sprintf("merge failed: %v", err)
		return result, err
	}

	// Step 8: Push target
	if err := c.Push(ctx); err != nil {
		result.Error = fmt.Sprintf("push failed: %v", err)
		return result, err
	}

	// Get the merged commit SHA
	sha, err := c.GetCommitSHA(ctx)
	if err == nil {
		result.MergedCommit = sha
	}

	// Step 9: Delete source branch if configured
	if opts.DeleteSourceBranch {
		// Delete remote first
		if err := c.DeleteRemoteBranch(ctx, opts.SourceBranch); err != nil {
			// Log but don't fail - the merge succeeded
			result.Error = fmt.Sprintf("warning: failed to delete remote branch: %v", err)
		}

		// Delete local
		if err := c.DeleteLocalBranch(ctx, opts.SourceBranch); err != nil {
			// Log but don't fail
			if result.Error == "" {
				result.Error = fmt.Sprintf("warning: failed to delete local branch: %v", err)
			}
		}
	}

	result.Success = true
	return result, nil
}

// runTests executes the test command in the repository.
func (c *Client) runTests(ctx context.Context, command string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = c.RepoDir
	cmd.Env = os.Environ()

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("test command failed: %w\nstderr: %s", err, stderr.String())
	}

	return nil
}
