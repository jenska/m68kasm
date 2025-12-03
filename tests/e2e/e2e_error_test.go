package e2e_test

import (
	"bytes"
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
	cmd := exec.Command("go", "run", "./cmd/m68kasm", "-i", src, "-o", "out.bin")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected CLI to fail for invalid source, but it succeeded.\nOUTPUT:\n%s", string(out))
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("unexpected error type: %v", err)
	}
	if exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code, got %d", exitErr.ExitCode())
	}

	if !bytes.Contains(out, []byte("unknown mnemonic")) {
		t.Fatalf("expected parser error mentioning unknown mnemonic, got:\n%s", string(out))
	}

	if !bytes.Contains(out, []byte("exit status 2")) {
		t.Fatalf("expected go run to report program exit status 2, got:\n%s", string(out))
	}

	t.Logf("CLI error output (expected failure):\n%s", string(out))
}

// Test_Assemble_MissingInput asserts that the CLI fails fast when the required
// input flag is omitted.
func Test_Assemble_MissingInput(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "run", "./cmd/m68kasm")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected CLI to fail without -i, but it succeeded. OUTPUT:\n%s", string(out))
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("unexpected error type: %v", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("unexpected exit code: got %d want 1", exitErr.ExitCode())
	}

	if !bytes.Contains(out, []byte("Usage")) {
		t.Fatalf("expected usage message, got:\n%s", string(out))
	}

	t.Logf("CLI output without -i (expected failure):\n%s", string(out))
}
