package instructions

import "fmt"

func init() {
	registerInstrDef(&defORIImm)
	registerInstrDef(&defANDIImm)
	registerInstrDef(&defEORIImm)
}

var defORIImm = InstrDef{
	Mnemonic: "ORI",
	Forms: []FormDef{
		{
			DefaultSize: SZ_B,
			Sizes:       []Size{SZ_B},
			OperKinds:   []OperandKind{OPK_Imm, OPK_CCR},
			Validate:    validateCCRImmediate,
			Steps: []EmitStep{
				{WordBits: 0x003C},
				{Trailer: []TrailerItem{T_ImmSized}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_Imm, OPK_SR},
			Validate:    validateSRImmediate,
			Steps: []EmitStep{
				{WordBits: 0x007C},
				{Trailer: []TrailerItem{T_ImmSized}},
			},
		},
	},
}

var defANDIImm = InstrDef{
	Mnemonic: "ANDI",
	Forms: []FormDef{
		{
			DefaultSize: SZ_B,
			Sizes:       []Size{SZ_B},
			OperKinds:   []OperandKind{OPK_Imm, OPK_CCR},
			Validate:    validateCCRImmediate,
			Steps: []EmitStep{
				{WordBits: 0x023C},
				{Trailer: []TrailerItem{T_ImmSized}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_Imm, OPK_SR},
			Validate:    validateSRImmediate,
			Steps: []EmitStep{
				{WordBits: 0x027C},
				{Trailer: []TrailerItem{T_ImmSized}},
			},
		},
	},
}

var defEORIImm = InstrDef{
	Mnemonic: "EORI",
	Forms: []FormDef{
		{
			DefaultSize: SZ_B,
			Sizes:       []Size{SZ_B},
			OperKinds:   []OperandKind{OPK_Imm, OPK_CCR},
			Validate:    validateCCRImmediate,
			Steps: []EmitStep{
				{WordBits: 0x0A3C},
				{Trailer: []TrailerItem{T_ImmSized}},
			},
		},
		{
			DefaultSize: SZ_W,
			Sizes:       []Size{SZ_W},
			OperKinds:   []OperandKind{OPK_Imm, OPK_SR},
			Validate:    validateSRImmediate,
			Steps: []EmitStep{
				{WordBits: 0x0A7C},
				{Trailer: []TrailerItem{T_ImmSized}},
			},
		},
	},
}

func validateSRImmediate(a *Args) error {
	if a.Dst.Kind != EAkSR {
		return fmt.Errorf("requires SR destination")
	}
	return checkImmediateRange(a.Src.Imm, SZ_W)
}

func validateCCRImmediate(a *Args) error {
	if a.Dst.Kind != EAkCCR {
		return fmt.Errorf("requires CCR destination")
	}
	return checkImmediateRange(a.Src.Imm, SZ_B)
}
