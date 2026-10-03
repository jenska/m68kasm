package instructions

import "fmt"

// This file holds the CPU32/68020+ long divides, following the operand
// syntax of the M68000 Family Programmer's Reference Manual:
//
//	DIVU.L/DIVS.L   <ea>,Dq      32/32 -> 32q
//	DIVU.L/DIVS.L   <ea>,Dr:Dq   64/32 -> 32r:32q
//	DIVUL.L/DIVSL.L <ea>,Dr:Dq   32/32 -> 32r:32q
//
// All share 0100 1100 01 eeeeee with the extension word
// 0qqq sz00 0000 0rrr: s selects the signed divide and z the 64-bit
// dividend. GNU as spells the 64-bit form "divul"/"divsl" and Motorola's
// DIVUL/DIVSL "divull"/"divsll"; this assembler uses Motorola's names.
//
// They are tagged m68020up|cpu32 in GAS — CPU32 has them too — so they're
// gated on Target{CPU: CPU32}, not Target{CPU: CPU68020}.
var requireDiv = Target{CPU: CPU32}

func init() {
	registerInstrDef(&InstrDef{Mnemonic: "DIVSL", Forms: []FormDef{newDivLongForm("DIVSL", 0x0800, false)}})
	registerInstrDef(&InstrDef{Mnemonic: "DIVUL", Forms: []FormDef{newDivLongForm("DIVUL", 0x0000, false)}})
}

// newDivLongForm builds a long divide. wide reports whether a "Dr:Dq"
// operand selects the 64-bit dividend (DIVU.L/DIVS.L) rather than a 32-bit
// one with the remainder kept (DIVUL/DIVSL); a bare "Dq" always divides 32
// bits, with Dr = Dq.
func newDivLongForm(name string, signBit uint16, wide bool) FormDef {
	fields := []FieldRef{FDivRegQ, FDivRegR}
	if wide {
		fields = append(fields, FDivWide)
	}
	return FormDef{
		DefaultSize: LongSize,
		Sizes:       []Size{LongSize},
		OperKinds:   []OperandKind{OpkEA, OpkRegPair},
		Validate:    func(a *Args) error { return validateDiv(name, a) },
		Requires:    requireDiv,
		Steps: []EmitStep{
			{WordBits: 0x4C40, Fields: []FieldRef{FSrcEA}},
			{WordBits: signBit, Fields: fields},
			{Trailer: []TrailerItem{TSrcEAExt, TSrcImm}},
		},
	}
}

func validateDiv(name string, a *Args) error {
	if a.Src.Kind == EAkAn {
		return fmt.Errorf("%s does not support address register operands", name)
	}
	if !readableDataEA[a.Src.Kind] {
		if a.Src.Kind == EAkNone {
			return fmt.Errorf("%s requires a source operand", name)
		}
		return fmt.Errorf("%s source must be a readable addressing mode", name)
	}
	return nil
}
