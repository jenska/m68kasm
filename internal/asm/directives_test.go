package asm_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenska/m68kasm/internal/asm"
	"github.com/jenska/m68kasm/internal/asm/instructions"
)

func TestLocationCounter(t *testing.T) {
	tests := []struct {
		name   string
		src    string
		target instructions.Target
		want   []byte
	}{
		{"BRA", "BRA *\n", instructions.Target{}, []byte{0x60, 0xFE}},
		{"DBRA", "DBRA D0,*\n", instructions.Target{}, []byte{0x51, 0xC8, 0xFF, 0xFE}},
		{"LEA", "LEA *+4,A0\n", instructions.Target{}, []byte{0x41, 0xF9, 0, 0, 0, 4}},
		{"ImmediateAfterNOP", "NOP\nMOVE.L #*,D0\n", instructions.Target{}, []byte{0x4E, 0x71, 0x20, 0x3C, 0, 0, 0, 2}},
		{"DCL", "NOP\nDC.L *\n", instructions.Target{}, []byte{0x4E, 0x71, 0, 0, 0, 2}},
		{"MultiplyStillWorks", "DC.L 2*3\n", instructions.Target{}, []byte{0, 0, 0, 6}},
		{"MultiplyByCounter", "NOP\nDC.L 3*(*)\n", instructions.Target{}, []byte{0x4E, 0x71, 0, 0, 0, 6}},
		{"FBEQ", "FBEQ *\n", targetFPU, []byte{0xF2, 0x81, 0xFF, 0xFE}},
		{"PDBIC", "PDBIC D0,*\n", instructions.Target{CPU: instructions.CPU68030, Features: instructions.FeatPMMU}, []byte{0xF0, 0x48, 0x00, 0x0B, 0xFF, 0xFC}},
		{"IndexScale", "MOVE.W (0,A0,D0.L*4),D1\n", instructions.Target{CPU: instructions.CPU68020}, []byte{0x32, 0x30, 0x0C, 0x00}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := assembleForTarget(t, tc.src, tc.target)
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("got % X want % X", got, tc.want)
			}
		})
	}
}

func TestDirectives(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []byte
	}{
		{"EQU", "VAL EQU 5\nMOVEQ #VAL,D0\n", []byte{0x70, 0x05}},
		{"EQUColon", "VAL: EQU 5\nMOVEQ #VAL,D0\n", []byte{0x70, 0x05}},
		{"EQUDot", "VAL .equ 5\nMOVEQ #VAL,D0\n", []byte{0x70, 0x05}},
		{"EQUForward", "MOVEQ #LATER,D0\nLATER EQU 3\n", []byte{0x70, 0x03}},
		{"SETRedefine", "X SET 1\nDC.B X\nX SET 2\nDC.B X\n", []byte{1, 2}},
		{"DSB", "DS.B 3\n", []byte{0, 0, 0}},
		{"DSW", "DC.B 1\nDS.W 1\nDC.B 2\n", []byte{1, 0, 0, 2}},
		{"DSL", "DS.L 1\n", []byte{0, 0, 0, 0}},
		{"DSZero", "DS.B 0\nDC.B 1\n", []byte{1}},
		{"DCBB", "DCB.B 3,$AA\n", []byte{0xAA, 0xAA, 0xAA}},
		{"DCBW", "DCB.W 2,$1234\n", []byte{0x12, 0x34, 0x12, 0x34}},
		{"DCBL", "DCB.L 2,$DEADBEEF\n", []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xDE, 0xAD, 0xBE, 0xEF}},
		{"DCBNegative", "DCB.B 2,-1\n", []byte{0xFF, 0xFF}},
		{"Space", ".SPACE 3\n", []byte{0, 0, 0}},
		{"Skip", ".skip 2\n", []byte{0, 0}},
		{"Ascii", ".ASCII \"abc\"\n", []byte("abc")},
		{"AsciiList", ".ASCII \"a\",\"b\"\n", []byte("ab")},
		{"Asciz", ".ASCIZ \"ab\"\n", []byte{'a', 'b', 0}},
		{"AscizList", ".ASCIZ \"a\",\"b\"\n", []byte{'a', 0, 'b', 0}},
		{"DCBString", "DC.B \"abc\"\n", []byte("abc")},
		{"DCBQuoted", "DC.B 'abc'\n", []byte("abc")},
		{"DCBMixed", "DC.B 1,\"ab\",2\n", []byte{1, 'a', 'b', 2}},
		{"DCBChars", "DC.B 'a','b'\n", []byte("ab")},
		{"DCBEscape", "DC.B \"a\\n\"\n", []byte{'a', '\n'}},
		{"End", "NOP\nEND\nNOP\n", []byte{0x4E, 0x71}},
		{"EndIgnoresRest", "NOP\n END start\n this is ' not valid\n", []byte{0x4E, 0x71}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := assembleForTarget(t, tc.src, instructions.Target{})
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("got % X want % X", got, tc.want)
			}
		})
	}
}

