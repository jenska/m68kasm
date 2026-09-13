package instructions

// This file adds the PMMU's own conditional branch/set/trap family —
// PBcc/PDBcc/PScc/PTRAPcc — the largest remaining slice of PMMU support
// by instruction count (112 instructions: 16 conditions × 4 families,
// PTRAPcc counted 3× for its bare/word/long-immediate forms), chosen by
// the maintainer from the open items list after milestone 18. Every
// encoding was decoded from GNU binutils' GAS m68k opcode table
// (opcodes/m68k-opc.c) and cross-checked against gas/config/tc-m68k.c's
// install_operand logic.
//
// Structurally, this is the exact same shape already built twice before
// — once for the integer ISA's Bcc/DBcc/Scc/TRAPcc, once for the FPU's
// own 32-condition FBcc/FDBcc/FScc/FTRAPcc (cpu020_fpu2.go) — just with
// the PMMU's own 16 conditions and base opcodes (no coprocessor-ID bit
// to fold in this time, unlike the FPU family: PMMU's word1 base is a
// bare 0xF0xx, not 0xF2xx).
//
// Condition values (0-15) and their two-letter suffixes were decoded
// from the low nibble of each PBcc mnemonic's literal opcode (e.g.
// "pbac" = 0xF087, low nibble 7) and cross-checked against Pscc's own
// table, which independently confirms the identical value assignment —
// see pmmuConditions below.
var pmmuConditions = []string{
	"BS", "BC", "LS", "LC", "SS", "SC", "AS", "AC",
	"WS", "WC", "IS", "IC", "GS", "GC", "CS", "CC",
}

func init() {
	for cc, suffix := range pmmuConditions {
		registerInstrDef(newPBccDef("PB"+suffix, uint16(cc)))
		registerInstrDef(newPDBccDef("PDB"+suffix, uint16(cc)))
		registerInstrDef(newPSccDef("PS"+suffix, uint16(cc)))
		registerInstrDef(newPTrapccDef("PTRAP"+suffix, uint16(cc)))
	}
}

// newPBccDef builds one PBcc mnemonic's two forms: a word displacement
// (word1 = 0xF080|cc, bit 6 clear) and an explicit 32-bit displacement
// (word1 = 0xF0C0|cc, bit 6 set). GAS itself spells the long form as a
// separate "pbccl"-suffixed mnemonic name (or, for the bare mnemonic,
// auto-upgrades from word to long via a two-pass frag resolution) —
// this codebase again commits to its own .W/.L-suffix convention
// instead, the same choice already made for FBcc (cpu020_fpu2.go).
func newPBccDef(name string, cc uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkDispRel},
				Requires:    requirePMMU,
				Steps: []EmitStep{
					{WordBits: 0xF080 | cc},
					{Trailer: []TrailerItem{TBranchWordIfNeeded}},
				},
			},
			{
				DefaultSize: LongSize,
				Sizes:       []Size{LongSize},
				OperKinds:   []OperandKind{OpkDispRel},
				Requires:    requirePMMU,
				Steps: []EmitStep{
					{WordBits: 0xF0C0 | cc},
					{Trailer: []TrailerItem{TBranchLongIfNeeded}},
				},
			},
		},
	}
}

// newPDBccDef builds one PDBcc mnemonic: word1 = 0xF048 | Dn (bits
// 2-0), word2 = the condition value verbatim (0x0000-0x000F, fully
// fixed), then a word-only branch displacement — PDBcc has no 32-bit
// displacement form, matching FDBcc/integer DBcc.
func newPDBccDef(name string, cc uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkDn, OpkDispRel},
				Requires:    requirePMMU,
				Steps: []EmitStep{
					{WordBits: 0xF048, Fields: []FieldRef{FSrcDnReg}},
					{WordBits: cc},
					{Trailer: []TrailerItem{TBranchWordIfNeeded}},
				},
			},
		},
	}
}

// newPSccDef builds one Pscc mnemonic: word1 = 0xF040 | <ea>
// (data-alterable, same restriction as integer Scc and FScc — GAS's
// '$' argument type here matches the '$' Scc itself already uses),
// word2 = the condition value verbatim, fully fixed.
func newPSccDef(name string, cc uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: ByteSize,
				Sizes:       []Size{ByteSize},
				OperKinds:   []OperandKind{OpkEA},
				Validate:    validateDataAlterable(name),
				Requires:    requirePMMU,
				Steps: []EmitStep{
					{WordBits: 0xF040, Fields: []FieldRef{FDstEA}},
					{WordBits: cc},
					{Trailer: []TrailerItem{TDstEAExt}},
				},
			},
		},
	}
}

// newPTrapccDef builds one PTRAPcc mnemonic's three forms (bare, word
// immediate, long immediate) — word1 differs per form (0xF07C bare,
// 0xF07A word, 0xF07B long), matching GAS's "ptrapCC"/"ptrapCCw"/
// "ptrapCCl" table entries, with the condition in a dedicated,
// fully-fixed word2 (0x0000-0x000F) exactly like FTRAPcc/integer
// TRAPcc.
func newPTrapccDef(name string, cc uint16) *InstrDef {
	validateImm := func(a *Args) error { return validateTrapccImm(name, a) }
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{},
				Requires:    requirePMMU,
				Steps: []EmitStep{
					{WordBits: 0xF07C},
					{WordBits: cc},
				},
			},
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkImm},
				Validate:    validateImm,
				Requires:    requirePMMU,
				Steps: []EmitStep{
					{WordBits: 0xF07A},
					{WordBits: cc},
					{Trailer: []TrailerItem{TSrcImm}},
				},
			},
			{
				DefaultSize: LongSize,
				Sizes:       []Size{LongSize},
				OperKinds:   []OperandKind{OpkImm},
				Validate:    validateImm,
				Requires:    requirePMMU,
				Steps: []EmitStep{
					{WordBits: 0xF07B},
					{WordBits: cc},
					{Trailer: []TrailerItem{TSrcImm}},
				},
			},
		},
	}
}
