# Style Guide

## Naming Conventions

### Functions

camelCase:

```zig
fn myFunction() void {}
fn calculateValue() i32 {}
```

### Types (struct, enum, union, opaque)

TitleCase:

```zig
const MyStruct = struct {};
const MyEnum = enum {};
const MyUnion = union {};
```

### Variables and Constants

snake_case:

```zig
var my_variable: i32 = 0;
const my_constant: i32 = 42;
```

### File Names

snake_case for files:

```
my_file.zig
helper_utils.zig
```

### Modules/Directories

snake_case:

```
utils/
memory/
```

## Formatting

Use `zig fmt`:

```bash
zig fmt your_file.zig
```

### Rules

- 4 space indentation
- Open braces on same line
- 100 character line limit (soft)
- Trailing commas allowed

## Comments

### Regular Comments

```zig
// This is a comment
```

### Doc Comments

```zig
/// This documents the following declaration
const ANSWER: i32 = 42;
```

### Top-Level Doc

```zig
//! This documents the current file/module
```

## Best Practices

### Avoid Redundant Names

```zig
// Bad
const StringValue = struct { value: []u8 };

// Good
const String = struct { value: []u8 };
```

### Use Descriptive Names

```zig
// Good
const max_connections = 100;
const file_not_found_error = error.FileNotFound;

// Avoid
const x = 100;
const err = error.A;
```

### Prefer Explicit Over Implicit

```zig
// Clearer
const count: usize = items.len;

// Less clear
const count = items.len;
```

## Imports

```zig
const std = @import("std");
const os = std.os;
const expect = std.testing.expect;
```

## Type Annotations

Use when inference might be unclear:

```zig
// Clear return type
fn process(data: []const u8) !void {}

// Clear parameter type
fn handleError(err: error) void {}
```
