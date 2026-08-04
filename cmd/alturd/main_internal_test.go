// Package main (internal test file) covers reportError, the exit-code
// router main() delegates to. A dedicated file (not
// difftool_internal_test.go, which is scoped to install-difftool) so this
// plan's addition stays surgically separate from the existing internal-test
// suite; package main_test's subprocess suite cannot reach reportError since
// it is unexported.
package main

import (
	"bytes"
	"errors"
	"testing"

	"github.com/alturd/alturd/internal/git"
)

// TestReportError is table-driven over the three rows in 04-05-PLAN.md Task
// 3's behavior block: a silent-abort ExitCodeError (empty Msg, code 1), a
// non-empty-Msg ExitCodeError (git not found, code 127), and a plain error
// (code 1, message printed verbatim).
func TestReportError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
		wantOut  string
	}{
		{
			name:     "silent_abort_sentinel",
			err:      &git.ExitCodeError{Code: 1},
			wantCode: 1,
			wantOut:  "",
		},
		{
			name:     "exit_code_error_with_message",
			err:      git.ErrGitNotFound,
			wantCode: 127,
			wantOut:  "git: command not found (is git installed and on PATH?)\n",
		},
		{
			name:     "plain_error",
			err:      errors.New("boom"),
			wantCode: 1,
			wantOut:  "boom\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			gotCode := reportError(tc.err, &buf)
			if gotCode != tc.wantCode {
				t.Errorf("reportError() code = %d, want %d", gotCode, tc.wantCode)
			}
			if got := buf.String(); got != tc.wantOut {
				t.Errorf("reportError() wrote %q, want %q", got, tc.wantOut)
			}
		})
	}
}
