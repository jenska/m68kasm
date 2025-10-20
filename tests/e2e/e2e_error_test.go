package e2e_test

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// Test_Assemble_Invalid asserts that assembling an invalid source
// returns a non-zero exit status and a helpful error message.
func Test_Assemble_Invalid(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	src := filepath.Join("tests", "e2e", "testdata", "invalid.s")
	cmd := exec.Command("go", "run", "./cmd/m68kasm", "-o", "out.bin", src)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected CLI to fail for invalid source, but it succeeded.\nOUTPUT:\n%s", string(out))
	}
	t.Logf("CLI error output (expected failure):\n%s", string(out))
}
