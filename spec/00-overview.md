# Coco Sandbox - Specification Overview

**Status:** Approved Specification

---

## 1. Executive Summary

Coco is a next-generation sandbox runtime that provides hardware-level isolation with extreme performance. Built with Go for control plane and Zig for data plane, Coco is designed to beat CubeSandbox in every dimension while maintaining a cleaner, more maintainable codebase.

The primary goal of Coco is to provide a secure, high-performance sandbox environment for AI agents and code execution workloads. Coco achieves this through careful optimization of every component, from VM startup time to network throughput.

### 1.1 Core Philosophy

**Performance First**: Every component is optimized for speed. Cold start under 30 milliseconds, fork operations under 10 milliseconds, and memory overhead under 2 megabytes per sandbox.

**Security by Default**: Multiple layers of isolation ensure maximum security. Network filtering at the kernel level, hardware virtualization through KVM, and optional hardware enclaves through TDX and SGX.

**Developer Experience**: Clean APIs, comprehensive documentation, and native SDKs make integration straightforward. E2B compatibility mode ensures easy migration.

### 1.2 Comparison with CubeSandbox

Coco beats Cube in every measurable dimension. Cold start is twice as fast, fork operations are ten times faster, and memory overhead is sixty percent lower. These improvements come from careful architectural choices rather than incremental optimizations.

The choice of Go and Zig over Rust provides better maintainability. Go offers excellent networking and concurrency primitives for the control plane, while Zig provides direct memory control and minimal binary size for the data plane. Both languages have simpler compilation models than Rust, reducing build times and complexity.

---

## 2. High-Level Architecture

Coco follows a layered architecture that separates concerns cleanly. The control plane handles API serving, scheduling, and cluster management in Go. The data plane handles VM lifecycle, execution, and networking in Zig. This separation allows each component to be optimized for its specific purpose.

### 2.1 System Components

The system consists of six primary components that work together to provide sandbox isolation. Each component has a clear responsibility and communicates with others through well-defined interfaces.

**Coco Gateway** serves as the entry point for all external requests. It exposes a REST API compatible with the E2B standard, handles authentication and rate limiting, and forwards requests to appropriate internal components. Gateway is written in Go to leverage its excellent HTTP handling and middleware ecosystem.

**Coco Master** coordinates cluster-wide operations. It maintains cluster state in etcd, performs leader election for high availability, and schedules sandbox creation across nodes. Master ensures the cluster operates smoothly even during node failures.

**Coco Node** runs on each host machine and manages local resources. It maintains a pool of pre-created VMs for instant spawning, tracks resource usage, and communicates with the local Visor process. Node handles the actual sandbox lifecycle on its host.

**Coco Visor** manages individual virtual machines. Written in Zig, it interfaces directly with KVM to create and control MicroVMs. Visor handles VM creation, execution, forking, checkpointing, and cleanup. Its minimal binary size and direct memory control make it extremely efficient.

**Coco Agent** runs inside each sandbox as the init process. Also written in Zig, it handles command execution, signal propagation, and communication with the host through VSock. Agent is designed to be extremely small, under one megabyte, to minimize attack surface and memory usage.

**Coco Net** provides network isolation and routing. It combines Go for control logic with eBPF programs for packet processing at the kernel level. Net implements default-deny policies, rate limiting, and flow tracking.

### 2.2 Communication Patterns

Components communicate through different protocols depending on their roles and performance requirements.

External clients communicate with Gateway through REST over HTTP. This provides broad compatibility with existing tools and SDKs.

Gateway communicates with Master and Node components through gRPC. This provides efficient binary serialization and streaming support for internal operations.

Node communicates with Visor through a Unix domain socket using a custom binary protocol. This minimizes latency for VM operations.

Visor communicates with Agent through VSock when available, falling back to TCP for development environments. VSock provides near-zero-overhead communication between host and guest.

---

## 3. Component Specifications

This section provides detailed specifications for each component, including their responsibilities, interfaces, and internal design.

### 3.1 Coco Gateway

Gateway is the external-facing API server. It accepts requests from clients, validates them, applies rate limits, and forwards them to appropriate internal components.

**Responsibilities**: HTTP server management, request validation, authentication, rate limiting, metrics collection, request routing.

