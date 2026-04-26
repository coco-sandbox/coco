# Coco Sandbox – API specification

**Scope:** Public HTTP API shape, resources, and error model for Gateway clients.  
**Status:** Authoritative.  
**Index:** [Specification index](index.md)

---

## 1. API overview

### 1.1 Protocol

| Type | Usage | Description |
|------|-------|-------------|
| REST | Public-facing | HTTP/1.1 with JSON (agents/SDKs) |
| gRPC | Internal | node–master, node–visor and related internal RPC |
| Streaming | Both | Server-side streaming via HTTP chunked or gRPC |

### 1.2 Base URL

The public API is rooted at a path such as **/v1** under a host. Scheme, host, and port are **deployment-defined** in general. For the **coco-gateway** binary, the in-tree default bind is **:4747** (all interfaces) with `COCO_LISTEN_ADDR` override; the same HTTP server also serves health and metrics paths (see `08-observability.md` and `11-repository-conformance.md`). A typical local base URL is **http** plus host, plus **/v1**; operators choose TLS and external ports per `10-self-hosting-and-operations.md`.

### 1.3 Communication pattern

| Communication | Protocol | Purpose |
|--------------|----------|---------|
| Client → Gateway | REST | Public API (agents/SDKs) |
| Gateway → Master | gRPC | Internal scheduling |
| Master → Node | gRPC | Task distribution |
| Node → Visor | Unix domain socket | VM operations |
| Visor ↔ Agent | VSock (TCP fallback in dev) | Guest communication |

## 2. Core services (RPC names)

### 2.1 Sandbox service

| Method | Description | Streaming response |
|--------|-------------|-------------------|
| CreateSandbox | Create new sandbox | No |
| GetSandbox | Get sandbox by ID | No |
| ListSandboxes | List sandboxes (filters as implemented) | No |
| DeleteSandbox | Delete sandbox | No |
| PauseSandbox | Pause running sandbox | No |
| ResumeSandbox | Resume paused sandbox | No |
| HibernateSandbox | Hibernate sandbox to disk | No |
| RestoreSandbox | Restore from checkpoint | No |
| ForkSandbox | Fork existing sandbox | No |
| CreateCheckpoint | Create checkpoint | No |

### 2.2 Execution service

| Method | Description | Streaming response |
|--------|-------------|-------------------|
| Exec | Run command; complete result when the process exits | No |
| StreamingExec | Run command; stream output as produced | Yes |
| InteractiveExec | Interactive shell or session | Yes, bidirectional |

### 2.3 Template service

| Method | Description |
|--------|-------------|
| CreateTemplate | Create template |
| GetTemplate | Get template by ID |
| ListTemplates | List templates |
| DeleteTemplate | Delete template |
| BuildTemplate | Build from image |

### 2.4 Cluster service

| Method | Description |
|--------|-------------|
| GetClusterInfo | Cluster info |
| ListNodes | List nodes |
| GetNode | Get node by ID |
| DrainNode | Drain node (maintenance) |

## 3. Resources and fields

### 3.1 Sandbox resource

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique identifier |
| name | string | Display name |
| state | enum | See Sandbox state |
| template_id | string | Template used for creation |
| memory_mb | int32 | Memory (MiB) |
| vcpus | int32 | Virtual CPUs |
| vsock_cid | uint32 | VSock context ID |
| pid | int64 | Host-side process identifier (when applicable) |
| parent_id | string | Parent sandbox if forked (empty if none) |
| fork_depth | int32 | Fork depth from root |
| labels | string map | Arbitrary labels |
| created_at | timestamp | Creation time (RFC 3339) |

### 3.2 Sandbox state

| State | Description |
|-------|-------------|
| CREATING | Provisioning |
| RUNNING | Running |
| PAUSED | Paused |
| STOPPING | Tearing down |
| STOPPED | Stopped |
| HIBERNATED | Persisted to disk (hibernate) |
| ERROR | Error state |

### 3.3 Template resource

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique identifier |
| name | string | Display name |
| description | string | Description |
| rootfs | string | Root filesystem path or reference |
| kernel | string | Kernel path |
| initrd | string | initrd path |
| default_memory_mb | int32 | Default memory (MiB) |
| default_vcpus | int32 | Default vCPUs |
| size_bytes | int64 | On-disk or logical size |
| state | enum | Build state of template |

