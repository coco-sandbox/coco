# Variables

## Syntax

```zig
const identifier: Type = value;  // Immutable
var identifier: Type = value;     // Mutable
```

## const - Immutable

```zig
const constant: i32 = 5;
const name: []const u8 = "coco";
const computed = 1 + 2;  // evaluated at compile-time
```

## var - Mutable

```zig
var counter: u32 = 0;
counter += 1;  // OK - can modify
```

## undefined

Use to leave variables uninitialized:

```zig
var x: i32 = undefined;
x = 1;  // now initialized
```

**Important**: `undefined` means the value could be anything. Using it before assignment is undefined behavior.

## Type Inference with @as

```zig
const inferred = @as(i32, 5);
var mutable = @as(u64, 100);
```

## Destructuring

```zig
const tuple = .{ 1, 2, 3 };
const a, const b, const c = tuple;

var arr = [_]u32{ 4, 5, 6 };
var x: u32 = undefined;
var y: u32 = undefined;
x, y = arr;
```

## Container Level Variables

Variables at file scope have static lifetime:

```zig
var global: i32 = 123;

const S = struct {
    var counter: i32 = 0;
};

fn increment() void {
    S.counter += 1;
}
```

## Thread Local Variables

```zig
threadlocal var thread_id: u32 = 0;
```

Each thread gets its own copy.

## Static Local Variables

```zig
fn counter() i32 {
    const S = struct {
        var count: i32 = 0;
    };
    S.count += 1;
    return S.count;
}
```

## comptime Variables

```zig
comptime var x: i32 = 0;
x += 1;  // evaluated at compile-time
```

## Best Practices

- Prefer `const` over `var`
- Initialize all variables
- Use explicit types when inference may be unclear
- Avoid `undefined` when possible
