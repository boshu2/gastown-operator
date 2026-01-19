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

package pod

import (
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
)

const (
	// Container names
	GitInitContainerName = "git-init"
	ClaudeContainerName  = "claude"

	// Volume names
	WorkspaceVolumeName   = "workspace"
	GitCredsVolumeName    = "git-creds"
	ClaudeCredsVolumeName = "claude-creds"
	TmpVolumeName         = "tmp"
	HomeVolumeName        = "home"

	// Mount paths
	WorkspaceMountPath   = "/workspace"
	GitCredsMountPath    = "/git-creds"
	ClaudeCredsMountPath = "/claude-creds" // Temporary mount for credentials (copied to $HOME/.claude at startup)
	TmpMountPath         = "/tmp"
	HomeMountPath        = "/home/nonroot"

	// Environment variable names for image configuration
	EnvGitImage    = "GASTOWN_GIT_IMAGE"
	EnvClaudeImage = "GASTOWN_CLAUDE_IMAGE"

	// Default images (community edition - vanilla Kubernetes)
	// For enterprise/FIPS, set env vars to UBI images:
	//   GASTOWN_GIT_IMAGE=registry.access.redhat.com/ubi9/ubi-minimal:9.3
	//   GASTOWN_CLAUDE_IMAGE=registry.access.redhat.com/ubi9/nodejs-20:1
	DefaultGitImage    = "alpine/git:2.43.0"
	DefaultClaudeImage = "node:20-slim"

	// Default resource values
	DefaultCPURequest    = "500m"
	DefaultCPULimit      = "2"
	DefaultMemoryRequest = "1Gi"
	DefaultMemoryLimit   = "4Gi"
)

// Builder constructs Pods for Polecat kubernetes execution
type Builder struct {
	polecat *gastownv1alpha1.Polecat
}

// NewBuilder creates a new Pod builder for the given Polecat
func NewBuilder(polecat *gastownv1alpha1.Polecat) *Builder {
	return &Builder{polecat: polecat}
}

// GetGitImage returns the git image to use, checking environment variable first
func GetGitImage() string {
	if img := os.Getenv(EnvGitImage); img != "" {
		return img
	}
	return DefaultGitImage
}

// GetClaudeImage returns the Claude image to use, checking environment variable first
func GetClaudeImage() string {
	if img := os.Getenv(EnvClaudeImage); img != "" {
		return img
	}
	return DefaultClaudeImage
}

// Build constructs the complete Pod spec for the Polecat
func (b *Builder) Build() (*corev1.Pod, error) {
	if b.polecat.Spec.Kubernetes == nil {
		return nil, fmt.Errorf("kubernetes spec is required for kubernetes execution mode")
	}

	k8sSpec := b.polecat.Spec.Kubernetes
	podName := fmt.Sprintf("polecat-%s", b.polecat.Name)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: b.polecat.Namespace,
			Labels: map[string]string{
				"gastown.io/polecat": b.polecat.Name,
				"gastown.io/rig":     b.polecat.Spec.Rig,
				"gastown.io/bead":    b.polecat.Spec.BeadID,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy:         corev1.RestartPolicyNever,
			ActiveDeadlineSeconds: k8sSpec.ActiveDeadlineSeconds,
			SecurityContext:       b.buildPodSecurityContext(),
			InitContainers: []corev1.Container{
				b.buildGitInitContainer(),
			},
			Containers: []corev1.Container{
				b.buildClaudeContainer(),
			},
			Volumes: b.buildVolumes(),
		},
	}

	return pod, nil
}

// buildGitInitContainer creates the git init container spec
func (b *Builder) buildGitInitContainer() corev1.Container {
	k8sSpec := b.polecat.Spec.Kubernetes

	// Determine work branch name
	workBranch := k8sSpec.WorkBranch
	if workBranch == "" {
		workBranch = fmt.Sprintf("feature/%s", b.polecat.Spec.BeadID)
	}

	gitScript := fmt.Sprintf(`
set -e

# Setup SSH
mkdir -p ~/.ssh
cp %s/ssh-privatekey ~/.ssh/id_rsa 2>/dev/null || cp %s/id_rsa ~/.ssh/id_rsa
chmod 600 ~/.ssh/id_rsa

# Add known hosts (GitHub, GitLab, common hosts)
ssh-keyscan github.com gitlab.com bitbucket.org >> ~/.ssh/known_hosts 2>/dev/null || true

# Extract hostname from git URL and add to known_hosts
HOSTNAME=$(echo "%s" | sed -E 's/.*@([^:\/]+).*/\1/' | sed -E 's/.*\/\/([^\/]+).*/\1/')
if [ -n "$HOSTNAME" ]; then
    ssh-keyscan "$HOSTNAME" >> ~/.ssh/known_hosts 2>/dev/null || true
fi

# Clone the repository
echo "Cloning %s branch %s..."
git clone --depth=1 -b %s %s %s/repo

# Create work branch
cd %s/repo
git checkout -b %s
echo "Git setup complete. Working branch: %s"
`,
		GitCredsMountPath, GitCredsMountPath,
		k8sSpec.GitRepository,
		k8sSpec.GitRepository, k8sSpec.GitBranch,
		k8sSpec.GitBranch, k8sSpec.GitRepository, WorkspaceMountPath,
		WorkspaceMountPath, workBranch, workBranch,
	)

	return corev1.Container{
		Name:            GitInitContainerName,
		Image:           GetGitImage(),
		Command:         []string{"/bin/sh", "-c"},
		Args:            []string{gitScript},
		SecurityContext: b.buildSecurityContext(),
		Env: []corev1.EnvVar{
			{
				Name:  "HOME",
				Value: HomeMountPath,
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      WorkspaceVolumeName,
				MountPath: WorkspaceMountPath,
			},
			{
				Name:      GitCredsVolumeName,
				MountPath: GitCredsMountPath,
				ReadOnly:  true,
			},
			{
				Name:      TmpVolumeName,
				MountPath: TmpMountPath,
			},
			{
				Name:      HomeVolumeName,
				MountPath: HomeMountPath,
			},
		},
	}
}

