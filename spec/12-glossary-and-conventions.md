# Coco Sandbox – Glossary, normative language, and specification completeness

**Scope:** Terms used across the specification, rules for reading normative text, and what it means for this set to be **complete** for implementation planning without reading application source.  
**Status:** Authoritative.  
**Index:** [Specification index](index.md)

## 1. Normative language (BCP 14)

The key words **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT**, and **MAY** are to be interpreted as described in BCP 14 (RFC 2119 and RFC 8174) when they appear in **UPPERCASE** in a normative section of a specification document.

| Keyword | Meaning |
|--------|--------|
| **MUST** / **MUST NOT** | Absolute requirement or prohibition for a conforming implementation. |
| **SHOULD** / **SHOULD NOT** | Strong recommendation; deviation needs a clear rationale (security, environment, or staged rollout). |
| **MAY** | Truly optional. |

**Informative** sections (narrative, background, or “as implemented in a reference tree”) are not requirements unless they point to a table or paragraph that restates a requirement. When in doubt, follow the [**single-source**](#4-single-source-of-truth) rules in section 4.

Coco’s technical documents **do not** use uppercase keywords in every sentence; where plain language is used, **must-level** behavior is still expressed in **Resource**, **MUST** rows in **policy tables**, and **State** enums. Section 1 of `index.md` and this file define the reading contract for the set.

**Timelessness:** The `spec/` set describes the **current** system. It is **not** a backward-compatibility log. For that stance, read `14-deployment-topology-and-neutrality.md` section 1.

## 2. Glossary

| Term | Definition |
|------|------------|
| **Agent** | The init process inside a guest, responsible for executing commands and talking to the host. |
| **API key** | Credential presented by clients to the Gateway, when authentication is enabled. |
| **Checkpoint** | A persisted snapshot of a sandbox’s state for restore or migration. |
| **Cgroup** | Linux control group; used to bound CPU, memory, and I/O. |
| **CIDR** | Classless inter-domain routing prefix for IP allocation (e.g. private ranges). |
| **Cilium ebpf** | Go library for loading and interacting with eBPF programs; used by coco-net. |
| **Clang** | LLVM C compiler; compiles eBPF programs from C to BPF bytecode. |
| **Connect** | RPC stack used alongside gRPC-style definitions in the Go control plane, where used. |
| **Control plane** | Gateway, Master, Node, and related coordination (typically Go). |
| **COW (copy-on-write)** | Sharing of pages until write, used for fast fork and templates on supporting filesystems. |
| **Data plane** | Visor, Agent, and kernel/eBPF data paths that execute and isolate work. |
| **Default-deny (network)** | Packets are dropped unless an explicit allow rule exists. |
| **eBPF** | Extended Berkeley Packet Filter: in-kernel programs (often XDP or tc) for early packet handling and policy. Compiled from C with Clang, loaded with Cilium ebpf Go library. |
| **E2B-style** | Informal name for an HTTP/JSON style popularized by certain cloud sandboxes; **Coco’s** requirements are in `02-api.md`. |
| **etcd** | Distributed key-value store used for cluster state and master election, when in cluster mode. |
| **Fork (sandbox)** | A child sandbox derived from a parent with COW or equivalent semantics. |
| **Gateway** | The external HTTP service for clients. |
| **Grafana / Prometheus** | Ecosystem; metrics are **Prometheus text** at `/metrics` where enabled. |
| **gRPC** | Internal binary RPC used between control-plane components, where specified. |
| **Hibernation** | Save-to-disk and stop, distinct from a live pause. |
| **IPAM** | IP address management within a node or cluster pool. |
| **KVM** | Kernel virtual machine; hardware-accelerated virtualization. |
| **Label** | Key/value on resources for selection and policy. |
| **Master** | Component that coordinates cluster state and placement. |
| **MicroVM** | Small VM with minimal devices and a dedicated kernel, per sandbox. |
| **Net (Coco Net)** | The network agent with eBPF programs enforcing policy. |
| **Node** | A host that runs a Visor and manages local sandboxes. |
| **OpenTelemetry** | Tracing and context propagation, where implemented. |
| **Pool (VM pool)** | Pre-created VMs to reduce create latency. |
| **Resource** | API object: sandbox, template, checkpoint, node, etc. |
| **SLO** | Service level objective: target under stated assumptions (`06-performance.md`), not a universal guarantee. |
| **seccomp** | System-call filter; restrictive profile in the guest. |
| **TAP / TUN** | Virtual layer-2 / layer-3 interfaces for guest networking. |
| **Template** | Immutable base from which a sandbox is created. |
| **Token bucket** | Shaping/ limiting algorithm for request or packet rate. |
| **VSock** | Virtio socket channel between host and guest; primary host–guest command path. |
| **VSock CID** | Context identifier for a VSock endpoint. |
| **XDP** | eXpress Data Path: early network hook for eBPF. |

## 3. End-to-end relationships (reference)

| From | To | Mechanism (normative class) |
|------|----|------------------|
| External client | Gateway | HTTP REST, JSON, optional TLS at edge |
| Gateway | Master | gRPC / Connect (internal) |
| Master | Node | gRPC (internal) |
| Node | Visor | Unix domain socket, binary protocol |
| Visor | Agent | VSock; TCP to localhost in development (see `03-network.md`) |
| Node and/or Net | Kernel | eBPF maps, policy, XDP (see `03-network.md`) |
| Master | etcd | When clustering is in use (see `07-cluster.md`) |

## 4. Single source of truth

| Topic | Authoritative file |
|-------|-------------------|
| High-level architecture and component list (summary) | `00-overview.md` |
| Public API fields and errors | `02-api.md` |
| Network policy model | `03-network.md` |
| Seccomp and security layers | `04-security.md` |
| Storage layout and checkpoint pieces | `05-storage.md` |
| Numerical SLOs and bench intent | `06-performance.md` |
| Master/node scheduling | `07-cluster.md` |
| Log fields, metrics, health | `08-observability.md` |
| Toolchain | `09-dependencies.md` and repository pins |
| Operator-tunable vs product architecture | `10-self-hosting-and-operations.md` |
| Reference bind and environment names (open-source line) | `11-repository-conformance.md` |
| Glossary, BCP 14, completeness criteria | `12-glossary-and-conventions.md` |
| Who uses Coco and example agent-style consumers (informative only) | `13-use-cases-and-consumers.md` |
| Neutrality, “always current” spec stance, community-node pattern (informative) | `14-deployment-topology-and-neutrality.md` |

`00-overview.md` summarizes; if it conflicts with a row in this table, the **numbered file above wins** until the overview is corrected. `13` and `14` do not add data-plane **MUST**s; they orient readers.

## 5. What “complete specification” means for Coco (document-only)

The specification set is **complete** for a reader who implements or operates Coco **if and only if**:

1. The reader can name every **major component** and their **protocol edges** from `00-overview.md` and **section 3** of this file, without ad hoc tribal knowledge.  
2. A client author can build against **`02-api.md`**, including error shape and the documented path prefix (for example under `/v1`), without reading non-spec docs.  
3. A network operator can derive **default-deny** and **allow rule structure** from **`03-network.md`** and deployment tables in **`10`** / reference **`11`**.  
4. A security review can use **`04-security.md`**, not scattered remarks in other files, for isolation and syscall posture.  
5. Performance and capacity planning use **`06-performance.md`** SLOs as the **only** numeric targets, not ad hoc numbers in chat or `00`.  
6. Cluster behavior is described by **`07-cluster.md`**; single-node is still consistent with the same spec (subset of flows).  
7. Observability contracts (health, metrics format class) are in **`08-observability.md`**.  
8. Glossary and normative terms are fixed in **this file**; new public terms are added here when introduced in other spec files.  

Satisfaction of (1)–(8) is the bar for a **100** self-assessment **of the spec alone**, independent of whether a particular checkout compiles. Implementation quality is a separate review.

## 6. How spec updates propagate

When you add a **MUST** behavior or a new **resource field**, the change **MUST** land in the authoritative file from the table in section 4, **MUST** be reflected in `00-overview.md` if the executive summary is affected, and **SHOULD** add or link a **glossary entry** if a new **defined term** appears. **Product version numbers** and **compatibility history** do **not** live in `00`–`08` prose; toolchain pins use **`09-dependencies.md`** and repository files. Stance on “always current” spec: `14-deployment-topology-and-neutrality.md` section 1.

---

Related: [index.md](index.md) (index and reading order), [10-self-hosting-and-operations.md](10-self-hosting-and-operations.md) (deployment boundary).
