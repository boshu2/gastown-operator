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

	// Mount paths
	WorkspaceMountPath   = "/workspace"
	GitCredsMountPath    = "/git-creds"
	ClaudeCredsMountPath = "/claude-creds"

	// Default images
	DefaultGitImage    = "alpine/git:latest"
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
		Name:    GitInitContainerName,
		Image:   DefaultGitImage,
		Command: []string{"/bin/sh", "-c"},
		Args:    []string{gitScript},
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
		},
	}
}

// buildClaudeContainer creates the Claude agent container spec
func (b *Builder) buildClaudeContainer() corev1.Container {
	k8sSpec := b.polecat.Spec.Kubernetes

	// Use custom image if specified
	image := DefaultClaudeImage
	if k8sSpec.Image != "" {
		image = k8sSpec.Image
	}

	// Build the agent startup script
	agentScript := `
set -e

# Install Claude Code CLI
echo "Installing Claude Code CLI..."
npm install -g @anthropic-ai/claude-code

# Verify installation
claude --version || echo "Claude CLI installed"

# Run Claude with dangerously-skip-permissions for headless mode
echo "Starting Claude Code agent..."
echo "Working on issue: $GT_ISSUE"
exec claude --dangerously-skip-permissions
`

	container := corev1.Container{
		Name:       ClaudeContainerName,
		Image:      image,
		Command:    []string{"/bin/sh", "-c"},
		Args:       []string{agentScript},
		WorkingDir: fmt.Sprintf("%s/repo", WorkspaceMountPath),
		Env: []corev1.EnvVar{
			{
				Name:  "CLAUDE_CONFIG_DIR",
				Value: ClaudeCredsMountPath,
			},
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
