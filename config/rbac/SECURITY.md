# RBAC Security Notes

## Secret Access

The operator requires read access to Secrets across all namespaces (`get`, `list`, `watch`).

### Why This Access Is Needed

Secret access is required for:
- **GitSecretRef**: SSH keys for git operations in Polecats and Refineries
- **ClaudeCredsSecretRef**: Claude credentials for AI agent sessions
- **APIKeySecretRef**: API keys for Polecat AI configuration

### Why resourceNames Cannot Be Used

Secret names are user-defined via CRD specs (`spec.kubernetes.gitSecretRef.name`, etc.),
so we cannot restrict access to specific named secrets. Each Polecat/BeadStore/Refinery
may reference different secrets.

### Security Mitigations

1. **Read-only access**: No create/update/delete permissions on secrets
2. **Namespace isolation**: Secrets are only accessed in the same namespace as the referencing CRD
3. **No secret contents in logs**: The operator never logs secret contents
4. **Least privilege for other resources**: All other RBAC follows least-privilege principles

### Namespace-Scoped Alternative

For environments requiring strict namespace isolation, deploy the operator with namespace-scoped RBAC:

**Instead of ClusterRole** (cluster-wide access):
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole  # ← Grants access to ALL namespaces
metadata:
  name: gastown-operator
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list", "watch"]
```

**Use Role per namespace** (namespace-scoped access):
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role  # ← Scoped to single namespace
metadata:
  name: gastown-operator
  namespace: gastown-polecats  # Repeat for each namespace
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list", "watch"]
```

**Tradeoffs**:
- ✅ **Stronger isolation**: Compromised operator cannot read secrets in other namespaces
- ✅ **Compliance**: Meets stricter multi-tenancy requirements (e.g., PCI-DSS, HIPAA)
- ❌ **Operational overhead**: Requires RBAC creation per namespace where Polecats run
- ❌ **No cross-namespace secrets**: Cannot reference secrets in different namespaces

### Webhook Validation (Defense-in-Depth)

For clusters using ClusterRole but wanting to prevent cross-namespace secret access, implement a **validating admission webhook** that rejects Polecats referencing secrets in other namespaces:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: gastown-operator-webhook
webhooks:
  - name: validate.polecat.gastown.io
    rules:
      - operations: ["CREATE", "UPDATE"]
        apiGroups: ["gastown.gastown.io"]
        apiVersions: ["v1alpha1"]
        resources: ["polecats"]
    failurePolicy: Fail  # Reject on webhook failure (secure default)
```

This enforces same-namespace secrets at admission time, reducing blast radius.

**See**: `docs/SECURITY.md` for full cluster-wide secret access threat model and webhook implementation guidance.

## Pod Access

The operator requires full CRUD access to Pods for creating Polecat worker pods.

## Events Access

The operator can create and patch Events for status reporting.
