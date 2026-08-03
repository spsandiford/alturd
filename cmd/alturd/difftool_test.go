package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestDifftoolModeMissingFlags asserts that supplying only --difftool-local
// (with --difftool-remote and --difftool-path omitted) exits 1 with the
// exact all-three-required message (DIFFTOOL-01).
func TestDifftoolModeMissingFlags(t *testing.T) {
	stateDir := t.TempDir()

	cmd := exec.Command(alturdBin, "--difftool-local", "/tmp/does-not-matter")
	cmd.Dir = filepath.Join("..", "..")
	cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+stateDir)

	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("alturd --difftool-local only exited 0, want exit 1")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected exec.ExitError, got %T: %v", err, err)
	}
	if code := exitErr.ExitCode(); code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}

	wantMsg := "--difftool-local, --difftool-remote and --difftool-path must all be provided"
	if !strings.Contains(string(out), wantMsg) {
		t.Errorf("expected output to contain %q, got: %q", wantMsg, string(out))
	}
}

// TestDifftoolModeIdenticalFiles asserts that pointing --difftool-local and
// --difftool-remote at two identical temp files exits 0 and prints
// "No changes found." — mirroring the standalone empty-state guard
// (DIFFTOOL-01).
func TestDifftoolModeIdenticalFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH — skipping difftool smoke test")
	}

	stateDir := t.TempDir()
	dir := t.TempDir()

	localPath := filepath.Join(dir, "local.go")
	remotePath := filepath.Join(dir, "remote.go")
	content := []byte("package main\n\nfunc main() {}\n")
	if err := os.WriteFile(localPath, content, 0o644); err != nil {
		t.Fatalf("write local file: %v", err)
	}
	if err := os.WriteFile(remotePath, content, 0o644); err != nil {
		t.Fatalf("write remote file: %v", err)
	}

	cmd := exec.Command(alturdBin,
		"--difftool-local", localPath,
		"--difftool-remote", remotePath,
		"--difftool-path", "foo.go",
	)
	cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+stateDir)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("alturd difftool identical-files run exited non-zero: %v\noutput: %s", err, out)
	}
	if !strings.Contains(string(out), "No changes found.") {
		t.Errorf("expected 'No changes found.' in output, got: %q", string(out))
	}
}
