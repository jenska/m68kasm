package instructions

import "fmt"

// This file extends cpu020_fpu.go's first FPU slice with the FPU's own
// conditional-branch/set/trap family (FBcc/FDBcc/FScc/FTRAPcc — a
// distinct 32-condition space from the integer ISA's 16), FMOVECR (load
// a constant from the FPU's on-chip ROM), and FSAVE/FRESTORE (save/
// restore the FPU's internal state frame). Every encoding was decoded
// from GNU binutils' GAS m68k opcode table (opcodes/m68k-opc.c) and
// cross-checked against gas/config/tc-m68k.c's install_operand logic —
// see fpuWord1Base's doc comment in cpu020_fpu.go for the coprocessor-ID
// bit these all share.
//
// Deliberately NOT included here (left for a follow-up milestone):
// FMOVEM. Unlike the rest of this file, FMOVEM needs its own register-
// list infrastructure comparable in size to a new milestone on its own:
// a "list of FPn/FPCR/FPSR/FPIAR" syntax distinct from integer MOVEM's
// Dn/An mask, a *dynamic* list form (the register set named by a Dn at
// runtime, not fixed at assembly time), and three separate EA-class/
// bit-pattern splits (control-alterable, predecrement, postincrement)
// each with their own static-vs-dynamic-list bit and register-order
// rules — genuinely more than an incremental addition to what's here.
// Also still not included: the transcendental function set (FSIN,
// FLOGN, …), packed-BCD format (.p), and FPCR/FPSR/FPIAR as general
// operands (register-list-only) — see docs/design/cpu-family-support.md.

// fpConditions names the FPU's 32 condition codes, indexed by condition
// value (0-31) — a distinct, wider space from the integer ISA's 16
// (branchConditions/dbConditions/sccConditions in xcc.go). One table
// serves all four condition-driven FPU instruction families (FBcc/
// FDBcc/FScc/FTRAPcc): each mnemonic is just this suffix prefixed with
// the family's own letter(s) (FB/FDB/FS/FTRAP), so unlike the integer
// families' four independently-spelled tables, a single canonical list
// is enough here and avoids retyping the same 32 names four times.
var fpConditions = []string{
	"F", "EQ", "OGT", "OGE", "OLT", "OLE", "OGL", "OR",
	"UN", "UEQ", "UGT", "UGE", "ULT", "ULE", "NE", "T",
	"SF", "SEQ", "GT", "GE", "LT", "LE", "GL", "GLE",
	"NGLE", "NGL", "NLE", "NLT", "NGE", "NGT", "SNE", "ST",
}

func init() {
	for cc, suffix := range fpConditions {
		registerInstrDef(newFBccDef("FB"+suffix, uint16(cc)))
		registerInstrDef(newFDBccDef("FDB"+suffix, uint16(cc)))
		registerInstrDef(newFSccDef("FS"+suffix, uint16(cc)))
		registerInstrDef(newFTrapccDef("FTRAP"+suffix, uint16(cc)))
	}
	registerInstrDef(&defFMOVECR)
	registerInstrDef(newFSaveRestoreDef("FSAVE", true))
	registerInstrDef(newFSaveRestoreDef("FRESTORE", false))
}

// newFBccDef builds one FBcc mnemonic's two forms: a word displacement
// (word1 = 0xF280|cc, matching GAS's "fbCC" table entries) and an
// explicit 32-bit displacement (word1 = 0xF2C0|cc, "fbCCl"). Unlike
// integer Bcc, FPU branches never inline an 8-bit displacement in the
// opcode itself — the condition already occupies the opcode's low bits,
// so every form needs a full extension word, and this codebase commits
// to picking the size via the usual .W/.L suffix machinery rather than
// GAS's separate "l"-suffixed mnemonic spelling (consistent with how
// milestone 7 already reused the .L-suffix pattern for BRA.L/BSR.L
// rather than a differently-spelled mnemonic).
func newFBccDef(name string, cc uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkDispRel},
				Requires:    requireFPU,
				Steps: []EmitStep{
					{WordBits: 0xF280 | cc},
					{Trailer: []TrailerItem{TBranchWordIfNeeded}},
				},
			},
			{
				DefaultSize: LongSize,
				Sizes:       []Size{LongSize},
				OperKinds:   []OperandKind{OpkDispRel},
				Requires:    requireFPU,
				Steps: []EmitStep{
					{WordBits: 0xF2C0 | cc},
					{Trailer: []TrailerItem{TBranchLongIfNeeded}},
				},
			},
		},
	}
}

