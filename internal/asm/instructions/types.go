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

	// Aux is a third operand, for the handful of instructions with more
	// than the usual Src/Dst pair (CAS's <ea>, PACK/UNPK's #adjustment).
	// It is a full EAExpr rather than a plain int64 so it can hold
	// either an immediate (PACK/UNPK) or a general EA needing its own
	// extension words (CAS) — see FAuxEA/TAuxEAExt/TAuxImmWord.
	Aux EAExpr
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
