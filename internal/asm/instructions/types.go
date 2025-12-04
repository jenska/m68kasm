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
	ByteSize Size = iota
	WordSize
	LongSize
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
}

func registerInstrDef(def *InstrDef) {
	if Instructions[def.Mnemonic] != nil {
		panic(fmt.Errorf("instruction %s already rgistered", def.Mnemonic))
	}
	Instructions[def.Mnemonic] = def
}

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
	Target   string
	Src, Dst EAExpr
	Size     Size

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
