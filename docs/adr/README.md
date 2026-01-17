# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) documenting significant design decisions for the Gas Town Operator.

## Index

| ADR | Title | Status |
|-----|-------|--------|
| [ADR-001](ADR-001-crds-as-views.md) | CRDs as Views Pattern | Accepted |
| [ADR-002](ADR-002-execution-modes.md) | Local vs Kubernetes Execution Modes | Accepted |
| [ADR-003](ADR-003-finalizer-cleanup.md) | Finalizer Cleanup Strategy | Accepted |

## Template

When creating new ADRs, use this template:

```markdown
# ADR-NNN: Title

**Status**: Proposed | Accepted | Deprecated | Superseded
**Date**: YYYY-MM-DD

## Context

What is the issue that we're seeing that is motivating this decision?

## Decision

What is the change that we're proposing and/or doing?

## Consequences

What becomes easier or more difficult to do because of this change?
```
