# m68kasm

[![CI](https://github.com/jenska/m68kasm/actions/workflows/ci.yml/badge.svg)](../../actions)
![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-informational)

A compact, **table-driven Motorola 68000 assembler** written in Go.

**Current version:** v1.4.0

The goal of this project is to provide a clean, maintainable, and easily extensible assembler for the 68k family — focusing on clarity, modularity, and full control over binary output. It is particularly well suited for educational use, embedded projects, and retro computing enthusiasts who prefer a minimal toolchain.

---

## 🚀 Features

- Two-pass macro assembler with deterministic binary output
- Supports all mnemonics of 68000 CPU
- Include paths, pseudo ops, pre-defined symbols and rich expressions
- Local numeric labels (e.g., `1f`/`1b`) with validated forward/backward resolution
- Simple and fast command-line tool with optimized performance
- Embeddable directly into Go programs via a public API
- Optional source listings to pair machine code with source lines
- Output formats: flat binary, Motorola S-record (S0/S3/S7), and ELF32 (m68k) with a single load segment plus standard section/symbol tables
- Documented pseudo-ops including `.org`, `.byte`, `.word`, `.long`, `.align`, `.even`, `.text`, `.data`, `.bss`, `.section`, `.macro`/`.endmacro`, and `DC.B`/`DC.W`/`DC.L`
- Table-driven instruction encoding (based on `InstDef`, `FormDef`, and `EmitStep` structures)
- Clear modular design in Go (`lexer`, `parser`, `expr`, `instructions`, `encode`, `assemble`)
- Performance optimizations:
  - Map-based effective address (EA) validation for O(1) lookups
  - Consolidated validation helpers to reduce code duplication
  - Priority-based instruction registration for correct opcode pattern matching
  - Minimized string operations in hot paths
- Correct PC-relative displacement calculations for `d16(PC)` and `d8(PC,Xn)` addressing modes
- Proper branch displacement handling for word-sized branches and DBcc instructions
- Support for `$` as current program counter in expressions
- Support for `.w` and `.l` suffixes on labels in PC-relative expressions
- CPU targeting scaffold (`--cpu` flag, `ParseOptions.Target`) gating CPU-specific syntax; see Known Limitations below for what's implemented so far
- MC68010/MC68012 instructions `MOVEC`, `MOVES`, `RTD`, and `BKPT` (`--cpu 68010` or later)
- 32-bit branch displacements `BRA.L`/`BSR.L`/`Bcc.L` (`--cpu 68020` or later)
- 68020+ memory-indirect addressing: `([bd,An],Xn,od)`, `([bd,An,Xn],od)`, and PC-relative equivalents (`--cpu 68020` or later)
- 68020 instructions `EXTB.L`, `CHK2`, `CMP2`, and `TRAPcc` (all 16 conditions, bare/`.W`/`.L` forms) (`--cpu 68020` or later)
- A first slice of FPU instructions — `FMOVE`, `FADD`, `FSUB`, `FMUL`, `FDIV`, `FCMP`, `FABS`, `FNEG`, `FSQRT`, `FTST`, `FNOP` — with integer and floating memory/register operands (`--fpu`)
- The FPU's 32-condition branch/set/trap family — `FBcc`/`FDBcc`/`FScc`/`FTRAPcc` — plus `FMOVECR` (ROM constant load) and `FSAVE`/`FRESTORE` (state-frame save/restore) (`--fpu`)
- `FMOVEM` — FPn register-list save/restore, static (`FP0-FP3/FP5`) and dynamic (a `Dn` holding the list at runtime), both directions (`--fpu`)
- `MOVE16` (16-byte cache-line-aligned block move) and cache control (`CINVA`/`CINVL`/`CINVP`, `CPUSHA`/`CPUSHL`/`CPUSHP`) (`--cpu 68040` or `68060`)
- A non-fatal warning (not an error) when a 68060 target uses one of the handful of 68020-era forms real 68060 silicon traps and software-emulates instead of executing natively (`--cpu 68060`)
- `PACK`/`UNPK`, `CAS`, `CALLM`, and `RTM` (`--cpu 68020` or later; `CALLM`/`RTM` are 68020-exclusive, dropped again on 68030+)
- The 68020 bit-field instructions (`BFTST`/`BFCHG`/`BFCLR`/`BFSET`/`BFEXTU`/`BFEXTS`/`BFFFO`/`BFINS`) with full `{offset:width}` syntax, including register-specified offset/width (`--cpu 68020` or later)
- `DIVSL.L`/`DIVUL.L <ea>,Dr:Dq` (or `<ea>,Dq` for the 32-bit shorthand) — the 64-bit-dividend divide forms (`--cpu 68020` or `cpu32`)
- `CAS2.W`/`CAS2.L Dc1:Dc2,Du1:Du2,(Rn1):(Rn2)` — dual-location compare-and-swap (`--cpu 68020` or later; not on `cpu32`)
- `PMOVE.L <ea>,TC`/`PMOVE.L TC,<ea>` and `PFLUSHA` — a first, deliberately narrow slice of 68030/68851 PMMU support (`--mmu`)

---

## ⚠️ Known Limitations

- **CPU Generation:** A `--cpu`/`ParseOptions.Target` selector exists (68000, 68008, 68010, 68012, CPU32, 68020, 68030, 68040, 68060). It gates CPU-specific *addressing-mode* syntax (68020+ scale factors on indexed addressing), the full 68010/68012 instruction set (`MOVEC`, `MOVES`, `RTD`, `BKPT`), 32-bit branch displacements (`BRA.L`/`BSR.L`/`Bcc.L`), and 68020+ memory-indirect addressing (`([bd,An],Xn,od)` pre-indexed and `([bd,An,Xn],od)` post-indexed, plus PC-relative equivalents — `--cpu 68020` or later). Within memory-indirect addressing, a base register is always required (register suppression is not supported), `bd` must always be written explicitly (no `([An],...)` shorthand for a zero base displacement), and a `bd`/`od` that references a symbol must carry an explicit `.w`/`.l` size (a symbol's value can change between the assembler's two passes, so its encoded length can never be auto-detected from it). The 68020 EA upgrade does *not* extend the classic brief indexed form `(d,An,Xn)`/`(d,PC,Xn)` to accept displacements beyond a signed byte on any CPU tier — that form is unconditionally limited to 8 bits by every real 68k CPU, and now errors clearly instead of silently truncating as it previously did. `--cpu 68020` also unlocks `EXTB.L`, `CHK2`/`CMP2`, and all 16 conditions of `TRAPcc` (bare, `.W`, and `.L` forms). A separate `--fpu` flag (independent of `--cpu`, since FPU presence is an attached-coprocessor question, not a CPU-tier one) unlocks a first slice of the 68881/68882/integrated-FPU instruction set: `FMOVE`, `FADD`, `FSUB`, `FMUL`, `FDIV`, `FCMP`, `FABS`, `FNEG`, `FSQRT`, `FTST`, and `FNOP`, with `.b`/`.w`/`.l`/`.s`/`.d`/`.x` sizes — but **not** floating-point immediate literals (`#1.5`; the lexer only tokenizes integers, so only integer immediates work), the transcendental functions, packed BCD (`.p`), or the `FPCR`/`FPSR`/`FPIAR` control registers. `--fpu` also unlocks the FPU's own 32-condition branch/set/trap family — `FBcc` (`.W`/`.L`, e.g. `FBEQ.W`/`FBEQ.L` — unlike integer `Bcc`, there is no 8-bit-inline displacement form, since the condition already occupies the opcode's low bits), `FDBcc` (word displacement only), `FScc` (data-alterable destination, like integer `Scc`), and `FTRAPcc` (bare/`.W`/`.L`, like integer `TRAPcc`) — plus `FMOVECR #<0-127>,FPn` (load one of the FPU's built-in ROM constants by numeric index only; GAS's named aliases like `fp_pi` are not implemented) and `FSAVE`/`FRESTORE` (save/restore the FPU's internal state frame to/from memory — `FSAVE` accepts `-(An)` but not `(An)+`, `FRESTORE` the reverse, matching real hardware's descending-write/ascending-read convention). A real encoding bug was found and fixed while researching this: every FPU instruction's first word needs coprocessor ID 1 in bits 11-9 (`0x0200`), which the original `--fpu` slice omitted (see `fpuWord1Base`'s doc comment in `internal/asm/instructions/cpu020_fpu.go`) — already-generated FADD/FMOVE/etc. output from before this fix will differ. `--fpu` also unlocks `FMOVEM` — FPn register-list save/restore, e.g. `FMOVEM.X FP0-FP7,-(SP)` to spill all eight FPU data registers — with both static (`FP0-FP3/FP5`) and dynamic (`FMOVEM.X D0,-(SP)`, the list computed at runtime from `D0`'s low 8 bits) forms in both directions; `-(An)` is store-only and `(An)+` load-only, exactly like integer `MOVEM`. `FMOVEM` for the `FPCR`/`FPSR`/`FPIAR` control registers is a separate, still-unimplemented form (a different bit width/position and its own EA rules) — only the FP0-FP7 data-register list is supported. Researching this also surfaced a second, unrelated pre-existing bug in integer `MOVEM` itself: it incorrectly accepted `-(An)` as a load source (`MOVEM -(A0),D0-D7`), which real hardware does not support — fixed separately (see `movemLoadEA`'s doc comment in `internal/asm/instructions/validate.go`). `--cpu 68020` also unlocks `PACK`/`UNPK` (BCD pack/unpack, register and predecrement forms), `CAS` (single-location compare-and-swap, all sizes), and `CALLM`/`RTM` — the last two are gated so they are rejected again on `--cpu 68030` or later, since real 68030+ silicon dropped them (confirmed against GNU binutils' own architecture-bit header, which tags them with a narrower bit than every other 68020 addition). `--cpu 68020` also unlocks the bit-field instructions — `BFTST`, `BFCHG`, `BFCLR`, `BFSET`, `BFEXTU`, `BFEXTS`, `BFFFO`, `BFINS` — with `{offset:width}` syntax on any data-alterable or readable EA (including `Dn`, but not `(An)+`/`-(An)`); offset and width can each independently be a literal (`0`-`31` for offset, `1`-`32` for width) or a `Dn` register, e.g. `BFEXTU (A0){D1:8},D2`. `DIVSL.L`/`DIVUL.L <ea>,Dr:Dq` (the 64-bit-dividend divide forms — kept on `--cpu cpu32` too, like `Bcc.L`/`CHK2`/`EXTB`/`TRAPcc`) also work, with a `<ea>,Dq` shorthand (no colon) for the plain 32-bit-quotient case. `CAS2.W`/`CAS2.L Dc1:Dc2,Du1:Du2,(Rn1):(Rn2)` (dual-location compare-and-swap) also works — unlike `CAS`, it is **not** available on `--cpu cpu32`, matching GAS's own architecture tagging. A separate `--mmu` flag (independent of `--cpu`, like `--fpu`, since PMMU presence is also an attached-coprocessor question) unlocks a deliberately narrow first slice of the 68851/68030-integrated PMMU instruction set: `PMOVE.L <ea>,TC` and `PMOVE.L TC,<ea>` (load/store the Translation Control register — the single register most essential to basic address-translation setup), plus `PFLUSHA` (flush the entire ATC). The full PMMU family turned out to be one of the largest corners of the 68k ISA — dozens of `Pcc` condition variants mirroring `Bcc`/`Scc`/`DBcc` for MMU status flags, six different `PFLUSH` operand shapes, and `PMOVE` forms for half a dozen other MMU registers (`CRP`, `SRP`, `TT0`, `TT1`, `MMUSR`, …), each with its own selector-code scheme in GAS — and is deliberately out of scope here; only `TC` and `PFLUSHA` are implemented, `TBLS`/`TBLU` and the rest of PMMU are still not implemented. `--cpu 68040` (also granted to `--cpu 68060`, since 68060 is a strict superset here) unlocks `MOVE16` (16-byte cache-line-aligned block move — `(An)+,(An)+`, `(An)+`/`ABS.L` in either order, or plain `(An)`/`ABS.L` in either order) and the cache-control instructions `CINVA`/`CINVL`/`CINVP` and `CPUSHA`/`CPUSHL`/`CPUSHP` (invalidate/push a cache line, page, or the whole cache, with an `NC`/`DC`/`IC`/`BC` cache selector). `--cpu 68060` additionally makes the assembler print a non-fatal `warning:` (to stderr, not a build-breaking error) whenever the source uses one of the handful of 68020-era forms real 68060 silicon doesn't execute natively — it traps and the kernel emulates it in software instead, a real performance cliff worth knowing about even though the instruction assembles and runs correctly: `CAS2`, `CHK2`/`CMP2`, `MOVEP` unconditionally, plus the 68020 bit-field instructions **only** when the offset or width is register-specified (`{Dn:Dn}`, not a literal `{0:8}`) and `DIVSL`/`DIVUL` **only** for the wide 64-bit `Dr:Dq` dividend form (the plain 32-bit `Dq` shorthand is native). See [`docs/design/cpu-family-support.md`](docs/design/cpu-family-support.md) for the roadmap.

`--cpu cpu32` targets the CPU32 embedded core (68330/68331/68332/68340/…), which is *not* simply "68010 or 68020": it gets `MOVEC`/`MOVES`/`RTD`/`BKPT` (the full 68010 tier) plus the specific 68020-era additions it backported — `BRA.L`/`BSR.L`/`Bcc.L`, scale factors on indexed addressing, `CHK2`/`CMP2`, `EXTB.L`, and `TRAPcc` — but it does **not** get 68020 memory-indirect addressing, bit-field ops, `CAS`/`CAS2`, or the coprocessor interface (FPU included). It also has one instruction of its own that no other tier has: `BGND` (enter background debug mode). This split was verified against GNU binutils' own `cpu32` tagging rather than assumed; `TBLS`/`TBLU`/`TBLSN`/`TBLUN` (CPU32's table-lookup-and-interpolate instructions) are still open — their operand syntax needs more research than could be responsibly rushed.
- **Linker Support:** ELF output now includes standard `.text`, `.data`, `.bss`, `.symtab`, `.strtab`, and `.shstrtab` metadata, and source can switch between `.text`, `.data`, and `.bss`, but the assembler still emits a single executable-style image rather than relocatable objects.
- **Section Layout:** Section directives are intentionally minimal and forward-only: `.text` -> `.data` -> `.bss`. `.bss` content must remain zero-initialized.
- **Optimizations:** The assembler prioritizes deterministic output over optimization. It does not automatically relax instructions (e.g., `JMP` to `BRA`) or substitute shorter instruction forms unless explicitly handled by the instruction selection logic.

