package instructions

import "fmt"

// This file extends cpu030_pmmu.go's PMOVE with five more PMMU
// registers, chosen by the maintainer as the ones actually needed to
// configure address translation beyond TC: CRP/SRP (the Cursor/Supervisor
// Root Pointer registers — 64-bit table descriptors; TC only enables
// translation, CRP/SRP say where the translation tables actually live),
// TT0/TT1 (Transparent Translation — direct, untranslated pass-through
// windows), and MMUSR (the MMU status register — 16-bit, read after a
// PTEST-style probe). Every other PMMU register/mnemonic remains
// deliberately out of scope — see cpu030_pmmu.go's own header comment
// for the full accounting of what was cut and why.
//
// Every one of these registers has a compile-time-known selector value
// (unlike, say, FMOVEM's runtime-varying register-list masks), so —
// exactly like TC's own already-shipped Forms — no FieldRef is needed
// anywhere here: each Form's word2 is a single, fully-fixed literal.
// Word1 is 0xF000|<ea> in every case, identical to TC.
//
// Selector values (bits 12-10 of word2, shared bit position across
// every PMMU control register PMOVE can move) were cross-checked
// against gas/config/tc-m68k.c's "case 'W':" (DRP=1, SRP=2, CRP=3) and
// "case '3':" (TT0=2, TT1=3) — the same install_operand dispatch TC's
// own selector (0, case '0'/'1'/'2') was confirmed against in
// milestone 12. MMUSR's row ("case 'Y':") calls no install_operand at
// all — GAS just asserts the register is exactly PSR/MMUSR and emits
// the row's own literal unchanged, confirming it needs no selector bits
// of its own (there is nothing else 'Y' could mean).
//
// EA restrictions differ by register, also cross-checked against GAS's
// own per-row argument-type letters:
//   - CRP/SRP ('|' load-side, '~' store-side): memory only, no Dn/An/
//     immediate — GAS's own comment on the immediate exclusion notes
//     these registers are "quad word" (64-bit) and it doesn't support
//     that; a memory-only restriction is the only sensible match on
//     this codebase's side too, so memoryAlterableEA is used for both
//     directions (both GAS argument letters exclude exactly the same
//     set: DREG/AREG/CONTROL/FPREG/IMMED/REGLST).
//   - TT0/TT1 and MMUSR ('*' load-side, '%' store-side): the same
//     broad restriction TC's own Forms already use — readableDataEA
//     for the load direction, dataAlterableEA for the store direction.
func init() {
	newPmmuFixedReg("CRP", OpkCRP, LongSize, 0x4000|(3<<10), 0x4200|(3<<10), memoryAlterableEA, memoryAlterableEA)
	newPmmuFixedReg("SRP", OpkSRP, LongSize, 0x4000|(2<<10), 0x4200|(2<<10), memoryAlterableEA, memoryAlterableEA)
	newPmmuFixedReg("TT0", OpkTT0, LongSize, 0x0800, 0x0A00, nil, nil)
	newPmmuFixedReg("TT1", OpkTT1, LongSize, 0x0C00, 0x0E00, nil, nil)
	newPmmuFixedReg("MMUSR", OpkMMUSR, WordSize, 0x6000, 0x6200, nil, nil) // GAS's own "*w.../ ...%s" size hint — MMUSR is 16 bits
}

// newPmmuFixedReg appends a load Form ("PMOVE.<sz> <ea>,REG") and a
// store Form ("PMOVE.<sz> REG,<ea>") to the existing PMOVE InstrDef for
// one PMMU register whose word2 is fully known at compile time. sz is
// explicit, not inferred, because GAS's own per-register type-check
// restricts different PMOVE registers to different sizes (confirmed
// per-register, not assumed uniform — see cpu030_pmmu3.go's own header
// comment for the full breakdown: TC is .L, AC is .W, CAL/VAL/SCC are
// .B). loadEA/storeEA are the destination-EA-kind sets to validate
// against; nil means "use TC's own existing readableDataEA/
// dataAlterableEA restriction" (TT0/TT1/MMUSR), non-nil overrides it
// (CRP/SRP's stricter memory-only restriction).
func newPmmuFixedReg(name string, opk OperandKind, sz Size, loadWord2, storeWord2 uint16, loadEA, storeEA map[EAExprKind]bool) {
	loadValidate := func(a *Args) error {
		set := loadEA
		if set == nil {
			set = readableDataEA
		}
		if !set[a.Src.Kind] {
			if a.Src.Kind == EAkNone {
				return fmt.Errorf("PMOVE requires a source operand")
			}
			return fmt.Errorf("PMOVE source must be a readable addressing mode")
		}
		return nil
	}
	storeValidate := func(a *Args) error {
		set := storeEA
		if set == nil {
			set = dataAlterableEA
		}
		if !set[a.Dst.Kind] {
			if a.Dst.Kind == EAkNone {
				return fmt.Errorf("PMOVE requires a destination operand")
			}
			return fmt.Errorf("PMOVE destination must be data-alterable")
		}
		return nil
	}
	defPMOVE.Forms = append(defPMOVE.Forms,
		FormDef{
			DefaultSize: sz,
			Sizes:       []Size{sz},
			OperKinds:   []OperandKind{OpkEA, opk},
			Validate:    loadValidate,
			Requires:    requirePMMU,
			Steps: []EmitStep{
				{WordBits: 0xF000, Fields: []FieldRef{FSrcEA}},
				{WordBits: loadWord2},
				{Trailer: []TrailerItem{TSrcEAExt}},
			},
		},
		FormDef{
			DefaultSize: sz,
			Sizes:       []Size{sz},
			OperKinds:   []OperandKind{opk, OpkEA},
			Validate:    storeValidate,
			Requires:    requirePMMU,
			Steps: []EmitStep{
				{WordBits: 0xF000, Fields: []FieldRef{FDstEA}},
				{WordBits: storeWord2},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		},
	)
}
