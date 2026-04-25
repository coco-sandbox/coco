# Coco JavaScript/TypeScript SDK

The official JS/TS SDK for [Coco](https://github.com/cocoai/coco), an agent-native sandbox runtime.

## Installation

```bash
npm install jscco
```

## Quick Start

```javascript
import { Sandbox } from 'jscco';

// Create sandbox
const sandbox = await Sandbox.create({ template: 'alpine', memoryMb: 512 });

// Execute code
const result = await sandbox.runCode("console.log('hello')");
console.log(result.stdout); // "hello\n"

// Fork for parallel exploration
const fork = await sandbox.fork();

// Create checkpoint
const cp = await sandbox.checkpoint("before-test");

// Undo if needed
const restored = await sandbox.undo();

// Replay recording
const replay = await sandbox.replayStart();
// ... do things ...
const stopped = await sandbox.replayStop();

// Hibernate during idle
const hib = await sandbox.hibernate();
console.log(hib.durationMs);

// Resume when needed
await sandbox.resume();

// Cleanup
await sandbox.destroy();
```

## Features

- **Sandbox Management**: Create, pause, resume, hibernate, destroy
- **Code Execution**: Run arbitrary code with timeout support
- **Fork**: Clone running sandboxes for parallel exploration
- **Checkpoint**: Snapshot sandbox state for later restore
- **Hibernate**: Persist to disk for ultra-fast resume
- **Replay**: Record and replay execution sessions

## API Reference

### Sandbox.create()

```javascript
const sandbox = await Sandbox.create({
  template: 'alpine',
  memoryMb: 512,
  vcpus: 2,
  name: 'my-sandbox',
  labels: { env: 'test' },
});
```

### sandbox.runCode()

```javascript
const result = await sandbox.runCode("echo $HOME", {
  timeoutMs: 30000,
  workingDir: '/tmp',
  env: { MY_VAR: 'value' },
});
console.log(result.stdout);
console.log(result.exitCode);
```

### sandbox.fork()

```javascript
const fork = await sandbox.fork();
```

### sandbox.checkpoint()

```javascript
const cp = await sandbox.checkpoint("before-test");
```

### sandbox.hibernate()

```javascript
const result = await sandbox.hibernate();
console.log(result.sizeBytes);
console.log(result.durationMs);
```

## License

Apache 2.0
