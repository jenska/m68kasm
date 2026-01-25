# M68K Assembler - AI Coding Agent Instructions

## Project Overview
A **table-driven Motorola 68000 assembler** in Go with deterministic binary output. Designed for educational use, embedded projects, and retro computing. Two-pass parsing architecture: lexer → parser (with symbol resolution) → instruction encoding.

## Architecture

### Core Processing Pipeline
1. **Lexer** (`internal/asm/lexer.go`): Tokenizes source into `Token` structs with line/column tracking
2. **Parser** (`internal/asm/parser.go`): Two-pass symbol resolution and macro expansion
3. **Assembler** (`internal/asm/assemble.go`): Walks `Program.Items` (instructions/data blocks), encodes to bytes
4. **Instruction Encoding** (`internal/asm/encode.go`): Table-driven via `EmitStep` sequences with field/trailer modifications

### Key Data Structures
- **`Program`**: Contains `Items []any`, `Labels map[string]uint32`, `Origin uint32`
- **`Instr`**: `Def *InstrDef`, `Args instructions.Args`, `PC`, `Line`
- **`InstrDef`**: `Mnemonic`, `Forms []FormDef` (instruction variants by operand kinds)
- **`FormDef`**: `DefaultSize`, `Sizes[]`, `OperKinds[]`, `Validate func`, `Steps []EmitStep`
- **`EmitStep`**: `WordBits uint16`, `Fields[]FieldRef`, `Trailer[]TrailerItem` — how to emit one word/extension

## Instruction Encoding (Table-Driven Pattern)

Instructions **must** be defined in `internal/asm/instructions/` files (grouped by type: `000_divmul.go`, `move.go`, `shift.go`, etc.).

### Example: MULU Encoding
```go
// 000_divmul.go
registerInstrDef(&InstrDef{
    Mnemonic: "MULU",
    Forms: []FormDef{{
        DefaultSize: WordSize,
        OperKinds: []OperandKind{OpkEA, OpkDn},
        Validate: func(a *Args) error { /* check constraints */ },
        Steps: []EmitStep{
            {WordBits: 0xC0C0, Fields: []FieldRef{FDnReg, FSrcEA}},
            {Trailer: []TrailerItem{TSrcEAExt}},
        },
    }},
})
```

### Encoding Workflow
1. Parser creates `Instr` with `DefDef`, `Args`, `PC`
2. `selectForm()` matches operand kinds to a `FormDef`
3. `FormDef.Validate()` checks 68k-specific rules (e.g., MULU forbids An source)
4. For each `EmitStep`:
   - Start with `WordBits` as base opcode
   - Apply `Fields[]` via `applyField()` (inserts register/EA bits)
   - Apply `Trailer[]` items (extension words, immediates, register masks)
5. `appendWord()` writes big-endian to output

### Field References
- **`FDnReg`, `FAnReg`**: Insert Dn/An bits at position 9 (3 bits)
- **`FSrcEA`, `FDstEA`**: Insert EA mode/register bits
- **`FSizeBits`**: Add size field (0x0000=byte, 0x0040=word, 0x0080=long)
- **`FImmLow8`**: Insert 8-bit immediate

## Effective Address (EA) Encoding

### Core Concept
EAs encode memory/register operands as a **mode (3-bit)** + **register (3-bit)** pair, plus optional extension words. The `EAEncoded` struct holds both:
```go
type EAEncoded struct {
    Mode int        // 0-7: selects addressing mode
    Reg  int        // 0-7: selects register within that mode
    Ext  []uint16   // Extension words (displacement, index, absolute address)
}
```

### Addressing Modes (Mode values)
| Mode | Reg | Meaning | Extension |
|------|-----|---------|-----------|
| 0 | Dn | Data register | None |
| 1 | An | Address register | None |
| 2 | An | Address indirect `(An)` | None |
| 3 | An | Post-increment `(An)+` | None |
| 4 | An | Pre-decrement `-(An)` | None |
| 5 | An | Address + 16-bit disp `d16(An)` | 1 word (16-bit displacement) |
| 6 | An | Indexed indirect `d8(An,Rx.size*scale)` | 1 word (brief format) |
| 7 | Special | Absolute/PC-relative/Immediate | Varies |

