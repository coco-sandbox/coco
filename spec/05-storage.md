# Coco Sandbox – Storage specification

**Scope:** Template, checkpoint, and runtime storage layout and behavior.  
**Status:** Authoritative.  
**Index:** [Specification index](index.md)

## 1. Storage architecture overview

Coco manages three types of data: template images that define sandbox configurations, checkpoint data that preserves VM state, and runtime metadata that tracks sandbox lifecycle. Each type has different performance requirements and storage characteristics.

Template storage prioritizes read performance, as templates are loaded frequently. Checkpoint storage must handle large files efficiently, both for creation and restoration. Runtime storage requires fast random access for metadata operations.

## 2. Template Storage

Templates define the base configuration for sandboxes. They include the root filesystem, kernel, initrd, and default parameters.

### 2.1 Storage Location

Templates are stored under a **deployment-defined** root (commonly a path under the node data directory). The exact path is configuration, not fixed by this spec (see `10-self-hosting-and-operations.md`).

### 2.2 Template directory layout

Each template is a directory whose name is the template id or a subdirectory identified in configuration. Required artifacts:

| Path (relative to template root) | Purpose |
|----------------------------------|--------|
| metadata | File or embedded record with id, name, description, optional OS metadata, default memory and vCPU, creation time, size |
| rootfs | Directory, CPIO archive, or compressed archive of the guest root |
| vmlinuz | Guest kernel image |
| initrd.img | Initial ramdisk for the guest boot (if used) |

The **metadata** document includes at least: **id** (string), **name** (string), **description** (string), optional **os** (type, distribution, version as strings), **defaults** (memory_mb, vcpus as integers), **created_at** (timestamp), **size_bytes** (int64).

### 2.3 Root filesystem

The root filesystem contains the files visible to processes inside the sandbox. It can be a directory, a CPIO archive, or a compressed archive.

For performance, the root filesystem is loaded into memory when the sandbox starts. The memory requirement is the template size plus the configured sandbox memory.

### 2.4 Kernel and Initrd

The kernel (vmlinuz) and initial ramdisk (initrd.img) are loaded by the hypervisor when starting the VM. These files must be compatible with the KVM hypervisor.

### 2.5 Template Compression

Templates can be compressed to save disk space. Compression happens at creation time. Supported formats include gzip and zstd.

zstd provides the best balance between compression ratio and decompression speed. It typically achieves three to six times compression with fast decompression.

### 2.6 Template Deduplication

When multiple templates share common base layers, btrfs reflinks can share disk blocks between them. This reduces storage requirements without affecting performance.

Deduplication happens automatically when templates are created on btrfs filesystems. No configuration is required.

## 3. Checkpoint Storage

Checkpoints preserve complete VM state for later restoration. They enable use cases including disaster recovery, migration, and state snapshots.

### 3.1 Storage Location

Checkpoints are stored under a **deployment-defined** base directory, often organized by sandbox id. Layout on disk is implementation-specific; logically each checkpoint contains the components in §3.2.

### 3.2 Checkpoint components

A complete checkpoint consists of several components.

The memory image contains all RAM contents, compressed with zstd. This is typically the largest component.

The CPU state includes register contents, page table mappings, and other CPU-specific data needed to resume execution.

The device state captures the state of virtual devices like VSock and virtual block devices.

The metadata includes checkpoint timestamp, sandbox configuration, and checksums for integrity verification.

### 3.3 Compression

Memory pages are compressed with zstd before being written to disk. Compression happens in parallel across multiple CPU cores to minimize checkpoint time.

The compression level is configurable. Higher levels achieve better ratios but take longer. The default level provides good compression with fast compression time.

### 3.4 Incremental Checkpoints

After the first checkpoint, subsequent checkpoints can be incremental, storing only pages that have changed since the previous checkpoint.

This dramatically reduces storage requirements and checkpoint time for workloads with low memory modification rates. The checkpoint system tracks which pages have changed using a dirty page bitmap.

### 3.5 Compression Formats

| Format | Compression Ratio | Compression Speed | Decompression Speed |
|--------|------------------|-------------------|---------------------|
| gzip | 3-5x | Medium | Medium |
| zstd | 3-6x | Fast | Very Fast |
| lz4 | 2-3x | Very Fast | Very Fast |

## 4. Runtime Storage

Runtime storage maintains sandbox metadata and cluster state. It must support fast reads and writes.

