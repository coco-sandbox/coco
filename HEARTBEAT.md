# HEARTBEAT.md

Periodic checks every 10 minutes.

---

## Phase Tracker

**Active phase:** P0 — Foundation + Implementation

**Phase gate (P0 → P1):**
- [ ] `coco-core` starts on port 4747
- [ ] `POST /v1/sandboxes` returns `{"id":..., "state":"running"}`
- [ ] `POST /v1/sandboxes/:id/exec` returns stdout
- [ ] Smoke test passes
- [ ] All benchmarks green

**Current blockers:** None

---

## Build Health (Every 10 Minutes)

```bash
# Zig repos
for dir in src/cocovisor src/coconet src/cocofork src/cocod; do
  cd $dir && zig fmt --check . && zig build test
  cd ../..
done

# Go repos
cd core && go fmt ./... && go vet ./... && go build -o coco-core .
cd ../ctl && go fmt ./... && go vet ./... && go build -o cococtl .
```

---

## Scope (SPEC.md — must implement)

### core/ — Go HTTP/gRPC API server (port 4747)
- [ ] POST /v1/sandboxes
- [ ] GET /v1/sandboxes/:id
- [ ] DELETE /v1/sandboxes/:id
- [ ] POST /v1/sandboxes/:id/exec (streaming)
- [ ] POST /v1/sandboxes/:id/fork
- [ ] POST /v1/sandboxes/:id/hibernate
- [ ] POST /v1/sandboxes/:id/resume
- [ ] POST /v1/sandboxes/:id/replay/start
- [ ] POST /v1/sandboxes/:id/replay/stop
- [ ] POST /v1/sandboxes/:id/checkpoint
- [ ] GET /v1/sandboxes/:id/checkpoints
- [ ] POST /v1/sandboxes/:id/undo
- [ ] POST /v1/sandboxes/:id/redo
- [ ] GET /health + /ready
- [ ] GET /v1/sandboxes/:id/fs/{ls,tree,cat,write,mkdir,rm}
- [ ] BadgerDB state store
- [ ] Prometheus metrics (port 9090)
- [ ] OpenTelemetry tracing
- [ ] mTLS + RBAC auth

### ctl/ — Go CLI tool
- [ ] sandbox create/list/destroy/get
- [ ] exec command
- [ ] fs ls/tree/cat/write/mkdir/rm
- [ ] checkpoint/undo/redo commands

### src/cocovisor/ — Zig hypervisor wrapper
- [ ] Unix socket RPC server (/run/coco/visor.sock)
- [ ] Binary frame protocol (Boot/Exec/Destroy/GetState/Pause/Resume/Fork/Hibernate)
- [ ] Cloud Hypervisor integration via ch-remote
- [ ] vsock CID allocation
- [ ] Memory ballooning

### src/coconet/ — Zig + eBPF networking
- [ ] from_sandbox.bpf.c (egress SNAT, TC egress hook)
- [ ] from_world.bpf.c (ingress DNAT, TC ingress hook)
- [ ] AF_XDP fast path (Intel E810)
- [ ] Bloom + LPM policy engine
- [ ] Sandbox network namespace creation

### src/cocofork/ — Zig fork/hibernate/checkpoints
- [ ] Snapshot-fork with CoW (reflink)
- [ ] Hibernate to NVMe < 4s
- [ ] Resume from NVMe < 200ms
- [ ] Checkpoints (undo/redo) < 5ms
- [ ] Replay event recording

### src/cocod/ — Zig guest agent
- [ ] PID 1 inside MicroVM
- [ ] vsock listener (port 4747)
- [ ] exec handler (stdout/stderr streaming)
- [ ] fs operations (read/write/mkdir/rm)

### proto/ — Protocol Buffers
- [ ] proto/v1/sandbox.proto (REST + gRPC API)
- [ ] proto/internal/visor.proto (internal visor RPC)

### c/ — eBPF C programs
- [ ] from_sandbox.bpf.c
- [ ] from_world.bpf.c
- [ ] common.h, maps.h

---

## Benchmark Targets

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
| RAM per 100 sandboxes | < 7 GB |

---

## Doc Drift Check

- [ ] `SPEC.md` matches architecture?
- [ ] Phase status updated?
- [ ] TODO items checked off?
- [ ] Blockers updated?

---

## Phase Change Checklist

Saat transisi ke phase baru:
1. Semua benchmark sudah hit?
2. Semua acceptance tests pass?
3. SPEC.md updated?
4. Phase status updated?
5. Release tag dibuat? (`v0.x.0-pN`)
6. Demo / community update published?
