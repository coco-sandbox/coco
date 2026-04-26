# Coco Sandbox - Security Specification

This document defines the security architecture for Coco, including isolation layers, capability management, and hardware security features.

## 1. Security Architecture Overview

Coco implements defense in depth through multiple security layers. No single layer is sufficient; together they provide comprehensive protection. The layers work from outside in, with each layer providing a barrier that must be crossed before reaching inner layers.

The outermost layer handles network traffic, filtering packets at the kernel level. Inside that, KVM provides hardware virtualization, isolating each sandbox in its own virtual machine. Optional hardware enclaves add encryption and attestation. Finally, syscall filtering restricts what the sandbox can do even if it escapes virtualization.

## 2. Isolation Layers

Four distinct isolation layers protect the system from compromised or malicious sandboxes.

### 2.1 Network Isolation

Network isolation is provided by the eBPF-based filter running at XDP level. All packets pass through this filter before entering or leaving a sandbox. The filter implements default-deny policies, dropping any packet not explicitly allowed.

This layer prevents compromised sandboxes from attacking other sandboxes or external systems. Even if an attacker gains code execution inside a sandbox, they cannot send network packets without explicit permission.

### 2.2 VM Isolation

VM isolation uses KVM hardware virtualization. Each sandbox runs in its own MicroVM with a dedicated kernel, separate memory space, and isolated devices. There is no resource sharing between sandboxes.

This layer prevents attacks that rely on shared kernel state. It also provides protection against side-channel attacks that plague container-based isolation.

### 2.3 Hardware Enclaves

Optional hardware enclaves provide additional protection for sensitive workloads. Three technologies are supported: Intel TDX, Intel SGX, and AMD SEV.

These technologies encrypt memory contents, preventing even the host operator from reading sandbox data. They also provide hardware-based attestation, allowing remote parties to verify the sandbox's configuration.

### 2.4 Syscall Filtering

Syscall filtering uses seccomp to restrict available system calls. Only a minimal set of syscalls is allowed. All others result in the sandbox being terminated.

This layer limits the blast radius of any successful exploit. Even if an attacker gains code execution, they cannot perform dangerous operations like loading kernel modules or accessing raw devices.

## 3. VM Isolation Details

Each sandbox runs in its own KVM MicroVM. This provides strong isolation from the host and other sandboxes.

### 3.1 Memory Isolation

Each VM has dedicated memory that is not shared with the host or other VMs. Memory is allocated at VM creation and released at VM destruction. The hypervisor ensures one VM cannot read or write another VM's memory.

### 3.2 CPU Isolation

VMs are scheduled on isolated vCPUs. The scheduler ensures fair CPU sharing while preventing one VM from starving others. CPU affinity can be configured for real-time workloads.

### 3.3 Device Isolation

Each VM has virtual devices that are isolated from the host and other VMs. The only path to the host is through the virtual console and VSock, both of which are controlled by the hypervisor.

## 4. Hardware Enclaves

Three hardware enclave technologies are supported for workloads requiring additional protection.

### 4.1 Intel TDX

Trust Domain Extensions (TDX) provides a hardware-encrypted execution environment. The VM memory is encrypted with a key stored in the CPU. Even the hypervisor cannot read the memory contents.

TDX supports remote attestation, allowing remote parties to verify that a sandbox is running in a genuine TDX environment with the expected configuration. This enables sensitive workloads like key management or cryptographic operations.

### 4.2 Intel SGX

Software Guard Extensions (SGX) provides even stronger isolation than TDX, with protected memory regions called enclaves. SGX enclaves can encrypt their own data and provide hardware-level attestation.

SGX is suitable for workloads requiring the highest levels of security, such as processing encrypted data or performing cryptographic operations.

### 4.3 AMD SEV

Secure Encrypted Virtualization (SEV) encrypts VM memory with a key that is only available to the CPU. The hypervisor cannot access VM memory contents.

SEV is the AMD equivalent of Intel TDX, providing similar protection for AMD-based systems.

## 5. Capability Management

The guest runs with minimal Linux capabilities. Dangerous capabilities are completely removed.

### 5.1 Dropped Capabilities

These capabilities are never available inside a sandbox.

CAP_SYS_ADMIN enables mount operations, namespace creation, and many other privileged operations. It is never granted.

CAP_NET_ADMIN allows network configuration changes. Attackers could use this to bypass network filters.

CAP_SYS_MODULE allows loading kernel modules. This would completely compromise kernel security.

CAP_SYS_RAWIO allows direct hardware access. This could bypass many security controls.

CAP_SYS_PTRACE allows tracing other processes. Attackers could use this to steal sensitive data.

