# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.x.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in gastown-operator, please report it
responsibly.

### How to Report

1. **Do not** open a public GitHub issue for security vulnerabilities
2. Email the maintainers directly or use GitHub's private vulnerability reporting
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Any suggested fixes (optional)

### What to Expect

- Acknowledgment within 48 hours
- Regular updates on progress
- Credit in the security advisory (if desired)

### Scope

This policy applies to:

- The gastown-operator codebase
- Official container images
- CRD definitions and RBAC configurations

Out of scope:

- Third-party dependencies (report to upstream)
- Issues in Kubernetes itself

## Security Best Practices

When deploying gastown-operator:

1. **RBAC**: Use minimal required permissions
2. **Network Policies**: Restrict pod communication
3. **Image Scanning**: Scan images before deployment
4. **Secrets**: Use Kubernetes secrets or external secret managers
5. **Updates**: Keep the operator updated to latest stable version
