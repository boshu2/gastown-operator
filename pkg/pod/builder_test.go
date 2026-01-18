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
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
)

func TestNewBuilder(t *testing.T) {
	polecat := &gastownv1alpha1.Polecat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-polecat",
			Namespace: "default",
		},
		Spec: gastownv1alpha1.PolecatSpec{
			Rig:    "test-rig",
			BeadID: "test-bead",
		},
	}

	builder := NewBuilder(polecat)
	if builder == nil {
		t.Error("expected builder to be created")
	}
}

func TestBuilderBuild(t *testing.T) {
	t.Run("returns error without kubernetes spec", func(t *testing.T) {
		polecat := &gastownv1alpha1.Polecat{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-polecat",
				Namespace: "default",
			},
			Spec: gastownv1alpha1.PolecatSpec{
				Rig:    "test-rig",
				BeadID: "test-bead",
			},
		}

		builder := NewBuilder(polecat)
		_, err := builder.Build()

		if err == nil {
			t.Error("expected error without kubernetes spec")
		}
	})

	t.Run("creates valid pod with kubernetes spec", func(t *testing.T) {
		polecat := &gastownv1alpha1.Polecat{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-polecat",
				Namespace: "test-ns",
			},
			Spec: gastownv1alpha1.PolecatSpec{
				Rig:    "test-rig",
				BeadID: "test-bead",
				Kubernetes: &gastownv1alpha1.KubernetesSpec{
					GitRepository:        "git@github.com:org/repo.git",
					GitBranch:            "main",
					GitSecretRef:         gastownv1alpha1.SecretReference{Name: "git-secret"},
					ClaudeCredsSecretRef: gastownv1alpha1.SecretReference{Name: "claude-secret"},
				},
			},
		}

		builder := NewBuilder(polecat)
		pod, err := builder.Build()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify pod name
		if pod.Name != "polecat-test-polecat" {
			t.Errorf("expected pod name polecat-test-polecat, got %s", pod.Name)
		}

		// Verify namespace
		if pod.Namespace != "test-ns" {
			t.Errorf("expected namespace test-ns, got %s", pod.Namespace)
		}

		// Verify labels
		if pod.Labels["gastown.io/polecat"] != "test-polecat" {
			t.Error("missing or incorrect polecat label")
		}
		if pod.Labels["gastown.io/rig"] != "test-rig" {
			t.Error("missing or incorrect rig label")
		}
		if pod.Labels["gastown.io/bead"] != "test-bead" {
			t.Error("missing or incorrect bead label")
		}

		// Verify restart policy
		if pod.Spec.RestartPolicy != corev1.RestartPolicyNever {
			t.Error("expected RestartPolicyNever")
		}

		// Verify init containers
		if len(pod.Spec.InitContainers) != 1 {
			t.Fatalf("expected 1 init container, got %d", len(pod.Spec.InitContainers))
		}
		if pod.Spec.InitContainers[0].Name != GitInitContainerName {
			t.Error("expected git-init container")
		}

		// Verify containers
		if len(pod.Spec.Containers) != 1 {
			t.Fatalf("expected 1 container, got %d", len(pod.Spec.Containers))
		}
		if pod.Spec.Containers[0].Name != ClaudeContainerName {
			t.Error("expected claude container")
		}

		// Verify volumes
		expectedVolumes := map[string]bool{
			WorkspaceVolumeName:   false,
			GitCredsVolumeName:    false,
			ClaudeCredsVolumeName: false,
			TmpVolumeName:         false,
			HomeVolumeName:        false,
		}
		for _, vol := range pod.Spec.Volumes {
			if _, ok := expectedVolumes[vol.Name]; ok {
				expectedVolumes[vol.Name] = true
			}
		}
		for name, found := range expectedVolumes {
			if !found {
				t.Errorf("missing expected volume: %s", name)
			}
		}
	})

	t.Run("uses custom work branch", func(t *testing.T) {
		polecat := &gastownv1alpha1.Polecat{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-polecat",
				Namespace: "default",
			},
			Spec: gastownv1alpha1.PolecatSpec{
				Rig:    "test-rig",
				BeadID: "test-bead",
				Kubernetes: &gastownv1alpha1.KubernetesSpec{
					GitRepository:        "git@github.com:org/repo.git",
					GitBranch:            "main",
					WorkBranch:           "custom-branch",
					GitSecretRef:         gastownv1alpha1.SecretReference{Name: "git-secret"},
					ClaudeCredsSecretRef: gastownv1alpha1.SecretReference{Name: "claude-secret"},
				},
			},
		}

		builder := NewBuilder(polecat)
		pod, err := builder.Build()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check that init container script contains custom branch
		initScript := pod.Spec.InitContainers[0].Args[0]
		if initScript == "" {
			t.Error("init container script is empty")
		}
	})

	t.Run("uses custom image", func(t *testing.T) {
		polecat := &gastownv1alpha1.Polecat{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-polecat",
				Namespace: "default",
			},
			Spec: gastownv1alpha1.PolecatSpec{
				Rig:    "test-rig",
				BeadID: "test-bead",
				Kubernetes: &gastownv1alpha1.KubernetesSpec{
					GitRepository:        "git@github.com:org/repo.git",
					GitBranch:            "main",
					Image:                "custom/image:v1",
					GitSecretRef:         gastownv1alpha1.SecretReference{Name: "git-secret"},
					ClaudeCredsSecretRef: gastownv1alpha1.SecretReference{Name: "claude-secret"},
				},
			},
		}

		builder := NewBuilder(polecat)
		pod, err := builder.Build()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pod.Spec.Containers[0].Image != "custom/image:v1" {
			t.Errorf("expected custom image, got %s", pod.Spec.Containers[0].Image)
		}
	})
}

