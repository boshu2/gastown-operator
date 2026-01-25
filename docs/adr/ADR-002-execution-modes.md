# ADR-002: Kubernetes-Only Execution Mode

**Status**: Superseded (was: Accepted)
**Date**: 2026-01-01
**Superseded**: 2026-01-24

## Context

Originally, the operator supported two execution modes:

1. **Local**: Agent runs in a tmux session on the operator host
2. **Kubernetes**: Agent runs as a Pod in the cluster

## Decision (Superseded)

After evaluation, **local mode has been removed**. The operator now supports **Kubernetes-only execution**.

### Why Remove Local Mode?

1. **Complexity**: Two code paths doubled maintenance burden
2. **Security**: Local mode required tmux/host access - security anti-pattern
3. **Portability**: Local mode tied operator to specific host configuration
4. **Focus**: One execution mode done well > two done poorly
5. **OSS Readiness**: Clean standalone operator is more adoptable

## Current Implementation

All Polecats run as Kubernetes Pods:

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
spec:
  desiredState: Working
  kubernetes:
    gitRepository: "git@github.com:org/repo.git"
    gitSecretRef:
      name: git-creds
    apiKeySecretRef:
      name: anthropic-api-key
      key: api-key
```

The `executionMode` field defaults to `kubernetes` and only accepts `kubernetes`.

## Consequences

### Positive

- **Simpler codebase**: Single execution path to maintain
- **Cleaner dependencies**: No gt CLI or tmux required
- **True isolation**: Every polecat runs in its own pod
- **Cloud-native**: Works in any Kubernetes cluster

### Negative

- **No local debugging**: Can't attach to tmux for quick inspection
- **Slower iteration**: Pod startup adds latency vs local tmux

### Mitigations

- Use `kubectl logs -f` for live log streaming
- Use `kubectl exec` for interactive debugging
- Use short `activeDeadlineSeconds` during development
