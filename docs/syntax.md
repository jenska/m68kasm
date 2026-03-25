# m68kasm Assembly Syntax

This document describes the source language accepted by `m68kasm` today.
It focuses on the syntax the current parser and encoder actually implement.

## 1. Lexical Conventions

- Encoding: UTF-8 text files.
- Line structure: one logical statement per line.
- Whitespace: spaces and tabs separate tokens; indentation is optional.
- Case:
  - Mnemonics, directives, register names, and size suffixes are case-insensitive.
  - Labels and user-defined symbols are case-sensitive.
- Comments: `;` starts a comment that runs to end of line.

```asm
MOVEQ #7, D3   ; set D3 to 7
```

## 2. Labels And Symbols

- Global labels use `name:`.
- Local numeric labels use `1:` style definitions and `1f` / `1b` references.
- Forward references are supported through the assembler's two-pass parse.
- Constants can be defined with either `NAME = expr` or `NAME .equ expr`.

```asm
START:
COUNT = 4
LIMIT .equ $1234

BRA 1f
NOP
1:
BRA 1b
```

## 3. Literals And Expressions

### Integer Literals

- Decimal: `1234`
- Hexadecimal: `0x7F` or `$7F`
- Binary: `%10100110`
- Octal: `@377`
- Character literal: `'A'`

### String Literals

The lexer accepts double-quoted strings with escapes such as `\n`, `\r`, `\t`,
`\\`, `\"`, and `\0`. Current pseudo-ops operate on numeric expressions rather
than string operands, so string literals are mainly reserved for future syntax
extensions.

### Expressions

Expressions support labels, constants, character literals, parentheses, and
the following operators:

- Unary: `+`, `-`, `~`, `!`
- Multiplicative: `*`, `/`, `%`
- Additive: `+`, `-`
- Shifts: `<<`, `>>`
- Comparisons: `<`, `>`, `<=`, `>=`, `==`, `!=`
- Bitwise: `&`, `^`, `|`
- Logical: `&&`, `||`

Results are truncated to the destination field width where appropriate, with
range validation for directives and instruction fields that require it.

## 4. Pseudo-Ops

All currently implemented pseudo-ops are documented below.

### `.org <expr>`

Sets the location counter.

- The first `.org` also establishes the program origin.
- Later forward-only `.org` changes emit zero-filled padding.
- Backward `.org` changes are rejected.

```asm
.org $1000
.byte 1
.org $1010
.byte 2
```

### `.byte <expr[, expr ...]>`

Emits one byte per expression.

- Values are truncated to the low 8 bits.
- Multiple expressions are comma-separated.

```asm
.byte 1, $FF, 'A', 2+3
```

### `.word <expr[, expr ...]>`

Emits 16-bit big-endian words.

- Accepted range: `-0x8000` to `0xFFFF`

```asm
.word $1234, LABEL-START
```

### `.long <expr[, expr ...]>`

Emits 32-bit big-endian long words.

- Accepted range: `-0x80000000` to `0xFFFFFFFF`

```asm
.long $11223344, -1
```

### `.align <n[, fill]>`

Pads output until the location counter is aligned to a multiple of `n`.

- `n` must be at least `1`
- `fill` is optional and defaults to `0`

```asm
.align 4
.align 8, $CC
```

### `.even`

Pads by one zero byte when needed so the location counter becomes even.

```asm
.byte 1
.even
```

### `.text`, `.data`, `.bss`

Switches the current ELF section classification for subsequent labels and bytes.

- Sections are forward-only: `.text` -> `.data` -> `.bss`
- Binary and S-record output remain flat and keep the original byte order
- `.bss` is zero-initialized only; use zero-valued data directives or padding/alignment there

```asm
.text
start:
MOVEQ #1, D0

.data
value:
.word $1234

.bss
scratch:
.byte 0, 0, 0, 0
```

### `.section <name>`

Named form of the same section switch. Supported names are `.text`, `.data`, and `.bss`
(with or without the leading dot, or as a quoted string).

```asm
.section ".data"
table:
.word 1, 2, 3
```

### `.macro name [param[, param ...]]`

Begins a macro definition.

- Parameters are simple identifier substitutions.
- Macro bodies are expanded inline during parsing.
- Nested macro use is supported.
- Expansion depth is limited to prevent runaway recursion.

Definitions end with `.endmacro`.

```asm
.macro PAIR a, b
.byte a, b
.endmacro

PAIR 1, 2
```

### `.endmacro`

Terminates the current macro definition.

### `DC.B`, `DC.W`, `DC.L`

`m68kasm` also supports `DC` forms as aliases for the basic data directives:

- `DC.B` -> `.byte`
- `DC.W` -> `.word`
- `DC.L` -> `.long`

```asm
DC.B 1, 2, 3
DC.W $1234
DC.L $11223344
```

## 5. Instruction Form

General form:

```text
<mnemonic><.size> [operand[, operand]]
```

- Size suffixes are instruction-dependent: `.b`, `.w`, `.l`
- Some instructions also accept `.s` as a short-byte synonym where the parser
  maps it to byte-sized branch encoding
- Operand legality is validated by the instruction table and EA validators

## 6. Effective Address Forms

Supported 68000-style forms:

| Form | Syntax | Meaning |
|---|---|---|
| Data register | `D0` .. `D7` | Data register direct |
| Address register | `A0` .. `A7`, `SP`, `SSP` | Address register direct |
| Indirect | `(A0)` | Address register indirect |
| Post-increment | `(A0)+` | Use then increment |
| Pre-decrement | `-(A0)` | Decrement then use |
| Displacement | `(disp,A0)` or `disp(A0)` | 16-bit displacement |
| Indexed | `(disp,A0,D1.W)` | 8-bit brief displacement plus index |
| PC-relative | `(disp,PC)` | 16-bit PC displacement |
| PC-indexed | `(disp,PC,D1.W)` | 8-bit brief displacement plus index |
| Absolute short | `($1234).W` or `$1234.W` | 16-bit absolute |
| Absolute long | `($123456).L` or `$123456.L` | 32-bit absolute |
| Immediate | `#expr` | Immediate operand |
| Special registers | `SR`, `CCR`, `USP` | Instruction-specific special operands |

Notes:

- Indexed forms accept `Dn` or `An` index registers with `.W` or `.L`.
- Index scale factors `*1`, `*2`, `*4`, and `*8` are parsed.
- Register lists for `MOVEM` support `/` or commas and ascending ranges such as `D0-D3/A6`.

## 7. Diagnostics

Parse and assembly failures are reported with source location context.

- Errors include line numbers.
- When column information is available, errors include a caret.
- The public API exposes these as `m68kasm.Error`.

Example shape:

```text
line 1, col 1: unknown mnemonic
    FOOBAR D0, D1
    ^
```

## 8. Practical Examples

```asm
.org $1000
COUNT = 3

.macro BYTEPAIR a, b
.byte a, b
.endmacro

START:
BYTEPAIR 1, 2
DC.W $1234
.align 4, $CC
MOVEQ #COUNT, D0
BRA 1f
.even
1:
```

## 9. Notes

- The assembler targets the Motorola 68000 instruction set.
- The parser accepts Motorola-style syntax, not GAS/AT&T syntax.
- ELF output is executable-oriented: one flat load segment plus `.text`/`.data`/`.bss` metadata, not relocatable object generation.
- Section directives are intentionally lightweight and currently support only forward-only `.text` -> `.data` -> `.bss` layout.
