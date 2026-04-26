# Coco API Reference

## Sandbox Operations

### Create Sandbox
```
POST /v1/sandboxes
Content-Type: application/json

{
    "template": "python-3.11",
    "memory_mb": 512,
    "vcpus": 2,
    "name": "my-sandbox"
}
```

Response: `201 Created`
```json
{
    "sandbox": {
        "id": "sb_1234567890",
        "state": "running",
        "template": "python-3.11"
    }
}
```

### List Sandboxes
```
GET /v1/sandboxes
```

Response: `200 OK`
```json
{
    "items": [...],
    "total": 5
}
```

### Get Sandbox
```
GET /v1/sandboxes/:id
```

### Delete Sandbox
```
DELETE /v1/sandboxes/:id
```

### Execute Code
```
POST /v1/sandboxes/:id/exec
Content-Type: application/json

{
    "command": "python -c 'print(1+1)'",
    "timeout_ms": 30000
}
```

Response: `200 OK`
```json
{
    "stdout": "2\n",
    "stderr": "",
    "exit_code": 0
}
```

### Streaming Exec
```
POST /v1/sandboxes/:id/streaming-exec

Response: text/event-stream
data: {"stream_type":1,"data":"2\n"}
data: {"stream_type":3,"exit_code":0}
```

### Pause
```
POST /v1/sandboxes/:id/pause
```

### Resume
```
POST /v1/sandboxes/:id/resume
```

### Fork
```
POST /v1/sandboxes/:id/fork

Response: 201 Created
{
    "sandbox": {
        "id": "sb_child_123",
        "parent_id": "sb_parent_456"
    }
}
```

### Hibernate
```
POST /v1/sandboxes/:id/hibernate
```

### Resume from Hibernate
```
POST /v1/sandboxes/:id/resume-hibernate
```

## Checkpoint Operations

### Create Checkpoint
```
POST /v1/sandboxes/:id/checkpoints
{"name": "before-test", "description": "before running test"}
```

### List Checkpoints
```
GET /v1/sandboxes/:id/checkpoints
```

### Restore Checkpoint
```
POST /v1/sandboxes/:id/checkpoints/:name/restore
```

## Replay Operations

### Start Recording
```
POST /v1/sandboxes/:id/replay/start
{"name": "debug-session"}
```

### Stop Recording
```
POST /v1/sandboxes/:id/replay/stop
```

### Get Replay Events
```
GET /v1/sandboxes/:id/replays/:replay_id/events
```

## Template Operations

### List Templates
```
GET /v1/templates
```

### Create Template
```
POST /v1/templates
{"name": "python-3.11", "base_image": "python:3.11-slim"}
```
