# Coco Sandbox - Folder Structure Specification

This document defines the complete repository structure for Coco. Every directory and file is specified here to serve as the source of truth for development.

## 1. Root Directory Structure

The root directory contains all project components organized by function.

```
coco/                              # Main repository root
├── cmd/                           # Executable command entry points (Go)
├── daemon/                        # Long-running service daemons (Zig + Go)
├── pkg/                           # Shared Go packages
├── proto/                         # Protocol buffer definitions
├── ebpf/                         # eBPF kernel programs (C)
├── test/                          # Test suites
├── configs/                       # Configuration file templates
├── scripts/                       # Build and utility scripts
├── deploy/                       # Deployment manifests
└── spec/                         # This specification
```

## 2. Command Layer (cmd/)

The cmd layer contains executable entry points. Each subdirectory produces a separate binary.

```
cmd/
├── coco-gateway/                 # API Gateway service
│   ├── main.go                  # Entry point, argument parsing
│   ├── server.go                # HTTP server setup, middleware chain
│   ├── router.go                # Request routing
│   ├── handlers/                # HTTP request handlers
│   │   ├── sandbox.go          # Sandbox CRUD handlers
│   │   ├── exec.go             # Execution handlers
│   │   ├── template.go         # Template handlers
│   │   └── cluster.go          # Cluster handlers
│   ├── middleware/             # HTTP middleware
│   │   ├── auth.go             # Authentication
│   │   ├── ratelimit.go       # Rate limiting
│   │   ├── logging.go          # Structured logging
│   │   ├── tracing.go         # OpenTelemetry tracing
│   │   ├── cors.go            # CORS handling
│   │   └── recovery.go        # Panic recovery
│   ├── validators/             # Request validation
│   │   ├── sandbox.go
│   │   └── exec.go
│   └── docker/                 # Container build
│       └── Dockerfile
│
├── coco-node/                   # Node daemon
│   ├── main.go                  # Entry point
│   ├── node.go                  # Node initialization
│   ├── scheduler.go             # Local scheduling
│   ├── pool.go                 # VM pool management
│   ├── metrics.go              # Metrics collection
│   ├── health.go              # Health checking
│   ├── discovery.go            # Node discovery
│   ├── resource.go             # Resource tracking
│   ├── watcher.go              # Event watching
│   └── docker/
│       └── Dockerfile
│
├── coco-master/                 # Cluster master
│   ├── main.go                  # Entry point
│   ├── master.go                # Master initialization
│   ├── election.go             # Leader election via etcd
│   ├── scheduler.go            # Cluster scheduling
│   ├── failover.go             # Failover handling
│   ├── queue.go                # Request queue
│   ├── balancer.go             # Load balancing
│   └── docker/
│       └── Dockerfile
│
└── cococtl/                    # CLI client
    ├── main.go                  # Entry point, CLI setup
    ├── sandbox/                # Sandbox commands
    │   ├── create.go           # Create sandbox
    │   ├── list.go            # List sandboxes
    │   ├── get.go             # Get sandbox details
    │   ├── delete.go          # Delete sandbox
    │   ├── pause.go           # Pause sandbox
    │   ├── resume.go          # Resume sandbox
    │   ├── fork.go            # Fork sandbox
    │   └── exec.go            # Execute command
    ├── template/               # Template commands
    │   ├── list.go            # List templates
    │   ├── create.go         # Create template
    │   └── delete.go          # Delete template
    ├── checkpoint/             # Checkpoint commands
    │   ├── create.go         # Create checkpoint
    │   ├── list.go           # List checkpoints
    │   └── restore.go        # Restore from checkpoint
    └── cluster/                # Cluster commands
        ├── info.go           # Cluster info
        └── nodes.go          # Node management
```

## 3. Daemon Layer (daemon/)

The daemon layer contains long-running services. These implement the core sandbox functionality.

