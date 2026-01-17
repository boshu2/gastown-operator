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

## Pod Access

The operator requires full CRUD access to Pods for creating Polecat worker pods.

## Events Access

The operator can create and patch Events for status reporting.