### 4.1 Local State

Each node maintains local state using BadgerDB, an embedded key-value database. This includes sandbox metadata, resource usage, and local configuration.

Local state is specific to each node. It is not replicated to other nodes. When a node fails, its local state is lost.

### 4.2 Cluster State

Cluster-wide state is maintained in etcd. This includes the node registry, sandbox registry, scheduling queue, and configuration.

etcd provides strong consistency through the Raft consensus algorithm. Data is replicated across multiple nodes for fault tolerance.

### 4.3 State Schema

Sandbox entries include ID, name, state, node assignment, configuration, resource usage, and timestamps.

Node entries include ID, address, capacity, current load, health status, and last seen timestamp.

## 5. Storage Backend Options

Multiple storage backends are supported for different deployment scenarios.

### 5.1 Local Storage

Local storage uses the local filesystem. It provides the best performance for single-node deployments. This is the default configuration.

Local storage is appropriate when all data is stored on a single machine. It is simple to configure and requires no additional infrastructure.

### 5.2 Network Storage

Network storage uses NFS or similar network filesystems. It allows multiple nodes to access the same data.

Network storage is appropriate for multi-node deployments where checkpoints need to be accessible from any node. However, network latency affects performance.

### 5.3 Object Storage

Object storage uses S3-compatible APIs. Checkpoints can be stored in cloud object storage for durability.

Object storage is appropriate for long-term checkpoint archival. It provides durability and offsite backup capabilities.

## 6. Storage Management

Storage management handles allocation, cleanup, and monitoring.

### 6.1 Quotas

Storage quotas limit how much data each sandbox or user can store. Quotas are enforced at the storage layer.

When a quota is exceeded, new data cannot be written. The sandbox receives an error and must delete existing data to continue.

### 6.2 Cleanup

When a sandbox is deleted, its checkpoint data is automatically cleaned up. The cleanup process removes all associated files and frees the storage.

Checkpoints can also be automatically deleted based on retention policies. For example, checkpoints older than 30 days can be automatically removed.

### 6.3 Monitoring

Storage metrics are collected and exported for monitoring. Key metrics include disk usage, checkpoint sizes, and storage throughput.

Alerts trigger when disk usage exceeds thresholds. This prevents running out of storage space.

## 7. Storage Performance

Storage performance is critical for checkpoint operations.

### 7.1 Checkpoint Creation

Checkpoint creation must complete quickly to minimize sandbox downtime. The target is under five seconds for a 512MB sandbox.

Compression uses multiple CPU cores in parallel. The disk write is sequential for maximum throughput.

### 7.2 Checkpoint Restoration

Restoration must also complete quickly. The target is under three seconds for a 512MB sandbox.

Decompression uses multiple cores. Memory is allocated in advance to avoid allocation pauses during restoration.

### 7.3 Template Loading

Template loading happens during sandbox creation. The target is under one second.

Templates are loaded into memory in parallel with VM creation. The total time includes both loading and VM startup.

## 8. Storage Optimization

Several optimizations improve storage performance.

### 8.1 SSD Storage

Solid-state drives are strongly recommended for checkpoint storage. The random access patterns of checkpoint operations benefit from SSD performance.

NVMe drives provide the best performance for high-throughput workloads.

### 8.2 Memory Deduplication

KSM (Kernel Samepage Merging) automatically deduplicates identical memory pages across VMs. This reduces effective memory usage by 10-20 percent.

KSM is enabled by default in most Linux distributions. It requires the kernel to be compiled with KSM support.

### 8.3 btrfs Reflinks

btrfs reflinks create instant copies without duplicating data. When a VM is forked, the filesystem creates a reflink rather than copying data.

This makes fork operations extremely fast, as no data is copied initially. Memory pages are shared between parent and child until either writes to them.

## 9. Disaster Recovery

Storage architecture supports disaster recovery scenarios.

### 9.1 Checkpoint Restoration

If a node fails, checkpoints can be restored on a different node. The checkpoint data must be accessible from the new node.

For local storage, a shared filesystem or object storage is required for cross-node restoration.

### 9.2 Backup

Checkpoints can be copied to remote storage for backup. The backup destination should be geographically separated from the primary location.

Backup copies are encrypted before transmission. The encryption key is stored separately from the backup data.

### 9.3 Verification

Checkpoints include checksums for integrity verification. Before restoration, the checksum is verified. If verification fails, restoration is aborted.
