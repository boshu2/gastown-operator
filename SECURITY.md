# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **Do not** open a public issue for security vulnerabilities
2. Email the maintainers directly with details
3. Include steps to reproduce the vulnerability
4. Allow reasonable time for a fix before public disclosure

## Scope

This operator manages Gas Town resources in Kubernetes. Security considerations include:

- **RBAC**: The operator requires cluster-level permissions for CRD management
- **Pod Security**: Polecat pods run with restricted security contexts
- **Secrets**: Git SSH keys and Claude API credentials are stored as K8s Secrets
- **Network**: Pods may need egress to git remotes and Anthropic API

## Security Features

### Enterprise Edition (FIPS)

- FIPS 140-2 validated cryptography via Go BoringCrypto
- Red Hat UBI9 base images (security-hardened)
- Passes OpenShift restricted SCC

### Pod Security

All managed pods run with:
- `runAsNonRoot: true`
- `readOnlyRootFilesystem: true`
- `allowPrivilegeEscalation: false`
- `capabilities.drop: ["ALL"]`
- `seccompProfile: RuntimeDefault`

## Best Practices

When deploying:

1. Use NetworkPolicies to restrict pod egress
2. Rotate credentials regularly
3. Use separate namespaces for different trust levels
4. Enable audit logging
5. Review RBAC permissions for least privilege

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Updates

Security updates will be released as patch versions.
