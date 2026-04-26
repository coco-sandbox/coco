# Coco Architecture

## Three-Layer Architecture

### Agent Layer (Go)
- REST API server (coco-core) on port 4747
- Prometheus metrics on port 9090
- Handles HTTP requests, routing, middleware

### Execution Engine (Zig + C)
- cocovisor: VM lifecycle management via clh-remote
- coco-fork: Snapshot-based forking with CoW
- coco-agent: In-VM agent (PID 1)
- Hot paths in C for performance

### Networking (Go + eBPF)
- TAP device management
- IP address allocation (169.254.68.0/24 subnet)
- eBPF for kernel-level SNAT/DNAT
- Session tracking for NAT

## Sandbox Lifecycle

```
     ┌──────────┐
     │ Created │
     └────┬────┘
          │ boot
          ▼
     ┌──────────┐
     │ Booting  │
     └────┬────┘
          │
    ┌─────▼─────┐
    │  Running  │◄─────────┐
    └─────┬─────┘          │
       │  │                │ pause
       │  │                │
   ┌───▼──▼───┐            │
   │  Paused  │─────────────┘
   └──────────┘            │
                           │ hibernate
                           ▼
                    ┌────────────┐
                    │ Hibernated │
                    └──────┬─────┘
                           │ resume
                           ▼
                      Running

     │ delete
     ▼
  Stopped
```

## Template System

Templates enable fast boot:
1. Build rootfs from OCI image
2. Boot VM and wait for environment ready
3. Snapshot memory/state
4. Clone snapshot on sandbox create

## Network Isolation

Each sandbox gets:
- Unique TAP device
- Unique IP from 169.254.68.0/24
- eBPF rules for SNAT/DNAT
- Session tracking for NAT

## VM Communication

- cocovisor ↔ coco-core: Unix socket `/run/coco/visor.sock`
- coco-agent ↔ host: VSock (cid=2) with TCP fallback
- Hypervisor control: clh-remote via `/run/coco/vm/<id>/sock`