---

## 🧩 Project Goals

The assembler implements core principles of assembler construction:
- **Lexical and syntactic clarity:** each stage is well-separated and testable.  
- **Declarative instruction definitions:** encoding logic defined via compact data tables.  
- **Binary precision:** full control over emitted bytes without hidden abstractions.  
- **Go idioms:** idiomatic use of Go’s type system, slices, and maps for maintainability.

---

## 🛠️ Installation

You can install the CLI directly from GitHub using Go 1.27+:

```bash
go install github.com/jenska/m68kasm/cmd/m68kasm@latest
```

Or build locally:

```bash
git clone https://github.com/jenska/m68kasm.git
cd m68kasm
go build ./cmd/m68kasm
```

---

## ⚙️ Usage

```bash
m68kasm [options] <source-files>
```

**Options**  
| Option | Description |
|---------|--------------|
| `-o <file>` | Write binary output (default: `a.out`) |
| `--format <bin|srec|elf>` | Select output format (binary, Motorola S-record, or ELF32) |
| `--cpu <name>` | Target CPU: `68000` (default), `68008`, `68010`, `68012`, `cpu32`, `68020`, `68030`, `68040`, `68060` |
| `--fpu` | Enable FPU instructions (68881/68882 or an integrated FPU) |
| `--mmu` | Enable PMMU instructions (68851 or an integrated PMMU) |
| `-I <path>` | Add include search path |
| `-D name=val` | Define symbol |
| `--list <file>` | Generate a source listing (use `-` for stdout) |
| `--version` | Print assembler version and exit |
| `-v` | Verbose logging |

