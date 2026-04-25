# Coco Sandbox

> Open-source agent-native sandbox runtime. Hot start < 50ms. 23+ Gbps throughput. Instant fork.

**Stack: Zig + C + Go** — no Rust, single repository.

## Architecture

```
┌─────────────────────────────────────────────┐
│           cococtl (Go CLI)                  │
│           coco-core (Go HTTP/gRPC)          │
└─────────────────────────────────────────────┘
                      │
    ┌─────────────────┼─────────────────┐
    ▼                 ▼                 ▼
cocovisor         coconet           cocofork
(Zig + KVM)       (Zig + eBPF)      (Zig + snapshot)
    │                 │                 │
    └─────────────────┴─────────────────┘
                      │
              ┌───────┴───────┐
              │   MicroVM     │
              │   (cocod)     │
              │   [Zig]        │
              └───────────────┘
```

## Components

| Component | Language | Purpose |
|-----------|----------|---------|
| `core/` | Go | HTTP/gRPC API server |
| `ctl/` | Go | CLI tool |
| `src/cocovisor/` | Zig | KVM MicroVM lifecycle |
| `src/coconet/` | Zig | eBPF network daemon |
| `src/cocofork/` | Zig | Fork/hibernate primitives |
| `src/cocod/` | Zig | Guest agent (inside VM) |
| `c/*.bpf.c` | C | eBPF programs |

## Quick Start

```bash
# Build Go components
cd core && go build -o coco-core .
cd ../ctl && go build -o cococtl .

# Build Zig components
cd src/cocovisor && zig build
cd src/coconet && zig build

# Run
./coco-core &
./cococtl sandbox create test alpine
```

## Performance Targets

| Metric | Target |
|--------|--------|
| Cold start median | < 50ms |
| Cold start p99 | < 70ms |
| Throughput | > 23 Gbps |
| Intra-host RPC p99 | < 5µs |
| Fork latency | < 30ms |

## License

Apache 2.0