**External Interface**: REST API over HTTP on port 8080 by default. Supports E2B-compatible endpoints for seamless migration.

**Internal Interface**: Connects to Master for scheduling decisions and to Nodes for sandbox operations.

**Design Decisions**: Gateway maintains no persistent state. All state is stored in etcd or on Nodes. This allows Gateway to be scaled horizontally behind a load balancer. Rate limiting uses a token bucket algorithm per API key. Authentication supports API keys and OAuth2 tokens.

**Dependencies**: Connect for gRPC compatibility, standard library HTTP server, Prometheus client for metrics.

### 3.2 Coco Master

Master provides cluster coordination and scheduling. It ensures the cluster remains available even when individual nodes fail.

**Responsibilities**: Leader election, cluster state management, sandbox scheduling, failover coordination, resource tracking.

**External Interface**: No direct external interface. Gateway connects to Master for scheduling decisions.

**Internal Interface**: Connects to etcd for state storage and to Nodes for task distribution.

**Design Decisions**: Master uses etcd with Raft consensus for leader election. When the leader fails, etcd automatically promotes a new leader within seconds. The scheduler considers node capacity, current load, and sandbox requirements when making placement decisions. It supports multiple scheduling strategies including least loaded, binpack, and random.

**Dependencies**: etcd client for state management, Raft implementation for consensus.

### 3.3 Coco Node

Node manages local resources on a single host. It bridges the gap between cluster-level requests and local VM operations.

**Responsibilities**: VM pool management, resource tracking, local scheduling, health monitoring, Visor communication.

**External Interface**: No direct external interface. Master sends requests to Node through gRPC.

**Internal Interface**: Communicates with local Visor through Unix socket and with Master through gRPC.

**Design Decisions**: Node maintains a pool of pre-created VMs ready for instant assignment. When a sandbox is requested, Node assigns a pre-created VM instead of creating one from scratch, achieving sub-30ms cold starts. When a sandbox is deleted, the VM is returned to the pool for recycling rather than being destroyed. Pool size is configurable based on available memory.

Node continuously monitors resource usage including memory, CPU, and disk. It reports this information to Master for global scheduling decisions. Health checks verify Visor responsiveness and VM availability.

**Dependencies**: Visor client for VM operations, BadgerDB for local state, Prometheus for metrics.

### 3.4 Coco Visor

Visor is the heart of the sandbox runtime. It directly manages KVM-based MicroVMs, handling creation, execution, forking, and cleanup.

**Responsibilities**: KVM interface, VM lifecycle, memory management, vCPU scheduling, VSock setup, snapshot creation, checkpoint handling.

**External Interface**: Unix socket at /run/coco/visor.sock. Node communicates with Visor through this socket.

**Internal Interface**: Communicates with Agent through VSock inside the guest.

**Design Decisions**: Visor is written in Zig to achieve minimal binary size and direct memory control. It interfaces directly with KVM syscalls rather than using a wrapper like Cloud Hypervisor. This reduces latency and overhead.

VM creation involves setting up memory regions, configuring vCPUs, and launching the kernel. Each VM receives a unique VSock Context ID for host communication.

Fork operations use btrfs reflinks to create copy-on-write snapshots. This allows forking in under 10 milliseconds because no data is copied initially. Memory pages are shared between parent and child until either writes to them.

Checkpoint operations save VM state to disk for later restoration. Memory pages are compressed using zstd for fast compression with good ratios. Checkpoints support incremental mode where only changed pages are saved after the first checkpoint.

**Dependencies**: Linux kernel headers, KVM module, btrfs for snapshots.

### 3.5 Coco Agent

Agent runs inside each sandbox as the init process. It handles command execution and host communication from inside the guest.

**Responsibilities**: Signal handling, process reaping, command execution, output streaming, VSock communication.

**External Interface**: No external interface. Runs as PID 1 inside the sandbox.

**Internal Interface**: Connects to Visor through VSock for host communication.

**Design Decisions**: Agent is written in Zig to minimize binary size to under one megabyte. This reduces the attack surface and memory footprint of every sandbox.

As PID 1, Agent must handle zombie reaping, signal propagation, and orphaned process adoption. It implements proper SIGTERM handling for graceful shutdown, forwarding the signal to child processes and waiting for them to exit.

Command execution involves spawning the requested process, capturing its stdout and stderr, and streaming the output back to the host through VSock. Exit codes are captured and returned to the caller.

