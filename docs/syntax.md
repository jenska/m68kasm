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

The lexer has no floating-point literal syntax — only integers. FPU
instructions that take an immediate (e.g. `FADD.L #100,FP0`) therefore only
accept integer immediates; a floating-point literal such as `#1.5` is
rejected.

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
m.even
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

- Size suffixes are instruction-dependent: `.b`, `.w`, `.l` for the base
  integer ISA.
- Some instructions also accept `.s` as a short-byte synonym where the parser
  maps it to byte-sized branch encoding.
- FPU mnemonics (any mnemonic starting with `F`) use a separate size
  namespace: `.b`/`.w`/`.l` still select an integer conversion size, and
  `.s`/`.d`/`.x` select single/double/extended-precision floating formats —
  `.s` means "single-precision", not "short branch", when it follows an FPU
  mnemonic (§7.3).
- Operand legality is validated by the instruction table and EA validators.
- This assembler targets more than the base 68000 — see §7 for the CPU
  family, FPU, and PMMU instructions the `--cpu`/`--fpu`/`--mmu` flags
  unlock, and for size/displacement conventions (like `.W`/`.L` branch
  suffixes) that this codebase settled on itself rather than copying GNU
  `as`'s exact mnemonic spelling.

## 6. Effective Address Forms

Supported 68000-baseline forms (available on every CPU tier):

| Form | Syntax | Meaning |
| --- | --- | --- |
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
- Index scale factors `*1`, `*2`, `*4`, and `*8` are parsed on the brief
  indexed form (`(disp,A0,D1.W*2)`), but only assemble on `--cpu 68020` or
  later (or `cpu32`) — a bare 68000 target rejects a scale factor other than
  `*1`. The classic brief form's displacement stays limited to a signed byte
  on every CPU tier; it never auto-upgrades to a wider encoding.
- Register lists for `MOVEM` support `/` or commas and ascending ranges such
  as `D0-D3/A6`.

See §7 for every effective-address and register form the CPU-family,
FPU, and PMMU extensions add on top of this baseline (memory-indirect
addressing, `FPn`, coprocessor control registers, bit-field specifiers,
register pairs, and more).

## 7. CPU Family, Coprocessors, and Extended Syntax

`m68kasm` targets the full 68000-through-68060 family plus CPU32, an
attached or integrated FPU, and a PMMU, selected via CLI flags
(`--cpu <name>`, `--fpu`, `--mmu`) or the equivalent `Target` value through
the public API. The default target is a bare 68000 with no coprocessors,
so every example elsewhere in this document keeps working unchanged.
Everything in this section is additive syntax gated behind one of those
flags; see [`README.md`](../README.md) for the exact instruction lists and
scope caveats, and
[`docs/design/cpu-family-support.md`](design/cpu-family-support.md) for the
full milestone-by-milestone design history.

### 7.1 CPU tiers (`--cpu`)

`--cpu` accepts `68000` (default), `68008`, `68010`, `68012`, `cpu32`,
`68020`, `68030`, `68040`, and `68060`. Each tier is additive over the
previous one except `cpu32`, which sits in its own branch: it gets the full
68010 tier plus the specific 68020-era additions Motorola backported to it
(`BRA.L`/`BSR.L`/`Bcc.L`, scale factors, `CHK2`/`CMP2`, `EXTB.L`, `TRAPcc`),
but not 68020's memory-indirect addressing, bit-field instructions,
`CAS`/`CAS2`, or the coprocessor interface.

```asm
MOVEC VBR,A0          ; 68010+
BRA.L faraway          ; 68020+ or cpu32
CHK2.B (A0),D1          ; 68020+ or cpu32
MOVE.L (0,A0,D1.W*2),D2 ; 68020+ scale factor
CAS.B D0,D1,(A0)         ; 68020+
TBLS.B (A0),D0            ; cpu32 only
MOVE16 (A0)+,(A1)+         ; 68040+ (also 68060)
```

### 7.2 68020+ memory-indirect addressing

```text
([bd,An],Xn,od)     pre-indexed
([bd,An,Xn],od)      post-indexed
```

plus PC-relative equivalents (`([bd,PC],Xn,od)`, `([bd,PC,Xn],od)`). Notes:

- A base register is always required — register suppression is not
  supported.
- `bd` must always be written explicitly inside the brackets; there is no
  `([An],...)` shorthand for a zero base displacement.
