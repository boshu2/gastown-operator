package pod

import corev1 "k8s.io/api/core/v1"

// AgentContainerExitCode returns the claude agent container exit code when terminated.
func AgentContainerExitCode(p *corev1.Pod) (int32, bool) {
	for _, cs := range p.Status.ContainerStatuses {
		if cs.Name != ClaudeContainerName {
			continue
		}
		if cs.State.Terminated != nil {
			return cs.State.Terminated.ExitCode, true
		}
		return 0, false
	}
	return 0, false
}
