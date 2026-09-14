package instructions

// This file adds the FPU's "math extensions" — FGETEXP/FGETMAN
// (extract the exponent/mantissa of a floating value) and
// FSCALE/FMOD/FREM (scale by a power of two, and the two IEEE
// remainder operations) — the group the design doc's own accounting
// (§6) has always tracked separately from "the transcendental set"
// (cpu020_fpu_trans.go), even though all five share GAS's identical
// "IiF8F7" opcode-table argument shape. Chosen by the maintainer from
// the open items list after milestone 23.
//
// FGETEXP and FGETMAN are monadic — the exact shape FABS/FNEG/FSQRT
// already use — so newFPMonadicDef is reused directly. FSCALE/FMOD/
// FREM are genuinely binary (two floating operands, e.g. "FSCALE
// FPm,FPn" scales FPn by FPm, matching FADD's own shape, not FABS's),
// so newFPBinaryDef is reused instead, with allowStore=false: like
// FADD/FSUB/etc. (and unlike FMOVE), the result only ever goes to an
// FPn register, never out to memory.
//
// Gated requireFPUFull, the same as the transcendental set: like those,
// these are specialized math operations beyond the basic move/
// arithmetic/compare set a 68040/68060's integrated FPU implements
// natively, so assembling any of them means asserting a real discrete
// 68881/68882 the same way FSIN and friends already do.
//
// opBase values were decoded from GNU binutils' GAS m68k opcode table
// (opcodes/m68k-opc.c)'s "fgetexpx"/"fgetmanx"/"fscalex"/"fmodx"/
// "fremx" rows.
func init() {
	registerInstrDef(newFPMonadicDef("FGETEXP", 0x1E, requireFPUFull))
	registerInstrDef(newFPMonadicDef("FGETMAN", 0x1F, requireFPUFull))
	registerInstrDef(newFPBinaryDef("FSCALE", 0x26, false, requireFPUFull))
	registerInstrDef(newFPBinaryDef("FMOD", 0x21, false, requireFPUFull))
	registerInstrDef(newFPBinaryDef("FREM", 0x25, false, requireFPUFull))
}