- A `bd`/`od` that references a symbol must carry an explicit `.w`/`.l`
  size, since a symbol's value can change between the assembler's two
  passes and its encoded length can never be auto-detected from it.

```asm
MOVE.L ([0,A0],D1,0),D2
MOVE.L ([0,A0,D1],4),D2
LEA ([100.L,PC],A0,2.W),A1
```

### 7.3 FPU (`--fpu`)

`FPn` (`FP0`-`FP7`) is a new register class, usable as either operand of
`FMOVE`/`FADD`/`FSUB`/`FMUL`/`FDIV`/`FCMP`/`FABS`/`FNEG`/`FSQRT`/`FTST`. FPU
size suffixes are `.b`/`.w`/`.l` (integer conversion) and `.s`/`.d`/`.x`
(single/double/extended-precision) — see §5.

```asm
FADD.X (A0),FP0
FMOVE.L D0,FP0
FMOVE.X FP2,(A0)
FABS FP0                 ; single-operand shorthand: FP0 = |FP0|
```

A floating-point size (`.s`/`.d`/`.x`) also accepts an immediate — either
a bare float literal or a plain integer (auto-promoted):

```asm
FMOVE.X #3.14159,FP0     ; extended-precision literal
FADD.S #1.5e-3,FP0       ; exponent form
FADD.X #5,FP0            ; integer literal, promoted to float
```

A float literal is a single bare constant, not a general expression
(`#1.5+2.5` is not supported); an integer size (`.b`/`.w`/`.l`) still
requires an integer literal — a fractional one is rejected. Using a
float literal anywhere other than an FPU immediate (e.g. `MOVE.W
#1.5,D0`) is always an error.

The FPU's own 32-condition branch/set/trap family mirrors the integer ISA's
`Bcc`/`Scc`/`DBcc`/`TRAPcc`, just with `F`-prefixed mnemonics, 32 conditions
instead of 16, and this codebase's own `.W`/`.L` suffix convention (not GNU
`as`'s separately-spelled `fbeq`/`fbeql` mnemonics) for the branch
displacement size:

```asm
FBEQ.W target             ; word displacement
FBEQ.L target              ; 32-bit displacement — no 8-bit inline form exists
FDBEQ D0,target              ; word displacement only
FSEQ D0                        ; data-alterable destination, like Scc
FTRAPEQ                          ; bare, .W, or .L immediate forms
```

`FMOVECR #<0-127>,FPn` loads one of the FPU's built-in ROM constants by
numeric index (named aliases like `fp_pi` are not implemented).
`FSAVE`/`FRESTORE <ea>` save/restore the FPU's internal state frame
(`FSAVE` accepts `-(An)`, not `(An)+`; `FRESTORE` the reverse).

`FMOVEM` covers two distinct register-list operands:

```asm
FMOVEM.X FP0-FP7,-(SP)          ; static FPn list — range/slash syntax like MOVEM
FMOVEM.X D0,-(SP)                ; dynamic list: the mask is read from D0 at runtime
FMOVEM FPIAR,D0                    ; a single named control register may target Dn/An too
FMOVEM FPCR/FPSR,(A0)                ; a genuine multi-register combination needs memory
```

`FPCR`, `FPSR`, and `FPIAR` are also valid as plain register operands
elsewhere `--fpu` accepts a control register.

`.p` (packed BCD) works as a source format anywhere any other size does
(`FMOVE.P (A0),FP0`, `FADD.P (A0),FP0`, …). As a *store* destination it
needs an extra `{#k}` (static) or `{Dn}` (dynamic) k-factor suffix,
specifying the number of mantissa digits to generate (`-64` to `17`) —
mandatory, unlike every other size, since packed has no implicit
default precision:

```asm
FMOVE.P FP0,(A0){#7}    ; static k-factor: 7 digits after the point
FMOVE.P FP0,(A0){D1}    ; dynamic: the k-factor is read from D1
```

There is no packed BCD immediate literal syntax (`#<value>` written as
packed digits) — only a memory operand already holding packed BCD bytes
is supported as a source.

### 7.4 PMMU (`--mmu`)

`PMOVE` moves one of several PMMU registers to or from memory, `Dn`, or
`An` (restrictions vary by register — see the README):

