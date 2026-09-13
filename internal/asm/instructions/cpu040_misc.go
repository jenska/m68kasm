package instructions

import "fmt"

// This file adds the first slice of 68040/68060-only integer
// instructions: MOVE16 (16-byte cache-line-aligned block move) and the
// cache-control instructions CINVA/CINVL/CINVP and CPUSHA/CPUSHL/CPUSHP
// (invalidate/push a cache line, page, or the whole cache). All are
// gated Target{CPU: CPU68040}, which — via Supports' plain CPU-floor
// comparison — grants both 68040 and 68060 (CPU68060 sits above
// CPU68040 in CPUKind's enum order), matching GNU binutils' own
// m68040up tag (`m68040 | m68060`, include/opcode/m68k.h).
//
// The warning mechanism for 68060's trap-emulated subset (CAS2,
// CHK2/CMP2, MOVEP, dynamic-offset/width BFxxx, and the 64-bit Dr:Dq
// form of DIVSL/DIVUL) lives separately, in internal/asm/emulated060.go
// (checkEmulatedOn68060) and Program.Warnings (assemble.go) — not in
// this package at all, and not wired into the FeatEmulated bit's
// Requires/Supports gating. It has to be a per-Args runtime check
// rather than a static per-Form flag: whether a given BFxxx or
// DIVSL/DIVUL use is emulated depends on which operand shape was
// actually used (a register-specified bit-field offset/width, or the
// wide 64-bit dividend form), not on which Form matched — the same
// Form handles both the emulated and the native shape. CINV/CPUSH's own
// scope, memory-management unit interactions, and 68040/68060-specific
// exception-frame layout details are out of scope for an assembler in
// the same way real trap/exception semantics generally are throughout
// this codebase.

var require68040 = Target{CPU: CPU68040}

func init() {
	registerInstrDef(&defMOVE16)
	registerInstrDef(newCInvCPushDef("CINVA", 0xF400|scopeAll, false))
	registerInstrDef(newCInvCPushDef("CINVL", 0xF400|scopeLine, true))
	registerInstrDef(newCInvCPushDef("CINVP", 0xF400|scopePage, true))
	registerInstrDef(newCInvCPushDef("CPUSHA", 0xF420|scopeAll, false))
	registerInstrDef(newCInvCPushDef("CPUSHL", 0xF420|scopeLine, true))
	registerInstrDef(newCInvCPushDef("CPUSHP", 0xF420|scopePage, true))
}

// scopeLine/scopePage/scopeAll are CINV/CPUSH's "scope" field (bits
// 4-3), matching GAS's own SCOPE_LINE/SCOPE_PAGE/SCOPE_ALL macros
// (opcodes/m68k-opc.c). There is no SCOPE_NONE opcode variant — "no
// scope" is instead expressed as the NC cache-selector operand value,
// a completely different field (bits 7-6).
const (
	scopeLine = 0x1 << 3
	scopePage = 0x2 << 3
	scopeAll  = 0x3 << 3
)

var defMOVE16 = InstrDef{
	Mnemonic: "MOVE16",
	Forms: []FormDef{
		{
			// MOVE16 (An1)+,(An2)+ — the common case, both pointers
			// auto-incrementing by 16 after the move.
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OpkPostincAn, OpkPostincAn},
			Requires:    require68040,
			Steps: []EmitStep{
				{WordBits: 0xF620, Fields: []FieldRef{FSrcDnReg}},
				{WordBits: 0x8000, Fields: []FieldRef{FMove16Reg2_12}},
			},
		},
		{
			// MOVE16 (An)+,<absolute long>
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OpkPostincAn, OpkEA},
			Validate:    validateMove16AbsDst,
			Requires:    require68040,
			Steps: []EmitStep{
				{WordBits: 0xF600, Fields: []FieldRef{FSrcDnReg}},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		},
		{
			// MOVE16 <absolute long>,(An)+
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OpkEA, OpkPostincAn},
			Validate:    validateMove16AbsSrc,
			Requires:    require68040,
			Steps: []EmitStep{
				{WordBits: 0xF608, Fields: []FieldRef{FDstRegLow}},
				{Trailer: []TrailerItem{TSrcEAExt}},
			},
		},
		{
			// MOVE16 (An),<absolute long> or MOVE16 <absolute
			// long>,(An) — one Form for both directions.
			// FMove16AbsForm picks word1 (0xF610 vs 0xF618) and which
			// operand's register to place from which side is actually
			// (An) at runtime; see its doc comment in types.go for why
			// two Forms here would be ambiguous (both directions
			// classify as plain [OpkEA, OpkEA] — neither (An) nor an
			// absolute address has its own OperandKind distinguishing
			// which side is which the way -(An)/(An)+ do).
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OpkEA, OpkEA},
			Validate:    validateMove16AbsEitherOrder,
			Requires:    require68040,
			Steps: []EmitStep{
				{Fields: []FieldRef{FMove16AbsForm}},
				{Trailer: []TrailerItem{TSrcEAExt, TDstEAExt}},
			},
		},
	},
}

func validateMove16AbsDst(a *Args) error {
	if a.Dst.Kind != EAkAbsL {
		return fmt.Errorf("MOVE16 requires an absolute long address")
	}
	return nil
}

func validateMove16AbsSrc(a *Args) error {
	if a.Src.Kind != EAkAbsL {
		return fmt.Errorf("MOVE16 requires an absolute long address")
	}
	return nil
}

func validateMove16AbsEitherOrder(a *Args) error {
	if a.Src.Kind == EAkAddrInd && a.Dst.Kind == EAkAbsL {
		return nil
	}
	if a.Src.Kind == EAkAbsL && a.Dst.Kind == EAkAddrInd {
		return nil
	}
	return fmt.Errorf("MOVE16 requires (An) and an absolute long address, in either order")
}

// newCInvCPushDef builds one CINV/CPUSH mnemonic. hasAn selects between
// the "all" scope (cache selector only, e.g. CINVA) and the line/page
// scopes (cache selector plus an (An) operand, e.g. CINVL/CINVP) —
// GAS's own "ce" vs "ceas" argument-string split (opcodes/m68k-opc.c).
func newCInvCPushDef(name string, base uint16, hasAn bool) *InstrDef {
	if !hasAn {
		return &InstrDef{
			Mnemonic: name,
			Forms: []FormDef{
				{
					DefaultSize: WordSize,
					Sizes:       []Size{WordSize},
					OperKinds:   []OperandKind{OpkCacheSel},
					Requires:    require68040,
					Steps: []EmitStep{
						{WordBits: base, Fields: []FieldRef{FCacheSel6}},
					},
				},
			},
		}
	}
	validate := func(a *Args) error {
		if a.Dst.Kind != EAkAddrInd {
			return fmt.Errorf("%s requires (An)", name)
		}
		return nil
	}
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkCacheSel, OpkEA},
				Validate:    validate,
				Requires:    require68040,
				Steps: []EmitStep{
					{WordBits: base, Fields: []FieldRef{FCacheSel6, FDstRegLow}},
				},
			},
		},
	}
}
