# Coco Sandbox - API Specification

**Spec ID:** 04
**Status:** Authoritative

## 1. API Overview

### 1.1 Protocol

| Type | Usage | Description |
|------|-------|-------------|
| REST | Public-facing | HTTP/1.1 with JSON (for agents/SDKs) |
| gRPC | Internal | For node-to-master, node-to-visor communication |
| Streaming | Both | Server-side streaming via HTTP chunked or gRPC |

### 1.2 Base URL

```
Production: https://<your-coco-host>/v1
Development: http://localhost:8080/v1
```

The production URL is user-configurable based on deployment.

### 1.3 Communication Pattern

| Communication | Protocol | Purpose |
|--------------|----------|---------|
| Client → Gateway | REST | Public API (agents/SDKs) |
| Gateway → Master | gRPC | Internal scheduling |
| Master → Node | gRPC | Task distribution |
| Node → Visor | Unix Socket | VM operations |
| Visor ↔ Agent | VSock | Guest communication |

## 2. Core Services

### 2.1 Sandbox Service

| Method | Description | Streaming |
|--------|-------------|-----------|
| CreateSandbox | Create new sandbox | No |
| GetSandbox | Get sandbox by ID | No |
| ListSandboxes | List all sandboxes | No |
| DeleteSandbox | Delete sandbox | No |
| PauseSandbox | Pause running sandbox | No |
| ResumeSandbox | Resume paused sandbox | No |
| HibernateSandbox | Hibernate sandbox to disk | No |
| RestoreSandbox | Restore from checkpoint | No |
| ForkSandbox | Fork existing sandbox | No |
| CreateCheckpoint | Create checkpoint | No |

### 2.2 Execution Service

| Method | Description | Streaming |
|--------|-------------|-----------|
| Exec | Execute command (sync) | No |
| StreamingExec | Execute with output stream | Yes |
| InteractiveExec | Interactive shell | Yes (bidirectional) |

### 2.3 Template Service

| Method | Description |
|--------|-------------|
| CreateTemplate | Create template |
| GetTemplate | Get template by ID |
| ListTemplates | List all templates |
| DeleteTemplate | Delete template |
| BuildTemplate | Build from image |

### 2.4 Cluster Service

| Method | Description |
|--------|-------------|
| GetClusterInfo | Get cluster info |
| ListNodes | List cluster nodes |
| GetNode | Get node by ID |
| DrainNode | Drain node for maintenance |

## 3. Resources

### 3.1 Sandbox

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique identifier |
| name | string | Display name |
| state | enum | Current state |
| template_id | string | Template used |
| memory_mb | int32 | Memory in MB |
| vcpus | int32 | Virtual CPUs |
| vsock_cid | uint32 | VSock context ID |
| pid | int64 | Host PID |
| parent_id | string | Parent sandbox (if forked) |
| fork_depth | int32 | Fork depth |
| labels | map | User labels |
| created_at | timestamp | Creation time |

### 3.2 Sandbox State

| State | Description |
|-------|-------------|
| CREATING | Sandbox is being created |
| RUNNING | Sandbox is running |
| PAUSED | Sandbox is paused |
| STOPPING | Sandbox is stopping |
| STOPPED | Sandbox has stopped |
| HIBERNATED | Sandbox is hibernated |
| ERROR | Sandbox has error |

### 3.3 Template

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique identifier |
| name | string | Display name |
| description | string | Description |
| rootfs | string | Root filesystem path |
| kernel | string | Kernel path |
| initrd | string | Initrd path |
| default_memory_mb | int32 | Default memory |
| default_vcpus | int32 | Default vCPUs |
| size_bytes | int64 | Template size |
| state | enum | Build state |

### 3.4 Checkpoint

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique identifier |
| sandbox_id | string | Parent sandbox |
| name | string | Checkpoint name |
| path | string | Storage path |
| size_bytes | int64 | Checkpoint size |
| is_compressed | bool | Compression enabled |
| created_at | timestamp | Creation time |

## 4. Request/Response Examples