func TestBuildSecurityContext(t *testing.T) {
	polecat := &gastownv1alpha1.Polecat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-polecat",
			Namespace: "default",
		},
		Spec: gastownv1alpha1.PolecatSpec{
			Rig:    "test-rig",
			BeadID: "test-bead",
			Kubernetes: &gastownv1alpha1.KubernetesSpec{
				GitRepository:        "git@github.com:org/repo.git",
				GitBranch:            "main",
				GitSecretRef:         gastownv1alpha1.SecretReference{Name: "git-secret"},
				ClaudeCredsSecretRef: gastownv1alpha1.SecretReference{Name: "claude-secret"},
			},
		},
	}

	builder := NewBuilder(polecat)
	pod, err := builder.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("pod security context is restricted", func(t *testing.T) {
		psc := pod.Spec.SecurityContext
		if psc == nil {
			t.Fatal("pod security context is nil")
		}

		if psc.RunAsNonRoot == nil || !*psc.RunAsNonRoot {
			t.Error("expected RunAsNonRoot to be true")
		}

		if psc.RunAsUser == nil || *psc.RunAsUser != 65532 {
			t.Errorf("expected RunAsUser 65532, got %v", psc.RunAsUser)
		}

		if psc.RunAsGroup == nil || *psc.RunAsGroup != 65532 {
			t.Errorf("expected RunAsGroup 65532, got %v", psc.RunAsGroup)
		}

		if psc.FSGroup == nil || *psc.FSGroup != 65532 {
			t.Errorf("expected FSGroup 65532, got %v", psc.FSGroup)
		}

		if psc.SeccompProfile == nil || psc.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
			t.Error("expected RuntimeDefault seccomp profile")
		}
	})

	t.Run("container security context is restricted", func(t *testing.T) {
		csc := pod.Spec.Containers[0].SecurityContext
		if csc == nil {
			t.Fatal("container security context is nil")
		}

		if csc.RunAsNonRoot == nil || !*csc.RunAsNonRoot {
			t.Error("expected RunAsNonRoot to be true")
		}

		if csc.AllowPrivilegeEscalation == nil || *csc.AllowPrivilegeEscalation {
			t.Error("expected AllowPrivilegeEscalation to be false")
		}

		if csc.ReadOnlyRootFilesystem == nil || !*csc.ReadOnlyRootFilesystem {
			t.Error("expected ReadOnlyRootFilesystem to be true")
		}

		if csc.Capabilities == nil || len(csc.Capabilities.Drop) == 0 {
			t.Error("expected capabilities to be dropped")
		} else {
			foundAll := false
			for _, cap := range csc.Capabilities.Drop {
				if cap == "ALL" {
					foundAll = true
					break
				}
			}
			if !foundAll {
				t.Error("expected ALL capabilities to be dropped")
			}
		}
	})
}

func TestBuildResources(t *testing.T) {
	t.Run("uses default resources", func(t *testing.T) {
		polecat := &gastownv1alpha1.Polecat{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-polecat",
				Namespace: "default",
			},
			Spec: gastownv1alpha1.PolecatSpec{
				Rig:    "test-rig",
				BeadID: "test-bead",
				Kubernetes: &gastownv1alpha1.KubernetesSpec{
					GitRepository:        "git@github.com:org/repo.git",
					GitBranch:            "main",
					GitSecretRef:         gastownv1alpha1.SecretReference{Name: "git-secret"},
					ClaudeCredsSecretRef: gastownv1alpha1.SecretReference{Name: "claude-secret"},
				},
			},
		}

		builder := NewBuilder(polecat)
		pod, err := builder.Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		resources := pod.Spec.Containers[0].Resources

		expectedCPURequest := resource.MustParse(DefaultCPURequest)
		if !resources.Requests.Cpu().Equal(expectedCPURequest) {
			t.Errorf("expected CPU request %s, got %s", DefaultCPURequest, resources.Requests.Cpu().String())
		}

		expectedMemRequest := resource.MustParse(DefaultMemoryRequest)
		if !resources.Requests.Memory().Equal(expectedMemRequest) {
			t.Errorf("expected memory request %s, got %s", DefaultMemoryRequest, resources.Requests.Memory().String())
		}
	})

	t.Run("uses custom resources", func(t *testing.T) {
		customResources := &corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
		}

		polecat := &gastownv1alpha1.Polecat{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-polecat",
				Namespace: "default",
			},
			Spec: gastownv1alpha1.PolecatSpec{
				Rig:    "test-rig",
				BeadID: "test-bead",
				Kubernetes: &gastownv1alpha1.KubernetesSpec{
					GitRepository:        "git@github.com:org/repo.git",
					GitBranch:            "main",
					Resources:            customResources,
					GitSecretRef:         gastownv1alpha1.SecretReference{Name: "git-secret"},
					ClaudeCredsSecretRef: gastownv1alpha1.SecretReference{Name: "claude-secret"},
				},
			},
		}

		builder := NewBuilder(polecat)
		pod, err := builder.Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		resources := pod.Spec.Containers[0].Resources

		expectedCPU := resource.MustParse("1")
		if !resources.Requests.Cpu().Equal(expectedCPU) {
			t.Errorf("expected CPU request 1, got %s", resources.Requests.Cpu().String())
		}

		expectedMem := resource.MustParse("2Gi")
		if !resources.Requests.Memory().Equal(expectedMem) {
			t.Errorf("expected memory request 2Gi, got %s", resources.Requests.Memory().String())
		}
	})
}