```
daemon/
├── coco-visor/                 # Hypervisor (Zig)
│   ├── src/                   # Zig source code
│   │   ├── main.zig           # Entry point, socket server
│   │   ├── vmm.zig          # VM management orchestration
│   │   ├── kvm.zig          # KVM syscall bindings
│   │   ├── vm.zig           # VM instance
│   │   ├── memory.zig       # Memory region setup
│   │   ├── vcpu.zig         # Virtual CPU management
│   │   ├── vsock.zig        # VSock communication
│   │   ├── snapshot.zig     # Snapshot operations
│   │   ├── fork.zig         # Fork implementation
│   │   ├── checkpoint.zig    # Checkpoint creation
│   │   ├── restore.zig      # Checkpoint restoration
│   │   ├── tdx.zig          # TDX support
│   │   ├── sgx.zig          # SGX support
│   │   ├── config.zig       # Configuration
│   │   ├── logger.zig       # Logging
│   │   ├── metrics.zig      # Metrics
│   │   └── protocol.zig      # Protocol definitions
│   ├── build.zig             # Zig build configuration
│   └── tests/                # Unit tests
│
├── coco-agent/                 # Guest agent (Zig)
│   ├── src/
│   │   ├── main.zig         # PID 1 entry point
│   │   ├── exec.zig          # Command execution
│   │   ├── vsock.zig         # Host communication
│   │   ├── signal.zig        # Signal handling
│   │   ├── process.zig       # Process management
│   │   ├── pty.zig          # PTY handling
│   │   ├── logger.zig        # Logging
│   │   └── protocol.zig      # Protocol definitions
│   └── build.zig
│
├── coco-fork/                  # Fork engine (Zig)
│   ├── src/
│   │   ├── main.zig
│   │   ├── snapshot.zig       # COW snapshot
│   │   ├── clone.zig          # VM cloning
│   │   ├── reflink.zig       # btrfs reflink
│   │   └── diff.zig          # Snapshot diff
│   └── build.zig
│
├── coco-net/                   # Network agent (Go + eBPF)
│   ├── cmd/
│   │   └── main.go           # Entry point
│   ├── agent.go               # Agent initialization
│   ├── xdp/                   # XDP programs (C)
│   │   ├── xdp_fwd.c         # Packet forwarding
│   │   ├── xdp_filter.c      # Packet filtering
│   │   ├── xdp_nat.c         # NAT
│   │   ├── xdp_stats.c       # Statistics
│   │   └── Makefile
│   ├── ebpf/                  # eBPF loader (Go)
│   │   ├── loader.go         # Program loading
│   │   ├── maps.go           # Map management
│   │   └── objects.go        # eBPF objects
│   ├── conntrack/             # Connection tracking
│   │   ├── conntrack.go
│   │   └── flow.go
│   ├── rate/                  # Rate limiting
│   │   ├── bucket.go         # Token bucket
│   │   └── limiter.go        # Rate limiter
│   ├── policy/               # Network policies
│   │   ├── engine.go        # Policy engine
│   │   ├── evaluator.go      # Rule evaluation
│   │   └── cache.go         # Policy cache
│   ├── ipam/                 # IP address management
│   │   ├── ipam.go
│   │   ├── allocator.go
│   │   └── pool.go
│   └── netns/                 # Network namespace
│       └── namespace.go
│
├── coco-checkpoint/            # Checkpoint engine
│   ├── cmd/
│   │   └── main.go
│   ├── checkpoint.go          # Checkpoint operations
│   ├── image/                 # Image handling
│   │   ├── builder.go
│   │   └── unpacker.go
│   ├── compress/              # Compression
│   │   ├── zstd.go          # Zstandard compression
│   │   └── compressor.go     # Compression interface
│   └── restore/               # Restoration
│       ├── restorer.go
│       └── loader.go
│
└── coco-proxy/                 # Request proxy
    ├── cmd/
    │   └── main.go
    ├── proxy.go                # Proxy logic
    ├── balancer.go             # Load balancing
    └── cache.go               # Response caching
```

## 4. Package Layer (pkg/)

Shared Go packages used by command and daemon components.

