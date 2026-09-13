package instructions

import "fmt"

type FieldRef uint16

const (
	FSrcEA FieldRef = iota
	FDstEA
	FSizeBits
	FAnReg
	FDnReg
	FImmLow8
	FBranchLow8
	FMoveDestEA
	FMoveSize
	FQuickData
	FSrcDnReg
	FSrcAnReg
	FMovemSize
	FAddaSize
	FSrcDnRegHi
	FDstRegLow
	// FCtrlRegSel places the 12-bit MOVEC control-register selector code
	// (see ControlRegisterSelector) into bits 0-11 of the current word.
	FCtrlRegSel
	// FExtRegSrc places the Src operand's register-specifier nibble (A/D
	// select in bit 3, register number in bits 2-0) into bits 15-12 of
	// the current word, per the MOVEC/MOVES extension-word format.
	FExtRegSrc
	// FExtRegDst is FExtRegSrc for the Dst operand.
	FExtRegDst
	// FFPFormat places the FPU source/destination format code (see
	// fpFormatCode in encode.go) into bits 12-10 of the current word.
	FFPFormat
	// FFPDstReg7 places the Dst operand's FPn register number into bits
	// 9-7 of the current word (the "F7" field in Motorola's own FPU
	// documentation).
	FFPDstReg7
	// FFPSrcReg7 is FFPDstReg7 for the Src operand, used when the FPn
	// register being encoded is the source (e.g. FMOVE FPn,<ea>, where
	// the ea occupies the Dst slot but the F7 bit field still names the
	// register being moved).
	FFPSrcReg7
	// FFPSrcReg10 places the Src operand's FPn register number into bits
	// 12-10 (the "F8" field), used for the register-to-register form's
	// source register and FTST's single register operand.
	FFPSrcReg10
	// FAuxEA places the Aux operand's mode/reg pair into bits 5-0 of the
	// current word — CAS's <ea> is its third operand, not Src/Dst.
	FAuxEA
	// FDstReg6 places the Dst operand's register number into bits 8-6 of
	// the current word — CAS's update register (Du); no other
	// instruction in this codebase uses this exact bit position.
	FDstReg6
	// FSrcRegNibble0 places the Src operand's register-specifier nibble
	// (A/D select in bit 3, register number in bits 2-0) into bits 3-0
	// of the current word — RTM's Rn, the same nibble shape MOVEC uses
	// at bits 15-12 (FExtRegSrc), just at the opposite end of the word.
	FSrcRegNibble0
	// FBFOffsetSrc places the Src operand's bit-field offset code (see
	// bitFieldSpecCode) into bits 11-6 of the current word.
	FBFOffsetSrc
	// FBFWidthSrc places the Src operand's bit-field width code into
	// bits 5-0 of the current word.
	FBFWidthSrc
	// FBFOffsetDst is FBFOffsetSrc for the Dst operand.
	FBFOffsetDst
	// FBFWidthDst is FBFWidthSrc for the Dst operand.
	FBFWidthDst
	// FDivRegQ places the Dst operand's quotient register (Reg2) into
	// bits 14-12 of the current word — DIVSL/DIVUL's Dq.
	FDivRegQ
	// FDivRegR places the Dst operand's remainder register (Reg) into
	// bits 2-0 of the current word — DIVSL/DIVUL's Dr.
	FDivRegR
	// FDivWide sets bit 10 when the Dst operand used the "Dr:Dq" form
	// (two distinct registers, 64-bit dividend) rather than the "Dq"
	// shorthand (32-bit dividend, remainder discarded).
	FDivWide
	// FCas2Word2 combines CAS2's first-of-each-pair fields into one
	// word: the pointer register (Aux.Reg) at bits 14-12, the update
	// register (Dst.Reg) at bits 8-6, and the compare register (Src.Reg)
	// at bits 2-0.
	FCas2Word2
	// FCas2Word3 is FCas2Word2 for the second of each pair (Aux.Reg2,
	// Dst.Reg2, Src.Reg2).
	FCas2Word3
	// FFPRomConst places FMOVECR's 7-bit ROM constant offset (Src.Imm)
	// into bits 6-0 of the current word — GAS's "C" install code.
	FFPRomConst
	// FFPRegMaskDst places the Dst operand's FPn register-list mask
	// (FPRegMaskDst, bits 0-7 = FP0-FP7) into the low byte of the
	// current word, unreversed — FMOVEM's "ea,list" load direction
	// (general memory or postincrement; GAS's word2 is identical for
	// both, so unlike the store direction below there is no runtime
	// mode check to make and no predecrement-load form to reverse for).
	FFPRegMaskDst
	// FFPDynDstReg4 places the Dst operand's Dn register number into
	// bits 7-4 of the current word — FMOVEM's dynamic-list "ea,Dn" load
	// form (Dn holds the FPn mask at runtime).
	FFPDynDstReg4
	// FFPMovemStoreWord2 computes FMOVEM's entire store-direction word2
	// (the 0xF000/0xE000 "general vs predecrement" selector bit combined
	// with the FPn register-list mask, reversed only for predecrement)
	// from the Dst EA's *runtime* mode — not from which Form matched.
	// GAS's general-memory and predecrement rows differ in word2 by more
	// than an EA artifact (unlike FSAVE/FRESTORE's equivalent forms),
	// so two Forms sharing identical OperKinds ([OpkFPRegList, OpkEA])
	// would hit the same selectForm hazard FSAVE/FRESTORE's doc comment
	// describes — the first Form would always win, making the second
	// unreachable. One Form with this single runtime-computed field
	// avoids that entirely, mirroring how TSrcRegMask (below) already
	// resolves integer MOVEM's identical general-vs-predecrement split
	// by checking p.DstEA.Mode directly instead of via a second Form.
	FFPMovemStoreWord2
	// FFPMovemDynStoreWord2 is FFPMovemStoreWord2 for FMOVEM's dynamic
	// (Dn-specified) store list — same general-vs-predecrement word2
	// split, but placing the Dn register number (bits 7-4) instead of a
	// static mask.
	FFPMovemDynStoreWord2
	// FCacheSel6 places the Src operand's cache-selector value
	// (EAkCacheSel's Reg, 0-3) into bits 7-6 of the current word —
	// CINV/CPUSH's "e" install code.
	FCacheSel6
	// FMove16Reg2_12 places the Dst operand's An register number into
	// bits 14-12 of the current word — MOVE16's register-to-register
	// form's second register ("(An1)+,(An2)+"; An1 sits in word1's low
	// 3 bits via the ordinary FSrcDnReg, reused for its generic "low 3
	// bits of SrcReg" shape).
	FMove16Reg2_12
	// FMove16AbsForm computes MOVE16's entire word1 for the "(An),
	// absolute-long" pairing from which side is actually (An) at
	// *runtime* (p.SrcEA.Mode == 2 selects "(An),ABS.L", word1 0xF610;
	// otherwise "ABS.L,(An)", word1 0xF618) — both directions classify
	// as plain OpkEA on both operands (neither -(An)/(An)+ nor a bare
	// (An) that's paired only with an absolute address has its own
	// OperandKind distinguishing which side is which), so two Forms
	// here would hit the identical selectForm hazard FFPMovemStoreWord2
	// documents above. TSrcEAExt and TDstEAExt are both included
	// unconditionally in this Form's Steps rather than choosing one:
	// (An) always contributes zero extension bytes (EncodeEA gives it
	// an empty Ext slice) and the absolute-long side always contributes
	// its 4-byte address, so including both trailers is correct
	// regardless of which side is which — no per-direction Step choice
	// needed there, only for this word.
	FMove16AbsForm
	// FSrcReg2Low places the Src operand's second register (Reg2, e.g.
	// a register-pair's Dn2) into bits 2-0 of the current word — CPU32's
	// TBLS/TBLU register-to-register form's second source register.
	FSrcReg2Low
	// FFPCtrlSelSrc10 places the Src operand's FPCR/FPSR/FPIAR selector
	// (FPCtrlMaskSrc, bits 0-2) into bits 12-10 of the current word —
	// FMOVEM's control-register store direction ("FPIAR/FPSR,<ea>").
	FFPCtrlSelSrc10
	// FFPCtrlSelDst10 is FFPCtrlSelSrc10 for the Dst operand — FMOVEM's
	// control-register load direction ("<ea>,FPIAR/FPSR").
	FFPCtrlSelDst10
	// FSrcRegShift2 places the Src operand's register number into bits
	// 4-2 of the current word — PMOVE's BADn/BACn store direction
	// ("BADn/BACn,<ea>"), where the register number occupies a
	// different bit position than every other PMMU register's selector.
	FSrcRegShift2
	// FDstRegShift2 is FSrcRegShift2 for the Dst operand — PMOVE's
	// BADn/BACn load direction ("<ea>,BADn/BACn").
	FDstRegShift2
	// FFCSpecWord places the Src operand's function-code specifier
	// (Src.FCMode/Reg/Imm) into the low bits of the current word:
	// 0x00|value for named SFC(0)/DFC(1), 0x08|Dn for a data register,
	// 0x10|value for an immediate FC — PFLUSH/PLOAD/PTEST's first
	// operand, always Src. See cpu030_pmmu_ptest.go.
	FFCSpecWord
	// FDstImmShift5 places the Dst operand's immediate value into bits
	// 9-5 of the current word — PFLUSH's mask operand ("FC,#mask[,<ea>]"),
	// the one case in this codebase where Dst holds a plain immediate
	// rather than a writable location.
	FDstImmShift5
	// FAuxImmShift10 places the Aux operand's immediate value into bits
	// 12-10 of the current word — PTEST's #level operand
	// ("FC,<ea>,#level[,An]").
	FAuxImmShift10
	// FAux2RegShift5 places the Aux2 operand's register number into
	// bits 9-5 of the current word — PTEST's optional trailing An
	// result register.
	FAux2RegShift5
)

