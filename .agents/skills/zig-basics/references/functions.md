# Functions

## Function Declaration

```zig
fn functionName(arg: Type) ReturnType {
    return value;
}
```

- All arguments are immutable
- Functions are camelCase
- Return type follows argument list

## Examples

```zig
fn addFive(x: u32) u32 {
    return x + 5;
}

fn greet(name: []const u8) void {
    std.debug.print("Hello, {s}!\n", .{name});
}
```

## Recursion

```zig
fn fibonacci(n: u16) u16 {
    if (n == 0 or n == 1) return n;
    return fibonacci(n - 1) + fibonacci(n - 2);
}
```

**Warning**: Compiler cannot determine max stack size for recursion - may cause stack overflow.

## Ignoring Return Values

Use `_` to ignore return values (only inside functions):

```zig
_ = someFunction();  // discard result

// At global scope, this doesn't work
```

## Function Parameters Are Immutable

```zig
fn increment(x: u32) void {
    x += 1;  // ERROR - x is immutable
}
```

If you need mutation, pass a pointer:

```zig
fn increment(x: *u32) void {
    x.* += 1;
}

var num: u32 = 5;
increment(&num);
```

## Naked Functions (No Stack Frame)

For low-level code:

```zig
export fn _start() noreturn {
    // Entry point, no stack frame
    while (true) {}
}