CAP_SYS_TIME allows changing system time. This could cause issues with authentication and logging.

CAP_SYS_BOOT allows rebooting the system. Attackers could use this to cause denial of service.

CAP_AUDIT_CONTROL allows modifying audit configuration. Attackers could disable security logging.

### 5.2 Seccomp Policy

A strict seccomp policy further restricts available operations. The policy allows a minimal set of syscalls needed for basic operation.

Allowed syscalls include read, write, exit, brk, mmap, mprotect, clock_gettime, and nanosleep. These are sufficient for most applications to run.

All other syscalls result in immediate termination of the process. This severely limits what an attacker can do even if they find a code execution vulnerability.

## 6. Resource Limits

Resource limits prevent individual sandboxes from consuming excessive resources and affecting other sandboxes.

### 6.1 Memory Limits

Each sandbox has a hard memory limit. When the limit is reached, the kernel invokes the out-of-memory killer within the sandbox. The sandbox can configure what happens when OOM occurs.

Memory limits are enforced through cgroups, which the host kernel enforces regardless of what happens inside the sandbox.

### 6.2 CPU Limits

CPU limits use the Completely Fair Scheduler (CFS) bandwidth controller. Each sandbox gets a quota of CPU time per period. The quota is enforced by the host scheduler.

Real-time CPU scheduling is not allowed inside sandboxes. This prevents sandbox workloads from affecting host latency.

### 6.3 I/O Limits

I/O limits use the blkio cgroup controller. Limits can be set for read bytes per second, write bytes per second, read operations per second, and write operations per second.

These limits ensure that one sandbox cannot monopolize disk bandwidth and degrade performance for others.

## 7. Authentication and Authorization

Access to the API requires authentication.

### 7.1 API Keys

API keys are the primary authentication mechanism. Keys are created through the administrative interface. Each key has associated roles and scopes that determine what operations are allowed.

Keys can be rotated without downtime by creating a new key and removing the old one.

### 7.2 Roles

Roles define collections of permissions.

Admin role has full access to all operations.

Operator role can manage sandboxes but cannot manage users or system configuration.

Developer role can create, modify, and delete their own sandboxes.

Readonly role can only view sandboxes and cannot make changes.

### 7.3 Scopes

Scopes provide fine-grained control over API access.

sandbox:create allows creating new sandboxes.

sandbox:read allows viewing sandbox details.

sandbox:write allows modifying sandboxes.

sandbox:delete allows deleting sandboxes.

template:read allows viewing templates.

template:write allows managing templates.

cluster:admin allows cluster management operations.

## 8. Network Security

Network security is implemented through default-deny policies and kernel-level filtering.

### 8.1 Default Deny

All egress traffic is denied by default. No sandbox can send network packets to any destination without an explicit policy allowing it.

This is the most important security measure. Even if an attacker compromises a sandbox, they cannot reach other systems without explicit permission.

### 8.2 Packet Filtering

The eBPF packet filter runs at the XDP level, before the main network stack. This provides protection even against high-volume attacks.

Each packet is checked against the policy allowlist. Packets not matching any rule are dropped immediately. This happens before any kernel processing, minimizing overhead.

### 8.3 Rate Limiting

Rate limiting prevents individual sandboxes from consuming excessive bandwidth. The token bucket algorithm allows bursts while maintaining long-term limits.

If a sandbox exceeds its rate limit, packets are dropped. The sandbox doesn't receive any indication that packets were dropped; they simply never arrive at their destination.

## 9. Audit Logging

Security-relevant events are logged for forensic analysis and compliance.

### 9.1 Logged Events

Sandbox creation and deletion are logged with the requesting user and timestamp.

Authentication events including successful logins and failed attempts are logged.

Authorization failures are logged when a user attempts an operation they don't have permission for.

Configuration changes to security policies are logged.

### 9.2 Log Format

Each log entry includes the timestamp, event type, user identity, source IP, outcome, and relevant details. Logs are formatted as JSON for easy parsing by log aggregation systems.

### 9.3 Retention

Audit logs are retained according to organizational policy. Default retention is 90 days. Logs are stored in a separate partition from operational data to prevent tampering.

## 10. Security Comparison

Coco provides stronger security than alternatives through multiple defense layers.

| Feature | Container | VM | Coco |
|---------|-----------|-----|------|
| Shared Kernel | Yes | No | No |
| Network Isolation | Namespace | VF | eBPF |
| Default Deny | No | No | Yes |
| Hardware Enclave | No | Optional | Optional |
| Syscall Filter | Optional | No | Yes |
| Memory Encryption | No | Optional | Optional |
