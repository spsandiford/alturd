// Package main_test contains integration tests that build and exec the alturd
// binary as a subprocess. Because main() calls os.Exit, testing via an
// in-process main() call is not safe — subprocess testing is required.
package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// alturdBin is the path to the binary built once in TestMain.
// It persists for the duration of the test binary run.
var alturdBin string

// TestMain builds the alturd binary once before all tests, then tears it down
// after. Building once (rather than per-test) avoids repeated compilation cost
// and ensures all subtests exercise the same binary.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "alturd-test-bin-*")
	if err != nil {
		panic("failed to create temp dir for binary: " + err.Error())
	}
	alturdBin = filepath.Join(dir, "alturd")
	// go test cwd is the package directory (cmd/alturd); go up two levels to
	// reach the module root so 'go build ./cmd/alturd' resolves correctly.
	buildCmd := exec.Command("go", "build", "-o", alturdBin, "./cmd/alturd")
	buildCmd.Dir = filepath.Join("..", "..")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		panic("failed to build alturd: " + err.Error() + "\noutput: " + string(out))
	}

	code := m.Run()
	os.RemoveAll(dir) // explicit cleanup before exit — defer is bypassed by os.Exit
	os.Exit(code)
}

// TestVersionExitsZeroNoLog asserts that --version exits 0 and creates no log
// file under the isolated XDG_STATE_HOME (GIT-04, D-10, T-02-08).
func TestVersionExitsZeroNoLog(t *testing.T) {
	stateDir := t.TempDir()

	cmd := exec.Command(alturdBin, "--version")
	cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+stateDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("--version exited non-zero: %v", err)
	}

	logPath := filepath.Join(stateDir, "alturd", "alturd.log")
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("--version must not create log file, but %s exists (or Stat err: %v)", logPath, err)
	}
}

// TestHelpExitsZeroNoLog asserts that --help exits 0 and creates no log file
// under the isolated XDG_STATE_HOME (GIT-04, D-10, T-02-08).
func TestHelpExitsZeroNoLog(t *testing.T) {
	stateDir := t.TempDir()

	cmd := exec.Command(alturdBin, "--help")
	cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+stateDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("--help exited non-zero: %v", err)
	}

	logPath := filepath.Join(stateDir, "alturd", "alturd.log")
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("--help must not create log file, but %s exists (or Stat err: %v)", logPath, err)
	}
}

// TestSmokeRunInRepoExitsZero asserts that running alturd with no args inside
// the alturd git repository exits 0 (proves the full
// ParseRefArgs -> ExecRunner -> diff.Parse -> diff.Render -> stdout path).
// The test is skipped if git is not on PATH.
func TestSmokeRunInRepoExitsZero(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH — skipping smoke test")
	}

	stateDir := t.TempDir()

	// Run with no args from the module root (a valid git repo with git 2.39.5).
	// go test cwd is cmd/alturd; go two levels up to reach the repo root.
	repoRoot := filepath.Join("..", "..")

	cmd := exec.Command(alturdBin)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+stateDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("alturd no-args run exited non-zero: %v", err)
	}
}
