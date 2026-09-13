package instructions

// This file adds the FPU's transcendental function set — deferred since
// the original FPU milestone (cpu020_fpu.go), chosen by the maintainer
// from the open items list. Every one of these is a plain monadic FPU
// instruction ("<ea>,FPn", "FPm,FPn", or the single-operand shorthand
// "FPn") — the exact same shape FABS/FNEG/FSQRT already use — so
// newFPMonadicDef (cpu020_fpu.go) is reused directly, just with
// requireFPUFull instead of requireFPU: a discrete 68881/68882
// executes these natively, while a bare "some FPU is attached" target
// (FeatFPU alone, matching an integrated/reduced FPU on 68040/68060)
// may not, so assembling any of these means asserting FeatFPUFull too
// (the new --fpu-full CLI flag).
//
// opBase values were decoded from GNU binutils' GAS m68k opcode table
// (opcodes/m68k-opc.c)'s "IiF8F7"/"Ii;xF7"-style rows for each
// mnemonic's extended-precision form (e.g. "fsinx", "fatanx"), the
// same argument shape FABS/FNEG/FSQRT already use and this file's own
// header comment cross-checks against.
//
// Deliberately NOT included:
//   - FSINCOS. Its own row ("Ii;bF3F7") has TWO destination fields
//     (F7 at bits 9-7, F3 at bits 2-0) — a genuinely different,
//     dual-result shape ("FSINCOS <ea>,FPcos:FPsin") needing its own
//     colon-joined register-pair destination syntax, not a variant of
//     the single-destination shape every other function here uses.
//   - FGETEXP, FGETMAN, FSCALE, FMOD, FREM. These share the same
//     "IiF8F7" bit layout (and FSCALE/FMOD/FREM are genuinely binary,
//     not monadic — real two-FPn-operand instructions, matching
//     FADD/FSUB's own shape rather than FABS/FNEG's), but the design
//     doc's own accounting keeps them a separate "FPU math extensions"
//     bucket distinct from "the transcendental set" this milestone
//     was scoped to — left for a future milestone rather than folded
//     in here.
//
// See docs/design/cpu-family-support.md.
func init() {
	registerInstrDef(newFPMonadicDef("FSIN", 0x0E, requireFPUFull))
	registerInstrDef(newFPMonadicDef("FCOS", 0x1D, requireFPUFull))
	registerInstrDef(newFPMonadicDef("FTAN", 0x0F, requireFPUFull))
	registerInstrDef(newFPMonadicDef("FATAN", 0x0A, requireFPUFull))
	registerInstrDef(newFPMonadicDef("FASIN", 0x0C, requireFPUFull))
	registerInstrDef(newFPMonadicDef("FACOS", 0x1C, requireFPUFull))
	registerInstrDef(newFPMonadicDef("FATANH", 0x0D, requireFPUFull))
	registerInstrDef(newFPMonadicDef("FSINH", 0x02, requireFPUFull))
	registerInstrDef(newFPMonadicDef("FCOSH", 0x19, requireFPUFull))
	registerInstrDef(newFPMonadicDef("FTANH", 0x09, requireFPUFull))
	registerInstrDef(newFPMonadicDef("FETOX", 0x10, requireFPUFull))
	registerInstrDef(newFPMonadicDef("FETOXM1", 0x08, requireFPUFull))
	registerInstrDef(newFPMonadicDef("FLOGN", 0x14, requireFPUFull))
	registerInstrDef(newFPMonadicDef("FLOGNP1", 0x06, requireFPUFull))
	registerInstrDef(newFPMonadicDef("FLOG10", 0x15, requireFPUFull))
	registerInstrDef(newFPMonadicDef("FLOG2", 0x16, requireFPUFull))
	registerInstrDef(newFPMonadicDef("FTWOTOX", 0x11, requireFPUFull))
	registerInstrDef(newFPMonadicDef("FTENTOX", 0x12, requireFPUFull))
}
