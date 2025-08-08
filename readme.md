# m68kasm – 68k Assembler (Go)

A modular Motorola 68k assembler written in Go, designed for extensibility, clarity, and correctness.  
Supports a substantial subset of the 68k instruction set and pseudo-ops, with robust label, expression, and error handling.


### 🚀 Core Features

- **68k Instruction Parsing:**  
  Parses a broad range of Motorola 68k instructions, including registers, immediate, addressing modes, and register lists (e.g., for `MOVEM`).

- **Pseudo-Op Support:**  
  - `ORG`, `ALIGN`, `EVEN`
  - Data constants: `DC.B`, `DC.W`, `DC.L` (with string, char, numeric, and label/expr operands)
  - Space reservation: `DS.B`, `DS.W`, `DS.L`

- **Label Support:**  
  - Define labels anywhere (e.g., `start:` or `.loop:`)
  - Use labels as operands in data and instructions
  - Forward and backward label references resolved

- **Expression Evaluation:**  
  - Arithmetic expressions in operands:  
    - Supports `+`, `-`, `*`, `/`, parentheses, decimal, `$hex`, and label names  
    - Example: `dc.w ($100 + 2) - (loop * 2)`

- **Two-Pass Assembly:**  
  - First pass: collect labels and addresses
  - Second pass: evaluate expressions, resolve labels, and generate output

- **Generic Pseudo-Directive Handling:**  
  - Table-driven design for pseudo-ops
  - Per-directive operand count checks (arity enforced)
  - Easy to add new pseudo-ops or custom handlers

- **Robust Operand Parsing:**  
  - Handles quoted strings, chars, parenthesis, register lists, and comma-separated operands
  - Modular, extensible design for adding more addressing modes

- **Error Checking:**  
  - Operand count/type checks for pseudo-ops
  - Duplicate/undefined label errors
  - Expression and syntax error reporting (with line numbers)

- **Output:**  
  - Assembles to in-memory binary
  - (Easy to adapt to write to file or ROM image)

---

### 🛠️ Extensible Architecture

- **Directory Structure:**  
  - `internal/parser/` – Parsing and lexer logic  
  - `internal/codegen/` – Pseudo-op and output logic  
  - `main.go` – Main program (two-pass, label table, output)

- **Easy to Add Features:**  
  - New pseudo-ops: Edit a table, provide a handler  
  - More instructions: Extend parser/encoder modules  
  - Operand/label features: Expression evaluation is pluggable

---

## Example: Assembly Source

```assembly
start:      org $1000
            dc.b "Hello", 0
mainloop:   dc.w start + 4
            align 4
alignlbl:   ds.b ($20 + 4) / 2
            even
loop:       move.b #1, d0
            bra loop
```

---

## Usage

```sh
go run main.go input.asm
```

### Output

- By default, outputs parsing and address info.
- Adapts easily to write binary to a file.

---

## Limitations & Roadmap

- **Instruction encoding:** Only a subset of 68k instructions are fully encoded
- **No macros/includes/conditional assembly** (yet)
- **No listing file, symbol export, or debug info**
- **No automated tests** (PRs welcome!)
- **No CLI options (output file, etc)**

---

## Contributing

PRs, issues, and suggestions are very welcome!  
See code comments and structure for extension points.

---

## License

MIT

## Project Structure

```
m68kasm/
├── cmd/
│   └── m68kasm/          # Main CLI entrypoint (main.go)
├── internal/
│   ├── assembler/        # Core assembler logic
│   ├── parser/           # Parses tokens into instructions
│   ├── codegen/          # Encodes instructions into binary
│   └── util/             # Utility/helpers
├── testdata/             # Example assembly files for testing
├── go.mod
├── go.sum
├── README.md
└── LICENSE
```

## Quick Start

### Prerequisites

- [Go 1.21+](https://golang.org/doc/install)


### Usage

```sh
./m68kasm path/to/source.asm
```

The output binary or hex file will be generated in the current directory (see CLI help for options).

## Example

Input (`testdata/example1.asm`):

```
START:  MOVE.L  #$1234, D0
         ADD.L   D1, D0
         BRA     START
```

Command:

```sh
./m68kasm testdata/example1.asm
```

Output:

```
0000: MOVE.L  #$1234, D0
0002: ADD.L   D1, D0
0004: BRA     START
...
```
*(Binary output will be generated in the future.)*

## Roadmap

- [x] Basic lexer and parser
- [x] Label and symbol resolution
- [ ] Full instruction encoding
- [x] Support for more addressing modes and directives
- [x] Output as binary/hex files
- [ ] Error messages and diagnostics
- [ ] Comprehensive tests

## Contributing

Contributions, bug reports, and feature requests are welcome! Please open issues and pull requests.

## License

[MIT License](LICENSE)

## References

- [Motorola 68000 Programmer’s Reference Manual (PDF)](http://www.nxp.com/docs/en/reference-manual/MC68000UM.pdf)
- [Easy68k Assembler Docs](https://www.easy68k.com/)

---