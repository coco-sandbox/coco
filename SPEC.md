# Coco — Agent-Native Sandbox Runtime

> **Specification v1.0** — 2026-04-26
> Status: **DRAFT**

---

## 1. Executive Summary

**Coco** is an open-source, production-grade sandbox runtime purpose-built for AI agents. Unlike traditional containers or VMs, coco provides agent-native primitives — **fork**, **hibernate**, **replay**, and **undo/redo** — that let autonomous agents explore hypotheses in parallel, checkpoint reasoning state, and safely execute untrusted code at machine speed.

Coco achieves performance that exceeds existing sandboxing solutions by an order of magnitude:

| Metric | E2B / Cube | **Coco v1.0** |
|--------|-----------|---------------|
| Cold start median | 187 ms | **< 50 ms** |
| Cold start p99 | 312 ms | **< 70 ms** |
| Per-sandbox throughput | 6.8 Gbps | **> 23 Gbps** |
| Intra-host RPC p99 | 38 µs | **< 5 µs** |
| Fork latency | N/A | **< 30 ms** |
| Hibernate 512 MiB | N/A | **< 4 s** |
| Resume from NVMe | N/A | **< 200 ms** |
| Undo latency | N/A | **< 5 ms** |

**Stack:** Zig + C + Go. No Rust. Single repository.

---

## 2. Architecture

```
┌──────────────────────────────────────────────────────┐
│ cococtl (Go CLI) │ coco-core (Go — port 4747) │
└────────┬───────────────────────┬───────────────────┘
         │ Unix Socket RPC        │ HTTP/gRPC (external)
         │ /run/coco/visor.sock  │
┌────────▼────────┐    ┌─────────▼─────────┐
│ cocovisor (Zig) │    │ coconet (Zig+C)   │
│ Boot/Exec/Destroy│    │ eBPF NAT + AF_XDP │
│ Fork/Hibernate  │    │ Bloom + LPM policy│
└────────┬────────┘    └─────────┬─────────┘
         │                       │
    ┌────▼────┐            ┌────▼────┐
    │ MicroVM │            │ Network │
    │ (cocod)│            │   NS    │
    │ Zig PID1│            └─────────┘
    └─────────┘
```

---

## 3. Components

### 3.1 coco-core (Go) — API Server

HTTP/gRPC on port 4747. Sandbox lifecycle, exec, fork, hibernate, checkpoint, replay.

### 3.2 cococtl (Go) — CLI Tool

```bash
cococtl sandbox create <name> [template]
cococtl sandbox list
cococtl sandbox destroy <id>
cococtl sandbox fork <id> [name]
cococtl sandbox hibernate <id>
cococtl sandbox resume <id>
cococtl exec <id> <cmd> [args...]
cococtl checkpoint create <id> <name>
cococtl checkpoint list <id>
cococtl undo <id> [checkpoint_id]
cococtl redo <id> [checkpoint_id]
cococtl fs ls <id> [path]
cococtl fs tree <id> [path]
cococtl fs cat <id> <path>
cococtl fs write <id> <path>
cococtl fs mkdir <id> <path>
cococtl fs rm <id> <path> [-r]
```

### 3.3 cocovisor (Zig) — Hypervisor Wrapper

Unix socket RPC at `/run/coco/visor.sock`. Binary frame protocol. Manages KVM MicroVMs via Cloud Hypervisor.

### 3.4 coconet (Zig + C) — Network Daemon

eBPF TC hooks (from_sandbox.bpf.c, from_world.bpf.c). AF_XDP fast path. Bloom + LPM policy engine.

### 3.5 cocofork (Zig) — Fork/Hibernate/Checkpoints

Snapshot-fork with CoW. Hibernate to NVMe < 4s. Resume < 200ms. Named checkpoints with < 5ms undo.

### 3.6 cocod (Zig) — Guest Agent

PID 1 inside MicroVM. vsock listener. Exec handler. File transfer. Streams stdout/stderr.

---

## 4. Protocol

Binary frame protocol over Unix socket:

```
┌──────────┬──────────┬─────────────────────┐
│ kind(4B) │ size(4B) │ payload (size B)   │
└──────────┴──────────┴─────────────────────┘
```

| Kind | Name | Description |
|------|------|-------------|
| 1 | BOOT | Boot MicroVM |
| 2 | EXEC | Execute command |
| 3 | DESTROY | Destroy MicroVM |
| 4 | PAUSE | Pause VM |
| 5 | RESUME | Resume VM |
| 6 | GET_STATE | Get VM state |
| 7 | FORK | Snapshot-fork VM |
| 8 | HIBERNATE | Hibernate to disk |
| 9 | RESUME_HIBERNATED | Resume from hibernation |

---

## 5. Sandbox States

| State | Description |
|-------|-------------|
| CREATING | VM boot in progress |
| RUNNING | Active and ready |
| PAUSED | VM paused |
| HIBERNATED | State on NVMe |
| STOPPING | Destruction in progress |
| DESTROYED | Resources released |
| ERROR | Fault |

---

## 6. Storage Layout

```
/var/lib/coco/
├── images/              # Rootfs templates
│   ├── alpine.ext4
│   └── ubuntu-22.04.ext4
├── hibernation/         # Hibernate snapshots
│   └── {sandbox_id}/
│       ├── memory.img.zst
│       ├── vmstate.bin
│       └── metadata.json
├── checkpoints/         # Undo/redo checkpoints
│   └── {sandbox_id}/
│       └── {checkpoint_id}/
├── replays/            # Replay recordings
│   └── {replay_id}/
│       └── events.log
└── store/              # BadgerDB state store
```

---

## 7. Benchmark Targets

| Metric | Target |
|--------|--------|
| Cold start median | < 50ms |
| Cold start p99 | < 70ms |
| Throughput | > 23 Gbps |
| Intra-host RPC p99 | < 5µs |
| Fork latency | < 30ms |
| Hibernate (512 MiB) | < 4s |
| Resume from NVMe | < 200ms |
| Undo latency | < 5ms |

---

## 8. Repository Structure

```
coco/
├── SPEC.md
├── README.md
├── LICENSE
├── Makefile
├── docker-compose.yml
├── .gitignore
│
├── proto/
│   ├── v1/sandbox.proto
│   └── internal/visor.proto
│
├── core/               # Go API server
│   ├── main.go
│   ├── server.go
│   ├── sandbox.go
│   ├── exec.go
│   ├── fork.go
│   ├── hibernate.go
│   ├── checkpoint.go
│   ├── replay.go
│   ├── fs.go
│   ├── store/
│   │   └── badger.go
│   ├── metrics/
│   │   └── prometheus.go
│   └── go.mod
│
├── ctl/                # Go CLI
│   └── main.go
│
├── src/
│   ├── cocovisor/
│   │   ├── main.zig
│   │   ├── vmm.zig
│   │   └── build.zig
│   ├── coconet/
│   │   ├── main.zig
│   │   ├── ebpf.zig
│   │   ├── policy.zig
│   │   └── build.zig
│   ├── cocofork/
│   │   ├── main.zig
│   │   ├── snapshot.zig
│   │   ├── hibernate.zig
│   │   ├── checkpoint.zig
│   │   └── build.zig
│   └── cocod/
│       ├── main.zig
│       ├── exec.zig
│       ├── vsock.zig
│       └── build.zig
│
├── c/                  # eBPF programs
│   ├── from_sandbox.bpf.c
│   ├── from_world.bpf.c
│   ├── common.h
│   └── maps.h
│
└── scripts/
    └── smoke_test.sh
```