Agent supports both VSock for production and TCP fallback for development. When VSock is unavailable, it connects to 127.0.0.1 on the configured port.

**Dependencies**: Standard Zig library, POSIX bindings.

### 3.6 Coco Net

Net provides network isolation and policy enforcement through a combination of Go control logic and eBPF kernel programs.

**Responsibilities**: Packet filtering, traffic shaping, NAT, flow tracking, network policy enforcement, IP address management.

**External Interface**: No direct external interface. Works at the kernel level.

**Internal Interface**: Interfaces with the host network stack and manages per-sandbox network configuration.

**Design Decisions**: Net uses XDP, the Express Data Path, to process packets at the earliest possible point in the Linux network stack. This provides near-native performance with zero copy between the network card and the filter.

All egress traffic is denied by default. Explicit policy rules must be created to allow traffic. This follows the principle of least privilege.

The XDP filter program runs at the driver level, before the kernel's main network stack. This provides protection against DDoS attacks and allows for high-throughput packet processing.

Rate limiting uses a token bucket algorithm implemented in eBPF. Each sandbox has configurable limits on packets per second and bytes per second with burst allowances.

Flow tracking maintains connection state in eBPF maps. This allows proper handling of multi-packet flows and enables connection tracking.

IP address allocation uses a simple pool managed by Net. Each sandbox receives an IP address from the pool when created, and the address is returned to the pool when the sandbox is destroyed.

---

## 4. Network Architecture

Network architecture provides secure, high-performance networking for sandboxes while maintaining strict isolation between them.

### 4.1 Network Modes

Coco supports multiple network modes to accommodate different use cases.

**Bridge Mode**: The default mode. Sandboxes share a bridge device with NAT to the host network. This provides full internet access while maintaining isolation between sandboxes. Performance is good but not optimal.

**Tap Mode**: Each sandbox gets its own TAP device. This provides near-native network performance but requires more setup. Recommended for high-throughput workloads.

**VSock Mode**: Sandboxes have VSock connectivity only, no IP networking. Communication is limited to the host through VSock. Use this for maximum isolation.

**Isolated Mode**: No network connectivity whatsoever. The sandbox cannot communicate with any external system. Use this for untrusted workloads.

### 4.2 eBPF Programs

Net uses three eBPF programs working together to provide comprehensive network functionality.

**XDP Filter**: Processes incoming and outgoing packets at the earliest possible point. Implements default-deny policy, checking each packet against allowed rules. Packets not matching any rule are dropped. Also implements rate limiting using token bucket.

**Flow Table**: Maintains connection state in an eBPF hash map. Tracks packets, bytes, and timestamps for each flow. Enables proper handling of multi-packet connections and connection tracking.

**Traffic Shaper**: Implements quality of service by queuing packets based on sandbox priority. Ensures fair bandwidth distribution when multiple sandboxes compete for network resources.

### 4.3 VSock Communication

VSock provides direct communication between host and guest with minimal overhead.

**Addressing**: VSock uses a Context ID (CID) to identify endpoints. CID 2 is reserved for the host. Guest VMs receive unique CIDs starting from 3.

**Port**: All VSock communication uses port 4747 by default. This can be overridden through environment variables.

**Fallback**: When VSock is unavailable, Agent falls back to TCP on 127.0.0.1. This simplifies development and testing.

---

## 5. Security Architecture

Security is fundamental to Coco's design. Multiple layers of defense ensure that sandboxes remain isolated from each other and from the host.

### 5.1 Isolation Layers

Coco implements defense in depth with four distinct isolation layers.

**Network Isolation**: Handled by Net through eBPF. All egress is denied by default. Even if an attacker compromises a sandbox, they cannot reach other sandboxes or external systems without explicit policy.

**VM Isolation**: Provided by KVM hardware virtualization. Each sandbox runs in its own MicroVM with separate kernel, memory, and devices. No resource sharing between sandboxes.

**Hardware Enclaves**: Optional support for TDX, SGX, and SEV provides additional protection for sensitive workloads. These technologies encrypt memory and provide hardware-based attestation.

**Syscall Filtering**: Seccomp filtering restricts available system calls to a minimal set. Only necessary syscalls like read, write, and execve are allowed.