**Example:**
```bash
m68kasm -o hello.bin tests/e2e/testdata/hello.s
hexdump -C hello.bin
```

### Programmatic use (Go API)

The assembler can also be embedded directly into Go programs via the public API
provided by the root module:

```go
package main

import "github.com/jenska/m68kasm"

func main() {
        // Assemble source that comes from a string and keep listing metadata.
        bin, listing, err := m68kasm.AssembleStringWithListing(".byte 0x12\nMOVEQ #1,D0\n")
        _ = listing // listing contains per-line PCs and bytes
        _ = err

        // Emit Motorola S-record text directly from the same source.
        srec, _ := m68kasm.AssembleStringSRecord(".org 0x1000\n.byte 0x12,0x34\n")
        // Produce an ELF image with section headers and label symbols.
        elf, _ := m68kasm.AssembleStringELF(".org 0x2000\n.byte 0x12\n")
        _ = bin
        _ = srec
        _ = elf
}
```

Additional helpers support assembling from `[]byte`, `io.Reader`, or file paths
with or without listings, and can append results into an existing destination
buffer.

Errors returned by the public API include source location context and, when
available, the original source line with a caret marker. Type-assert to
`m68kasm.Error` when you want structured access to line and column data.

### Quick start: assemble and run the sample program

If you want to see the assembler in action immediately, clone the repository and build the CLI, then assemble the bundled
`hello.s` example. The following commands will produce a binary and print it as hexadecimal bytes:

