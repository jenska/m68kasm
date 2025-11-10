package instructions

import "fmt"

func validateMOVE(a *Args) error {
	if a.Src.Kind == EAkNone || a.Dst.Kind == EAkNone {
		return fmt.Errorf("MOVE requires source and destination")
	}
	if a.Dst.Kind == EAkImm {
		return fmt.Errorf("MOVE destination cannot be immediate")
	}
	if isPCRelativeKind(a.Dst.Kind) {
		return fmt.Errorf("MOVE destination cannot be PC-relative")
	}
	if a.Size == SZ_B {
		if a.Src.Kind == EAkAn {
			return fmt.Errorf("MOVE.B cannot read from address register")
		}
		if a.Dst.Kind == EAkAn {
			return fmt.Errorf("MOVE.B cannot write to address register")
		}
	}
	if a.Src.Kind == EAkImm {
		if err := checkImmediateRange(a.Imm, a.Size); err != nil {
			return err
		}
	}
	return nil
}

func validateMOVEQ(a *Args) error {
	if !a.HasImm {
		return fmt.Errorf("MOVEQ needs immediate")
	}
	if a.Imm < -128 || a.Imm > 127 {
		return fmt.Errorf("MOVEQ immediate out of range")
	}
	return nil
}

func validateAddSub(name string, a *Args) error {
	if a.Src.Kind == EAkNone || a.Dst.Kind != EAkDn {
		return fmt.Errorf("%s requires Dn destination", name)
	}
	if a.Size == SZ_B && a.Src.Kind == EAkAn {
		return fmt.Errorf("%s.B does not allow address register source", name)
	}
	if a.Src.Kind == EAkImm {
		if err := checkImmediateRange(a.Imm, a.Size); err != nil {
			return err
		}
	}
	return nil
}

func validateMULTI(a *Args) error {
	if a.Dst.Kind != EAkDn {
		return fmt.Errorf("MULTI destination must be data register")
	}
	if a.Size != SZ_W {
		return fmt.Errorf("MULTI operates on word size")
	}
	if a.Src.Kind == EAkImm {
		return fmt.Errorf("MULTI does not allow immediate source")
	}
	if a.Src.Kind == EAkAn {
		return fmt.Errorf("MULTI does not allow address register source")
	}
	return nil
}

func validateDIV(a *Args) error {
	if a.Dst.Kind != EAkDn {
		return fmt.Errorf("DIV destination must be data register")
	}
	if a.Size != SZ_W {
		return fmt.Errorf("DIV operates on word size")
	}
	if a.Src.Kind == EAkImm {
		return fmt.Errorf("DIV does not allow immediate source")
	}
	if a.Src.Kind == EAkAn {
		return fmt.Errorf("DIV does not allow address register source")
	}
	return nil
}

func validateCMP(a *Args) error {
	if a.Src.Kind == EAkNone || a.Dst.Kind != EAkDn {
		return fmt.Errorf("CMP requires Dn destination")
	}
	if a.Src.Kind == EAkImm {
		if err := checkImmediateRange(a.Imm, a.Size); err != nil {
			return err
		}
	}
	return nil
}

func checkImmediateRange(v int64, sz Size) error {
	switch sz {
	case SZ_B:
		if v < -128 || v > 255 {
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
