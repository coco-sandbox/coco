# Control Flow

## If Expressions

Zig's `if` only accepts `bool` - no truthy/falsy coercion.

### Basic

```zig
if (condition) {
    // true branch
} else {
    // false branch
}
```

### As Expression

```zig
const max = if (a > b) a else b;
```

### With Optionals

```zig
const value: ?i32 = 42;
if (value) |v| {
    // v is i32 (not null)
} else {
    // value was null
}
```

### With Error Unions

```zig
const result: anyerror!i32 = 42;
if (result) |v| {
    // success - v is i32
} else |err| {
    // error - err is the error
}
```

## While Loops

### Basic

```zig
var i: usize = 0;
while (i < 10) {
    i += 1;
}
```

### With Continue Expression

```zig
var sum: usize = 0;
var i: usize = 1;
while (i <= 10) : (i += 1) {
    sum += i;
}
```

### With Break

```zig
while (true) {
    if (done) break;
    // ...
}
```

### With Label

```zig
outer: while (true) {
    while (true) {
        break :outer;
    }
}
```

### With Optionals

```zig
var value: ?i32 = 5;
while (value) |v| {
    std.debug.print("{}\n", .{v});
    value = if (v > 0) v - 1 else null;
}
```

### Inline While

```zig
comptime var i = 0;
inline while (i < 3) : (i += 1) {
    const T = switch (i) {
        0 => f32,
        1 => i8,
        else => bool,
    };
}
```

## For Loops

### Basic

```zig
const items = [_]i32{ 4, 5, 3 };

for (items) |value| {
    // value
}
```

### With Index

```zig
for (items, 0..) |value, index| {
    // value and index
}
```

### Over Range

```zig
for (0..10) |i| {
    // i from 0 to 9
}
```

### With Label

```zig
outer: for (1..6) |_| {
    for (1..6) |_| {
        break :outer;
    }
}
```

### Reference Modification

```zig
var items = [_]i32{ 3, 4, 2 };
for (&items) |*value| {
    value.* += 1;
}
```

### Inline For

```zig
const nums = [_]i32{ 2, 4, 6 };
inline for (nums) |i| {
    const T = switch (i) { ... };
}
```

## Switch

### As Statement

```zig
const x: i8 = 10;
switch (x) {
    -1...1 => x = -x,
    10, 100 => x = @divExact(x, 10),
    else => {},
}
```

### As Expression

```zig
const result = switch (value) {
    1 => "one",
    2 => "two",
    else => "other",
};
```

### With Tagged Union

```zig
const Item = union(enum) {
    a: u32,
    c: Point,
};

switch (item) {
    .a => |v| {},
    .c => |*v| { v.x += 1; },
}
```

### Exhaustive Switching

All cases must be handled:

```zig
const Color = enum { auto, off, on };
switch (color) {
    .auto => {},
    .on => {},
    .off => {},  // required - exhaustive
}
```

### Non-exhaustive

```zig
const Number = enum(u8) { one, two, three, _ };
switch (num) {
    .one => {},
    .two => {},
    _ => {},  // catch-all
}
```

## Blocks

### Basic Block

```zig
{
    const x = 1;
    const y = 2;
    const z = x + y;
}
```

### Labeled Block with Break

```zig
const result = blk: {
    const x = 1;
    break :blk x + 2;
};
```

### Empty Block

```zig
const a = {};
const b = void{};
```

## Labeled Switch

```zig
sw: switch (value) {
    5 => continue :sw 4,
    1 => return,
    else => unreachable,
}
```

## Best Practices

- Use `switch` over multiple `if-else` for enum types
- Prefer `for` over `while` when iterating
- Use labeled blocks for complex early returns
- Handle all enum cases in switches