```bash
git clone https://github.com/jenska/m68kasm.git
cd m68kasm
go build ./cmd/m68kasm
./m68kasm -o hello.bin tests/e2e/testdata/hello.s
hexdump -C hello.bin
```

The `tests/e2e/testdata/hello.s` file demonstrates the currently implemented instructions (`MOVEQ`, `LEA`, and `BRA`) and is
exercised by the automated end-to-end tests.

---

## 📁 Project Structure

```text
cmd/m68kasm/              # Command-line frontend
internal/asm/             # Assembler pipeline (lexer, parser, evaluation, encoding)
internal/asm/instructions # Declarative instruction tables and helpers
tests/e2e/                # End-to-end tests for the CLI
tests/e2e/testdata/       # Sample assembly sources and expected binaries used by the tests
docs/                     # Reference material including grammar and opcode tables
```

### Further documentation
- [`docs/syntax.md`](docs/syntax.md) documents the accepted assembly syntax and directives.
- [`docs/grammar.ebnf`](docs/grammar.ebnf) provides the EBNF grammar used by the parser.
- [`docs/M68kOpcodes.pdf`](docs/M68kOpcodes.pdf) is a handy opcode reference while extending the instruction tables.

### Pseudo-op summary
- `.org <expr>` sets the location counter and program origin.
- `.byte`, `.word`, and `.long` emit big-endian data items.
- `.align <n[, fill]>` aligns the location counter with optional fill bytes.
- `.even` aligns the location counter to an even address.
- `.macro` / `.endmacro` define parameterized macros.
- `DC.B`, `DC.W`, and `DC.L` are aliases for the corresponding data directives.

