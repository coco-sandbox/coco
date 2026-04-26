---
name: zig-basics
description: Zig programming language fundamentals including testing, assignment, arrays, control flow, functions, defer, error handling, switch expressions, and runtime safety. Use when writing Zig code for coco-visor, coco-agent, or any Zig component in the Coco project.
license: MIT
metadata:
  author: Coco Team
  version: 1.0.0
  domain: language
  triggers: Zig, zig, @import, std, zig test, KVM, visor, agent
  role: specialist
  scope: implementation
  output-format: code
  related-skills: golang-pro, golang-grpc
---

# Zig Basics for Coco

Zig developer specializing in low-level systems programming for Coco's data plane components (Visor, Agent, Fork). Expert in Zig 0.12+ syntax, memory management, and Linux system programming.

## Core Workflow

1. **Analyze requirements** — Understand the component's purpose and constraints
2. **Design** — Plan memory layout, error handling strategy, and syscall usage
3. **Implement** — Write idiomatic Zig with explicit memory management
4. **Test** — Run `zig test` and verify all tests pass
5. **Build** — Use `zig build -Doptimize=ReleaseSafe` for production

## Running Tests

Run Zig tests with:

```bash
zig test <file.zig>
```

Test file example:

```zig
const std = @import("std");
const expect = std.testing.expect;

test "always succeeds" {
    try expect(true);
}
```

Output: `All 1 tests passed.`

**Without `try`** - the test will fail because errors must be handled.

## Reference Guide

| Topic | Reference | Load When |
|-------|-----------|-----------|
| Variables | `references/assignment.md` | const, var, @as, undefined |
| Arrays | `references/arrays.md` | [N]T, len, slices |
| Control Flow | `references/control-flow.md` | if, while, for, switch |
| Functions | `references/functions.md` | fn, recursion, parameters |
| Defer | `references/defer.md` | defer, errdefer, cleanup |
| Errors | `references/errors.md` | error sets, error unions, try, catch |
| Safety | `references/safety.md` | @setRuntimeSafety, unreachable |

## Key Patterns for Coco

### Memory Management

```zig
// Allocate aligned memory
const aligned_mem = try allocator.alignedAlloc(u8, 16, size);

// Slice from buffer
const slice = buffer[start..end];
```

### KVM Bindings

```zig
// Direct syscall for KVM
const ioctl_ret = linux.ioctl(vm_fd, KVM_CREATE_VM, 0);
```

### VSock Communication

```zig
// Non-blocking VSock setup
const vsock = try std.os.socket(std.os.AF_VSOCK, std.os.SOCK_STREAM, 0);
```

### Error Handling

```zig
fn initVisor() error{InitFailed}!Visor {
    const vm = try createVm();
    errdefer destroyVm(vm);
    return Visor{ .vm = vm };
}
```

## Constraints

### MUST DO
- Use `zig fmt` before committing
- Run `zig test` with safety on during development
- Use `errdefer` for all cleanup that must run on error
- Prefer `const` over `var`
- Use `@as` for explicit type coercion

### MUST NOT
- Use `undefined` unless type annotation is required
- Ignore errors with `_ = error_val`
- Use try inside loops (use catch instead)

### Build Commands for Coco

```bash
# Build Visor
cd daemon/coco-visor && zig build -Doptimize=ReleaseSafe

# Build Agent
cd daemon/coco-agent && zig build -Doptimize=ReleaseSafe

# Run tests
zig test test/
```
