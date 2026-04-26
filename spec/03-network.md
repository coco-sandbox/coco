# Coco Sandbox – Network specification

**Scope:** Network modes, eBPF data path, policy model, and performance goals for Coco Net.  
**Status:** Authoritative.  
**Index:** [Specification index](index.md)

## 1. Network architecture overview

Coco implements network isolation at multiple levels. The host runs a network agent that programs eBPF filters in the kernel. Each sandbox connects to the host through either a virtual ethernet pair or directly through VSock. All traffic passes through the eBPF filters for policy enforcement.

The architecture prioritizes security through default-deny policies while maintaining high performance through kernel-level packet processing. The Express Data Path (XDP) allows processing packets before they reach the main network stack, providing near-native throughput.

## 2. Network Modes

Coco supports four network modes, each providing different levels of connectivity and isolation.

### 2.1 Bridge Mode

This is the default mode. Sandboxes connect to a bridge device on the host. The bridge performs NAT to allow outbound connections. Each sandbox receives an IP address from a private subnet.

This mode provides full internet access while maintaining isolation. Performance is good but not optimal due to the bridge and NAT overhead.

### 2.2 Tap Mode

In this mode, each sandbox receives its own TAP device. The TAP device connects directly to the host network stack without bridging. This provides near-native performance but requires more configuration.

Recommended for high-throughput workloads where network performance is critical.

### 2.3 VSock Mode

Sandboxes have VSock connectivity only. No IP networking is configured. Communication with the host happens through VSock, which provides extremely low latency.

Use this mode when network isolation is paramount or when the sandbox only needs to communicate with the host.

### 2.4 Isolated Mode

No network connectivity whatsoever. The sandbox cannot reach the host or any external systems. This provides the highest level of isolation.

Use this mode for untrusted workloads that should have no network access.

## 3. eBPF Programs

Three eBPF programs work together to provide network functionality. These programs run at different points in the packet processing pipeline.

### 3.1 XDP Filter Program

This program runs at the earliest possible point in the network stack, directly at the network driver level. It processes both incoming and outgoing packets.

The program implements a default-deny policy. Every packet is checked against a set of allow rules. Packets not matching any rule are dropped immediately. This happens before the packet even reaches the main network stack, providing protection against volumetric attacks.

The program also implements rate limiting using a token bucket algorithm. Each sandbox has tokens that replenish at a configured rate. When tokens are exhausted, packets are dropped. This prevents any single sandbox from consuming excessive bandwidth.

### 3.2 Flow Table Program

This program maintains connection state in an eBPF hash map. It tracks packets, bytes, and timestamps for each flow.

The flow table enables proper handling of multi-packet connections. When a new packet arrives, the program looks up the flow to determine if the packet belongs to an established connection. This allows the filter to distinguish between new connections and ongoing traffic.

Flow entries expire after a configurable idle timeout. This prevents the flow table from filling with stale entries.

### 3.3 Traffic Shaper Program

This program implements quality of service by queuing packets based on sandbox priority. High-priority sandboxes get preferential treatment during congestion.

The shaper uses a hierarchical token bucket algorithm that supports multiple priority levels. This ensures fair bandwidth distribution while allowing critical workloads to receive guaranteed bandwidth.

## 4. Network Policies

Network policies define what traffic is allowed. All policies are explicit; nothing is allowed by default.

### 4.1 Policy structure

Each policy consists of a **selector** and a set of **rules**. The selector determines which sandboxes the policy applies to (for example by sandbox ID pattern or label match). Each rule is a record with at least: **direction** (ingress or egress), **protocol** (for example tcp, udp, icmp as supported), **ports** (set of port numbers or ranges), and **remote** constraint (**cidr** or equivalent). Multiple rules are evaluated such that if no rule allows a packet, the default in §4.2 applies.

| Part | Semantics |
|------|-----------|
| selector | Binds the policy to one or more sandboxes (ID glob, label query, or explicit list as implemented) |
| rules[] | Set of allow rules; each rule permits matching traffic |
| direction | Whether the rule applies to traffic leaving the sandbox (egress) or entering (ingress) |

