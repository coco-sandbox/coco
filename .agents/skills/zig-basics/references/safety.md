# Runtime Safety

## Safety Modes

| Mode | Optimizations | Safety | Use |
|------|---------------|--------|-----|
| Debug | Off | Full | Development |
| ReleaseSafe | On | Full | Production |
| ReleaseSmall | On | Off | Small binaries |
| ReleaseFast | On | Off | Performance |

## Build Examples

```bash
zig build                    # Debug
zig build -Doptimize=ReleaseSafe  # Safe release
zig build -Doptimize=ReleaseFast   # Fast release
zig build -Doptimize=ReleaseSmall  # Small release
```

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

### Integer Overflow (non-wrapping)

```zig
var byte: u8 = 255;
byte += 1;  // PANIC in safe modes
```

## Disabling Safety

### Per Block

```zig
{
    @setRuntimeSafety(false);
    // unsafe code here
}
```

### Wrap with Wrapping Operators

```zig
const wrapped = @as(u8, 255) +% 1;  // wraps to 0
```

## unreachable

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

In ReleaseFast/ReleaseSmall: undefined behavior, not panic.

## @setRuntimeSafety

Toggle safety checks per scope:

```zig
test "safety toggle" {
    {
        @setRuntimeSafety(false);
        var x: u8 = 255;
        x += 1;  // no panic here
    }
    // safety restored
}
```

## Build Mode Detection

```zig
const builtin = @import("builtin");

fn debugOnly(code: fn() void) void {
    if (builtin.mode == .Debug) {
        code();
    }
}
```

## Common Safety Patterns

### Optional Unwrap

```zig
const value: ?i32 = null;
const x = value.?;  // PANIC if null
```

### Error Unwrap

```zig
const value: anyerror!i32 = error.SomeError;
const x = value catch unreachable;  // PANIC if error
```

### Type Cast

```zig
const x: u16 = 300;
const y: u8 = @intCast(x);  // PANIC: doesn't fit
```

## Best Practices

1. Develop with safety on (Debug mode)
2. Test with safety before releasing
3. Use ReleaseSafe for production
4. Use unreachable only for truly impossible cases
5. Profile after disabling safety to verify necessity

## Coco Build Modes

```bash
# Debug (full safety)
zig build

# Release (safety on)
zig build -Doptimize=ReleaseSafe

# ReleaseFast (no safety, fastest)
zig build -Doptimize=ReleaseFast
```
