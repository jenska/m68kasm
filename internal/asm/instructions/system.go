package instructions

import "fmt"

func init() {
	registerInstrDef(&DefNOP)
	registerInstrDef(&DefRESET)
	registerInstrDef(&DefTRAP)
	registerInstrDef(&DefTRAPV)
	registerInstrDef(&DefSTOP)
	registerInstrDef(&DefRTS)
	registerInstrDef(&DefRTE)
}

var DefNOP = InstrDef{
	Mnemonic: "NOP",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{},
			Validate:    nil,
			Steps: []EmitStep{
				{WordBits: 0x4E71},
			},
		},
	},
}

var DefTRAPV = InstrDef{
	Mnemonic: "TRAPV",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{},
			Validate:    nil,
			Steps: []EmitStep{
				{WordBits: 0x4E76},
			},
		},
	},
}

var DefSTOP = InstrDef{
	Mnemonic: "STOP",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_Imm},
			Validate:    validateSTOP,
			Steps: []EmitStep{
				{WordBits: 0x4E72},
				{Trailer: []TrailerItem{T_ImmSized}},
			},
		},
	},
}

var DefRESET = InstrDef{
	Mnemonic: "RESET",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{},
			Validate:    nil,
			Steps: []EmitStep{
				{WordBits: 0x4E70},
			},
		},
	},
}

var DefRTS = InstrDef{
	Mnemonic: "RTS",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{},
			Validate:    nil,
			Steps: []EmitStep{
				{WordBits: 0x4E75},
			},
		},
	},
}

var DefRTE = InstrDef{
	Mnemonic: "RTE",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{},
			Validate:    nil,
			Steps: []EmitStep{
				{WordBits: 0x4E73},
			},
		},
	},
}

var DefTRAP = InstrDef{
	Mnemonic: "TRAP",
	Forms: []FormDef{
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_ImmQuick},
			Validate:    validateTRAP,
			Steps: []EmitStep{
				{WordBits: 0x4E40, Fields: []FieldRef{F_ImmLow8}},
			},
		},
	},
}

func validateTRAP(a *Args) error {
	if a.Src.Imm < 0 || a.Src.Imm > 15 {
		return fmt.Errorf("TRAP vector out of range: %d", a.Src.Imm)
	}
	return nil
}

func validateSTOP(a *Args) error {
	if a.Src.Kind != EAkImm {
		return fmt.Errorf("STOP requires immediate operand")
	}
	return checkImmediateRange(a.Src.Imm, SZ_W)
}
