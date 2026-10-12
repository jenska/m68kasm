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
// BAD0-7/BAC0-7 (breakpoint registers, 8 numbered instances each, with
// a register number at bits 4-2) are in cpu030_pmmu_badbac.go.
func init() {
	newPmmuFixedReg(OpkDRP, LongSize, 0x4000|(1<<10), 0x4200|(1<<10), memoryAlterableEA, memoryAlterableEA)
	newPmmuFixedReg(OpkCAL, ByteSize, 0x4000|(4<<10), 0x4200|(4<<10), nil, nil)
	newPmmuFixedReg(OpkVAL, ByteSize, 0x4000|(5<<10), 0x4200|(5<<10), nil, nil)
	newPmmuFixedReg(OpkSCC, ByteSize, 0x4000|(6<<10), 0x4200|(6<<10), nil, nil)
	newPmmuFixedReg(OpkAC, WordSize, 0x4000|(7<<10), 0x4200|(7<<10), nil, nil)

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