// buildClaudeContainer creates the Claude agent container spec
func (b *Builder) buildClaudeContainer() corev1.Container {
	k8sSpec := b.polecat.Spec.Kubernetes

	// Use custom image if specified, otherwise use configured default
	image := GetClaudeImage()
	if k8sSpec.Image != "" {
		image = k8sSpec.Image
	}

	// Build the agent startup script
	agentScript := fmt.Sprintf(`
set -e

# Configure npm for non-root global installs
export NPM_CONFIG_PREFIX="$HOME/.npm-global"
export PATH="$HOME/.npm-global/bin:$PATH"
mkdir -p "$HOME/.npm-global"

# Copy Claude credentials from read-only mount to writable HOME
mkdir -p "$HOME/.claude"
if [ -f "%s/.credentials.json" ]; then
    cp "%s/.credentials.json" "$HOME/.claude/.credentials.json"
    echo "Claude credentials copied to $HOME/.claude/"
fi

# Install Claude Code CLI
echo "Installing Claude Code CLI..."
npm install -g @anthropic-ai/claude-code

# Verify installation
claude --version || echo "Claude CLI installed"

# Run Claude with dangerously-skip-permissions for headless mode
echo "Starting Claude Code agent..."
echo "Working on issue: $GT_ISSUE"
PROMPT="You are a Gas Town polecat worker assigned to issue $GT_ISSUE. "
PROMPT+="Read the repository CLAUDE.md and .beads/ directory to understand the task. "
PROMPT+="Execute the work autonomously."
exec claude --print --dangerously-skip-permissions "$PROMPT"
`, ClaudeCredsMountPath, ClaudeCredsMountPath)

	container := corev1.Container{
		Name:            ClaudeContainerName,
		Image:           image,
		Command:         []string{"/bin/sh", "-c"},
		Args:            []string{agentScript},
		WorkingDir:      fmt.Sprintf("%s/repo", WorkspaceMountPath),
		SecurityContext: b.buildSecurityContext(),
		Env: []corev1.EnvVar{
			// Claude credentials mounted at $HOME/.claude (standard Linux location)
			// No CLAUDE_CONFIG_DIR needed - Claude uses default path
			{
				Name:  "GT_ISSUE",
				Value: b.polecat.Spec.BeadID,
			},
			{
				Name:  "GT_POLECAT",
				Value: b.polecat.Name,
			},
			{
				Name:  "GT_RIG",
				Value: b.polecat.Spec.Rig,
			},
			{
				Name:  "HOME",
				Value: HomeMountPath,
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      WorkspaceVolumeName,
				MountPath: WorkspaceMountPath,
			},
			{
				Name:      ClaudeCredsVolumeName,
				MountPath: ClaudeCredsMountPath,
				ReadOnly:  true,
			},
			{
				Name:      TmpVolumeName,
				MountPath: TmpMountPath,
			},
			{
				Name:      HomeVolumeName,
				MountPath: HomeMountPath,
			},
		},
		Resources: b.buildResources(),
	}

	return container
}

// buildVolumes creates the volume specifications
func (b *Builder) buildVolumes() []corev1.Volume {
	k8sSpec := b.polecat.Spec.Kubernetes

	return []corev1.Volume{
		{
			Name: WorkspaceVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: GitCredsVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  k8sSpec.GitSecretRef.Name,
					DefaultMode: int32Ptr(0400),
				},
			},
		},
		{
			Name: ClaudeCredsVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: k8sSpec.ClaudeCredsSecretRef.Name,
				},
			},
		},
		{
			Name: TmpVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: HomeVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}

// buildResources creates resource requirements
func (b *Builder) buildResources() corev1.ResourceRequirements {
	k8sSpec := b.polecat.Spec.Kubernetes

	// Use custom resources if specified
	if k8sSpec.Resources != nil {
		return *k8sSpec.Resources
	}

	// Default resources
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(DefaultCPURequest),
			corev1.ResourceMemory: resource.MustParse(DefaultMemoryRequest),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(DefaultCPULimit),
			corev1.ResourceMemory: resource.MustParse(DefaultMemoryLimit),
		},
	}
}

// int32Ptr returns a pointer to an int32
func int32Ptr(i int32) *int32 {
	return &i
}

// int64Ptr returns a pointer to an int64
func int64Ptr(i int64) *int64 {
	return &i
}

// boolPtr returns a pointer to a bool
func boolPtr(b bool) *bool {
	return &b
}

// buildPodSecurityContext returns a restricted Pod-level SecurityContext
// compliant with OpenShift's restricted SCC and Kubernetes Pod Security Standards
func (b *Builder) buildPodSecurityContext() *corev1.PodSecurityContext {
	return &corev1.PodSecurityContext{
		RunAsNonRoot: boolPtr(true),
		RunAsUser:    int64Ptr(65532),
		RunAsGroup:   int64Ptr(65532),
		FSGroup:      int64Ptr(65532),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}

// buildSecurityContext returns a restricted container-level SecurityContext
// compliant with OpenShift's restricted SCC and Kubernetes Pod Security Standards
func (b *Builder) buildSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		RunAsNonRoot:             boolPtr(true),
		RunAsUser:                int64Ptr(65532),
		RunAsGroup:               int64Ptr(65532),
		AllowPrivilegeEscalation: boolPtr(false),
		ReadOnlyRootFilesystem:   boolPtr(true),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}
