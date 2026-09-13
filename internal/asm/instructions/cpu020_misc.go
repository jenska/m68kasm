package instructions

import "fmt"

// This file adds a first batch of 68020 integer instructions: EXTB.L,
// CHK2/CMP2, and TRAPcc (all three of its forms — no operand, word
// operand, long operand — for all 16 condition codes). Encodings were
// cross-checked against GNU binutils' GAS m68k opcode table
// (opcodes/m68k-opc.c), not just recalled from memory, per the same
// discipline used for the 68010 additions in cpu010.go.
//
// Deliberately NOT included here (left for a follow-up milestone): bit
// field instructions (BFTST/BFCHG/.../BFINS, which need a new operand
// kind and lexer support for "{offset:width}" syntax), CAS/CAS2, PACK/
// UNPK, DIVSL/DIVUL (its 64-bit-dividend "Dr:Dq" register-pair syntax
// needs its own new operand kind too), and CALLM/RTM (obscure and
// rarely implemented even on real 68020 silicon). See
// docs/design/cpu-family-support.md.

// require68020orCPU32 tags EXTB.L, CHK2/CMP2, and TRAPcc: GNU binutils'
// gas/config/tc-m68k.c tags every one of these mnemonics with both
// m68020up and cpu32, confirming CPU32 backported them alongside
// 68020's other integer additions. Target{CPU: CPU68020} would be wrong
// here — Supports' plain CPU-floor comparison would then deny CPU32
// (its enum value sits below CPU68020), even though real CPU32 hardware
// has these instructions.
var require68020orCPU32 = Target{CPU: CPU32}

func init() {
	registerInstrDef(&defEXTB)
	registerInstrDef(newChk2Cmp2Def("CHK2", 0x0800))
	registerInstrDef(newChk2Cmp2Def("CMP2", 0x0000))
	for c, m := range trapConditions {
		registerInstrDef(newTrapccDef(m, uint16(c)))
	}
	registerInstrDef(&defBGND)
}

// defBGND is BGND (enter background debug mode): unlike everything else
// in this file, it is CPU32-*exclusive* — no other tier, including
// 68020+, has it (GNU binutils' gas/config/tc-m68k.c tags it cpu32
// alone, not m68020up|cpu32) — hence requireCPU32Only rather than
// require68020orCPU32.
var defBGND = InstrDef{
	Mnemonic: "BGND",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{},
			Requires:    requireCPU32Only,
			Steps: []EmitStep{
				{WordBits: 0x4AFA},
			},
		},
	},
}

var defEXTB = InstrDef{
	Mnemonic: "EXTB",
	Forms: []FormDef{
		{
			// EXTB.L Dn: sign-extend byte directly to long. Distinct
			// mnemonic and opcode from EXT.L (word-to-long); EXT has no
			// byte-to-word form (byte can't be EXTended to word and get a
			// useful new mnemonic — EXTB is specifically byte-to-long).
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OpkDn},
			Validate:    validateEXTB,
			Requires:    require68020orCPU32,
			Steps: []EmitStep{
				{WordBits: 0x49C0, Fields: []FieldRef{FDstRegLow}},
			},
		},
	},
}

func validateEXTB(a *Args) error {
	swapSrcDstIfDstNone(a)
	if a.Dst.Kind != EAkDn {
		return fmt.Errorf("EXTB requires Dn destination")
	}
	return nil
}

// newChk2Cmp2Def builds CHK2/CMP2, which share one opcode per size
// (0x00C0/0x02C0/0x04C0 for .B/.W/.L) and differ only in bit 11 of the
// extension word (chkBit: 0x0800 for CHK2, 0 for CMP2). The extension
// word's other half (bits 15-12: A/D select + register number) is the
// same "register specifier nibble" MOVEC/MOVES already use — see
// FExtRegDst in cpu010.go and encode.go.
func newChk2Cmp2Def(name string, chkBit uint16) *InstrDef {
	validate := func(a *Args) error { return validateChk2Cmp2(name, a) }
	sizeBase := func(base uint16, sz Size) FormDef {
		return FormDef{
			DefaultSize: sz,
			Sizes:       []Size{sz},
			OperKinds:   []OperandKind{OpkEA, OpkEA},
			Validate:    validate,
			Requires:    require68020orCPU32,
			Steps: []EmitStep{
				{WordBits: base, Fields: []FieldRef{FSrcEA}},
				{Trailer: []TrailerItem{TSrcEAExt}},
				{WordBits: chkBit, Fields: []FieldRef{FExtRegDst}},
			},
		}
	}
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			sizeBase(0x00C0, ByteSize),
			sizeBase(0x02C0, WordSize),
			sizeBase(0x04C0, LongSize),
		},
	}
}

func validateChk2Cmp2(name string, a *Args) error {
	if !controlAlterableEA[a.Src.Kind] {
		if a.Src.Kind == EAkNone {
			return fmt.Errorf("%s requires a source operand", name)
		}
		return fmt.Errorf("%s requires control addressing mode", name)
	}
	if a.Dst.Kind != EAkDn && a.Dst.Kind != EAkAn {
		return fmt.Errorf("%s requires a data or address register", name)
	}
	return nil
}

// trapConditions names TRAPcc's 16 condition-code variants, in standard
// 68k condition-code order (index == condition code), matching this
// codebase's own HS/LO naming for indices 4/5 (branchConditions,
// sccConditions) rather than GAS's CC/CS spelling of the same bits.
var trapConditions = []string{
	"TRAPT", "TRAPF", "TRAPHI", "TRAPLS", "TRAPHS", "TRAPLO", "TRAPNE", "TRAPEQ",
	"TRAPVC", "TRAPVS", "TRAPPL", "TRAPMI", "TRAPGE", "TRAPLT", "TRAPGT", "TRAPLE",
}

// newTrapccDef builds one TRAPcc mnemonic's three forms: no operand
// (0x50FC | cc<<8), a word immediate (0x50FA | cc<<8), and a long
// immediate (0x50FB | cc<<8). The condition code occupies bits 11-8,
// the same position Scc/DBcc/Bcc use for their own condition field.
func newTrapccDef(name string, cc uint16) *InstrDef {
	base := 0x50FC | cc<<8
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{},
				Requires:    require68020orCPU32,
				Steps:       []EmitStep{{WordBits: base}},
			},
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkImm},
				Validate:    func(a *Args) error { return validateTrapccImm(name, a) },
				Requires:    require68020orCPU32,
				Steps: []EmitStep{
					{WordBits: base - 2},
					{Trailer: []TrailerItem{TSrcImm}},
				},
			},
			{
				DefaultSize: LongSize,
				Sizes:       []Size{LongSize},
				OperKinds:   []OperandKind{OpkImm},
				Validate:    func(a *Args) error { return validateTrapccImm(name, a) },
				Requires:    require68020orCPU32,
				Steps: []EmitStep{
					{WordBits: base - 1},
					{Trailer: []TrailerItem{TSrcImm}},
				},
			},
		},
	}
}

func validateTrapccImm(name string, a *Args) error {
	if a.Src.Kind != EAkImm {
		return fmt.Errorf("%s requires an immediate operand", name)
	}
	return checkImmediateRange(a.Src.Imm, a.Size)
}
