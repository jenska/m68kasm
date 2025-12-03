package instructions

import "fmt"

func checkImmediateRange(v int64, sz Size) error {
	switch sz {
	case SZ_B:
		if v < -128 || v > 128 {
			return fmt.Errorf("immediate out of range for .b: %d", v)
		}
	case SZ_W:
		if v < -32768 || v > 65535 {
			return fmt.Errorf("immediate out of range for .w: %d", v)
		}
	case SZ_L:
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
