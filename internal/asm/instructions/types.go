package instructions

import "fmt"

type FieldRef uint16

const (
	F_SrcEA FieldRef = iota
	F_DstEA
	F_SizeBits
	F_AnReg
	F_DnReg
	F_ImmLow8
	F_BranchLow8
	F_MoveDestEA
	F_MoveSize
	F_QuickData
	F_SrcDnReg
	F_SrcAnReg
	F_MovemSize
	F_AddaSize
	F_SrcDnRegHi
	F_DstRegLow
)

type TrailerItem uint16

const (
	T_SrcEAExt TrailerItem = iota
	T_DstEAExt
	T_ImmSized
	T_SrcImm
	T_BranchWordIfNeeded
	T_SrcRegMask
	T_DstRegMask
)

type Size uint16

const (
	ByteSize Size = iota
	WordSize
	LongSize
)

type OperandKind uint16

const (
	OPK_None OperandKind = iota
	OPK_Imm
	OPK_ImmQuick
	OPK_Dn
	OPK_An
	OPK_SR
	OPK_CCR
	OPK_USP
	OPK_PredecAn
	OPK_RegList
	OPK_EA
	OPK_DispRel
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
