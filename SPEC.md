# Coco Sandbox v1.0 — Specification

> Open-source agent-native sandbox runtime. Apache 2.0.
> Stack: **Zig + C + Go** only. Single repo.

---

## 1. Mission

Build sandbox runtime yang **outperform Cube/E2B/Modal di setiap
measurable axis**, dengan agent-native primitives (fork, hibernate,
replay) yang nggak ada di kompetitor.

---

## 2. Performance Targets

| Metric | Cube/E2B baseline | **Coco v1.0** |
|---|---|---|
| Cold start (median) | 187 ms | **< 50 ms** |
| Cold start (p99) | 312 ms | **< 70 ms** |
| Per-sandbox throughput | 6.8 Gbps | **> 23 Gbps** |
| Intra-host RPC p99 | 38 µs | **< 5 µs** |
| Fork latency | N/A | **< 30 ms** |
| Hibernate (512 MiB) | N/A | **< 4 s** |

---

## 3. Components

### 3.1 core/ — API Server (Go)

HTTP/gRPC server listening on port 4747.

**Sandbox Management:**
- `POST /v1/sandboxes` — create sandbox
- `GET /v1/sandboxes/:id` — describe sandbox
- `DELETE /v1/sandboxes/:id` — destroy sandbox
- `POST /v1/sandboxes/:id/exec` — execute command (streaming)
- `GET /health` — health check

**File Operations:**
- `GET /v1/sandboxes/:id/fs/ls` — list directory contents
  - Query: `?path=/some/dir` (default: `/`)
  - Returns: `[{name, type, size, mode, mtime}, ...]`
- `GET /v1/sandboxes/:id/fs/tree` — recursive directory tree
  - Query: `?path=/some/dir&depth=3` (default: `/`, depth: unlimited)
  - Returns: tree structure with children
- `GET /v1/sandboxes/:id/fs/cat` — read file contents
  - Query: `?path=/some/file`
  - Returns: raw file bytes
- `PUT /v1/sandboxes/:id/fs/write` — write file contents
  - Body: raw bytes + `?path=/some/file`
- `POST /v1/sandboxes/:id/fs/mkdir` — create directory
  - Body: `{"path": "/some/dir"}`
- `DELETE /v1/sandboxes/:id/fs/rm` — remove file or directory
  - Query: `?path=/some/path&recursive=true`

### 3.2 ctl/ — CLI Tool (Go)

```bash
# Sandbox management
cococtl sandbox create <name> <template>
cococtl sandbox list
cococtl sandbox destroy <id>
cococtl exec <id> <cmd> [args...]

# File operations
cococtl fs ls <id> [path]
cococtl fs tree <id> [path] [depth]
cococtl fs cat <id> <path>
cococtl fs write <id> <path> <content>
cococtl fs mkdir <id> <path>
cococtl fs rm <id> <path> [-r|--recursive]
```

### 3.3 src/cocovisor/ — Hypervisor Wrapper (Zig)

- Manages KVM MicroVMs via Cloud Hypervisor
- Unix socket RPC server at `/run/coco/visor.sock`
- Binary protocol for Boot/Exec/Destroy

### 3.4 src/coconet/ — Network Daemon (Zig + eBPF C)

- `c/from_sandbox.bpf.c` — Egress SNAT
- `c/from_world.bpf.c` — Ingress DNAT
- AF_XDP fast path (Intel E810)
- Bloom + LPM policy engine

### 3.5 src/cocofork/ — Fork/Hibernate (Zig)

- Snapshot-fork with CoW memory
- Hibernate to NVMe < 4s
- Resume from NVMe < 200ms

### 3.6 src/cocod/ — Guest Agent (Zig)

- Runs inside MicroVM as PID 1
- Listens on vsock for exec commands
- Streams stdout/stderr back to host

---

## 4. Protocol

Internal communication via Unix socket RPC:

```
Boot(sandbox_id, rootfs, memory_mb, vcpus) → (vsock_cid, pid)
Exec(cmd, args, env, working_dir) → stream(stdout|stderr|exit)
Destroy(force) → ()
GetState() → (state, pid, vsock_cid)
```

---

## 5. Repository Structure

```
coco/
├── proto/              # Protobuf definitions
│   ├── v1/sandbox.proto
│   └── internal/visor.proto
├── core/               # Go API server
│   ├── main.go
│   └── go.mod
├── ctl/                # Go CLI
│   └── main.go
├── src/                # Zig components
│   ├── cocovisor/
│   ├── coconet/
│   ├── cocofork/
│   └── cocod/
├── c/                  # eBPF C programs
│   ├── from_sandbox.bpf.c
│   └── from_world.bpf.c
└── Makefile
```

---

## 6. Benchmark Targets

| Metric | Target |
|--------|--------|
| Cold start median | < 50ms |
| Cold start p99 | < 70ms |
| Throughput | > 23 Gbps |
| Intra-host RPC p99 | < 5µs |
| Fork latency | < 30ms |
| Hibernate (512 MiB) | < 4s |
| Resume from NVMe | < 200ms |
| RAM per 100 sandboxes | < 7 GB |