```
pkg/
├── api/                        # Generated API types
│   ├── v1/
│   │   ├── coco.pb.go        # Generated protobuf
│   │   ├── coco.pb.gw.go    # REST gateway
│   │   └── coco.connect.go   # Connect handlers
│   └── handlers/              # HTTP/gRPC handlers
│       ├── sandbox.go
│       ├── exec.go
│       └── template.go
│
├── types/                      # Core type definitions
│   ├── sandbox.go            # Sandbox resource
│   ├── template.go           # Template resource
│   ├── cluster.go            # Cluster types
│   ├── checkpoint.go         # Checkpoint types
│   ├── exec.go              # Execution types
│   └── node.go              # Node types
│
├── visor/                     # Visor client library
│   ├── client.go            # Unix socket client
│   ├── pool.go             # Connection pooling
│   ├── protocol.go         # Binary protocol
│   └── types.go             # Protocol types
│
├── scheduler/                 # Scheduling logic
│   ├── scheduler.go         # Main scheduler
│   ├── filters/            # Scheduling filters
│   │   ├── capacity.go     # Capacity filtering
│   │   ├── affinity.go     # Affinity rules
│   │   └── label.go       # Label filtering
│   └── strategies/         # Placement strategies
│       ├── leastloaded.go
│       ├── binpack.go
│       └── random.go
│
├── pool/                      # VM pool management
│   ├── pool.go              # Main pool logic
│   ├── prealloc.go         # Pre-allocation
│   ├── recycler.go          # VM recycling
│   ├── watcher.go          # Pool watcher
│   └── metrics.go          # Pool metrics
│
├── store/                    # State storage
│   ├── badger.go           # BadgerDB wrapper
│   ├── types.go           # Storage types
│   └── transaction.go     # Transaction support
│
├── net/                      # Network utilities
│   ├── tap.go              # TAP device management
│   ├── veth.go            # Virtual ethernet
│   ├── bridge.go           # Bridge management
│   ├── ipam.go            # IP allocation
│   └── ebpf_loader.go    # eBPF loading
│
├── template/                  # Template management
│   ├── manager.go          # Template operations
│   ├── store.go            # Template storage
│   ├── builder.go         # Template building
│   ├── oci.go             # OCI image handling
│   └── docker.go          # Docker image handling
│
├── cluster/                   # Cluster management
│   ├── manager.go          # Cluster operations
│   ├── node.go            # Node management
│   ├── discovery.go       # Node discovery
│   └── membership.go      # Membership tracking
│
├── metrics/                   # Metrics collection
│   ├── prometheus.go      # Prometheus export
│   ├── stats.go           # Statistics
│   └── collector.go       # Metric collection
│
├── crypto/                   # Cryptography
│   ├── crypto.go          # Encryption utilities
│   └── hash.go            # Hashing functions
│
├── replay/                   # Event replay
│   ├── recorder.go        # Event recording
│   ├── replayer.go       # Event replay
│   └── recorder.go       # Event store
│
├── middleware/              # HTTP middleware
│   ├── logging.go        # Structured logging
│   ├── tracing.go       # Distributed tracing
│   ├── recovery.go      # Panic recovery
│   ├── cors.go          # CORS handling
│   └── compress.go      # Compression
│
├── signer/                   # Request signing
│   ├── signer.go
│   └── verifier.go
│
├── time/                     # Time utilities
│   ├── duration.go
│   └── ticker.go
│
└── version/                  # Version information
    ├── version.go
    └── info.go
```

## 5. Protocol Buffers (proto/)

Protocol buffer definitions for API and internal communication.

```
proto/
├── coco/
│   └── v1/
│       ├── coco.proto           # Main API definitions
│       ├── sandbox.proto       # Sandbox resource
│       ├── exec.proto         # Execution messages
│       ├── template.proto     # Template messages
│       ├── cluster.proto      # Cluster messages
│       ├── checkpoint.proto   # Checkpoint messages
│       └── network.proto      # Network messages
│
└── internal/
    ├── visor.proto             # Visor IPC protocol
    └── agent.proto             # Agent protocol
```

## 6. eBPF Programs (ebpf/)

eBPF programs that run in the kernel for network processing.

```
ebpf/
├── from_sandbox/              # Sandbox to host
│   ├── from_sandbox.bpf.c
│   ├── maps.h
│   └── Makefile
│
├── from_host/                # Host to sandbox
│   ├── from_host.bpf.c
│   └── Makefile
│
├── xdp/                      # XDP programs
│   ├── xdp_fwd.bpf.c        # Packet forwarding
│   ├── xdp_filter.bpf.c     # Packet filtering
│   ├── xdp_nat.bpf.c        # NAT
│   ├── xdp_stats.bpf.c      # Statistics
│   ├── xdp_conntrack.bpf.c  # Connection tracking
│   ├── common.h              # Common definitions
│   └── Makefile
│
└── headers/                   # Shared headers
    ├── common.h
    ├── maps.h
    └── utils.h
```

## 7. Tests (test/)

Test suites for validation.

