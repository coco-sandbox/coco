# Coco Sandbox – Specification index

This directory is the **source of truth** for developers working on Coco: architecture, interfaces, and behavior classes. **Normative** content describes what the system must do; **SLO and target** numbers live primarily in `06-performance.md`; **deployment-tuned** values and **in-tree defaults** (environment variables, gateway listen address) are documented in `10-self-hosting-and-operations.md` and the **reference** mapping in `11-repository-conformance.md`. **Defined terms, BCP 14 normative language, and the bar for a document-complete spec** are in `12-glossary-and-conventions.md`. **Neutrality, “always current” spec writing (no backward-compatibility story inside `spec/`)**, and **where community worker nodes** fit are in `14-deployment-topology-and-neutrality.md`.

**Conventions:** No task-list checkboxes; no fenced *code* blocks in most files (use tables and prose). **Mermaid** is allowed **only** in [13-use-cases-and-consumers.md](13-use-cases-and-consumers.md) section 5 for informative flow diagrams. Elsewhere, relationships remain tabular. Cross-references use relative file names.

**“100% specification” (self-check):** a reader can verify the document set against the eight criteria in `12-glossary-and-conventions.md` section 5. That definition applies to **docs only** and is independent of build or test status.

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
| [11-repository-conformance.md](11-repository-conformance.md) | Reference bind and env names for the open-source line; convenience layer on top of 02/10 |
| [12-glossary-and-conventions.md](12-glossary-and-conventions.md) | Glossary, BCP 14, single-source table, spec completeness criteria |
| [13-use-cases-and-consumers.md](13-use-cases-and-consumers.md) | Informative: who uses Coco (dev, research, platforms, agent runtimes) and how that maps to the spec |
| [14-deployment-topology-and-neutrality.md](14-deployment-topology-and-neutrality.md) | Neutrality, spec-as-current-only, community node pattern (platform on top) |

## Quick reference

### Component languages

| Component | Language |
|-----------|----------|
| Gateway, Master, Node, cococtl, Net | Go |
| Visor, Agent, Fork | Zig |
| eBPF | C (compiled), loaded via Go + Cilium ebpf |

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

1. [12-glossary-and-conventions.md](12-glossary-and-conventions.md) – terms, normative language, single-source table, and **completeness criteria (section 5)**  
2. [00-overview.md](00-overview.md) – mental model and component list  
3. [10-self-hosting-and-operations.md](10-self-hosting-and-operations.md) – what is fixed by architecture vs by operators  
4. [11-repository-conformance.md](11-repository-conformance.md) – reference environment and bind defaults (optional if you only need abstract behavior)  
5. [01-folder-structure.md](01-folder-structure.md) – where to add code  
6. [02-api.md](02-api.md) and [04-security.md](04-security.md) for API and trust boundaries  
7. [03-network.md](03-network.md), [05-storage.md](05-storage.md), [07-cluster.md](07-cluster.md) as needed  
8. [06-performance.md](06-performance.md), [08-observability.md](08-observability.md), [09-dependencies.md](09-dependencies.md) for SLOs, operations, and tooling  
9. [13-use-cases-and-consumers.md](13-use-cases-and-consumers.md) (optional) – narrative map of **consumers and intent**; not a second normative path  
10. [14-deployment-topology-and-neutrality.md](14-deployment-topology-and-neutrality.md) (recommended for **platform** builders) – **neutral** Coco, **always-fresh** spec policy, **community workers**

## Status

The specification set is **authoritative** for the Coco design and is maintained as **the current design**, not a compatibility log ([14-deployment-topology-and-neutrality.md](14-deployment-topology-and-neutrality.md) section 1). **Toolchain pins** follow [09-dependencies.md](09-dependencies.md) and the repository. A **document-only** “complete” bar is defined in [12-glossary-and-conventions.md](12-glossary-and-conventions.md) section 5.
