package instructions

import "fmt"

// This file extends PMOVE with the last remaining PMMU registers:
// BAD0-BAD7 and BAC0-BAC7 (Breakpoint Address/Access registers, 8
// numbered instances each), deliberately deferred in milestone 20
// because their encoding is genuinely different in shape from every
// other PMOVE register:
//
//   - a register NUMBER (0-7) at bits 4-2, not just a fixed selector —
//     the "numbered instance" shape EAkFPn/OpkFPn already use, unlike
//     every other PMMU register in cpu030_pmmu2.go/cpu030_pmmu3.go
//     (each its own single fixed register, no number to encode);
//   - an INVERTED load/store direction bit relative to every other
//     PMOVE register. Every other register in this file follows a
//     "word2 = base | direction(0x0000 load / 0x0200 store) | selector"
//     convention (see cpu030_pmmu.go's TC, the very first one); GAS's
//     own opcode-table rows for BAD/BAC ("*wX3" load, word2 0x6200;
//     "X3%s" store, word2 0x6000, gas/config/tc-m68k.c) have that bit
//     backwards — 0x0200 set means LOAD here, not store.
//
// Because load and store already have distinct OperKinds ([OpkEA,
// OpkBAD/OpkBAC] vs [OpkBAD/OpkBAC,OpkEA] — unlike, say, MOVE16's
// "(An),abs" pairing, which needed a single runtime-computed Form to
// avoid a real ambiguity), the direction and the family (BAD vs BAC)
// are both decided per-Form at definition time; only the register
// NUMBER (FSrcRegShift2/FDstRegShift2, types.go) needs a runtime
// FieldRef.
//
// Word2 bases, derived from GAS's own literals (0x6200 load / 0x6000
// store) combined with the family bit ("case 'X':" in
// gas/config/tc-m68k.c: BAD tmpreg = 0x1000 | (n<<2), BAC tmpreg =
// 0x1400 | (n<<2)):
//
//	BADn load  = 0x6200 | 0x1000 = 0x7200 | (n<<2)
//	BADn store = 0x6000 | 0x1000 = 0x7000 | (n<<2)
//	BACn load  = 0x6200 | 0x1400 = 0x7600 | (n<<2)
//	BACn store = 0x6000 | 0x1400 = 0x7400 | (n<<2)
//
// EA restrictions match TC/TT0/TT1/MMUSR's own broad ones (GAS's '*'
// load-side / '%' store-side argument types here are exactly the same
// classes those registers already use): readableDataEA for the load
// direction, dataAlterableEA for the store direction.
//
// This completes PMOVE's register set entirely.
func init() {
	newPmmuNumberedReg("BAD", OpkBAD, 0x7200, 0x7000)
	newPmmuNumberedReg("BAC", OpkBAC, 0x7600, 0x7400)
}

// newPmmuNumberedReg appends a load Form ("PMOVE.L <ea>,REGn") and a
// store Form ("PMOVE.L REGn,<ea>") to the existing PMOVE InstrDef for
// one numbered PMMU register family (BAD or BAC).
func newPmmuNumberedReg(name string, opk OperandKind, loadBase, storeBase uint16) {
	loadValidate := func(a *Args) error {
		if !readableDataEA[a.Src.Kind] {
			if a.Src.Kind == EAkNone {
				return fmt.Errorf("PMOVE requires a source operand")
			}
			return fmt.Errorf("PMOVE source must be a readable addressing mode")
		}
		return nil
	}
	storeValidate := func(a *Args) error {
		if !dataAlterableEA[a.Dst.Kind] {
			if a.Dst.Kind == EAkNone {
				return fmt.Errorf("PMOVE requires a destination operand")
			}
			return fmt.Errorf("PMOVE destination must be data-alterable")
		}
		return nil
	}
	defPMOVE.Forms = append(defPMOVE.Forms,
		FormDef{
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OpkEA, opk},
			Validate:    loadValidate,
			Requires:    requirePMMU,
			Steps: []EmitStep{
				{WordBits: 0xF000, Fields: []FieldRef{FSrcEA}},
				{WordBits: loadBase, Fields: []FieldRef{FDstRegShift2}},
				{Trailer: []TrailerItem{TSrcEAExt}},
			},
		},
		FormDef{
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{opk, OpkEA},
			Validate:    storeValidate,
			Requires:    requirePMMU,
			Steps: []EmitStep{
				{WordBits: 0xF000, Fields: []FieldRef{FDstEA}},
				{WordBits: storeBase, Fields: []FieldRef{FSrcRegShift2}},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		},
	)
}