```asm
PMOVE.L (A0),TC             ; enable/configure address translation
PMOVE.L CRP,(A0)              ; root pointer descriptor (memory only, 64-bit)
PMOVE.L (A0),TT0                ; transparent translation window
PMOVE.B D0,CAL                    ; access-level register, byte-sized
PMOVE.W (A0),MMUSR                  ; or PMOVE.W (A0),PSR — same register, two names
```

`PFLUSHA` flushes the entire address translation cache and takes no
operands. The PMMU's own 16-condition branch/set/trap family mirrors §7.3's
FPU family and the integer ISA, with its own `PBcc`/`PDBcc`/`PScc`/`PTRAPcc`
mnemonics, conditions, and (for `PBcc`) `.W`/`.L` suffix convention.

On `--cpu 68040`/`68060`, `PFLUSHA`/`PFLUSH`/`PTESTR`/`PTESTW` (plus two
68040-only mnemonics, `PFLUSHAN` and `PFLUSHN`) switch to the 68040's own
simplified, single-word encoding — `PFLUSH`/`PFLUSHN`/`PTESTR`/`PTESTW`
then take a single address register, written as either `(An)` or bare
`An`:

```asm
PFLUSH (A0)              ; or "PFLUSH A0" — identical encoding
PTESTR A2
```

`PTESTR`/`PTESTW`'s single-word form is 68040-only — the 68060 dropped
it, so on `--cpu 68060` only the general 68030/68851-style form
(`PTESTR SFC,(A0),#3`) remains available; the other single-word forms
here work on both `68040` and `68060`.

A cache selector (`NC`, `DC`, `IC`, or `BC`) is `CINV`/`CPUSH`'s first
operand on `--cpu 68040` or later:

```asm
CINVA BC
CINVL DC,(A0)
CPUSHP IC,(A3)
```

### 7.5 Register pairs and multi-part operands

A handful of 68020+ instructions take a colon-joined register pair or a
bit-field specifier rather than a single register or plain `<ea>`:

```asm
DIVSL.L (A0),D0:D1            ; Dr:Dq — 64-bit dividend; "D0" alone means Dq only
CAS2.L D0:D1,D2:D3,(A0):(A1)    ; compare/update pairs, plus a memory-pointer pair
BFEXTU (A0){0:8},D2               ; bit-field {offset:width} — either half can be
BFINS D3,(A0){D1:D2}                 ; a literal or a Dn register
TBLS.B D0:D1,D2                        ; cpu32 table-lookup register form
FSINCOS FP0,FP1:FP2                      ; FPc:FPs dual result — cosine, then sine;
                                           ; no bare-single-register shorthand (--fpu-full)
```

Bit-field offset/width syntax is available on any data-alterable or
readable EA (including `Dn`, but not `(An)+`/`-(An)`) for
`BFTST`/`BFCHG`/`BFCLR`/`BFSET`/`BFEXTU`/`BFEXTS`/`BFFFO`/`BFINS`; a
literal offset is `0`-`31` and a literal width is `1`-`32`.

## 8. Diagnostics

Parse and assembly failures are reported with source location context.

- Errors include line numbers.
- When column information is available, errors include a caret.
- The public API exposes these as `m68kasm.Error`.
- `--cpu 68060` also produces non-fatal `warning:` messages (to stderr, not
  `m68kasm.Error`) for a handful of 68020-era forms real 68060 silicon
  traps and software-emulates instead of executing natively — see the
  README for the exact list. Assembly still succeeds; this is purely
  informational.

Example shape:

```text
line 1, col 1: unknown mnemonic
    FOOBAR D0, D1
    ^
```

## 9. Practical Examples

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

## 10. Notes

- The assembler's default target is the Motorola 68000 instruction set;
  `--cpu`/`--fpu`/`--mmu` extend it up through 68060, CPU32, an attached or
  integrated FPU, and a PMMU (§7). Every 68000-only example elsewhere in
  this document keeps working unchanged on every tier.
- The parser accepts Motorola-style syntax, not GAS/AT&T syntax. Where this
  codebase's own convention diverges from GNU binutils' `as` (e.g. `.W`/`.L`
  suffixes on branch instructions instead of separately-spelled mnemonics),
  that choice is called out explicitly in §7 and in the README.
- ELF output is executable-oriented: one flat load segment plus `.text`/`.data`/`.bss` metadata, not relocatable object generation.
- Section directives are intentionally lightweight and currently support only forward-only `.text` -> `.data` -> `.bss` layout.
