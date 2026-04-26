# Basics

## Hello World

```zig
const std = @import("std");

pub fn main() void {
    std.debug.print("Hello, World!\n", .{});
}
```

Build and run:
```bash
zig build-exe hello.zig
./hello
```

## Comments

Zig supports 3 types of comments:

```zig
// Single-line comment (ignored)

// Doc comment - documents the following declaration
/// This is a doc comment
const ANSWER: i32 = 42;

// Top-level doc comment - documents the current module
//! This module provides utility functions
```

## Identifiers

- Must start with alphabetic character or underscore
- May be followed by alphanumeric characters or underscores
- Cannot overlap with keywords

```zig
const normalName = 1;
const _private = 2;
const @"identifier with spaces" = 3;
```

## Primitive Types

| Type | Description |
|------|-------------|
| `i8`, `u8` | 8-bit signed/unsigned |
| `i16`, `u16` | 16-bit signed/unsigned |
| `i32`, `u32` | 32-bit signed/unsigned |
| `i64`, `u64` | 64-bit signed/unsigned |
| `i128`, `u128` | 128-bit signed/unsigned |
| `isize`, `usize` | Pointer-sized signed/unsigned |
| `f32`, `f64` | 32/64-bit floating point |
| `bool` | true or false |
| `void` | Always the value `void{}` |
| `type` | The type of types |
| `anyerror` | An error code |
| `comptime_int` | Integer literal type (compile-time only) |
| `comptime_float` | Float literal type (compile-time only) |

## Primitive Values

```zig
const true_val: bool = true;
const false_val: bool = false;
const null_val: ?u32 = null;
const undef: u32 = undefined;
```

## String Literals

String literals are constant pointers to null-terminated byte arrays:

```zig
const str: *const [5:0]u8 = "hello";
const slice: []const u8 = "hello";  // coerces to slice
```

### Escape Sequences

| Sequence | Name |
|----------|------|
| `\n` | Newline |
| `\r` | Carriage Return |
| `\t` | Tab |
| `\\` | Backslash |
| `\'` | Single Quote |
| `\"` | Double Quote |
| `\xNN` | Hex byte |
| `\u{NNNNNN}` | Unicode scalar |

### Multiline Strings

```zig
const c_code =
    \\#include <stdio.h>
    \\int main() { return 0; }
;
```

## Values Example

```zig
const std = @import("std");

pub fn main() void {
    // Integers
    const one_plus_one: i32 = 1 + 1;

    // Floats
    const seven_div_three: f32 = 7.0 / 3.0;

    // Boolean
    const and_result = true and false;
    const or_result = true or false;
    const not_result = !true;

    // Optional
    var optional: ?[]const u8 = null;
    optional = "hi";

    // Error union
    var number_or_error: anyerror!i32 = 1234;
    number_or_error = error.SomeError;
}
```

## Zig Test

```zig
const std = @import("std");

test "expect addOne adds one to 41" {
    try std.testing.expect(addOne(41) == 42);
    try std.testing.expectEqual(42, addOne(41));
}

fn addOne(number: i32) i32 {
    return number + 1;
}
```

Run tests:
```bash
zig test file.zig
```

## @import

The `@import` builtin imports files:

```zig
const std = @import("std");           // Standard library
const builtin = @import("builtin");    // Compiler info
const root = @import("root");         // Root module
```
