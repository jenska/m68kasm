package instructions

import "fmt"

// This file adds FMOVEM (FPn register-list save/restore), the piece of
// the first FPU milestone (cpu020_fpu.go) deferred twice already — once
// in the original FPU slice, once again in cpu020_fpu2.go — because it
// needs its own register-list infrastructure rather than being an
// incremental addition. It covers only the general-purpose FPn (FP0-
// FP7) register-list form, both static ("FP0-FP3/FP5") and dynamic (a
// Dn holding the list at runtime); the separate FPCR/FPSR/FPIAR
// control-register list form ("fmoveml" in GAS's table) was deferred
// out of this file — see cpu020_fpu_movem2.go, milestone 17.
//
// Word1 is always fpuWord1Base (0xF200, the coprocessor-ID bit every
// FPU instruction needs — see its doc comment in cpu020_fpu.go) ORed
// with the EA field, exactly like FSAVE/FRESTORE (cpu020_fpu2.go): GAS's
// own "general" and "postincrement-load" rows are, for the LOAD
// direction, the same EA-artifact equivalence FSAVE/FRESTORE already
// rely on (general word1 0xF000/0xF200 combined with FSrcEA's own mode
// bits reproduces the postincrement row's baked 0xF018/0xF218 literal
// exactly) — one Form covers both. The STORE direction's word2 is NOT
// an EA artifact, though: GAS's predecrement-store row's word2 (0xE000)
// differs from the general-store row's word2 (0xF000) by a genuine
// "direction/mode" bit no FieldRef already computes from EncodeEA. Two
// Forms sharing identical OperKinds would hit exactly the selectForm
// hazard FFPMovemStoreWord2's doc comment (types.go) describes, so
// instead one Form's word2 is computed entirely at encode time from the
// Dst EA's runtime mode (FFPMovemStoreWord2/FFPMovemDynStoreWord2),
// mirroring how integer MOVEM's own TSrcRegMask already resolves this
// identical general-vs-predecrement split by checking p.DstEA.Mode
// directly rather than via a second Form — including the same list
// bit-reversal for predecrement (reverse16, encode.go).
//
// The FPCR/FPSR/FPIAR control-register list form was deferred out of
// this file for the same reason (a real bit-width/position difference —
// a 3-bit selector at bits 12-10 of word2, versus the 8-bit FPn mask at
// bits 7-0 — and its own type-check rules allowing Dn/An as the EA
// side, unlike the FPn form's real-memory-address-only restriction) but
// was picked up as its own milestone shortly after — see
// cpu020_fpu_movem2.go.

func init() {
	registerInstrDef(&defFMOVEM)
}

