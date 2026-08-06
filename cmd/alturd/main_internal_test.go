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
	"slices"
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

// TestDiffArgsDisablesExternalDiff pins diffArgs's argv composition
// (04-06-PLAN.md Task 2, GIT-01 non-regression against the G-04-2 root-cause
// family): the standalone `alturd [ref]` path must ask git for git's own
// unified diff explicitly, so a developer's diff.external configuration
// (delta, difftastic, ...) never silently feeds a third-party renderer's
// output into diff.Parse.
func TestDiffArgsDisablesExternalDiff(t *testing.T) {
	tests := []struct {
		name     string
		refArgs  []string
		wantArgs []string
	}{
		{
			name:     "nil_refs",
			refArgs:  nil,
			wantArgs: []string{"diff", "--no-ext-diff"},
		},
		{
			name:     "single_ref",
			refArgs:  []string{"HEAD"},
			wantArgs: []string{"diff", "--no-ext-diff", "HEAD"},
		},
		{
			name:     "range_ref",
			refArgs:  []string{"main..feature"},
			wantArgs: []string{"diff", "--no-ext-diff", "main..feature"},
		},
		{
			name:     "dash_separated_pathspec",
			refArgs:  []string{"--", "internal/tui"},
			wantArgs: []string{"diff", "--no-ext-diff", "--", "internal/tui"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := diffArgs(tc.refArgs)
			if !slices.Equal(got, tc.wantArgs) {
				t.Errorf("diffArgs(%v) = %v, want %v", tc.refArgs, got, tc.wantArgs)
			}
		})
	}

	t.Run("does_not_mutate_caller_slice", func(t *testing.T) {
		input := []string{"HEAD"}
		original := slices.Clone(input)

		_ = diffArgs(input)

		if !slices.Equal(input, original) {
			t.Errorf("diffArgs mutated caller slice: got %v, want unchanged %v", input, original)
		}
	})
}
