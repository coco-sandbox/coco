# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Coco Sandbox

Coco is a sandbox runtime that provides hardware-level isolation using KVM. Each sandbox runs in its own MicroVM.

## Architecture Overview

**Control plane (Go):** Gateway (REST API entry point) → Master (cluster coordination) → Node (local host resources)  
**Data plane (Zig):** Visor (KVM VM manager) → Agent (in-VM init) → Fork (cloning)  
**Network:** Go + C/eBPF programs (XDP filters at kernel level)  
**External communication:** REST API (port 8080); internal use gRPC and Unix sockets.

Build output: Go binaries in `bin/`, Zig binaries in `daemon/*/zig-out/bin/`.

## Quick Start

```bash
make all            # Build everything
make build-go       # Just Go binaries (gateway, master, node, cococtl)
make build-zig      # Just Zig components (visor, agent, fork)
make test           # Run all tests
make proto          # Regenerate protobuf code
make clean          # Remove bin/ and zig-out/
```

`make build-zig` does **not** build `coco-net` (Go + C). `make test-zig` runs tests for both `coco-visor` and `coco-agent`. eBPF programs are not built by `make`; use the build script (or invoke clang directly with `-Iheaders`):

```bash
./scripts/build/build-ebpf.sh                              # iterates ebpf/{from_sandbox,from_host,from_world,xdp}
clang -O2 -target bpf -c <file>.bpf.c -o <file>.o -Iheaders  # single file
```

Build individual Go binaries:
```bash
go build -o bin/coco-gateway ./cmd/coco-gateway
go build -o bin/coco-master ./cmd/coco-master
go build -o bin/coco-node ./cmd/coco-node
go build -o bin/cococtl ./cmd/cococtl
```

Build individual Zig components:
```bash
cd daemon/coco-visor && zig build -Doptimize=ReleaseSafe
cd daemon/coco-agent && zig build -Doptimize=ReleaseSafe
cd daemon/coco-fork && zig build -Doptimize=ReleaseSafe
```

## Testing

Run all tests:
```bash
make test           # Go + Zig tests
make test-go        # Go tests only
make test-zig       # Zig tests only
```

Run individual Go test packages or functions:
```bash
go test ./pkg/api/...           # Test entire api package
go test -run TestSandbox ./pkg/api/...  # Run specific test by name
go test -v -race ./pkg/...      # Verbose output with race detector
```

Run Zig tests for a component:
```bash
cd daemon/coco-visor && zig build test
```

## Scripts

`make` targets are the primary entry points, but `scripts/` has a few extras not exposed via Make:

- `scripts/build/build-ebpf.sh` — compile eBPF programs (no Make target)
- `scripts/test/test-integration.sh` — integration tests (`make test` runs unit tests only)
- `scripts/test/test-unit.sh`, `scripts/test/test-all.sh` — alternates to `make test-go`/`make test`
- `scripts/tools/generate-proto.sh` — alternate to `make proto`
- `scripts/docker/build-images.sh`, `scripts/docker/push-images.sh` — container image lifecycle

## Linting

```bash
go fmt ./... && go vet ./...                # Go
cd daemon/coco-visor && zig fmt --check src # Zig (per component)
```

Run `go fmt`/`zig fmt` before declaring work done — this is a project convention recorded in `.claude/rules/`.

## Project Layout

- `cmd/` — Go entry points (coco-gateway, coco-master, coco-node, cococtl)
- `pkg/` — Shared Go packages (api, checkpoint, cluster, config, net, pool, scheduler, store, etc.)
- `daemon/` — Zig services (coco-visor, coco-agent, coco-fork, coco-net)
- `proto/` — Protocol buffer definitions (regenerate with `make proto`)
- `ebpf/` — C/eBPF network programs
- `spec/` — Architecture specifications (source of truth for design decisions)
- `test/` — Cross-component tests (core_test.go, integration, e2e, benchmark)
- `configs/` — Configuration templates
- `deploy/` — Deployment scripts

## Dependencies

- **Go 1.23** — for control plane
- **Zig 0.16** — for data plane
- **protoc** — for protobuf code generation
- **KVM-enabled host** — for running sandbox tests and integration tests
- **btrfs** — for reflink/fork operations (optional, affects fork performance)

## Development Practices

**No comments unless requested.** Code should be self-documenting. Only add comments for non-obvious "why" (hidden constraints, workarounds, subtle invariants).

**No version numbers in specs.** Versioning belongs in code, not specifications.

**`spec/` is the source of truth.** All architectural decisions live in `spec/`. Check `spec/00-overview.md` and `spec/01-folder-structure.md` before making structural changes — file paths and component layouts are normative, not aspirational. Do not invent requirements beyond what the spec documents.

**Per-language conventions (from `.claude/rules/`):**
- **Go** (`cmd/`, `pkg/`, `daemon/coco-*/**/*.go`): use standard library; run `go fmt` before finishing.
- **Zig** (`daemon/coco-{visor,agent,fork}/`): use direct KVM syscalls via `linux/uapi.h`; run `zig fmt` before finishing.
- **eBPF** (`ebpf/`): follow kernel coding style; use XDP for packet processing; compile with clang.

**Proto regeneration.** After modifying `proto/coco/v1/*.proto`, run `make proto`. Generated Go lands under `pkg/api/v1/` (and `pkg/api/v1/v1connect/` for Connect handlers).

**Integration tests.** Require docker-compose. Health check at `http://localhost:4747/health`.

## Communication Flows

**External clients** → REST API → **Gateway** (Go)  
**Gateway** → gRPC → **Master** (cluster decisions) and **Node** (local operations)  
**Node** → Unix socket → **Visor** (Zig, handles KVM)  
**Visor** → VSock/TCP → **Agent** (Zig, runs inside VM)

Each layer handles its responsibility cleanly. Modifying communication protocols requires updating both sides and potentially regenerating proto code.

## References

- Architecture specs: `spec/00-overview.md`, `spec/01-folder-structure.md`
- Build commands and structure: `AGENTS.md`
- API design: `spec/02-api.md`
- Network design: `spec/03-network.md`