var defFMOVEM = InstrDef{
	Mnemonic: "FMOVEM",
	Forms: []FormDef{
		{
			// FMOVEM FPn-list,<ea> (static list; general memory or
			// -(An)). One Form covers both: FFPMovemStoreWord2 picks
			// word2's general-vs-predecrement bit (and reverses the
			// list only for predecrement) from the Dst EA's runtime
			// mode — see its doc comment in types.go for why two Forms
			// here would have been actively wrong, not just redundant.
			DefaultSize: ExtendedSize,
			Sizes:       []Size{ExtendedSize},
			OperKinds:   []OperandKind{OpkFPRegList, OpkEA},
			Validate:    validateFMovemStore,
			Requires:    requireFPU,
			Steps: []EmitStep{
				{WordBits: fpuWord1Base, Fields: []FieldRef{FDstEA}},
				{Fields: []FieldRef{FFPMovemStoreWord2}},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		},
		{
			// FMOVEM Dn,<ea> (dynamic list; general memory or -(An)) —
			// same general-vs-predecrement word2 split as the static
			// form above, via FFPMovemDynStoreWord2.
			DefaultSize: ExtendedSize,
			Sizes:       []Size{ExtendedSize},
			OperKinds:   []OperandKind{OpkDn, OpkEA},
			Validate:    validateFMovemDynStore,
			Requires:    requireFPU,
			Steps: []EmitStep{
				{WordBits: fpuWord1Base, Fields: []FieldRef{FDstEA}},
				{Fields: []FieldRef{FFPMovemDynStoreWord2}},
				{Trailer: []TrailerItem{TDstEAExt}},
			},
		},
		{
			// FMOVEM <ea>,FPn-list (static, general memory or
			// postincrement — the EA-artifact equivalence covers both).
			DefaultSize: ExtendedSize,
			Sizes:       []Size{ExtendedSize},
			OperKinds:   []OperandKind{OpkEA, OpkFPRegList},
			Validate:    validateFMovemLoadGeneral,
			Requires:    requireFPU,
			Steps: []EmitStep{
				{WordBits: fpuWord1Base, Fields: []FieldRef{FSrcEA}},
				{WordBits: 0xD000, Fields: []FieldRef{FFPRegMaskDst}},
				{Trailer: []TrailerItem{TSrcEAExt}},
			},
		},
		{
			// FMOVEM <ea>,Dn (dynamic list, general memory or
			// postincrement).
			DefaultSize: ExtendedSize,
			Sizes:       []Size{ExtendedSize},
			OperKinds:   []OperandKind{OpkEA, OpkDn},
			Validate:    validateFMovemDynLoad,
			Requires:    requireFPU,
			Steps: []EmitStep{
				{WordBits: fpuWord1Base, Fields: []FieldRef{FSrcEA}},
				{WordBits: 0xD800, Fields: []FieldRef{FFPDynDstReg4}},
				{Trailer: []TrailerItem{TSrcEAExt}},
			},
		},
	},
}

// validateFMovemStore allows general memory or -(An) as the store
// destination, but not (An)+ — matching real hardware and GAS's own
// row split (a postincrement *store* form doesn't exist; postincrement
// is load-only, predecrement store-only, exactly like integer MOVEM).
func validateFMovemStore(a *Args) error {
	if a.FPRegMaskSrc == 0 {
		return fmt.Errorf("FMOVEM requires an FPn register list")
	}
	if a.Dst.Kind == EAkNone {
		return fmt.Errorf("FMOVEM requires a destination")
	}
	if !memoryAlterableEA[a.Dst.Kind] || a.Dst.Kind == EAkAddrPostinc {
		return fmt.Errorf("FMOVEM destination must be a memory addressing mode (not (An)+)")
	}
	return nil
}

func validateFMovemDynStore(a *Args) error {
	if a.Src.Kind != EAkDn {
		return fmt.Errorf("FMOVEM requires a Dn register or FPn list source")
	}
	if a.Dst.Kind == EAkNone {
		return fmt.Errorf("FMOVEM requires a destination")
	}
	if !memoryAlterableEA[a.Dst.Kind] || a.Dst.Kind == EAkAddrPostinc {
		return fmt.Errorf("FMOVEM destination must be a memory addressing mode (not (An)+)")
	}
	return nil
}

func validateFMovemLoadGeneral(a *Args) error {
	if a.Src.Kind == EAkNone {
		return fmt.Errorf("FMOVEM requires a source")
	}
	if !memoryAlterableEA[a.Src.Kind] || a.Src.Kind == EAkAddrPredec {
		return fmt.Errorf("FMOVEM source must be a memory addressing mode (not -(An))")
	}
	if a.FPRegMaskDst == 0 {
		return fmt.Errorf("FMOVEM requires an FPn register list destination")
	}
	return nil
}

func validateFMovemDynLoad(a *Args) error {
	if a.Src.Kind == EAkNone {
		return fmt.Errorf("FMOVEM requires a source")
	}
	if !memoryAlterableEA[a.Src.Kind] || a.Src.Kind == EAkAddrPredec {
		return fmt.Errorf("FMOVEM source must be a memory addressing mode (not -(An))")
	}
	if a.Dst.Kind != EAkDn {
		return fmt.Errorf("FMOVEM requires a Dn register destination")
	}
	return nil
}