type TrailerItem uint16

const (
	TSrcEAExt TrailerItem = iota
	TDstEAExt
	TImmSized
	TSrcImm
	TBranchWordIfNeeded
	// TBranchLongIfNeeded emits a 32-bit branch displacement (68020+
	// Bcc.L/BSR.L) when the resolved size is LongSize; a no-op otherwise.
	TBranchLongIfNeeded
	TSrcRegMask
	TDstRegMask
	// TAuxEAExt emits the Aux operand's own addressing-mode extension
	// words (e.g. a displacement) — CAS's <ea>.
	TAuxEAExt
	// TAuxImmWord unconditionally emits Aux.Imm as one 16-bit word —
	// PACK/UNPK's #adjustment, always exactly one word regardless of
	// the instruction's data size.
	TAuxImmWord
)

// DispSize is the encoded width of a 68020+ full-format base or outer
// displacement: absent entirely (DispNull), one word (DispWord), or two
// words (DispLong). This is distinct from Size (operand data size).
type DispSize uint8

const (
	DispNull DispSize = iota
	DispWord
	DispLong
)

type Size uint16

const (
	ByteSize Size = 0
	WordSize Size = 4
	LongSize Size = 8
	// SingleSize, DoubleSize, and ExtendedSize are FPU (68881/68882 or
	// integrated) floating-point formats: IEEE single, IEEE double, and
	// Motorola's 80-bit extended precision. Their mnemonic suffixes are
	// .s/.d/.x — note .s collides with the existing byte-branch alias
	// "s" in sizeFromIdent, which is why FPU mnemonics parse their size
	// suffix through the separate fpSizeFromIdent (parser_operand.go)
	// instead: the same letter means something different depending on
	// which instruction family it follows.
	SingleSize   Size = 12
	DoubleSize   Size = 16
	ExtendedSize Size = 20
)

