package instructions

import "fmt"

// PVALID is the MC68851's pointer validation (MC68851 User's Manual,
// appendix A): "PVALID VAL,<ea>" compares the access level of <ea> with
// the VAL register, "PVALID An,<ea>" with the upper bits of An. Word2 is
// 0x2800 for VAL and 0x2C00|n for An. Only control alterable modes are
// allowed. The MC68030 does not have it, so the forms require exactly the
// MC68020 with the PMMU feature, which means an MC68851.
var requireMC68851 = Target{CPU: CPU68020, Features: featM68020Only | FeatPMMU}

func init() {
	validate := func(a *Args) error {
		if !controlAlterableEA[a.Dst.Kind] {
			if a.Dst.Kind == EAkNone {
				return fmt.Errorf("PVALID requires an effective address")
			}
			return fmt.Errorf("PVALID requires a control alterable addressing mode")
		}
		return nil
	}
	registerInstrDef(&InstrDef{
		Mnemonic: "PVALID",
		Forms: []FormDef{
			{
				DefaultSize: LongSize,
				Sizes:       []Size{LongSize},
				OperKinds:   []OperandKind{OpkVAL, OpkEA},
				Validate:    validate,
				Requires:    requireMC68851,
				Steps: []EmitStep{
					{WordBits: 0xF000, Fields: []FieldRef{FDstEA}},
					{WordBits: 0x2800},
					{Trailer: []TrailerItem{TDstEAExt}},
				},
			},
			{
				DefaultSize: LongSize,
				Sizes:       []Size{LongSize},
				OperKinds:   []OperandKind{OpkAn, OpkEA},
				Validate:    validate,
				Requires:    requireMC68851,
				Steps: []EmitStep{
					{WordBits: 0xF000, Fields: []FieldRef{FDstEA}},
					{WordBits: 0x2C00, Fields: []FieldRef{FSrcAnReg}},
					{Trailer: []TrailerItem{TDstEAExt}},
				},
			},
		},
	})
}
