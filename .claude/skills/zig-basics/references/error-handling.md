# Error Handling

## Error Sets

Define custom error types:

```zig
const FileError = error{
    AccessDenied,
    OutOfMemory,
    FileNotFound,
};
```

## Error Unions

Combine error set with value type:

```zig
// May return error or u32
const result: FileError!u32 = mayFail();
```

## Creating Error Values

```zig
return error.FileNotFound;
```

## Handling Errors

### try

Propagate error up the call stack:

```zig
fn openConfig() error{NotFound}!Config {
    const file = try std.fs.cwd().openFile("config.toml", .{});
    defer file.close();
    // file is now available
}
```

### catch

Provide fallback value:

```zig
const value = maybeError catch 0;  // fallback to 0
```

### catch with Payload

```zig
maybeError catch |err| {
    std.debug.print("Error: {}\n", .{err});
    return err;
};
```

### if Expression

```zig
if (maybeError) |value| {
    // success
} else |err| {
    // failure
}
```

## errdefer

Execute only when function returns with an error:

```zig
fn initVm() error{Failed}!Vm {
    const vm = try createVm();
    errdefer destroyVm(vm);  // runs on error only
    return vm;
}
```

### errdefer with Capture

```zig
errdefer |err| {
    std.debug.print("Failed: {}\n", .{err});
}
```

## Error Set Coercion

Smaller error set coerces to larger:

```zig
const SmallError = error{NotFound};
const LargeError = error{NotFound, OutOfMemory};

const err: LargeError = SmallError.NotFound;  // OK
```

## Merging Error Sets

```zig
const A = error{ One };
const B = error{ Two };
const C = A || B;  // error{One, Two}
```

## Inferred Error Sets

```zig
pub fn add(a: i32, b: i32) !i32 {
    const ov = @addWithOverflow(a, b);
    if (ov[1] != 0) return error.Overflow;
    return ov[0];
}
```

## anyerror

Global error set (use sparingly):

```zig
fn risky() anyerror!void {
    // can return any error
}
```

## Error Return Traces

Enabled in Debug builds, shows full error path:

```bash
$ ./program
error: PermissionDenied
at bang1 (file.zig:34)
at baz (file.zig:22)
at bar (file.zig:17)
at foo (file.zig:7)
at main (file.zig:2)
```

## Best Practices

- Use specific error sets for modules
- Use `try` when you can't handle locally
- Use `catch` with fallback for recoverable errors
- Use `errdefer` for cleanup on failure
- Handle all errors in switches
