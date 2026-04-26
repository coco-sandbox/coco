# Coco Sandbox – Specification index

This directory is the **source of truth** for developers working on Coco: architecture, interfaces, and behavior classes. **Normative** content describes what the system must do; **SLO and target** numbers live primarily in `06-performance.md`; **deployment-tuned** values (ports, rate limits, retention) are not fixed here except where explicitly stated as examples—see `10-self-hosting-and-operations.md`.

**Conventions:** No checklist task lists; no fenced code blocks in spec files (use tables and prose). Cross-references use relative file names.

## Specification files

| File | Scope |
|------|--------|
| [00-overview.md](00-overview.md) | Architecture, components, cross-cutting narrative; points to other docs for detail |
| [01-folder-structure.md](01-folder-structure.md) | Repository layout rules and top-level map (not an exhaustive file tree) |
| [02-api.md](02-api.md) | Public HTTP API: resources, fields, errors, REST mapping |
| [03-network.md](03-network.md) | Network modes, eBPF path, policy and rate semantics |
| [04-security.md](04-security.md) | Isolation, seccomp, capabilities, authZ model |
| [05-storage.md](05-storage.md) | Template and checkpoint storage, runtime state |
| [06-performance.md](06-performance.md) | SLOs, optimization strategies, benchmark intent |
| [07-cluster.md](07-cluster.md) | Master, nodes, scheduling, failover, K8s CRD logic |
| [08-observability.md](08-observability.md) | Logs, metrics, traces, health endpoints |
| [09-dependencies.md](09-dependencies.md) | Build and runtime toolchains; `go.mod` is the Go version pin |
| [10-self-hosting-and-operations.md](10-self-hosting-and-operations.md) | What operators configure; fork and stability boundaries |

## Quick reference

### Component languages

| Component | Language |
|-----------|----------|
| Gateway, Master, Node, cococtl, much of Net | Go |
| Visor, Agent, Fork | Zig |
| eBPF | C |

### Communication paths

| Path | Protocol |
|------|----------|
| Client → Gateway | REST (JSON) |
| Gateway → Master, Master → Node | gRPC (internal) |
| Node → Visor | Unix domain socket (binary) |
| Visor ↔ Agent | VSock (TCP fallback in dev) |

### Performance targets (detail in 06)

Design targets, not universal guarantees: cold start, fork, memory overhead, network aggregate—see [06-performance.md](06-performance.md).

| Metric | Target (reference stack) |
|--------|--------------------------|
| Cold start | under 30 ms |
| Fork | under 10 ms |
| Memory overhead | under 2 MiB per sandbox (control plane) |
| Network throughput | 20 Gbps aggregate (node goal) |

## Reading order

1. [00-overview.md](00-overview.md) – mental model and component list  
2. [10-self-hosting-and-operations.md](10-self-hosting-and-operations.md) – what is fixed by architecture vs by operators  
3. [01-folder-structure.md](01-folder-structure.md) – where to add code  
4. [02-api.md](02-api.md) and [04-security.md](04-security.md) for API and trust boundaries  
5. [03-network.md](03-network.md), [05-storage.md](05-storage.md), [07-cluster.md](07-cluster.md) as needed  
6. [06-performance.md](06-performance.md), [08-observability.md](08-observability.md), [09-dependencies.md](09-dependencies.md) for SLOs, operations, and tooling  

## Status

The specification set is **authoritative** for the Coco design. **Toolchain versions** follow [09-dependencies.md](09-dependencies.md) and the repository `go.mod` and Zig build metadata.
