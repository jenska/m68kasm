package instructions

import "fmt"

// This file adds a deliberately narrow slice of the 68030/68851 PMMU
// instruction set: PMOVE (only its Translation Control register form,
// "PMOVE.L <ea>,TC" / "PMOVE.L TC,<ea>") and PFLUSHA. The full PMMU
// instruction family turned out to be one of the largest corners of the
// 68k ISA — dozens of Pcc branch/set/trap condition variants mirroring
// the entire Bcc/Scc/DBcc families but for MMU status flags, PFLUSH
// alone with six different operand shapes, and PMOVE covering half a
// dozen different MMU registers (TC, CRP, SRP, TT0, TT1, MMUSR, …) each
// with its own selector-code scheme — clearly out of proportion with
// every other milestone in this project, and PMMU (kernel-level virtual
// memory setup) is realistically the least likely of this assembler's
// features to see real use. TC and PFLUSHA were chosen as the two
// instructions actually needed for basic MMU setup; every other PMMU
// form is deliberately left for a future, separately-scoped milestone.
// See docs/design/cpu-family-support.md.
//
// Encodings were cross-checked against GNU binutils' GAS m68k opcode
// table (opcodes/m68k-opc.c) and gas/config/tc-m68k.c's control-register
// enumeration (which resolved TC's specific selector value — 0 — among
// several PMMU registers sharing the same install_operand cases).
//
// Gated on Target{Features: FeatPMMU} (the --mmu CLI flag), not a CPU
// floor — like FPU, PMMU presence is an attached-coprocessor question
// independent of the integer CPU tier (a bare 68020 with an external
// 68851 and a 68030's on-chip PMMU both just need this bit set).
var requirePMMU = Target{Features: FeatPMMU}

func init() {
	registerInstrDef(&defPMOVE)
	registerInstrDef(&defPFLUSHA)
}

var defPMOVE = InstrDef{
	Mnemonic: "PMOVE",
	Forms: []FormDef{
		{
			// PMOVE.L <ea>,TC (load): word2 0x4000 | (TC's selector, 0,
			// contributing nothing) — see cross-check note above.
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OpkEA, OpkTC},
			Validate:    validatePmoveLoad,
			Requires:    requirePMMU,
			Steps: []EmitStep{
				{WordBits: 0xF000, Fields: []FieldRef{FSrcEA}},
				{WordBits: 0x4000},
				{Trailer: []TrailerItem{TSrcEAExt}},
			},
		},
		{
			// PMOVE.L TC,<ea> (store): word2 0x4200 — bit 9 set is the
			// direction flag relative to the load form's 0x4000.
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OpkTC, OpkEA},
			Validate:    validatePmoveStore,
			Requires:    requirePMMU,
			Steps: []EmitStep{
				{WordBits: 0xF000, Fields: []FieldRef{FDstEA}},
				{WordBits: 0x4200},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		},
	},
}

func validatePmoveLoad(a *Args) error {
	if !readableDataEA[a.Src.Kind] {
		if a.Src.Kind == EAkNone {
			return fmt.Errorf("PMOVE requires a source operand")
		}
		return fmt.Errorf("PMOVE source must be a readable addressing mode")
	}
	return nil
}

func validatePmoveStore(a *Args) error {
	if !dataAlterableEA[a.Dst.Kind] {
		if a.Dst.Kind == EAkNone {
			return fmt.Errorf("PMOVE requires a destination operand")
		}
		return fmt.Errorf("PMOVE destination must be data-alterable")
	}
	return nil
}

var defPFLUSHA = InstrDef{
	Mnemonic: "PFLUSHA",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{},
			Requires:    requirePMMU,
			Steps: []EmitStep{
				{WordBits: 0xF000},
				{WordBits: 0x2400},
			},
		},
	},
}