### 5.2 Capability Management

The guest runs with minimal capabilities. All dangerous capabilities are dropped including CAP_SYS_ADMIN, CAP_NET_ADMIN, CAP_SYS_MODULE, and CAP_SYS_PTRACE.

### 5.3 Resource Limits

Each sandbox can be configured with limits on memory, CPU, and I/O. These limits are enforced through cgroups and cannot be exceeded by the sandbox.

**Memory Limit**: Hard limit on usable memory. OOM behavior is configurable.

**CPU Quota**: CPU bandwidth limit using CFS bandwidth control. Ensures fair CPU sharing between sandboxes.

**I/O Limits**: Throttled reads and writes using cgroup blkio controller. Prevents sandbox from monopolizing disk bandwidth.

---

## 6. Storage Architecture

Storage handles template images, checkpoint data, and runtime state.

### 6.1 Template Storage

Templates define the base image and configuration for sandboxes. Template storage must support fast image loading and efficient disk usage.

**Location**: /var/lib/coco/templates/ by default.

**Format**: Directory containing rootfs, kernel, initrd, and metadata.json.

**Compression**: Rootfs can be compressed with gzip or zstd. Decompression happens at load time.

**Deduplication**: Multiple templates sharing base layers can use btrfs reflinks to share disk space.

### 6.2 Checkpoint Storage

Checkpoints save complete VM state for later restoration. Storage must handle large files efficiently.

**Location**: /var/lib/coco/checkpoints/ by default.

**Format**: Directory containing compressed memory image, CPU state, device state, and metadata.json.

**Compression**: Memory pages are compressed with zstd for fast compression with good ratios. Compression happens in parallel across multiple cores.

**Incremental**: After the first checkpoint, subsequent checkpoints can be incremental, storing only changed pages. This dramatically reduces storage requirements and checkpoint time for frequently checkpointed VMs.

### 6.3 Runtime Storage

Runtime storage maintains sandbox metadata and cluster state.

**Local State**: BadgerDB stores sandbox metadata on each Node. This includes sandbox ID, state, configuration, and resource usage.

**Cluster State**: etcd stores cluster-wide state including node registry, sandbox registry, and scheduling queue.

---

## 7. API Specification

The API provides programmatic access to all Coco functionality. The public API uses REST over HTTP for maximum compatibility.

### 7.1 Sandbox Operations

**Create Sandbox**: Creates a new sandbox from a template. Returns sandbox ID and initial state.

**Get Sandbox**: Retrieves sandbox details by ID. Includes state, configuration, and resource usage.

**List Sandboxes**: Returns all sandboxes, optionally filtered by state, node, or labels.

**Delete Sandbox**: Permanently removes a sandbox and releases its resources.

**Pause Sandbox**: Suspends a running sandbox. Memory is preserved.

**Resume Sandbox**: Continues execution of a paused sandbox.

**Hibernate Sandbox**: Saves sandbox state to disk and stops it. State can be restored later.

**Restore Sandbox**: Creates a new sandbox from a checkpoint.

**Fork Sandbox**: Creates a copy of a running sandbox using COW snapshots.

### 7.2 Execution Operations

**Exec**: Executes a command in a sandbox. Returns stdout, stderr, and exit code. For short-running commands.

**Streaming Exec**: Executes a command and streams output as it becomes available. For long-running or interactive commands.

### 7.3 Template Operations

**List Templates**: Returns all available templates.

**Create Template**: Creates a new template from an OCI image or directory.

**Delete Template**: Removes a template and its associated files.

### 7.4 Cluster Operations

**Get Cluster Info**: Returns cluster-wide statistics including node count, sandbox count, and health status.

**List Nodes**: Returns all nodes in the cluster with their status and capacity.

**Drain Node**: Marks a node as draining, preventing new sandbox placement and allowing existing sandboxes to complete.

---

## 8. Performance Specifications

Performance is a primary design goal. Coco is optimized for minimal latency and maximum throughput.

### 8.1 Latency Targets

**Cold Start**: Under 30 milliseconds from request to sandbox ready. Achieved through VM pool pre-creation.

**Fork**: Under 10 milliseconds from request to child sandbox ready. Achieved through btrfs reflink snapshots.

**Resume from Hibernate**: Under 5 milliseconds. Achieved through pre-loaded snapshots.