type OperandKind uint16

const (
	OpkNone OperandKind = iota
	OpkImm
	OpkImmQuick
	OpkDn
	OpkAn
	OpkSR
	OpkCCR
	OpkUSP
	OpkPredecAn
	OpkRegList
	OpkEA
	OpkDispRel
	// OpkCtrlReg matches one of the MOVEC control registers (SFC, DFC,
	// USP, VBR). USP is also reachable as OpkUSP for MOVE USP,An/An,USP;
	// operandKindCompatible treats OpkUSP as satisfying an OpkCtrlReg
	// requirement too, so MOVEC's forms accept it without a duplicate kind.
	OpkCtrlReg
	// OpkFPn matches an FPU data register, FP0-FP7.
	OpkFPn
	// OpkRegPair matches DIVSL/DIVUL's "Dr:Dq" (or bare "Dq") operand,
	// and (with the colon always required — see validateCas2) CAS2's
	// "Dc1:Dc2" and "Du1:Du2" pairs.
	OpkRegPair
	// OpkAnIndPair matches CAS2's "(Rn1):(Rn2)" memory-pointer pair.
	OpkAnIndPair
	// OpkTC matches the 68030/68851 PMMU Translation Control register,
	// "TC" — PMOVE's only supported target for now; see cpu030_pmmu.go.
	OpkTC
	// OpkFPRegList matches FMOVEM's FPn register-list operand ("FP0-FP3",
	// "FP1/FP4/FP6", or a bare "FP0") — the FPU analogue of OpkRegList,
	// kept distinct because it parses a different register namespace
	// into a different Args field (FPRegMaskSrc/Dst, not RegMaskSrc/Dst)
	// and encodes into a different bit width/position.
	OpkFPRegList
	// OpkPostincAn matches "(An)+" exactly (rejecting every other
	// addressing mode) — the postincrement analogue of OpkPredecAn,
	// needed so MOVE16's register-to-register form ("(An)+,(An)+") is
	// distinguishable, at the OperandKind level, from its other forms
	// that pair a postincrement or plain (An) operand with an absolute
	// long address (see cpu040_misc.go and operandKindByEA).
	OpkPostincAn
	// OpkCacheSel matches CINV/CPUSH's cache-selector operand (NC/DC/
	// IC/BC).
	OpkCacheSel
	// OpkFPCtrlRegList matches FMOVEM's FPCR/FPSR/FPIAR control-register
	// list operand (a bare name, or a slash-separated combination) —
	// the deferred-until-now second half of FMOVEM (see
	// cpu020_fpu_movem2.go), kept distinct from OpkFPRegList: a
	// different, 3-bit selector field (bits 12-10 of word2, not the
	// 8-bit FP0-FP7 mask at bits 7-0) and different EA rules (a *single*
	// named register may target Dn/An, but a genuine multi-register
	// combination may not).
	OpkFPCtrlRegList
	// OpkCRP, OpkSRP, OpkTT0, OpkTT1, and OpkMMUSR match PMOVE's other
	// PMMU registers (see cpu030_pmmu2.go), each parsed the same way
	// OpkTC already is — a fixed name via parseExpectedSpecialRegister.
	// Kept as separate kinds, one per register, rather than a shared
	// "any PMMU register" kind: each ends up needing its own Form with
	// its own fully-fixed word2 literal (every one of these registers
	// has a compile-time-known selector value, so — like OpkTC's own
	// existing Forms — no runtime FieldRef is needed at all), and nothing
	// is gained by routing them through one dynamic dispatch point.
	OpkCRP
	OpkSRP
	OpkTT0
	OpkTT1
	OpkMMUSR
	// OpkDRP, OpkCAL, OpkVAL, OpkSCC, OpkAC, and OpkPCSR match PMOVE's
	// remaining fixed-selector PMMU registers (see cpu030_pmmu3.go) —
	// same shape as OpkCRP/OpkSRP/etc. above.
	OpkDRP
	OpkCAL
	OpkVAL
	OpkSCC
	OpkAC
	OpkPCSR
	// OpkBAD and OpkBAC match PMOVE's numbered breakpoint address/access
	// registers, BAD0-BAD7 and BAC0-BAC7 (see cpu030_pmmu4.go).
	OpkBAD
	OpkBAC
	// OpkFCSpec matches PFLUSH/PLOAD/PTEST's function-code specifier
	// operand — SFC, DFC, a plain Dn, or "#<imm>" (see cpu030_pmmu_ptest.go).
	OpkFCSpec
)

