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
	GitInitContainerName   = "git-init"
	ClaudeContainerName    = "claude"
	TelemetryContainerName = "telemetry"

	// Volume names
	WorkspaceVolumeName     = "workspace"
	GitCredsVolumeName      = "git-creds"
	ClaudeCredsVolumeName   = "claude-creds"
	TmpVolumeName           = "tmp"
	HomeVolumeName          = "home"
	MetricsVolumeName       = "metrics"
	SSHKnownHostsVolumeName = "ssh-known-hosts"

	// Mount paths
	WorkspaceMountPath     = "/workspace"
	GitCredsMountPath      = "/git-creds"
	ClaudeCredsMountPath   = "/claude-creds" // Temporary mount for credentials (copied to $HOME/.claude at startup)
	TmpMountPath           = "/tmp"
	HomeMountPath          = "/home/nonroot"
	MetricsMountPath       = "/metrics"
	SSHKnownHostsMountPath = "/ssh-known-hosts"

	// Environment variable names for image configuration
	EnvGitImage       = "GASTOWN_GIT_IMAGE"
	EnvClaudeImage    = "GASTOWN_CLAUDE_IMAGE"
	EnvTelemetryImage = "GASTOWN_TELEMETRY_IMAGE"

	// Default images (community edition - vanilla Kubernetes)
	// Note: Git init uses polecat-agent because it has proper non-root user setup (UID 65532)
	// For enterprise/FIPS, set env vars to UBI images:
	//   GASTOWN_GIT_IMAGE=ghcr.io/boshu2/polecat-agent:0.4.0-fips
	//   GASTOWN_CLAUDE_IMAGE=ghcr.io/boshu2/polecat-agent:0.4.0-fips
	//   GASTOWN_TELEMETRY_IMAGE=registry.access.redhat.com/ubi9/ubi-minimal:9.3
	DefaultGitImage       = "ghcr.io/boshu2/polecat-agent:0.4.0"
	DefaultClaudeImage    = "ghcr.io/boshu2/polecat-agent:0.4.0"
	DefaultTelemetryImage = "alpine:latest"

	// Default resource values
	DefaultCPURequest    = "500m"
	DefaultCPULimit      = "2"
	DefaultMemoryRequest = "1Gi"
	DefaultMemoryLimit   = "4Gi"

	// Telemetry sidecar resource defaults
	TelemetryCPURequest    = "100m"
	TelemetryCPULimit      = "200m"
	TelemetryMemoryRequest = "128Mi"
	TelemetryMemoryLimit   = "256Mi"
)