func TestGetGitImage(t *testing.T) {
	t.Run("returns default image", func(t *testing.T) {
		_ = os.Unsetenv(EnvGitImage)
		img := GetGitImage()
		if img != DefaultGitImage {
			t.Errorf("expected %s, got %s", DefaultGitImage, img)
		}
	})

	t.Run("returns env var image", func(t *testing.T) {
		_ = os.Setenv(EnvGitImage, "custom/git:latest")
		defer func() { _ = os.Unsetenv(EnvGitImage) }()

		img := GetGitImage()
		if img != "custom/git:latest" {
			t.Errorf("expected custom/git:latest, got %s", img)
		}
	})
}

func TestGetClaudeImage(t *testing.T) {
	t.Run("returns default image", func(t *testing.T) {
		_ = os.Unsetenv(EnvClaudeImage)
		img := GetClaudeImage()
		if img != DefaultClaudeImage {
			t.Errorf("expected %s, got %s", DefaultClaudeImage, img)
		}
	})

	t.Run("returns env var image", func(t *testing.T) {
		_ = os.Setenv(EnvClaudeImage, "custom/claude:latest")
		defer func() { _ = os.Unsetenv(EnvClaudeImage) }()

		img := GetClaudeImage()
		if img != "custom/claude:latest" {
			t.Errorf("expected custom/claude:latest, got %s", img)
		}
	})
}

func TestContainerEnvironment(t *testing.T) {
	polecat := &gastownv1alpha1.Polecat{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-polecat",
			Namespace: "default",
		},
		Spec: gastownv1alpha1.PolecatSpec{
			Rig:    "test-rig",
			BeadID: "test-bead-123",
			Kubernetes: &gastownv1alpha1.KubernetesSpec{
				GitRepository:        "git@github.com:org/repo.git",
				GitBranch:            "main",
				GitSecretRef:         gastownv1alpha1.SecretReference{Name: "git-secret"},
				ClaudeCredsSecretRef: gastownv1alpha1.SecretReference{Name: "claude-secret"},
			},
		},
	}

	builder := NewBuilder(polecat)
	pod, err := builder.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	envVars := pod.Spec.Containers[0].Env
	envMap := make(map[string]string)
	for _, env := range envVars {
		envMap[env.Name] = env.Value
	}

	t.Run("sets CLAUDE_CONFIG_DIR", func(t *testing.T) {
		if envMap["CLAUDE_CONFIG_DIR"] != ClaudeCredsMountPath {
			t.Errorf("expected CLAUDE_CONFIG_DIR=%s, got %s", ClaudeCredsMountPath, envMap["CLAUDE_CONFIG_DIR"])
		}
	})

	t.Run("sets GT_ISSUE", func(t *testing.T) {
		if envMap["GT_ISSUE"] != "test-bead-123" {
			t.Errorf("expected GT_ISSUE=test-bead-123, got %s", envMap["GT_ISSUE"])
		}
	})

	t.Run("sets GT_POLECAT", func(t *testing.T) {
		if envMap["GT_POLECAT"] != "test-polecat" {
			t.Errorf("expected GT_POLECAT=test-polecat, got %s", envMap["GT_POLECAT"])
		}
	})

	t.Run("sets GT_RIG", func(t *testing.T) {
		if envMap["GT_RIG"] != "test-rig" {
			t.Errorf("expected GT_RIG=test-rig, got %s", envMap["GT_RIG"])
		}
	})

	t.Run("sets HOME", func(t *testing.T) {
		if envMap["HOME"] != HomeMountPath {
			t.Errorf("expected HOME=%s, got %s", HomeMountPath, envMap["HOME"])
		}
	})
}