// newFDBccDef builds one FDBcc mnemonic: word1 = 0xF248 | Dn (bits 2-0),
// word2 = the condition value verbatim (0x0000-0x001F, GAS's "fdbCC"
// entries bake the raw condition code into a fully-fixed word2), then a
// word-only branch displacement — FDBcc has no 32-bit-displacement form.
func newFDBccDef(name string, cc uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkDn, OpkDispRel},
				Requires:    requireFPU,
				Steps: []EmitStep{
					{WordBits: 0xF248, Fields: []FieldRef{FSrcDnReg}},
					{WordBits: cc},
					{Trailer: []TrailerItem{TBranchWordIfNeeded}},
				},
			},
		},
	}
}

// newFSccDef builds one Fscc mnemonic: word1 = 0xF240 | <ea> (data-
// alterable, same restriction as integer Scc), word2 = the condition
// value verbatim, fully fixed (GAS's "fsCC" entries).
func newFSccDef(name string, cc uint16) *InstrDef {
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: ByteSize,
				Sizes:       []Size{ByteSize},
				OperKinds:   []OperandKind{OpkEA},
				Validate:    validateDataAlterable(name),
				Requires:    requireFPU,
				Steps: []EmitStep{
					{WordBits: 0xF240, Fields: []FieldRef{FDstEA}},
					{WordBits: cc},
					{Trailer: []TrailerItem{TDstEAExt}},
				},
			},
		},
	}
}

// newFTrapccDef builds one FTRAPcc mnemonic's three forms (bare, word
// immediate, long immediate), mirroring integer TRAPcc's own three-form
// shape (newTrapccDef, cpu020_misc.go) but with the condition living in
// a dedicated, fully-fixed word2 (0x0000-0x001F) rather than shifted
// into word1's bits 11-8 — word1 itself differs per form (0xF27C bare,
// 0xF27A word, 0xF27B long), matching GAS's "ftrapCC"/"ftrapCCw"/
// "ftrapCCl" table entries.
func newFTrapccDef(name string, cc uint16) *InstrDef {
	validateImm := func(a *Args) error {
		if a.Src.Kind != EAkImm {
			return fmt.Errorf("%s requires an immediate operand", name)
		}
		return checkImmediateRange(a.Src.Imm, a.Size)
	}
	return &InstrDef{
		Mnemonic: name,
		Forms: []FormDef{
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{},
				Requires:    requireFPU,
				Steps: []EmitStep{
					{WordBits: 0xF27C},
					{WordBits: cc},
				},
			},
			{
				DefaultSize: WordSize,
				Sizes:       []Size{WordSize},
				OperKinds:   []OperandKind{OpkImm},
				Validate:    validateImm,
				Requires:    requireFPU,
				Steps: []EmitStep{
					{WordBits: 0xF27A},
					{WordBits: cc},
					{Trailer: []TrailerItem{TSrcImm}},
				},
			},
			{
				DefaultSize: LongSize,
				Sizes:       []Size{LongSize},
				OperKinds:   []OperandKind{OpkImm},
				Validate:    validateImm,
				Requires:    requireFPU,
				Steps: []EmitStep{
					{WordBits: 0xF27B},
					{WordBits: cc},
					{Trailer: []TrailerItem{TSrcImm}},
				},
			},
		},
	}
}

