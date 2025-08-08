package codegen

import (
	"fmt"
	"os"
)

// WriteS19File writes the output as Motorola S-Record (S19) to the given filename.
// data: the full binary image, addr: starting address (typically 0).
func WriteS19File(filename string, data []byte, addr int) error {
	const recLen = 16 // 16 bytes per line
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create S19 file: %w", err)
	}
	defer f.Close()

	// S1 records (2-byte address)
	for i := 0; i < len(data); i += recLen {
		n := recLen
		if i+n > len(data) {
			n = len(data) - i
		}
		record := make([]byte, 0, 2*n+20)
		record = append(record, "S1"...)
		// Byte count: (n data bytes) + (2 address bytes) + (1 checksum)
		bcount := byte(n + 3)
		record = append(record, hexByte(bcount)...)
		hi := byte((addr + i) >> 8)
		lo := byte((addr + i) & 0xFF)
		record = append(record, hexByte(hi)...)
		record = append(record, hexByte(lo)...)
		sum := bcount + hi + lo
		for j := 0; j < n; j++ {
			b := data[i+j]
			record = append(record, hexByte(b)...)
			sum += b
		}
		// Checksum: one's complement of sum
		cs := ^sum
		record = append(record, hexByte(cs)...)
		record = append(record, '\n')
		_, err = f.Write(record)
		if err != nil {
			return fmt.Errorf("failed to write S19 record: %w", err)
		}
	}

	// S9 termination record (address 0)
	s9 := []byte("S9030000FC\n")
	_, err = f.Write(s9)
	if err != nil {
		return fmt.Errorf("failed to write S9 record: %w", err)
	}
	return nil
}