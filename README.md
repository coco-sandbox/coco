# Coco Native - Agent-Native Sandbox Runtime

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-0.2.0-blue.svg)](CHANGELOG.md)

Coco is an open-source **agent-native sandbox runtime** that provides hardware-level isolated execution environments for AI agents. Built with Go, Zig, and eBPF for maximum performance and security.

## Why Coco?

Unlike traditional sandbox solutions, Coco is designed from the ground up for AI agent workloads:

| Feature | CubeSandbox | Coco Native |
|---------|-------------|-------------|
| Cold Start | <60ms | <60ms |
| Replay | ❌ | ✅ Full session recording |
| Checkpoint | ❌ | ✅ Named snapshots |
| Fork/Clone | ❌ | ✅ CoW memory cloning |
| Hibernate | ❌ | ✅ Disk suspend |
| Agent-native API | ❌ | ✅ Streaming exec, state inspection |
| E2B Compatible | ✅ | ✅ |

## Quick Start

### Python

```python
from coco import Sandbox

async with Sandbox.create(template="python-3.11") as sb:
    result = await sb.exec("print('hello from coco!')")
    print(result.stdout)
```

### Go

```go
client, _ := coco.NewClient("http://localhost:4747")
sb, _ := client.Sandbox.Create(ctx, &coco.CreateOptions{
    Template: "python-3.11",
})

result, _ := sb.Exec(ctx, &coco.ExecRequest{
    Command: "echo hello",
})
fmt.Println(result.Stdout)
```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Agent Layer (Go)                                       │
│  REST API, Streaming Exec, Cluster Orchestration        │
│  Port 4747: REST API, Port 9090: Prometheus Metrics      │
└─────────────────────────────────────────────────────────┘
                            │ Unix Socket
                            ▼
┌─────────────────────────────────────────────────────────┐
│  Execution Engine (Zig)                                  │
│  Cocovisor - VM Lifecycle, Checkpoint, Fork, Hibernate │
│  Socket: /run/coco/visor.sock                          │
└─────────────────────────────────────────────────────────┘
                            │ KVM
                            ▼
┌─────────────────────────────────────────────────────────┐
│  MicroVM (isolated)                                     │
│  Linux kernel + coco-agent (PID 1)                     │
└─────────────────────────────────────────────────────────┘
                            ▲
                            │ eBPF
┌─────────────────────────────────────────────────────────┐
│  Networking Layer (Go + eBPF)                            │
│  TAP devices, IPAM, SNAT/DNAT, Policies                │
└─────────────────────────────────────────────────────────┘
```

## Features

### Sub-60ms Cold Start
Templates use snapshot cloning for instant VM boot.

### Replay
Record and replay execution sessions for debugging and retry.

### Checkpoint
Named snapshots for undo/redo and branching.

### Fork
Clone running sandboxes for parallel exploration with CoW memory.

### Hibernate
Suspend VMs to disk for ultra-fast resume.

### eBPF Networking
Kernel-level network isolation with SNAT/DNAT and allow/deny policies.

## Installation

```bash
# Clone
git clone https://github.com/coco-sandbox/coco.git
cd coco

# Build
make all

# Run
./cmd/coco-core/coco-core
```

## API Reference

See [docs/api.md](docs/api.md) for full API reference.

## Architecture

See [docs/architecture.md](docs/architecture.md) for detailed architecture.

## License

Apache 2.0
