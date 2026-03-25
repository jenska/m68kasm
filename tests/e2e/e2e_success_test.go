package e2e_test

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Test_Assemble_Hello invokes the CLI on testdata/hello.s from the repo
// and asserts that a non-empty binary is produced. It logs a SHA-256
// for regression tracking without hard-coding bytes.
func Test_Assemble_Hello(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "tests", "testdata", "hello.s")

	outDir := t.TempDir()
	out := filepath.Join(outDir, "out.bin")

	cmd := exec.Command("go", "run", "./cmd/m68kasm", "-i", src, "-o", out)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI failed: %v\nOUTPUT:\n%s", err, string(outBytes))
	}

	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("output file missing: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("output file is empty")
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("cannot read output file: %v", err)
	}
	sum := sha256.Sum256(data)
	t.Logf("out.bin bytes=%d sha256=%s", len(data), hex.EncodeToString(sum[:]))
}

// Test_Assemble_Hello_Listing ensures the CLI can emit a source listing file
// alongside the assembled binary.
func Test_Assemble_Hello_Listing(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "tests", "testdata", "hello.s")

	outDir := t.TempDir()
	out := filepath.Join(outDir, "out.bin")
	listFile := filepath.Join(outDir, "out.lst")

	cmd := exec.Command("go", "run", "./cmd/m68kasm", "-i", src, "-o", out, "--list", listFile)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("CLI failed: %v\nOUTPUT:\n%s", err, string(outBytes))
	}

	data, err := os.ReadFile(listFile)
	if err != nil {
		t.Fatalf("cannot read listing file: %v", err)
	}

	listing := string(data)
	for _, want := range []string{
		"Line  Address",
		"0x00000010",
		"76 07",
		"0x00000028",
		"11 22 33 44",
	} {
		if !strings.Contains(listing, want) {
			t.Fatalf("listing missing %q\n%s", want, listing)
		}
	}
}

// Test_Assemble_SRecord_Output ensures the CLI can emit Motorola S-record text.
func Test_Assemble_SRecord_Output(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "tests", "testdata", "hello.s")

	outDir := t.TempDir()
	out := filepath.Join(outDir, "out.srec")

	cmd := exec.Command("go", "run", "./cmd/m68kasm", "-i", src, "-o", out, "--format", "srec")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("CLI failed: %v\nOUTPUT:\n%s", err, string(outBytes))
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("cannot read s-record file: %v", err)
	}

	text := string(data)
	for _, want := range []string{"S0", "S3", "S7", "00000010", "6D36386B61736D"} {
		if !strings.Contains(text, want) {
			t.Fatalf("s-record output missing %q\n%s", want, text)
		}
	}
}

// Test_Assemble_ELF_Output ensures the CLI can emit an ELF32 image for m68k.
func Test_Assemble_ELF_Output(t *testing.T) {
	root := repoRoot(t)
	src := filepath.Join(root, "tests", "testdata", "hello.s")

	outDir := t.TempDir()
	out := filepath.Join(outDir, "out.elf")

	cmd := exec.Command("go", "run", "./cmd/m68kasm", "-i", src, "-o", out, "--format", "elf")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("CLI failed: %v\nOUTPUT:\n%s", err, string(outBytes))
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("cannot read elf file: %v", err)
	}

	if string(data[:4]) != "\x7fELF" {
		t.Fatalf("missing ELF magic: %x", data[:4])
	}
	// Entry point should match the .org in hello.s (0x10)
	entry := binary.BigEndian.Uint32(data[24:28])
	if entry != 0x10 {
		t.Fatalf("unexpected entry point: 0x%X", entry)
	}

	shoff := int(binary.BigEndian.Uint32(data[32:36]))
	if shoff == 0 {
		t.Fatalf("expected section headers to be present")
	}
	shnum := int(binary.BigEndian.Uint16(data[48:50]))
	if shnum < 7 {
		t.Fatalf("expected standard ELF sections, got %d", shnum)
	}

	shstrtabIndex := int(binary.BigEndian.Uint16(data[50:52]))
	shstrtab := readELFSectionHeader(data, shoff, shstrtabIndex)
	sectionNames := data[shstrtab.offset : shstrtab.offset+shstrtab.size]

	textSection := readELFSectionHeader(data, shoff, 1)
	if got := readELFString(sectionNames, textSection.name); got != ".text" {
		t.Fatalf("unexpected text section name: %q", got)
	}

	symtabSection := readELFSectionHeader(data, shoff, 4)
	if got := readELFString(sectionNames, symtabSection.name); got != ".symtab" {
		t.Fatalf("unexpected symtab section name: %q", got)
	}

	strtabSection := readELFSectionHeader(data, shoff, 5)
	strtab := data[strtabSection.offset : strtabSection.offset+strtabSection.size]
	foundStart := false
	for off := 0; off+16 <= len(data[symtabSection.offset:symtabSection.offset+symtabSection.size]); off += 16 {
		entry := data[symtabSection.offset+off : symtabSection.offset+off+16]
		nameOff := binary.BigEndian.Uint32(entry[0:4])
		value := binary.BigEndian.Uint32(entry[4:8])
		if readELFString(strtab, nameOff) == "start" {
			foundStart = true
			if value != 0x10 {
				t.Fatalf("unexpected start symbol value: 0x%X", value)
			}
		}
	}
	if !foundStart {
		t.Fatalf("expected start symbol in ELF symtab")
	}
}

type elfSectionHeader struct {
	name   uint32
	offset int
	size   int
}

func readELFSectionHeader(data []byte, shoff, index int) elfSectionHeader {
	base := shoff + index*40
	return elfSectionHeader{
		name:   binary.BigEndian.Uint32(data[base:]),
		offset: int(binary.BigEndian.Uint32(data[base+16:])),
		size:   int(binary.BigEndian.Uint32(data[base+20:])),
	}
}

func readELFString(table []byte, off uint32) string {
	if off >= uint32(len(table)) {
		return ""
	}
	end := off
	for end < uint32(len(table)) && table[end] != 0 {
		end++
	}
	return string(table[off:end])
}
