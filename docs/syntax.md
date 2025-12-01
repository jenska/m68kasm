# m68kasm Assembly Syntax

This document describes the **source language** accepted by `m68kasm`. It follows classic Motorola 68k conventions with a compact, predictable grammar. Items marked *(planned)* are reserved for upcoming releases.

---

## 1. File & Lexical Conventions

- **Encoding:** UTF‑8 text files.
- **Line structure:** One logical statement per line (empty lines allowed).
- **Whitespace:** Spaces/tabs separate tokens. Indentation is optional.
- **Case:** Mnemonics and register names are **case‑insensitive**; labels are **case‑sensitive**.
- **Comments:** `;` starts a comment until end of line.
  ```asm
  MOVEQ #7, D3   ; set D3 to 7
  ```
- **Identifiers (labels/symbols):** `[A-Za-z_][A-Za-z0-9_]*`
- **Keywords reserved (non‑exhaustive):** all mnemonics, directive names, and register names `D0..D7 A0..A7 PC SR USP`.

---

## 2. Labels

- A label is an identifier at the **start of a line** followed by `:`
  ```asm
  start:  MOVEQ #0, D0
  loop:   ADDQ  #1, D0
          BRA.S  loop
  ```
- Labels are global; local labels (`1f/1b` style) are not yet supported.
- **Forward references** are allowed (two‑pass assembly).

---

## 3. Literals & Expressions

### 3.1 Integer literals
- Decimal: `1234`
- Hexadecimal: `0x7F` or `$7F`
- Binary: `%10100110`
- Octal: `0o755`
- **Sign:** A leading sign is parsed as a **unary operator** (`-1` = unary minus + decimal 1).

### 3.2 Character & string literals
- Char: `'A'` → 65 (8‑bit)
- String: `"Hello"` with C‑style escapes `" \ 
 	` (used by data directives).

### 3.3 Expressions
- Allowed operators:
  - **Unary:** `+` `-` `~`
  - **Binary:** `+` `-` `*` `/` `&` `|` `^` `<<` `>>`
  - Parentheses `( … )`
- Operands:
  - Integer literals, character literals
  - Previously defined **labels** (forward allowed)
- Expression results are truncated to the target field width (e.g., byte/word/long) with **range checks** where applicable.

---

## 4. Directives (Pseudo‑ops)

- `.org <expr>` — Set absolute location counter. The first `.org` captures the program origin; later forward jumps emit zero-fill bytes to preserve gaps, and backward jumps are rejected.
- `.byte <expr[, expr ...]>` — Emit bytes (low 8 bits).
- *(planned)* `.word <expr[, ...]>` — Emit 16‑bit words (big‑endian).
- *(planned)* `.long <expr[, ...]>` — Emit 32‑bit longs (big‑endian).
- *(planned)* `.align <n>` — Align location counter to multiple of `n` (power of two).

**Examples**
```asm
.org $1000
.byte 1, 2, 3, 'A'
; Planned:
; .word 0x1234, label
; .long $C0FFEE
; .align 16
```

---

## 5. Instructions

### 5.1 General form
```
<mnemonic><.size>  <operand>[, <operand>]
```
- `.size` is optional and instruction‑dependent: `.b` (byte), `.w` (word), `.l` (long).
- Operand count and allowed **EA (effective address) forms** depend on the selected instruction form.

### 5.2 Effective Address (EA) forms (68000)

| Form            | Syntax                    | Meaning                                |
|-----------------|---------------------------|----------------------------------------|
| Data register   | `D0` … `D7`              | Data register direct                    |
| Address reg.    | `A0` … `A7`              | Address register direct                 |
| Address ind.    | `(A<n>)`                 | Memory at address in `A<n>`             |
| Post‑increment  | `(A<n>)+`                | Use then increment `A<n>`               |
| Pre‑decrement   | `-(A<n>)`                | Decrement `A<n>` then use               |
| Displacement    | `(disp, A<n>)`           | 16‑bit displacement + `A<n>`            |
| Indexed         | `(disp, A<n>, Xn.s)`     | 8‑bit disp + index reg (`Dn`/`An`), size `.w`/`.l` |
| Abs. short      | `(<addr>).W`             | 16‑bit absolute                         |
| Abs. long       | `(<addr>).L`             | 32‑bit absolute                         |
| PC‑relative     | `(disp, PC)`             | 16‑bit displacement from `PC`           |
| PC‑indexed      | `(disp, PC, Xn.s)`       | 8‑bit disp + index from `PC`            |
| Immediate       | `#<expr>`                | Literal value                           |

