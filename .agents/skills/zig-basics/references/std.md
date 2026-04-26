# Zig Standard Library (std)

The Zig Standard Library (`std`) provides commonly used algorithms, data structures, and definitions.

## Accessing std

```zig
const std = @import("std");
```

## Common Modules

| Module | Description |
|--------|-------------|
| `std.debug` | Debugging utilities |
| `std.mem` | Memory operations |
| `std.heap` | Allocators |
| `std.fs` | File system |
| `std.os` | OS syscalls |
| `std.process` | Process management |
| `std.math` | Math functions |
| `std.time` | Time operations |
| `std.io` | I/O utilities |
| `std.builtin` | Compiler info |
| `std.testing` | Testing utilities |

## std.debug

```zig
const std = @import("std");

// Print to stderr
std.debug.print("Hello {}!\n", .{world});

// Assertions
std.debug.assert(condition);
```

## std.mem

```zig
const mem = std.mem;

// Copy memory
@memcpy(dest_ptr, source_ptr);

// Set memory
@memset(ptr, value);

// Compare
const equal = mem.eql(u8, str1, str2);

// Find
const found = mem.indexOf(u8, data, pattern);

// Split
var it = mem.split(u8, data, ",");
```

## std.heap

```zig
const heap = std.heap;

// Page allocator
const alloc = heap.page_allocator;

// Arena allocator (recommended for CLI)
var arena = heap.ArenaAllocator.init(heap.page_allocator);
defer arena.deinit();
const allocator = arena.allocator();

// Fixed buffer
var buffer: [1000]u8 = undefined;
var fba = heap.FixedBufferAllocator.init(&buffer);
const alloc2 = fba.allocator();
```

## std.fs

```zig
const fs = std.fs;

// Open file
const file = try fs.cwd().openFile("path", .{});
defer file.close();

// Read entire file
const contents = try file.readToEndAlloc(allocator, max_size);

// Create directory
try fs.cwd().makeDir("dir");
```

## std.os

```zig
const os = std.os;

// Syscalls
const fd = os.socket(os.AF_INET, os.SOCK_STREAM, 0);
const ret = os.write(fd, data);

// Get errno
const errno = os.errno(ret);
```

## std.math

```zig
const math = std.math;

// Basic
const sin = math.sin(f64, x);
const cos = math.cos(f64, x);
const sqrt = math.sqrt(f64, x);
const pow = math.pow(f64, base, exp);

// Min/Max
const min = math.min(a, b);
const max = math.max(a, b);

// Integer
const abs = math.absInt(i32, -5);
const div = math.divFloor(i32, 10, 3);
```

## std.testing

```zig
const testing = std.testing;

// Basic assertions
try testing.expect(condition);
try testing.expectEqual(expected, actual);
try testing.expectError(expected_error, actual_error);

// Failing allocator
var fa = testing.FailingAllocator.init(allocator, .{
    .fail_index = 10,
});
```

## std.process

```zig
const process = std.process;

// Command line args
const args = try std.process.argsAlloc(allocator);
defer std.process.argsFree(allocator, args);

// Exit
process.exit(0);
```

## std.time

```zig
const time = std.time;

// Current timestamp
const now = time.nanoTimestamp();

// Sleep
time.sleep(1_000_000_000); // 1 second
```

## std.builtin

```zig
const builtin = @import("builtin");

// Target info
const target = builtin.target;
const os_tag = builtin.os.tag;
const arch = builtin.cpu.arch;

// Build mode
const mode = builtin.mode; // .Debug, .ReleaseSafe, etc.

// Is test build
const is_test = builtin.is_test;
```

## Formatted Output

```zig
const std = @import("std");

// Print with formatting
std.debug.print("int: {}, float: {:.2}, str: {s}\n", .{
    42,
    3.14159,
    "hello",
});
```

Format specifiers:
- `{}` - default
- `{d}` - decimal
- `{x}` - hex
- `{s}` - string
- `{.2}` - precision

## Local Documentation

View std docs locally:
```bash
zig std
```
