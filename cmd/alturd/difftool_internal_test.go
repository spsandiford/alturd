// Package main (internal test file) covers the pure validators and the
// git-config-aware subprocess helpers that back the install-difftool
// subcommand. Subprocess tests here isolate every invocation from the
// developer's real gitconfig via GIT_CONFIG_GLOBAL/GIT_CONFIG_SYSTEM, mirroring
// the isolation convention used by cmd/alturd/installdifftool_test.go.
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alturd/alturd/internal/git"
)

func TestValidateToolName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string // substring expected in error; "" means no error
	}{
		{name: "simple name", input: "alturd", wantErr: ""},
		{name: "name with dot dash underscore digits", input: "my-tool_2.0", wantErr: ""},
		{name: "empty string", input: "", wantErr: "--name must not be empty"},
		{name: "contains space", input: "a b", wantErr: "1-64 characters"},
		{name: "contains quote and bracket", input: "a\"]\nfoo", wantErr: "1-64 characters"},
		{name: "100 character name", input: strings.Repeat("a", 100), wantErr: "1-64 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateToolName(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateToolName(%q) = %v, want nil", tt.input, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validateToolName(%q) = nil, want error containing %q", tt.input, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("validateToolName(%q) error = %q, want substring %q", tt.input, err.Error(), tt.wantErr)
			}
			if strings.Contains(err.Error(), "\n") {
				t.Errorf("validateToolName(%q) error must be single-line, got %q", tt.input, err.Error())
			}
		})
	}
}

func TestValidateScope(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantFlag  string
		wantError bool
	}{
		{name: "global", input: "global", wantFlag: "--global"},
		{name: "local", input: "local", wantFlag: "--local"},
		{name: "empty", input: "", wantError: true},
		{name: "system not allowed", input: "system", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag, err := validateScope(tt.input)
			if tt.wantError {
				if err == nil {
					t.Fatalf("validateScope(%q) = %q, nil, want error", tt.input, flag)
				}
				if err.Error() != `--scope must be "global" or "local"` {
					t.Errorf("validateScope(%q) error = %q, want exact message", tt.input, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("validateScope(%q) unexpected error: %v", tt.input, err)
			}
			if flag != tt.wantFlag {
				t.Errorf("validateScope(%q) = %q, want %q", tt.input, flag, tt.wantFlag)
			}
		})
	}
}

// TestGitConfigGetUnsetKeyIsNotAnError asserts that gitConfigGet on an unset
// key returns ("", false, nil) — git config --get exits 1 for "key not
// found", which is a normal outcome here, never a wrapped error (04-RESEARCH.md
// Pitfall C).
func TestGitConfigGetUnsetKeyIsNotAnError(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH — skipping")
	}

	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)

	value, present, err := gitConfigGet("--global", "diff.tool")
	if err != nil {
		t.Fatalf("gitConfigGet unset key returned error: %v", err)
	}
	if present {
		t.Errorf("gitConfigGet unset key: present = true, want false")
	}
	if value != "" {
		t.Errorf("gitConfigGet unset key: value = %q, want empty", value)
	}
}

// TestGitConfigGetLocalScopeOutsideRepo asserts that gitConfigGet("--local", ...)
// run outside a git repository returns git.ErrLocalScopeOutsideRepo.
func TestGitConfigGetLocalScopeOutsideRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH — skipping")
	}

	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Chdir(t.TempDir())

	_, _, err := gitConfigGet("--local", "diff.tool")
	if err != git.ErrLocalScopeOutsideRepo {
		t.Fatalf("gitConfigGet(--local, ...) outside a repo err = %v, want git.ErrLocalScopeOutsideRepo", err)
	}
}