---

## ⚡ Continuous Integration (CI)

A ready-to-use **GitHub Actions** workflow (`.github/workflows/ci.yml`) is provided.  
It performs:
- Module verification (`go mod verify`)  
- Vetting (`go vet`)  
- Unit and E2E tests (`go test ./...`)  
- CLI build validation

---

## 🧰 Development Guidelines

To keep the repository clean and consistent, please follow these steps:

```bash
# Format source code
go fmt ./...

# Lint and vet
go vet ./...

# Run all tests
go test ./... -v
```

---

## 💡 Contributing

Contributions are welcome!  
If you want to add new instructions, improve encoding tables, or extend pseudo-ops:

1. Fork the repository.  
2. Create a feature branch (`feature/add-cmp-instruction`).  
3. Add or update relevant test cases.  
4. Submit a pull request.

Make sure the CI passes before submitting.

---

### Code Quality & Performance

**Instruction Pattern Matching**
- Implemented priority-based instruction registration system to fix opcode pattern conflicts (MULU/MULS vs AND/OR)
- Replaced filename-based ordering with explicit Priority field in `InstrDef`

**Validation & Code Consolidation**
- Created reusable EA (effective address) validation helpers with map-based lookups:
  - `isMemoryAlterable()`, `isDataAlterable()`, `isReadableEA()`, `isReadableDataEA()`, `isMovemLoadEA()`
