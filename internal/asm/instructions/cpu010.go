package instructions

import "fmt"

// This file adds the MC68010/MC68012 instructions: MOVEC, MOVES, RTD, and
// BKPT. Encodings were cross-checked against GNU binutils' GAS m68k backend
// (opcodes/m68k-opc.c and gas/config/tc-m68k.c), not just recalled from
// memory, since a wrong bit placement here would silently emit incorrect
// machine code. See docs/design/cpu-family-support.md, milestone 2.

func init() {
	registerInstrDef(&defMOVEC)
	registerInstrDef(&defMOVES)
	registerInstrDef(&defRTD)
	registerInstrDef(&defBKPT)
}

var require68010 = Target{CPU: CPU68010}

// controlRegSelector maps a control-register EAExprKind to the 12-bit
// selector code MOVEC's extension word carries in bits 0-11. Only the
// registers introduced by the 68010 are listed; 68020+ added many more
// (CACR, CAAR, MSP, ISP, ...) that belong to a later milestone.
var controlRegSelector = map[EAExprKind]uint16{
	EAkSFC: 0x000,
	EAkDFC: 0x001,
	EAkUSP: 0x800,
	EAkVBR: 0x801,
}

// ControlRegisterSelector returns the 12-bit MOVEC selector code for k,
// and whether k names a recognized control register.
func ControlRegisterSelector(k EAExprKind) (uint16, bool) {
	v, ok := controlRegSelector[k]
	return v, ok
}

func isControlRegKind(k EAExprKind) bool {
	_, ok := controlRegSelector[k]
	return ok
}

var defMOVEC = InstrDef{
	Mnemonic: "MOVEC",
	Forms: []FormDef{
		{
			// MOVEC Rc,Rn (control register to general register): 0x4E7A.
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OpkCtrlReg, OpkEA},
			Validate:    validateMOVEC,
			Requires:    require68010,
			Steps: []EmitStep{
				{WordBits: 0x4E7A},
				{WordBits: 0x0000, Fields: []FieldRef{FCtrlRegSel, FExtRegDst}},
			},
		},
		{
			// MOVEC Rn,Rc (general register to control register): 0x4E7B.
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OpkEA, OpkCtrlReg},
			Validate:    validateMOVEC,
			Requires:    require68010,
			Steps: []EmitStep{
				{WordBits: 0x4E7B},
				{WordBits: 0x0000, Fields: []FieldRef{FCtrlRegSel, FExtRegSrc}},
			},
		},
	},
}

func validateMOVEC(a *Args) error {
	gen := a.Dst
	if !isControlRegKind(a.Src.Kind) {
		gen = a.Src
	}
	if gen.Kind != EAkDn && gen.Kind != EAkAn {
		return fmt.Errorf("MOVEC requires a data or address register operand")
	}
	return nil
}

var defMOVES = InstrDef{
	Mnemonic: "MOVES",
	Forms: []FormDef{
		{
			// MOVES <ea>,Dn (dr=0): 0x0E00 | size, ext word bit11=0.
			DefaultSize: WordSize,
			Sizes:       []Size{ByteSize, WordSize, LongSize},
			OperKinds:   []OperandKind{OpkEA, OpkDn},
			Validate:    validateMOVES,
			Requires:    require68010,
			Steps: []EmitStep{
				{WordBits: 0x0E00, Fields: []FieldRef{FSizeBits, FSrcEA}},
				{WordBits: 0x0000, Fields: []FieldRef{FExtRegDst}},
				{Trailer: []TrailerItem{TSrcEAExt}},
			},
		},
		{
			// MOVES <ea>,An (dr=0).
			DefaultSize: WordSize,
			Sizes:       []Size{ByteSize, WordSize, LongSize},
			OperKinds:   []OperandKind{OpkEA, OpkAn},
			Validate:    validateMOVES,
			Requires:    require68010,
			Steps: []EmitStep{
				{WordBits: 0x0E00, Fields: []FieldRef{FSizeBits, FSrcEA}},
				{WordBits: 0x0000, Fields: []FieldRef{FExtRegDst}},
				{Trailer: []TrailerItem{TSrcEAExt}},
			},
		},
		{
			// MOVES Dn,<ea> (dr=1): 0x0E00 | size, ext word bit11=1.
			DefaultSize: WordSize,
			Sizes:       []Size{ByteSize, WordSize, LongSize},
			OperKinds:   []OperandKind{OpkDn, OpkEA},
			Validate:    validateMOVES,
			Requires:    require68010,
			Steps: []EmitStep{
				{WordBits: 0x0E00, Fields: []FieldRef{FSizeBits, FDstEA}},
				{WordBits: 0x0800, Fields: []FieldRef{FExtRegSrc}},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		},
		{
			// MOVES An,<ea> (dr=1).
			DefaultSize: WordSize,
			Sizes:       []Size{ByteSize, WordSize, LongSize},
			OperKinds:   []OperandKind{OpkAn, OpkEA},
			Validate:    validateMOVES,
			Requires:    require68010,
			Steps: []EmitStep{
				{WordBits: 0x0E00, Fields: []FieldRef{FSizeBits, FDstEA}},
				{WordBits: 0x0800, Fields: []FieldRef{FExtRegSrc}},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		},
	},
}

func validateMOVES(a *Args) error {
	if a.Src.Kind == EAkDn || a.Src.Kind == EAkAn {
		if !isMemoryAlterable(a.Dst.Kind) {
			return fmt.Errorf("MOVES requires a memory-alterable destination")
		}
		return nil
	}
	if !isMemoryAlterable(a.Src.Kind) {
		return fmt.Errorf("MOVES requires a memory-alterable source")
	}
	return nil
}

var defRTD = InstrDef{
	Mnemonic: "RTD",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkImm},
			Validate:    validateRTD,
			Requires:    require68010,
			Steps: []EmitStep{
				{WordBits: 0x4E74},
				{Trailer: []TrailerItem{TImmSized}},
			},
		},
	},
}

func validateRTD(a *Args) error {
	if a.Src.Kind != EAkImm {
		return fmt.Errorf("RTD requires an immediate displacement")
	}
	if a.Src.Imm < -32768 || a.Src.Imm > 32767 {
		return fmt.Errorf("RTD displacement out of range: %d", a.Src.Imm)
	}
	return nil
}

var defBKPT = InstrDef{
	Mnemonic: "BKPT",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkImmQuick},
			Validate:    validateBKPT,
			Requires:    require68010,
			Steps: []EmitStep{
				{WordBits: 0x4848, Fields: []FieldRef{FImmLow8}},
			},
		},
	},
}

func validateBKPT(a *Args) error {
	if a.Src.Imm < 0 || a.Src.Imm > 7 {
		return fmt.Errorf("BKPT vector out of range: %d", a.Src.Imm)
	}
	return nil
}
