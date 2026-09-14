package instructions

import "fmt"

// This file adds the 68040's own PMMU instructions — a simplified,
// re-encoded interface distinct from the 68030/68851 coprocessor-style
// forms already implemented (cpu030_pmmu*.go): every instruction here
// is a single 16-bit word, not a 0xF000-prefixed two-word coprocessor
// instruction, and each one's sole operand (when it has one) is always
// an address register (accepted as either "(An)" or bare "An", GAS's
// own dual argument-letter row for this operand) — never a general
// <ea>, function-code specifier, or immediate mask. Chosen by the
// maintainer from the open items list to close out the PMMU surface's
// remaining PFLUSH/PTEST variants (see cpu030_pmmu_ptest.go's own
// PFLUSH/PFLUSHS/PFLUSHR, added alongside this milestone).
//
// Cross-checked against GNU binutils' GAS m68k opcode table
// (opcodes/m68k-opc.c):
//
//	{"pflusha",  one(0xf518), one(0xfff8), "",   m68040up },
//	{"pflushan", one(0xf510), one(0xfff8), "",   m68040up },
//	{"pflushn",  one(0xf500), one(0xfff8), "as", m68040up },
//	{"pflush",   one(0xf508), one(0xfff8), "as", m68040up },
//	{"ptestr",   one(0xf568), one(0xfff8), "as", m68040 },
//	{"ptestw",   one(0xf548), one(0xfff8), "as", m68040 },
//
// PFLUSHA/PFLUSHAN/PFLUSHN/PFLUSH's single-word forms are tagged
// `m68040up` (a plain CPU-floor Requires: Target{CPU: CPU68040} extends
// to the 68060 exactly as intended), but PTESTR/PTESTW's are tagged the
// bare `m68040` — the 68060 dropped them — so those two use
// requireM68040Only (cpu.go), the same exact-tier mechanism already
// established for CALLM/RTM (featM68020Only) and TBLS/TBLU/BGND
// (featCPU32Only).
//
// PFLUSHA and PFLUSH already exist in this codebase as the 68030/68851
// two-word coprocessor forms (defPFLUSHA, cpu030_pmmu.go; PFLUSH,
// cpu030_pmmu_ptest.go) with OperKinds [] and [OpkFCSpec,OpkImm(,OpkEA)]
// respectively — neither collides with this file's own new forms except
// PFLUSHA, whose 68040 form ALSO has OperKinds [] (zero operands, same
// as the two-word form): two Forms an otherwise-ambiguous target (68040
// with PMMU enabled) could both satisfy after Table.ForTarget's
// Requires-filtering, since a bare CPU-floor Requires never excludes
// higher tiers. Resolved by ordering, not by Validate: the 68040 form is
// PREPENDED to defPFLUSHA.Forms, so parseInstruction's own "try Forms in
// order, first match wins" picks it first on a 68040+ target, while
// Table.ForTarget correctly drops it entirely below CPU68040 — the exact
// same 0-operand-form ordering hazard already documented for TRAPcc
// (docs/design/cpu-family-support.md, milestone 5), applied here
// proactively rather than found as a bug.
func init() {
	require68040up := Target{CPU: CPU68040, Features: FeatPMMU}

	defPFLUSHA.Forms = append([]FormDef{{
		DefaultSize: WordSize,
		Sizes:       []Size{WordSize},
		OperKinds:   []OperandKind{},
		Requires:    require68040up,
		Steps:       []EmitStep{{WordBits: 0xF518}},
	}}, defPFLUSHA.Forms...)

	registerInstrDef(&InstrDef{
		Mnemonic: "PFLUSHAN",
		Forms: []FormDef{{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{},
			Requires:    require68040up,
			Steps:       []EmitStep{{WordBits: 0xF510}},
		}},
	})

	registerInstrDef(&InstrDef{
		Mnemonic: "PFLUSHN",
		Forms:    []FormDef{newAnOnlyForm("PFLUSHN", 0xF500, require68040up)},
	})

	defPFLUSH.Forms = append(defPFLUSH.Forms, newAnOnlyForm("PFLUSH", 0xF508, require68040up))

	defPTESTR.Forms = append(defPTESTR.Forms, newAnOnlyForm("PTESTR", 0xF568, requireM68040Only))
	defPTESTW.Forms = append(defPTESTW.Forms, newAnOnlyForm("PTESTW", 0xF548, requireM68040Only))
}

// newAnOnlyForm builds one Form for a single-word, single-operand 68040
// PMMU instruction whose sole operand is always an address register
// (either "(An)" or bare "An" — validateAnOperand accepts both, since
// GAS's own table gives this operand two argument-letter rows sharing
// one encoding). word1 is the fully-fixed base literal; FSrcRegOnly ORs
// in the register number at bits 2-0.
func newAnOnlyForm(name string, word1 uint16, requires Target) FormDef {
	return FormDef{
		DefaultSize: WordSize,
		Sizes:       []Size{WordSize},
		OperKinds:   []OperandKind{OpkEA},
		Validate:    func(a *Args) error { return validateAnOperand(name, a.Src.Kind) },
		Requires:    requires,
		Steps: []EmitStep{
			{WordBits: word1, Fields: []FieldRef{FSrcRegOnly}},
		},
	}
}

func validateAnOperand(name string, k EAExprKind) error {
	if k != EAkAn && k != EAkAddrInd {
		if k == EAkNone {
			return fmt.Errorf("%s requires an address register operand", name)
		}
		return fmt.Errorf("%s requires an address register, \"An\" or \"(An)\"", name)
	}
	return nil
}
