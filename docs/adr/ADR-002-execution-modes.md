# ADR-002: Local vs Kubernetes Execution Modes

**Status**: Accepted
**Date**: 2026-01-01

## Context

Polecats (AI agent workers) need to execute tasks. Two execution environments exist:

1. **Local**: Agent runs in a tmux session on the operator host
2. **Kubernetes**: Agent runs as a Pod in the cluster

Users have different needs:
- Development: Quick iteration, easy debugging (favor local)
- Production: Isolation, scalability, resource management (favor Kubernetes)

## Decision

Support **both execution modes**, selectable per-Polecat via `spec.executionMode`.

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
spec:
  executionMode: kubernetes  # or "local"
  kubernetes:                # Only when mode=kubernetes
    gitRepository: "..."
    image: "..."
```

## Implementation

### Local Mode (default)

- Controller calls `gt sling` to dispatch work
- Agent runs in tmux session on operator host
- State tracked via `gt polecat status`

### Kubernetes Mode

- Controller creates a Pod with:
  - Init container: clones git repo with SSH key
  - Main container: runs Claude agent
  - Secrets mounted for credentials
- Pod lifecycle tracks work completion
- Results pushed to git before termination

## Consequences

### Positive

- **Gradual migration**: Users can adopt K8s execution incrementally
- **Development flexibility**: Local mode for rapid iteration
- **Production readiness**: K8s mode for isolation and scale
- **Resource management**: K8s provides CPU/memory limits

### Negative

- **Two code paths**: More complexity to maintain
- **Feature parity**: Some features may not work identically in both modes
- **Testing burden**: Need to test both execution paths

### Mitigations

- Clear documentation on mode differences
- Common interface for work dispatch
- Mode-specific validation in webhooks

## Mode Comparison

| Feature | Local | Kubernetes |
|---------|-------|------------|
| Isolation | Shared host | Pod sandbox |
| Scaling | Limited by host | Cluster capacity |
| Debugging | tmux attach | kubectl logs |
| Network | Host network | Pod network |
| Resources | Shared | Dedicated |
| Startup time | Fast | Slower (pod scheduling) |
| Cost | Free | Cluster resources |

## Migration Path

1. Start with local mode for development
2. Test kubernetes mode with non-critical work
3. Migrate production workloads to kubernetes mode
4. Keep local mode for debugging and development