type InstrDef struct {
	Mnemonic string
	Forms    []FormDef
	// Priority determines matching order for overlapping opcode patterns.
	// Higher priority patterns are checked first.
	// Used for patterns that share opcode bits (e.g., MULU 0xC0C0 vs AND 0xC000).
	Priority int
}

// registrationOrder tracks which instruction families must register before others.
// This ensures more specific opcode patterns are registered before generic ones.
var registrationOrder = []string{
	// Phase 1: Specific multiply/divide patterns (0xC0C0, 0xC1C0, 0x80C0, 0x81C0)
	// must register before generic logical ops (AND 0xC000/0xC100, OR 0x8000/0x8100)
	"MULU", "MULS", "DIVU", "DIVS",
	// Phase 2: BCD patterns (ABCD 0xC100, SBCD 0x8100) before logical ops
	"ABCD", "SBCD",
	// Phase 3: All other instructions (no ordering constraint)
}

// registerInstrDef registers an instruction definition. This is the ONLY way to add
// instructions to the Instructions map. Direct map assignments are prohibited.
func registerInstrDef(def *InstrDef) {
	if Instructions[def.Mnemonic] != nil {
		panic(fmt.Errorf("instruction %s already registered", def.Mnemonic))
	}
	// Assign priority based on registration order
	for i, mnemonic := range registrationOrder {
		if def.Mnemonic == mnemonic {
			def.Priority = 1000 - i // Higher phase = higher priority
			break
		}
	}
	if def.Priority == 0 {
		def.Priority = 100 // Default priority for unordered instructions
	}
	Instructions[def.Mnemonic] = def
}

