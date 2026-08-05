package pod

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestAgentContainerExitCode(t *testing.T) {
	t.Run("claude exited zero", func(t *testing.T) {
		p := &corev1.Pod{
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: ClaudeContainerName,
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
						},
					},
					{
						Name: TelemetryContainerName,
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: 1},
						},
					},
				},
			},
		}
		code, ok := AgentContainerExitCode(p)
		if !ok || code != 0 {
			t.Fatalf("expected claude exit 0, got code=%d ok=%v", code, ok)
		}
	})

	t.Run("claude still running", func(t *testing.T) {
		p := &corev1.Pod{
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:  ClaudeContainerName,
						State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
					},
				},
			},
		}
		_, ok := AgentContainerExitCode(p)
		if ok {
			t.Fatal("expected not terminated")
		}
	})
}