// Pre-verified SSH host keys for common Git hosting providers.
// These keys were verified from official sources:
// - GitHub: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/githubs-ssh-key-fingerprints
// - GitLab: https://docs.gitlab.com/ee/user/gitlab_com/index.html#ssh-host-keys-fingerprints
// - Bitbucket: https://support.atlassian.com/bitbucket-cloud/docs/configure-ssh-and-two-step-verification/
//
// Last updated: 2026-01-20
// SECURITY: Using pre-verified keys prevents MITM attacks on first connection (TOFU vulnerability).
// For private Git servers, users should provide their own known_hosts via SSHKnownHostsConfigMapRef.
//
//nolint:lll // SSH public keys cannot be broken across lines; this is expected.
const PreVerifiedSSHKnownHosts = `# GitHub (verified 2026-01-20)
github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl
github.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wAdT/y6v0mKV0U2w0WZ2YB/++Tpockg=
github.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQowgcQnjshcLrqPEiiphnt+VTTvDP6mHBL9j1aNUkY4Ue1gvwnGLVlOhGeYrnZaMgRK6+PKCUXaDbC7qtbW8gIkhL7aGCsOr/C56SJMy/BCZfxd1nWzAOxSDPgVsmerOBYfNqltV9/hWCqBywINIR+5dIg6JTJ72pcEpEjcYgXkE2YEFXV1JHnsKgbLWNlhScqb2UmyRkQyytRLtL+38TGxkxCflmO+5Z8CSSNY7GidjMIZ7Q4zMjA2n1nGrlTDkzwDCsw+wqFPGQA179cnfGWOWRVruj16z6XyvxvjJwbz0wQZ75XK5tKSb7FNyeIEs4TT4jk+S4dhPeAUC5y+bDYirYgM4GC7uEnztnZyaVWQ7B381AK4Qdrwt51ZqExKbQpTUNn+EjqoTwvqNj4kqx5QUCI0ThS/YkOxJCXmPUWZbhjpCg56i+2aB6CmK2JGhn57K5mj0MNdBXA4/WnwH6XoPWJzK5Nyu2zB3nAZp+S5hpQs+p1vN1/wsjk=
# GitLab (verified 2026-01-20)
gitlab.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAfuCHKVTjquxvt6CM6tdG4SLp1Btn/nOeHHE5UOzRdf
gitlab.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBFSMqzJeV9rUzU4kWitGjeR4PWSa29SPqJ1fVkhtj3Hw9xjLVXVYrU9QlYWrOLXBpQ6KWjbjTDTdDkoohFzgbEY=
gitlab.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCsj2bNKTBSpIYDEGk9KxsGh3mySTRgMtXL583qmBpzeQ+jqCMRgBqB98u3z++J1sKlXHWfM9dyhSevkMwSbhoR8XIq/U0tCNyokEi/ueaBMCvbcTHhO7FcwzY92WK4 Voices0rWtH2Lxbvt9jW/rlyf+ClGSuOHDJALO9mz1ApbdM/V8Q3IUehzBAKy4qqvzT3+0dHAAePj1Ej5g+7G0SqUpjCi5DNbvZIBIlINmVbAmLKWNsE8bz0XE0n0zQbGNkmkKsP8pEPHe9XzHz+TfnhpKpLJ7NxrN3P+a/2yjZsLKMhiT+xwSLRwLQoKEE7X1JNPMi/1XPxQaP5cFlQ25+W
# Bitbucket (verified 2026-01-20)
bitbucket.org ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIazEu89wgQZ4bqs3d63QSMzYVa0MuJ2e2gKTKqu+UUO
bitbucket.org ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBPIQmuzMBuKdWeF4+a2sjSSpBK0iqitSQ+5BM9KhpexuGt20JpTVM7u5BDZngncgrqDMbWdxMWWOGtZ9UgbqgZE=
bitbucket.org ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDQeJzhupRu0u0cdegZIa8e86EG2qOCsIsD1Xw0xSeiPDlCr7kq97NLmMbpKTX6Esc30NuoqEEHCuc7yWtwp8dI76EEEB1VqY9QJq6vk+aySyboD5QF61I/1WeTwu+deCbgKMGbUijeXhtfbxSxm6JwGrXrhBdofTsbKRUsrN1WoNgUa8uqN1Vx6WAJw1JHPhglEGGHea6QICwJOAr/6mrui/oB7pkaWKHj3z7d1IC4KWLtY47elvjbaTlkN04Kc/5LFEirorGYVbt15kAUlqGM65pk6ZBxtaO3+30LVlORZkxOh+LKL/BvbZ/iRNhItLqNyieoQj/uj/4Lf0MagUQ/F7c+b6z1OawA/7FmbnyJlUH/1LPbM0lprrzj/qHqhOpK/xv/Kj7yM3TeqbAbN7zlWdLH/xj/cPk0O5EuCLquOmDwz0XHr3vdfl0Sgh8yoB+nlk6Q3X9DP/PbLyBHHEUi/bHy/TqBZxRWdPSCXMFBiqLsKK0xvb7fUY=
`

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

