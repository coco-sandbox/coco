# Functions

## Declaration

```zig
fn functionName(arg: Type) ReturnType {
    return value;
}
```

### Examples

```zig
fn add(a: i32, b: i32) i32 {
    return a + b;
}

fn greet(name: []const u8) void {
    std.debug.print("Hello, {s}!\n", .{name});
}
```

## Parameters

### Immutable Parameters

Function parameters are always immutable:

```zig
fn increment(x: u32) void {
    x += 1;  // ERROR - x is immutable
}
```

### Pointer Parameters

```zig
fn increment(x: *u32) void {
    x.* += 1;
}

var num: u32 = 5;
increment(&num);
```

### anytype Parameters

```zig
fn printValue(x: anytype) void {
    std.debug.print("{}\n", .{x});
}
```

## Return Types

### Void Return

```zig
fn doNothing() void {}
```

### Error Union Return

```zig
fn mayFail() error{Failed}!i32 {
    if (something) return error.Failed;
    return 42;
}
```

### Optional Return

```zig
fn findItem(items: []const i32, target: i32) ?usize {
    for (items, 0..) |item, i| {
        if (item == target) return i;
    }
    return null;
}
```

## Recursion

```zig
fn fibonacci(n: u16) u16 {
    if (n <= 1) return n;
    return fibonacci(n - 1) + fibonacci(n - 2);
}
```

**Warning**: Zig cannot determine max stack size. May cause stack overflow.

## inline Functions

```zig
inline fn double(x: i32) i32 {
    return x * 2;
}
```

Forces inlining at call site. Useful for:
- Debugging (single stack frame)
- Compile-time evaluation

## naked Functions

```zig
export fn _start() noreturn {
    // Entry point with no stack frame
    while (true) {}
}
```

## Function Pointers

```zig
const Op = *const fn(i32, i32) i32;
const add: Op = add;
const result = add(1, 2);
```

## Calling Conventions

```zig
extern "c" fn printf(format: [*:0]const u8, ...) c_int;

fn syscall(num: usize, arg: usize) usize {
    return asm volatile ("syscall"
        : [ret] "={rax}" (-> usize)
        : [num] "{rax}" (num), [arg] "{rdi}" (arg)
        : .{ .rcx = true, .r11 = true });
}
```

## Ignoring Return Values

```zig
_ = someFunction();  // discard result
```

## Defer in Functions

```zig
fn readFile(path: []const u8) ![]u8 {
    const file = try std.fs.cwd().openFile(path, .{});
    defer file.close();
    return try file.readToEndAlloc(std.heap.page_allocator, std.math.maxInt(usize));
}
```

## Best Practices

- Use `errdefer` for cleanup on error
- Prefer small functions for readability
- Use function pointers for callbacks
- Document error returns
