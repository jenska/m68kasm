package instructions

import "fmt"

// This file adds PSAVE/PRESTORE (save/restore the 68851 PMMU's internal
// state frame), chosen by the maintainer from the open items list after
// milestone 24 to help close out the core PMMU surface. They mirror
// cpu020_fpu2.go's FSAVE/FRESTORE almost exactly: same single-<ea>-
// operand shape, same "write descending / read ascending" EA
// restriction (PSAVE only -(An), PRESTORE only (An)+), and the same
// GAS-table oddity where the "dedicated" predecrement/postincrement
// literal is just the general literal with that EA's own mode bits
// already present.
//
// Cross-checked against GNU binutils' GAS m68k opcode table
// (opcodes/m68k-opc.c): unlike FSAVE/FRESTORE, GAS gives PSAVE/PRESTORE
// only ONE table row each, not two:
//
//	{"prestore", one(0xf140), one(0xffc0), "<s", m68851 },
//	{"psave",    one(0xf100), one(0xffc0), ">s", m68851 },
//
// The ">s"/"<s" argument codes are the same ones MOVEM's own store/load
// directions use (gas/config/tc-m68k.c's md_assemble, "case '>'"/"case
// '<'": '>' = control-alterable-or-predecrement, matching a register
// list written descending; '<' = control-alterable-or-postincrement,
// matching one read back ascending) — i.e. GAS itself expresses PSAVE/
// PRESTORE's EA restriction as exactly the union this codebase already
// builds via memoryAlterableEA plus a single forbidden-kind check, with
// no separate "dedicated" row needed on either side. The mask 0xffc0
// leaves only the low 6 EA bits variable, confirming there is no
// coprocessor-ID field here (unlike FSAVE/FRESTORE's "Id"-prefixed FPU
// rows) — PSAVE/PRESTORE's word1 is used as the fully-fixed literal
// 0xF100/0xF140, no fpuWord1Base-style adjustment applies.
func init() {
	registerInstrDef(newPmmuSaveRestoreDef("PSAVE", true))
	registerInstrDef(newPmmuSaveRestoreDef("PRESTORE", false))
}

// newPmmuSaveRestoreDef builds PSAVE (store, isSave=true) or PRESTORE
// (load, isSave=false), reusing the exact single-Form EA-equivalence
// trick newFSaveRestoreDef established for FSAVE/FRESTORE (see that
// function's own doc comment in cpu020_fpu2.go for the full reasoning):
// one Form with the general-form literal as WordBits, combined with
// FSrcEA, already produces the identical bits GAS's "dedicated" -(An)/
// (An)+ row would, so a second Form would be actively wrong (a
// selectForm hazard), not just redundant.
func newPmmuSaveRestoreDef(name string, isSave bool) *InstrDef {
	word1 := uint16(0xF100)
	forbidden := EAkAddrPostinc // PSAVE allows -(An), not (An)+
	if !isSave {
		word1 = 0xF140
		forbidden = EAkAddrPredec // PRESTORE allows (An)+, not -(An)
	}
	validate := func(a *Args) error {
		if a.Src.Kind == EAkNone {
			return fmt.Errorf("%s requires an operand", name)
		}
		if !memoryAlterableEA[a.Src.Kind] || a.Src.Kind == forbidden {
			return fmt.Errorf("%s requires a memory addressing mode", name)
		}
		return nil
	}
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkEA},
				Validate:    validate,
				Requires:    requirePMMU,
				Steps: []EmitStep{
					{WordBits: word1, Fields: []FieldRef{FSrcEA}},
					{Trailer: []TrailerItem{TSrcEAExt}},
				},
			},
		},
	}
}
