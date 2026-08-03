package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"

	"github.com/alturd/alturd/internal/config"
)

// setXDGConfigHome sets XDG_CONFIG_HOME to dir and reloads xdg package
// globals, restoring both in LIFO order on cleanup. Mirrors
// internal/log/log_test.go's setXDGStateHome pattern (Phase 2 decision log).
func setXDGConfigHome(t *testing.T, dir string) {
	t.Helper()
	t.Cleanup(func() { xdg.Reload() })
	t.Setenv("XDG_CONFIG_HOME", dir)
	xdg.Reload()
}

// TestLoad_ExplicitPath verifies that Load(path) reads exactly the named
// file rather than searching the XDG default location (CONFIG-01).
func TestLoad_ExplicitPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.toml")
	if err := os.WriteFile(path, []byte("theme = \"dark\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load(%q) = _, %v, want nil error", path, err)
	}
	if cfg.Theme != "dark" {
		t.Errorf("Load(%q).Theme = %q, want %q", path, cfg.Theme, "dark")
	}
}

// TestLoad_PartialKeybindingOverride verifies D-01: a config that rebinds
// only the quit action keeps the default keys for the other nine actions,
// and the rebound key resolves through Lookup to the right action.
func TestLoad_PartialKeybindingOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	contents := "[keybindings]\nquit = \"x\"\n"
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load(%q) = _, %v, want nil error", path, err)
	}

	if got := cfg.Keys.Lookup("x"); got != config.ActionQuit {
		t.Errorf("Lookup(%q) = %v, want %v", "x", got, config.ActionQuit)
	}
	if got := cfg.Keys.Lookup("q"); got != config.ActionNone {
		t.Errorf("Lookup(%q) = %v, want %v (old quit key no longer bound)", "q", got, config.ActionNone)
	}
	// D-01: an unspecified action keeps its default key.
	if got := cfg.Keys.Lookup("n"); got != config.ActionNextHunk {
		t.Errorf("Lookup(%q) = %v, want %v (untouched default)", "n", got, config.ActionNextHunk)
	}
}

// TestLoad_MissingExplicitPath verifies that a missing --config target is a
// startup error, never a silent fallback to defaults (CONFIG-01), and that
// the underlying os.Open error is wrapped in the "config: {underlying
// error}" template rather than reaching the user raw (04-UI-SPEC.md
// Copywriting Contract, "Config file unreadable" row).
func TestLoad_MissingExplicitPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.toml")

	_, err := config.Load(path)
	if err == nil {
		t.Fatalf("Load(%q) = nil error, want an error for a missing explicit path", path)
	}
	if got := err.Error(); !strings.HasPrefix(got, "config: ") {
		t.Errorf("Load(%q) error = %q, want prefix %q", path, got, "config: ")
	}
}

// TestLoad_ErrorsAreSingleLine verifies that every error-producing Load path
// this plan touches returns a single-line message with no embedded newline,
// so no multi-line stack trace or raw library error dump ever reaches the
// user (04-UI-SPEC.md Copywriting Contract).
func TestLoad_ErrorsAreSingleLine(t *testing.T) {
	dir := t.TempDir()

	missingPath := filepath.Join(dir, "does-not-exist.toml")

	unknownFieldPath := filepath.Join(dir, "unknown-field.toml")
	if err := os.WriteFile(unknownFieldPath, []byte("themez = \"dark\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	unknownActionPath := filepath.Join(dir, "unknown-action.toml")
	if err := os.WriteFile(unknownActionPath, []byte("[keybindings]\nnex_hunk = \"n\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	malformedPath := filepath.Join(dir, "malformed.toml")
	if err := os.WriteFile(malformedPath, []byte("this is not = = valid toml\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	for _, path := range []string{missingPath, unknownFieldPath, unknownActionPath, malformedPath} {
		_, err := config.Load(path)
		if err == nil {
			t.Fatalf("Load(%q) = nil error, want an error", path)
		}
		if strings.Contains(err.Error(), "\n") {
			t.Errorf("Load(%q) error = %q, contains a newline (want single-line)", path, err.Error())
		}
	}
}

// TestLoad_NoConfigFileUsesDefaults verifies that Load("") with nothing at
// the XDG default location returns DefaultConfig unchanged.
func TestLoad_NoConfigFileUsesDefaults(t *testing.T) {
	setXDGConfigHome(t, t.TempDir())

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load(\"\") = _, %v, want nil error", err)
	}
	want := config.DefaultConfig()
	if cfg.Theme != want.Theme {
		t.Errorf("Load(\"\").Theme = %q, want %q", cfg.Theme, want.Theme)
	}
	if cfg.Keys.Lookup("q") != config.ActionQuit {
		t.Errorf("Load(\"\").Keys.Lookup(%q) = %v, want %v", "q", cfg.Keys.Lookup("q"), config.ActionQuit)
	}
}

// TestLoad_UnknownField verifies that a top-level TOML key not recognized by
// rawConfig aborts loading with the exact single-line 04-UI-SPEC.md template,
// substituting the real field name and path (D-02).
func TestLoad_UnknownField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("themez = \"dark\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := config.Load(path)
	wantErr := "config: unknown field \"themez\" in " + path
	if err == nil {
		t.Fatalf("Load(%q) = nil error, want %q", path, wantErr)
	}
	if got := err.Error(); got != wantErr {
		t.Errorf("Load(%q) error = %q, want %q", path, got, wantErr)
	}
}

// TestLoad_NoSideEffectsOnFirstRun verifies D-03: a first run with a clean
// XDG_CONFIG_HOME and no --config flag leaves that directory byte-for-byte
// unchanged — no directory and no file is created by the read-only discovery
// path. This is the only reliable regression guard against accidentally
// swapping xdg.SearchConfigFile for one of the directory-creating helpers in
// the same package.
func TestLoad_NoSideEffectsOnFirstRun(t *testing.T) {
	setXDGConfigHome(t, t.TempDir())
	dir := xdg.ConfigHome

	if _, err := config.Load(""); err != nil {
		t.Fatalf("Load(\"\") = _, %v, want nil error", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", dir, err)
	}
	if len(entries) != 0 {
		t.Errorf("ReadDir(%q) = %d entries, want 0 (Load must not create files/dirs, D-03)", dir, len(entries))
	}
}