// Instructions is the global registry of instruction definitions.
// IMPORTANT: Always use registerInstrDef() to add instructions. Never assign directly to this map.
var Instructions = map[string]*InstrDef{}

type FormDef struct {
	DefaultSize Size
	Sizes       []Size
	OperKinds   []OperandKind
	Validate    func(*Args) error
	Steps       []EmitStep
	// Requires gates this form to targets for which Target.Supports(Requires)
	// is true. The zero value (Target68000) matches every target, so leaving
	// it unset keeps a form available everywhere, as before Target existed.
	Requires Target
}

type EmitStep struct {
	WordBits uint16
	Fields   []FieldRef
	Trailer  []TrailerItem
}

type Args struct {
	Target        string
	TargetAddr    int64
	HasTargetAddr bool
	Src, Dst      EAExpr
	Size          Size

	HasImmQuick bool
	RegMaskSrc  uint16
	RegMaskDst  uint16

	// FPRegMaskSrc/FPRegMaskDst hold FMOVEM's FPn register-list mask (one
	// bit per FP0-FP7), kept separate from RegMaskSrc/RegMaskDst (Dn/An)
	// even though both are plain uint16 bitmasks — the two are never
	// used together on one instruction, but sharing a field would make
	// operandKinds (assemble.go) unable to tell "an FPn list" from "a
	// Dn/An list" apart when picking an OperandKind.
	FPRegMaskSrc uint16
	FPRegMaskDst uint16

	// FPCtrlMaskSrc/FPCtrlMaskDst hold FMOVEM's FPCR/FPSR/FPIAR
	// control-register selection (bit 0 = FPIAR, bit 1 = FPSR, bit 2 =
	// FPCR — GAS's own bit assignment), kept separate from
	// FPRegMaskSrc/Dst for the same reason that field is kept separate
	// from RegMaskSrc/Dst: a different register namespace needs its own
	// OperandKind to be distinguishable during form matching.
	FPCtrlMaskSrc uint16
	FPCtrlMaskDst uint16

	// Aux is a third operand, for the handful of instructions with more
	// than the usual Src/Dst pair (CAS's <ea>, PACK/UNPK's #adjustment,
	// PFLUSH's optional <ea>, PTEST's #level).
	// It is a full EAExpr rather than a plain int64 so it can hold
	// either an immediate (PACK/UNPK) or a general EA needing its own
	// extension words (CAS) — see FAuxEA/TAuxEAExt/TAuxImmWord.
	Aux EAExpr

	// Aux2 is a fourth operand. Only PTEST's optional trailing An
	// result register ("PTESTR FC,<ea>,#level,An") needs it — no other
	// instruction in this codebase has more than three operands.
	Aux2 EAExpr
}

