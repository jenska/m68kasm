package instructions

import "fmt"

type FieldRef int

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
)

type TrailerItem int

const (
	T_SrcEAExt TrailerItem = iota
	T_DstEAExt
	T_ImmSized
	T_SrcImm
	T_BranchWordIfNeeded
)

type Size int

const (
	SZ_B Size = 0
	SZ_W Size = 4
	SZ_L Size = 8
)

type OperandKind int

type cond uint8

const (
	condT   cond = 0x0
	condBSR cond = 0x1
	condHI  cond = 0x2
	condLS  cond = 0x3
	condCC  cond = 0x4
	condCS  cond = 0x5
	condNE  cond = 0x6
	condEQ  cond = 0x7
	condVC  cond = 0x8
	condVS  cond = 0x9
	condPL  cond = 0xA
	condMI  cond = 0xB
	condGE  cond = 0xC
	condLT  cond = 0xD
	condGT  cond = 0xE
	condLE  cond = 0xF
)

const (
	OPK_None OperandKind = iota
	OPK_Imm
	OPK_ImmQuick
	OPK_Dn
	OPK_An
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
}

type EAExprKind int

const (
	EAkNone EAExprKind = iota
	EAkImm
	EAkDn
	EAkAn
	EAkAddrInd
	EAkAddrDisp16
	EAkPCDisp16
	EAkIdxAnBrief
	EAkIdxPCBrief
	EAkAbsW
	EAkAbsL
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
