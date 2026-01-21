# Agent Instructions

See **[AGENT_INSTRUCTIONS.md](AGENT_INSTRUCTIONS.md)** for complete agent context and instructions.

This file exists for compatibility with tools that look for AGENTS.md.

## Quick Reference

```bash
# Install operator
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.3.2 \
  --namespace gastown-system \
  --create-namespace

# Create secrets (gastown-workers namespace)
kubectl create secret generic git-credentials -n gastown-workers \
  --from-file=ssh-privatekey=$HOME/.ssh/id_ed25519

kubectl create secret generic claude-credentials -n gastown-workers \
  --from-literal=api-key=$ANTHROPIC_API_KEY

# Verify
kubectl get pods -n gastown-system
kubectl get crds | grep gastown
```

## Full Setup

For complete setup instructions including OpenShift, OAuth credentials, and Polecat examples, see:
- [USER_GUIDE.md](docs/USER_GUIDE.md) - Complete walkthrough
- [AGENT_INSTRUCTIONS.md](AGENT_INSTRUCTIONS.md) - Agent context

## Key Warning

**DO NOT manually write Polecat YAML** in normal workflows. The Mayor dispatches work with `gt sling`, and Claude generates the appropriate Kubernetes resources. The operator handles the rest.
