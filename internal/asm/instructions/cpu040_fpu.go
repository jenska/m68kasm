package instructions

// This file adds the MC68040 FPU instructions that round their result to
// single or double precision regardless of the FPCR rounding precision:
// FSMOVE/FDMOVE, FSSQRT/FDSQRT, FSABS/FDABS, FSNEG/FDNEG, FSDIV/FDDIV,
// FSADD/FDADD, FSMUL/FDMUL and FSSUB/FDSUB (MC68040 User's Manual, M68000
// PRM). They take the same operand forms and the same encoding as the
// general instruction they round (FADD, FSQRT, ...); only the opmode in
// bits 6-0 of the command word differs.
//
// The 68881/68882 have no such opcodes (the coprocessor takes them as
// line-F), so every form is gated on the 68040 CPU tier in addition to
// FeatFPU. Supports' CPU-floor comparison grants the 68060 as well, which
// executes them too.

var require68040FPU = Target{CPU: CPU68040, Features: FeatFPU}

func init() {
	for _, op := range []struct {
		name   string
		opmode uint16
	}{
		{"FSMOVE", 0x40}, {"FDMOVE", 0x44},
		{"FSDIV", 0x60}, {"FDDIV", 0x64},
		{"FSADD", 0x62}, {"FDADD", 0x66},
		{"FSMUL", 0x63}, {"FDMUL", 0x67},
		{"FSSUB", 0x68}, {"FDSUB", 0x6C},
	} {
		registerInstrDef(newFPBinaryDef(op.name, op.opmode, false, require68040FPU))
	}
	for _, op := range []struct {
		name   string
		opmode uint16
	}{
		{"FSSQRT", 0x41}, {"FDSQRT", 0x45},
		{"FSABS", 0x58}, {"FDABS", 0x5C},
		{"FSNEG", 0x5A}, {"FDNEG", 0x5E},
	} {
		registerInstrDef(newFPMonadicDef(op.name, op.opmode, require68040FPU))
	}
}
