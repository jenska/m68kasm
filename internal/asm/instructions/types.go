package instructions

// Opcode identifies each supported instruction.
type Opcode int

const (
	OP_MOVEQ Opcode = iota
	OP_MOVE
	OP_ADD
	OP_SUB
	OP_MULTI
	OP_DIV
	OP_CMP
	OP_LEA
	OP_BCC
)

type FieldRef int

const (
	F_SrcEA FieldRef = iota
	F_DstEA
	F_SizeBits
	F_AnReg
	F_DnReg
	F_Cond
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
	SZ_B Size = iota
	SZ_W
	SZ_L
)

type Cond uint8

const (
	CondT   Cond = 0x0
	CondBSR Cond = 0x1
	CondHI  Cond = 0x2
	CondLS  Cond = 0x3
	CondCC  Cond = 0x4
	CondCS  Cond = 0x5
	CondNE  Cond = 0x6
	CondEQ  Cond = 0x7
	CondVC  Cond = 0x8
	CondVS  Cond = 0x9
	CondPL  Cond = 0xA
	CondMI  Cond = 0xB
	CondGE  Cond = 0xC
	CondLT  Cond = 0xD
	CondGT  Cond = 0xE
	CondLE  Cond = 0xF
)

type OperandKind int

const (
	OPK_None OperandKind = iota
	OPK_Imm
	OPK_Dn
	OPK_An
	OPK_EA
	OPK_DispRel
)

type InstrDef struct {
	Op       Opcode
	Mnemonic string
	Forms    []FormDef
}

type FormDef struct {
	Sizes     []Size
	OperKinds []OperandKind
	Validate  func(*Args) error
	Steps     []EmitStep
}

type EmitStep struct {
	WordBits uint16
	Fields   []FieldRef
	Trailer  []TrailerItem
}

type Args struct {
	HasImm   bool
	Imm      int64
	Dn, An   int
	Cond     Cond
	Target   string
	Src, Dst EAExpr
	Size     Size
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
