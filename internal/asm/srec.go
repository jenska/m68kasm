package asm

import (
	"fmt"
	"strings"
)

const srecDataBytesPerRecord = 16

// AssembleSRecord assembles the given program and returns its Motorola S-record
// representation. The provided header text is used for the S0 record; if empty,
// a default identifier is emitted.
func AssembleSRecord(p *Program, header string) ([]byte, error) {
	_, listing, err := assemble(nil, p, true)
	if err != nil {
		return nil, err
	}

	return FormatSRecords(listing, p.Origin, header), nil
}

// FormatSRecords converts listing metadata into Motorola S-record text. The
// header is placed into the S0 record; if header is empty, "m68kasm" is used.
func FormatSRecords(entries []ListingEntry, origin uint32, header string) []byte {
	if header == "" {
		header = "m68kasm"
	}
	segments := listingSegments(entries)

	lines := make([]string, 0, len(segments)+2)
	lines = append(lines, srecHeader(header))
	for _, seg := range segments {
		for offset := 0; offset < len(seg.Data); offset += srecDataBytesPerRecord {
			end := offset + srecDataBytesPerRecord
			if end > len(seg.Data) {
				end = len(seg.Data)
			}
			lines = append(lines, s3Record(seg.Addr+uint32(offset), seg.Data[offset:end]))
		}
	}
	lines = append(lines, s7Record(origin))

	return []byte(strings.Join(lines, "\n") + "\n")
}

type srecSegment struct {
	Addr uint32
	Data []byte
}

func listingSegments(entries []ListingEntry) []srecSegment {
	segs := make([]srecSegment, 0, len(entries))
	var current *srecSegment
	var nextAddr uint32

	for i := range entries {
		entry := entries[i]
		if len(entry.Bytes) == 0 {
			continue
		}

		if current == nil || entry.PC != nextAddr {
			segs = append(segs, srecSegment{Addr: entry.PC})
			current = &segs[len(segs)-1]
			nextAddr = entry.PC
		}

		current.Data = append(current.Data, entry.Bytes...)
		nextAddr += uint32(len(entry.Bytes))
	}

	return segs
}

func srecHeader(text string) string {
	if len(text) > 252 {
		text = text[:252]
	}
	return formatSRecord('0', 2, 0, []byte(text))
}

func s3Record(addr uint32, data []byte) string {
	return formatSRecord('3', 4, addr, data)
}

func s7Record(addr uint32) string {
	return formatSRecord('7', 4, addr, nil)
}

func formatSRecord(prefix rune, addrLen int, address uint32, data []byte) string {
	count := byte(addrLen + len(data) + 1)
	sum := uint32(count)

	var sb strings.Builder
	sb.WriteRune('S')
	sb.WriteRune(prefix)
	fmt.Fprintf(&sb, "%02X", count)

	for i := addrLen - 1; i >= 0; i-- {
		b := byte(address >> uint(i*8))
		sum += uint32(b)
		fmt.Fprintf(&sb, "%02X", b)
	}

	for _, b := range data {
		sum += uint32(b)
		fmt.Fprintf(&sb, "%02X", b)
	}

	checksum := byte(^sum)
	fmt.Fprintf(&sb, "%02X", checksum)
	return sb.String()
}
