package instructions

import "fmt"

// This file adds CAS2, the dual-location compare-and-swap milestone 8
// deferred: its six registers (two compare, two update, two memory
// pointers) collapse naturally into exactly three colon-joined pairs —
// "CAS2.W Dc1:Dc2,Du1:Du2,(Rn1):(Rn2)" — which is exactly three
// operands from the parser's perspective, fitting the existing
// Src/Dst/Aux slots without any further generalization beyond what
// milestone 10's OpkRegPair already introduced.
//
// Encodings were cross-checked against GNU binutils' GAS m68k opcode
// table (opcodes/m68k-opc.c) for the two size words, and its
// install_operand logic (gas/config/tc-m68k.c) for the register
// bit-field layout — including a comment in the 'case '6'' handler
// explicitly flagging it as "a hack to force cas2l and cas2w cmds to be
// three words long," which is what first confirmed CAS2 has no
// classic <ea> at all: the entire first word is a fixed literal (mask
// 0xFFFF, no variable bits whatsoever), and both memory locations are
// specified as register-only pointers in the second and third words
// instead. See docs/design/cpu-family-support.md.

func init() {
	registerInstrDef(newCas2Def())
}

func newCas2Def() *InstrDef {
	form := func(sz Size, base uint16) FormDef {
		return FormDef{
			DefaultSize: sz,
			Sizes:       []Size{sz},
			OperKinds:   []OperandKind{OpkRegPair, OpkRegPair, OpkAnIndPair},
			Validate:    validateCas2,
			Requires:    require68020,
			Steps: []EmitStep{
				{WordBits: base},
				{WordBits: 0x0000, Fields: []FieldRef{FCas2Word2}},
				{WordBits: 0x0000, Fields: []FieldRef{FCas2Word3}},
			},
		}
	}
	return &InstrDef{
		Mnemonic: "CAS2",
		Forms: []FormDef{
			form(WordSize, 0x0CFC),
			form(LongSize, 0x0EFC),
		},
	}
}

func validateCas2(a *Args) error {
	if !a.Src.RegPairWide {
		return fmt.Errorf("CAS2 requires a Dc1:Dc2 compare-register pair")
	}
	if !a.Dst.RegPairWide {
		return fmt.Errorf("CAS2 requires a Du1:Du2 update-register pair")
	}
	return nil
}
