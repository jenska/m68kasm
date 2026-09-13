package instructions

import "fmt"

// This file adds a first, deliberately bounded slice of the 68881/68882
// (or 68040/68060-integrated) FPU instruction set: FMOVE, FADD, FSUB,
// FMUL, FDIV, FCMP, FABS, FNEG, FSQRT, FTST, and FNOP. Every encoding
// was decoded from GNU binutils' GAS m68k opcode table
// (opcodes/m68k-opc.c) rather than recalled from memory — see the
// bit-layout notes on fpFormatCode (encode.go) and newFPBinaryDef below.
//
// Deliberately NOT included (left for a follow-up): floating-point
// immediate literals ("#1.5") — the lexer has no float-literal syntax,
// only integers, so only integer (.b/.w/.l) immediates work today;
// FMOVEM (register-list save/restore — needs its own list syntax
// parallel to MOVEM's); FMOVECR (ROM constant load); FBcc/FDBcc/FScc/
// FTRAPcc (floating conditional branches — a distinct 32-entry FPU
// condition-code space); the transcendental function set (FSIN, FLOGN,
// …); FSAVE/FRESTORE; packed-BCD format (.p); and FPCR/FPSR/FPIAR
// control-register operands. See docs/design/cpu-family-support.md.
//
// Gating: Requires: Target{Features: FeatFPU} rather than a CPU floor —
// FPU presence is an attached-coprocessor question independent of the
// integer CPU tier (a bare 68020+external 68881 and a 68040's built-in
// FPU both just need FeatFPU set), matching the Target model cpu.go
// introduced in milestone 1.

var requireFPU = Target{Features: FeatFPU}

func init() {
	registerInstrDef(newFPBinaryDef("FMOVE", 0x00, true))
	registerInstrDef(newFPBinaryDef("FADD", 0x22, false))
	registerInstrDef(newFPBinaryDef("FSUB", 0x28, false))
	registerInstrDef(newFPBinaryDef("FMUL", 0x23, false))
	registerInstrDef(newFPBinaryDef("FDIV", 0x20, false))
	registerInstrDef(newFPBinaryDef("FCMP", 0x38, false))
	registerInstrDef(newFPMonadicDef("FABS", 0x18))
	registerInstrDef(newFPMonadicDef("FNEG", 0x1A))
	registerInstrDef(newFPMonadicDef("FSQRT", 0x04))
	registerInstrDef(&defFTST)
	registerInstrDef(&defFNOP)
}

// fpSizes are the sizes every plain FPU arithmetic/move form accepts:
// integer conversions (.b/.w/.l) and the two supported floating formats
// (.s/.d/.x — packed BCD, .p, is not implemented).
var fpSizes = []Size{ByteSize, WordSize, LongSize, SingleSize, DoubleSize, ExtendedSize}

func isFPIntSize(sz Size) bool {
	return sz == ByteSize || sz == WordSize || sz == LongSize
}

// validateFPUOperand checks one <ea>/register operand of an FPU
// instruction. isStore means this operand is being written (the
// destination of FMOVE FPn,<ea>); it is always false for every other
// instruction in this file, since none of them can store to memory.
func validateFPUOperand(name string, isStore bool, k EAExprKind, sz Size) error {
	switch k {
	case EAkFPn:
		return nil
	case EAkAn:
		return fmt.Errorf("%s does not support address register operands", name)
	case EAkImm:
		if isStore {
			return fmt.Errorf("%s destination cannot be immediate", name)
		}
		if !isFPIntSize(sz) {
			return fmt.Errorf("%s: floating-point immediate literals are not yet supported (use .b/.w/.l for an integer immediate)", name)
		}
		return nil
	case EAkDn:
		if !isFPIntSize(sz) {
			return fmt.Errorf("%s: a floating-point size requires a memory operand, not Dn", name)
		}
		return nil
	default:
		if isStore {
			if !memoryAlterableEA[k] {
				return fmt.Errorf("%s destination must be memory-alterable", name)
			}
			return nil
		}
		if !readableDataEA[k] {
			return fmt.Errorf("%s source must be a readable addressing mode", name)
		}
		return nil
	}
}

