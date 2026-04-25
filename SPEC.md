# Coco — Agent-Native Sandbox Runtime Specification

> **Specification** — Draft for Development
> Status: **DESIGN IN PROGRESS**
> Supersedes: CubeSandbox

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Architecture](#2-architecture)
3. [Component Specifications](#3-component-specifications)
4. [Performance Targets](#4-performance-targets)
5. [Security Model](#5-security-model)
6. [Network Architecture](#6-network-architecture)
7. [Storage Architecture](#7-storage-architecture)
8. [API Specification](#8-api-specification)
9. [Observability](#9-observability)
10. [Multi-Node Clustering](#10-multi-node-clustering)
11. [Developer Experience](#11-developer-experience)
12. [Error Handling](#12-error-handling)
13. [Testing Strategy](#13-testing-strategy)
14. [Repository Structure](#14-repository-structure)

---

## 1. Executive Summary

### 1.1 Vision

**Coco** is the next-generation sandbox runtime purpose-built for AI agents. Unlike traditional sandbox solutions (Docker, Firecracker, CubeSandbox), Coco introduces **agent-native primitives** that enable autonomous agents to:

- **Fork** — Create instant copies of running environments for parallel hypothesis exploration
- **Hibernate** — Suspend to NVMe in seconds, resume in milliseconds
- **Replay** — Record and replay entire execution sessions for debugging
- **Undo/Redo** — Rollback to any checkpoint with millisecond latency

### 1.2 Competitive Analysis

| Metric | CubeSandbox | Docker | Firecracker | **Coco (dev)** |
|--------|-------------|--------|------------|---------------|
| Cold start median | 60ms | 200ms | 150ms | **< 30ms** |
| Cold start p99 | 137ms | 500ms | 300ms | **< 50ms** |
| Fork latency | N/A | N/A | N/A | **< 15ms** |
| Hibernate (512 MiB) | N/A | N/A | N/A | **< 2s** |
| Resume from NVMe | N/A | N/A | N/A | **< 100ms** |
| Undo latency | N/A | N/A | N/A | **< 2ms** |
| Per-sandbox throughput | 5 Gbps | 10 Gbps | 8 Gbps | **> 30 Gbps** |
| Intra-host RPC p99 | 38 µs | N/A | 20 µs | **< 2 µs** |
| Memory overhead (per sandbox) | <5 MB | 50 MB | 5 MB | **< 3 MB** |
| Network isolation | eBPF TC | namespaces | none | **eBPF XDP + TC** |
| Agent primitives | None | None | None | **Full fork/hibernate/replay** |
| Cluster support | Yes | Swarm | No | **Advanced sharding** |
| E2B compatible | Yes | No | No | **Yes (drop-in)** |

### 1.3 Technology Stack

- **Primary Language**: Zig (for performance-critical components)
- **Systems Language**: C (for eBPF programs)
- ** orchestration**: Go (for API server and CLI)
- **No Rust** — Avoiding Rust ecosystem fragmentation and build complexity

### 1.4 Design Principles

1. **Performance First** — Every millisecond counts. Optimize hot paths aggressively.
2. **Agent-Native** — Build primitives that AI agents actually need, not generic VM features.
3. **Zero-Cost Abstraction** — Use Zig's comptime to eliminate runtime overhead.
4. **Defense in Depth** — Hardware isolation + network isolation + capability stripping.
5. **Operational Simplicity** — One binary to run, zero config for 95% of use cases.

---

## 2. Architecture

### 2.1 System Overview

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                    HOST                                          │
│                                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────────┐ │
│  │                         coco-core (Go) — Port 4747/4748                    │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │ │
│  │  │ HTTP Server │  │ gRPC Server │  │ State Store │  │ Cluster Mgr    │   │ │
│  │  │ (REST API)  │  │ ( Streaming)│  │ (BadgerDB)  │  │ (Leader Election)│   │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────────┘   │ │
│  └───────────────────────────────┬─────────────────────────────────────────────┘ │
│                                │                                                  │
│                    ┌───────────▼────────────┐                                   │
│                    │   cocovisor (Zig)      │                                   │
│                    │   /run/coco/visor.sock │                                   │
│                    │   - Boot/Destroy       │                                   │
│                    │   - Fork/Hibernate     │                                   │
│                    │   - Pause/Resume       │                                   │
│                    │   - Exec streaming     │                                   │
│                    └───────────┬────────────┘                                   │
│                                │                                                │
│              ┌─────────────────┼─────────────────┐                              │
│              │                 │                 │                              │
│              ▼                 ▼                 ▼                              │
│    ┌─────────────────┐ ┌─────────────┐ ┌──────────────────┐                  │
│    │ cocofork (Zig)  │ │ coconet (Zig)│ │ cocovisor (Zig) │                  │
│    │ - Snapshot-fork │ │ - eBPF XDP  │ │ - VM lifecycle  │                  │
│    │ - Hibernate    │ │ - NAT       │ │ - Cloud Hyper.  │                  │
│    │ - Checkpoints  │ │ - Policies  │ │ - Vsock CID    │                  │
│    │ - Replay       │ │ - AF_XDP    │ │                 │                  │
│    └────────┬────────┘ └──────┬──────┘ └─────────────────┘                  │
│             │                 │                                                │
│             │     ┌──────────▼──────────┐                                     │
│             │     │   cocod (Zig)       │                                     │
│             │     │   PID 1 inside VM  │                                     │
│             │     │   - vsock exec     │                                     │
│             │     │   - FS ops         │                                     │
│             │     │   - Signal forward │                                     │
│             │     └─────────────────────┘                                     │
│             │                                                                   │
│    ┌────────▼────────┐                                                        │
│    │   MicroVM       │◄────────────────────────────────┐                      │
│    │ ┌────────────┐  │                                 │                      │
│    │ │ cocod      │◄─┤ vsock CID                     │                      │
│    │ │ (PID 1)    │  │                                 │                      │
│    │ └────────────┘  │                                 │                      │
│    └─────────────────┘                                 │                      │
│                                                         │                      │
│  ┌──────────────────────────────────────────────────────▼───────────────────┐  │
│  │                    coconet (Zig + C)                                    │  │
│  │  ┌────────────────────────────────────────────────────────────────────┐ │  │
│  │  │                    eBPF Programs                                 │ │  │
│  │  │  from_sandbox (TC egress) — SNAT, policy, session tracking    │ │  │
│  │  │  from_world (TC ingress) — DNAT, port mapping, reverse NAT    │ │  │
│  │  │  from_envoy (XDP) — Overlay traffic, early reject             │ │  │
│  │  └────────────────────────────────────────────────────────────────────┘ │  │
│  │  ┌────────────────────────────────────────────────────────────────────┐ │  │
│  │  │                    AF_XDP Fast Path                               │ │  │
│  │  │  Intel E810 / Mellanox ConnectX — Direct DMA, zero-copy        │ │  │
│  │  └────────────────────────────────────────────────────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                                  │
│  Directories:                                                                   │
│  /var/lib/coco/          — Images, snapshots, checkpoints, store              │
│  /run/coco/              — Runtime sockets and pidfiles                         │
│  /sys/fs/bpf/coco/      — Pinned eBPF maps                                     │
└──────────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Component Responsibilities

| Component | Language | Binary | Responsibility |
|-----------|----------|--------|----------------|
| `coco-core` | Go | `coco-core` | HTTP/gRPC API, cluster orchestration, state management, rate limiting |
| `cococtl` | Go | `cococtl` | CLI for operators and agents |
| `cocovisor` | Zig | `cocovisor` | Unix socket RPC server, VM lifecycle via Cloud Hypervisor |
| `coconet` | Zig + C | `coconet` | eBPF/XDP network, NAT, policies, AF_XDP fast path |
| `cocofork` | Zig | `cocofork` | Snapshot-fork, hibernate, checkpoints, replay |
| `cocod` | Zig | `cocod` | PID 1 inside MicroVM, vsock exec handler, fs ops |
| `cocogate` | Go | `cocogate` | API gateway with auth, rate limiting, load balancing |

### 2.3 Data Flow

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              Request Flow                                        │
└─────────────────────────────────────────────────────────────────────────────────┘

Agent (Python/JS SDK)
        │
        ▼
┌───────────────────┐
│  coco-core (Go)  │  ◄── HTTP/gRPC (port 4747/4748)
│  - Validate req   │
│  - Check auth     │
│  - Rate limit     │
│  - Route to node  │
└────────┬──────────┘
         │
         │ (if local)
         ▼
┌───────────────────────┐
│  cocovisor (Zig)    │  ◄── Unix socket /run/coco/visor.sock
│  - Boot/Destroy     │
│  - Fork/Hibernate   │
│  - Exec streaming   │
└───────────┬───────────┘
            │
            ▼
┌───────────────────────┐     ┌───────────────────┐
│  Cloud Hypervisor    │────►│    MicroVM        │
│  (ch-remote)         │     │  ┌─────────────┐  │
│                      │     │  │ cocod (PID1)│  │
│  - KVM MicroVM       │     │  │ vsock:4747   │  │
│  - vsock CID alloc  │     │  └─────────────┘  │
└───────────────────────┘     └───────────────────┘
                                    │
                                    │ vsock
                                    ▼
┌────────────────────────────────────────────────────────────────┐
│                    coconet (eBPF)                              │
│                                                                 │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐   │
│  │ from_sandbox│    │ from_world   │    │ from_envoy  │   │
│  │ (TC egress) │    │ (TC ingress) │    │ (XDP)       │   │
│  │ - SNAT      │    │ - DNAT       │    │ - DNAT      │   │
│  │ - Policy    │    │ - Rev NAT    │    │ - Early rej │   │
│  └──────────────┘    └──────────────┘    └──────────────┘   │
│         │                   │                   │              │
│         ▼                   ▼                   ▼              │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │              Physical NIC / vswitch                     │  │
│  └─────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────┐
│                            Response Flow                                         │
└─────────────────────────────────────────────────────────────────────────────────┘

cocod stdout/stderr
         │ vsock
         ▼
cocovisor (frame)
         │ Unix socket
         ▼
coco-core (stream)
         │ HTTP SSE / gRPC stream
         ▼
Agent SDK (callback)
```

### 2.4 IPC Mechanisms

| Path | Protocol | Participants | Purpose |
|------|----------|--------------|---------|
| `/run/coco/visor.sock` | Binary frame (Unix) | coco-core ↔ cocovisor | VM lifecycle RPC |
| `vsock CID:4747` | Custom framing | cocovisor ↔ cocod | Exec commands, fs ops |
| `port 4747` | HTTP/1.1 | Agent ↔ coco-core | REST API |
| `port 4748` | gRPC | Agent ↔ coco-core | Streaming API |
| `port 4749` | HTTP/2 | Agent ↔ cocogate | Gateway with auth |
| `port 9090` | Prometheus | Metrics scraper | Prometheus metrics |
| `port 4317` | OTLP/gRPC | Trace collector | OpenTelemetry |

---

## 3. Component Specifications

### 3.1 coco-core (Go) — API Server and Cluster Manager

**Binary:** `coco-core`
**Ports:** 4747 (HTTP), 4748 (gRPC)
**Dependencies:** BadgerDB, Linux KVM

#### 3.1.1 Core Responsibilities

1. **HTTP/gRPC Server** — Handle all external API requests
2. **State Management** — Persist sandbox state in BadgerDB
3. **Cluster Coordination** — Leader election, node health, sharding
4. **Rate Limiting** — Per-tenant, per-endpoint rate limits
5. **Authentication** — mTLS validation, JWT tokens
6. **Metrics** — Prometheus scrape endpoint

#### 3.1.2 Directory Structure

```
core/
├── main.go                 # Entry point, signal handling
├── server.go               # HTTP/gRPC server setup
├── router.go               # Request routing
├── middleware/
│   ├── auth.go            # mTLS + JWT authentication
│   ├── rate_limit.go      # Token bucket rate limiting
│   ├── logging.go         # Structured request logging
│   └── tracing.go         # OpenTelemetry integration
├── handlers/
│   ├── sandbox.go         # CRUD handlers
│   ├── exec.go            # Exec streaming
│   ├── fork.go            # Fork handler
│   ├── hibernate.go       # Hibernate/resume
│   ├── checkpoint.go      # Checkpoint management
│   ├── replay.go          # Replay control
│   ├── fs.go              # File operations
│   └── health.go          # Health checks
├── cluster/
│   ├── node.go            # Node registration
│   ├── sharding.go        # Consistent hashing
│   ├── election.go        # Leader election (Raft)
│   └── discovery.go       # Node discovery
├── store/
│   ├── badger.go          # BadgerDB wrapper
│   ├── sandbox.go         # Sandbox state CRUD
│   └── checkpoint.go      # Checkpoint metadata
├── visor/
│   ├── client.go          # Unix socket client
│   ├── protocol.go        # Frame encoding/decoding
│   └── pool.go            # Connection pooling
├── metrics/
│   ├── prometheus.go      # Prometheus exporter
│   └── custom.go          # Custom metrics
└── config/
    └── config.go          # Configuration loading
```

#### 3.1.3 HTTP Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness probe |
| GET | `/ready` | Readiness probe (checks cocovisor, coconet) |
| POST | `/v1/sandboxes` | Create sandbox |
| GET | `/v1/sandboxes` | List sandboxes (paginated) |
| GET | `/v1/sandboxes/:id` | Get sandbox details |
| DELETE | `/v1/sandboxes/:id` | Destroy sandbox |
| POST | `/v1/sandboxes/:id/exec` | Execute command (streaming) |
| POST | `/v1/sandboxes/:id/fork` | Fork sandbox |
| POST | `/v1/sandboxes/:id/hibernate` | Hibernate sandbox |
| POST | `/v1/sandboxes/:id/resume` | Resume from hibernate |
| POST | `/v1/sandboxes/:id/checkpoint` | Create checkpoint |
| GET | `/v1/sandboxes/:id/checkpoints` | List checkpoints |
| POST | `/v1/sandboxes/:id/undo` | Undo to checkpoint |
| POST | `/v1/sandboxes/:id/redo` | Redo checkpoint |
| POST | `/v1/sandboxes/:id/replay/start` | Start replay recording |
| POST | `/v1/sandboxes/:id/replay/stop` | Stop replay recording |
| GET | `/v1/sandboxes/:id/replay/events` | Get replay events |
| GET | `/v1/sandboxes/:id/fs/ls` | List directory |
| GET | `/v1/sandboxes/:id/fs/tree` | Tree view |
| GET | `/v1/sandboxes/:id/fs/cat` | Read file |
| PUT | `/v1/sandboxes/:id/fs/write` | Write file |
| POST | `/v1/sandboxes/:id/fs/mkdir` | Create directory |
| DELETE | `/v1/sandboxes/:id/fs/rm` | Remove file/directory |
| POST | `/v1/sandboxes/:id/pause` | Pause sandbox |
| POST | `/v1/sandboxes/:id/resume` | Resume sandbox |
| GET | `/v1/nodes` | List cluster nodes |
| GET | `/v1/metrics` | Prometheus metrics |

#### 3.1.4 gRPC Services

```protobuf
service CocoService {
  // Sandbox lifecycle
  rpc CreateSandbox(CreateSandboxRequest) returns (CreateSandboxResponse);
  rpc GetSandbox(GetSandboxRequest) returns (Sandbox);
  rpc ListSandboxes(ListSandboxesRequest) returns (ListSandboxesResponse);
  rpc DeleteSandbox(DeleteSandboxRequest) returns (DeleteSandboxResponse);
  
  // Execution
  rpc Exec(ExecRequest) returns (stream ExecChunk);
  rpc ExecInteractive(ExecRequest) returns (stream ExecChunk);
  
  // Fork/Hibernate
  rpc Fork(ForkRequest) returns (ForkResponse);
  rpc Hibernate(HibernateRequest) returns (HibernateResponse);
  rpc Resume(ResumeRequest) returns (ResumeResponse);
  
  // Checkpoints
  rpc CreateCheckpoint(CreateCheckpointRequest) returns (Checkpoint);
  rpc ListCheckpoints(ListCheckpointsRequest) returns (ListCheckpointsResponse);
  rpc Undo(UndoRequest) returns (UndoResponse);
  rpc Redo(RedoRequest) returns (RedoResponse);
  
  // Replay
  rpc StartReplay(StartReplayRequest) returns (StartReplayResponse);
  rpc StopReplay(StopReplayRequest) returns (StopReplayResponse);
  rpc GetReplayEvents(GetReplayEventsRequest) returns (stream ReplayEvent);
  
  // File operations
  rpc ReadFile(ReadFileRequest) returns (stream FileChunk);
  rpc WriteFile(WriteFileRequest) returns (WriteFileResponse);
  rpc ListDirectory(ListDirectoryRequest) returns (ListDirectoryResponse);
}

service VisorService {
  rpc Boot(BootRequest) returns (BootResponse);
  rpc Exec(ExecRequest) returns (stream ExecChunk);
  rpc Destroy(DestroyRequest) returns (DestroyResponse);
  rpc GetState(GetStateRequest) returns (GetStateResponse);
  rpc Pause(PauseRequest) returns (PauseResponse);
  rpc Resume(ResumeRequest) returns (ResumeResponse);
  rpc Fork(ForkRequest) returns (ForkResponse);
  rpc Hibernate(HibernateRequest) returns (HibernateResponse);
  rpc ResumeHibernated(ResumeHibernatedRequest) returns (ResumeHibernatedResponse);
}
```

#### 3.1.5 State Machine

```
                    ┌──────────────┐
                    │   CREATING   │
                    └──────┬───────┘
                           │ VM booted
                           ▼
                    ┌──────────────┐
              ┌────►│   RUNNING    │◄────┐
              │     └──────┬───────┘     │
              │            │             │
         pause│            │             │resume
              │            │             │
              │            ▼             │
              │     ┌──────────────┐     │
              └─────│   PAUSED    │─────┘
                    └──────┬───────┘
                           │
                    hibernate│
                           ▼
                    ┌──────────────┐
                    │  HIBERNATED  │
                    └──────┬───────┘
                           │ resume
                           ▼
                    ┌──────────────┐
                    │  RESUMING    │
                    └──────┬───────┘
                           │ VM running
                           ▼
                    ┌──────────────┐
                    │   RUNNING    │
                    └──────┬───────┘
                           │
                     destroy│
                           ▼
                    ┌──────────────┐
                    │ DESTROYING   │
                    └──────┬───────┘
                           │ cleanup done
                           ▼
                    ┌──────────────┐
                    │  DESTROYED   │
                    └──────────────┘
```

#### 3.1.6 Rate Limiting

| Tier | Requests/min | Burst | Priority |
|------|-------------|-------|----------|
| Free | 60 | 10 | Low |
| Basic | 600 | 100 | Medium |
| Pro | 6000 | 1000 | High |
| Enterprise | Unlimited | Unlimited | Critical |

Rate limits are enforced via token bucket algorithm:
- Each tenant has a bucket with `rate` tokens per second
- Burst allows up to `capacity` tokens
- 429 response with `Retry-After` header when exceeded

### 3.2 cococtl (Go) — CLI Tool

**Binary:** `cococtl`

```
cococtl [global flags] <command> [arguments]

Global flags:
  --host string       API host (default "localhost:4747")
  --timeout int      Request timeout in seconds (default 30)
  --output string    Output format: text, json, yaml (default "text")
  --verbose          Enable verbose output
  --config string    Config file path (default "~/.coco/config.yaml")

Commands:
  sandbox            Manage sandboxes
    create          Create a new sandbox
    list            List all sandboxes
    get             Get sandbox details
    delete          Delete a sandbox
    fork            Fork a sandbox
    hibernate       Hibernate a sandbox
    resume          Resume a sandbox
    pause           Pause a sandbox
    unpause         Unpause a sandbox
    
  exec               Execute commands
    run             Run a command (shorthand)
    interactive     Run interactive shell
    
  fs                 File operations
    ls              List directory
    tree            Tree view
    cat             Read file
    write           Write file
    mkdir           Create directory
    rm              Remove file/directory
    upload          Upload file
    download        Download file
    
  checkpoint         Checkpoint operations
    create          Create checkpoint
    list            List checkpoints
    undo            Undo to checkpoint
    redo            Redo checkpoint
    delete          Delete checkpoint
    
  replay             Replay operations
    start           Start recording
    stop            Stop recording
    events          List replay events
    
  node               Cluster management
    list            List nodes
    stats           Node statistics
    
  template           Template management
    list            List templates
    create          Create template
    delete          Delete template

Examples:
  # Create sandbox
  cococtl sandbox create my-agent --template alpine --memory 512 --vcpus 2
  
  # Execute command
  cococtl exec run sandbox-abc "python3 -c 'print(1+1)'"
  
  # Fork for parallel exploration
  cococtl sandbox fork sandbox-abc experiment-1
  cococtl sandbox fork sandbox-abc experiment-2
  
  # Create checkpoint before risky operation
  cococtl checkpoint create sandbox-abc "before-mutation"
  
  # Undo if something goes wrong
  cococtl checkpoint undo sandbox-abc before-mutation
  
  # Hibernate during idle
  cococtl sandbox hibernate sandbox-abc
  
  # Resume when needed
  cococtl sandbox resume sandbox-abc
```

### 3.3 cocovisor (Zig) — Hypervisor Wrapper

**Binary:** `cocovisor`
**Socket:** `/run/coco/visor.sock`
**Language:** Zig 0.14.0+

#### 3.3.1 Responsibilities

1. **Unix Socket Server** — Handle boot/exec/destroy requests from coco-core
2. **VM Lifecycle** — Create, start, stop, pause, resume VMs via Cloud Hypervisor
3. **VSOCK CID Management** — Allocate and track vsock context IDs
4. **Metrics Collection** — Record boot, exec, fork, hibernate latencies

#### 3.3.2 Binary Protocol

Frame format: `[kind:u32][size:u32][payload:u8[size]]`

**Request Types:**

| Kind | Name | Payload |
|------|------|---------|
| 1 | BOOT | `BootRequest` + variable-length strings |
| 2 | EXEC | `ExecRequest` + variable-length strings |
| 3 | DESTROY | sandbox_id bytes |
| 4 | PAUSE | sandbox_id bytes |
| 5 | RESUME | sandbox_id bytes |
| 6 | GET_STATE | sandbox_id bytes |
| 7 | FORK | parent_id_len(4) + parent_id + child_name_len(4) + child_name |
| 8 | HIBERNATE | sandbox_id bytes |
| 9 | RESUME_HIBERNATED | sandbox_id bytes |
| 10 | LIST_VMS | empty |

**Response Types:**

| Kind | Name | Payload |
|------|------|---------|
| 100 | OK | empty or `{}` |
| 101 | BOOT | `[vsock_cid:u32][pid:u32][state:u32]` |
| 102 | EXEC | streaming `ExecStreamChunk` frames |
| 103 | DESTROY | empty |
| 104 | PAUSE | empty |
| 105 | RESUME | empty |
| 106 | GET_STATE | `[state:u32][pid:u32][vsock_cid:u32]` |
| 107 | FORK | `[child_vsock_cid:u32][child_pid:u32][duration_ms:u32]` |
| 108 | HIBERNATE | `[duration_ms:u32]` |
| 109 | RESUME_HIBERNATED | `[vsock_cid:u32][pid:u32]` |
| 110 | LIST_VMS | `[count:u32][vm_info*]` |
| 199 | ERROR | `[msg_len:u32][msg:u8[msg_len]]` |

#### 3.3.3 BootRequest Structure

```zig
const BootRequest = extern struct {
    rootfs_path_len: u32,
    memory_mb: u32,
    vcpu_count: u32,
    kernel_path_len: u32,
    initrd_path_len: u32,
    sandbox_id_len: u32,
    vsock_port: u32,
    enable_vsock: u32,
    enable_network: u32,
    padding: u32,
};
// Followed by: sandbox_id || rootfs_path || kernel_path || initrd_path
```

#### 3.3.4 ExecStreamChunk

```zig
const ExecStreamChunk = struct {
    stream_type: u32, // 1=stdout, 2=stderr, 3=exit, 4=signal
    data_len: u32,
    exit_code: u32,
};
// Followed by: data bytes
```

#### 3.3.5 VM Management

```zig
const VMState = enum(u32) {
    UNKNOWN = 0,
    CREATING = 1,
    RUNNING = 2,
    PAUSED = 3,
    STOPPING = 4,
    STOPPED = 5,
    HIBERNATED = 6,
    ERROR = 7,
};

const VM = struct {
    id: []u8,
    state: VMState,
    pid: u32,
    vsock_cid: u32,
    config: VMConfig,
    created_at: u64,
};
```

### 3.4 coconet (Zig + C) — Network Daemon

**Binary:** `coconet`
**Language:** Zig 0.14.0+ with C eBPF programs

#### 3.4.1 Responsibilities

1. **Sandbox Network Namespace** — Create and manage per-sandbox network namespaces
2. **eBPF Program Loading** — Load and attach TC/XDP eBPF programs
3. **NAT Translation** — SNAT for egress, DNAT for ingress
4. **Policy Engine** — Bloom filter + LPM for allow/deny rules
5. **AF_XDP Fast Path** — Zero-copy packet processing for high-throughput workloads
6. **Port Mapping** — Expose services running inside sandboxes
7. **Session Tracking** — Stateful connection tracking for NAT

#### 3.4.2 eBPF Programs

**from_sandbox.bpf.c** — Egress hook (sandbox → world)
- Attach: TC egress on each sandbox TAP device
- Functions:
  - SNAT: Replace source IP with host IP from NAT pool
  - Policy check: Bloom filter pre-filter, LPM trie exact match
  - Session creation: Create NAT session in `egress_sessions` map
  - ARP proxy: Respond to ARP requests for gateway IP

**from_world.bpf.c** — Ingress hook (world → sandbox)
- Attach: TC ingress on host NIC
- Functions:
  - DNAT: Replace destination IP with sandbox internal IP
  - Session lookup: Reverse NAT from `ingress_sessions` map
  - Port mapping: Static port forward to sandbox services

**from_envoy.bpf.c** — Overlay traffic (XDP)
- Attach: XDP on overlay interface (cube-dev)
- Functions:
  - Early reject: Drop unsolicited inbound traffic
  - DNAT: Rewrite overlay IP to sandbox internal IP

#### 3.4.3 BPF Maps

| Map | Type | Key | Value | Purpose |
|-----|------|-----|-------|---------|
| `sandbox_ip_to_ifindex` | Hash | IPv4 | ifindex | IP to device lookup |
| `ifindex_to_meta` | Hash | ifindex | Sandbox metadata | Device to metadata |
| `egress_sessions` | Hash | 5-tuple | NAT session | Outbound connection tracking |
| `ingress_sessions` | Hash | 5-tuple | Reverse metadata | Inbound connection tracking |
| `snat_pool` | Array[4] | index | SNAT IP entry | SNAT IP pool |
| `allow_rules` | Hash-of-Maps | ifindex | LPM trie | Per-sandbox allow list |
| `deny_rules` | Hash-of-Maps | ifindex | LPM trie | Per-sandbox deny list |
| `port_mapping` | Hash | host port | (ifindex, sandbox port) | Inbound port forward |
| `local_port_mapping` | Hash | (ifindex, port) | host port | Outbound optimization |

#### 3.4.4 Network Configuration

```zig
const NetworkConfig = struct {
    host_interface: []u8,      // e.g., "eth0"
    overlay_interface: []u8,   // e.g., "cube-dev"
    host_ip: [4]u8,            // e.g., {192, 168, 1, 1}
    gateway_ip: [4]u8,         // e.g., {169, 254, 68, 1}
    sandbox_ip_base: [4]u8,    // e.g., {169, 254, 68, 2}
    snat_pool_start: u16,      // e.g., 30000
    snat_pool_end: u16,       // e.g., 60000
    mtu: u16,                 // e.g., 1500
};
```

#### 3.4.5 Policy Engine

```
Evaluation Order:
1. Is destination the gateway?     → Allow (internal traffic)
2. Is destination private/link-local? → Deny (security)
3. Does allow_rules match?         → Allow (explicit allow)
4. Does deny_rules match?          → Deny (explicit deny)
5. Default                         → Allow

Always-Denied CIDRs:
- 10.0.0.0/8         (private)
- 127.0.0.0/8        (loopback)
- 169.254.0.0/16     (link-local)
- 172.16.0.0/12      (private)
- 192.168.0.0/16     (private)
- 224.0.0.0/4        (multicast)
- 240.0.0.0/4        (reserved)
```

#### 3.4.6 AF_XDP Fast Path

For workloads requiring >10 Gbps throughput:

```c
struct xdp_desc {
    __u64 addr;
    __u32 len;
    __u32 options;
};

// Zero-copy path for Intel E810 / Mellanox ConnectX
// Bypasses kernel networking stack entirely
// Direct DMA to/from userspace UMEM
```

### 3.5 cocofork (Zig) — Fork/Hibernate/Checkpoints

**Binary:** `cocofork` (linked into cocovisor)
**Language:** Zig 0.14.0+

#### 3.5.1 Responsibilities

1. **Snapshot-Fork** — Create VM copies via copy-on-write
2. **Hibernate** — Suspend VM to NVMe with compression
3. **Resume** — Restore VM from NVMe snapshot
4. **Checkpoints** — Named snapshots for undo/redo
5. **Replay** — Record and replay execution sessions

#### 3.5.2 Snapshot-Fork Algorithm

```
1. Pause parent VM
2. Create memory snapshot (dirty pages only)
3. Allocate new vsock CID
4. Create child VM with shared memory (CoW)
5. Resume both VMs
6. Track CoW pages with userfaultfd

Total: < 15ms for 512 MiB VM
```

**Implementation Details:**

```zig
const ForkOptions = struct {
    memory_mb: u32,
    copy_on_write: bool = true,
    share_page_tables: bool = true,
    track_dirty_pages: bool = true,
};

const ForkResult = struct {
    child_sandbox_id: []u8,
    child_vsock_cid: u32,
    child_pid: u32,
    duration_ns: u64,
    pages_copied: u64,
    pages_shared: u64,
};
```

#### 3.5.3 Hibernate Algorithm

```
1. Pause VM
2. Serialize CPU state (registers, FPU, CR3, etc.)
3. Walk page tables, collect dirty pages
4. Compress pages (Zstandard level 3)
5. Write sequentially to NVMe (O_DIRECT)
6. Store metadata (sandbox_id, memory, timestamps)
7. Release VM resources

Total: < 2s for 512 MiB
```

**Implementation Details:**

```zig
const HibernateOptions = struct {
    compression: enum { none, lz4, zstd } = .zstd,
    compression_level: u32 = 3,
    async_write: bool = true,
    verify_checksum: bool = true,
    include_clean_pages: bool = false, // Skip clean pages (CoW optimization)
};

const HibernateResult = struct {
    snapshot_id: []u8,
    snapshot_path: []u8,
    memory_mb: u32,
    compressed_size_mb: f32,
    duration_ns: u64,
    compression_ratio: f32,
    pages_total: u64,
    pages_dirty: u64,
};
```

#### 3.5.4 Resume Algorithm

```
1. Read metadata from NVMe
2. Validate checksum
3. Decompress pages (parallel workers)
4. Restore CPU state
5. Create new VM with restored memory
6. Resume VM

Total: < 100ms for 512 MiB
```

**Implementation Details:**

```zig
const ResumeOptions = struct {
    preallocate_memory: bool = true,
    parallel_decompress: bool = true,
    worker_count: u32 = 4,
};

const ResumeResult = struct {
    sandbox_id: []u8,
    vsock_cid: u32,
    pid: u32,
    duration_ns: u64,
    memory_mb: u32,
};
```

#### 3.5.5 Checkpoint System

**Design:** Chain-based with diff compression

```
Checkpoint Chain:
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│  Root   │───►│ Check-1 │───►│ Check-2 │───►│ Check-3 │
│ (clean) │    │ (+diff) │    │ (+diff) │    │ (+diff) │
└─────────┘    └─────────┘    └─────────┘    └─────────┘

Undo: Walk backward, apply inverse diff
Redo: Walk forward, apply diff
```

**Operations:**

```zig
const Checkpoint = struct {
    id: []u8,
    parent_id: ?[]u8,
    created_at: u64,
    description: []u8,
    memory_diff_size_mb: f32,
    state_size_kb: u32,
};

const CheckpointOptions = struct {
    name: []u8,
    description: []u8 = "",
    include_memory: bool = true,
    include_state: bool = true,
    compress_diff: bool = true,
};

const UndoResult = struct {
    checkpoint_id: []u8,
    duration_ns: u64,
    pages_restored: u64,
};

const RedoResult = struct {
    checkpoint_id: []u8,
    duration_ns: u64,
    pages_applied: u64,
};
```

#### 3.5.6 Replay System

**Design:** Event-based recording with deterministic replay

```
Recorded Events:
- exec(command, args, env, cwd, exit_code, stdout, stderr)
- fs_read(path, offset, length, data)
- fs_write(path, offset, data)
- fs_create(path, mode)
- fs_delete(path)
- fork(child_id)
- hibernate()
- resume()

Replay Modes:
- record: Capture all events
- replay: Replay from recorded events
- diff: Compare replay with original
```

**Operations:**

```zig
const ReplaySession = struct {
    id: []u8,
    sandbox_id: []u8,
    started_at: u64,
    event_count: u64,
    total_bytes: u64,
};

const ReplayEvent = struct {
    timestamp_ns: u64,
    event_type: enum { exec, fs_read, fs_write, fs_create, fs_delete, fork },
    event_data: []u8,
};

const ReplayOptions = struct {
    mode: enum { record, replay, diff },
    start_event: u64 = 0,
    end_event: u64 = max,
    speed: f32 = 1.0, // 1.0 = real-time, 2.0 = 2x speed
};
```

### 3.6 cocod (Zig) — Guest Agent

**Binary:** `cocod` (initrd/initramfs)
**Language:** Zig 0.14.0+

#### 3.6.1 Responsibilities

1. **PID 1** — Init process inside MicroVM
2. **vsock Listener** — Receive exec commands from host
3. **Command Execution** — Spawn processes, stream output
4. **File Operations** — Handle read/write/mkdir/rm from host
5. **Signal Forwarding** — Forward SIGTERM/SIGKILL to child processes
6. **Zombie Reaping** — Reap exited child processes

#### 3.6.2 vsock Protocol

**Host → cocod:**

```
[msg_type:u32][cmd_len:u32][cmd:u8[cmd_len]]
    [args_len:u32][args:u8[args_len]]
    [env_len:u32][env:u8[env_len]]
    [working_dir_len:u32][working_dir:u8[working_dir_len]]

msg_type:
  1 = exec
  2 = fs_read
  3 = fs_write
  4 = fs_mkdir
  5 = fs_delete
  6 = fs_list
  7 = ping
```

**cocod → host:**

```
[msg_type:u32][payload_len:u32][payload:u8[payload_len]]

msg_type:
  1 = stdout
  2 = stderr
  3 = exit_code
  4 = fs_data
  5 = fs_status
  6 = pong
```

#### 3.6.3 Execution Model

```zig
const ExecOptions = struct {
    command: []u8,
    args: [][]u8,
    env: [][]u8,
    working_dir: []u8,
    timeout_ns: u64 = 0, // 0 = no timeout
    stdin: ?[]u8 = null,
};

const ExecResult = struct {
    exit_code: i32,
    stdout: []u8,
    stderr: []u8,
    duration_ns: u64,
    oom_killed: bool,
    signal: ?u32,
};
```

#### 3.6.4 File Operations

```zig
const FSReadRequest = struct {
    path: []u8,
    offset: u64 = 0,
    length: u64 = max,
};

const FSWriteRequest = struct {
    path: []u8,
    offset: u64 = 0,
    data: []u8,
    create: bool = false,
    mode: u16 = 0o644,
};

const FSMkdirRequest = struct {
    path: []u8,
    mode: u16 = 0o755,
    recursive: bool = true,
};

const FSDeleteRequest = struct {
    path: []u8,
    recursive: bool = false,
};

const FSListRequest = struct {
    path: []u8,
    recursive: bool = false,
};

const FSEntry = struct {
    name: []u8,
    is_dir: bool,
    size: u64,
    mode: u16,
    modified_at: u64,
};
```

### 3.7 cocogate (Go) — API Gateway

**Binary:** `cocogate`
**Port:** 4749 (HTTP), 4750 (gRPC)
**Purpose:** Auth, rate limiting, load balancing, protocol translation

#### 3.7.1 Responsibilities

1. **Authentication** — Validate JWT tokens, mTLS certificates
2. **Rate Limiting** — Per-tenant, per-endpoint rate limits
3. **Load Balancing** — Round-robin, least-connections, consistent hashing
4. **Protocol Translation** — HTTP ↔ gRPC ↔ WebSocket
5. **Request Validation** — Schema validation, input sanitization
6. **Response Transformation** — Convert internal format to SDK format

#### 3.7.2 Features

- **Connection pooling** — Reuse connections to backend
- **Circuit breaker** — Auto-disable failing backends
- **Request retries** — Automatic retry with exponential backoff
- **Compression** — Gzip, Brotli for responses
- **Caching** — Cache GET responses by TTL
- **WebSocket** — Long-lived connections for interactive sessions

---

## 4. Performance Targets

### 4.1 Latency Targets

| Operation | Target | P99 Target | Measurement |
|-----------|--------|------------|-------------|
| Cold start | < 30ms | < 50ms | From BOOT frame to RUNNING state |
| Fork | < 15ms | < 25ms | From FORK frame to child RUNNING |
| Hibernate (512 MiB) | < 2s | < 3s | From HIBERNATE to HIBERNATED |
| Resume (512 MiB) | < 100ms | < 150ms | From RESUME_HIBERNATED to RUNNING |
| Undo | < 2ms | < 5ms | From UNDO to state restored |
| Redo | < 2ms | < 5ms | From REDO to state restored |
| Exec round-trip | < 5ms | < 10ms | From exec request to first output |
| vsock RPC | < 100µs | < 500µs | Round-trip, 1KB payload |

### 4.2 Throughput Targets

| Metric | Target |
|--------|--------|
| Per-sandbox throughput | > 30 Gbps |
| Intra-host RPC | > 500K ops/s |
| Exec throughput | > 10K exec/s/node |
| Fork throughput | > 1K forks/s/node |

### 4.3 Resource Efficiency

| Metric | Target |
|--------|--------|
| Memory overhead per sandbox | < 3 MB |
| RAM for 100 sandboxes | < 5 GB |
| CPU per idle sandbox | < 0.1% |
| Storage per sandbox (runtime) | < 1 MB |

### 4.4 Scalability

| Metric | Target |
|--------|--------|
| Max sandboxes per node | > 5,000 |
| Max sandboxes per cluster | > 100,000 |
| Cluster join time | < 5s |
| Failure detection time | < 1s |

---

## 5. Security Model

### 5.1 Isolation Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Hardware Isolation                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │  MicroVM 1  │  │  MicroVM 2  │  │  MicroVM N  │          │
│  │  (KVM)      │  │  (KVM)      │  │  (KVM)      │          │
│  │  - Separate │  │  - Separate │  │  - Separate │          │
│  │    vCPU     │  │    vCPU     │  │    vCPU     │          │
│  │    memory   │  │    memory   │  │    memory   │          │
│  │    kernel   │  │    kernel   │  │    kernel   │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Network Isolation (eBPF)                     │
│  - Separate TAP devices per sandbox                           │
│  - No cross-sandbox communication                            │
│  - Strict SNAT/DNAT                                           │
│  - Policy enforcement at kernel level                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Capability Stripping                         │
│  - No CAP_SYS_ADMIN                                           │
│  - No CAP_NET_ADMIN                                           │
│  - No CAP_SYS_RAWIO                                           │
│  - Read-only /host filesystem                                 │
│  - No device access except vsock                              │
└─────────────────────────────────────────────────────────────────┘
```

### 5.2 Authentication

#### 5.2.1 mTLS (Mutual TLS)

```
Client Certificate:
  - CN: agent-001
  - O: acme-corp
  - Key usage: digitalSignature
  - Extended key usage: clientAuth

Server Certificate:
  - CN: coco.acme.com
  - O: acme-corp
  - Key usage: digitalSignature, keyEncipherment
  - Extended key usage: serverAuth
```

#### 5.2.2 JWT Tokens

```json
{
  "iss": "coco.acme.com",
  "sub": "agent-001",
  "aud": "coco-api",
  "exp": 1700000000,
  "iat": 1699900000,
  "tenant_id": "acme-corp",
  "roles": ["agent", "exec"],
  "rate_limit": 600
}
```

### 5.3 Authorization (RBAC)

| Role | Permissions |
|------|-------------|
| `admin` | Full access: sandboxes, templates, nodes, keys, config |
| `operator` | Sandboxes, templates, nodes, no keys, no config |
| `agent` | Own sandboxes: exec, fs, fork, hibernate, checkpoint |
| `readonly` | Read-only access to own resources |

### 5.4 Network Security

#### 5.4.1 Always-Denied Traffic

- All private IP ranges (10.x, 172.16.x, 192.168.x)
- Loopback (127.x)
- Link-local (169.254.x)
- Multicast (224.x)
- Broadcast (255.255.255.255)

#### 5.4.2 Per-Sandbox Policies

```json
{
  "sandbox_id": "sandbox-abc123",
  "allow_out": ["0.0.0.0/0"],  // Allow all outbound
  "deny_out": [],               // No explicit denies
  "allow_in": [],               // No inbound allowed
  "port_mappings": {
    "8080": "80"               // host:8080 -> sandbox:80
  }
}
```

### 5.5 Data Security

#### 5.5.1 Encryption at Rest

- **Hibernate images**: AES-256-GCM encryption
- **Checkpoints**: Encrypted with tenant key
- **Replay logs**: Encrypted with tenant key
- **State store**: BadgerDB with encryption

#### 5.5.2 Key Management

```
Key Hierarchy:
  - Master Key (HSM/TPM)
    - Tenant Key (derived)
      - Sandbox Key (derived)
        - Checkpoint Key (derived)
```

### 5.6 Audit Logging

```json
{
  "timestamp": "2026-04-26T10:00:00Z",
  "actor": "agent-001",
  "action": "sandbox.exec",
  "resource": "sandbox-abc123",
  "result": "success",
  "metadata": {
    "command": "python3",
    "exit_code": 0
  },
  "source_ip": "203.0.113.1",
  "trace_id": "abc123"
}
```

---

## 6. Network Architecture

### 6.1 Overview

```
┌──────────────────────────────────────────────────────────────────────┐
│                         Host Network                                │
│                                                                       │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                    coconet (eBPF + AF_XDP)                     │ │
│  │                                                                  │ │
│  │  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐   │ │
│  │  │ TAP dev  │   │ TAP dev  │   │ TAP dev  │   │ TAP dev  │   │ │
│  │  │  (sb-1)  │   │  (sb-2)  │   │  (sb-3)  │   │  (sb-N)  │   │ │
│  │  └────┬─────┘   └────┬─────┘   └────┬─────┘   └────┬─────┘   │ │
│  │       │              │              │              │          │ │
│  │       └──────────────┼──────────────┼──────────────┘          │ │
│  │                      │              │                          │ │
│  │               ┌──────▼──────────────▼──────┐                  │ │
│  │               │     from_sandbox (TC)      │                  │ │
│  │               │  - SNAT                    │                  │ │
│  │               │  - Policy check            │                  │ │
│  │               │  - Session create          │                  │ │
│  │               └─────────────┬──────────────┘                  │ │
│  │                             │                                   │ │
│  └─────────────────────────────┼───────────────────────────────────┘ │
│                                │                                    │
│                    ┌───────────▼────────────┐                        │
│                    │    Host NIC (eth0)     │                        │
│                    │    from_world (TC)    │                        │
│                    │  - DNAT               │                        │
│                    │  - Rev NAT            │                        │
│                    │  - Port mapping       │                        │
│                    └───────────┬────────────┘                        │
│                                │                                    │
│                                ▼                                    │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                     External Network                           │ │
│  └─────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────┘
```

### 6.2 Traffic Flows

#### 6.2.1 Egress (Sandbox → Internet)

```
Sandbox (169.254.68.2:40000)
    │
    │ Packet: src=169.254.68.2:40000, dst=8.8.8.8:53
    ▼
TAP device
    │
    │ TC ingress
    ▼
from_sandbox (eBPF)
    │
    ├─► Policy check (Bloom → LPM)
    │    - Allow? Yes → continue
    │    - Deny? → drop
    │
    ├─► Create session in egress_sessions
    │    - Key: (169.254.68.2, 40000, 8.8.8.8, 53, UDP)
    │    - Value: (192.168.1.100, 30001, ...)
    │
    ├─► SNAT
    │    - Replace src: 169.254.68.2 → 192.168.1.100
    │    - Replace sport: 40000 → 30001
    │    - Update checksums
    │
    └─► Redirect to eth0
         │
         ▼
    Internet (8.8.8.8:53)
```

#### 6.2.2 Ingress (Internet → Sandbox)

```
Internet (8.8.8.8:53)
    │
    │ Reply: src=8.8.8.8:53, dst=192.168.1.100:30001
    ▼
Host NIC (eth0)
    │
    │ TC ingress
    ▼
from_world (eBPF)
    │
    ├─► Session lookup (ingress_sessions)
    │    - Key: (8.8.8.8, 53, 192.168.1.100, 30001, UDP)
    │    - Found → continue
    │    - Not found → check port mapping
    │
    ├─► DNAT
    │    - Replace dst: 192.168.1.100:30001 → 169.254.68.2:40000
    │    - Update checksums
    │
    └─► Redirect to TAP device
         │
         ▼
    Sandbox (169.254.68.2:40000)
```

#### 6.2.3 Port Mapping (Exposing Services)

```
External client (203.0.113.1:50000)
    │
    │ Connect to node:8080
    ▼
Host NIC → from_world
    │
    ├─► Port mapping lookup
    │    - Key: 8080
    │    - Value: (sandbox-123 TAP, port 80)
    │
    ├─► DNAT
    │    - Replace dst: node:8080 → 169.254.68.2:80
    │
    └─► Redirect to TAP
         │
         ▼
    Sandbox HTTP server (port 80)
```

### 6.3 NAT Pool

```
SNAT IP Selection: index = hash(sandbox_ip) % pool_size

Pool Configuration:
  - 4 SNAT IPs (expandable)
  - Port range: 30000-60000 (30,000 ports per IP)
  - Total capacity: 120,000 concurrent connections

Port Allocation:
  - Per-IP waterline starts at 30000
  - Increment on each new connection
  - Wrap at 65535
  - Collision detection: retry up to 10 times
```

### 6.4 Session Tracking

#### 6.4.1 TCP State Machine

```
States: SYN_SENT, SYN_RECV, ESTABLISHED, FIN_WAIT, CLOSE_WAIT,
        LAST_ACK, TIME_WAIT, CLOSE, SYN_SENT2

Timeouts:
  - SYN_SENT/SYN_RECV: 60s
  - ESTABLISHED: 3 hours
  - FIN_WAIT/CLOSE_WAIT: 120s
  - TIME_WAIT: 10s
  - CLOSE: immediate
```

#### 6.4.2 UDP State Machine

```
States: UNREPLIED, REPLIED

Timeouts:
  - UNREPLIED: 30s
  - REPLIED: 180s
```

### 6.5 AF_XDP Fast Path

For workloads requiring >10 Gbps:

```
┌────────────────────────────────────────────────────────────────┐
│                     Standard Path                              │
│  ┌─────┐    ┌─────┐    ┌─────┐    ┌─────┐    ┌─────┐       │
│  │ NIC │───►│ NIC │───►│ TCP │───►│ App │───►│ App │       │
│  │driver│    │queue│    │stack│    │buf  │    │     │       │
│  └─────┘    └─────┘    └─────┘    └─────┘    └─────┘       │
│                                                                 │
│  Copy: 4-5x                                                     │
│  Latency: ~10µs                                                 │
└────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────┐
│                     AF_XDP Path                                │
│  ┌─────┐    ┌─────┐    ┌─────┐    ┌─────┐                   │
│  │ NIC │───►│ DMA │───►│ UMEM │───►│ App │                   │
│  │driver│    │     │    │     │    │ buf │                   │
│  └─────┘    └─────┘    └─────┘    └─────┘                   │
│                                                                 │
│  Copy: 0x (zero-copy)                                          │
│  Latency: ~1µs                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## 7. Storage Architecture

### 7.1 Directory Layout

```
/var/lib/coco/
├── images/                           # Rootfs templates
│   ├── alpine/
│   │   ├── rootfs.ext4              # 50 MB
│   │   └── metadata.yaml            # { version, size, checksum }
│   ├── ubuntu-22.04/
│   │   ├── rootfs.ext4
│   │   └── metadata.yaml
│   ├── debian-slim/
│   │   ├── rootfs.ext4
│   │   └── metadata.yaml
│   └── templates.yaml               # Index of all templates
│
├── hibernation/                      # Hibernate snapshots
│   └── {sandbox_id}/
│       ├── memory.img.zst           # Compressed memory
│       ├── vmstate.bin              # VM register state
│       ├── metadata.json            # { sandbox_id, memory_mb, created_at, ... }
│       └── checksum.sha256
│
├── checkpoints/                      # Named checkpoints (undo/redo)
│   └── {sandbox_id}/
│       └── {checkpoint_id}/
│           ├── diff.img.zst         # Memory page diff
│           ├── state.bin            # CPU state
│           ├── metadata.json        # { id, parent_id, created_at, description }
│           └── checksum.sha256
│
├── replays/                          # Replay event recordings
│   └── {sandbox_id}/
│       └── {replay_id}/
│           ├── events.log            # Binary event stream
│           ├── metadata.json        # { sandbox_id, started_at, event_count }
│           └── index.log            # Event offset index
│
├── store/                            # BadgerDB state store
│   ├── 000001.vlog
│   ├── 000002.vlog
│   ├── MANIFEST
│   └── lock.mdb
│
└── keys/                             # Encryption keys (protected)
    └── {tenant_id}/
        └── sandbox.key             # Per-sandbox encryption key

/run/coco/
├── visor.sock                       # cocovisor Unix socket
├── coconet.sock                     # coconet Unix socket
├── cocofork.sock                   # cocofork Unix socket
├── cocogate.sock                   # cocogate Unix socket
├── pid/
│   ├── cocovisor.pid
│   ├── coconet.pid
│   ├── cocofork.pid
│   └── cocogate.pid
└── logs/
    ├── cocovisor.log
    ├── coconet.log
    ├── cocofork.log
    └── cocogate.log

/sys/fs/bpf/coco/                    # Pinned eBPF maps
├── sandbox_ip_to_ifindex
├── ifindex_to_meta
├── egress_sessions
├── ingress_sessions
├── snat_pool
├── allow_rules
├── deny_rules
├── port_mapping
└── local_port_mapping
```

### 7.2 BadgerDB Schema

```
Keys:
  sandbox:{id}                       → Sandbox state JSON
  sandbox_index                      → [id1, id2, ...] (sorted set)
  sandbox_by_tenant:{tenant_id}      → [id1, id2, ...]
  
  checkpoint:{sandbox_id}:{checkpoint_id} → Checkpoint metadata
  checkpoint_index:{sandbox_id}       → [checkpoint_id1, ...] (linked list)
  
  replay:{sandbox_id}:{replay_id}   → Replay session metadata
  
  node:{id}                         → Node state JSON
  node_index                        → [node_id1, ...]
  
  template:{name}                   → Template metadata
  
  config:{key}                     → Global config
  
  rate_limit:{tenant_id}            → Rate limit state
```

### 7.3 Snapshot Format

```
┌────────────────────────────────────────┐
│ SnapshotHeader                         │
├────────────────────────────────────────┤
│ magic: u64          (0x434F434F4658)  │
│ version: u32         (1)              │
│ flags: u32           (compression...) │
│ memory_size: u64                      │
│ compressed_size: u64                  │
│ checksum: u32                         │
│ created_at: u64                       │
│ parent_snapshot: ?u64                 │
├────────────────────────────────────────┤
│ VMState                                │
├────────────────────────────────────────┤
│ registers: [16]u64   (RIP, RSP, etc.) │
│ fpu_state: [512]u8   (x87 + SSE)      │
│ cr3: u64            (page table)      │
│ msrs: [16]u64       (model-specific)  │
├────────────────────────────────────────┤
│ MemoryPages (compressed)              │
├────────────────────────────────────────┤
│ Page 1 (if dirty)                     │
│ Page 2 (if dirty)                     │
│ ...                                    │
└────────────────────────────────────────┘
```

---

## 8. API Specification

### 8.1 REST API

#### 8.1.1 Create Sandbox

**Request:**
```http
POST /v1/sandboxes
Content-Type: application/json
Authorization: Bearer <token>

{
  "name": "agent-42-reasoning",
  "template": "alpine",
  "memory_mb": 512,
  "vcpus": 2,
  "labels": {
    "tenant": "acme-corp",
    "purpose": "reasoning"
  },
  "network": {
    "allow_out": ["0.0.0.0/0"],
    "deny_out": [],
    "port_mappings": {}
  }
}
```

**Response (202 Created):**
```json
{
  "id": "sandbox-abc123",
  "name": "agent-42-reasoning",
  "state": "creating",
  "template": "alpine",
  "memory_mb": 512,
  "vcpus": 2,
  "vsock_cid": 3,
  "created_at": "2026-04-26T10:00:00Z"
}
```

#### 8.1.2 Exec (Streaming)

**Request:**
```http
POST /v1/sandboxes/sandbox-abc123/exec
Content-Type: application/json
Authorization: Bearer <token>

{
  "cmd": "python3",
  "args": ["-c", "print('hello world')"],
  "env": {"PYTHONPATH": "/app"},
  "working_dir": "/workspace",
  "timeout_ms": 30000
}
```

**Response (200 OK):**
```
Content-Type: text/event-stream

data: {"type":"stdout","data":"hello world\n"}
data: {"type":"exit","code":0}
```

#### 8.1.3 Fork

**Request:**
```http
POST /v1/sandboxes/sandbox-abc123/fork
Content-Type: application/json
Authorization: Bearer <token>

{
  "name": "experiment-1",
  "labels": {"parent": "sandbox-abc123"}
}
```

**Response (202 Accepted):**
```json
{
  "id": "sandbox-def456",
  "name": "experiment-1",
  "state": "running",
  "parent_id": "sandbox-abc123",
  "fork_duration_ms": 12
}
```

#### 8.1.4 Hibernate

**Request:**
```http
POST /v1/sandboxes/sandbox-abc123/hibernate
Authorization: Bearer <token>
```

**Response (202 Accepted):**
```json
{
  "state": "hibernated",
  "snapshot_id": "snap-xyz789",
  "hibernation_duration_ms": 1500
}
```

#### 8.1.5 Create Checkpoint

**Request:**
```http
POST /v1/sandboxes/sandbox-abc123/checkpoints
Content-Type: application/json
Authorization: Bearer <token>

{
  "name": "before-mutation",
  "description": "State before code mutation"
}
```

**Response (202 Created):**
```json
{
  "id": "checkpoint-001",
  "name": "before-mutation",
  "description": "State before code mutation",
  "created_at": "2026-04-26T10:00:00Z",
  "size_kb": 1024
}
```

#### 8.1.6 Undo

**Request:**
```http
POST /v1/sandboxes/sandbox-abc123/undo
Content-Type: application/json
Authorization: Bearer <token>

{
  "checkpoint": "checkpoint-001"
}
```

**Response (200 OK):**
```json
{
  "checkpoint_id": "checkpoint-001",
  "undo_duration_ms": 1
}
```

### 8.2 gRPC API

See `proto/v1/sandbox.proto` and `proto/internal/visor.proto` for full definitions.

### 8.3 Error Responses

```json
{
  "error": {
    "code": "SANDBOX_NOT_FOUND",
    "message": "Sandbox 'sandbox-xyz' not found",
    "details": {
      "sandbox_id": "sandbox-xyz"
    },
    "request_id": "req-abc123"
  }
}
```

**Error Codes:**

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `SANDBOX_NOT_FOUND` | 404 | Sandbox ID does not exist |
| `SANDBOX_ALREADY_RUNNING` | 409 | Cannot boot already-running sandbox |
| `SANDBOX_NOT_RUNNING` | 409 | Operation requires RUNNING state |
| `SANDBOX_HIBERNATED` | 409 | Cannot exec in hibernated sandbox |
| `CHECKPOINT_NOT_FOUND` | 404 | Checkpoint ID not found |
| `CHECKPOINT_CHAIN_BROKEN` | 400 | Undo requires valid parent chain |
| `VSOCK_CID_EXHAUSTED` | 503 | No more vsock CIDs available |
| `EXEC_TIMEOUT` | 408 | Command exceeded timeout |
| `EXEC_FAILED` | 500 | Command exited with non-zero code |
| `HYPERVISOR_ERROR` | 500 | Cloud Hypervisor error |
| `NETWORK_ERROR` | 500 | Network namespace creation failed |
| `EBPF_ERROR` | 500 | eBPF program failed to load |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `UNAUTHORIZED` | 401 | Invalid or missing token |
| `FORBIDDEN` | 403 | Insufficient permissions |

---

## 9. Observability

### 9.1 Prometheus Metrics

**Port:** 9090

#### 9.1.1 Sandbox Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `coco_sandbox_count` | Gauge | state, tenant | Number of sandboxes by state |
| `coco_sandbox_created_total` | Counter | tenant, template | Total sandboxes created |
| `coco_sandbox_destroyed_total` | Counter | tenant, template | Total sandboxes destroyed |

#### 9.1.2 Latency Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `coco_boot_duration_seconds` | Histogram | template | VM boot latency |
| `coco_fork_duration_seconds` | Histogram | — | Fork latency |
| `coco_hibernate_duration_seconds` | Histogram | memory_mb | Hibernate latency |
| `coco_resume_duration_seconds` | Histogram | memory_mb | Resume latency |
| `coco_exec_duration_seconds` | Histogram | exit_code | Exec latency |
| `coco_undo_duration_seconds` | Histogram | — | Undo latency |
| `coco_redo_duration_seconds` | Histogram | — | Redo latency |

#### 9.1.3 Network Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `coco_network_bytes_total` | Counter | direction, sandbox_id | Network bytes |
| `coco_network_packets_total` | Counter | action, sandbox_id | Packets (allow/deny) |
| `coco_network_sessions_total` | Counter | protocol, sandbox_id | NAT sessions created |
| `coco_network_session_expiry_total` | Counter | protocol, reason | Sessions expired |

#### 9.1.4 Resource Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `coco_memory_usage_bytes` | Gauge | sandbox_id | Memory used by sandbox |
| `coco_cpu_usage_percent` | Gauge | sandbox_id | CPU usage |
| `coco_vcpu_count` | Gauge | sandbox_id | vCPUs allocated |

#### 9.1.5 Cluster Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `coco_node_count` | Gauge | state | Nodes by state |
| `coco_node_sandboxes` | Gauge | node_id | Sandboxes per node |
| `coco_election_total` | Counter | result | Leader elections |

### 9.2 OpenTelemetry Tracing

**Port:** 4317 (gRPC), 4318 (HTTP)

**Spans:**

- `sandbox.create` — Full sandbox creation flow
- `sandbox.boot` — VM boot only
- `sandbox.exec` — Command execution
- `sandbox.fork` — Fork operation
- `sandbox.hibernate` — Hibernate operation
- `sandbox.checkpoint` — Checkpoint creation
- `sandbox.undo` — Undo operation
- `network.snat` — SNAT processing
- `network.dnat` — DNAT processing

**Attributes:**
- `sandbox.id`
- `sandbox.vsock_cid`
- `command`
- `exit_code`
- `duration`

### 9.3 Logging

**Format:** JSON structured logs

```json
{
  "timestamp": "2026-04-26T10:00:00.000Z",
  "level": "INFO",
  "component": "cocovisor",
  "sandbox_id": "sandbox-abc123",
  "trace_id": "abc123",
  "message": "Sandbox booted",
  "vsock_cid": 3,
  "pid": 12345,
  "duration_ms": 28
}
```

**Log Levels:**
- `DEBUG` — Detailed debugging info
- `INFO` — Normal operation
- `WARN` — Potential issues
- `ERROR` — Errors with stack traces

**Components:**
- `coco-core` — API server
- `cocovisor` — Hypervisor wrapper
- `coconet` — Network daemon
- `cocofork` — Fork/hibernate engine
- `cocod` — Guest agent
- `cocogate` — API gateway

---

## 10. Multi-Node Clustering

### 10.1 Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Cluster                                  │
│                                                                  │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐     │
│  │   Node 1    │     │   Node 2    │     │   Node 3    │     │
│  │  (Leader)   │◄───►│  (Follower) │◄───►│  (Follower) │     │
│  │ coco-core   │     │ coco-core   │     │ coco-core   │     │
│  │ cocovisor  │     │ cocovisor  │     │ cocovisor  │     │
│  │ coconet    │     │ coconet    │     │ coconet    │     │
│  └──────┬──────┘     └──────┬──────┘     └──────┬──────┘     │
│         │                    │                    │             │
│         │     ┌──────────────┼──────────────┐    │             │
│         └────►│   Data Plane  │              ◄────┘             │
│               │  (vswitch/    │                               │
│               │   overlay)    │                               │
│               └───────────────┘                               │
└─────────────────────────────────────────────────────────────────┘
```

### 10.2 Node Discovery

1. **Static** — List of nodes in config file
2. **DNS** — DNS SRV records
3. **Consul** — Service discovery via Consul
4. **Kubernetes** — Pod/Service discovery

### 10.3 Consensus (Raft)

- **Leader election** — 3-node majority for leader
- **Log replication** — Replicate sandbox state changes
- **Failure detection** — Heartbeat timeout 1s
- **Leader failover** — < 5s

### 10.4 Sharding

**Strategy:** Consistent hashing by sandbox ID

```
Sharding Algorithm:
  shard = hash(sandbox_id) % shard_count
  
Rebalancing:
  - When node joins/leaves
  - Minimize data movement (move 1/N keys)
  - Online rebalancing
```

### 10.5 Cross-Node Communication

| Method | Use Case |
|--------|----------|
| vsock | Exec commands to sandbox on any node |
| gRPC | API calls, cluster coordination |
| WireGuard | Encrypted overlay network |
| RDMA (future) | High-throughput cross-node traffic |

### 10.6 Data Distribution

```
Sandbox Metadata:
  - Stored in BadgerDB on leader
  - Replicated to followers via Raft
  
Checkpoints/Hibernation:
  - Stored on local node (for performance)
  - Optional: replicated to distributed storage
  
Network State:
  - Local per node (NAT sessions)
  - Shared via consistency protocol
```

---

## 11. Developer Experience

### 11.1 CLI (cococtl)

#### 11.1.1 Interactive Mode

```bash
$ cococtl shell sandbox-abc
[sandbox-abc] $ pwd
/workspace
[sandbox-abc] $ python3 -c "print('hello')"
hello
[sandbox-abc] $ exit
```

#### 11.1.2 Debug Mode

```bash
$ cococtl sandbox debug sandbox-abc --attach
[debug] Attached to sandbox-abc (PID 12345)
[debug] VM state: RUNNING
[debug] Memory: 512 MB / 512 MB
[debug] vCPUs: 2
[debug] vsock CID: 3
[debug] Network: 192.168.1.100
[debug] Press Ctrl+C to detach
```

### 11.2 SDKs

#### 11.2.1 Python SDK

```python
from coco import Sandbox

# Create sandbox
sandbox = Sandbox.create(template="alpine", memory_mb=512)

# Execute code
result = sandbox.run_code("print('hello')")
print(result.stdout)  # "hello\n"

# Fork for parallel exploration
fork = sandbox.fork()

# Create checkpoint
sandbox.checkpoint("before-mutation")

# Undo if needed
sandbox.undo("before-mutation")

# Hibernate during idle
sandbox.hibernate()

# Resume when needed
sandbox.resume()

# Cleanup
sandbox.destroy()
```

#### 11.2.2 JavaScript SDK

```javascript
import { Sandbox } from '@cocoai/coco';

// Create sandbox
const sandbox = await Sandbox.create({ template: 'alpine' });

// Execute code
const result = await sandbox.runCode("console.log('hello')");
console.log(result.stdout); // "hello\n"

// Fork
const fork = await sandbox.fork();

// Cleanup
await sandbox.destroy();
```

### 11.3 Debugging Tools

#### 11.3.1 Network Debugging

```bash
# View NAT sessions
cococtl debug net sessions sandbox-abc

# View eBPF maps
cococtl debug ebpf dump sessions

# Capture packets
cococtl debug net capture sandbox-abc --port 80

# View policy
cococtl debug net policy sandbox-abc
```

#### 11.3.2 VM Debugging

```bash
# View VM state
cococtl debug vm state sandbox-abc

# View memory usage
cococtl debug vm memory sandbox-abc

# View vCPU usage
cococtl debug vm vcpu sandbox-abc

# Core dump
cococtl debug vm coredump sandbox-abc
```

### 11.4 Development Environment

```bash
# Start dev environment
make dev

# Run tests
make test

# Run benchmarks
make bench

# Format code
make fmt

# Lint
make lint

# Build release
make release
```

---

## 12. Error Handling

### 12.1 Error Recovery

| Scenario | Recovery Action |
|----------|-----------------|
| cocovisor crash | Auto-restart via systemd, VMs remain intact |
| coco-core crash | Clients retry with backoff, idempotent operations |
| MicroVM crash | cocovisor detects via waitpid, updates state to ERROR |
| coconet restart | eBPF programs reloaded, existing connections preserved |
| Network partition | Leader election, automatic failover |

### 12.2 Graceful Degradation

- **API**: Return cached state if store is slow
- **Network**: Fallback to slower path if AF_XDP unavailable
- **Exec**: Fallback to vsock if fast path fails

### 12.3 Panic Handling

- **cocovisor**: Catch panics, log, return error to client
- **coconet**: eBPF programs handle errors gracefully
- **cocofork**: Validate state before operations

---

## 13. Testing Strategy

### 13.1 Unit Tests

```bash
# Run all unit tests
zig build test
go test ./...

# Coverage
zig build test --coverage
go test -coverprofile=coverage.out ./...
```

### 13.2 Integration Tests

```bash
# Run integration tests
make integration

# Specific test
make integration TEST=cocovisor
```

### 13.3 Benchmark Tests

```bash
# Run benchmarks
make bench

# Specific benchmark
zig build bench --bench=cold_start
go test -bench=BenchmarkColdStart ./...
```

### 13.4 Chaos Testing

- Random VM kills
- Network partition simulation
- Resource exhaustion
- Clock skew

### 13.5 EBPF Tests

- Load programs in test environment
- Verify packet transformations
- Test policy enforcement

---

## 14. Repository Structure

```
coco/
├── SPEC.md                     # This file
├── README.md                   # Project overview
├── LICENSE                     # Apache 2.0
├── Makefile                    # Build targets
├── docker-compose.yml          # Local dev environment
├── .gitignore
│
├── proto/
│   ├── v1/
│   │   ├── sandbox.proto       # REST + gRPC API
│   │   └── coco.proto         # Additional APIs
│   └── internal/
│       └── visor.proto        # Internal cocovisor protocol
│
├── core/                       # Go API server
│   ├── main.go
│   ├── server.go
│   ├── router.go
│   ├── middleware/
│   ├── handlers/
│   ├── cluster/
│   ├── store/
│   ├── visor/
│   ├── metrics/
│   └── config/
│
├── ctl/                       # Go CLI tool
│   ├── main.go
│   ├── sandbox.go
│   ├── exec.go
│   ├── fs.go
│   ├── checkpoint.go
│   ├── replay.go
│   ├── node.go
│   ├── template.go
│   └── debug/
│
├── cocogate/                  # Go API gateway
│   ├── main.go
│   ├── server.go
│   ├── auth/
│   ├── rate_limit/
│   └── load_balance/
│
├── src/                       # Zig components
│   ├── cocovisor/             # Hypervisor wrapper
│   │   ├── main.zig
│   │   ├── vmm.zig
│   │   ├── protocol.zig
│   │   ├── metrics.zig
│   │   └── build.zig
│   │
│   ├── coconet/               # Network daemon
│   │   ├── main.zig
│   │   ├── ebpf.zig
│   │   ├── nat.zig
│   │   ├── policy.zig
│   │   ├── af_xdp.zig
│   │   └── build.zig
│   │
│   ├── cocofork/              # Fork/hibernate
│   │   ├── main.zig
│   │   ├── snapshot.zig
│   │   ├── hibernate.zig
│   │   ├── checkpoint.zig
│   │   ├── replay.zig
│   │   └── build.zig
│   │
│   └── cocod/                 # Guest agent
│       ├── main.zig
│       ├── exec.zig
│       ├── fs.zig
│       ├── vsock.zig
│       └── build.zig
│
├── c/                         # eBPF C programs
│   ├── from_sandbox.bpf.c
│   ├── from_world.bpf.c
│   ├── from_envoy.bpf.c
│   ├── common.h
│   └── maps.h
│
├── images/                    # Rootfs images (built)
│   ├── alpine/
│   └── ubuntu/
│
├── scripts/                   # Build/test scripts
│   ├── build.sh
│   ├── test.sh
│   ├── bench.sh
│   └── smoke_test.sh
│
├── docs/                      # Documentation
│   ├── architecture/
│   ├── api/
│   └── deployment/
│
└── tests/                     # Test fixtures
    ├── integration/
    └── fixtures/
```

---

## Appendix A: Performance Comparison Details

### A.1 Cold Start Breakdown

| Step | CubeSandbox | Coco (dev) |
|------|-------------|-----------|
| API processing | 5ms | 1ms |
| VM creation | 30ms | 10ms |
| Kernel boot | 10ms | 5ms |
| cocod startup | 5ms | 2ms |
| Network setup | 5ms | 1ms |
| **Total** | **55ms** | **19ms** |

### A.2 Fork Breakdown

| Step | CubeSandbox | Coco (dev) |
|------|-------------|-----------|
| Pause parent | N/A | 1ms |
| Snapshot memory | N/A | 2ms |
| Allocate CID | N/A | 0.5ms |
| Create child VM | N/A | 8ms |
| Resume both | N/A | 2ms |
| **Total** | **N/A** | **13.5ms** |

---

## Appendix B: Compatibility

### B.1 E2B SDK Compatibility

```python
# Replace E2B with Coco
from e2b_code_interpreter import Sandbox

# Old (E2B)
sandbox = Sandbox.create(template="my-template")

# New (Coco with E2B compatibility)
sandbox = Sandbox.create(template="my-template")
# Just set environment variable
# E2B_API_URL=http://localhost:4747
```

### B.2 Firecracker API Compatibility

```bash
# Firecracker API
curl --unix-socket /run/firecracker.sock \
  -X PUT 'http://localhost/v1/vm' \
  -d '{ "state": "Running" }'

# Coco API
curl http://localhost:4747/v1/sandboxes/sandbox-abc \
  -X POST \
  -d '{ "action": "resume" }'
```

---

## Appendix C: Security Hardening

### C.1 Kernel Parameters

```bash
# Disable unused filesystems
echo "install squashfs /bin/true" > /etc/modprobe.d/squashfs.conf
echo "install udf /bin/true" > /etc/modprobe.d/udf.conf

# Network hardening
sysctl -w net.ipv4.conf.all.rp_filter=1
sysctl -w net.ipv4.conf.default.rp_filter=1
sysctl -w net.ipv4.icmp_echo_ignore_broadcasts=1
sysctl -w net.ipv4.conf.all.accept_redirects=0
sysctl -w net.ipv4.conf.default.accept_redirects=0
sysctl -w net.ipv6.conf.all.accept_redirects=0
sysctl -w net.ipv6.conf.default.accept_redirects=0
```

### C.2 Container Hardening

```dockerfile
# If running in containers
RUN addgroup -g 1000 -S coco && \
    adduser -u 1000 -S coco -G coco
USER coco:coco
```

---

*End of Specification*
