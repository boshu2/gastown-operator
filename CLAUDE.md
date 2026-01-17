# gastown-operator

> **Recovery**: Run `gt prime` after compaction, clear, or new session

K8s operator for Gas Town local execution claims.

---

## Purpose

gastown-operator is the **bridge between K8s and local execution**. It allows Gas Town (local tmux-based execution) to claim and execute Automaton CRs that would otherwise run in K8s pods.

### Architecture Context (2026-01 Restructuring)

```
BEFORE (monolithic hephaestus):
┌─────────────────────────────────────────┐
│              HEPHAESTUS                 │
│  kagent + fractal + gastown-operator    │
└─────────────────────────────────────────┘

AFTER (separated concerns):
┌──────────────┐  ┌──────────────┐  ┌──────────────────┐
│  HEPHAESTUS  │  │   CYCLOPES   │  │ GASTOWN-OPERATOR │
│  (olympus.io │  │   (kagent    │  │  (local claim    │
│   operator)  │  │    fork)     │  │   webhook)       │
└──────────────┘  └──────────────┘  └──────────────────┘
       │                                     │
       │         ┌──────────────┐           │
       └────────>│   DAEDALUS   │<──────────┘
                 │  (gt CLI -   │
                 │   local exec)│
                 └──────────────┘
```

### Role in the Platform

| Component | Responsibility |
|-----------|----------------|
| **Hephaestus** | K8s operator for olympus.io CRDs (Automaton, Phalanx, Forge) |
| **Cyclopes** | kagent fork - AI agent SDK |
| **Daedalus** | Gas Town CLI - local tmux execution |
| **gastown-operator** | Webhook allowing local claims of Automaton CRs |

---

## How It Works

1. **Automaton CR created** in K8s (via Hephaestus or directly)
2. **gastown-operator webhook** intercepts
3. If `spec.executionMode: local` or local claim requested:
   - Marks Automaton as claimed by local
   - Sends notification to Gas Town
4. **Gas Town (gt sling)** picks up and executes locally
5. **Status reported back** to Automaton CR

---

## Key Files

| Path | Purpose |
|------|---------|
| `api/v1alpha1/` | CRD types (mirrors hephaestus olympus.io types) |
| `internal/controller/` | Reconciliation logic |
| `internal/webhook/` | Mutating/validating webhooks |
| `config/` | Kustomize manifests, RBAC |
| `helm/` | Helm chart for deployment |

---

## Development

```bash
# Run locally
make run

# Run tests
make test

# Build image
make docker-build IMG=gastown-operator:dev

# Deploy to cluster
make deploy IMG=gastown-operator:dev
```

---

## Beads

This rig uses prefix `go` for gastown-operator issues.

```bash
bd ready           # See unblocked issues
bd show go-xxxx    # View issue details
bd sync            # Sync with git
```

---

## Related Rigs

| Rig | Relationship |
|-----|--------------|
| hephaestus | Provides olympus.io CRDs we watch |
| daedalus | GT CLI that executes claimed work |
| athena | LiteLLM gateway for model access |

---

## See Also

- `~/gt/olympus/crew/goku/.agents/research/2026-01-14-hephaestus-complete-rewrite-scope.md`
- `~/gt/olympus/crew/goku/.agents/research/2026-01-15-post-titan-hephaestus-plan.md`