func TestDirectiveErrors(t *testing.T) {
	for _, src := range []string{
		"X EQU 1\nX EQU 2\n",
		"X EQU 1\nX SET 2\n",
		"X SET 1\nX EQU 2\n",
		"X = 1\nX EQU 2\n",
		"DS.B -1\n",
		"DS.X 2\n",
		"DS.B\n",
		"DS.L $10000000\n",
		"DCB.B 2,$100\n",
		"DCB.W 1,$10000\n",
		"DCB.B 2\n",
		"DCB.B -1,0\n",
		".SPACE -1\n",
		".ASCII\n",
		".ASCII 5\n",
		"DC.B 'abc\n",
		"INCLUDE \"x.s\"\n", // no directory to resolve against
		"INCBIN \"x.bin\"\n",
	} {
		t.Run(src, func(t *testing.T) { mustAssembleErr(t, src, instructions.Target{}) })
	}
}

func TestIncludeAndIncbin(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("main.s", "INCLUDE \"inc/defs.s\"\nMOVEQ #VAL,D0\nINCBIN \"data.bin\"\n")
	write("inc/defs.s", "VAL EQU 7\nINCLUDE \"more.s\"\n")
	write("inc/more.s", "DC.B $99\n")
	write("data.bin", "\x01\x02\x03")

	prog, err := asm.ParseFile(filepath.Join(dir, "main.s"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := asm.Assemble(prog)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte{0x99, 0x70, 0x07, 1, 2, 3}; !bytes.Equal(got, want) {
		t.Fatalf("got % X", got)
	}

	// END inside an included file ends the whole assembly.
	write("end.s", "NOP\nINCLUDE \"inc/endinc.s\"\nNOP\n")
	write("inc/endinc.s", "NOP\nEND\nNOP\n")
	prog, err = asm.ParseFile(filepath.Join(dir, "end.s"))
	if err != nil {
		t.Fatal(err)
	}
	if got, _ = asm.Assemble(prog); !bytes.Equal(got, []byte{0x4E, 0x71, 0x4E, 0x71}) {
		t.Fatalf("END in include: got % X", got)
	}

	// BaseDir lets a reader-based source include files.
	prog, err = asm.ParseWithOptions(strings.NewReader("INCBIN \"data.bin\"\n"), asm.ParseOptions{BaseDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ = asm.Assemble(prog); !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Fatalf("BaseDir INCBIN: got % X", got)
	}

	// Errors: missing file, self inclusion, error text names the file.
	write("self.s", "INCLUDE \"self.s\"\n")
	write("bad.s", "FROB\n")
	write("usesbad.s", "INCLUDE \"bad.s\"\n")
	for _, name := range []string{"missing.s", "self.s", "usesbad.s"} {
		src := "INCLUDE \"" + name + "\"\n"
		if name == "self.s" || name == "usesbad.s" {
			src = ""
		}
		var perr error
		if src == "" {
			_, perr = asm.ParseFile(filepath.Join(dir, name))
		} else {
			_, perr = asm.ParseWithOptions(strings.NewReader(src), asm.ParseOptions{BaseDir: dir})
		}
		if perr == nil {
			t.Fatalf("%s: expected an error", name)
		}
	}
	_, err = asm.ParseFile(filepath.Join(dir, "usesbad.s"))
	if err == nil || !strings.Contains(err.Error(), "bad.s") {
		t.Fatalf("error should name the included file: %v", err)
	}
}
