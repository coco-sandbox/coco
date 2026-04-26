# Data Structures

## Arrays

### Syntax

```zig
[N]T  // Array of N elements of type T
```

### Array Literals

```zig
const arr = [5]u8{ 'h', 'e', 'l', 'l', 'o' };
const inferred = [_]u8{ 'h', 'e', 'l', 'l', 'o' };
```

### Accessing Elements

```zig
const first = arr[0];
const last = arr[arr.len - 1];
```

### Multi-dimensional Arrays

```zig
const matrix = [2][3]u32{
    [_]u32{ 1, 2, 3 },
    [_]u32{ 4, 5, 6 },
};
```

### Sentinel-Terminated Arrays

```zig
const terminated: [5:0]u8 = .{ 1, 2, 3, 4, 0 };
```

## Vectors

Vectors are SIMD-optimized groups of values:

```zig
const vec: @Vector(4, i32) = .{ 1, 2, 3, 4 };
const doubled = vec * vec;  // element-wise
```

## Pointers

### Single-Item Pointer

```zig
*T        // pointer to exactly one item
const ptr: *i32 = &x;
const val = ptr.*;
```

### Many-Item Pointer

```zig
[*]T      // pointer to unknown number of items
```

### Array Pointer

```zig
*[N]T     // pointer to N items
const arr_ptr: *[5]i32 = &array;
```

### Pointer Arithmetic

```zig
var ptr: [*]i32 = &array;
ptr += 1;  // advance
const val = ptr[0];
```

### Optional Pointers

```zig
var opt_ptr: ?*i32 = null;
opt_ptr = &x;
if (opt_ptr) |p| {
    // p is *i32
}
```

## Slices

Slices are pointers with length:

```zig
[]T        // slice of T
const slice: []const u8 = arr[1..4];
const len = slice.len;
```

### Sentinel-Terminated Slices

```zig
[:0]T     // slice guaranteed to end with 0
const terminated_slice: [:0]const u8 = "hello";
```

## Structs

### Declaration

```zig
const Point = struct {
    x: f32,
    y: f32,
};

const p = Point{ .x = 0.5, .y = 0.5 };
```

### Methods

```zig
const Vec3 = struct {
    x: f32,
    y: f32,
    z: f32,

    pub fn init(x: f32, y: f32, z: f32) Vec3 {
        return Vec3{ .x = x, .y = y, .z = z };
    }

    pub fn dot(self: Vec3, other: Vec3) f32 {
        return self.x * other.x + self.y * other.y + self.z * other.z;
    }
};
```

### Default Field Values

```zig
const Foo = struct {
    a: i32 = 123,
    b: i32,
};
```

### packed struct

Packed structs have guaranteed in-memory layout:

```zig
const Flags = packed struct {
    a: u4,
    b: u4,
};
```

### Anonymous Struct

```zig
const point = .{
    .x = 1.0,
    .y = 2.0,
};
```

## Enums

### Basic Enum

```zig
const Color = enum {
    red,
    green,
    blue,
};

const c: Color = .red;
```

### Enum with Tag Type

```zig
const Value = enum(u2) {
    zero,
    one,
    two,
};
```

### Enum Methods

```zig
const Suit = enum {
    clubs,
    spades,
    diamonds,
    hearts,

    pub fn isRed(self: Suit) bool {
        return self == .diamonds or self == .hearts;
    }
};
```

### Non-exhaustive Enum

```zig
const Number = enum(u8) {
    one,
    two,
    three,
    _,
};
```

## Unions

### Bare Union

```zig
const Payload = union {
    int: i64,
    float: f64,
    boolean: bool,
};
```

Only one field active at a time.

### Tagged Union

```zig
const Result = union(enum) {
    ok: u32,
    err: void,
};

const r = Result{ .ok = 42 };
switch (r) {
    .ok => |v| std.debug.print("{}\n", .{v}),
    .err => {},
}
```

### packed union

```zig
const PackedU = packed union {
    a: u4,
    b: i4,
};
```

## Tuples

Tuples are anonymous structs with numeric field names:

```zig
const tuple = .{ 1, "hello", true };
const first = tuple.0;
const len = tuple.len;
```

### Tuple Destructuring

```zig
const a, const b, const c = tuple;
```

## Opaque

Opaque types have unknown size:

```zig
const Handle = opaque {};
const handle: *Handle = undefined;
```
