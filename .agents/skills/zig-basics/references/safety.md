# Runtime Safety

## Safety Modes

Zig provides safety checks that catch illegal behavior at runtime:

- **Debug/Safe** (default for `zig test`): Full safety checks
- **ReleaseSafe**: Safety on, optimized
- **ReleaseSmall**: Safety off, smallest binary
- **ReleaseFast**: Safety off, fastest

## Detectable Illegal Behavior

### Out of Bounds Access

```zig
const arr = [3]u8{ 1, 2, 3 };
const x = arr[5];  // PANIC: index out of bounds
```

### Division by Zero

```zig
const x = 5 / 0;  // PANIC
```

### Null Pointer Dereference

```zig
const ptr: *u32 = null;
const x = ptr.*;  // PANIC
```

## Disabling Safety

Use `@setRuntimeSafety(false)`:

```zig
test "unsafe access" {
    @setRuntimeSafety(false);
    const arr = [3]u8{ 1, 2, 3 };
    const x = arr[5];  // No panic, undefined behavior
}
```

## Unreachable

Assert that code cannot be reached:

```zig
fn asciiToUpper(x: u8) u8 {
    return switch (x) {
        'a'...'z' => x + 'A' - 'a',
        'A'...'Z' => x,
        else => unreachable,  // PANIC if reached
    };
}
```

**Use case:** Inform compiler that a branch is impossible, enabling optimizations.

## Best Practices for Coco

1. **Develop with safety on** - Use `zig test` with default settings
2. **Test with safety** before releasing
3. **Use ReleaseSafe** for production
4. **Use unreachable** only for truly impossible cases
5. **Profile** after disabling safety to ensure it's necessary

## Coco Build Modes

```bash
# Debug (full safety)
zig build

# Release (safety on)
zig build -Doptimize=ReleaseSafe

# ReleaseFast (no safety, fastest)
zig build -Doptimize=ReleaseFast
```
