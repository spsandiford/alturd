package git_test

import (
	"slices"
	"testing"

	"github.com/alturd/alturd/internal/git"
)

// TestParseRefArgs validates all six CLI invocation forms for alturd's
// ref-grammar parser. See RESEARCH.md §Pattern 4 and §Pitfall 2 for the full
// design rationale.
func TestParseRefArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		dashIdx  int
		wantArgs []string
	}{
		{
			name:     "no_args",
			args:     []string{},
			dashIdx:  -1,
			wantArgs: []string{},
		},
		{
			name:     "single_ref",
			args:     []string{"HEAD~1"},
			dashIdx:  -1,
			wantArgs: []string{"HEAD~1"},
		},
		{
			name:     "two_dot_range",
			args:     []string{"HEAD~1..HEAD"},
			dashIdx:  -1,
			wantArgs: []string{"HEAD~1..HEAD"},
		},
		{
			name:     "three_dot_range",
			args:     []string{"main...feature"},
			dashIdx:  -1,
			wantArgs: []string{"main...feature"},
		},
		{
			name:     "two_args",
			args:     []string{"HEAD~1", "HEAD"},
			dashIdx:  -1,
			wantArgs: []string{"HEAD~1", "HEAD"},
		},
		{
			name:     "paths_only",
			args:     []string{"internal/"},
			dashIdx:  0,
			wantArgs: []string{"--", "internal/"},
		},
		{
			name:     "ref_plus_paths",
			args:     []string{"HEAD~1", "internal/"},
			dashIdx:  1,
			wantArgs: []string{"HEAD~1", "--", "internal/"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := git.ParseRefArgs(tc.args, tc.dashIdx)

			// Treat nil and empty slice as equivalent for the no_args case.
			if len(got) == 0 && len(tc.wantArgs) == 0 {
				return
			}

			if !slices.Equal(got, tc.wantArgs) {
				t.Errorf("ParseRefArgs(%v, %d) = %v, want %v",
					tc.args, tc.dashIdx, got, tc.wantArgs)
			}
		})
	}
}
