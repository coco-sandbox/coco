# Coco Native: Agent-Native Sandbox Runtime

**Date:** 2026-04-26
**Status:** Draft

---

## 1. Overview

Coco Native is an agent-native sandbox runtime that provides hardware-level isolated execution environments for AI agents. It combines the fast cold-start capabilities of CubeSandbox with unique features (replay, fork, hibernate) that make it ideal for agent workloads.

**Design Goals:**
- Sub-100ms cold start via template snapshots
- Hardware-level isolation (KVM MicroVMs)
- Agent-native APIs for streaming exec, state inspection, composition
- Replay & checkpoint for agent debugging and retry
- Fork for parallel exploration
- Hibernate for ultra-fast resume

**Architecture Layers:**
1. **Agent Layer (Go)** — REST/gRPC API, streaming exec, cluster orchestration
2. **Networking Layer (Go + eBPF)** — SNAT/DNAT, policies, session tracking
3. **Execution Engine (Zig + minimal C)** — Hypervisor client, VM lifecycle, checkpointing

---

## 2. Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  Agent Layer (Go)                                               │
│                                                                 │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐│
│  │ REST API    │  │ Streaming    │  │ Cluster Orchestration  ││
│  │ (coco-core) │  │ Exec Engine  │  │ (membership, sched)     ││
│  │ :4747       │  │              │  │                        ││
│  └──────┬──────┘  └──────┬───────┘  └───────────┬────────────┘│
│         │                │                       │             │
│         └────────────────┼───────────────────────┘             │
│                          │                                      │
│  ┌──────────────────────┴──────────────────────────────────┐  │
│  │              SDK Layer (Go, Python, JS)                  │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────┬───────────────────────────────┘
                                  │ Unix socket / gRPC
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│  Execution Engine (Zig)                                         │
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────┐  │
│  │ coco-visor   │  │ coco-agent    │  │ coco-fork          │  │
│  │ Hypervisor   │  │ Agent Runtime │  │ Sandbox cloning    │  │
│  │ Client       │  │ (in-VM)       │  │                    │  │
│  │              │  │               │  │                    │  │
│  └──────┬───────┘  └──────┬───────┘  └─────────┬──────────┘  │
│         │                 │                     │              │
│         └─────────────────┼─────────────────────┘              │
│                           │                                     │
│  ┌────────────────────────┴──────────────────────────────────┐│
│  │  Hot Path (C)  —  VM boot, memory copy, snapshot write   ││
│  └───────────────────────────────────────────────────────────┘│
└─────────────────────────────────┬───────────────────────────────┘
                                  │ KVM / VM lifecycle
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│  MicroVM (isolated)                                             │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ coco-agent (PID 1) → user workload                        │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                                  ▲
                                  │ eBPF
┌─────────────────────────────────────────────────────────────────┐
│  Networking Layer (Go + eBPF)                                   │
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────┐  │
│  │ coco-net     │  │ eBPF (TC)    │  │ Policies           │  │
│  │ TAP/IPAM     │  │ SNAT/DNAT    │  │ allow/deny rules   │  │
│  └──────────────┘  └──────────────┘  └────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. Components

### 3.1 Agent Layer (Go)

#### coco-core (REST API Server)
- **Port:** 4747
- **Purpose:** Main entry point for SDK clients
- **Responsibilities:**
  - Sandbox CRUD operations
  - Streaming exec (chunked upload/download)
  - Template management
  - Cluster membership
  - Metrics (Prometheus on :9090)

**Key Endpoints:**
```
POST   /v1/sandboxes          — Create sandbox
GET    /v1/sandboxes/:id      — Get sandbox state
DELETE /v1/sandboxes/:id      — Destroy sandbox
POST   /v1/sandboxes/:id/exec — Stream code execution
POST   /v1/sandboxes/:id/fork — Fork running sandbox
POST   /v1/sandboxes/:id/checkpoint — Create checkpoint
POST   /v1/sandboxes/:id/hibernate — Hibernate to disk
POST   /v1/sandboxes/:id/resume    — Resume from hibernate
GET    /v1/sandboxes/:id/replay/:replay_id — Replay session
GET    /v1/templates          — List templates
POST   /v1/templates          — Create template
```

#### coco-gate (Gateway)
- **Port:** 4749
- **Purpose:** Rate limiting, circuit breakers, multi-tenant
- **Features:**
  - Token bucket rate limiter (600 cap, 10 req/s, burst 100)
  - Round-robin load balancing
  - Circuit breaker (5 pools, 30s timeout)

#### Streaming Exec Protocol
```protobuf
message ExecRequest {
  string sandbox_id = 1;
  bytes code_chunk = 2;  // streaming chunks
  bool streaming = 3;    // stream output back
  int64 timeout_ms = 4;
}

message ExecResponse {
  bytes output_chunk = 1;  // streaming output
  string error = 2;
  ExecResult final_result = 3;
}

message ExecResult {
  string stdout = 1;
  string stderr = 2;
  int32 exit_code = 3;
  string execution_time_ms = 4;
}
```

---

### 3.2 Execution Engine (Zig)

