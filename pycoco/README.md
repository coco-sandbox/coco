# Coco Python SDK

The official Python SDK for [Coco](https://github.com/cocoai/coco), an agent-native sandbox runtime.

## Installation

```bash
pip install coco
```

## Quick Start

```python
from coco import Sandbox

# Create sandbox
sandbox = Sandbox.create(template="alpine", memory_mb=512)

# Execute code
result = sandbox.run_code("print('hello')")
print(result.stdout)  # "hello\n"

# Fork for parallel exploration
fork = sandbox.fork()

# Create checkpoint
sandbox.checkpoint("before-mutation")

# Undo if needed
checkpoint = sandbox.undo()

# Redo if needed
checkpoint = sandbox.redo()

# Replay recording
replay = sandbox.replay_start()
# ... do things ...
replay = sandbox.replay_stop()

# Hibernate during idle
sandbox.hibernate()

# Resume when needed
sandbox.resume()

# Cleanup
sandbox.destroy()
```

## Features

- **Sandbox Management**: Create, pause, resume, hibernate, destroy
- **Code Execution**: Run arbitrary code with timeout support
- **Fork**: Clone running sandboxes for parallel exploration
- **Checkpoint**: Snapshot sandbox state for later restore
- **Hibernate**: Persist to disk for ultra-fast resume

## API Reference

### Sandbox.create()

Create a new sandbox.

```python
sandbox = Sandbox.create(
    template="alpine",  # OS template
    memory_mb=512,       # Memory in MB
    vcpus=2,            # Virtual CPUs
    name="my-sandbox",  # Optional name
    labels={"env": "test"},  # Optional labels
)
```

### sandbox.run_code()

Execute code in the sandbox.

```python
result = sandbox.run_code(
    "echo $HOME",
    timeout_ms=30000,
    working_dir="/tmp",
    env={"MY_VAR": "value"},
)
print(result.stdout)
print(result.stderr)
print(result.exit_code)
```

### sandbox.fork()

Clone the sandbox for parallel work.

```python
fork = sandbox.fork()
```

### sandbox.checkpoint()

Create a named snapshot.

```python
cp = sandbox.checkpoint("before-test")
```

### sandbox.hibernate()

Suspend to disk for fast resume.

```python
result = sandbox.hibernate()
print(result.size_bytes)
print(result.duration_ms)
```

## License

Apache 2.0