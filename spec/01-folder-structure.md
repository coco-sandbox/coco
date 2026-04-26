# Coco Sandbox – Folder structure specification

**Scope:** Where code lives, naming rules, and how each top-level area maps to the architecture.  
**Status:** Authoritative.  
**Index:** [Specification index](index.md)

## Normative approach

Exhaustive **per-file** trees in documentation **drift** as the repository changes. This document defines the **intended layout**, **ownership**, and **naming conventions**. The on-disk **source of truth** is the repository; if a path or file is missing, either the work is not merged yet or the spec should be updated. Do not copy long directory trees from here into the spec.

## 1. Root layout

| Path | Purpose |
|------|---------|
| cmd/ | Go `main` packages: gateway, node, master, cli |
| daemon/ | Data-plane and long-running services (Zig and some Go) |
| pkg/ | Shared Go libraries consumed by `cmd/` and `daemon/*` (Go) |
| proto/ | Protobuf and RPC source definitions |
| ebpf/ | eBPF and XDP C sources and build glue |
| test/ | Unit, integration, e2e, and benchmark tests (as organized in tree) |
| configs/ | Example and environment-specific configuration **templates** |
| scripts/ | Build, test, and tooling scripts |
| deploy/ | Kubernetes, compose, and other deployment artifacts |
| spec/ | This specification set |

## 2. Command layer (cmd/)

Each subdirectory under `cmd/` is one **binary** (Go), with a `main` package, HTTP or RPC wiring, and subpackages for handlers, routes, and middleware as needed.

| Directory | Binary role |
|-----------|-------------|
| coco-gateway/ | Public HTTP API, middleware chain, internal clients to master/node |
| coco-node/ | Per-host agent: visor client, pool, local resources, health |
| coco-master/ | Cluster coordination: scheduling, etcd, node routing |
| cococtl/ | Operator and developer CLI: sandboxes, templates, checkpoints, cluster read-only ops |

**Typical sub-areas (names may vary):** `handlers/` or equivalent for HTTP/RPC, `middleware/` for auth, rate limits, logging; `docker/` for Dockerfile when present.

## 3. Daemon layer (daemon/)

| Directory | Technology | Role |
|-----------|------------|------|
| coco-visor/ | Zig | KVM micro-VM control, visor socket server, snapshots |
| coco-agent/ | Zig | Guest init, command execution, host channel |
| coco-fork/ | Zig | COW or fork pipeline support when split out |
| coco-net/ | Go + C (eBPF) | Policy, maps, and control for net stack |
| coco-checkpoint/ | Go ± Zig helpers | Checkpoint and restore when not only inside visor |

Zig projects use that component’s `build.zig` and `src/` layout; exact file names are **not** fixed here.

## 4. Package layer (pkg/)

Shared **Go** packages. Common areas:

| Area | Responsibility |
|------|----------------|
| api/ | Generated protobuf / Connect / HTTP bindings |
| visor/ | Visor socket client and protocol types |
| scheduler/ | Placement strategies and filters |
| pool/ | VM pool lifecycle for nodes |
| store/ | Local persistent state (for example key-value) |
| net/ | Host networking helpers, IPAM, eBPF load helpers |
| template/ | Template build, OCI, layout |
| cluster/ | Node registry and membership (client side) |
| metrics/ | Prometheus registration and common collectors |
| config/ | Default and env-driven configuration (when used) |

New packages should follow a single clear responsibility; avoid new top-level `pkg` roots without a design pass.

## 5. Protocol buffers (proto/)

- Public and internal APIs are defined under `proto/` (for example `coco/v1/*.proto` for external RPC and resource messages).
- Internal-only IPC (visor, agent) may live under a separate `proto` subtree.
- **Regenerate** code after changes using the project `Makefile` or `scripts/`.

## 6. eBPF (ebpf/)

| Subdirectory (concept) | Role |
|------------------------|------|
| from_sandbox / from_host / from_world / xdp (or project-specific) | Program boundaries by traffic direction and attach point |
| headers/ or common headers | Shared structs and map definitions |

Build uses the repository script or documented `clang` invocation; paths are not duplicated here as shell.

## 7. Tests (test/)

Organize by **scope**: unit (fast, mocked), integration (with real or fake components), e2e (full stack), benchmark. Exact filenames belong in the tree; the spec only requires that each major feature area (API, pool, visor client, network) have coverage appropriate to its risk.

## 8. Configuration and scripts

| Path | Role |
|------|------|
| configs/*.yaml | Templates for default, dev, node, gateway, master; not guaranteed to be loaded unless wired to a binary |
| scripts/build, scripts/test, scripts/docker, scripts/tools | Automation entry points referenced from `make` and docs |

## 9. Deployment (deploy/)

Kubernetes (operator, CRD, raw manifests, Helm), Dockerfiles, and optional other automation. Content is **manifest** structure, not Go layout; see `07-cluster.md` for operator behavior.

## 10. Language by component

| Layer | Component | Language |
|-------|-----------|----------|
| cmd | coco-gateway, coco-node, coco-master, cococtl | Go |
| daemon | coco-visor, coco-agent, coco-fork | Zig |
| daemon | coco-net | Go + eBPF (C) |
| daemon | coco-checkpoint | Go, optional Zig for close-to-hypervisor pieces |
| pkg | shared libraries | Go |
| ebpf | XDP, tc, or helper programs | C (restricted) |

## 11. Build outputs

| Output | Location (normative pattern) |
|--------|------------------------------|
| Go control-plane binaries | `bin/` (as produced by `go build` / make targets) |
| Zig components | `daemon/<component>/zig-out/bin/` or as defined in that `build.zig` and Makefile |
| eBPF objects | Next to sources or a build staging dir, per `scripts` |

Exact artifact names (for example `coco-gateway` vs `cocovisor`) follow the `Makefile` and `build.zig` in tree.

## 12. File naming conventions

- Go: lowercase with underscores for multi-word file names: `sandbox_service.go`.
- Zig: lowercase with underscores, aligned with `zig fmt` defaults.
- Proto: `snake_case.proto`.
- Config templates: `lowercase-with-dashes.yaml` for multi-word.
- eBPF: `name.bpf.c` for loadable C sources.
- All names should reflect **purpose** (resource or subsystem), not dates or personal prefixes.

---

Related: `00-overview.md` (architecture), `02-api.md` (HTTP surface), `09-dependencies.md` (toolchain).
