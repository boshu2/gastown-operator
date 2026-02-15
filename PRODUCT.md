---
last_reviewed: 2026-02-15
---

# PRODUCT.md

## Mission

A Kubernetes operator that lets developers and teams run hundreds of Claude Code AI agents as pods, scaling beyond laptop limits for autonomous software development.

## Target Personas

### Persona 1: Solo Dev with Big Ambitions
- **Goal:** Parallelize work across 50+ coding tasks without keeping their laptop open
- **Pain point:** Local machine caps out at 20-30 agents, must stay online, CPU-bound

### Persona 2: Platform / DevOps Engineer
- **Goal:** Provide AI agent infrastructure to development teams with enterprise controls (FIPS, RBAC, resource limits)
- **Pain point:** No standardized way to run AI coding agents on Kubernetes with production-grade security and observability

### Persona 3: Engineering Team Lead
- **Goal:** Dispatch coding tasks to autonomous agents at scale — queue issues, close laptop, come back to PRs
- **Pain point:** Coordinating multiple agents locally is fragile (tmux, SSH, manual monitoring) and doesn't scale beyond one machine

## Core Value Propositions

- **Horizontal scale** — Go from 20-30 local agents to hundreds on a cluster. Queue 50 issues, dispatch 50 polecats, close your laptop.
- **Zero-ops agent lifecycle** — Full lifecycle management (spawn, monitor, merge, cleanup) handled by Kubernetes. No tmux, no SSH, no babysitting.
- **Enterprise-ready from day one** — OpenShift native, FIPS-compliant edition, SBOM, Trivy scans, provenance attestations. Production-grade security out of the box.
- **Same workflow, more compute** — Your `gt` CLI works exactly the same. The operator just gives you more compute — as much as your cluster can handle.

## Competitive Landscape

No direct competitors identified. Kubernetes-native AI agent orchestration is a novel category. The closest alternative is DIY — teams writing their own Dockerfiles and K8s manifests to run Claude Code in pods — but this lacks lifecycle management, health monitoring, merge automation, and enterprise security controls.

## Usage

This file enables product-aware council reviews:

- **`/pre-mortem`** — Automatically includes `product` perspectives (user-value, adoption-barriers, competitive-position) alongside plan-review judges when this file exists.
- **`/vibe`** — Automatically includes `developer-experience` perspectives (api-clarity, error-experience, discoverability) alongside code-review judges when this file exists.
- **`/council --preset=product`** — Run product review on demand.
- **`/council --preset=developer-experience`** — Run DX review on demand.

Explicit `--preset` overrides from the user skip auto-include (user intent takes precedence).
