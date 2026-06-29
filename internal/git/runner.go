package git

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

// Runner is the subprocess boundary for git invocations.
// The single method Run accepts an args slice whose first element is the git
// subcommand (e.g. "diff") and whose remaining elements are the subcommand
// arguments. The caller is responsible for composing the args slice — Run
// passes it verbatim to exec.Command("git", args...) without interpretation.
//
// Returning io.Reader (rather than []byte) keeps the interface compatible with
// diff.Parse(io.Reader) without an intermediate buffer allocation.
type Runner interface {
	Run(args []string) (io.Reader, error)
}

// ExecRunner is the production implementation of Runner. It spawns a real
// "git" subprocess using exec.Command in argv form — never via a shell
// interpreter — so user-supplied ref and path strings are passed as distinct
// argv elements with no shell injection surface (ASVS V5, T-02-01).
//
// ExecRunner is deliberately stateless so that callers can inject it via
// dependency injection without package-level singleton state.
type ExecRunner struct{}

// Run executes "git <args...>" and returns the combined stdout as an io.Reader.
// CRLF sequences in the output are normalized to LF immediately after
// cmd.Output() returns, before bytes leave this function (D-09).
//
// Error mapping (D-11):
//   - git binary not on PATH → ErrGitNotFound (Code 127)
//   - git exits with code 128 (not a git repo) → ErrNotGitRepo (Code 1)
//   - other errors → wrapped with fmt.Errorf
func (ExecRunner) Run(args []string) (io.Reader, error) {
	// SECURITY: exec.Command uses argv form — each element is a separate argument.
	// Shell metacharacters in ref or path strings are never interpreted.
	cmd := exec.Command("git", args...) //nolint:gosec
	out, err := cmd.Output()
	if err != nil {
		// Check for "git binary not found" — exec.Error wraps exec.ErrNotFound.
		var execErr *exec.Error
		if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
			return nil, ErrGitNotFound
		}
		// Check for git exit code 128 — "not a git repository".
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 128 {
			return nil, ErrNotGitRepo
		}
		return nil, fmt.Errorf("git diff: %w", err)
	}

	// D-09: normalize CRLF→LF immediately after cmd.Output(), before the bytes
	// reach diff.Parse(). Windows git may emit CRLF line endings on some code paths.
	out = NormalizeCRLF(out)
	return bytes.NewReader(out), nil
}

// NormalizeCRLF replaces every CRLF (\r\n) sequence in b with LF (\n).
// It is exported so that runner_test.go can test the normalization logic
// directly without spawning a subprocess.
func NormalizeCRLF(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte{0x0d, 0x0a}, []byte{0x0a})
}
