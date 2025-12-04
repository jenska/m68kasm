package e2e_test

import (
	"crypto/sha256"
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
	src := filepath.Join(root, "tests", "e2e", "testdata", "hello.s")

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
	src := filepath.Join(root, "tests", "e2e", "testdata", "hello.s")

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
	src := filepath.Join(root, "tests", "e2e", "testdata", "hello.s")

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