A typical pattern for web egress is two rules: one for TCP to destination ports 80 and 443 to a broad CIDR, and one for UDP to port 53 for DNS. Exact encoding is implementation-defined; the spec requires default-deny and explicit allows only.

### 4.2 Default policy

If no policy matches, the default action is DENY. This applies to all traffic that doesn't match any explicit rule.

### 4.3 DNS Policy

DNS traffic on port 53 is commonly needed. A default allow rule typically permits UDP DNS to any destination.

### 4.4 Rate limiting

Rate limits can be applied per sandbox or per policy. The control plane uses a token-bucket (or equivalent) model: **sustained rate** (packets per second and/or bytes per second), and **burst** capacity. All numeric values are **deployment-defined** (see `10-self-hosting-and-operations.md`).

| Field (conceptual) | Meaning |
|--------------------|--------|
| packets_per_second | Sustained packet rate cap |
| bytes_per_second | Sustained byte rate cap |
| burst | Maximum tokens or bytes of burst before drops |

## 5. IP address management

IPAM allocates and manages IP addresses for sandboxes.

### 5.1 Address Pool

The address pool is configured with a CIDR block. Addresses are allocated from this block when sandboxes are created. When sandboxes are destroyed, addresses are returned to the pool.

The CIDR is **operator-configured**. A **non-normative** example is a private /16; capacity depends on the prefix.

### 5.2 Allocation Strategy

When a sandbox requests an IP address, the allocator finds the first available address in the pool. The allocator tries to minimize fragmentation by preferring consecutive addresses.

### 5.3 Reservation

Specific IP addresses can be reserved for particular sandboxes. This is useful for predictable addressing or for whitelisting external access.

## 6. VSock Communication

VSock provides direct communication between host and guest with minimal overhead.

### 6.1 Context IDs

VSock uses Context IDs (CIDs) to identify endpoints. CID 2 is reserved for the host. Guest VMs receive unique CIDs starting from 3. The CID is assigned when the VM is created.

### 6.2 Ports

All VSock communication uses port 4747 by default. This port is used for the command channel between Agent and Visor.

### 6.3 Fallback

When VSock is unavailable, Agent falls back to TCP on 127.0.0.1:4747. This allows development and testing without VSock support.

## 7. Network Performance

Network performance is optimized for high throughput and low latency.

### 7.1 Throughput Targets

The goal is 20 gigabits per second aggregate throughput per node. This is achieved through XDP zero-copy processing and minimal per-packet overhead.

### 7.2 Latency Targets

Host-to-guest round-trip latency should be under 0.5 milliseconds. This is achieved through VSock, which bypasses the TCP/IP stack entirely.

### 7.3 Optimization Techniques

XDP provides zero-copy packet processing by handling packets at the driver level. Batch processing allows processing multiple packets in a single loop, reducing per-packet overhead. Flow caching avoids expensive rule lookups for established connections.

## 8. Network Monitoring

Network metrics are collected and exported for monitoring.

### 8.1 Per-Sandbox Metrics

Each sandbox tracks packets sent, packets received, bytes sent, bytes received, and dropped packets. These are available through the metrics endpoint.

### 8.2 Aggregate Metrics

Aggregate metrics include total packets processed, total bytes transferred, packets dropped by rate limiter, and active flow count.

### 8.3 Alerts

Alerts trigger when packet drop rate exceeds thresholds, when bandwidth limits are hit, or when flow table utilization gets high.

## 9. Security Considerations

### 9.1 Default Deny

All egress traffic is denied by default. Explicit policies must allow any outbound connections. This prevents compromised sandboxes from reaching external systems.

### 9.2 Sandboxed Network Namespaces

Each sandbox can optionally run in its own network namespace. This provides additional isolation beyond the eBPF filters.

### 9.3 DDoS Protection

The XDP filter handles volumetric attacks at the edge, before they can overwhelm the system. Rate limiting ensures that even if a sandbox is compromised, it cannot flood the network.

### 9.4 No Direct Host Access

Sandboxes cannot directly access host services. All access goes through the defined policy rules, which are enforced at the network level.
