# Arrays

## Syntax

```zig
[N]T  // N elements of type T
```

## Array Literals

```zig
const a = [5]u8{ 'h', 'e', 'l', 'l', 'o' };
const b = [_]u8{ 'w', 'o', 'r', 'l', 'd' };  // infer size
```

## Array Length

```zig
const arr = [_]u8{ 'a', 'b', 'c' };
const len = arr.len;  // 3
```

## Accessing Elements

```zig
const first = arr[0];
const last = arr[len - 1];
```

## Multi-dimensional Arrays

```zig
const matrix = [2][3]u32{
    [_]u32{ 1, 2, 3 },
    [_]u32{ 4, 5, 6 },
};
```

## Slices

Slices are pointers to arrays with length:

```zig
const arr = [_]u8{ 1, 2, 3, 4, 5 };
const slice: []const u8 = arr[1..4];  // [2, 3, 4]
```

## Common Operations

```zig
// Copy array
const src = [_]u8{ 1, 2, 3 };
var dst = src;
@memcpy(&dst, &src);

// Compare arrays
const a = [_]u8{ 1, 2, 3 };
const b = [_]u8{ 1, 2, 3 };
const equal = std.mem.eql(u8, &a, &b);
```

## For Coco

Used heavily in:
- Memory buffers for VM state
- Packet data in networking
- Configuration arrays
