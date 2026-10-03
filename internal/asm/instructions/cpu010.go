package instructions

import (
	"fmt"
	"slices"
	"strings"
)

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

// controlRegSelector maps the 68010's control-register EAExprKinds to the
// 12-bit selector code MOVEC's extension word carries in bits 0-11. Later
// CPUs' registers use EAkCtrlReg with the selector in Reg; see
// laterControlRegisters.
var controlRegSelector = map[EAExprKind]uint16{
	EAkSFC: 0x000,
	EAkDFC: 0x001,
	EAkUSP: 0x800,
	EAkVBR: 0x801,
}

// laterControlRegister is a MOVEC control register added after the 68010,
// with the CPUs that implement it (MC68020/030/040/060 user's manuals).
type laterControlRegister struct {
	sel  uint16
	cpus []CPUKind
}

var laterControlRegisters = map[string]laterControlRegister{
	"CACR":  {0x002, []CPUKind{CPU68020, CPU68030, CPU68040, CPU68060}},
	"TC":    {0x003, []CPUKind{CPU68040, CPU68060}},
	"ITT0":  {0x004, []CPUKind{CPU68040, CPU68060}},
	"ITT1":  {0x005, []CPUKind{CPU68040, CPU68060}},
	"DTT0":  {0x006, []CPUKind{CPU68040, CPU68060}},
	"DTT1":  {0x007, []CPUKind{CPU68040, CPU68060}},
	"BUSCR": {0x008, []CPUKind{CPU68060}},
	"CAAR":  {0x802, []CPUKind{CPU68020, CPU68030}},
	"MSP":   {0x803, []CPUKind{CPU68020, CPU68030, CPU68040}},
	"ISP":   {0x804, []CPUKind{CPU68020, CPU68030, CPU68040}},
	"MMUSR": {0x805, []CPUKind{CPU68040}},
	"URP":   {0x806, []CPUKind{CPU68040, CPU68060}},
	"SRP":   {0x807, []CPUKind{CPU68040, CPU68060}},
	"PCR":   {0x808, []CPUKind{CPU68060}},
}

// LaterControlRegisterSelector returns the MOVEC selector of a control
// register added after the 68010, by name.
func LaterControlRegisterSelector(name string) (uint16, bool) {
	r, ok := laterControlRegisters[strings.ToUpper(name)]
	return r.sel, ok
}

// ControlRegisterSelector returns the 12-bit MOVEC selector code for e,
// and whether e names a recognized control register.
func ControlRegisterSelector(e EAExpr) (uint16, bool) {
	if e.Kind == EAkCtrlReg {
		return uint16(e.Reg), true
	}
	v, ok := controlRegSelector[e.Kind]
	return v, ok
}

func isControlRegKind(k EAExprKind) bool {
	_, ok := controlRegSelector[k]
	return ok || k == EAkCtrlReg
}

// controlRegisterOnCPU reports whether cpu implements the MOVEC control
// register e. The 68010's four exist on every MOVEC-capable CPU.
func controlRegisterOnCPU(e EAExpr, cpu CPUKind) bool {
	if e.Kind != EAkCtrlReg {
		return true
	}
	for _, r := range laterControlRegisters {
		if r.sel == uint16(e.Reg) {
			return slices.Contains(r.cpus, cpu)
		}
	}
	return false
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
	ctrl, gen := a.Src, a.Dst
	if !isControlRegKind(a.Src.Kind) {
		ctrl, gen = a.Dst, a.Src
	}
	if gen.Kind != EAkDn && gen.Kind != EAkAn {
		return fmt.Errorf("MOVEC requires a data or address register operand")
	}
	if !controlRegisterOnCPU(ctrl, a.CPU) {
		return fmt.Errorf("MOVEC control register $%03X does not exist on the target CPU", ctrl.Reg)
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
