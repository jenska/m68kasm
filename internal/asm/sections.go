package asm

import "strings"

type SectionKind uint8

const (
	SectionText SectionKind = iota
	SectionData
	SectionBSS
)

func (s SectionKind) Name() string {
	switch s {
	case SectionText:
		return ".text"
	case SectionData:
		return ".data"
	case SectionBSS:
		return ".bss"
	default:
		return ".text"
	}
}

func parseSectionName(name string) (SectionKind, bool) {
	trimmed := strings.TrimSpace(strings.ToLower(name))
	switch trimmed {
	case "text", ".text":
		return SectionText, true
	case "data", ".data":
		return SectionData, true
	case "bss", ".bss":
		return SectionBSS, true
	default:
		return SectionText, false
	}
}

func sectionToELFIndex(section SectionKind) uint16 {
	switch section {
	case SectionText:
		return elfSectionText
	case SectionData:
		return elfSectionData
	case SectionBSS:
		return elfSectionBSS
	default:
		return elfSectionText
	}
}
