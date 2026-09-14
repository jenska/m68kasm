# m68kasm User Manual

A practical guide to installing, running, and embedding `m68kasm` — a
table-driven Motorola 68000-family assembler targeting the full
68000–68060 range plus CPU32, an optional FPU, and a PMMU.

This manual covers the CLI and the Go API end to end. For the full
grammar of the assembly language itself (every directive, operand form,
and CPU/FPU/PMMU syntax extension), see [`syntax.md`](syntax.md) — this
manual points to it rather than repeating it. For the history of *why*
things are encoded the way they are, see
[`design/cpu-family-support.md`](design/cpu-family-support.md).

## Table of Contents

1. [Installation](#1-installation)
2. [Quick Start](#2-quick-start)
3. [The CLI](#3-the-cli)
4. [Output Formats](#4-output-formats)
5. [Targeting a CPU, FPU, and PMMU](#5-targeting-a-cpu-fpu-and-pmmu)
6. [The Assembly Language, Briefly](#6-the-assembly-language-briefly)
7. [The Go API](#7-the-go-api)
8. [Diagnostics and Warnings](#8-diagnostics-and-warnings)
9. [Known Limitations](#9-known-limitations)
10. [Further Reading](#10-further-reading)

---

## 1. Installation

Requires Go 1.27 or later.

```bash
go install github.com/jenska/m68kasm/cmd/m68kasm@latest
```

Or build from a local checkout:

```bash
git clone https://github.com/jenska/m68kasm.git
cd m68kasm
go build -o m68kasm ./cmd/m68kasm
```

Check it worked:

```bash
./m68kasm --version
```

## 2. Quick Start

```bash
cat > hello.s <<'EOF'
        MOVEQ   #1,D0
        LEA     $2000,A0
        BRA     start
start:
EOF
m68kasm -i hello.s -o hello.bin
hexdump -C hello.bin
```

`m68kasm` always needs `-i <file>`; there is no positional-argument form.
Output defaults to `a.out` if `-o` is omitted.

## 3. The CLI

```text
m68kasm -i input.s [-o out.bin] [--list out.lst] [--format bin|srec|elf]
        [--cpu 68000|68010|68020|68030|68040|68060|cpu32]
        [--fpu] [--fpu-full] [--mmu] [-I path] [-D name[=val]]
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `-i <file>` | *(required)* | Input assembly source file. |
| `-o <file>` | `out.bin` | Output file path. |
| `--format <bin\|srec\|elf>` | `bin` | Output container — see [§4](#4-output-formats). |
| `--cpu <name>` | `68000` | Target CPU tier — see [§5](#5-targeting-a-cpu-fpu-and-pmmu). |
| `--fpu` | off | Enable FPU instructions. |
| `--fpu-full` | off | Enable the FPU transcendental/math-extension set (implies `--fpu`). |
| `--mmu` | off | Enable PMMU instructions. |
| `-I <path>` | *(none)* | Add a search directory for resolving `-i` (repeatable). |
| `-D name[=val]` | *(none)* | Predefine a symbol, optionally with a value (repeatable; value parsed with `strconv.ParseUint`, base auto-detected — `0x`-prefixed hex, `0`-prefixed octal, `0b`-prefixed binary, or decimal; note this is Go's own prefix rules, not the assembly language's `$`/`%`/`@` literal prefixes). Bare `-D NAME` defines it as `1`. |
| `--list <file>` | *(none)* | Write a source listing; `-` writes to stdout. |
| `--version` | | Print the assembler version and exit. |

Notes:

- `-I` doesn't make `-i` search *every* directory blindly: if the path
  given to `-i` is absolute, or exists relative to the current directory,
  it's used as-is; `-I` directories are only consulted as a fallback for
  a bare filename that isn't found either way.
- There is no `.include` source directive — `-I` only affects how the
  single `-i` input file is *located*, not how you split source across
  multiple files from within a `.s` file.
- `--list` and `--format srec` both need per-line listing metadata
  internally, so requesting either (even just `--list -` to preview on
  stdout) costs a little more than a bare binary assemble — irrelevant
  for normal file sizes.
- 68060 targets can print non-fatal `warning:` lines to stderr after a
  successful assemble — see [§8](#8-diagnostics-and-warnings).

### Example: listing output

```bash
m68kasm -i hello.s -o hello.bin --list -
```

```text
Line  Address    Bytes                     Source
----- -------- -------------------------------- ------------------------------
    1  0x00000000  70 01                                    MOVEQ   #1,D0
    2  0x00000002  41 F9 00 00 20 00                        LEA     $2000,A0
    3  0x00000008  60 00                                    BRA     start
```

(A label-only line like `start:` emits no bytes, so it has no listing
entry of its own.)

### Example: predefined symbols and include paths

```bash
m68kasm -i main.s -I lib/ -I ../shared -D DEBUG -D BUFSIZE=0x400 -o main.bin
```

`main.s` can then reference `DEBUG` and `BUFSIZE` as ordinary symbols
without defining them itself, and a bare `INCLUDE_ME.s`-style filename
passed to some future multi-file workflow would resolve against `lib/`
or `../shared` if not found next to the working directory.

## 4. Output Formats

Selected with `--format` (CLI) or by calling the matching `Assemble*`
family in the Go API ([§7](#7-the-go-api)).

- **`bin`** (default) — a flat binary: just the assembled bytes, no
  header, starting at whatever `.org` set (or address 0 if none).
- **`srec`** — Motorola S-record text (S0 header record, S3 32-bit-address
  data records, S7 termination record). The S0 header embeds the
  assembler version string.
- **`elf`** — an ELF32 executable image for the `m68k` architecture: one
  load segment covering the assembled bytes, plus standard section
  (`.text`/`.data`/`.bss`, per the source's own `.section` directives)
  and symbol tables for ELF-aware tooling (debuggers, `objdump`, etc.).
  This targets *loading and inspection*, not relocatable object-file
  generation — there's no relocation table, and it isn't meant to be
  fed to a linker.

`.org`'s value becomes both the load address and, for ELF, the entry
point.

## 5. Targeting a CPU, FPU, and PMMU

The default target is a bare 68000 with no coprocessors — every
68000-only example in this manual and in `syntax.md` keeps working
unchanged regardless of what you target.

| Flag | Unlocks |
| --- | --- |
| `--cpu 68010`/`68012` | `MOVEC`, `MOVES`, `RTD`, `BKPT` |
| `--cpu 68020` | 32-bit branches, memory-indirect addressing, scale factors, bit-field ops, `CAS`/`CAS2`, `CHK2`/`CMP2`, `EXTB.L`, `TRAPcc`, `PACK`/`UNPK`, `DIVSL`/`DIVUL`, `CALLM`/`RTM` (68020-only) |
| `--cpu cpu32` | The 68010 tier plus the 68020-era subset Motorola backported, plus `BGND` and `TBLS`/`TBLU` |
| `--cpu 68030` | Everything above except `CALLM`/`RTM` |
| `--cpu 68040`/`68060` | `MOVE16`, cache control (`CINV`/`CPUSH`); `68060` also emits non-fatal warnings for the handful of forms it trap-emulates |
| `--fpu` | `FMOVE`/`FADD`/etc. (with floating-point immediate literals, e.g. `#3.14`, and packed BCD, `.p`), the `F`-condition branch/set/trap family, `FMOVECR`, `FSAVE`/`FRESTORE`, `FMOVEM` |
| `--fpu-full` | The transcendental function set (`FSIN`/`FCOS`/`FSINCOS`/`FLOGN`/etc.) and math extensions (`FGETEXP`/`FGETMAN`/`FSCALE`/`FMOD`/`FREM`) — a real discrete 68881/68882, not just an integrated FPU |
| `--mmu` | `PMOVE` (every PMMU register), the full `PFLUSHA`/`PFLUSH`/`PFLUSHS`/`PFLUSHR`/`PLOADR`/`PLOADW`/`PTESTR`/`PTESTW` family (plus the 68040's own simplified single-word forms on `--cpu 68040`/`68060`), `PSAVE`/`PRESTORE`, `PMOVEFD`, the `P`-condition branch/set/trap family |

`--cpu`, `--fpu`, `--mmu` are independent axes, not a strict hierarchy:
a bare 68020 with an external 68881 and a 68040 with its FPU subset
integrated on-die both just need `--fpu` set, regardless of `--cpu`.
`--fpu-full` additionally asserts a *real discrete* 68881/68882, since a
68040/68060's own integrated FPU only trap-emulates the transcendental
set rather than executing it natively.

### From the Go API

The same selection is a `Target` value:

```go
package main

import "github.com/jenska/m68kasm"

func main() {
	opts := m68kasm.ParseOptions{
		Target: m68kasm.Target{
			CPU:      m68kasm.CPU68020,
			Features: m68kasm.FeatFPU,
		},
	}
	bin, err := m68kasm.AssembleStringWithOptions("FADD.X (A0),FP0\n", opts)
	_ = bin
	_ = err
}
```

- `CPU`: `CPU68000` (zero value/default), `CPU68010`, `CPU32`,
  `CPU68020`, `CPU68030`, `CPU68040`, `CPU68060`.
- `Features`: a bitmask — OR together `FeatFPU`, `FeatFPUFull`, and
  `FeatPMMU` as needed.
- Have a CPU name as a string instead (e.g. from your own `--cpu`-style
  flag)? Use `m68kasm.ParseCPUKind(name)`, which accepts the same names
  the CLI does (case-insensitive; `68008`/`68012` resolve to the
  `68000`/`68010` tier respectively) and reports `ok == false` for an
  unrecognized name.

This is the complete picture for what each flag *unlocks*; caveats,
scope cuts, and the (small number of) real encoding bugs found and
fixed while building this are in
[`design/cpu-family-support.md`](design/cpu-family-support.md).

## 6. The Assembly Language, Briefly

The full grammar — literals, expressions, every pseudo-op, every
addressing-mode form, and the CPU/FPU/PMMU syntax extensions — is
[`syntax.md`](syntax.md). This section is a five-minute cheat sheet, not
a replacement for it.

```asm
; comments start with ';'
START:                        ; global label
COUNT = 4                     ; constant (or: COUNT .equ 4)

.org $1000                    ; set the location counter / load address
.text
BEGIN:
        MOVEQ   #COUNT,D0      ; #imm, Dn
        LEA     TABLE,A0        ; label reference
        MOVE.L  (A0)+,D1          ; post-increment
        BRA     1f                 ; forward local-label reference
        NOP
1:      BRA     1b                  ; backward local-label reference

.data
TABLE:  DC.L    1, 2, 3, 4            ; DC.L is an alias for .long
.align  4

.macro  PAIR a, b
        .byte   a, b
.endmacro
        PAIR    1, 2
```

Key points worth knowing up front:

- Motorola-style syntax (`MOVE.L D0,D1`), not GNU `as`/AT&T syntax.
- Mnemonics, directives, and register names are case-insensitive;
  labels and symbols are case-sensitive.
- Two-pass assembly, so forward references (to labels defined later in
  the file) just work.
- Local numeric labels (`1:` / `1f` / `1b`) are independent of global
  labels and can be reused throughout a file.
- Integer literals: decimal (`123`), hex (`$7F` or `0x7F`), binary
  (`%1010`), octal (`@17`), character (`'A'`).
- No `.include` directive — see [§3](#3-the-cli)'s note on `-I`.

## 7. The Go API

Import path: `github.com/jenska/m68kasm`.

The package is built around one core operation — parse source, apply a
`Target`, encode — exposed through a large but mechanically predictable
family of `Assemble*` functions, plus a few higher-level helpers on top.

### 7.1 The naming pattern

Every assembly entry point follows the same recipe, so once you
recognize the pieces you can predict the exact function name you want
rather than needing to look each one up:

```text
Assemble [Bytes|String|File] [<format>] [WithOptions] [Into] [WithListing]
```

- **Input source** — `Assemble(io.Reader, ...)` is the base form;
  `AssembleBytes*` takes `[]byte`, `AssembleString*` takes `string`,
  `AssembleFile*` takes a file `path string`.
- **Output format** — no suffix means flat binary; `*ELF` returns an
  ELF32 image; `*SRecord` returns Motorola S-records (see
  [§4](#4-output-formats) for what each contains).
- **`WithOptions`** — takes a `ParseOptions` (CPU/FPU/PMMU target,
  predefined symbols) as the last argument; omit it for the zero-value
  default (bare 68000, no coprocessors, no predefined symbols).
- **`Into`** — appends to a caller-supplied `dst []byte` instead of
  allocating a fresh slice (binary format only).
- **`WithListing`** — also returns `[]ListingEntry` (one entry per
  source line: line number, PC, and the bytes it assembled to), in
  addition to the assembled bytes.

So `AssembleStringELFWithOptions(src, opts)` is exactly what it says:
assemble a `string`, apply `opts`, return an ELF32 image. Every
combination that makes sense exists; browse `godoc` (or just the
package source — `assemble_api.go`) for the exhaustive list rather than
treating this manual as that list.

A few representative ones:

```go
bin, err := m68kasm.AssembleString(".byte 1,2,3\n")

bin, listing, err := m68kasm.AssembleStringWithListing(src)

elf, err := m68kasm.AssembleFileELFWithOptions("boot.s", opts)

srec, err := m68kasm.AssembleBytesSRecord(srcBytes)

out, err := m68kasm.AssembleStringInto(existingBuf, moreSrc) // append
```

### 7.2 Configuration: `ParseOptions`

```go
type ParseOptions struct {
	Symbols    map[string]uint32 // predefined symbol values
	InstrTable *instructions.Table // advanced: see note below
	Target     m68kasm.Target       // CPU tier + coprocessor features
}
```

`Symbols` is the programmatic equivalent of the CLI's `-D`: predefine
label values before assembling, without needing `.equ` lines in the
source itself.

```go
opts := m68kasm.ParseOptions{
	Symbols: map[string]uint32{"BUFSIZE": 0x400},
	Target:  m68kasm.Target{CPU: m68kasm.CPU68020},
}
```

`InstrTable` exists for callers embedding a customized instruction
table, but its type lives in an internal package this module doesn't
currently re-export — see [§9](#9-known-limitations). Leave it `nil`
(the default instruction set) unless you have your own fork wiring
something in here.

### 7.3 Detailed results: `AssembleStringDetailed` and friends

The plain `Assemble*` family returns just bytes (plus, with
`WithListing`, a line-by-line byte breakdown). When you want richer
metadata — resolved label addresses, per-instruction canonical text,
line-to-address mapping — use the `*Detailed*` family instead
(`AssembleDetailed`, `AssembleBytesDetailed`, `AssembleStringDetailed`,
`AssembleFileDetailed`, each with a `WithOptions` variant):

```go
result, err := m68kasm.AssembleStringDetailedWithOptions(src, opts)
if err != nil {
	// ...
}
result.Bytes         // []byte — the assembled program
result.Origin        // uint32 — the .org address
result.Labels        // map[string]uint32 — every global label's address
result.DefinedLabels // []DefinedLabel — labels in source-definition order
result.Instructions  // []InstructionMetadata — one entry per instruction

addr, ok := result.AddressOf("START")
addr, ok  = result.AddressForLine(12)
```

`InstructionMetadata` gives you, per assembled instruction: `Line`,
`PC`, `Bytes`, `Size` (byte count), `Words`, and `Canonical` (a
normalized text spelling of the instruction — handy for diffing
"what did this actually assemble to" against expectations in tests).

### 7.4 Single-instruction helpers

For tooling that works one instruction at a time (an interactive
REPL, a disassembler round-trip check, a test harness):

```go
res, err := m68kasm.AssembleInstructionString("MOVEQ #1,D0")
// res.Bytes, res.PC, res.EncodedSize, res.Words, res.Canonical

canon, err := m68kasm.CanonicalizeInstructionString("move.l d0,d1")
// canon == "MOVE.L D0,D1"
```

Both return an error if the input isn't exactly one instruction.

### 7.5 Building source programmatically: `ProgramBuilder`

A small fluent helper for generating source text without manual string
concatenation — handy for tests and codegen, not a general-purpose
macro system:

```go
src := m68kasm.NewProgramBuilder().
	Origin(0x1000).
	Label("START").
	Byte(1, 2, 3).
	Align(4, 0xCC).
	String() // the generated source text

result, err := m68kasm.NewProgramBuilder().
	Line("MOVEQ #1,D0").
	AssembleWithOptions(opts)
```

Other methods: `Line`, `Instruction` (alias for `Line`), `Text`, `Data`,
`BSS`, `Section`, `Word`, `Long`, `ByteExpr`/`WordExpr`/`LongExpr` (for
expressions rather than literal numbers), `Even`, and `VectorTable`
(emits a zero-filled `.long` exception-vector table up to the highest
supplied index).

### 7.6 Errors

```go
bin, err := m68kasm.AssembleString(src)
if err != nil {
	if asmErr, ok := err.(*m68kasm.Error); ok {
		fmt.Println(asmErr.Line, asmErr.Col, asmErr.Message())
	}
	fmt.Println(err) // full formatted message with source-line + caret, when available
}
```

`Error.Error()` returns the same human-readable, location-prefixed
message the CLI prints. `m68kasm.NormalizeError(err)` strips the
line/column prefix, returning just the underlying message — useful in
tests that want to assert on error *content* independent of exactly
which line/column it occurred at.

### 7.7 Version

```go
m68kasm.Version // e.g. "v1.4.0"
```

Used internally as the S-record header text; also handy for embedding
in your own tool's `--version` output if you're wrapping this package.

## 8. Diagnostics and Warnings

Parse and assembly failures carry source location context (line,
column when available, and — when the failing line's text is known — a
caret pointing at the exact column):

```text
line 1, col 1: unknown mnemonic
    FOOBAR D0, D1
    ^
```

On a `--cpu 68060` target, a handful of 68020-era instruction forms
(`CAS2`, `CHK2`/`CMP2`, `MOVEP`, dynamic-offset/width bit-field ops, and
`DIVSL`/`DIVUL`'s 64-bit `Dr:Dq` form) assemble successfully but only
because real 68060 silicon traps and software-emulates them rather than
executing them natively. The CLI prints these as non-fatal
`warning: ...` lines to stderr after a successful assemble; they are
not part of the returned bytes or an `Error`, since assembly genuinely
succeeded. As of this manual, that warning path is wired into the CLI
(`cmd/m68kasm`) via the internal package directly — see
[§9](#9-known-limitations) for the current state of surfacing it
through the public Go API.

## 9. Known Limitations

Honest gaps in the current public API surface, so you don't spend time
looking for something that isn't wired up yet:

- **68060 emulation warnings aren't reachable from the public API.**
  `internal/asm`'s `Program.Warnings` field carries them, and the CLI
  reads it directly (since `cmd/m68kasm` imports `internal/asm`), but
  the top-level `m68kasm` package's result types (`AssemblyResult`,
  plain `[]byte` returns) don't currently surface it. If you're
  embedding this assembler and specifically need those warnings
  programmatically, you'd need your own fork or a local patch adding a
  `Warnings` field to `AssemblyResult`.
- **`ParseOptions.InstrTable` isn't practically usable from outside this
  module.** Its type, `*instructions.Table`, comes from
  `internal/asm/instructions` — a Go `internal/` package, which by
  language rules can't be imported by code outside
  `github.com/jenska/m68kasm`. There's currently no re-exported way to
  obtain or construct a `Table` value from the public package, so this
  field is effectively dead for external callers; leave it `nil`.
- **No `.include` directive.** Splitting source across files from
  *within* a `.s` file isn't supported — only `-I`/multiple `-i`-style
  resolution of the single top-level input file (see [§3](#3-the-cli)).
- A few memory-indirect addressing-mode edge cases (base-register
  suppression, an implicit-zero-displacement bracket shorthand, and
  auto-upgrading a brief displacement to full format on overflow) are
  deliberately unimplemented — each for a specific technical reason,
  not for lack of time. See `syntax.md` §7.2 and
  `design/cpu-family-support.md`'s closing section for why.

## 10. Further Reading

- [`syntax.md`](syntax.md) — the complete assembly language grammar:
  every directive, literal form, addressing mode, and the CPU/FPU/PMMU
  syntax extensions.
- [`design/cpu-family-support.md`](design/cpu-family-support.md) — the
  full design history behind the CPU/FPU/PMMU work: what was built in
  what order, why specific encodings were chosen, real encoding bugs
  found and fixed along the way, and exactly what's still open.
- [`../README.md`](../README.md) — a shorter project overview, if this
  manual is more than you need right now.
