# 🧠 M68k Assembler in Go

[![Go](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A compact, table-driven **Motorola 68000 assembler**, implemented entirely in **Go**.

---

## ✨ Features

- Lexer and panic-free parser for 68k assembly syntax  
- Expression parser with operator precedence and label references  
- Effective Address (EA) parser and encoder (brief index, PC-relative, abs.W/.L)  
- Two-pass assembler with automatic label resolution  
- Generic encoder based on instruction tables (`InstrDef`, `FormDef`, `EmitStep`)  
- Implemented instructions: MOVEQ, LEA, BRA  
- Pseudo-operations: `.org`, `.byte`  
- Deterministic binary output

---

## 🧩 Example Source

```asm
        .org $0000 + 4*4
start:  moveq #5+3-1,d3
        lea (16,a1),a0
        lea (8,pc,d2*2),a1
        bra start
        .byte $AA,$BB,$CC
```

---

## ⚙️ Building

Requires **Go 1.21+**

```bash
git clone https://github.com/jenska/m68kasm.git
cd m68kasm
go build ./cmd/m68kasm
```

---

## ▶️ Usage

```bash
./m68kasm -i testdata/hello.s -o out.bin
hexdump -C out.bin
```

Expected output:
```
76 07 41 e9 00 10 43 fb 22 08 60 f6 aa bb cc
```

---

## 🧱 Project Structure

```
m68kasm/
├── cmd/m68kasm/
├── internal/
│   └── asn/
└── go.mod
```

Key components:

| File | Purpose |
|------|----------|
| internal/asm/lexer.go | Tokenizes assembly text |
| internal/asm/parser.go | Parses lines, labels, pseudo-ops |
| internal/asm/expr.go | Expression evaluator |
| internal/asm/ea.go | Effective Address encoding |
| internal/asm/encode.go | Generic instruction encoder |
| internal/asm/opdefs.go | Instruction tables |
| internal/asm/assemble.go | Full assembly pass orchestrator |

---

## 🧭 Roadmap

- [ ] Add MOVE, ADD, SUB, CMP instructions  
- [ ] More branch types (Bcc, BSR)  
- [ ] .word, .long, .align pseudo-ops  
- [ ] Object format (S-record / ELF)  
- [ ] Source listing with address + opcode dump  
- [ ] Unit tests for parser and encoder

---

## 🪪 License

Licensed under the **MIT License**.  
See [LICENSE](LICENSE) for details.

---

## ❤️ Contributing

Pull requests are welcome!  
Please run `go fmt ./...` before submitting.