type EAExprKind uint16

const (
	EAkNone EAExprKind = iota
	EAkImm
	EAkDn
	EAkAn
	EAkAddrPredec
	EAkAddrPostinc
	EAkAddrInd
	EAkAddrDisp16
	EAkPCDisp16
	EAkIdxAnBrief
	EAkIdxPCBrief
	EAkAbsW
	EAkAbsL
	EAkSR
	EAkCCR
	EAkUSP
	// EAkSFC, EAkDFC, and EAkVBR are MOVEC control registers (68010+).
	// USP doubles as a control register too; see ControlRegisterSelector.
	EAkSFC
	EAkDFC
	EAkVBR
	// EAkMemPreAn/EAkMemPrePC and EAkMemPostAn/EAkMemPostPC are 68020+
	// memory-indirect addressing with an An or PC base: "([bd,An],Xn,od)"
	// (pre-indexed: indirection happens before adding the index) and
	// "([bd,An,Xn],od)" (post-indexed: the index is added before
	// indirection). See EAExpr's full-format fields and
	// docs/design/cpu-family-support.md.
	EAkMemPreAn
	EAkMemPrePC
	EAkMemPostAn
	EAkMemPostPC
	// EAkFPn is an FPU data register, FP0-FP7 (Reg holds 0-7). It never
	// goes through the generic OpkEA/EncodeEA machinery for its actual
	// bit placement (FPU register fields live at specific bit positions
	// in the extension word, via FFPDstReg7/FFPSrcReg7/FFPSrcReg10 —
	// see cpu020_fpu.go); it still needs an eaTable entry purely because
	// Encode unconditionally calls EncodeEA for any non-EAkNone operand.
	EAkFPn
	// EAkRegPair is DIVSL/DIVUL's "Dr:Dq" (or bare "Dq") operand: Reg
	// holds Dr (the remainder register — same as Dq when the shorthand
	// was used) and Reg2 holds Dq (the quotient register); RegPairWide
	// is true only when the user wrote the explicit "Dr:Dq" form, since
	// that changes bit 10 of the extension word (a 64-bit dividend)
	// even on the rare occasion Dr and Dq happen to be the same
	// register. Like EAkFPn, it never goes through the generic
	// OpkEA/EncodeEA machinery for its actual bit placement. CAS2 also
	// uses this kind for its "Dc1:Dc2"/"Du1:Du2" pairs (Reg/Reg2 hold
	// the two registers; RegPairWide is always true there — see
	// validateCas2, which rejects the bare shorthand for those).
	EAkRegPair
	// EAkAnIndPair is CAS2's "(Rn1):(Rn2)" memory-pointer pair: Reg and
	// Reg2 hold the two An register numbers directly (0-7) — unlike a
	// normal EAkAddrInd, there is no mode/reg pair to encode generically,
	// since these are register-specifier fields, not addressing modes.
	EAkAnIndPair
	// EAkTC is the PMMU Translation Control register, "TC" — a fixed,
	// unnumbered special register like EAkSR/EAkCCR/EAkUSP, not one of
	// several numbered instances.
	EAkTC
	// EAkCacheSel is one of CINV/CPUSH's cache-selector pseudo-registers
	// (NC/DC/IC/BC — none/data/instruction/both), with the numeric
	// selector (0-3, GAS's own encoding) held directly in Reg, the same
	// "numbered instance" shape EAkFPn uses.
	EAkCacheSel
	// EAkCRP, EAkSRP, EAkTT0, EAkTT1, and EAkMMUSR are PMOVE's other
	// PMMU registers (see cpu030_pmmu2.go) — each a fixed, unnumbered
	// special register like EAkTC, not a numbered instance.
	EAkCRP
	EAkSRP
	EAkTT0
	EAkTT1
	EAkMMUSR
	// EAkDRP, EAkCAL, EAkVAL, EAkSCC, EAkAC, and EAkPCSR are PMOVE's
	// remaining fixed, unnumbered PMMU registers (see cpu030_pmmu3.go).
	// EAkMMUSR itself now also matches the 68851's own name for the
	// same register, "PSR" — see the OpkMMUSR parser case.
	EAkDRP
	EAkCAL
	EAkVAL
	EAkSCC
	EAkAC
	EAkPCSR
	// EAkBAD and EAkBAC are PMOVE's numbered breakpoint address/access
	// registers, BAD0-BAD7 and BAC0-BAC7 (see cpu030_pmmu4.go) — a
	// "numbered instance" shape like EAkFPn, with the register number
	// (0-7) held directly in Reg, unlike every other PMMU register in
	// this file (each of which is its own single fixed register).
	EAkBAD
	EAkBAC
	// EAkFCSpec is PFLUSH/PLOAD/PTEST's function-code specifier operand
	// (SFC/DFC/Dn/#imm — see FCMode above and cpu030_pmmu_ptest.go).
	EAkFCSpec
)