### Mode 7 (Special) Sub-Modes
| Reg | Meaning | Extension |
|-----|---------|-----------|
| 0 | Absolute word address | 1 word (16-bit) |
| 1 | Absolute long address | 2 words (32-bit) |
| 2 | PC + 16-bit disp `d16(PC)` | 1 word (16-bit displacement) |
| 3 | PC + indexed `d8(PC,Rx)` | 1 word (brief format) |
| 4 | Immediate `#imm` | Size-dependent (1-2 words) |

### Extension Word Formats

**16-bit Displacement** (Mode 5, 7.2, 7.3): Plain signed 16-bit value.

**Brief Index Format** (Mode 6, 7.3):
```
Bit 15:    Sign bit (not used in 68000, reserved for 68020+)
Bit 7:     Address register flag (1=An, 0=Dn)
Bits 6-4:  Register number
Bit 3:     Size (1=long, 0=word)
Bits 2-1:  Scale (00=×1, 01=×2, 10=×4, 11=×8)
Bits 7-0:  8-bit signed displacement
```

### Encoding in Instructions
`applyField()` combines mode/reg bits into the opcode word:
```go
case instructions.FSrcEA:
    return wordVal | (uint16(p.SrcEA.Mode&7) << 3) | uint16(p.SrcEA.Reg&7)
```
This shifts **mode left 3 bits** (bits 5-3) and **reg stays in bits 2-0**.

`emitTrailer()` appends any extension words from `EAEncoded.Ext`:
```go
case instructions.TSrcEAExt:
    for _, w := range p.SrcEA.Ext {
        out = appendWord(out, w)  // Append each extension word
    }
```

### Parsing and Encoding Flow
1. **Parser** (`parser.go`): Tokenizes operands into `EAExpr` (kind, register, displacement, index info)
2. **`EncodeEA()`** (`instructions/ea.go`): Looks up mode/reg from `eaTable` based on `EAExpr.Kind`, calculates extension words
3. **`prepared` struct** (`encode.go`): Stores both `SrcEA` and `DstEA` for the instruction
4. **`applyField()`**: Inserts mode/reg bits during opcode word construction
5. **`emitTrailer()`**: Writes extension words after the opcode

### Common Examples
- `MULU.W (A0),D0`: Mode 2, Reg 0 → bits `010` (shift 3) + `000` = `0x10`
- `MULU.W #100,D0`: Mode 7, Reg 4 → bits `111` (shift 3) + `100` = `0xFC`
- `MULU.W 100(A0),D0`: Mode 5, Reg 0 → bits `101` (shift 3) + `000` = `0x28`; extension word = 0x0064

## Parser & Symbol Resolution

### Two-Pass Behavior
- **Pass 1** (parsing): Resolve labels, measure program size, expand macros in-place
- **Pass 2** (implicit): During assembly, labels are looked up from `Program.Labels`

### Directives (Pseudo-ops)
Defined in `pseudos.go`:
- **`.ORG <addr>`**: Set origin; can only move forward
- **`.BYTE/.WORD/.LONG <expr>`**: Emit data with expression evaluation
- **`.ALIGN <n>`**: Pad to alignment (fails if > `maxProgramSize = 64 MiB`)
- **`.MACRO name(params)`...`ENDM`**: Define reusable token sequences

### Expression Evaluation
`parseExpr()` supports: arithmetic (`+`, `-`, `*`, `/`, `%`), bitwise (`&`, `|`, `^`, `<<`, `>>`), comparisons (`==`, `<`, `>`, `<=`, `>=`), and logical (`&&`, `||`). Operands: integer literals, symbols, label references, `$` (current PC).

### Local Numeric Labels
- Syntax: `1:` (define), `1f` (forward ref), `1b` (backward ref)
- Internally mapped to unique names like `__local_1_0`
- Must be resolved in order during parsing

## API Surface

### High-Level Functions (Package-Level)
```go
Assemble(r io.Reader) ([]byte, error)
AssembleStream(w io.Writer, r io.Reader) (int64, error)
AssembleWithListing(r io.Reader) ([]byte, []ListingEntry, error)
```