- Consolidated redundant validation patterns across 15+ validation functions
- Removed ~70 lines of duplicate switch statements in validation logic
- Eliminated 12 instances of repeated operand swapping with `swapSrcDstIfDstNone()` helper

**Performance Optimizations**
- Replaced switch statements with O(1) map lookups in hot paths (`isPCRelativeKind()`, `validateControlEA()`)
- Removed unnecessary string operations (replaced `strings.HasPrefix()` with direct byte comparison)
- Simplified `reverse16()` bit manipulation for MOVEM encoding
- Improved immediate range validation performance

**Testing & Benchmarks**
- All 200+ unit tests pass
- All e2e tests pass
- All benchmarks pass with improved performance metrics
- Fixed validation benchmark tests

---

## 🔭 Next up (post-v1.4.x)

- **Diagnostics and listing upgrades:** Enhance listings with symbol resolutions, relocation notes, and per-instruction metadata, while improving error spans and suggestion text for a friendlier workflow.
- **Additional output conveniences:** Support formats like Intel HEX or extended S-record variants, and explore a “linkable object” mode with separated sections/symbols to integrate with broader toolchains.
- **Output Optimizations:** Implement instruction relaxation (e.g., `JMP` → `BRA.S`) and optimize internal form matching to reduce assembly time.


## 🧠 Design Philosophy

The assembler aims to balance **authentic 68k semantics** with **modern Go idioms**.  
By representing instruction encoding as data rather than code, it reduces complexity and simplifies maintenance.

Each instruction is described declaratively, for example:

```go
InstDef{
    Mnemonic: "MOVEQ",
    Forms: []FormDef{
        {Mask: 0x7000, Size: Byte, Src: Imm8, Dst: Dn},
    },
}
```

This structure allows new instructions to be added without modifying the assembler’s logic — only its data tables.

---

## 📄 License

Released under the [MIT License](LICENSE).  
You are free to use, modify, and distribute the project with attribution.

---

## 🧱 Example Output

For the included `hello.s` example, assembling yields:

```
$ hexdump -C hello.bin
00000000  76 07 41 e9 00 10 43 fb  22 08 60 00 f4 aa bb cc  |v.A...C.".`.....|
00000010
```
---

## ❤️ Acknowledgments

Special thanks to the open-source 68k community for documentation and references, including:
- Motorola M68000 Programmer’s Reference Manual (3rd Ed.)  
- Easy68k and vasm project maintainers for inspiration on encoding tables.  
- The Go community for encouraging clean, modular software design.

---
**Author:** Jens Kaiser  
**Repository:** [github.com/jenska/m68kasm](https://github.com/jenska/m68kasm)  
**Status:** Active – under continuous development  