type EAExpr struct {
	Kind   EAExprKind
	Reg    int
	Imm    int64
	Disp16 int32
	Index  EAIndex
	Abs16  uint16
	Abs32  uint32

	// The fields below are used only by the 68020+ full-format kinds
	// (EAkMemPreAn/PrePC/PostAn/PostPC). IndexPresent false means the
	// index is suppressed (Index is ignored); BaseDisp/OuterDisp hold the
	// raw displacement values, sized per BaseDispSize/OuterDispSize.
	IndexPresent  bool
	BaseDisp      int32
	BaseDispSize  DispSize
	OuterDisp     int32
	OuterDispSize DispSize

	// HasBitField and the four fields below attach a 68020 bit-field
	// specifier ("{offset:width}") to this EA — a suffix on the base
	// addressing mode (Kind/Reg/etc. above still describe that base
	// unchanged), not a new EAExprKind, since any data-alterable or
	// readable EA (including EAkDn) can carry one. Each of offset and
	// width is independently either a register number (IsReg true, Val
	// 0-7) or a literal (IsReg false, Val = the offset 0-31 or the
	// width 1-32, encoded as 0-31 with 32 wrapping to 0 by hardware
	// convention — see bitFieldSpecCode in encode.go).
	HasBitField   bool
	BFOffsetIsReg bool
	BFOffsetVal   int32
	BFWidthIsReg  bool
	BFWidthVal    int32

	// Reg2 and RegPairWide are used only by EAkRegPair: Reg2 is Dq (the
	// quotient register) alongside Reg's Dr (remainder), and
	// RegPairWide records whether the explicit "Dr:Dq" syntax was used
	// (as opposed to the bare-Dq shorthand, which sets Reg2 == Reg).
	Reg2        int
	RegPairWide bool

	// FCMode is used only by EAkFCSpec (PFLUSH/PLOAD/PTEST's function-
	// code specifier operand), distinguishing which of its three
	// alternate spellings this is: 0 = named SFC/DFC (the selector, 0
	// or 1, is in Reg), 1 = Dn (the register number is in Reg), 2 =
	// immediate (the value is in Imm). See cpu030_pmmu_ptest.go.
	FCMode int
}

type EAIndex struct {
	Reg   int
	IsA   bool
	Long  bool
	Scale uint8
	Disp8 int8
}

type EAEncoded struct {
	Mode, Reg int
	Ext       []uint16
}