**Exec Latency**: Under 1 millisecond from request to command start. Achieved through VSock.

**Network Round Trip**: Under 0.5 milliseconds host-to-guest-to-host. Achieved through VSock and XDP.

### 8.2 Throughput Targets

**Sandbox Creation**: 33 sandboxes per second sustained.

**Fork Operations**: 100 forks per second sustained.

**Network Throughput**: 20 gigabits per second aggregate.

**Maximum Concurrent Sandboxes**: 2000 per node.

### 8.3 Resource Efficiency

**Memory Overhead**: Under 2 megabytes per sandbox beyond the configured VM memory.

**Disk Usage**: Approximately 50 megabytes for base templates.

**CPU Overhead**: Under 0.5 percent of host CPU for management operations.

### 8.4 Optimization Techniques

**VM Pool**: Nodes maintain a pool of pre-created VMs ready for immediate assignment. This eliminates the kernel boot time from the critical path.

**COW Fork**: Btrfs reflink creates instant snapshots without copying data. Memory pages are shared between parent and child until written.

**Zero-Copy Networking**: XDP processes packets at the earliest point in the stack, avoiding expensive skb allocations.

**VSock**: VSock provides near-zero-overhead communication between host and guest, bypassing the TCP/IP stack entirely.

**Memory Deduplication**: KSM merges identical memory pages across VMs, reducing overall memory usage by 10-20 percent.

---

## 9. Cluster Architecture

Coco operates as a distributed system across multiple machines for high availability and capacity.

### 9.1 Master Election

Cluster uses etcd with Raft consensus for leader election. One Master acts as leader, handling all scheduling decisions. Others remain available as hot standbys.

When the leader fails, etcd automatically promotes a new leader within seconds. The promotion process is transparent to clients; requests continue to be served during the transition.

### 9.2 Node Discovery

Nodes register themselves in etcd on startup with their capacity and current load. They maintain an ephemeral key with TTL that must be periodically renewed.

Master monitors node health through the TTL mechanism. If a node fails to renew its key, it is marked as unavailable and its sandboxes are rescheduled.

### 9.3 Scheduling

When a sandbox creation request arrives, Master selects the optimal node based on current capacity, load, and sandbox requirements. The request is then forwarded to that node for execution.

Scheduling supports multiple strategies. Least loaded places sandboxes on nodes with most available resources. Binpack packs sandboxes tightly to minimize resource fragmentation. Random distributes load evenly for better fault tolerance.

### 9.4 Failover

When a node fails, Master detects the failure through missing heartbeats. All sandboxes on that node are marked as failed. If checkpoints exist, sandboxes are automatically restored on other nodes.

Client applications are notified of sandbox failures through the API. They can choose to recreate sandboxes or handle the failure as appropriate for their use case.

### 9.5 Kubernetes Integration

Coco provides a Kubernetes operator for native integration. The operator manages Sandbox custom resources, handling creation, updating, and deletion according to user specifications.

Helm charts simplify deployment on existing Kubernetes clusters. The operator watches for Sandbox resources and communicates with the Coco cluster to manage sandbox lifecycle.

---

## 10. Observability

Comprehensive observability helps operators understand system behavior and diagnose issues.

### 10.1 Logging

All components produce structured JSON logs with consistent formatting. Each log entry includes timestamp, level, component, message, and contextual fields like sandbox ID and request ID.

Log levels are DEBUG, INFO, WARN, and ERROR. Production typically runs at INFO, with DEBUG available for troubleshooting.

Logs are written to stdout in development and to files or syslog in production.

### 10.2 Metrics

Prometheus metrics are exposed on port 9090 by default. Key metrics include sandbox creation duration, fork duration, active sandbox count, pool availability, and resource usage.

Metrics include histogram_quantiles for latency measurement and gauges for current state. All metrics include appropriate labels for filtering by node, state, and template.

### 10.3 Tracing

OpenTelemetry tracing provides distributed tracing across component boundaries. Each operation generates spans that can be correlated across the request lifecycle.

Tracing is exported via OTLP to compatible backends like Jaeger or Tempo. Sampling rates are configurable to balance overhead and visibility.

### 10.4 Health Checks

Components expose liveness and readiness probes. Liveness indicates the process is running. Readiness indicates the component can serve requests, including any dependencies being available.

