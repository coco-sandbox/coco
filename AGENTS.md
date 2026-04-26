# AGENTS.md

## Build Commands

```bash
make all           # Build everything (Go + Zig)
make build-go      # Go: gateway, master, node, cococtl
make build-zig     # Zig: visor, agent, fork (net excluded)
make proto         # Generate Go from proto/coco/v1/*.proto
make test          # Run all tests
make clean         # Remove bin/ and zig-out/
```

**After modifying proto files, run `make proto` before building.**

## Linting

```bash
go fmt ./... && go vet ./...           # Go lint
cd daemon/coco-visor && zig fmt --check  # Zig format check
```

## Architecture

- Control plane (Go): Gateway (REST) → Master (gRPC) → Node (gRPC)
- Data plane (Zig): Visor (Unix socket) → Agent (VSock)
- Network: Go + C/eBPF (XDP filters)
- Go binaries: `bin/`, Zig binaries: `daemon/*/zig-out/bin/`

## Requirements

- Go 1.23, Zig 0.14, protoc
- KVM-enabled host (tests/integration)
- btrfs (reflink/fork operations)

## Directory Structure

- `cmd/` - Go entry points (coco-gateway, coco-master, coco-node, cococtl)
- `daemon/` - Zig services (coco-visor, coco-agent, coco-fork, coco-net)
- `pkg/` - Shared Go packages
- `proto/` - Protocol buffer definitions
- `ebpf/` - C/eBPF network programs
- `spec/` - Architecture specs (source of truth)

## eBPF Compilation

```bash
cd ebpf && clang -O2 -target bpf -c <file>.bpf.c -o <file>.o
```

## Testing

Integration tests require docker-compose. Health endpoint: `http://localhost:4747/health`

## Style Rules

- No comments unless requested
- No version numbers in specs
- Follow existing file naming conventions (lowercase with underscores)

## Reference

- Full architecture in `spec/00-overview.md` and `spec/01-folder-structure.md`
- Existing guidance in `CLAUDE.md`
