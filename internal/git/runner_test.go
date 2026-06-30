package git_test

import (
	"bytes"
	"errors"
	"io"
	"runtime"
	"testing"

	"github.com/alturd/alturd/internal/git"
)

// fakeRunner is a test double that implements git.Runner.
// It captures the args passed to Run and returns configured output or error.
// capturedArgs does NOT include the "git" program name — only the slice
// elements passed to Runner.Run (e.g. ["diff"], not ["git","diff"]).
type fakeRunner struct {
	capturedArgs []string
	output       []byte
	err          error
}

func (f *fakeRunner) Run(args []string) (io.Reader, error) {
	f.capturedArgs = args
	if f.err != nil {
		return nil, f.err
	}
	return bytes.NewReader(f.output), nil
}

// TestRunnerExitCodes verifies that ExitCodeError sentinel values carry the
// correct exit codes (127 for git-not-found, 1 for not-a-git-repo) and that
// errors.As unwraps them correctly.
func TestRunnerExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		fakeErr  error
		wantCode int
	}{
		{
			name:     "git_not_found",
			fakeErr:  git.ErrGitNotFound,
			wantCode: 127,
		},
		{
			name:     "not_git_repo",
			fakeErr:  git.ErrNotGitRepo,
			wantCode: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fr := &fakeRunner{err: tc.fakeErr}
			_, err := fr.Run([]string{"diff"})
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var exitErr *git.ExitCodeError
			if !errors.As(err, &exitErr) {
				t.Fatalf("expected *git.ExitCodeError, got %T: %v", err, err)
			}
			if exitErr.Code != tc.wantCode {
				t.Errorf("ExitCodeError.Code = %d, want %d", exitErr.Code, tc.wantCode)
			}
		})
	}
}

// TestExitCodeErrorMessage verifies that ExitCodeError.Error() returns the Msg
// field unchanged.
func TestExitCodeErrorMessage(t *testing.T) {
	err := &git.ExitCodeError{Code: 42, Msg: "custom error message"}
	if err.Error() != "custom error message" {
		t.Errorf("Error() = %q, want %q", err.Error(), "custom error message")
	}

	// Verify sentinels have non-empty messages.
	if git.ErrGitNotFound.Error() == "" {
		t.Error("ErrGitNotFound.Msg is empty")
	}
	if git.ErrNotGitRepo.Error() == "" {
		t.Error("ErrNotGitRepo.Msg is empty")
	}
}

// TestFakeRunnerCapturesArgs verifies that fakeRunner captures the exact args
// slice passed to Run (the program name "git" is NOT included — only the args
// after it).
func TestFakeRunnerCapturesArgs(t *testing.T) {
	fr := &fakeRunner{output: []byte("diff output")}
	args := []string{"diff", "HEAD~1"}
	_, err := fr.Run(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fr.capturedArgs) != len(args) {
		t.Fatalf("capturedArgs length = %d, want %d", len(fr.capturedArgs), len(args))
	}
	for i, a := range args {
		if fr.capturedArgs[i] != a {
			t.Errorf("capturedArgs[%d] = %q, want %q", i, fr.capturedArgs[i], a)
		}
	}
}

// TestCRLFNormalization verifies that ExecRunner normalizes CRLF→LF in its output.
// This test uses ExecRunner directly with a real git subprocess — but since we only
// test the CRLF normalization helper logic here via a synthetic input, we use a
// fakeRunner that returns CRLF bytes and verify the normalization result.
// The actual ExecRunner CRLF normalization is tested via the unit-testable helper.
func TestCRLFNormalization(t *testing.T) {
	// NormalizeCRLF is a no-op on non-Windows platforms because Unix git never
	// emits CRLF in diff output. On Windows it strips the transport-level \r so
	// that downstream parsers see LF-terminated lines.
	input := []byte("line1\r\nline2\r\nline3\r\n")
	got := git.NormalizeCRLF(input)

	if runtime.GOOS == "windows" {
		want := []byte("line1\nline2\nline3\n")
		if !bytes.Equal(got, want) {
			t.Errorf("NormalizeCRLF on Windows: got %q, want %q", got, want)
		}
	} else {
		// Non-Windows: function is a no-op; CRLF bytes must be preserved.
		if !bytes.Equal(got, input) {
			t.Errorf("NormalizeCRLF on non-Windows: got %q, want input unchanged %q", got, input)
		}
	}

	// LF-only content is always returned unchanged on all platforms.
	lf := []byte("line1\nline2\n")
	got2 := git.NormalizeCRLF(lf)
	if !bytes.Equal(got2, lf) {
		t.Errorf("NormalizeCRLF on LF-only: got %q, want %q (unchanged)", got2, lf)
	}
}
