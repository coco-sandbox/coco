# Coco Sandbox - Dependencies Specification

This document defines the build and runtime dependencies for Coco, including language versions, system requirements, and external services.

## 1. Build Dependencies

These tools are required to build Coco from source.

### 1.1 Go

Go version 1.21 or later is required to build Go components. Earlier versions may work but are not tested.

Go is used for the control plane components: Gateway, Master, Node, Net, and CLI.

Installation is through the official Go distribution. The Go module system manages dependencies.

### 1.2 Zig

Zig version 0.16 or later is required to build Zig components. Earlier versions have different syntax and standard library organization.

Zig is used for data plane components: Visor, Agent, and Fork.

Zig is installed from the official distribution. It is a static binary with no external dependencies.

### 1.3 Clang

Clang is required to compile eBPF programs. The eBPF programs are written in C and compiled with clang.

The version should be recent enough to support the target kernel's BPF features. clang-14 or later is recommended.

### 1.4 Protocol Buffer Compiler

The protocol buffer compiler (protoc) generates Go code from .proto files. Version 3 or later is required.

Plugins generate Go and Zig code. The plugins are specified in the Makefile.

### 1.5 Other Tools

Docker or Podman builds container images. Container builds are optional but recommended for deployment.

Git manages source code and versioning.

Make orchestrates the build process.

## 2. System Dependencies

These system components are required to run Coco.

### 2.1 Linux Kernel

Linux kernel version 5.10 or later is required. Earlier versions lack some features.

The KVM module must be loaded. This provides hardware virtualization support.

The kernel must be compiled with options required for KVM and eBPF. Most distribution kernels include these options.

### 2.2 Filesystem

btrfs is recommended for reflink support. Reflinks enable fast fork operations without data copying.

ext4 and xfs are supported but lack reflink support. Fork operations are slower on these filesystems.

The filesystem must support extended attributes for some eBPF features.

### 2.3 Kernel Modules

The KVM module must be loaded. For Intel processors, load kvm and kvm_intel. For AMD processors, load kvm and kvm_amd.

The required modules are typically loaded automatically when the CPU supports virtualization.

### 2.4 Memory and Disk

A minimum of 4GB RAM is required per node. 8GB or more is recommended for production.

A minimum of 50GB disk space is required for templates and checkpoints. More is needed for larger deployments.

SSD storage is strongly recommended for checkpoint operations. NVMe drives provide the best performance.

### 2.5 CPU

CPUs must support hardware virtualization. Intel processors need VT-x. AMD processors need AMD-V.

For best performance, CPUs should support additional virtualization extensions. These include VT-d for device assignment and AES-NI for encryption acceleration.

## 3. Runtime Dependencies

These services are required to run Coco.

### 3.1 etcd

etcd provides cluster state storage and leader election. Version 3.5 or later is recommended.

etcd can be embedded or external. For production, an external etcd cluster is recommended for reliability.

Three etcd instances are recommended for high availability. They should be spread across availability zones.

### 3.2 Prometheus

Prometheus collects and stores metrics. It is required for the metrics pipeline.

Prometheus can be embedded or external. For production, an external Prometheus is recommended.

Prometheus scrapes metrics from Coco components at regular intervals.

### 3.3 Optional Services

Grafana provides metrics visualization. It is optional but recommended for operational dashboards.

Jaeger provides distributed tracing. It is optional but recommended for debugging distributed issues.

## 4. Language-Specific Dependencies

### 4.1 Go Dependencies

Go dependencies are managed through go modules. The go.mod file specifies required versions.

Key dependencies include grpc and connect for API serving, etcd client for cluster communication, and Prometheus client for metrics.

Dependencies are downloaded during build. No separate installation step is required.

### 4.2 Zig Dependencies

Zig has no runtime dependencies. All required functionality is in the standard library.

Build dependencies include linux headers for system calls. Zig components use direct KVM syscalls through the linux/uapi.h headers, not libkvm or cgo.

### 4.3 eBPF Dependencies

libbpf provides the userspace API for loading eBPF programs. It is typically included with the kernel headers.

clang compiles eBPF programs. The Makefile invokes clang with the correct flags.

## 5. Version Compatibility

### 5.1 Component Versions

| Component | Minimum Version | Recommended Version |
|----------|-----------------|---------------------|
| Go | 1.21 | 1.22 |
| Zig | 0.12 | 0.13 |
| Linux Kernel | 5.10 | 6.1+ |
| etcd | 3.5 | 3.6 |
| clang | 14 | 17 |

### 5.2 Feature Compatibility

Some features require specific kernel versions. TDX requires kernel 5.19 or later with TDX support. SGX requires kernel 5.17 or later with SGX support.

XDP features may require specific kernel versions. Native XDP requires kernel 5.10 or later.

btrfs reflinks require btrfs-progs. The feature is available in most modern distributions.

## 6. Container Dependencies

When running in containers, additional dependencies apply.

### 6.1 Container Runtime

Docker 20.10 or later is recommended. Podman 4.0 or later is also supported.

The runtime must support privileged containers. KVM access requires privileged mode.

### 6.2 Base Images

Minimal base images are recommended to reduce attack surface. Alpine or distroless images work well.

The base image must include the required runtime: Go for Go binaries, Zig for Zig binaries.

### 6.3 Privileges

Containers running Node must run in privileged mode. This allows KVM device access.

The NET_ADMIN capability is required for network configuration.

The SYS_ADMIN capability is required for some container operations.

## 7. Development Dependencies

These additional tools are useful for development.

### 7.1 Testing

Go testing is built into the standard library. Additional testing frameworks are not required.

Zig has built-in testing with zig test.

### 7.2 Code Quality

gofmt formats Go code automatically.

golangci-lint provides comprehensive linting.

zig fmt formats Zig code.

### 7.3 Documentation

Hugo generates documentation sites from Markdown.

Docusaurus is an alternative documentation generator.

## 8. External Services

These external services are optionally used.

### 8.1 Cloud Providers

Cloud providers can provide managed Kubernetes (EKS, GKE, AKS).

Object storage (S3, GCS, Azure Blob) can store checkpoints.

Load balancers can distribute traffic across gateway instances.

### 8.2 Monitoring

Prometheus is required for metrics collection.

Grafana is recommended for visualization.

Jaeger is optional for distributed tracing.

### 8.3 Authentication

OAuth2 providers can authenticate users.

LDAP or Active Directory can integrate with existing identity systems.
