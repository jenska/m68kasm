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

// requireFPUFull gates the transcendental function set
// (cpu020_fpu_trans.go): both FeatFPU (some FPU attached) and
// FeatFPUFull (--fpu-full — a real discrete 68881/68882, not just an
// integrated/reduced one) must be set.
var requireFPUFull = Target{Features: FeatFPU | FeatFPUFull}

// fpuWord1Base is the fixed high nibble/coprocessor-ID portion of every
// FPU "general instruction" word1 (the word carrying <ea>, before the
// opmode/format word). It is 0xF200, not the more obvious-looking
// 0xF000: bits 15-12 = 1111 (the coprocessor-instruction line), and bits
// 11-9 = the coprocessor ID, which GAS's tc-m68k.c always synthesizes as
// COP1 (value 1, i.e. bits 11-9 = 001 = 0x0200) for every plain float
// mnemonic — see m68k_ip's "fake a first entry of type COP#1" comment,
// and install_operand's case 'i' (`opcode[0] |= val << 9`). This was
// originally shipped as a bare 0xF000 (cpID = 0) and silently wrong:
// GAS's disassembler confirms 1 is the expected default by only
// printing "(cpid=N)" when N isn't 1, and FNOP's own literal opcode
// (0xF280, unaffected since it's a fully-fixed word already copied
// verbatim from the table) has the same 0x0200 bit set, which is what
// exposed the discrepancy. Found and fixed while researching FBcc's
// encoding for the FPU-conditional-branch milestone — see
// docs/design/cpu-family-support.md and TestFPUCoprocessorIDBit.
const fpuWord1Base = 0xF200

// defFMOVE is captured in a package-level var, unlike every other
// newFPBinaryDef call, so cpu020_fpu_packed.go can append FMOVE.P's own
// dedicated store forms (with their k-factor operand) to defFMOVE.Forms
// directly — the same direct-variable-reference pattern
// cpu030_pmmu_xlate.go/cpu030_pmmu_access.go already use to extend defPMOVE,
// preferred over an Instructions["FMOVE"] map lookup precisely because
// it carries no init-ordering dependency between files.
var defFMOVE = newFPBinaryDef("FMOVE", 0x00, true, requireFPU)

func init() {
	registerInstrDef(defFMOVE)
	registerInstrDef(newFPBinaryDef("FADD", 0x22, false, requireFPU))
	registerInstrDef(newFPBinaryDef("FSUB", 0x28, false, requireFPU))
	registerInstrDef(newFPBinaryDef("FMUL", 0x23, false, requireFPU))
	registerInstrDef(newFPBinaryDef("FDIV", 0x20, false, requireFPU))
	registerInstrDef(newFPBinaryDef("FCMP", 0x38, false, requireFPU))
	registerInstrDef(newFPMonadicDef("FABS", 0x18, requireFPU))
	registerInstrDef(newFPMonadicDef("FNEG", 0x1A, requireFPU))
	registerInstrDef(newFPMonadicDef("FSQRT", 0x04, requireFPU))
	registerInstrDef(&defFTST)
	registerInstrDef(&defFNOP)
}

// fpSizes are the sizes every plain FPU arithmetic/move form accepts:
// integer conversions (.b/.w/.l) and the two supported floating formats
// (.s/.d/.x — packed BCD, .p, is not implemented).
var fpSizes = []Size{ByteSize, WordSize, LongSize, SingleSize, DoubleSize, ExtendedSize}

// fpLoadSizes is fpSizes plus PackedSize — every <ea>-as-source FPU form
// accepts packed BCD as an input format (the FPU converts it to
// extended precision like any other format), but the *store* direction
// deliberately does not: real hardware requires a k-factor operand for
// a packed destination (see cpu020_fpu_packed.go's own FMOVE.P forms),
// which newFPBinaryDef's shared store form has no syntax for — letting
// it accept PackedSize too would silently encode a k-factor of 0 rather
// than erroring or asking for one.
var fpLoadSizes = append(append([]Size{}, fpSizes...), PackedSize)

func isFPIntSize(sz Size) bool {
	return sz == ByteSize || sz == WordSize || sz == LongSize
}

// validateFPUOperand checks one <ea>/register operand of an FPU
// instruction. isStore means this operand is being written (the
// destination of FMOVE FPn,<ea>); it is always false for every other
// instruction in this file, since none of them can store to memory.
// Takes the full EAExpr, not just its Kind, because the EAkImm case
// needs ImmIsFloat to decide whether a floating-point literal is valid
// for the given size.
func validateFPUOperand(name string, isStore bool, e EAExpr, sz Size) error {
	switch e.Kind {
	case EAkFPn:
		return nil
	case EAkAn:
		return fmt.Errorf("%s does not support address register operands", name)
	case EAkImm:
		if isStore {
			return fmt.Errorf("%s destination cannot be immediate", name)
		}
		if sz == PackedSize {
			return fmt.Errorf("%s: packed BCD immediate literals are not supported (use a memory operand)", name)
		}
		if e.ImmIsFloat && isFPIntSize(sz) {
			return fmt.Errorf("%s: a floating-point literal immediate requires a floating-point size (.s/.d/.x), not an integer size", name)
		}
		return nil
	case EAkDn:
		if !isFPIntSize(sz) {
			return fmt.Errorf("%s: a floating-point size requires a memory operand, not Dn", name)
		}
		return nil
	default:
		if isStore {
			if !memoryAlterableEA[e.Kind] {
				return fmt.Errorf("%s destination must be memory-alterable", name)
			}
			return nil
		}
		if !readableDataEA[e.Kind] {
			return fmt.Errorf("%s source must be a readable addressing mode", name)
		}
		return nil
	}
}