#### coco-visor (Hypervisor Client)
- **Socket:** `/run/coco/visor.sock`
- **Protocol:** Binary frame-based (kind[4] + size[4] + payload)
- **Purpose:** Manages VM lifecycle via KVM

**Message Types:**
```
BOOT=1              — Boot VM from template
EXEC=2              — Execute command in VM
DESTROY=3           — Destroy VM
PAUSE=4             — Pause VM
RESUME=5            — Resume VM
GET_STATE=6         — Get VM state
FORK=7              — Clone VM (copy-on-write)
HIBERNATE=8         — Suspend to disk
RESUME_HIBERNATED=9 — Resume from hibernation
CREATE_CHECKPOINT=10— Create named snapshot
RESTORE_CHECKPOINT=11— Restore from snapshot
```

#### coco-agent (In-VM Agent)
- **Purpose:** Runs inside MicroVM as PID 1
- **Responsibilities:**
  - Container lifecycle (runc-compatible)
  - Environment setup (mounts, namespaces, networking)
  - I/O stream forwarding to visor
  - Prometheus metrics export via vsock

#### coco-fork (Sandbox Cloning)
- **Purpose:** Fast sandbox forking via CoW memory
- **Operation:** Creates a new VM by cloning running VM's memory state

#### Hot Path (C)
- **Purpose:** Performance-critical operations
- **Functions:**
  - VM memory copy (memcpy optimizations)
  - Snapshot delta compression
  - Page fault handling for CoW

---

### 3.3 Networking Layer (Go + eBPF)

#### coco-net (Network Daemon)
- **Purpose:** TAP device management, IPAM, port mapping
- **Communication:** HTTP REST + gRPC + Unix socket FD passing

#### eBPF Programs

**from_cube (TC ingress on TAP):**
- SNAT for outbound traffic
- Session creation and tracking
- Policy evaluation (allow/deny)

**from_world (TC ingress on host NIC):**
- DNAT for inbound traffic
- Port mapping resolution
- Session lookup

**xdp_fwd (XDP on host NIC):**
- Fast packet forwarding for overlay traffic

#### Session Tracking
```
TCP: 11-state machine (SYN_SENT → ESTABLISHED → FIN_WAIT → etc)
UDP: 2-state (UNREPLIED/REPLIED)
ICMP: 2-state with timeout
```

#### Policy Model
```
1. Gateway traffic → Always allowed
2. allow_out match → Allow
3. deny_out match → Drop
4. Default → Allow
```

**Always-denied CIDRs (non-overridable):**
- 10.0.0.0/8, 127.0.0.0/8, 169.254.0.0/16, 172.16.0.0/12, 192.168.0.0/16

---

## 4. Template System

Template enables <100ms cold start via memory snapshot cloning:

### Template Lifecycle
```
1. BUILD   — Build rootfs from OCI image
2. BOOT    — Cold boot VM, wait for environment ready
3. SNAPSHOT— Capture memory/state snapshot
4. DEPLOY  — Register template for sandbox creation
```

### On Sandbox Create
```
Template → Clone snapshot (CoW) → Resume execution → Ready in <100ms
```

### Template Storage
```
/var/lib/coco/templates/
  ├── <template_id>/
  │   ├── rootfs.img      # Root filesystem
  │   ├── kernel          # Linux kernel
  │   ├── snapshot.mem    # Memory snapshot
  │   └── metadata.json   # Template metadata
  └── templates.json      # Template index
```

---

## 5. Key Features

### 5.1 Replay
**Purpose:** Record and replay execution sessions for debugging, retry, audit

**Implementation:**
```
1. Session starts → Create replay buffer
2. All exec ops logged with timestamps
3. On replay request → Replay from buffer
4. Can step forward/backward (if checkpoints)
```

**Replay API:**
```
GET /v1/sandboxes/:id/replays          — List replays
GET /v1/sandboxes/:id/replays/:replay_id — Get replay
POST /v1/sandboxes/:id/replay/:replay_id — Start replay
```

### 5.2 Fork
**Purpose:** Clone running sandbox for parallel exploration

**Use cases:**
- Agent exploring multiple code paths simultaneously
- Parallel tool execution
- Safety isolation for untrusted code

**Implementation:**
```
1. Pause source VM
2. Create CoW snapshot
3. Fork visor process with new VM
4. Resume both VMs
5. Return new sandbox_id
```

### 5.3 Checkpoint
**Purpose:** Named snapshots for undo/redo, branching

**Operations:**
```
POST /v1/sandboxes/:id/checkpoint  — Create named checkpoint
POST /v1/sandboxes/:id/checkpoints/:name/restore — Restore
DELETE /v1/sandboxes/:id/checkpoints/:name — Delete
```

### 5.4 Hibernate
**Purpose:** Suspend VM to disk for ultra-fast resume

**Storage:**
```
/var/lib/coco/hibernation/
  └── <sandbox_id>/
      ├── memory.img
      └── state.bin
```

**Use cases:**
- Free up resources for inactive agents
- Rapid context switching
- Disaster recovery

---

