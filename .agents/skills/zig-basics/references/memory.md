# Memory

## Allocation

Zig has no default allocator. Functions that need memory accept an Allocator parameter:

```zig
const mem = try allocator.alloc(u8, 1024);
defer allocator.free(mem);
```

### Common Allocators

```zig
// Fixed buffer - for known upper bounds
var buffer: [1000]u8 = undefined;
var fba = std.heap.FixedBufferAllocator.init(&buffer);
const allocator = fba.allocator();

// Arena - for related allocations
var arena = std.heap.ArenaAllocator.init(std.heap.page_allocator);
defer arena.deinit();
const arena_alloc = arena.allocator();

// Page allocator (system memory)
const page_alloc = std.heap.page_allocator;
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
```

## Slices

Slices are pointers with length:

```zig
[]T                    // mutable slice
[]const T              // immutable slice
[:0]T                 // sentinel-terminated slice

const slice = buffer[start..end];
const len = slice.len;
```

## Alignment

### Variable Alignment

```zig
var aligned: u32 align(16) = 0;
```

### Pointer Alignment

```zig
const ptr: *align(16) u32 = &value;
```

### @alignCast

```zig
const aligned = @alignCast(ptr);
```

## Lifetime

### Stack Variables

```zig
fn foo() void {
    var x: i32 = 1;
    // x is valid until function returns
}
```

### Global Variables

```zig
var global: i32 = 123;  // lifetime is entire program
```

### Heap Allocation

```zig
const ptr = try allocator.create(i32);
defer allocator.destroy(ptr);
```

## Memory Operations

### @memcpy

```zig
@memcpy(dest_ptr, source_ptr);
```

### @memset

```zig
@memset(ptr, value);
```

### @memmove

```zig
@memmove(dest, source);  // handles overlapping
```

## Zero-Terminated Strings

```zig
const c_string: [*:0]const u8 = "hello";
const len = std.mem.len(c_string);  // 5
```

## Slice Operations

```zig
const data = [_]u8{ 1, 2, 3, 4, 5 };

// Slice
const slice = data[1..4];  // [2, 3, 4]

// Slice with sentinel
const term_slice: [:0]u8 = "hello";
```

## Best Practices

1. Accept Allocator as parameter for generic code
2. Use defer for all allocations
3. Prefer ArenaAllocator for related allocations
4. Use FixedBufferAllocator when bounds are known
5. Document ownership and lifetime
