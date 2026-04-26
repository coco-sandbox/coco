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
fn initVm() error{Failed}!Vm {
    const vm = try createVm();
    errdefer destroyVm(vm);  // Only runs on error
    return vm;
}
```

### With Error Capture

```zig
errdefer |err| {
    std.debug.print("Cleanup after error: {}\n", .{err});
}
```

## Common Patterns

### File Handle

```zig
const file = try std.fs.cwd().openFile("data.txt", .{});
defer file.close();
```

### Memory Allocation

```zig
const mem = try allocator.alloc(u8, 1024);
defer allocator.free(mem);
```

### Lock/Mutex

```zig
mutex.lock();
defer mutex.unlock();
// Safe to use, unlocked on block exit
```

### Multiple Resources

```zig
const a = try openResourceA();
errdefer closeResourceA(a);

const b = try openResourceB();
errdefer closeResourceB(b);

// If b fails, a is cleaned up via errdefer
```

## defer vs errdefer

| Feature | defer | errdefer |
|---------|-------|----------|
| Runs on success | Yes | No |
| Runs on error | Yes | Yes |
| Runs on return | Yes | No |
| Capture error | No | Yes |

## Best Practice

Always use `defer` or `errdefer` for:
- File handles
- Memory allocation
- Lock acquisition
- Any resource that needs cleanup
