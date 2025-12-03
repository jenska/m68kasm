package asm

import "strings"

type kw int

// add to the kw enum
const (
	KW_NONE kw = iota
	KW_ORG
	KW_ALIGN
	KW_BYTE
	KW_WORD
	KW_LONG
)

func kwOf(s string) kw {
	s = strings.ToUpper(s)
	if strings.HasPrefix(s, ".") {
		s = s[1:]
	}
	if idx := strings.IndexRune(s, '.'); idx > 0 {
		s = s[:idx]
	}

	return kwMap[s]
}

var kwMap = map[string]kw{
	"":      KW_NONE,
	"ORG":   KW_ORG,
	"BYTE":  KW_BYTE,
	"WORD":  KW_WORD,
	"LONG":  KW_LONG,
	"ALIGN": KW_ALIGN,
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
	return strings.EqualFold(s, "SP"), 7
}

func isPC(s string) bool { return strings.EqualFold(s, "PC") }
