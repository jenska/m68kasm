package instructions

import "fmt"

func init() {
	registerInstrDef(&defNOP)
	registerInstrDef(&defRESET)
	registerInstrDef(&defTRAP)
	registerInstrDef(&defTRAPV)
	registerInstrDef(&defSTOP)
	registerInstrDef(&defRTS)
	registerInstrDef(&defRTE)
}

var defNOP = InstrDef{
	Mnemonic: "NOP",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{},
			Validate:    nil,
			Steps: []EmitStep{
				{WordBits: 0x4E71},
			},
		},
	},
}

var defTRAPV = InstrDef{
	Mnemonic: "TRAPV",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{},
			Validate:    nil,
			Steps: []EmitStep{
				{WordBits: 0x4E76},
			},
		},
	},
}

var defSTOP = InstrDef{
	Mnemonic: "STOP",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkImm},
			Validate:    validateSTOP,
			Steps: []EmitStep{
				{WordBits: 0x4E72},
				{Trailer: []TrailerItem{TImmSized}},
			},
		},
	},
}

var defRESET = InstrDef{
	Mnemonic: "RESET",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{},
			Validate:    nil,
			Steps: []EmitStep{
				{WordBits: 0x4E70},
			},
		},
	},
}

var defRTS = InstrDef{
	Mnemonic: "RTS",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{},
			Validate:    nil,
			Steps: []EmitStep{
				{WordBits: 0x4E75},
			},
		},
	},
}

var defRTE = InstrDef{
	Mnemonic: "RTE",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{},
			Validate:    nil,
			Steps: []EmitStep{
				{WordBits: 0x4E73},
			},
		},
	},
}

var defTRAP = InstrDef{
	Mnemonic: "TRAP",
	Forms: []FormDef{
		{
			DefaultSize: WordSize,
			Sizes:       []Size{WordSize},
			OperKinds:   []OperandKind{OpkImmQuick},
			Validate:    validateTRAP,
			Steps: []EmitStep{
				{WordBits: 0x4E40, Fields: []FieldRef{FImmLow8}},
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
	return checkImmediateRange(a.Src.Imm, WordSize)
}
