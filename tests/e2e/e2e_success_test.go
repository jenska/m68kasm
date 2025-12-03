package e2e_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Test_Assemble_Hello invokes the CLI on testdata/hello.s from the repo
// and asserts that a non-empty binary is produced. It logs a SHA-256
// for regression tracking without hard-coding bytes.
func Test_Assemble_Hello(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	src := filepath.Join(repoRoot, "tests", "e2e", "testdata", "hello.s")

	outDir := t.TempDir()
	out := filepath.Join(outDir, "out.bin")

	cmd := exec.Command("go", "run", "./cmd/m68kasm", "-i", src, "-o", out)
	cmd.Dir = repoRoot
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
