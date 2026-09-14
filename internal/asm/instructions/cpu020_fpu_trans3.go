package instructions

// This file adds FSINCOS, the FPU transcendental function deliberately
// deferred out of cpu020_fpu_trans.go's own milestone (see that file's
// own header comment) for needing a genuinely different shape: every
// other transcendental function is monadic (one input, one output —
// newFPMonadicDef's exact shape), but FSINCOS computes sine and cosine
// simultaneously from one input, writing two *different* FPn registers
// at once. Chosen by the maintainer from the open items list after
// milestone 25 to close out that last deferred FPU function.
//
// Cross-checked against GNU binutils' GAS m68k opcode table
// (opcodes/m68k-opc.c):
//
//	{"fsincosx", two(0xF000, 0x0030), two(0xF1C0, 0xE078), "IiF8F3F7", mfloat },
//	{"fsincosx", two(0xF000, 0x4830), two(0xF1C0, 0xFC78), "Ii;xF3F7", mfloat },
//
// Same "Ii" coprocessor-ID prefix and "F8"/"F3"/"F7" register-field
// letters every other FPU instruction uses (see fpuWord1Base's doc
// comment and newFPBinaryDef's own "F8F7" shape) — F8 (bits 12-10) is
// the source FPm exactly like FADD's own source field; F7 (bits 9-7) is
// the *sine* result, sharing the same bit position FFPDstReg7 already
// uses for every other instruction's single destination (matching the
// mnemonic's leading word); F3 (bits 2-0, a field no other instruction
// in this codebase's FPU set uses) is the *cosine* result. Motorola's
// own documented mnemonic syntax, "FSINCOS <ea>,FPc:FPs" (cosine
// written first, sine second), confirms the field/position mapping.
func init() {
	registerInstrDef(&defFSINCOS)
}

var defFSINCOS = InstrDef{
	Mnemonic: "FSINCOS",
	Forms: []FormDef{
		{
			// <ea>,FPc:FPs
			DefaultSize:   ExtendedSize,
			Sizes:         fpLoadSizes,
			OperKinds:     []OperandKind{OpkEA, OpkFPRegPair},
			Validate:      func(a *Args) error { return validateFPUOperand("FSINCOS", false, a.Src, a.Size) },
			Requires:      requireFPUFull,
			AllowFloatImm: true,
			Steps: []EmitStep{
				{WordBits: fpuWord1Base, Fields: []FieldRef{FSrcEA}},
				{WordBits: 0x0030, Fields: []FieldRef{FFPFormat, FSincosRegSin7, FSincosRegCos0}},
				{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
			},
		},
		{
			// FPm,FPc:FPs
			DefaultSize: ExtendedSize,
			Sizes:       []Size{ExtendedSize},
			OperKinds:   []OperandKind{OpkFPn, OpkFPRegPair},
			Requires:    requireFPUFull,
			Steps: []EmitStep{
				{WordBits: fpuWord1Base},
				{WordBits: 0x0030, Fields: []FieldRef{FFPSrcReg10, FSincosRegSin7, FSincosRegCos0}},
			},
		},
	},
}
