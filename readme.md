# m68kasm

[![CI](https://github.com/jenska/m68kasm/actions/workflows/ci.yml/badge.svg)](../../actions)
![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-informational)

A compact, **table-driven Motorola 68k assembler** written in Go, targeting the
full 68000–68060 family plus CPU32, an optional FPU, and a PMMU.

**Current version:** v1.4.0

The goal of this project is to provide a clean, maintainable, and easily
extensible assembler for the 68k family — focusing on clarity, modularity,
and full control over binary output. It is particularly well suited for
educational use, embedded projects, and retro computing enthusiasts who
prefer a minimal toolchain.

---

## 🚀 Features

- Two-pass macro assembler with deterministic binary output
- Targets the full 68000/68008/68010/68012/68020/68030/68040/68060 family and
  CPU32, plus an optional FPU (68881/68882 or integrated) and PMMU
  (68851/68030+), selected via `--cpu`/`--fpu`/`--mmu` — see the "CPU &
  Coprocessor Support" section below
- Include paths, pseudo ops, pre-defined symbols, and rich expressions
- Local numeric labels (`1f`/`1b`) with validated forward/backward resolution
- Embeddable directly into Go programs via a public API
- Optional source listings to pair machine code with source lines
- Output formats: flat binary, Motorola S-record (S0/S3/S7), and ELF32 (m68k)
  with a single load segment plus standard section/symbol tables
- Pseudo-ops: `.org`, `.byte`, `.word`, `.long`, `.align`, `.even`, `.text`,
  `.data`, `.bss`, `.section`, `.macro`/`.endmacro`, `DC.B`/`DC.W`/`DC.L`
- Table-driven instruction encoding (`InstrDef`/`FormDef`/`EmitStep`), so
  adding an instruction is a data change, not new control flow

---

## 🖥️ CPU & Coprocessor Support

| Flag | Unlocks |
| --- | --- |
| `--cpu 68010`/`68012` | `MOVEC`, `MOVES`, `RTD`, `BKPT` |
| `--cpu 68020` | 32-bit branches, memory-indirect addressing, scale factors, bit-field ops, `CAS`/`CAS2`, `CHK2`/`CMP2`, `EXTB.L`, `TRAPcc`, `PACK`/`UNPK`, `DIVSL`/`DIVUL`, `CALLM`/`RTM` (68020-only) |
| `--cpu cpu32` | The 68010 tier plus the 68020-era subset Motorola backported, plus `BGND` and `TBLS`/`TBLU` |
| `--cpu 68030` | Everything above except `CALLM`/`RTM` |
| `--cpu 68040`/`68060` | `MOVE16`, cache control (`CINV`/`CPUSH`); `68060` also emits a non-fatal warning for the handful of forms it trap-emulates |
| `--fpu` | `FMOVE`/`FADD`/etc., the `F`-condition branch/set/trap family, `FMOVECR`, `FSAVE`/`FRESTORE`, `FMOVEM` |
| `--fpu-full` | The transcendental function set (`FSIN`/`FCOS`/`FLOGN`/etc.) — a real discrete 68881/68882, not just an integrated FPU |
| `--mmu` | `PMOVE` (every PMMU register), `PFLUSHA`/`PFLUSH`, `PLOADR`/`PLOADW`, `PTESTR`/`PTESTW`, the `P`-condition branch/set/trap family |

This is a summary, not the full picture — several caveats, scope cuts, and a
couple of real encoding bugs found and fixed along the way are documented in
detail in [`docs/design/cpu-family-support.md`](docs/design/cpu-family-support.md),
which also tracks exactly what's still open (some `PFLUSH`/`PLOAD`/`PTEST`
variants, `FSINCOS`, the FPU's `FGETEXP`/`FGETMAN`/`FSCALE`/`FMOD`/`FREM`
math extensions, and a few other narrow, deliberately deferred corners).
The default
target with no flags is a bare 68000, so every 68000-only example in this
README and in [`docs/syntax.md`](docs/syntax.md) keeps working unchanged.

---

## 🛠️ Installation

```bash
go install github.com/jenska/m68kasm/cmd/m68kasm@latest
```

Or build locally:

```bash
git clone https://github.com/jenska/m68kasm.git
cd m68kasm
go build -o m68kasm ./cmd/m68kasm
```

---

## ⚙️ Usage

```bash
m68kasm [options] <source-files>
```

**Options**
| Option | Description |
|---------|--------------|
| `-i <file>` | Input assembly file |
| `-o <file>` | Write binary output (default: `a.out`) |
| `--format <bin|srec|elf>` | Select output format (binary, Motorola S-record, or ELF32) |
| `--cpu <name>` | Target CPU: `68000` (default), `68008`, `68010`, `68012`, `cpu32`, `68020`, `68030`, `68040`, `68060` |
| `--fpu` | Enable FPU instructions |
| `--fpu-full` | Enable the FPU transcendental function set (implies `--fpu`) |
| `--mmu` | Enable PMMU instructions |
| `-I <path>` | Add include search path |
| `-D name=val` | Define symbol |
| `--list <file>` | Generate a source listing (use `-` for stdout) |
| `--version` | Print assembler version and exit |

**Example:**
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

### Programmatic use (Go API)

```go
package main

import "github.com/jenska/m68kasm"

func main() {
        bin, listing, err := m68kasm.AssembleStringWithListing(".byte 0x12\nMOVEQ #1,D0\n")
        _ = listing // per-line PCs and bytes
        _ = err

        srec, _ := m68kasm.AssembleStringSRecord(".org 0x1000\n.byte 0x12,0x34\n")
        elf, _ := m68kasm.AssembleStringELF(".org 0x2000\n.byte 0x12\n")
        _ = bin
        _ = srec
        _ = elf
}
```

Additional helpers assemble from `[]byte`, `io.Reader`, or file paths, with
or without listings, and can append into an existing destination buffer.
Errors implement `m68kasm.Error`, carrying source line/column and (when
available) the original source line with a caret marker.

---

## 📁 Project Structure

```text
cmd/m68kasm/              # Command-line frontend
internal/asm/             # Assembler pipeline (lexer, parser, evaluation, encoding)
internal/asm/instructions # Declarative instruction tables and helpers
docs/                     # Reference material: syntax, grammar, design history, opcode tables
```

- [`docs/syntax.md`](docs/syntax.md) — accepted assembly syntax and directives
- [`docs/design/cpu-family-support.md`](docs/design/cpu-family-support.md) — full CPU/FPU/PMMU design history, scope decisions, and open items

---

## 💡 Contributing

1. Fork the repository.
2. Create a feature branch.
3. Add or update tests for any change.
4. Run `go fmt ./... && go vet ./... && go test ./...` before submitting.
5. Submit a pull request — CI must pass.

---

## 📄 License

Released under the [MIT License](LICENSE). You are free to use, modify, and
distribute the project with attribution.

---

## ❤️ Acknowledgments

- Motorola M68000 Programmer's Reference Manual (3rd Ed.)
- Easy68k and vasm project maintainers for inspiration on encoding tables
- GNU binutils' GAS m68k backend, used throughout to cross-check encodings
