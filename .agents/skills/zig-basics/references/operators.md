# Operators

## Arithmetic Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `+` | Addition | `2 + 5 == 7` |
| `-` | Subtraction | `5 - 2 == 3` |
| `*` | Multiplication | `2 * 5 == 10` |
| `/` | Division | `10 / 5 == 2` |
| `%` | Remainder | `10 % 3 == 1` |

**Integer overflow** causes panic in safe modes. Use wrapping operators:

| Operator | Description | Example |
|----------|-------------|---------|
| `+%` | Wrapping addition | `@as(u32, 0xffffffff) +% 1 == 0` |
| `-%` | Wrapping subtraction | `@as(u8, 0) -% 1 == 255` |
| `*%` | Wrapping multiplication | `@as(u8, 200) *% 2 == 144` |

| Operator | Description |
|----------|-------------|
| `+\|` | Saturating addition |
| `-\|` | Saturating subtraction |
| `*\|` | Saturating multiplication |

## Bitwise Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `&` | Bitwise AND | `0b011 & 0b101 == 0b001` |
| `\|` | Bitwise OR | `0b010 \| 0b100 == 0b110` |
| `^` | Bitwise XOR | `0b011 ^ 0b101 == 0b110` |
| `~` | Bitwise NOT | `~@as(u8, 0b10101111) == 0b01010000` |

## Bit Shift Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `<<` | Shift left | `0b1 << 8 == 0b100000000` |
| `>>` | Shift right | `0b1010 >> 1 == 0b101` |
| `<<\|` | Saturating shift left | `@as(u8, 1) <<\| 8 == 255` |

## Logical Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `and` | Logical AND | `false and true == false` |
| `or` | Logical OR | `false or true == true` |
| `!` | Boolean NOT | `!false == true` |

**Note**: `and`/`or` short-circuit.

## Comparison Operators

| Operator | Description |
|----------|-------------|
| `==` | Equality |
| `!=` | Inequality |
| `<` | Less than |
| `>` | Greater than |
| `<=` | Less or equal |
| `>=` | Greater or equal |

## Optional Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `orelse` | Unwrap or default | `null orelse 1234 == 1234` |
| `.?` | Unwrap or unreachable | `value.?` |

## Error Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `catch` | Unwrap or default | `error_val catch 1234` |
| `catch \|err\|` | Capture error | `err_val catch \|e\| return e` |

## Pointer Operators

| Operator | Description |
|----------|-------------|
| `&` | Address of |
| `.*` | Dereference |

## Array Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `++` | Concatenation | `a ++ b` (compile-time) |
| `**` | Repetition | `"ab" ** 3 == "ababab"` |

## Precedence

Highest to lowest:

1. `x()` `x[]` `x.y` `x.*` `x.?`
2. `a!b` `x{}`
3. `!x` `-x` `-%x` `~x` `&x` `?x`
4. `*` `/` `%` `**` `*%` `*|`
5. `+` `-` `++` `+%` `-%` `+|` `-|`
6. `<<` `>>` `<<|`
7. `&`
8. `^` `|`
9. `orelse` `catch`
10. `==` `!=` `<` `>` `<=` `>=`
11. `and`
12. `or`
13. `=` `*=` `*%=` `*|=` `/=` `%=` `+=` `+%=` `+|=` `-=` `-%=` `-|=` `<<=` `<<|=` `>>=` `&=` `^=` `|=`

## Assignment

```zig
var x: i32 = 1;
x = 2;  // assign new value

// Compound assignment
x += 1;
x -= 2;
x *= 3;
```

## Type Coercion

Use `@as` for explicit conversion:

```zig
const x: u32 = @as(u32, 42);
const y: f64 = @as(f64, 3.14);
```
