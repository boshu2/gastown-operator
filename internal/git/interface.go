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

import "context"

// GitClient defines the interface for git operations needed by the refinery.
type GitClient interface {
	// Clone clones the repository.
	Clone(ctx context.Context) error

	// MergeBranch performs the full merge workflow.
	MergeBranch(ctx context.Context, opts MergeOptions) (*MergeResult, error)
}

// GitClientFactory creates git clients for merge operations.
type GitClientFactory func(repoDir, gitURL, sshKeyPath string) GitClient

// DefaultGitClientFactory creates real git clients.
func DefaultGitClientFactory(repoDir, gitURL, sshKeyPath string) GitClient {
	client := NewClient(repoDir, gitURL)
	if sshKeyPath != "" {
		client = client.WithSSHKey(sshKeyPath)
	}
	return client
}
