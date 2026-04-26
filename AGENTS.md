# AGENTS.md

## Build Commands

```bash
# Build all
make all

# Build Go binaries only (gateway, master, node, cococtl)
make build-go

# Build Zig binaries only (visor, agent, fork, net)
make build-zig

# Generate protobuf code
make proto

# Run all tests
make test

# Run Go tests only
make test-go

# Run Zig tests only
make test-zig

# Clean build artifacts
make clean
```

## Key Commands

- `go build -trimpath -o bin/coco-gateway ./cmd/coco-gateway` - Build gateway
- `cd daemon/coco-visor && zig build -Doptimize=ReleaseSafe` - Build Zig components
- `protoc --go_out=. --go-grpc_out=. proto/coco/v1/*.proto` - Generate proto

## Architecture

Control plane (Go): Gateway (REST) → Master (gRPC) → Node (gRPC)
Data plane (Zig): Visor (Unix socket) → Agent (VSock)
Network: Go + C/eBPF (XDP filters)

Binaries output to `bin/` for Go, `daemon/*/zig-out/bin/` for Zig.

## Requirements

- Go 1.23
- Zig 0.14
- protoc compiler
- KVM-enabled host (for running tests/integration)
- btrfs (for reflink/fork operations)

## Directory Structure

- `cmd/` - Go entry points (gateway, master, node, cococtl)
- `daemon/` - Zig services (coco-visor, coco-agent, coco-fork, coco-net)
- `pkg/` - Shared Go packages
- `proto/` - Protocol buffer definitions
- `ebpf/` - C/eBPF network programs
- `spec/` - Architecture specs (source of truth)

## Testing

Integration tests require docker-compose. Health endpoint at `http://localhost:4747/health`.

## Style Rules

- No comments unless requested
- No version numbers in specs
- Follow existing file naming conventions (lowercase with underscores)

## Reference

- Full architecture in `spec/00-overview.md` and `spec/01-folder-structure.md`
- Existing guidance in `CLAUDE.md`