Health checks are available at /health/live and /health/ready HTTP endpoints.

---

## 11. Repository Structure

The repository is organized to clearly separate components while maintaining coherence. Go code lives in cmd and pkg directories. Zig code lives in daemon directories. Shared definitions are in proto.

```
coco/
├── cmd/                    # Executable commands
│   ├── coco-gateway/      # API server
│   ├── coco-node/         # Node daemon
│   ├── coco-master/        # Cluster master
│   └── cococtl/           # CLI client
├── daemon/                  # Long-running services
│   ├── coco-visor/        # Hypervisor (Zig)
│   ├── coco-agent/        # Guest agent (Zig)
│   ├── coco-fork/         # Fork engine (Zig)
│   ├── coco-net/          # Network (Go + eBPF)
│   └── coco-checkpoint/   # Checkpoint (Go + Zig)
├── pkg/                    # Shared packages
│   ├── api/               # Generated API code
│   ├── visor/             # Visor client
│   ├── scheduler/         # Scheduling logic
│   ├── pool/              # VM pool management
│   └── ...                # Other packages
├── proto/                  # Protocol buffers
├── ebpf/                   # eBPF programs
├── test/                   # Tests
├── configs/                # Configuration files
├── deploy/                 # Deployment manifests
└── spec/                  # This specification
```

### 11.1 Go Components

Go handles control plane components that benefit from its excellent networking and concurrency features. These include Gateway, Master, Node, and parts of Net.

### 11.2 Zig Components

Zig handles data plane components that benefit from direct memory control and minimal binary size. These include Visor, Agent, and fork/checkpoint logic.

### 11.3 eBPF Components

C handles eBPF programs that run in the kernel. These include XDP filters, flow tracking, and traffic shaping.

---

## 12. Dependencies

### 12.1 Build Dependencies

**Go**: Version 1.21 or later for control plane components.

**Zig**: Version 0.12 or later for data plane components.

**Clang**: For compiling eBPF programs.

**Protocol Buffer Compiler**: For generating API code from proto files.

### 12.2 System Requirements

**Linux Kernel**: Version 5.10 or later with KVM module loaded.

**Filesystem**: btrfs recommended for reflink support. ext4 and xfs also work.

**Memory**: 4GB minimum per node, 8GB recommended.

**Disk**: 50GB minimum for templates and checkpoints.

**CPU**: Virtualization support (VT-x or AMD-V) required.

### 12.3 Runtime Dependencies

**etcd**: For cluster state and leader election.

**btrfs-progs**: For snapshot operations.

**libbpf**: For loading eBPF programs.

---

## 13. Roadmap

Coco development proceeds in phases, each delivering complete functionality while building toward the full vision.

### Phase 1: Foundation

The first phase establishes core functionality. This includes VM lifecycle management through Visor, basic REST API through Gateway, simple fork operations, VSock communication between host and guest, and single-node deployment. The goal is a working end-to-end system that demonstrates the core architecture.

### Phase 2: Performance

The second phase focuses on optimization. VM pool pre-allocation achieves sub-30ms cold starts. COW snapshots achieve sub-10ms fork operations. eBPF networking provides security and performance. Metrics and logging provide operational visibility.

### Phase 3: Enterprise

The third phase adds enterprise features. Multi-node clusters provide high availability and capacity. Checkpoint and restore enable disaster recovery and workload migration. TDX and SGX support provide hardware-based security. Kubernetes integration enables cloud-native deployment.

### Phase 4: Advanced

The fourth phase explores advanced capabilities. Live migration allows moving running sandboxes between nodes. Event replay enables debugging and analysis. Auto-scaling responds to demand changes automatically.

---

## 14. Conclusion

Coco represents a new generation of sandbox technology, designed from the ground up for performance, security, and maintainability. By combining Go and Zig, Coco achieves capabilities that exceed existing solutions while maintaining a cleaner, more approachable codebase.

The comprehensive specifications in this document provide the foundation for development. They define not just what Coco does, but how each component achieves its goals. With these specifications as guidance, developers can implement features with confidence that they align with the overall architecture.

Coco is designed to be extended. The modular architecture allows for new features like additional hardware enclave support, enhanced checkpoint formats, or alternative network implementations. The specifications provide the framework within which such extensions can be evaluated and integrated.
