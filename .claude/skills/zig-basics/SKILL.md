---
name: zig-basics
description: Zig programming language fundamentals for Coco project. Covers testing, assignment, arrays, control flow, functions, defer, error handling, switch expressions, memory management, safety, and C interop. Use when writing Zig code for coco-visor, coco-agent, coco-fork, or coco-net components.
license: MIT
metadata:
  author: Coco Team
  version: 2.0.0
  domain: language
  triggers: Zig, zig, @import, std, zig test, KVM, visor, agent
  role: specialist
  scope: implementation
  output-format: code
  related-skills: golang-pro, golang-grpc
---

# Zig Basics for Coco

Zig is a general-purpose programming language designed for maintaining robust, optimal, and reusable software. This skill covers Zig 0.16 fundamentals with emphasis on low-level systems programming for Coco's data plane components.

## Core Philosophy

- **Robust** - Correct behavior for edge cases (e.g., out of memory)
- **Optimal** - Write programs the best way they can behave and perform
- **Reusable** - Same code works in many environments
- **Maintainable** - Precisely communicate intent to compiler and programmers

## Quick Start

```zig
const std = @import("std");

pub fn main() void {
    std.debug.print("Hello, Coco!\n", .{});
}
```

## Build Commands

```bash
# Run tests
zig test <file.zig>

# Build for release
zig build -Doptimize=ReleaseSafe

# Format code
zig fmt <file.zig>
```

## Reference Guide

| Topic | Reference | When Needed |
|-------|-----------|-------------|
| Basics | `references/basics.md` | Hello world, types, strings |
| Variables | `references/variables.md` | const, var, undefined |
| Operators | `references/operators.md` | Arithmetic, bitwise, comparison |
| Data Structures | `references/data-structures.md` | Arrays, structs, enums, unions |
| Control Flow | `references/control-flow.md` | if, while, for, switch |
| Functions | `references/functions.md` | fn declarations, parameters |
| Error Handling | `references/error-handling.md` | Error sets, try, catch |
| Defer | `references/defer.md` | Cleanup on scope exit |
| Memory | `references/memory.md` | Allocation, pointers, slices |
| Safety | `references/safety.md` | Runtime checks, build modes |
| Std Library | `references/std.md` | std.debug, std.mem, std.heap, etc. |
| Build System | `references/build-system.md` | zig build, build.zig, targets |
| C Interop | `references/c-interop.md` | @cImport, extern |
| Style | `references/style.md` | Naming conventions |

## Coco-Specific Patterns

### Memory Management
```zig
const aligned_mem = try allocator.alignedAlloc(u8, 16, size);
const slice = buffer[start..end];
```

### KVM Bindings
```zig
const ioctl_ret = linux.ioctl(vm_fd, KVM_CREATE_VM, 0);
```

### VSock Communication
```zig
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
- Run `zig test` with safety during development
- Use `errdefer` for cleanup on error
- Prefer `const` over `var`
- Use `@as` for explicit type coercion

### MUST NOT
- Use `undefined` without type annotation
- Ignore errors with `_ = error_val`
- Use try inside loops (use catch instead)

## Build Commands for Coco

```bash
# Build Visor
cd daemon/coco-visor && zig build -Doptimize=ReleaseSafe

# Build Agent
cd daemon/coco-agent && zig build -Doptimize=ReleaseSafe

# Run tests
zig test test/
```