// defFMOVECR is "FMOVECR #<constant>,FPn": loads one of the FPU's
// built-in ROM constants (pi, e, log10(2), …) into FPn. word1 is just
// fpuWord1Base (no <ea> field — FMOVECR never reads memory), word2 =
// 0x5C00 | the 7-bit ROM offset (bits 6-0) | the destination FPn (bits
// 9-7). Only the numeric "#<n>" form is supported — GAS also accepts
// named constants (fp_pi, fp_e, …) as assembler-level aliases for
// specific indices, which this codebase does not implement; the lexer
// has no such identifier-as-immediate mechanism, and the numeric index
// is what actually gets encoded either way.
var defFMOVECR = InstrDef{
	Mnemonic: "FMOVECR",
	Forms: []FormDef{
		{
			DefaultSize: ExtendedSize,
			Sizes:       []Size{ExtendedSize},
			OperKinds:   []OperandKind{OpkImm, OpkFPn},
			Validate:    validateFMOVECR,
			Requires:    requireFPU,
			Steps: []EmitStep{
				{WordBits: fpuWord1Base},
				{WordBits: 0x5C00, Fields: []FieldRef{FFPRomConst, FFPDstReg7}},
			},
		},
	},
}

func validateFMOVECR(a *Args) error {
	if a.Src.Kind != EAkImm {
		return fmt.Errorf("FMOVECR requires an immediate ROM constant index")
	}
	if a.Src.Imm < 0 || a.Src.Imm > 0x7F {
		return fmt.Errorf("FMOVECR constant index must be 0-127")
	}
	if a.Dst.Kind != EAkFPn {
		return fmt.Errorf("FMOVECR requires an FPn destination")
	}
	return nil
}

// newFSaveRestoreDef builds FSAVE (store, isSave=true) or FRESTORE
// (load, isSave=false). Both take a single <ea> operand (parsed into
// Src, since neither has a second operand to swap it into Dst for).
//
// GAS's own opcode table lists these as two separate rows each — a
// general memory form (table literal 0xF100/0xF140, "&s"/"&s") and one
// dedicated to the one addressing mode real hardware actually needs it
// with (-(An) for FSAVE, since the state frame is written descending
// like MOVEM's predecrement form; (An)+ for FRESTORE, read back
// ascending — "−s"/"+s", table literal 0xF120/0xF158). But those two
// literals aren't actually two different encodings: 0xF120 is exactly
// 0xF100 with mode=4 (predecrement)'s own bits already OR'd in, and
// 0xF158 is exactly 0xF140 with mode=3 (postincrement)'s bits OR'd in —
// which is precisely what FSrcEA computes at encode time for any EA
// anyway. So one Form with the general-form literal as its base,
// combined with FSrcEA, already produces the *identical* bits GAS's
// "dedicated" row would for -(An)/(An)+ — the two-row split in GAS
// exists purely to state two different validation rules, not two
// different opcodes, so a single Form plus a Validate that accepts the
// union of both rules reproduces it exactly. Two separate Forms here
// would have been actively wrong, not just redundant: selectForm picks
// the first Form whose OperKinds/Sizes match before Validate ever runs,
// and both forms would share identical OperKinds ([OpkEA]) and Sizes —
// so whichever Form-with-a-narrower-Validate came first would silently
// swallow every operand, valid or not, and the other Form would never
// be reached.
//
// The 0xF100/0xF140 figures above are GAS's *table* literals, which
// exclude the coprocessor-ID field (bits 11-9) — FSAVE/FRESTORE use the
// same "Id" implicit-COP1-operand convention as every other FPU general
// instruction (see fpuWord1Base's doc comment), so the actual emitted
// word1 must OR in fpuWord1Base's 0x0200 cpid bit, giving 0xF300/0xF340.
// This was originally shipped as a bare 0xF100/0xF140 (cpid=0) — the
// exact same class of bug fpuWord1Base was introduced to fix for
// FMOVE/FADD/etc., just not applied here since this function predates
// that constant's use. Found while researching PSAVE's own encoding for
// a later milestone (docs/design/cpu-family-support.md); fixed by
// building word1 from fpuWord1Base instead of a bare 0xF1xx literal.
func newFSaveRestoreDef(name string, isSave bool) *InstrDef {
	word1 := uint16(fpuWord1Base | 0x0100)
	forbidden := EAkAddrPostinc // FSAVE allows -(An), not (An)+
	if !isSave {
		word1 = fpuWord1Base | 0x0140
		forbidden = EAkAddrPredec // FRESTORE allows (An)+, not -(An)
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
				Requires:    requireFPU,
				Steps: []EmitStep{
					{WordBits: word1, Fields: []FieldRef{FSrcEA}},
					{Trailer: []TrailerItem{TSrcEAExt}},
				},
			},
		},
	}
}
