package asm

type Size int

const (
	SZ_B Size = iota
	SZ_W
	SZ_L
)

type Opcode int

const (
	OP_MOVEQ Opcode = iota
	OP_LEA
	OP_BCC
)

type Cond uint8

const (
	CondT Cond = 0x0
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

type Program struct {
	Items  []any
	Labels map[string]uint32
	Origin uint32
}
type Instr struct {
	Op       Opcode
	Mnemonic string
	Size     Size
	Args     Args
	PC       uint32
	Line     int
}
type DataBytes struct {
	Bytes []byte
	PC    uint32
	Line  int
}
type Args struct {
	HasImm   bool
	Imm      int64
	Dn, An   int
	Cond     Cond
	Target   string
	Src, Dst EAExpr
}
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