## 6. Data Flow

### Create Sandbox
```
1. Client → coco-core: POST /v1/sandboxes {template_id}
2. coco-core → coco-visor: BOOT {template_id}
3. coco-visor → KVM: Launch VM from template snapshot
4. coco-visor → coco-net: EnsureNetwork (TAP, IP)
5. coco-net → eBPF: Install NAT rules
6. coco-visor → coco-core: Sandbox ready {id, ip, port}
7. coco-core → Client: 201 Created {sandbox_id}
```

### Execute Code (Streaming)
```
1. Client → coco-core: POST /v1/sandboxes/:id/exec {code, streaming: true}
2. coco-core → coco-visor: EXEC {code}
3. coco-visor → VM: Forward to coco-agent
4. coco-agent: Run code, stream stdout/stderr
5. VM → coco-visor: Output chunks
6. coco-visor → coco-core: Forward chunks
7. coco-core → Client: Stream output (chunked)
```

### Fork Sandbox
```
1. Client → coco-core: POST /v1/sandboxes/:id/fork
2. coco-core → coco-visor: FORK {sandbox_id}
3. coco-visor: Pause VM, CoW clone, spawn new visor
4. new visor → coco-net: EnsureNetwork
5. new visor → coco-core: Fork complete {new_sandbox_id}
6. coco-core → Client: 201 Created {new_sandbox_id}
```

---

## 7. Configuration

**Default paths:**
```
/var/lib/coco/
  ├── store/          # BadgerDB state
  ├── images/         # VM images
  ├── templates/      # Template snapshots
  ├── checkpoints/    # Named snapshots
  ├── hibernation/    # Hibernated VMs
  └── replays/        # Execution recordings

/run/coco/
  └── visor.sock      # Visor Unix socket

/etc/coco/
  └── config.yaml     # Runtime configuration
```

**Default ports:**
- coco-core: 4747
- coco-gate: 4749
- metrics: 9090

---

## 8. SDKs

### Go SDK (`github.com/coco-sandbox/coco/sdk/go`)
```go
client, _ := coco.NewClient("http://localhost:4747")

// Create sandbox from template
sb, _ := client.Sandbox.Create(ctx, coco.CreateOpts{
    TemplateID: "python-3.11",
})

// Stream code execution
stream, _ := sb.Exec(ctx, &coco.ExecRequest{
    Code:      "print('hello')",
    Streaming: true,
})
for chunk := range stream.Chunks() {
    fmt.Print(string(chunk))
}

// Fork for parallel work
forked, _ := sb.Fork(ctx)

// Checkpoint for undo
sb.Checkpoint(ctx, "before-fix")

// Hibernate (free resources)
sb.Hibernate(ctx)

// Resume later
sb.Resume(ctx)
```

### Python SDK (`pycoco`)
```python
sandbox = await Sandbox.create(template="python-3.11")

# Streaming exec
async for chunk in sandbox.run_code("print('hello')", stream=True):
    print(chunk)

# Fork
forked = await sandbox.fork()

# Checkpoint
await sandbox.checkpoint("before-test")

# Replay
replays = await sandbox.list_replays()
await sandbox.replay(replays[0].id)
```

---

## 9. Comparison with CubeSandbox

| Feature | CubeSandbox | Coco Native |
|---------|-------------|------------|
| Cold Start | <60ms via snapshot | <100ms via snapshot |
| Isolation | KVM MicroVM | KVM MicroVM |
| Networking | eBPF (CubeVS) | eBPF (from_cube/from_world) |
| Fork | Not supported | Supported (CoW clone) |
| Replay | Not supported | Supported (full session record) |
| Hibernate | Not supported | Supported (disk suspend) |
| Checkpoint | Not supported | Named snapshots |
| E2B Compatible | Yes | Yes |
| Agent-native APIs | No | Yes (streaming exec, state inspection) |
| Language | Rust VMM + Go | Zig + C (hot path) + Go (networking) |

---

## 10. Implementation Priorities

### Phase 1: Core Infrastructure
1. coco-visor (Zig) — VM lifecycle via KVM
2. Template system — Build, snapshot, deploy
3. coco-core (Go) — REST API, sandbox CRUD
4. coco-net (Go) — TAP management, IPAM

### Phase 2: Networking
5. eBPF programs — SNAT/DNAT, policies
6. Session tracking — TCP/UDP state machines

### Phase 3: Unique Features
7. Replay system — Record and replay
8. Fork — CoW sandbox cloning
9. Checkpoint — Named snapshots

### Phase 4: Production Ready
10. coco-gate — Rate limiting, circuit breakers
11. SDKs — Go, Python, JS
12. Observability — Metrics, tracing, logging
13. Testing — Unit, integration, e2e

---

## 11. Open Questions

- [ ] Snapshot storage format (raw vs qcow2 vs overlay)
- [ ] Memory allocation strategy (pre-allocate vs balloon)
- [ ] Multi-node clustering (leader election vs no-single-point)
- [ ] Template registry (local vs distributed)
- [ ] Replay compression strategy

---

*This spec will be refined during implementation planning.*