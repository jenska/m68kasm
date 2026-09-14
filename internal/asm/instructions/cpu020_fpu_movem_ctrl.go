package instructions

import (
	"fmt"
	"math/bits"
)

// This file adds FMOVEM's other register-list form, deferred from
// cpu020_fpu_movem.go: moving the FPU's FPCR/FPSR/FPIAR control
// registers to/from memory, Dn, or An — GAS's "fmoveml" table entries,
// a genuinely separate subsystem from the FP0-FP7 data-register list
// (a 3-bit selector at word2 bits 12-10, not the 8-bit mask at bits
// 7-0; its own, narrower set of valid destinations).
//
// Word1 is fpuWord1Base ORed with the EA field, as everywhere else in
// this FPU slice. Word2's base differs only by the store/load
// direction bit (0xA000 store, 0x8000 load — the FPn form's own
// direction split, 0xF000/0xD000, minus 0x5000; unrelated to the FPn
// form's separate general-vs-predecrement split, which doesn't apply
// here at all: GAS's fmoveml has no predecrement/postincrement-specific
// row, matching Motorola's real restriction that a control-register
// move never auto-increments).
//
// GAS's own table lists the store direction as effectively two
// overlapping rows — a stricter one requiring a single bare register
// name (allowing the destination to be Dn/An/memory alike) and a
// looser one accepting real list syntax (restricted to memory only) —
// with a FIXME comment noting the intended rule directly: "we should
// only permit %dn if the target is a single register." Rather than
// replicate two rows, validateFMovemlStore checks this directly: with
// exactly one register named, Dn/An/memory are all valid destinations;
// with more than one, only memory is (a single 32-bit Dn/An can't
// sensibly receive two distinct registers' values in one move). The
// load direction has no such split in GAS's table — its <ea> source
// accepts memory, Dn, An, or an immediate, unconditionally.
var requireFPUCtrlList = requireFPU

// init appends this file's two Forms onto the existing "FMOVEM"
// InstrDef that cpu020_fpu_movem.go's own init already registered,
// rather than registering a second InstrDef under the same mnemonic
// (registerInstrDef panics on a duplicate name — FMOVEM's FPn-list and
// control-register-list forms share one mnemonic in real 68k assembly,
// so they have to share one InstrDef too). This relies on
// cpu020_fpu_movem.go's init running first, which Go's spec both
// guarantees in practice and explicitly recommends build tools follow:
// init functions across a package's files run in the files' lexical
// name order, and "cpu020_fpu_movem.go" sorts before
// "cpu020_fpu_movem2.go". The explicit nil check turns a violated
// assumption into a clear panic instead of a silent nil dereference.
func init() {
	fmovem := Instructions["FMOVEM"]
	if fmovem == nil {
		panic("cpu020_fpu_movem2.go: FMOVEM must already be registered (by cpu020_fpu_movem.go) before its control-register forms can be appended")
	}
	fmovem.Forms = append(fmovem.Forms,
		FormDef{
			// FMOVEM FPIAR/FPSR/FPCR,<ea>
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OpkFPCtrlRegList, OpkEA},
			Validate:    validateFMovemlStore,
			Requires:    requireFPUCtrlList,
			Steps: []EmitStep{
				{WordBits: fpuWord1Base, Fields: []FieldRef{FDstEA}},
				{WordBits: 0xA000, Fields: []FieldRef{FFPCtrlSelSrc10}},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		},
		FormDef{
			// FMOVEM <ea>,FPIAR/FPSR/FPCR
			DefaultSize: LongSize,
			Sizes:       []Size{LongSize},
			OperKinds:   []OperandKind{OpkEA, OpkFPCtrlRegList},
			Validate:    validateFMovemlLoad,
			Requires:    requireFPUCtrlList,
			Steps: []EmitStep{
				{WordBits: fpuWord1Base, Fields: []FieldRef{FSrcEA}},
				{WordBits: 0x8000, Fields: []FieldRef{FFPCtrlSelDst10}},
				{Trailer: []TrailerItem{TSrcEAExt}},
			},
		},
	)
}

func validateFMovemlStore(a *Args) error {
	if a.Dst.Kind == EAkNone {
		return fmt.Errorf("FMOVEM requires a destination")
	}
	if bits.OnesCount16(a.FPCtrlMaskSrc) == 1 {
		if memoryAlterableEA[a.Dst.Kind] || a.Dst.Kind == EAkDn || a.Dst.Kind == EAkAn {
			return nil
		}
		return fmt.Errorf("FMOVEM destination must be a memory addressing mode, Dn, or An")
	}
	if !memoryAlterableEA[a.Dst.Kind] {
		return fmt.Errorf("FMOVEM destination must be a memory addressing mode when moving more than one control register")
	}
	return nil
}

func validateFMovemlLoad(a *Args) error {
	if a.Src.Kind == EAkNone {
		return fmt.Errorf("FMOVEM requires a source")
	}
	return nil
}
