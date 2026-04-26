# Control Flow

## If Expressions

Zig's `if` only accepts `bool` - no truthy/falsy coercion.

```zig
if (condition) {
    // true branch
} else {
    // false branch
}
```

**If as expression:**

```zig
const result = if (a > b) a else b;
```

## While Loops

```zig
var i: u8 = 2;
while (i < 100) {
    i *= 2;
}
```

**With continue expression:**

```zig
var sum: u8 = 0;
var i: u8 = 1;
while (i <= 10) : (i += 1) {
    sum += i;
}
```

**With break:**

```zig
while (true) {
    if (done) break;
    // ...
}
```

**With continue:**

```zig
while (i <= 3) : (i += 1) {
    if (i == 2) continue;
    sum += i;
}
```

## For Loops

Iterate over arrays:

```zig
const chars = [_]u8{ 'a', 'b', 'c' };

// With index
for (chars, 0..) |char, index| {
    _ = char;
    _ = index;
}

// Just values
for (chars) |char| {
    _ = char;
}

// Just index
for (chars, 0..) |_, index| {
    _ = index;
}
```

## Switch

Works as statement AND expression:

```zig
const x: i8 = 10;
switch (x) {
    -1...1 => x = -x,
    10, 100 => x = @divExact(x, 10),
    else => {},
}
```

**As expression:**

```zig
const result = switch (value) {
    1 => "one",
    2 => "two",
    else => "other",
};
```

**Important:**
- All cases must be handled (exhaustive)
- No fall-through between cases
- Use `else` for catch-all
