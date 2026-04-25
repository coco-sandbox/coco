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

- [ ] **core/** — Go HTTP/gRPC API server (port 4747)
  - [ ] POST /v1/sandboxes
  - [ ] GET /v1/sandboxes/:id
  - [ ] DELETE /v1/sandboxes/:id
  - [ ] POST /v1/sandboxes/:id/exec (streaming)
  - [ ] GET /health
  - [ ] GET /v1/sandboxes/:id/fs/ls (list directory)
  - [ ] GET /v1/sandboxes/:id/fs/tree (recursive tree)
  - [ ] GET /v1/sandboxes/:id/fs/cat (read file)
  - [ ] PUT /v1/sandboxes/:id/fs/write (write file)
  - [ ] POST /v1/sandboxes/:id/fs/mkdir (create dir)
  - [ ] DELETE /v1/sandboxes/:id/fs/rm (remove)

- [ ] **ctl/** — Go CLI tool
  - [ ] sandbox create/list/destroy
  - [ ] exec command

- [ ] **src/cocovisor/** — Zig hypervisor wrapper
  - [ ] Unix socket RPC server (/run/coco/visor.sock)
  - [ ] Boot/Exec/Destroy/GetState protocol
  - [ ] Cloud Hypervisor integration

- [ ] **src/coconet/** — Zig + eBPF networking
  - [ ] from_sandbox.bpf.c (egress SNAT)
  - [ ] from_world.bpf.c (ingress DNAT)
  - [ ] AF_XDP fast path
  - [ ] Bloom + LPM policy

- [ ] **src/cocofork/** — Zig fork/hibernate
  - [ ] Snapshot-fork with CoW
  - [ ] Hibernate to NVMe < 4s
  - [ ] Resume from NVMe < 200ms

- [ ] **src/cocod/** — Zig guest agent
  - [ ] PID 1 inside MicroVM
  - [ ] vsock listener
  - [ ] stdout/stderr streaming

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
| RAM per 100 sandboxes | < 7 GB |