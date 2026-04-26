# Coco Sandbox – Performance specification

**Scope:** SLO-style targets, optimization techniques, and how benchmarks relate to this document.  
**Status:** Authoritative for targets; exact CLI names live in the repository.  
**Index:** [Specification index](index.md)

## 1. Performance targets

These numbers are **implementation SLOs and design targets** for a reference stack (pooling, btrfs reflinks where available, XDP, and so on). They are not guaranteed for every hardware profile, build flag, or load pattern. Validate with the benchmarking tools shipped in the repository (for example `cococtl` subcommands or tests), not by copying invocations from this file.

### 1.1 Latency Targets

Cold start latency measures the time from a create request to a sandbox ready to execute commands. The target is under 30 milliseconds. This is achieved through VM pool pre-creation.

Fork latency measures the time from a fork request to a child sandbox ready to execute. The target is under 10 milliseconds. This is achieved through btrfs reflink snapshots.

Resume from hibernate latency measures the time from a resume request to a sandbox ready to execute. The target is under 5 milliseconds. This is achieved through pre-loaded snapshots.

Exec latency measures the overhead of starting command execution after a request. The target is under 1 millisecond. This is achieved through VSock.

Network round-trip latency measures the time for a packet to go from host to guest and back. The target is under 0.5 milliseconds. This is achieved through VSock and XDP.

### 1.2 Throughput Targets

Sandbox creation throughput measures the sustained rate of sandbox creation. The target is 33 sandboxes per second. This assumes a pool of pre-created VMs.

Fork throughput measures the sustained rate of fork operations. The target is 100 forks per second. This assumes COW snapshots.

Network throughput measures aggregate bandwidth. The target is 20 gigabits per second. This assumes XDP processing.

Maximum concurrent sandboxes per node is 2000. This assumes adequate memory and CPU resources.

### 1.3 Resource Efficiency

Memory overhead is the memory used by Coco itself beyond the configured VM memory. The target is under 2 megabytes per sandbox.

Disk usage for base templates is approximately 50 megabytes. This assumes a minimal Linux distribution.

CPU overhead is the percentage of host CPU used for management operations. The target is under 0.5 percent.

## 2. Optimization Techniques

Multiple optimization techniques achieve these targets.

### 2.1 VM Pool Pre-Creation

Nodes maintain a pool of pre-created VMs ready for immediate assignment. When a sandbox is requested, the pool provides a ready VM instead of creating one.

Pool size is configurable based on available memory. The default is enough VMs to handle expected creation rates with some headroom.

When a sandbox is deleted, its VM is reset and returned to the pool for recycling. This avoids the overhead of VM destruction and creation.

### 2.2 COW Fork

Fork operations use btrfs reflinks to create copy-on-write snapshots. When a VM is forked, the filesystem creates a reflink rather than copying data.

Memory pages are initially shared between parent and child. When either writes to a page, the write triggers a copy. This makes fork operations extremely fast.

The fork target of under 10 milliseconds includes creating the snapshot, starting the new VM, and establishing VSock connectivity.

### 2.3 Zero-Copy Networking

XDP processes packets at the earliest possible point in the Linux network stack, directly at the network driver. This avoids the overhead of the main network stack.

Packets are processed without allocating socket buffers. This reduces memory allocation overhead and improves cache efficiency.

The XDP program makes accept/deny decisions in the kernel, before packets consume resources in the host.

### 2.4 VSock Communication

VSock provides direct communication between host and guest. Unlike TCP/IP, VSock bypasses the entire network stack.

The overhead is minimal because communication happens through hypervisor buffers. There is no kernel network stack involvement.

For development environments where VSock is unavailable, Agent falls back to TCP on localhost.

### 2.5 Memory Deduplication

KSM (Kernel Samepage Merging) automatically identifies identical memory pages across VMs. The pages are merged into a single read-only page.

When either VM writes to the page, a copy is made. This is transparent to the VMs.

Memory deduplication typically reduces overall memory usage by 10-20 percent, depending on how similar the sandboxes are.

### 2.6 Binary Size Optimization

Agent is compiled as a static binary with minimal runtime. The target size is under one megabyte.

A smaller binary reduces memory footprint, attack surface, and disk usage. It also improves load time.

Zig provides excellent control over binary size through its manual memory management and lack of runtime.

## 3. Benchmarking

Performance is validated through **repeatable benchmark scenarios** in the tree. The spec defines **what** to measure and **SLO percentiles**; the repository defines **exact commands**, flags, and test harnesses (they may change between releases without invalidating the SLOs below).

| Scenario | SLO (percentile targets) | What is measured |
|----------|--------------------------|------------------|
| Cold start | P50 under 30ms, P95 under 40ms, P99 under 50ms (creation to ready) | Per-create latency over many runs |
| Fork | P50 under 10ms, P95 under 15ms, P99 under 20ms | Per-fork latency |
| Memory overhead | Control-plane overhead under 2 MiB per sandbox in reference setup | Host RSS / accounting minus guest working set |
| Network | Aggregate under 20 Gbps and sub-ms RTT in reference setup | Throughput and RTT under load |

## 4. Scalability

Coco scales horizontally across multiple nodes.

### 4.1 Single Node Capacity

A single node can run up to 2000 sandboxes, limited by memory and CPU. With 128GB memory and 64 cores, a node can run 2000 sandboxes with 512MB and 1 vCPU each.

For larger sandboxes, capacity decreases proportionally. A node running 4GB sandboxes can run approximately 30 sandboxes.

### 4.2 Cluster Capacity

A cluster of 10 nodes can run 20,000 sandboxes. The Master coordinates scheduling across nodes.

Clusters can scale to 100 nodes and beyond, limited by etcd performance and network bandwidth.

### 4.3 Scaling Strategies

Horizontal scaling adds more nodes to increase capacity. Each node runs the Node component and connects to Master.

Vertical scaling increases resources on existing nodes. More memory and CPU allow more sandboxes per node.

## 5. Performance Monitoring

Performance is continuously monitored through metrics.

### 5.1 Key Metrics

Sandbox creation duration tracks how long each creation takes. This is reported as a histogram with percentiles.

Fork duration tracks how long each fork takes. This is reported as a histogram with percentiles.

Active sandbox count tracks how many sandboxes are currently running. This is reported as a gauge.

Pool availability tracks how many VMs are available in the pool. This is reported as a gauge.

Memory usage tracks how much memory is in use. This is reported as a gauge.

### 5.2 Dashboards

Grafana dashboards visualize performance data. The main dashboard shows current performance against targets.

Panels include creation latency percentiles, fork latency percentiles, active sandboxes by node, pool utilization, and memory usage.

### 5.3 Alerts

Alerts trigger when performance degrades. An alert fires when P99 creation latency exceeds 50ms for more than five minutes.

Alerts also fire when pool utilization drops below 10%, indicating potential capacity issues.

## 6. Qualitative comparison (informative)

Rough qualitative tradeoffs that motivate the architecture (not a benchmark of other products):

| Concern | Typical container | Typical full VM | Coco (as specified) |
|--------|-------------------|-----------------|----------------------|
| Shared host kernel | Yes | No | No (guest kernel per sandbox) |
| Cold path / pool | Varies | Often slower | Targets low latency via pool + COW (see sections 1–2) |
| Network enforcement | Often later in stack | Varies | Default-deny at XDP with policy (see `03-network.md`) |

For numeric targets, use section 1 and the benchmarking section of this file, not informal comparisons to unnamed deployments.
