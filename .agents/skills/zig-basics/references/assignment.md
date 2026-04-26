# Assignment and Variables

## Syntax

```
(const|var) identifier[: type] = value;
```

## const - Immutable

```zig
const constant: i32 = 5;           // signed 32-bit constant
const name: []const u8 = "coco";   // string literal
```

## var - Mutable

```zig
var counter: u32 = 0;
counter += 1;  // OK
```

## Type Inference

Use `@as` for explicit coercion:

```zig
const inferred = @as(i32, 5);
var mutable = @as(u64, 100);
```

## undefined

Use only with type annotation:

```zig
const a: i32 = undefined;  // OK
var b: u32 = undefined;   // OK

// BAD - cannot infer type
// const c = undefined;  // ERROR
```

**Warning**: `undefined` in Zig means "undefined behavior" - cannot be detected or checked.

## Best Practices

- Prefer `const` over `var`
- Use explicit types when the inferred type may be unclear
- Initialize all variables - avoid `undefined` when possible
