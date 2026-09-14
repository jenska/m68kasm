package instructions

// This file extends PMOVE with six more PMMU registers, chosen by the
// maintainer to complete PMOVE's register set except for BAD/BAC (see
// below): DRP (a root-pointer descriptor sharing CRP/SRP's own
// encoding and restrictions exactly — GAS's own opcode table has one
// shared row for all three, "case 'W':", gas/config/tc-m68k.c), CAL/
// VAL/SCC (Current/Validate/Short-cycle Access Level — three registers
// sharing one row in GAS's table but, like DRP/CRP/SRP/TT0/TT1 before
// them, each still has its own individually fixed selector value once
// written out, so no runtime dispatch is needed here either), AC
// (Access Control), and PCSR (a store-only status pointer — GAS's own
// table has no load-direction row for it at all).
//
// Selector values (bits 12-10 of word2) were cross-checked against
// gas/config/tc-m68k.c's "case '2':" (CAL=4, VAL=5, SCC=6) and
// "case '1':" (AC=7) — the same install_operand dispatch DRP/CRP/SRP's
// own selectors were confirmed against in cpu030_pmmu_xlate.go.
//
// Sizes were cross-checked against GAS's own per-register type-check
// restriction, not assumed uniform with TC/DRP/CRP/SRP/TT0/TT1's own
// ".L": AC is ".W" ("case '1':" restricts it to PMOVE's word-sized
// row), and CAL/VAL/SCC are ".B" ("case '2':" restricts them to the
// byte-sized row) — GAS gives TC, AC, and CAL/VAL/SCC three physically
// different opcode-table rows despite all three groups sharing the
// exact same install_operand selector computation, which is what
// exposes that each group's *size* differs even though the underlying
// bit-shift arithmetic doesn't.
//
// "PSR" — the 68851's own name for the register the 68030 calls
// "MMUSR" — is handled in parser_operand.go's existing OpkMMUSR case,
// not here: it's the exact same encoding already shipped in
// cpu030_pmmu_xlate.go, so it needed a second accepted spelling, not a
// second Form.
//
// Deliberately NOT included: BAD0-7/BAC0-7 (breakpoint address/access
// registers, 8 numbered instances each). Their encoding is genuinely
// different in shape from everything else PMOVE moves — a register
// NUMBER at bits 4-2 in addition to the usual selector, and (unlike
// every other PMOVE register) an INVERTED load/store direction bit
// (GAS's own table: load word2 0x6200, store word2 0x6000 — the
// opposite of every other register's 0x...00-load/0x...200-store
// convention) — a distinct enough sub-feature, and obscure enough even
// among an already-rare instruction family (68851-only debug/
// breakpoint support), to warrant its own separately-scoped milestone
// rather than folding it in here.
func init() {
	newPmmuFixedReg("DRP", OpkDRP, LongSize, 0x4000|(1<<10), 0x4200|(1<<10), memoryAlterableEA, memoryAlterableEA)
	newPmmuFixedReg("CAL", OpkCAL, ByteSize, 0x4000|(4<<10), 0x4200|(4<<10), nil, nil)
	newPmmuFixedReg("VAL", OpkVAL, ByteSize, 0x4000|(5<<10), 0x4200|(5<<10), nil, nil)
	newPmmuFixedReg("SCC", OpkSCC, ByteSize, 0x4000|(6<<10), 0x4200|(6<<10), nil, nil)
	newPmmuFixedReg("AC", OpkAC, WordSize, 0x4000|(7<<10), 0x4200|(7<<10), nil, nil)

	// PMOVE PCSR,<ea> — store only; GAS's table has no load-side row.
	defPMOVE.Forms = append(defPMOVE.Forms, FormDef{
		DefaultSize: WordSize,
		Sizes:       []Size{WordSize},
		OperKinds:   []OperandKind{OpkPCSR, OpkEA},
		Validate:    validatePmoveStore,
		Requires:    requirePMMU,
		Steps: []EmitStep{
			{WordBits: 0xF000, Fields: []FieldRef{FDstEA}},
			{WordBits: 0x6600},
			{Trailer: []TrailerItem{TDstEAExt}},
		},
	})
}
