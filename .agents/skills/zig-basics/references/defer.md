# Defer

## Basic Defer

Execute code when exiting the current block:

```zig
{
    defer std.debug.print("Exiting block\n", .{});
    std.debug.print("Inside block\n", .{});
}
// "Exiting block" prints here
```

## Multiple Defers

Multiple defers execute in **reverse order**:

```zig
{
    defer std.debug.print("first\n", .{});
    defer std.debug.print("second\n", .{});
    defer std.debug.print("third\n", .{});
}
// Output: third, second, first
```

## errdefer

Execute only when function returns with an error:

```zig
fn initVm() error{InitFailed}!Vm {
    const vm = try createVm();
    errdefer destroyVm(vm);  // Only runs on error
    return vm;
}
```

**Use case**: Cleanup that should only run on failure (not success).

## Common Patterns

### File Handle

```zig
const file = try std.fs.cwd().openFile("data.txt", .{});
defer file.close();
```

### Memory

```zig
const mem = try allocator.alloc(u8, 1024);
defer allocator.free(mem);
```

### Lock

```zig
mutex.lock();
defer mutex.unlock();
// Safe to use, unlocked on block exit
```

## Best Practice

Always use `defer` or `errdefer` for:
- File handles
- Memory allocation
- Lock acquisition
- Any resource that needs cleanup
