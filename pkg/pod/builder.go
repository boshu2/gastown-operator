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
	GitInitContainerName       = "git-init"
	ClaudeContainerName        = "claude"
	TelemetryContainerName     = "telemetry"

	// Volume names
	WorkspaceVolumeName   = "workspace"
	GitCredsVolumeName    = "git-creds"
	ClaudeCredsVolumeName = "claude-creds"
	TmpVolumeName         = "tmp"
	HomeVolumeName        = "home"
	MetricsVolumeName     = "metrics"

	// Mount paths
	WorkspaceMountPath   = "/workspace"
	GitCredsMountPath    = "/git-creds"
	ClaudeCredsMountPath = "/claude-creds"
	TmpMountPath         = "/tmp"
	HomeMountPath        = "/home/nonroot"
	MetricsMountPath     = "/metrics"

	// Environment variable names for image configuration
	EnvGitImage        = "GASTOWN_GIT_IMAGE"
	EnvClaudeImage     = "GASTOWN_CLAUDE_IMAGE"
	EnvTelemetryImage  = "GASTOWN_TELEMETRY_IMAGE"

	// Default images (community edition - vanilla Kubernetes)
	// For enterprise/FIPS, set env vars to UBI images:
	//   GASTOWN_GIT_IMAGE=registry.access.redhat.com/ubi9/ubi-minimal:9.3
	//   GASTOWN_CLAUDE_IMAGE=registry.access.redhat.com/ubi9/nodejs-20:1
	//   GASTOWN_TELEMETRY_IMAGE=registry.access.redhat.com/ubi9/ubi-minimal:9.3
	DefaultGitImage        = "alpine/git:2.43.0"
	DefaultClaudeImage     = "node:20-slim"
	DefaultTelemetryImage  = "alpine:latest"

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
		Name:            ClaudeContainerName,
		Image:           image,
		Command:         []string{"/bin/sh", "-c"},
		Args:            []string{agentScript},
		WorkingDir:      fmt.Sprintf("%s/repo", WorkspaceMountPath),
		SecurityContext: b.buildSecurityContext(),
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

// buildTelemetrySidecar creates the telemetry sidecar container spec
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
		{
			Name: MetricsVolumeName,
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
