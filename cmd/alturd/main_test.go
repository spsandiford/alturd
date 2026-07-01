// Package main_test contains integration tests that build and exec the alturd
// binary as a subprocess. Because main() calls os.Exit, testing via an
// in-process main() call is not safe — subprocess testing is required.
package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// TestExitCodeNotGitRepo asserts that running alturd outside a git repository
// exits with code 1 (GIT-05). ExecRunner detects "not a git repository" stderr
// and returns ErrNotGitRepo.Code == 1.
func TestExitCodeNotGitRepo(t *testing.T) {
	// Create a temp dir that is NOT a git repository.
	notRepoDir := t.TempDir()
	stateDir := t.TempDir()

	cmd := exec.Command(alturdBin)
	cmd.Dir = notRepoDir
	cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+stateDir)

	err := cmd.Run()
	if err == nil {
		t.Fatal("alturd in non-repo dir exited 0, expected exit 1")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected exec.ExitError, got %T: %v", err, err)
	}

	if code := exitErr.ExitCode(); code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

// TestExitCodeGitNotFound asserts that running alturd when git is not on PATH
// exits with code 127 (GIT-05). ExecRunner catches exec.ErrNotFound and returns
// ErrGitNotFound.Code == 127.
func TestExitCodeGitNotFound(t *testing.T) {
	// Create a temp directory with an empty PATH so git cannot be found.
	emptyPathDir := t.TempDir()
	stateDir := t.TempDir()

	// Copy all env vars except PATH, then set PATH to empty dir.
	env := make([]string, 0, len(os.Environ()))
	for _, e := range os.Environ() {
		if len(e) > 0 && e[0:1] != "P" { // Skip PATH* vars
			if !strings.HasPrefix(e, "PATH") {
				env = append(env, e)
			}
		}
	}
	// Set PATH to a directory that contains no binaries (not even git).
	env = append(env, "PATH="+emptyPathDir)
	env = append(env, "XDG_STATE_HOME="+stateDir)

	cmd := exec.Command(alturdBin)
	cmd.Dir = filepath.Join("..", "..") // Run from repo root (valid git repo)
	cmd.Env = env

	err := cmd.Run()
	if err == nil {
		t.Fatal("alturd with no git on PATH exited 0, expected exit 127")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected exec.ExitError, got %T: %v", err, err)
	}

	if code := exitErr.ExitCode(); code != 127 {
		t.Errorf("expected exit code 127, got %d", code)
	}
}