// GetTelemetryImage returns the telemetry sidecar image to use, checking environment variable first
func GetTelemetryImage() string {
	if img := os.Getenv(EnvTelemetryImage); img != "" {
		return img
	}
	return DefaultTelemetryImage
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
				b.buildTelemetrySidecar(),
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
		if b.polecat.Spec.BeadID != "" {
			workBranch = fmt.Sprintf("feature/%s", b.polecat.Spec.BeadID)
		} else {
			// Fallback to polecat name if no BeadID
			workBranch = fmt.Sprintf("polecat/%s", b.polecat.Name)
		}
	}

	// Determine SSH strict host key checking mode
	// Default to "yes" (most secure) if not specified
	strictHostKeyChecking := k8sSpec.SSHStrictHostKeyChecking
	if strictHostKeyChecking == "" {
		strictHostKeyChecking = "yes"
	}

	// Build the known_hosts setup script based on configuration
	var knownHostsSetup string
	if k8sSpec.SSHKnownHostsConfigMapRef != nil {
		// User-provided known_hosts via ConfigMap
		knownHostsSetup = fmt.Sprintf(`
# Using user-provided known_hosts from ConfigMap
if [ -f "%s/known_hosts" ]; then
    cp "%s/known_hosts" ~/.ssh/known_hosts
    chmod 644 ~/.ssh/known_hosts
    echo "Using custom known_hosts from ConfigMap"
else
    echo "ERROR: ConfigMap mounted but known_hosts key not found"
    exit 1
fi
`, SSHKnownHostsMountPath, SSHKnownHostsMountPath)
	} else {
		// Pre-verified known_hosts for common Git hosts
		// SECURITY: Using pre-verified keys prevents MITM attacks on first connection.
		// For private Git servers, users must provide host keys via SSHKnownHostsConfigMapRef.
		knownHostsSetup = fmt.Sprintf(`
# SECURITY: Pre-verified SSH host keys for common Git hosting providers.
# These keys are verified from official documentation to prevent MITM attacks.
# See: pkg/pod/builder.go PreVerifiedSSHKnownHosts constant for verification sources.
cat > ~/.ssh/known_hosts << 'KNOWN_HOSTS_EOF'
%s
KNOWN_HOSTS_EOF
chmod 644 ~/.ssh/known_hosts

# Check if the Git host is in known_hosts
HOSTNAME=$(echo "%s" | sed -E 's/.*@([^:\/]+).*/\1/' | sed -E 's/.*\/\/([^\/]+).*/\1/')
if [ -n "$HOSTNAME" ] && ! grep -q "^$HOSTNAME " ~/.ssh/known_hosts; then
    echo "ERROR: Host $HOSTNAME not in pre-verified known_hosts."
    echo "For private Git servers, use SSHKnownHostsConfigMapRef to provide verified host keys."
    echo "See: https://github.com/boshu2/gastown-operator/blob/main/docs/SECURITY.md"
    exit 1
fi
`, PreVerifiedSSHKnownHosts, k8sSpec.GitRepository)
	}

	gitScript := fmt.Sprintf(`
set -e

# Setup SSH
mkdir -p ~/.ssh
cp %s/ssh-privatekey ~/.ssh/id_rsa 2>/dev/null || cp %s/id_rsa ~/.ssh/id_rsa
chmod 600 ~/.ssh/id_rsa

# Configure SSH strict host key checking
echo "StrictHostKeyChecking %s" >> ~/.ssh/config
%s

# Clone the repository
echo "Cloning %s branch %s..."
git clone --depth=1 -b %s %s %s/repo

# Create work branch
cd %s/repo
git checkout -b %s
echo "Git setup complete. Working branch: %s"
`,
		GitCredsMountPath, GitCredsMountPath,
		strictHostKeyChecking,
		knownHostsSetup,
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
		VolumeMounts: b.buildGitInitVolumeMounts(),
	}
}

// buildGitInitVolumeMounts creates volume mounts for the git init container
func (b *Builder) buildGitInitVolumeMounts() []corev1.VolumeMount {
	k8sSpec := b.polecat.Spec.Kubernetes

	mounts := []corev1.VolumeMount{
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
	}

	// Add known_hosts ConfigMap mount if configured
	if k8sSpec.SSHKnownHostsConfigMapRef != nil {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      SSHKnownHostsVolumeName,
			MountPath: SSHKnownHostsMountPath,
			ReadOnly:  true,
		})
	}

	return mounts
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

