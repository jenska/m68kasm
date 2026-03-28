# m68kasm

[![CI](https://github.com/jenska/m68kasm/actions/workflows/ci.yml/badge.svg)](../../actions)
![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-informational)

A compact, **table-driven Motorola 68000 assembler** written in Go.

**Current version:** v1.3.0

The goal of this project is to provide a clean, maintainable, and easily extensible assembler for the 68k family — focusing on clarity, modularity, and full control over binary output. It is particularly well suited for educational use, embedded projects, and retro computing enthusiasts who prefer a minimal toolchain.

---

## 🚀 Features (as of v2.0.1)

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

---

## ⚠️ Known Limitations

- **CPU Generation:** Strictly targets the **68000** instruction set. Extensions for 68010, 68020+, or FPU coprocessors are not currently supported.
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

You can install the CLI directly from GitHub using Go 1.22+:

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

## 🧭 Roadmap Overview

| Milestone | Description |
|------------|--------------|
| **v0.2** | Expand core instruction set (`MOVE`, `ADD`, `SUB`, `CMP`) |
| **v0.3** | Implement Bcc/BSR and pseudo-ops `.word`, `.long`, `.align` |
| **v0.4** | Introduce listing and S-record output |
| **v0.5** | Add ELF format and richer symbol handling |
| **v1.2.0** | Full assembler with macros, richer expressions, improved diagnostics, ELF section/symbol metadata, and documented pseudo-ops |
| **v1.2.1 (current)** | Adds source-level `.text`/`.data`/`.bss`/`.section` support with ELF-aware section and symbol placement |

---

## 🔧 Recent Improvements (v1.2.1)

### ELF Section Directives

- Added source-level `.text`, `.data`, `.bss`, and `.section` pseudo-ops
- Propagated section metadata into ELF section headers and symbol table entries
- Enforced the current forward-only section layout: `.text` -> `.data` -> `.bss`
- Kept `.bss` zero-initialized and emitted as `SHT_NOBITS` in ELF output

## 🔧 Previous Improvements (v1.2.0)

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

## 🔭 Next up (post-v1.2.x)

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
