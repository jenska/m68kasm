package m68kasm

import "github.com/jenska/m68kasm/internal/asm/instructions"

// CPUKind identifies the base 68k integer-unit ISA tier to assemble for.
// See ParseCPUKind and the CPU68xxx/CPU32 constants.
type CPUKind = instructions.CPUKind

// Feature is an orthogonal capability bit for coprocessors (FPU, PMMU)
// that can be present independently of the CPUKind tier.
type Feature = instructions.Feature

// Target selects the CPU tier and coprocessor features that ParseOptions
// should assemble for. The zero value targets a bare 68000, matching
// this package's behavior before Target existed.
type Target = instructions.Target

// Target68000 is the explicit spelling of the zero-value Target.
var Target68000 = instructions.Target68000

const (
	CPU68000 = instructions.CPU68000 // also covers the 68008
	CPU68010 = instructions.CPU68010 // also covers the 68012
	CPU32    = instructions.CPU32
	CPU68020 = instructions.CPU68020
	CPU68030 = instructions.CPU68030
	CPU68040 = instructions.CPU68040
	CPU68060 = instructions.CPU68060
)

const (
	// FeatFPU enables the FPU instructions — FMOVE/FADD/FSUB/FMUL/FDIV/
	// FCMP/FABS/FNEG/FSQRT/FTST, floating-point immediate literals
	// (e.g. "#3.14"), all seven data formats including packed BCD
	// (.p), the F-condition branch/set/trap family, FMOVECR,
	// FSAVE/FRESTORE, and FMOVEM (both the FPn data-register list and
	// the FPCR/FPSR/FPIAR control-register list) — see the --fpu CLI
	// flag. Independent of CPUKind: a bare 68020 with an external
	// 68881/68882 and a 68040's built-in FPU both just need this bit
	// set.
	FeatFPU = instructions.FeatFPU
	// FeatFPUFull additionally enables the transcendental function set
	// (FSIN, FCOS, FSINCOS, FLOGN, …) and math extensions (FGETEXP,
	// FGETMAN, FSCALE, FMOD, FREM) — see the --fpu-full CLI flag, which
	// also sets FeatFPU. A hard availability gate, not a performance
	// hint: it models "a real discrete 68881/68882 is present," since a
	// 68040/68060's integrated FPU only trap-emulates this subset.
	FeatFPUFull = instructions.FeatFPUFull
	// FeatPMMU enables the PMMU instructions: PMOVE (every 68851/68030
	// register — TC, CRP/SRP/DRP, TT0/TT1, MMUSR/PSR, CAL/VAL/SCC, AC,
	// PCSR, BAD0-7/BAC0-7), PFLUSHA/PFLUSH/PFLUSHS/PFLUSHR, PLOADR/
	// PLOADW, PTESTR/PTESTW, PSAVE/PRESTORE, PMOVEFD, the P-condition
	// branch/set/trap family, and (on --cpu 68040/68060) the 68040's
	// own simplified single-word PFLUSHA/PFLUSHAN/PFLUSHN/PFLUSH/
	// PTESTR/PTESTW forms — see the --mmu CLI flag. Independent of
	// CPUKind, like FeatFPU: a bare 68020 with an external 68851 and a
	// 68030's on-chip PMMU both just need this bit set.
	FeatPMMU = instructions.FeatPMMU
	// FeatEmulated is reserved and currently unused by this package's
	// own gating: the CLI's 68060 trap-emulation warning (several
	// 68020-era forms — CAS2, CHK2/CMP2, MOVEP, dynamic-offset/width
	// BFxxx, and DIVSL/DIVUL's 64-bit Dr:Dq form — assemble correctly
	// on a 68060 target but only trap-emulate, not execute natively)
	// already works automatically for any --cpu 68060 target, with no
	// flag needed — it's implemented as a separate runtime check against
	// the actual operand shapes used, not as a Requires/Supports gate on
	// this bit, since the same Form often handles both the emulated and
	// the native shape.
	FeatEmulated = instructions.FeatEmulated
)

// ParseCPUKind maps a CPU name, as accepted by the m68kasm CLI's --cpu
// flag (e.g. "68000", "68010", "cpu32", "68020"), to a CPUKind.
func ParseCPUKind(name string) (CPUKind, bool) {
	return instructions.ParseCPUKind(name)
}