### 3.4 Checkpoint resource

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique identifier |
| sandbox_id | string | Source sandbox |
| name | string | Human-readable name |
| path | string | Storage path (opaque to client in many deployments) |
| size_bytes | int64 | Size of checkpoint data |
| is_compressed | bool | Whether payload is compressed |
| created_at | timestamp | Creation time |

## 4. Request and response bodies (normative fields, not example payloads)

Use these tables when implementing or validating clients. Bodies are JSON unless otherwise specified.

### 4.1 Create sandbox

- **Method / path (HTTP):** POST, collection path for sandboxes (see §7.1).
- **Request body fields**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | as policy | Display name |
| template_id | string | yes | Template to instantiate |
| memory_mb | int32 | no | Override template default |
| vcpus | int32 | no | Override template default |
| labels | object | no | String keys and string values |

- **Response body:** `Sandbox` resource (§3.1) including assigned `id`, `state`, and `vsock_cid` when available.

### 4.2 Fork sandbox

- **Path parameter:** `id` of source sandbox. **Request body field:** `name` (string, display name of child). **Response:** object containing nested `Sandbox` (child) and optional `duration_ms` (int64) if the server reports timing.

### 4.3 Execute (streaming) command

- **Path parameter:** `id` of sandbox. **Request body fields:** `command` (string, argv0), `args` (string array, optional), `timeout_ms` (int64, optional), plus any options defined in implementation for env or cwd. **Response:** stream of events: event type (stdout, stderr, exit) and associated payload; exact event names and JSON field names are defined by the implementation and must be documented in generated API docs or OpenAPI, not duplicated here as literal JSON.

## 5. Error handling

### 5.1 Error codes (logical)

| Code | Description |
|------|-------------|
| NOT_FOUND | Resource not found |
| ALREADY_EXISTS | Resource already exists |
| INVALID_ARGUMENT | Invalid request |
| PERMISSION_DENIED | Not authorized |
| RESOURCE_EXHAUSTED | Capacity or quota exceeded |
| INTERNAL | Internal error |
| UNAVAILABLE | Dependency unavailable |
| TIMEOUT | Operation timeout |

### 5.2 Error response envelope

All error responses use a single top-level key **error** whose value is an object with at least **code** (string, one of the logical codes above) and **message** (string, human-readable). An optional **details** field (string or structured object) carries implementation-specific diagnostics.

## 6. Authentication and rate limit signaling

### 6.1 Request headers (common)

| Header | Semantics |
|--------|-----------|
| Authorization | `Bearer` plus API key or token, when auth is enabled |
| X-Coco-Project | Optional project or tenant scoping, when used |

### 6.2 Rate limit response headers

When rate limiting is enabled, responses may include rate-limit headers. Header names and semantics follow common HTTP practice (e.g. limit, remaining, reset). **Numeric values are deployment-defined**; see `10-self-hosting-and-operations.md`.

## 7. HTTP mapping (REST)

| Operation | Method | Path pattern |
|----------|--------|--------------|
| CreateSandbox | POST | `/v1/sandboxes` |
| GetSandbox | GET | `/v1/sandboxes/{id}` |
| ListSandboxes | GET | `/v1/sandboxes` |
| DeleteSandbox | DELETE | `/v1/sandboxes/{id}` |
| ForkSandbox | POST | `/v1/sandboxes/{id}/fork` |
| PauseSandbox | POST | `/v1/sandboxes/{id}/pause` |
| ResumeSandbox | POST | `/v1/sandboxes/{id}/resume` |
| StreamingExec | POST | `/v1/sandboxes/{id}/exec` |

**Health and metrics** are not part of the `/v1` public resource map above; for **coco-gateway** they share the same HTTP listener as the API (see `08-observability.md` and `11-repository-conformance.md`). Other deployments remain **operator-defined** (`10-self-hosting-and-operations.md`).

## 8. E2B-style client patterns (non-normative)

Some client libraries follow patterns similar to well-known hosted sandboxes. **Coco’s normative contract** is this document and `00`–`08`. Optional client-style coverage is a **product and testing** concern; unsupported calls return documented error codes (§5). This spec does not track **external** API histories.

| Informal E2B-style area | Support goal (product) |
|------------------------|-------------|
| Sandbox lifecycle | Target where implemented |
| Code / command execution | Target where implemented |
| Filesystem and process helpers | As implemented in tree |

---

Related: `00-overview.md` (context), `04-security.md` (auth model), `10-self-hosting-and-operations.md` (deployment boundaries), `11-repository-conformance.md` (defaults and `COCO_*` for gateway), `14-deployment-topology-and-neutrality.md` (neutrality and how `spec/` is written).
