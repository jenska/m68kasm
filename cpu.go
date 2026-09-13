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
	// FeatFPU enables the FPU instructions (FMOVE, FADD, FABS, …) — see
	// the --fpu CLI flag. Independent of CPUKind: a bare 68020 with an
	// external 68881/68882 and a 68040's built-in FPU both just need
	// this bit set.
	FeatFPU = instructions.FeatFPU
	// FeatFPUFull is reserved for the discrete 68881/68882's full
	// transcendental function set, not yet implemented.
	FeatFPUFull = instructions.FeatFPUFull
	// FeatPMMU enables the PMMU instructions (currently just PMOVE's TC
	// form and PFLUSHA — see the --mmu CLI flag). Independent of
	// CPUKind, like FeatFPU: a bare 68020 with an external 68851 and a
	// 68030's on-chip PMMU both just need this bit set.
	FeatPMMU = instructions.FeatPMMU
	// FeatEmulated is reserved for marking 68060 trap-emulated forms,
	// not yet implemented.
	FeatEmulated = instructions.FeatEmulated
)

// ParseCPUKind maps a CPU name, as accepted by the m68kasm CLI's --cpu
// flag (e.g. "68000", "68010", "cpu32", "68020"), to a CPUKind.
func ParseCPUKind(name string) (CPUKind, bool) {
	return instructions.ParseCPUKind(name)
}
