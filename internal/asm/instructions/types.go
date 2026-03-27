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
)

type TrailerItem uint16

const (
	TSrcEAExt TrailerItem = iota
	TDstEAExt
	TImmSized
	TSrcImm
	TBranchWordIfNeeded
	TSrcRegMask
	TDstRegMask
)

type Size uint16

const (
	ByteSize Size = 0
	WordSize Size = 4
	LongSize Size = 8
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
)

type EAExpr struct {
	Kind   EAExprKind
	Reg    int
	Imm    int64
	Disp16 int32
	Index  EAIndex
	Abs16  uint16
	Abs32  uint32
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
