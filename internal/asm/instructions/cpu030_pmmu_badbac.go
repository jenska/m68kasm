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
//     every other PMMU register in cpu030_pmmu_xlate.go/cpu030_pmmu_access.go
//     (each its own single fixed register, no number to encode);
//   - a 16-bit operand. The direction bit is the usual one ("word2 =
//     base | direction(0x0000 load / 0x0200 store) | selector", MC68851
//     User's Manual, PMOVE format 2); GAS's own opcode-table rows for
//     BAD/BAC ("*wX3" load, word2 0x6200; "X3%s" store, word2 0x6000)
//     have it reversed, which this assembler followed until v1.6.5.
//
// Because load and store already have distinct OperKinds ([OpkEA,
// OpkBAD/OpkBAC] vs [OpkBAD/OpkBAC,OpkEA] — unlike, say, MOVE16's
// "(An),abs" pairing, which needed a single runtime-computed Form to
// avoid a real ambiguity), the direction and the family (BAD vs BAC)
// are both decided per-Form at definition time; only the register
// NUMBER (FSrcRegShift2/FDstRegShift2, types.go) needs a runtime
// FieldRef.
//
// Word2 bases: format 2's 0x6000 with the family bit (BAD 0x1000, BAC
// 0x1400) and the direction bit:
//
//	BADn load  = 0x7000 | (n<<2)
//	BADn store = 0x7200 | (n<<2)
//	BACn load  = 0x7400 | (n<<2)
//	BACn store = 0x7600 | (n<<2)
//
// EA restrictions match TC/TT0/TT1/MMUSR's own broad ones (GAS's '*'
// load-side / '%' store-side argument types here are exactly the same
// classes those registers already use): readableDataEA for the load
// direction, dataAlterableEA for the store direction.
//
// This completes PMOVE's register set entirely.
func init() {
	newPmmuNumberedReg(OpkBAD, 0x7000, 0x7200)
	newPmmuNumberedReg(OpkBAC, 0x7400, 0x7600)
}

// newPmmuNumberedReg appends a load Form ("PMOVE.W <ea>,REGn") and a
// store Form ("PMOVE.W REGn,<ea>") to the existing PMOVE InstrDef for
// one numbered PMMU register family (BAD or BAC), both 16-bit registers
// (MC68851 User's Manual, PMOVE: "a word operation for BAC, BAD").
func newPmmuNumberedReg(opk OperandKind, loadBase, storeBase uint16) {
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
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkEA, opk},
			Validate:    loadValidate,
			Requires:    requirePMMU,
			Steps: []EmitStep{
				{WordBits: 0xF000, Fields: []FieldRef{FSrcEA}},
				{WordBits: loadBase, Fields: []FieldRef{FDstRegShift2}},
				{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
			},
		},
		FormDef{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
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
