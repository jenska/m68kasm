package codegen

import (
	"fmt"
	"os"
)

// WriteHexFile writes the output as Intel HEX to the given filename.
// data: the full binary image, addr: starting address (typically 0).
func WriteHexFile(filename string, data []byte, addr int) error {
	const recLen = 16 // 16 bytes per line
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create hex file: %w", err)
	}
	defer f.Close()

	for i := 0; i < len(data); i += recLen {
		n := recLen
		if i+n > len(data) {
			n = len(data) - i
		}
		// Intel HEX record: :llaaaatt[dd...]cc
		// ll = byte count, aaaa = address, tt = record type (00=data)
		line := make([]byte, 0, 11+2*n)
		line = append(line, ':')
		line = append(line, hexByte(byte(n))...)
		line = append(line, hexByte(byte((addr+i)>>8))...)
		line = append(line, hexByte(byte((addr+i)&0xFF))...)
		line = append(line, "00"...)
		sum := byte(n) + byte((addr+i)>>8) + byte((addr+i)&0xFF) // byte count + address
		for j := 0; j < n; j++ {
			b := data[i+j]
			line = append(line, hexByte(b)...)
			sum += b
		}
		sum += 0 // record type is 0x00
		cs := byte(-int8(sum))
		line = append(line, hexByte(cs)...)
		line = append(line, '\n')
		_, err = f.Write(line)
		if err != nil {
			return fmt.Errorf("failed to write hex line: %w", err)
		}
	}
	// Write end-of-file record
	_, err = f.WriteString(":00000001FF\n")
	if err != nil {
		return fmt.Errorf("failed to write EOF hex record: %w", err)
	}
	return nil
}

func hexByte(b byte) []byte {
	const hexdigits = "0123456789ABCDEF"
	return []byte{hexdigits[b>>4], hexdigits[b&0xF]}
}