package asm

import "strings"

type kw int

const (
	KW_NONE kw = iota
	KW_MOVEQ
	KW_LEA
	KW_BRA
	KW_ORG
	KW_BYTE
)

func kwOf(s string) kw {
	s = strings.ToUpper(s)
	switch s {
	case "MOVEQ":
		return KW_MOVEQ
	case "LEA":
		return KW_LEA
	case "BRA":
		return KW_BRA
	case ".ORG":
		return KW_ORG
	case ".BYTE":
		return KW_BYTE
	default:
		return KW_NONE
	}
}
func isRegDn(s string) (bool, int) {
	if len(s) == 2 && (s[0] == 'd' || s[0] == 'D') {
		r := int(s[1] - '0')
		if 0 <= r && r <= 7 {
			return true, r
		}
	}
	return false, 0
}
func isRegAn(s string) (bool, int) {
	if len(s) == 2 && (s[0] == 'a' || s[0] == 'A') {
		r := int(s[1] - '0')
		if 0 <= r && r <= 7 {
			return true, r
		}
	}
	return false, 0
}
func isPC(s string) bool { return strings.EqualFold(s, "PC") }
