package asm

import "github.com/jenska/m68kasm/internal/asm/instructions"

// checkEmulatedOn68060 reports whether ins uses one of the handful of
// 68020-era forms that a real 68060 does not execute natively — it
// traps and the kernel's exception handler emulates the instruction in
// software, a real and often large performance cliff a caller may want
// to know about even though the instruction assembles correctly and
// runs correctly. This is a per-Args runtime check rather than a static
// per-Form flag (unlike the CPU-tier/feature gating elsewhere in this
// package, which Target.Supports resolves purely from an InstrDef's
// Requires): CAS2/CHK2/CMP2/MOVEP are unconditionally emulated whenever
// they appear at all, but the 68020 bit-field instructions and
// DIVSL/DIVUL are emulated only for specific *operand shapes* (a
// register-specified bit-field offset/width, or the 64-bit Dr:Dq
// dividend form) that a static Form-level flag can't express — the same
// Form handles both the emulated and the native shape.
//
// Confirmed against the 68060 user's manual's software-emulated
// instruction list; this is deliberately not wired into the
// CPUKind/Feature Requires/Supports gating mechanism at all (these
// forms are not *unavailable* on a 68060 — they assemble and execute
// correctly — so blocking them the way an unsupported CPU tier would be
// wrong; a warning, not an error, is the correct signal here).
func checkEmulatedOn68060(mnemonic string, args *instructions.Args) string {
	switch mnemonic {
	case "CAS2":
		return "CAS2 traps and is software-emulated on 68060 (expect a large performance cliff)"
	case "CHK2", "CMP2":
		return mnemonic + " traps and is software-emulated on 68060 (expect a large performance cliff)"
	case "MOVEP":
		return "MOVEP traps and is software-emulated on 68060 (expect a large performance cliff)"
	case "BFTST", "BFCHG", "BFCLR", "BFSET", "BFEXTU", "BFEXTS", "BFFFO", "BFINS":
		if args.Src.BFOffsetIsReg || args.Src.BFWidthIsReg || args.Dst.BFOffsetIsReg || args.Dst.BFWidthIsReg {
			return mnemonic + " with a register-specified bit-field offset/width traps and is software-emulated on 68060 (expect a large performance cliff)"
		}
	case "DIVSL", "DIVUL":
		if args.Dst.RegPairWide {
			return mnemonic + " with a 64-bit (Dr:Dq) dividend traps and is software-emulated on 68060 (expect a large performance cliff)"
		}
	}
	return ""
}
