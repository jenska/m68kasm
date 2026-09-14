package instructions

import "fmt"

// This file adds PMOVEFD, chosen alongside PSAVE/PRESTORE (see
// cpu030_pmmu_save.go) to close out the core PMMU surface. PMOVEFD
// ("PMOVE, Function code lookup Disabled") loads a translation register
// from memory exactly like PMOVE's own load direction, except bit 8 of
// word2 is set to tell the PMMU to skip the function-code lookup it
// would normally perform while fetching the new table descriptor —
// relevant only when *loading* a register that itself controls how
// function-code lookups happen, which is why GAS gives it no store
// direction at all.
//
// Cross-checked against GNU binutils' GAS m68k opcode table
// (opcodes/m68k-opc.c), which lists exactly three rows, all load-only:
//
//	{"pmovefd", two(0xf000, 0x4100), two(0xffc0, 0xe3ff), "*l08", m68030 },
//	{"pmovefd", two(0xf000, 0x4100), two(0xffc0, 0xe3ff), "|sW8", m68030 },
//	{"pmovefd", two(0xf000, 0x0900), two(0xffc0, 0xfbff), "*l38", m68030 },
//
// These are the exact same argument-code shapes (and EA-restriction
// letters) as three of PMOVE's own load-direction rows in
// cpu030_pmmu.go/cpu030_pmmu_xlate.go — "*l08" (TC), "|sW8" (DRP/SRP/CRP,
// selected by GAS's own "case 'W'" in tc-m68k.c: DRP=1, SRP=2, CRP=3),
// and "*l38" (TT0/TT1, "case '3'": TT0=2, TT1=3) — with word2 in every
// case exactly PMOVE's own load word2 plus 0x0100 (TC: 0x4000->0x4100;
// DRP/SRP/CRP: 0x4000|(sel<<10)->that plus 0x0100; TT0/TT1:
// 0x0800->0x0900, 0x0C00->0x0D00). GAS's own install_operand dispatch
// for 'W'/'3'/'0' is unchanged from PMOVE's — no new selector logic is
// needed here, only the extra 0x0100 bit and the missing store forms.
// No CAL/VAL/SCC/AC/MMUSR/PCSR counterparts exist for PMOVEFD in GAS's
// table, so none are added here either.
func init() {
	registerInstrDef(&defPMOVEFD)
}

var defPMOVEFD = InstrDef{
	Mnemonic: "PMOVEFD",
	Forms: []FormDef{
		newPmoveFDForm(OpkTC, LongSize, 0x4100, nil),
		newPmoveFDForm(OpkDRP, LongSize, 0x4100|(1<<10), memoryAlterableEA),
		newPmoveFDForm(OpkSRP, LongSize, 0x4100|(2<<10), memoryAlterableEA),
		newPmoveFDForm(OpkCRP, LongSize, 0x4100|(3<<10), memoryAlterableEA),
		newPmoveFDForm(OpkTT0, LongSize, 0x0900, nil),
		newPmoveFDForm(OpkTT1, LongSize, 0x0D00, nil),
	},
}

// newPmoveFDForm builds one "PMOVEFD.<sz> <ea>,REG" Form. ea is the
// source-EA-kind restriction; nil means readableDataEA, matching
// PMOVE's own default for TC/TT0/TT1 (see newPmmuFixedReg's doc
// comment in cpu030_pmmu_xlate.go) — non-nil overrides it (DRP/SRP/CRP's
// stricter memory-only restriction).
func newPmoveFDForm(opk OperandKind, sz Size, word2 uint16, ea map[EAExprKind]bool) FormDef {
	validate := func(a *Args) error {
		set := ea
		if set == nil {
			set = readableDataEA
		}
		if !set[a.Src.Kind] {
			if a.Src.Kind == EAkNone {
				return fmt.Errorf("PMOVEFD requires a source operand")
			}
			return fmt.Errorf("PMOVEFD source must be a readable addressing mode")
		}
		return nil
	}
	return FormDef{
		DefaultSize: sz,
		Sizes:       []Size{sz},
		OperKinds:   []OperandKind{OpkEA, opk},
		Validate:    validate,
		Requires:    requirePMMU,
		Steps: []EmitStep{
			{WordBits: 0xF000, Fields: []FieldRef{FSrcEA}},
			{WordBits: word2},
			{Trailer: []TrailerItem{TSrcEAExt}},
		},
	}
}