# Configure SSH for git operations
mkdir -p "$HOME/.ssh"
if [ -f "%s/ssh-privatekey" ]; then
    cp "%s/ssh-privatekey" "$HOME/.ssh/id_rsa"
    chmod 600 "$HOME/.ssh/id_rsa"
    # Add GitHub to known hosts
    ssh-keyscan -t rsa github.com >> "$HOME/.ssh/known_hosts" 2>/dev/null || true
    echo "Git SSH key configured"
fi

# Configure git user for commits
git config --global user.name "Gas Town Polecat"
git config --global user.email "polecat@gastown.io"

# Verify Claude Code is available (pre-installed in polecat-agent image)
echo "Verifying Claude Code CLI..."
claude --version || { echo "ERROR: Claude CLI not found. Use ghcr.io/boshu2/polecat-agent image."; exit 1; }

# SECURITY: --dangerously-skip-permissions is required for headless operation.
# This grants elevated privileges to the Claude agent. Mitigations:
# - Pod runs as non-root with read-only root filesystem
# - Network policies should restrict outbound traffic
# - RBAC should limit polecat creation to trusted namespaces
# See docs/SECURITY.md for full threat model.
echo "Starting Claude Code agent..."
echo "Working on issue: $GT_ISSUE"

# Build the prompt with task description if available
if [ -n "$GT_TASK_DESCRIPTION" ]; then
    echo "=== Task Description ==="
    echo "$GT_TASK_DESCRIPTION"
    echo "========================"
    PROMPT="You are a Gas Town polecat worker. Your task:

ISSUE: $GT_ISSUE
TASK: $GT_TASK_DESCRIPTION

INSTRUCTIONS:
1. Implement the task described above
2. After completing the work:
   - git add the changed files
   - git commit -m 'feat($GT_ISSUE): <description>'
   - git push origin HEAD
   - gh pr create --fill

Stay focused on this specific task. Do not fix unrelated issues."
else
    PROMPT="You are a Gas Town polecat worker assigned to issue $GT_ISSUE. "
    PROMPT="${PROMPT}Read the repository and implement the task. "
    PROMPT="${PROMPT}After completing: git add, commit, push, and gh pr create --fill."
fi

exec claude --print --dangerously-skip-permissions "$PROMPT"
`, ClaudeCredsMountPath, ClaudeCredsMountPath, GitCredsMountPath, GitCredsMountPath)

	// Build environment variables
	envVars := []corev1.EnvVar{
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
			Name:  "GT_TASK_DESCRIPTION",
			Value: b.polecat.Spec.TaskDescription,
		},
		{
			Name:  "HOME",
			Value: HomeMountPath,
		},
	}

	// Add API key from secret if configured
	if k8sSpec.ApiKeySecretRef != nil {
		envVars = append(envVars, corev1.EnvVar{
			Name: "ANTHROPIC_API_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: k8sSpec.ApiKeySecretRef.Name,
					},
					Key: k8sSpec.ApiKeySecretRef.Key,
				},
			},
		})
	}

	// Build volume mounts
	volumeMounts := []corev1.VolumeMount{
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
	}

	// Add claude creds mount only if configured (for OAuth auth)
	if k8sSpec.ClaudeCredsSecretRef != nil {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      ClaudeCredsVolumeName,
			MountPath: ClaudeCredsMountPath,
			ReadOnly:  true,
		})
	}

	container := corev1.Container{
		Name:            ClaudeContainerName,
		Image:           image,
		Command:         []string{"/bin/sh", "-c"},
		Args:            []string{agentScript},
		WorkingDir:      fmt.Sprintf("%s/repo", WorkspaceMountPath),
		SecurityContext: b.buildSecurityContext(),
		Env:             envVars,
		VolumeMounts:    volumeMounts,
		Resources:       b.buildResources(),
	}

	return container
}

// buildTelemetrySidecar creates the telemetry sidecar container spec
//
//nolint:lll // Prometheus metric lines in embedded shell script cannot be broken
func (b *Builder) buildTelemetrySidecar() corev1.Container {
	// Telemetry script that monitors the pod and writes metrics
	telemetryScript := `
