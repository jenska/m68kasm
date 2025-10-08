# Changelog

## [0.1.0] - 2025-10-08
### Added
- Initial public release of the m68k assembler in Go
- Lexer, parser (panic-free), label resolution
- Expression parser with stop-sets (works inside EA)
- EA encoding incl. brief index
- Instructions: MOVEQ, LEA, BRA
- Pseudo-ops: .org, .byte
- Generic, table-driven encoder (fixes for MOVEQ imm8 and BRA short/word)
- Example: internal/asm/testdata/hello.s