// newFPBinaryDef builds a two-operand FPU instruction (FMOVE/FADD/FSUB/
// FMUL/FDIV/FCMP): "<ea>,FPn" (opBase in bits 6-0, source format in bits
// 12-10, dest FPn in bits 9-7) and "FPm,FPn" (opBase, source FPm in bits
// 12-10, dest FPn in bits 9-7 — no format field, since both operands are
// already extended precision internally). allowStore adds FMOVE's third
// form, "FPn,<ea>" (bit 13 set for the store direction, per fmovex's
// 0x4800-vs-0x6800 literals — see fpFormatCode's doc comment for the
// sibling load/store literal pair this generalizes from).
func newFPBinaryDef(name string, opBase uint16, allowStore bool) *InstrDef {
	validate := func(a *Args) error {
		if err := validateFPUOperand(name, false, a.Src.Kind, a.Size); err != nil {
			return err
		}
		return validateFPUOperand(name, false, a.Dst.Kind, a.Size)
	}
	forms := []FormDef{
		{
			// <ea>,FPn
			DefaultSize: ExtendedSize,
			Sizes:       fpSizes,
			OperKinds:   []OperandKind{OpkEA, OpkFPn},
			Validate:    validate,
			Requires:    requireFPU,
			Steps: []EmitStep{
				{WordBits: 0xF000, Fields: []FieldRef{FSrcEA}},
				{WordBits: opBase, Fields: []FieldRef{FFPFormat, FFPDstReg7}},
				{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
			},
		},
		{
			// FPm,FPn
			DefaultSize: ExtendedSize,
			Sizes:       []Size{ExtendedSize},
			OperKinds:   []OperandKind{OpkFPn, OpkFPn},
			Requires:    requireFPU,
			Steps: []EmitStep{
				{WordBits: 0xF000},
				{WordBits: opBase, Fields: []FieldRef{FFPSrcReg10, FFPDstReg7}},
			},
		},
	}
	if allowStore {
		storeValidate := func(a *Args) error {
			if err := validateFPUOperand(name, false, a.Src.Kind, a.Size); err != nil {
				return err
			}
			return validateFPUOperand(name, true, a.Dst.Kind, a.Size)
		}
		forms = append(forms, FormDef{
			// FPn,<ea>
			DefaultSize: ExtendedSize,
			Sizes:       fpSizes,
			OperKinds:   []OperandKind{OpkFPn, OpkEA},
			Validate:    storeValidate,
			Requires:    requireFPU,
			Steps: []EmitStep{
				{WordBits: 0xF000, Fields: []FieldRef{FDstEA}},
				{WordBits: 0x2000 | opBase, Fields: []FieldRef{FFPFormat, FFPSrcReg7}},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		})
	}
	return &InstrDef{Mnemonic: name, Forms: forms}
}

// newFPMonadicDef builds a one-input FPU instruction (FABS/FNEG/FSQRT):
// "<ea>,FPn", "FPm,FPn", and the single-operand shorthand "FPn" (result
// in place — copySrcToDstIfNone duplicates the register into Dst before
// encoding, so it reuses the FPm,FPn form's Steps unchanged).
func newFPMonadicDef(name string, opBase uint16) *InstrDef {
	validateEA := func(a *Args) error {
		if err := validateFPUOperand(name, false, a.Src.Kind, a.Size); err != nil {
			return err
		}
		return validateFPUOperand(name, false, a.Dst.Kind, a.Size)
	}
	validateReg := func(a *Args) error {
		copySrcToDstIfNone(a)
		if a.Src.Kind != EAkFPn || a.Dst.Kind != EAkFPn {
			return fmt.Errorf("%s requires FPn register operand(s)", name)
		}
		return nil
	}
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: ExtendedSize,
				Sizes:       fpSizes,
				OperKinds:   []OperandKind{OpkEA, OpkFPn},
				Validate:    validateEA,
				Requires:    requireFPU,
				Steps: []EmitStep{
					{WordBits: 0xF000, Fields: []FieldRef{FSrcEA}},
					{WordBits: opBase, Fields: []FieldRef{FFPFormat, FFPDstReg7}},
					{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
				},
			},
			{
				DefaultSize: ExtendedSize,
				Sizes:       []Size{ExtendedSize},
				OperKinds:   []OperandKind{OpkFPn, OpkFPn},
				Validate:    validateReg,
				Requires:    requireFPU,
				Steps: []EmitStep{
					{WordBits: 0xF000},
					{WordBits: opBase, Fields: []FieldRef{FFPSrcReg10, FFPDstReg7}},
				},
			},
			{
				DefaultSize: ExtendedSize,
				Sizes:       []Size{ExtendedSize},
				OperKinds:   []OperandKind{OpkFPn},
				Validate:    validateReg,
				Requires:    requireFPU,
				Steps: []EmitStep{
					{WordBits: 0xF000},
					{WordBits: opBase, Fields: []FieldRef{FFPSrcReg10, FFPDstReg7}},
				},
			},
		},
	}
}

// copySrcToDstIfNone duplicates Src into Dst when only one operand was
// given (Dst still empty) — the single-operand shorthand for FABS/FNEG/
// FSQRT ("FABS FPn" meaning "FPn = |FPn|"), as opposed to
// swapSrcDstIfDstNone (validate.go) which moves rather than copies.
func copySrcToDstIfNone(a *Args) {
	if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
		a.Dst = a.Src
	}
}

var defFTST = InstrDef{
	Mnemonic: "FTST",
	Forms: []FormDef{
		{
			DefaultSize: ExtendedSize,
			Sizes:       fpSizes,
			OperKinds:   []OperandKind{OpkEA},
			Validate:    func(a *Args) error { return validateFPUOperand("FTST", false, a.Src.Kind, a.Size) },
			Requires:    requireFPU,
			Steps: []EmitStep{
				{WordBits: 0xF000, Fields: []FieldRef{FSrcEA}},
				{WordBits: 0x3A, Fields: []FieldRef{FFPFormat}},
				{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
			},
		},
		{
			DefaultSize: ExtendedSize,
			Sizes:       []Size{ExtendedSize},
			OperKinds:   []OperandKind{OpkFPn},
			Requires:    requireFPU,
			Steps: []EmitStep{
				{WordBits: 0xF000},
				{WordBits: 0x3A, Fields: []FieldRef{FFPSrcReg10}},
			},
		},
	},
}

var defFNOP = InstrDef{
	Mnemonic: "FNOP",
	Forms: []FormDef{
		{
			DefaultSize: ExtendedSize,
			Sizes:       []Size{ExtendedSize},
			OperKinds:   []OperandKind{},
			Requires:    requireFPU,
			Steps: []EmitStep{
				{WordBits: 0xF280},
				{WordBits: 0x0000},
			},
		},
	},
}
