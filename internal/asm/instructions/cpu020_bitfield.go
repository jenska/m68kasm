package instructions

import "fmt"

// This file adds the 68020 bit-field instructions: BFTST, BFCHG, BFCLR,
// BFSET, BFEXTU, BFEXTS, BFFFO, and BFINS. Encodings were cross-checked
// against GNU binutils' GAS m68k opcode table (opcodes/m68k-opc.c) for
// the per-instruction opcode word, and gas/config/tc-m68k.c's 'O'
// argument case for the offset/width field format (see
// bitFieldSpecCode in encode.go) — not recalled from memory, per the
// same discipline used throughout this project.
//
// The 8-bit-field family shares one extension-word layout: bits 15-12
// (only present on BFEXTU/BFEXTS/BFFFO/BFINS) hold a Dn register — the
// EXTx/FFO result, or INS's source — via the same register-specifier
// nibble MOVEC/MOVES/CHK2/CMP2 already use (FExtRegSrc/FExtRegDst; the
// register is always Dn, so the nibble's A/D bit is always 0 and those
// fields work unchanged here). Bits 11-6 hold the offset code and bits
// 5-0 the width code (FBFOffsetSrc/Dst, FBFWidthSrc/Dst).

func init() {
	registerInstrDef(newBitFieldTestDef("BFTST", 0xE8C0))
	registerInstrDef(newBitFieldTestDef("BFCHG", 0xEAC0))
	registerInstrDef(newBitFieldTestDef("BFCLR", 0xECC0))
	registerInstrDef(newBitFieldTestDef("BFSET", 0xEEC0))
	registerInstrDef(newBitFieldExtractDef("BFEXTU", 0xE9C0))
	registerInstrDef(newBitFieldExtractDef("BFEXTS", 0xEBC0))
	registerInstrDef(newBitFieldExtractDef("BFFFO", 0xEDC0))
	registerInstrDef(&defBFINS)
}

// isBitFieldBase reports whether k is a legal base addressing mode for
// a bit field: Dn, or a data-alterable/readable memory mode excluding
// (An)+ and -(An) (auto-increment addressing makes no sense for a field
// of specific bits rather than a whole operand). isWrite selects which
// classification memory modes need (BFCHG/BFCLR/BFSET/BFINS modify the
// field in place; BFTST/BFEXTU/BFEXTS/BFFFO only read it).
func isBitFieldBase(k EAExprKind, isWrite bool) bool {
	if k == EAkDn {
		return true
	}
	if k == EAkAddrPredec || k == EAkAddrPostinc {
		return false
	}
	if isWrite {
		return memoryAlterableEA[k]
	}
	return readableDataEA[k]
}

func validateBitFieldBase(name string, e EAExpr, isWrite bool) error {
	if !e.HasBitField {
		return fmt.Errorf("%s requires a {offset:width} bit-field specifier", name)
	}
	if !isBitFieldBase(e.Kind, isWrite) {
		if e.Kind == EAkNone {
			return fmt.Errorf("%s requires a base operand", name)
		}
		return fmt.Errorf("%s requires a data register or non-auto-increment memory base", name)
	}
	return nil
}

// newBitFieldTestDef builds BFTST/BFCHG/BFCLR/BFSET: one operand,
// <ea>{offset:width}, no result register. BFTST only reads; the other
// three modify the field in place, hence isWrite.
func newBitFieldTestDef(name string, base uint16) *InstrDef {
	isWrite := name != "BFTST"
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkEA},
				Validate:    func(a *Args) error { return validateBitFieldBase(name, a.Src, isWrite) },
				Requires:    require68020,
				Steps: []EmitStep{
					{WordBits: base, Fields: []FieldRef{FSrcEA}},
					{WordBits: 0x0000, Fields: []FieldRef{FBFOffsetSrc, FBFWidthSrc}},
					{Trailer: []TrailerItem{TSrcEAExt}},
				},
			},
		},
	}
}

// newBitFieldExtractDef builds BFEXTU/BFEXTS/BFFFO: <ea>{offset:width},Dn
// — the extracted (or "first one found", for BFFFO) value is written to
// the trailing Dn.
func newBitFieldExtractDef(name string, base uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkEA, OpkDn},
				Validate:    func(a *Args) error { return validateBitFieldBase(name, a.Src, false) },
				Requires:    require68020,
				Steps: []EmitStep{
					{WordBits: base, Fields: []FieldRef{FSrcEA}},
					{WordBits: 0x0000, Fields: []FieldRef{FExtRegDst, FBFOffsetSrc, FBFWidthSrc}},
					{Trailer: []TrailerItem{TSrcEAExt}},
				},
			},
		},
	}
}

// defBFINS is BFINS Dn,<ea>{offset:width}: the base's field position in
// the operand list is reversed from every other bit-field instruction
// (Dn first, <ea> second), so unlike newBitFieldTestDef/
// newBitFieldExtractDef, the base lives in Dst and its fields use the
// Dst-side FieldRefs (FDstEA/FBFOffsetDst/FBFWidthDst), with the source
// value register read via FExtRegSrc.
var defBFINS = InstrDef{
	Mnemonic: "BFINS",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkDn, OpkEA},
			Validate:    func(a *Args) error { return validateBitFieldBase("BFINS", a.Dst, true) },
			Requires:    require68020,
			Steps: []EmitStep{
				{WordBits: 0xEFC0, Fields: []FieldRef{FDstEA}},
				{WordBits: 0x0000, Fields: []FieldRef{FExtRegSrc, FBFOffsetDst, FBFWidthDst}},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		},
	},
}
