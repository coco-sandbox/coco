# C Interop

## @cImport

Import C symbols directly:

```zig
const c = @cImport({
    @cDefine("_NO_CRT_STDIO_INLINE", "1");
    @cInclude("stdio.h");
});

pub fn main() void {
    _ = c.printf("Hello from C!\n");
}
```

## extern Functions

Declare external functions:

```zig
extern "c" fn printf(format: [*:0]const u8, ...) c_int;
extern fn malloc(size: usize) ?*anyopaque;
extern fn free(ptr: *anyopaque) void;
```

## extern Variables

```zig
extern var environ: [*]?[*]u8;
```

## export Functions

Export functions for C to call:

```zig
export fn add(a: i32, b: i32) i32 {
    return a + b;
}
```

Build as shared library:
```bash
zig build-lib math.zig -dynamic
```

## C Types

Zig provides C-compatible types:

```zig
c_char, c_short, c_ushort
c_int, c_uint
c_long, c_ulong
c_longlong, c_ulonglong
c_longdouble
```

## C Pointers

Avoid when possible. Use for translated C code only:

```zig
[*]c_char     // many-item C pointer
?[*]c_char    // optional C pointer
```

## @cDefine / @cInclude

```zig
const c = @cImport({
    @cDefine("MY_MACRO", "1");
    @cInclude("myheader.h");
});
```

## @cUndef

```zig
@cUndef("MY_MACRO");
```

## Building with libc

```bash
zig build-exe program.zig -lc
```

## Example: Using C Library

```zig
const c = @cImport(@cInclude("stdio.h"));

pub fn main() void {
    _ = c.fopen("file.txt", "r");
}
```

## Variadic Functions

```zig
extern fn printf(format: [*:0]const u8, ...) c_int;

printf("Value: %d\n", 42);
```

## Best Practices

- Prefer Zig-native code over C interop
- Use @cImport only for C headers
- Use extern for system calls
- Export with C ABI for libraries