set -e

# Create metrics endpoint script
cat > /metrics/collect.sh << 'SCRIPT'
#!/bin/sh
# Collect basic telemetry and output Prometheus metrics

POLECAT_NAME="${POLECAT_NAME:-unknown}"
POLECAT_RIG="${POLECAT_RIG:-unknown}"
POLECAT_BEAD="${POLECAT_BEAD:-unknown}"
START_TIME=$(date +%s)

while true; do
  CURRENT_TIME=$(date +%s)
  ELAPSED=$((CURRENT_TIME - START_TIME))

  # Write basic metrics in Prometheus format
  {
    echo "# HELP polecat_execution_duration_seconds Total execution time of the polecat"
    echo "# TYPE polecat_execution_duration_seconds counter"
    echo "polecat_execution_duration_seconds{polecat=\"$POLECAT_NAME\",rig=\"$POLECAT_RIG\",bead=\"$POLECAT_BEAD\"} $ELAPSED"

    # Check if main container is running (claude)
    if ps aux | grep -q '[n]ode.*claude'; then
      echo "# HELP polecat_agent_running Agent container status (1=running, 0=stopped)"
      echo "# TYPE polecat_agent_running gauge"
      echo "polecat_agent_running{polecat=\"$POLECAT_NAME\",rig=\"$POLECAT_RIG\",bead=\"$POLECAT_BEAD\"} 1"
    else
      echo "polecat_agent_running{polecat=\"$POLECAT_NAME\",rig=\"$POLECAT_RIG\",bead=\"$POLECAT_BEAD\"} 0"
    fi
  } > /metrics/metrics.txt

  sleep 5
done
SCRIPT

chmod +x /metrics/collect.sh

# Start the metrics collector in background
/metrics/collect.sh &

# Start a simple HTTP server to expose metrics
while true; do
  {
    echo "HTTP/1.1 200 OK"
    echo "Content-Type: text/plain; version=0.0.4"
    echo "Connection: close"
    echo ""
    cat /metrics/metrics.txt 2>/dev/null || echo "# No metrics available yet"
  } | nc -l -p 8080 -q 1
done
`

	return corev1.Container{
		Name:            TelemetryContainerName,
		Image:           GetTelemetryImage(),
		Command:         []string{"/bin/sh", "-c"},
		Args:            []string{telemetryScript},
		SecurityContext: b.buildSecurityContext(),
		Env: []corev1.EnvVar{
			{
				Name:  "POLECAT_NAME",
				Value: b.polecat.Name,
			},
			{
				Name:  "POLECAT_RIG",
				Value: b.polecat.Spec.Rig,
			},
			{
				Name:  "POLECAT_BEAD",
				Value: b.polecat.Spec.BeadID,
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      MetricsVolumeName,
				MountPath: MetricsMountPath,
			},
			{
				Name:      TmpVolumeName,
				MountPath: TmpMountPath,
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(TelemetryCPURequest),
				corev1.ResourceMemory: resource.MustParse(TelemetryMemoryRequest),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(TelemetryCPULimit),
				corev1.ResourceMemory: resource.MustParse(TelemetryMemoryLimit),
			},
		},
	}
}

// buildVolumes creates the volume specifications
func (b *Builder) buildVolumes() []corev1.Volume {
	k8sSpec := b.polecat.Spec.Kubernetes

	volumes := []corev1.Volume{
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
		{
			Name: MetricsVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	// Add claude creds volume only if configured (for OAuth auth)
	if k8sSpec.ClaudeCredsSecretRef != nil {
		volumes = append(volumes, corev1.Volume{
			Name: ClaudeCredsVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: k8sSpec.ClaudeCredsSecretRef.Name,
				},
			},
		})
	}

	// Add SSH known_hosts ConfigMap volume if configured
	if k8sSpec.SSHKnownHostsConfigMapRef != nil {
		volumes = append(volumes, corev1.Volume{
			Name: SSHKnownHostsVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: *k8sSpec.SSHKnownHostsConfigMapRef,
				},
			},
		})
	}

	return volumes
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