### 4.1 Create Sandbox

**Request:**
```json
POST /v1/sandboxes
{
  "name": "my-sandbox",
  "template_id": "ubuntu-base",
  "memory_mb": 512,
  "vcpus": 1,
  "labels": {"env": "dev"}
}
```

**Response:**
```json
{
  "id": "sb_abc123",
  "name": "my-sandbox",
  "state": "RUNNING",
  "template_id": "ubuntu-base",
  "memory_mb": 512,
  "vcpus": 1,
  "vsock_cid": 3,
  "created_at": "2024-01-01T00:00:00Z"
}
```

### 4.2 Fork Sandbox

**Request:**
```json
POST /v1/sandboxes/sb_abc123/fork
{
  "name": "forked-sandbox"
}
```

**Response:**
```json
{
  "sandbox": {
    "id": "sb_def456",
    "name": "forked-sandbox",
    "parent_id": "sb_abc123",
    "fork_depth": 1
  },
  "duration_ms": 8
}
```

### 4.3 Execute Command

**Request:**
```json
POST /v1/sandboxes/sb_abc123/exec
{
  "command": "python",
  "args": ["-c", "print(1+1)"],
  "timeout_ms": 30000
}
```

**Response (streaming):**
```
data: {"type": "STDOUT", "data": "2\n"}
data: {"type": "EXIT", "exit_code": 0}
```

## 5. Error Handling

### 5.1 Error Codes

| Code | Description |
|------|-------------|
| NOT_FOUND | Resource not found |
| ALREADY_EXISTS | Resource already exists |
| INVALID_ARGUMENT | Invalid request |
| PERMISSION_DENIED | Unauthorized |
| RESOURCE_EXHAUSTED | No resources available |
| INTERNAL | Internal error |
| UNAVAILABLE | Service unavailable |
| TIMEOUT | Operation timeout |

### 5.2 Error Response

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Sandbox sb_notfound not found",
    "details": "..."
  }
}
```

## 6. Authentication

### 6.1 Headers

```
Authorization: Bearer <api_key>
X-Coco-Project: <project_id>
```

### 6.2 Rate Limiting Headers

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1609459200
```

## 7. REST Compatibility

### 7.1 HTTP Mapping

| gRPC | HTTP |
|------|------|
| CreateSandbox | POST /v1/sandboxes |
| GetSandbox | GET /v1/sandboxes/{id} |
| ListSandboxes | GET /v1/sandboxes |
| DeleteSandbox | DELETE /v1/sandboxes/{id} |
| ForkSandbox | POST /v1/sandboxes/{id}/fork |
| PauseSandbox | POST /v1/sandboxes/{id}/pause |
| ResumeSandbox | POST /v1/sandboxes/{id}/resume |
| StreamingExec | POST /v1/sandboxes/{id}/exec |

## 8. E2B Compatibility

### 8.1 Drop-in Replacement

Coco provides full E2B compatibility - users can switch by changing the API endpoint:

```python
# E2B Cloud
from e2b_code_interpreter import Sandbox
sandbox = Sandbox.create("api_key", "https://api.e2b.dev")

# Coco (drop-in replacement)
from e2b_code_interpreter import Sandbox
sandbox = Sandbox.create("api_key", "http://localhost:8080")
```

### 8.2 Supported E2B APIs

| E2B API | Coco Support |
|---------|--------------|
| Sandbox.create() | ✅ Full |
| sandbox.run_code() | ✅ Full |
| sandbox.run_command() | ✅ Full |
| sandbox.filesystem.* | ✅ Full |
| sandbox.process.* | ✅ Full |

## 9. Comparison with Cube

| Aspect | CubeSandbox | Coco | Winner |
|--------|-------------|------|--------|
| Public API | REST | REST + gRPC | Coco |
| Streaming | No | Yes | Coco |
| WebSocket | No | Yes | Coco |
| E2B Compatible | Yes | Yes | Tie |
| gRPC API | No | Yes | Coco |
| Bidirectional | No | Yes | Coco |
