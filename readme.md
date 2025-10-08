# M68k Assembler in Go — First Release

This is a compact Motorola **68000 assembler** written in **Go**.
It includes:
- A lexer and token-based parser
- Expression parser with label resolution
- Effective Address (EA) parsing & encoding (brief index supported)
- Generic, table-driven encoder
- Instructions: **MOVEQ**, **LEA**, **BRA**
- Pseudo-ops: **.org**, **.byte**

## Example (`internal/asm/testdata/hello.s`)
```asm
        .org $0000 + 4*4
start:  moveq #5+3-1,d3
        lea (16,a1),a0
        lea (8,pc,d2*2),a1
        bra start
        .byte $AA,$BB,$CC
```

## Build & Run
```bash
go build ./cmd/m68kasm
./m68kasm -i internal/asm/testdata/hello.s -o out.bin
hexdump -C out.bin
```

Expected bytes:
```
76 07 41 e9 00 10 43 fb 22 08 60 f6 aa bb cc
```
