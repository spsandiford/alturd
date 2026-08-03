// Package git provides the git subprocess adapter for alturd.
// It exposes a Runner interface for dependency injection in tests,
// and maps CLI argument forms to git diff argument slices.
package git

// ExitCodeError carries a process exit code and a single-line user-facing
// message. It is used as the sentinel error type so that main() can extract
// the exit code via errors.As and call os.Exit with the right value.
//
// Code 127: git binary not found on PATH.
// Code 1:   working directory is not inside a git repository.
type ExitCodeError struct {
	Code int
	Msg  string
}

// Error implements the error interface. It returns the Msg field unchanged so
// that main() can print it directly to stderr without any wrapping prefix.
func (e *ExitCodeError) Error() string { return e.Msg }

// ErrGitNotFound is returned by ExecRunner.Run when exec.ErrNotFound indicates
// that the "git" binary is not on PATH. Exit code 127 matches the shell
// convention for "command not found" (D-11).
var ErrGitNotFound = &ExitCodeError{
	Code: 127,
	Msg:  "git: command not found (is git installed and on PATH?)",
}

// ErrNotGitRepo is returned by ExecRunner.Run when git exits with code 128,
// which git uses to signal that the working directory is not inside a git
// repository. Mapped to exit code 1 per D-11.
var ErrNotGitRepo = &ExitCodeError{
	Code: 1,
	Msg:  "not a git repository (run alturd inside a git working tree)",
}

// ErrLocalScopeOutsideRepo is returned by cmd/alturd's gitConfigRun/gitConfigGet
// when `install-difftool --scope local` is run outside a git repository.
// `git config --local` writes to .git/config, which does not exist outside a
// working tree; the message matches 04-UI-SPEC.md's install-difftool CLI
// Output table verbatim (04-RESEARCH.md Pitfall D).
var ErrLocalScopeOutsideRepo = &ExitCodeError{
	Code: 1,
	Msg:  "--scope local requires running inside a git repository.",
}