// newFPBinaryDef builds a two-operand FPU instruction (FMOVE/FADD/FSUB/
// FMUL/FDIV/FCMP, and — with requires set to requireFPUFull —
// FSCALE/FMOD/FREM, cpu020_fpu_mathext.go): "<ea>,FPn" (opBase in bits
// 6-0, source format in bits 12-10, dest FPn in bits 9-7) and "FPm,FPn"
// (opBase, source FPm in bits 12-10, dest FPn in bits 9-7 — no format
// field, since both operands are already extended precision
// internally). allowStore adds FMOVE's third form, "FPn,<ea>" (bit 13
// set for the store direction, per fmovex's 0x4800-vs-0x6800 literals
// — see fpFormatCode's doc comment for the sibling load/store literal
// pair this generalizes from). requires is a parameter for the same
// reason newFPMonadicDef's is — see that function's own doc comment.
func newFPBinaryDef(name string, opBase uint16, allowStore bool, requires Target) *InstrDef {
	validate := func(a *Args) error {
		if err := validateFPUOperand(name, false, a.Src, a.Size); err != nil {
			return err
		}
		return validateFPUOperand(name, false, a.Dst, a.Size)
	}
	forms := []FormDef{
		{
			// <ea>,FPn
			DefaultSize:   ExtendedSize,
			Sizes:         fpLoadSizes,
			OperKinds:     []OperandKind{OpkEA, OpkFPn},
			Validate:      validate,
			Requires:      requires,
			AllowFloatImm: true,
			Steps: []EmitStep{
				{WordBits: fpuWord1Base, Fields: []FieldRef{FSrcEA}},
				{WordBits: opBase, Fields: []FieldRef{FFPFormat, FFPDstReg7}},
				{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
			},
		},
		{
			// FPm,FPn
			DefaultSize: ExtendedSize,
			Sizes:       []Size{ExtendedSize},
			OperKinds:   []OperandKind{OpkFPn, OpkFPn},
			Requires:    requires,
			Steps: []EmitStep{
				{WordBits: fpuWord1Base},
				{WordBits: opBase, Fields: []FieldRef{FFPSrcReg10, FFPDstReg7}},
			},
		},
	}
	if allowStore {
		storeValidate := func(a *Args) error {
			if err := validateFPUOperand(name, false, a.Src, a.Size); err != nil {
				return err
			}
			return validateFPUOperand(name, true, a.Dst, a.Size)
		}
		forms = append(forms, FormDef{
			// FPn,<ea>
			DefaultSize: ExtendedSize,
			Sizes:       fpSizes,
			OperKinds:   []OperandKind{OpkFPn, OpkEA},
			Validate:    storeValidate,
			Requires:    requires,
			Steps: []EmitStep{
				{WordBits: fpuWord1Base, Fields: []FieldRef{FDstEA}},
				{WordBits: 0x2000 | opBase, Fields: []FieldRef{FFPFormat, FFPSrcReg7}},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		})
	}
	return &InstrDef{Mnemonic: name, Forms: forms}
}

// newFPMonadicDef builds a one-input FPU instruction (FABS/FNEG/FSQRT,
// and — with requires set to requireFPUFull — the transcendental
// function set, cpu020_fpu_trans.go): "<ea>,FPn", "FPm,FPn", and the
// single-operand shorthand "FPn" (result in place — copySrcToDstIfNone
// duplicates the register into Dst before encoding, so it reuses the
// FPm,FPn form's Steps unchanged). requires is a parameter, not always
// requireFPU, because a discrete 68881/68882 executes the
// transcendental subset natively while a bare "some FPU is attached"
// target (FeatFPU alone) may not — see requireFPUFull's own doc comment.
func newFPMonadicDef(name string, opBase uint16, requires Target) *InstrDef {
	validateEA := func(a *Args) error {
		if err := validateFPUOperand(name, false, a.Src, a.Size); err != nil {
			return err
		}
		return validateFPUOperand(name, false, a.Dst, a.Size)
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
				DefaultSize:   ExtendedSize,
				Sizes:         fpLoadSizes,
				OperKinds:     []OperandKind{OpkEA, OpkFPn},
				Validate:      validateEA,
				Requires:      requires,
				AllowFloatImm: true,
				Steps: []EmitStep{
					{WordBits: fpuWord1Base, Fields: []FieldRef{FSrcEA}},
					{WordBits: opBase, Fields: []FieldRef{FFPFormat, FFPDstReg7}},
					{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
				},
			},
			{
				DefaultSize: ExtendedSize,
				Sizes:       []Size{ExtendedSize},
				OperKinds:   []OperandKind{OpkFPn, OpkFPn},
				Validate:    validateReg,
				Requires:    requires,
				Steps: []EmitStep{
					{WordBits: fpuWord1Base},
					{WordBits: opBase, Fields: []FieldRef{FFPSrcReg10, FFPDstReg7}},
				},
			},
			{
				DefaultSize: ExtendedSize,
				Sizes:       []Size{ExtendedSize},
				OperKinds:   []OperandKind{OpkFPn},
				Validate:    validateReg,
				Requires:    requires,
				Steps: []EmitStep{
					{WordBits: fpuWord1Base},
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
			DefaultSize:   ExtendedSize,
			Sizes:         fpLoadSizes,
			OperKinds:     []OperandKind{OpkEA},
			Validate:      func(a *Args) error { return validateFPUOperand("FTST", false, a.Src, a.Size) },
			Requires:      requireFPU,
			AllowFloatImm: true,
			Steps: []EmitStep{
				{WordBits: fpuWord1Base, Fields: []FieldRef{FSrcEA}},
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
				{WordBits: fpuWord1Base},
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
