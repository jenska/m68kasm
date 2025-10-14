# m68kasm

[![CI](https://github.com/jenska/m68kasm/actions/workflows/ci.yml/badge.svg)](../../actions)
![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-informational)

Ein kompakter, tabellengesteuerter **Motorola 68000 Assembler** in Go. Fokus: klare Lexer/Parser-Struktur, 2‑Pass‑Assemble, flexibler EA‑Encoder und gut erweiterbare Instruktionsdefinitionen.

## Features (Stand: v0.1)
- 2‑Pass‑Assembler mit deterministischem Binär‑Output
- Table‑driven Encoding (InstrDef/FormDef/EmitSteps)
- Erste Instruktionen (z. B. `MOVEQ`, `LEA`, `BRA`) und Pseudo‑Ops (z. B. `.org`, `.byte`)
- Einfaches CLI: **m68kasm**

## Installation
```bash
go install github.com/jenska/m68kasm/cmd/m68kasm@latest
```
Oder lokal bauen:
```bash
go build ./cmd/m68kasm
```

## Quickstart
```bash
m68kasm -o out.bin testdata/hello.s
hexdump -C out.bin
```

## CLI
```text
Usage: m68kasm [options] <source-files>
  -o <file>     Schreibe Ausgabe-Binärdatei (default: a.out)
  -I <path>     (geplant) Include-Pfad hinzufügen
  -D name=val   (geplant) Symbol definieren
  --list        (geplant) Listing ausgeben
  -v            (optional) Verbose-Log
```

## Projektstruktur
```text
cmd/m68kasm/          # CLI-Einstieg
internal/asm/         # Lexer, Parser, Expr, EA, Encode, Opdefs, Assemble
testdata/             # Beispiel-Programme & Golden-Files
```

## Entwicklung
- Code formatieren: `go fmt ./...`
- Lint/Vet: `go vet ./...`
- Tests: `go test ./... -v`
