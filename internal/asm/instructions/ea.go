package instructions

import "fmt"

// EncodeEA converts an addressing expression into the mode/reg pair and any extension words.
func EncodeEA(e EAExpr) (EAEncoded, error) {
	var out EAEncoded
	switch e.Kind {
	case EAkDn:
		out.Mode, out.Reg = 0, e.Reg
	case EAkAn:
		out.Mode, out.Reg = 1, e.Reg
	case EAkAddrPredec:
		out.Mode, out.Reg = 4, e.Reg
	case EAkAddrInd:
		out.Mode, out.Reg = 2, e.Reg
	case EAkAddrDisp16:
		out.Mode, out.Reg = 5, e.Reg
		out.Ext = append(out.Ext, uint16(e.Disp16))
	case EAkPCDisp16:
		out.Mode, out.Reg = 7, 2
		out.Ext = append(out.Ext, uint16(e.Disp16))
	case EAkIdxAnBrief:
		out.Mode, out.Reg = 6, e.Reg
		out.Ext = append(out.Ext, encodeBriefIndex(e.Index))
	case EAkIdxPCBrief:
		out.Mode, out.Reg = 7, 3
		out.Ext = append(out.Ext, encodeBriefIndex(e.Index))
	case EAkAbsW:
		out.Mode, out.Reg = 7, 0
		out.Ext = append(out.Ext, e.Abs16)
	case EAkAbsL:
		out.Mode, out.Reg = 7, 1
		out.Ext = append(out.Ext, uint16(e.Abs32>>16), uint16(e.Abs32))
	case EAkImm:
		out.Mode, out.Reg = 7, 4
	case EAkNone:
		return EAEncoded{}, fmt.Errorf("unsupported EA kind: %d", e.Kind)
	default:
		return EAEncoded{}, fmt.Errorf("unsupported EA kind: %d", e.Kind)
	}
	return out, nil
}

func encodeBriefIndex(ix EAIndex) uint16 {
	hi := uint16(0)
	if ix.IsA {
		hi |= 1 << 7
	}
	hi |= (uint16(ix.Reg&7) << 4)
	if ix.Long {
		hi |= 1 << 3
	}
	switch ix.Scale {
	case 1:
	case 2:
		hi |= 1 << 1
	case 4:
		hi |= 2 << 1
	case 8:
		hi |= 3 << 1
	}
	return (hi << 8) | uint16(uint8(ix.Disp8))
}