### Options-Based Variants
All support `ParseOptions` (symbol definitions, include paths, etc.):
```go
type ParseOptions struct {
    Symbols map[string]uint32  // Pre-defined symbols
    Includes []string          // Include search paths
}
```

### Listing Support
`ListingEntry` captures `Line`, `PC`, `Bytes` for each source line. Enables source-listing generation.

## Error Handling

**All errors wrap with source context** via `Error` struct (implements `error`):
```go
type Error struct {
    Line, Col int
    LineText string
    Err error
}
```
- Validation errors in `FormDef.Validate()` are contextualized at the parse line
- Label resolution failures, operand kind mismatches, and encoding errors all report line/column

## Output Formats

### Internal Assembly Flow
1. `assemble()` loops over `Program.Items`
2. Each `Instr` → select form → encode to bytes via `EmitStep`
3. Each `DataBytes` → append raw bytes
4. Output to buffer, writer, or stream

### Format Adapters
- **Binary**: Raw bytes from `assemble()`
- **S-Record** (`srec.go`): Chunks output into Motorola S3 records with checksums
- **ELF32** (`elf.go`): Single load segment, m68k architecture

## Key Constraints & Patterns

### 68000 CPU Only
- No 68010, 68020+, or FPU coprocessor extensions
- Instruction table in `instructions/table.go` is the canonical source of truth

### Deterministic Output
- No instruction relaxation (JMP → BRA)
- Branch encoding: always checks distance and uses word/byte displacement as needed
- Fixed operand forms prevent ambiguous encodings

### Operand Kind System (`instructions/types.go`)
Each instruction `FormDef` specifies expected operand kinds:
- `OpkEA`: Effective address (registers, memory modes)
- `OpkImm`: Any immediate
- `OpkImmQuick`: Limited immediate (3-8 bits for ADDQ/SUBQ)
- `OpkDn`, `OpkAn`: Data/address registers
- `OpkRegList`: Register list for MOVEM
- `OpkPredecAn`: Predecrement addressing
- `OpkDispRel`: PC-relative (labels)

## Testing Strategy

### Test Organization
- **Unit tests** in `*_test.go` files test specific subsystems (parsing, encoding, macros)
- **E2E tests** in `tests/e2e/` test complete assembly workflows
- **Benchmarks** measure parse/assemble/stream performance

### Common Test Patterns
```go
// Verify instruction encoding
src := "MULU.W D1,D0"
prog, _ := internal.Parse(strings.NewReader(src))
bytes, _ := internal.Assemble(prog)
// Assert bytes match expected hex

// Test validation errors
src := "MULU.L D1,D0"  // Invalid size
_, err := internal.Parse(strings.NewReader(src))
// Assert error message contains "word size"
```

### Test Data
- Small test programs in test functions
- Larger examples in `examples/` and `tests/testdata/` (hello.s, qsort.s)

## Development Workflow

### Building
```bash
go build ./cmd/m68kasm  # Build CLI
go test ./...            # Run all tests
go vet ./...             # Run vet
staticcheck ./...        # Run staticcheck (if installed)
```

### Adding Instructions
1. Create/edit file in `internal/asm/instructions/` (e.g., `shift.go`)
2. Call `registerInstrDef()` in `init()` with `InstrDef` and `FormDef` list (never add directly to `Instructions` map)
3. Implement `FormDef.Validate()` for 68k-specific rules
4. Define `EmitStep` sequence: base opcode + field/trailer modifications
5. Add tests to `instructions_test.go` verifying encoding and validation

**Important:** Always use `registerInstrDef()` to register instructions. This function:
- Validates no duplicate mnemonics exist
- Maintains a single source of truth in `Instructions` map
- Prevents accidental overwrites or conflicts

### Common Pitfalls
- **Field overlap**: Two `FieldRef`s applying to the same bit region → use only one per instruction variant
- **Missing `TSrcEAExt`/`TDstEAExt`**: EA modes with extension words must include trailer
- **Size validation**: Always enforce 68k-correct sizes (e.g., MULU is word-only)
- **PC calculation**: Include extension words when computing branch displacement

## References
- Grammar: `docs/grammar.ebnf`
- Instruction syntax: `docs/syntax.md`
- Example programs: `examples/`
