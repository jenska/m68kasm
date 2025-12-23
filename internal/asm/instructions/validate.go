package instructions

import "fmt"

func validateControlEA(name string, a *Args) error {
	if a.Dst.Kind == EAkNone && a.Src.Kind != EAkNone {
		a.Dst = a.Src
		a.Src = EAExpr{}
	}
	switch a.Dst.Kind {
	case EAkAddrInd, EAkAddrDisp16, EAkIdxAnBrief, EAkAbsW, EAkAbsL, EAkPCDisp16, EAkIdxPCBrief:
		return nil
	case EAkNone:
		return fmt.Errorf("%s requires destination", name)
	default:
		return fmt.Errorf("%s requires control addressing mode", name)
	}
}

func checkImmediateRange(v int64, sz Size) error {
	switch sz {
	case ByteSize:
		if v < -128 || v > 255 {
			return fmt.Errorf("immediate out of range for .b: %d", v)
		}
	case WordSize:
		if v < -32768 || v > 65535 {
			return fmt.Errorf("immediate out of range for .w: %d", v)
		}
	case LongSize:
		if v < -0x80000000 || v > 0xFFFFFFFF {
			return fmt.Errorf("immediate out of range for .l: %d", v)
		}
	}
	return nil
}

func isPCRelativeKind(k EAExprKind) bool {
	switch k {
	case EAkPCDisp16, EAkIdxPCBrief:
		return true
	default:
		return false
	}
}
