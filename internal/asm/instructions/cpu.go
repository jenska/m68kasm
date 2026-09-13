package instructions

import "strings"

// CPUKind identifies the base integer-unit ISA tier an assembly targets.
// Every tier except CPU32 is a strict superset of the previous one
// (68000 -> 68010 -> 68020 -> 68030 -> 68040 -> 68060). CPU32 sits at
// its enum position (between CPU68010 and CPU68020) specifically so the
// ordinary "t.CPU >= req.CPU" comparison in Supports does the right
// thing for the *shared* subset of 68020-era additions CPU32's designers
// back-ported (Bcc.L/BSR.L, CHK2/CMP2, EXTB, TRAPcc, and scale factors
// on indexed addressing): tagging those forms Requires:Target{CPU:CPU32}
// makes them satisfied by CPU32 *and* every 68020+ tier via the plain
// ordering, with no special-casing needed. It is NOT a full 68020
// superset, though: 68020 memory-indirect addressing, bit-field ops,
// CAS/CAS2, and the coprocessor interface (FPU included) are all still
// correctly gated behind Target{CPU: CPU68020} exactly, since CPU32
// lacks them (confirmed against GNU binutils' gas/config/tc-m68k.c,
// which explicitly rejects memory-indirect addressing on cpu32 with
// "needs 68020 or higher"). The one thing plain ordering cannot express
// is a form CPU32 *exclusively* has that no other tier does (TBLS/TBLU
// and BGND) — see Supports and featCPU32Only for that case.
type CPUKind uint8

const (
	CPU68000 CPUKind = iota // baseline; also covers the 68008
	CPU68010                // also covers the 68012
	CPU32                   // 68330-family
	CPU68020
	CPU68030
	CPU68040
	CPU68060
)

func (k CPUKind) String() string {
	switch k {
	case CPU68000:
		return "68000"
	case CPU68010:
		return "68010"
	case CPU32:
		return "cpu32"
	case CPU68020:
		return "68020"
	case CPU68030:
		return "68030"
	case CPU68040:
		return "68040"
	case CPU68060:
		return "68060"
	default:
		return "unknown"
	}
}

// ParseCPUKind maps a CPU name, as accepted by the m68kasm CLI's --cpu
// flag, to a CPUKind. Matching is case-insensitive; part numbers that
// share an ISA tier with another (e.g. 68008 with 68000, 68012 with
// 68010) resolve to that tier.
func ParseCPUKind(name string) (CPUKind, bool) {
	switch strings.ToLower(name) {
	case "68000", "68008":
		return CPU68000, true
	case "68010", "68012":
		return CPU68010, true
	case "cpu32":
		return CPU32, true
	case "68020":
		return CPU68020, true
	case "68030":
		return CPU68030, true
	case "68040":
		return CPU68040, true
	case "68060":
		return CPU68060, true
	default:
		return 0, false
	}
}

// Feature is an orthogonal capability bit: two systems can share a
// CPUKind while differing in which coprocessors are attached (e.g. a
// bare 68EC020 vs. a 68020 paired with an external 68881 FPU).
type Feature uint32

const (
	// FeatFPU marks forms usable with any attached FPU (68881/68882, or
	// the reduced FPU integrated into the 68040/68060).
	FeatFPU Feature = 1 << iota
	// FeatFPUFull marks the transcendental FPU forms only a discrete
	// 68881/68882 executes natively (68040/68060 trap-emulate them) —
	// the --fpu-full CLI flag, combined with FeatFPU via requireFPUFull
	// below. Unlike FeatEmulated (a runtime-performance warning on an
	// otherwise-available form, milestone 15), this is a hard
	// availability gate: assembling FSIN and friends means asserting a
	// real discrete FPU is present, not just "some FPU, native or
	// integrated" the way plain --fpu already models.
	FeatFPUFull
	// FeatPMMU marks forms that need a paged MMU (68851, or the on-chip
	// PMMU in the 68030/68040/68060).
	FeatPMMU
	// FeatEmulated marks forms that assemble correctly but are
	// trap-and-emulate (not native) on the target, notably several
	// 68020-era forms on the 68060.
	FeatEmulated
	// featCPU32Only marks a form CPU32 has that no other tier does
	// (TBLS/TBLU, BGND) — see Supports. Unexported: unlike the other
	// Feature bits, this is never something a caller sets on a Target
	// themselves; Supports grants it implicitly to CPU32 targets. A
	// plain CPU-floor comparison can express "at or above tier X" but
	// never "exactly CPU32, not 68020+", which is why this exists.
	featCPU32Only
	// featM68020Only marks a form the plain 68020 has that was then
	// dropped from 68030 and later — CALLM/RTM, confirmed against GNU
	// binutils' include/opcode/m68k.h, which tags them with the single
	// bit `m68020` (0x004) rather than the `m68020up` union
	// (m68020|m68030up|...). Same mechanism as featCPU32Only, granted
	// implicitly to a target whose CPU is exactly CPU68020.
	featM68020Only
)

// Target bundles the base ISA tier with the coprocessor features present
// on the system being assembled for. The zero value targets a bare
// 68000 with no coprocessors, matching this assembler's behavior before
// Target existed — every existing caller sees no change in behavior.
type Target struct {
	CPU      CPUKind
	Features Feature
}

// Target68000 is the explicit spelling of the zero-value Target.
var Target68000 = Target{CPU: CPU68000}

// requireCPU32Only marks a form that only CPU32 has (TBLS/TBLU, BGND) —
// no other tier, including 68020+, supports it. Use this instead of a
// bare Target{CPU: CPU32}, which (via Supports' plain CPU-floor
// comparison) means "CPU32 and every 68020+ tier" — the right choice for
// the 68020-era additions CPU32 shares with 68020+ (Bcc.L, CHK2/CMP2,
// EXTB, TRAPcc, scale factors), but the wrong one for a form CPU32 alone
// implements.
var requireCPU32Only = Target{CPU: CPU32, Features: featCPU32Only}

// requireM68020Only marks a form the plain 68020 has that 68030 and
// later dropped (CALLM/RTM). Use this instead of a bare
// Target{CPU: CPU68020}, which means "68020 and every later tier" — the
// right choice for forms 68030+ kept, but the wrong one for a form only
// the original 68020 has.
var requireM68020Only = Target{CPU: CPU68020, Features: featM68020Only}

// Supports reports whether a form or instruction tagged with the
// requirement req is legal to assemble for this Target. A CPU32 target
// implicitly carries featCPU32Only, and a target whose CPU is exactly
// CPU68020 implicitly carries featM68020Only (no caller ever sets either
// themselves) — so a form tagged requireCPU32Only or requireM68020Only
// is satisfied only by that exact tier, even though a plain CPU-floor
// comparison alone would also (wrongly) grant it to every later tier
// too, since both have strictly higher CPU values. Every other
// requirement uses the ordinary rule: req.CPU must be at or below t.CPU,
// and every other Feature bit set in req.Features must also be set in
// t.Features.
func (t Target) Supports(req Target) bool {
	effFeatures := t.Features
	switch t.CPU {
	case CPU32:
		effFeatures |= featCPU32Only
	case CPU68020:
		effFeatures |= featM68020Only
	}
	if req.Features&^effFeatures != 0 {
		return false
	}
	return t.CPU >= req.CPU
}
