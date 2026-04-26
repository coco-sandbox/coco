# Coco Sandbox - Specification Index

This is the complete specification for Coco Sandbox. All development should reference these documents before writing code.

## Specification Files

| File | Description |
|------|-------------|
| [00-overview.md](./00-overview.md) | Complete specification with architecture, components, and design decisions |
| [01-folder-structure.md](./01-folder-structure.md) | Repository structure with complete file listing |
| [02-api.md](./02-api.md) | REST API definitions |
| [03-network.md](./03-network.md) | Network architecture and eBPF design |
| [04-security.md](./04-security.md) | Security architecture and isolation |
| [05-storage.md](./05-storage.md) | Storage and checkpoint design |
| [06-performance.md](./06-performance.md) | Performance targets and optimizations |
| [07-cluster.md](./07-cluster.md) | Cluster architecture and failover |
| [08-observability.md](./08-observability.md) | Logging, metrics, and tracing |
| [09-dependencies.md](./09-dependencies.md) | Build and runtime requirements |

## Quick Reference

### Performance Targets

| Metric | Target |
|--------|--------|
| Cold Start | <30ms |
| Fork | <10ms |
| Memory Overhead | <2MB |
| Network Throughput | 20Gbps |

### Component Languages

| Component | Language |
|-----------|----------|
| Gateway | Go |
| Master | Go |
| Node | Go |
| Visor | Zig |
| Agent | Zig |
| Net | Go + C (eBPF) |

### Communication

| Path | Protocol |
|------|----------|
| Client → Gateway | REST |
| Gateway → Master | gRPC |
| Master → Node | gRPC |
| Node → Visor | Unix Socket |
| Visor ↔ Agent | VSock |

### Comparison with CubeSandbox

| Metric | CubeSandbox | Coco |
|--------|-------------|------|
| Cold Start | <60ms | <30ms |
| Fork | ~100ms | <10ms |
| Memory | <5MB | <2MB |

## Reading Order

1. Start with 00-overview.md for complete architectural understanding
2. Use 01-folder-structure.md to understand repository organization
3. Reference other files for specific subsystems as needed

## Version

This is the approved specification for Coco Sandbox.
