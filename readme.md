# m68kasm

[![CI](https://github.com/jenska/m68kasm/actions/workflows/ci.yml/badge.svg)](../../actions)
![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-informational)

A compact, **table-driven Motorola 68000 assembler** written in Go.

The goal of this project is to provide a clean, maintainable, and easily extensible assembler for the 68k family — focusing on clarity, modularity, and full control over binary output. It is particularly well suited for educational use, embedded projects, and retro computing enthusiasts who prefer a minimal toolchain.

---

## 🚀 Features (as of v0.1)

- Two-pass assembler with deterministic binary output  
- Table-driven instruction encoding (based on `InstDef`, `FormDef`, and `EmitStep` structures)
- Implemented instructions (selected):
  - Data movement and addressing: `MOVE`, `MOVEQ`, `MOVEM`, `MOVEP`, `LEA`, `PEA`
  - Control transfer: `JMP`, `JSR`
  - Arithmetic: `ADD`, `SUB`, `ADDQ`, `SUBQ`, `ADDX`, `SUBX`, `NEGX`, `ABCD`, `SBCD`, `MULS`, `MULU`, `DIVS`, `DIVU`
  - Logical and bit operations: `AND`, `OR`, `EOR`, `NOT`, `ORI`/`ANDI`/`EORI` to `CCR` and `SR`, `BSET`, `BCLR`, `BCHG`, `BTST`
  - Comparison and tests: `CMP`, `TST`
  - Stack framing: `LINK`, `UNLK`
  - Branching: `BRA`, `BSR`, all `Bcc` conditions, loop counters (`DBcc` family)
  - System and traps: `NOP`, `RESET`, `STOP`, `TRAP`, `TRAPV`, `RTS`, `RTE`
- Pseudo-operations: `.org`, `.byte`, `.word`, `.long`, `.align`
- Clear modular design in Go (`lexer`, `parser`, `expr`, `instructions`, `encode`, `assemble`)
- Simple and fast command-line tool
- Optional source listings to pair machine code with source lines

### Planned for Upcoming Releases
- Support all mnemonics 0f 68000 CPU
- Output formats: Motorola S-Record, ELF (for cross-tool compatibility)
- Automated tests and regression validation

---

## 🧩 Project Goals

The assembler is designed to demonstrate and implement core principles of assembler construction:
- **Lexical and syntactic clarity:** each stage is well-separated and testable.  
- **Declarative instruction definitions:** encoding logic defined via compact data tables.  
- **Binary precision:** full control over emitted bytes without hidden abstractions.  
- **Go idioms:** idiomatic use of Go’s type system, slices, and maps for maintainability.

---

## 🛠️ Installation

You can install the CLI directly from GitHub using Go:

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
| `-I <path>` | *(planned)* Add include search path |
| `-D name=val` | *(planned)* Define symbol |
| `--list <file>` | Generate a source listing (use `-` for stdout) |
| `-v` | Verbose logging |

**Example:**
```bash
m68kasm -o hello.bin tests/e2e/testdata/hello.s
hexdump -C hello.bin
```

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
```

---

## ⚡ Continuous Integration (CI)

A ready-to-use **GitHub Actions** workflow (`.github/workflows/ci.yml`) is provided.  
It performs:
- Module verification (`go mod verify`)  
- Vetting (`go vet`)  
- Unit and E2E tests (`go test ./...`)  
- CLI build validation

You can extend it with optional steps like `staticcheck` or release automation.

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

### Recommended Future Additions
- `Makefile` with targets for `build`, `test`, and `release`
- Static analysis integration (`staticcheck`)
- Example programs under `examples/`

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
| **v1.0** | Full assembler with macros, expressions, and rich error reporting |

---

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