```
test/
├── unit/                      # Unit tests (Go)
│   ├── api_test.go
│   ├── scheduler_test.go
│   ├── pool_test.go
│   ├── visortest/
│   │   └── client_test.go
│   └── helpers.go
│
├── integration/               # Integration tests
│   ├── sandbox_test.go
│   ├── fork_test.go
│   ├── checkpoint_test.go
│   ├── cluster_test.go
│   ├── network_test.go
│   └── helpers.go
│
├── benchmark/                # Performance tests
│   ├── start_test.go
│   ├── fork_test.go
│   ├── memory_test.go
│   ├── network_test.go
│   └── benchmark_test.go
│
├── e2e/                     # End-to-end tests
│   ├── smoke_test.go
│   ├── full_test.go
│   └── helpers.go
│
└── zig/                     # Zig tests
    ├── vm_test.zig
    ├── kvm_test.zig
    └── helpers.zig
```

## 8. Configuration (configs/)

Configuration file templates.

```
configs/
├── default.yaml               # Default configuration
├── production.yaml           # Production overrides
├── development.yaml         # Development overrides
├── node.yaml               # Node-specific config
├── gateway.yaml            # Gateway config
├── master.yaml             # Master config
└── templates/              # Template configurations
    ├── ubuntu.yaml
    ├── alpine.yaml
    ├── debian.yaml
    └── custom.yaml
```

## 9. Scripts (scripts/)

Build and utility scripts.

```
scripts/
├── build/                    # Build scripts
│   ├── build-all.sh        # Build all binaries
│   ├── build-go.sh        # Build Go binaries
│   ├── build-zig.sh       # Build Zig binaries
│   └── build-ebpf.sh      # Build eBPF programs
│
├── test/                    # Test scripts
│   ├── test-all.sh        # Run all tests
│   ├── test-unit.sh       # Run unit tests
│   └── test-integration.sh
│
├── docker/                  # Docker scripts
│   ├── build-images.sh    # Build container images
│   └── push-images.sh    # Push images
│
└── tools/                   # Utility tools
    ├── generate-proto.sh   # Generate protobuf
    └── format-code.sh     # Format code
```

## 10. Deployment (deploy/)

Deployment manifests for various platforms.

```
deploy/
├── kubernetes/
│   ├── operator/            # K8s operator
│   │   ├── crd.yaml
│   │   ├── rbac.yaml
│   │   ├── controller.yaml
│   │   └── deployment.yaml
│   ├── manifests/           # Raw manifests
│   │   ├── gateway.yaml
│   │   ├── node.yaml
│   │   ├── master.yaml
│   │   └── namespace.yaml
│   └── helm/               # Helm charts
│       ├── coco/
│       │   ├── Chart.yaml
│       │   ├── values.yaml
│       │   └── templates/
│       └── coco-operator/
│
├── docker/                  # Docker builds
│   ├── Dockerfile.gateway
│   ├── Dockerfile.node
│   ├── Dockerfile.master
│   ├── Dockerfile.visor
│   ├── Dockerfile.agent
│   ├── Dockerfile.net
│   └── docker-compose.yml
│
└── ansible/                 # Ansible playbooks
    └── playbooks/
        ├── coco.yml
        ├── node.yml
        └── master.yml
```

## 11. Language Summary

| Layer | Component | Language |
|-------|-----------|----------|
| cmd | coco-gateway | Go |
| cmd | coco-node | Go |
| cmd | coco-master | Go |
| cmd | cococtl | Go |
| daemon | coco-visor | Zig |
| daemon | coco-agent | Zig |
| daemon | coco-fork | Zig |
| daemon | coco-net | Go + C |
| daemon | coco-checkpoint | Go + Zig |
| pkg | All packages | Go |
| ebpf | All programs | C |

## 12. Build Outputs

| Binary | Location | Purpose |
|--------|----------|---------|
| coco-gateway | bin/ | API server |
| coco-node | bin/ | Node daemon |
| coco-master | bin/ | Cluster master |
| cococtl | bin/ | CLI client |
| coco-visor | bin/ | Hypervisor |
| coco-agent | bin/ | Guest agent |
| coco-net | bin/ | Network daemon |
| coco-checkpoint | bin/ | Checkpoint tool |

## 13. File Naming Conventions

Go files use lowercase with underscores: sandbox_service.go
Zig files use lowercase with underscores: sandbox_service.zig
Protocol buffers use lowercase: sandbox.proto
Configuration uses lowercase with dashes: production.yaml
eBPF programs use lowercase: xdp_filter.bpf.c

All file names should be descriptive and indicate purpose.
