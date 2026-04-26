# AGENTS.md

## Build Commands

```bash
make all           # Build everything (Go + Zig)
make build-go      # Go: gateway, master, node, checkpoint, net, cococtl
make build-zig     # Zig: visor, agent, fork (net excluded - built as Go)
make proto         # Generate Go from proto/coco/v1/*.proto
make test          # Run all tests (go test + zig build test)
make clean         # Remove bin/ and zig-out/
```

**Proto → build order required:** After modifying proto files, run `make proto` before building.

**Individual Go binaries:**
```bash
go build -o bin/coco-gateway ./cmd/coco-gateway
go build -o bin/coco-master ./cmd/coco-master
go build -o bin/coco-node ./cmd/coco-node
go build -o bin/cococtl ./cmd/cococtl
```

**Individual Zig components:**
```bash
cd daemon/coco-visor && zig build -Doptimize=ReleaseSafe
cd daemon/coco-agent && zig build -Doptimize=ReleaseSmall
cd daemon/coco-fork && zig build -Doptimize=ReleaseSafe
```

**Individual test (Go):** `go test ./pkg/api/...` or `go test -run TestName ./pkg/...`

## Linting

```bash
go fmt ./... && go vet ./...           # Go lint
cd daemon/coco-visor && zig fmt --check  # Zig format check
```

## Architecture

- Control plane (Go): Gateway (REST) → Master (gRPC) → Node (gRPC)
- Data plane (Zig): Visor (Unix socket) → Agent (VSock)
- Network: Go + C/eBPF (XDP filters)
- Checkpoint/Restore: Go (daemon/coco-checkpoint)
- Go binaries: `bin/`, Zig binaries: `daemon/*/zig-out/bin/`

## Requirements

- Go 1.23, Zig 0.16, protoc
- KVM-enabled host (tests/integration)
- btrfs (reflink/fork operations)

## Directory Structure

- `cmd/` - Go entry points (coco-gateway, coco-master, coco-node, cococtl)
- `daemon/` - Zig services (coco-visor, coco-agent, coco-fork); Go (coco-net, coco-checkpoint)
- `pkg/` - Shared Go packages
- `proto/` - Protocol buffer definitions
- `ebpf/` - C/eBPF network programs
- `spec/` - Architecture specs (source of truth)
- `test/` - Cross-component tests (unit, integration, e2e, benchmark)
- `configs/` - Configuration templates
- `deploy/` - Kubernetes and deployment manifests

## eBPF Compilation

```bash
cd ebpf && clang -O2 -target bpf -c <file>.bpf.c -o <file>.o -Iheaders
```

## Testing

- `make test-go` - Go tests only
- `make test-zig` - Zig tests only
- Integration tests require docker-compose. Health endpoint: `http://localhost:4747/health`

## Style Rules

- No comments unless requested
- No version numbers in specs
- Follow existing file naming conventions (lowercase with underscores)

## Reference

- Full architecture in `spec/00-overview.md` and `spec/01-folder-structure.md`
- Existing guidance in `CLAUDE.md`
