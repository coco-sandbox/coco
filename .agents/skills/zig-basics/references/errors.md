# Error Handling

## Error Sets

Define custom error types:

```zig
const FileError = error{
    AccessDenied,
    OutOfMemory,
    FileNotFound,
};

const IoError = error{
    WriteFailed,
    ReadFailed,
};
```

## Error Unions

Combine error set with value type:

```zig
// May return error or u32
const result: FileError!u32 = tryOpenFile();
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
    // ...
}
```

### catch

Provide fallback value:

```zig
const value = maybeError catch 0;  // fallback to 0
```

### catch with payload

```zig
maybeError catch |err| {
    std.debug.print("Error: {}\n", .{err});
    return err;
};
```

## errdefer

Cleanup on error only:

```zig
fn initVm() error{Failed}!Vm {
    const vm = try createVm();
    errdefer destroyVm(vm);  // runs on error
    return vm;
}
```

## Error Set Coercion

Smaller error set coerces to larger:

```zig
const SmallError = error{NotFound};
const LargeError = error{NotFound, OutOfMemory};

const err: LargeError = SmallError.NotFound;  // OK
```

## anyerror

Global error set (use sparingly):

```zig
fn risky() anyerror!void {
    // can return any error
}
```

## Best Practices

- Use `try` when you can't handle the error locally
- Use `catch` with fallback for recoverable errors
- Use `errdefer` for cleanup on failure
- Create specific error sets for your module
