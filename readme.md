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
  (68851/68030+), selected via `--cpu`/`--fpu`/`--mmu`
- Embeddable directly into Go programs via a public API
- Output formats: flat binary, Motorola S-record, and ELF32 (m68k)
- Table-driven instruction encoding (`InstrDef`/`FormDef`/`EmitStep`), so
  adding an instruction is a data change, not new control flow

See [`docs/user-manual.md`](docs/user-manual.md) for the full CLI reference,
what each `--cpu`/`--fpu`/`--mmu` flag unlocks, output-format details, and
the complete Go API.

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

## ⚙️ Quick Start

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

For every CLI flag, output-format details, and the Go API (`go get
github.com/jenska/m68kasm`), see [`docs/user-manual.md`](docs/user-manual.md).

---

## 📁 Project Structure

```text
cmd/m68kasm/              # Command-line frontend
internal/asm/             # Assembler pipeline (lexer, parser, evaluation, encoding)
internal/asm/instructions # Declarative instruction tables and helpers
docs/                     # Reference material: syntax, grammar, design history, opcode tables
```

- [`docs/user-manual.md`](docs/user-manual.md) — full CLI and Go API guide, start to finish
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