**Notes**
- `disp` and `<addr>` are expressions.
- `Xn.s` is `D0..D7`/`A0..A7` with size suffix `.w` or `.l`, e.g. `(8, A0, D1.w)`.
- The assembler enforces per‑instruction EA legality via encoding tables.

### 5.3 Examples
```asm
; Load effective address of label into A0
LEA     label, A0

; Move immediate small constant into D3
MOVEQ   #7, D3

; Absolute addressing (long)
MOVE.L  (0x10000).L, D0

; Address with displacement and index
MOVE.W  (12, A2, D1.w), D3

; PC‑relative branch
loop:   ADDQ #1, D0
        BNE  loop
```

### 5.4 Sizes & ranges
- Byte/word/long suffixes are checked by the encoder.
- Branches (`BRA/Bcc/BSR`) verify displacement fits **short** (‑128..+127) or **word** (‑32768..+32767) ranges; out‑of‑range branches are errors.

---

## 6. Errors & Diagnostics

Typical errors reported by the assembler include:
- **Unknown mnemonic or directive**
- **Illegal addressing mode** for the chosen instruction form
- **Out‑of‑range** immediate, displacement, or branch offset
- **Undefined symbol** at end of pass 2
- **Syntax error** (unexpected token, missing comma/paren)
- **Misaligned** `.org`/data (where relevant)

Errors are reported with **file:line:column** and a concise message.

---

## 7. Minimal Grammar (EBNF)

A practical subset of the grammar is given below; the full language is a superset by table‑driven forms.

```ebnf
source      = { line } ;
line        = [ label ] ( directive | instruction ) [ comment ] EOL
            | [ comment ] EOL ;

label       = ident ":" ;

directive   = "." ident [ arguments ] ;
arguments   = argument { "," argument } ;
argument    = expr | string ;

instruction = mnemonic [ size ] [ operands ] ;
operands    = operand { "," operand } ;

size        = "." ( "b" | "w" | "l" ) ;

operand     = ea | immediate ;

immediate   = "#" expr ;

ea          = data_reg
            | addr_reg
            | "(" "A" digit ")"                     (* address indirect *)
            | "(" "A" digit ")" "+"                 (* post-increment   *)
            | "-" "(" "A" digit ")"                 (* pre-decrement    *)
            | "(" expr "," "A" digit ")"            (* displacement     *)
            | "(" expr "," "A" digit "," index ")"  (* indexed          *)
            | "(" expr ")" "." "W"                  (* absolute short   *)
            | "(" expr ")" "." "L"                  (* absolute long    *)
            | "(" expr "," "PC" ")"                 (* PC-relative      *)
            | "(" expr "," "PC" "," index ")" ;     (* PC-indexed       *)

index       = ( "D" | "A" ) digit "." ( "w" | "l" ) ;

data_reg    = "D" digit ;
addr_reg    = "A" digit ;

mnemonic    = ident ;
ident       = letter { letter | digit | "_" } ;

expr        = term { ( "+" | "-" | "|" | "^" | "<<" | ">>" | "&" ) term } ;
term        = factor { ( "*" | "/" ) factor } ;
factor      = [ "+" | "-" | "~" ] ( number | char | ident | "(" expr ")" ) ;

number      = dec | hex | bin | oct ;
dec         = digit { digit } ;
hex         = ( "0x" | "$" ) hex_digit { hex_digit } ;
bin         = "%" bin_digit { bin_digit } ;
oct         = "0o" oct_digit { oct_digit } ;

char        = "'" printable "'" ;
string      = """ { printable | escape } """ ;

escape      = "\" ( "\" | """ | "n" | "t" ) ;

digit       = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
hex_digit   = digit | "a" | "b" | "c" | "d" | "e" | "f"
                    | "A" | "B" | "C" | "D" | "E" | "F" ;
bin_digit   = "0" | "1" ;
oct_digit   = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" ;

letter      = "A"…"Z" | "a"…"z" | "_" ;

comment     = ";" { printable } ;
EOL         = "\n" ;
printable   = ? any printable character except EOL, according to UTF-8 ? ;
```

> **Note:** The encoder tables, not the grammar, ultimately decide which instruction + EA combinations are legal.

---

## 8. Compatibility

- Motorola/vasm‑style syntax is targeted. GAS (AT&T) syntax is not supported.
- The assembler currently focuses on 68000 core addressing modes; later models may relax/extend certain forms.

---

## 9. Changelog

- **v0.1:** initial syntax; directives `.org`, `.byte`; instructions `MOVEQ`, `LEA`, `BRA`.
- **Planned:** `.word`, `.long`, `.align`, branch family `Bcc/BSR`, arithmetic/logic basics.
