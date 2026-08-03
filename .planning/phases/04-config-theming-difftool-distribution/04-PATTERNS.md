# Phase 4: Config + Theming + Difftool + Distribution - Pattern Map

**Mapped:** 2026-08-03
**Files analyzed:** 12 (Go source) + 5 (CI/build config, no code analog)
**Analogs found:** 10 / 12 Go files (2 net-new package types with no direct analog: theme-precedence resolver, gitconfig-writer — closest partial analogs still identified)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `internal/config/config.go` | config | file-I/O (read TOML, decode+validate) | `internal/log/log.go` (XDG path resolution + startup init) | role-match |
| `internal/config/keybindings.go` | config | transform (merge defaults + overrides, validate) | `internal/git/args.go` (`ParseRefArgs` — pure transform function, table-driven tested) | role-match |
| `internal/config/theme.go` | config | transform (precedence resolution) + streaming (goroutine-raced OSC 11 read) | `cmd/alturd/main.go` (existing `termenv.HasDarkBackground()` call site) | partial-match (no existing precedence-resolver file; closest is the call site being wrapped) |
| `internal/config/config_test.go` | test | — | `internal/git/args_test.go` (table-driven, `_test` external package, `slices` equality) | exact |
| `internal/config/theme_test.go` | test | — | `internal/diff/highlight_test.go` (state-toggle unit test around a package-level behavior switch) | role-match |
| `cmd/alturd/main.go` (modified) | controller (CLI entry) | request-response (flag parse → dispatch) | itself (existing file, extend in place) | exact |
| `cmd/alturd/difftool.go` (new: `install-difftool` cobra subcommand + gitconfig writer) | controller + service | event-driven (subprocess `git config` calls) | `internal/git/runner.go` (`ExecRunner.Run` — subprocess exec pattern) + `cmd/alturd/main.go` (`rootCmd`/subcommand wiring) | role-match (git.Runner interface reused, NOT verbatim — see Pitfall C below) |
| `cmd/alturd/difftool_test.go` | test | — | `cmd/alturd/main_test.go` (subprocess `TestMain` build-once pattern) | exact |
| `internal/tui/model.go` (modified: `NewModel` signature extended, `handleKey` keymap lookup, difftool title bar) | component (TUI model) | event-driven (bubbletea Update loop) | itself (existing file, extend in place) | exact |
| `internal/git/errors.go` (reference only — no new file, but `ExitCodeError` type reused) | model/error-type | — | itself | exact (reused as-is) |
| `.goreleaser.yaml` | config | batch (build artifact generation) | none (new project artifact type) | no analog — RESEARCH.md Code Examples is the source of truth |
| `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `.golangci.yml` | config | batch (CI pipeline) | none (new project artifact type) | no analog — RESEARCH.md Code Examples is the source of truth |

## Pattern Assignments

### `internal/config/config.go` (config, file-I/O)

**Analog:** `internal/log/log.go` (XDG resolution + startup-init pattern) combined with the `go-toml/v2` decode pattern already fully specified in RESEARCH.md Pattern 1/2.

**Imports pattern** (`internal/log/log.go` lines 10-19):
```go
import (
	"fmt"
	"os"
	"path/filepath"

	charmlog "github.com/charmbracelet/log"

	"github.com/adrg/xdg"
)
```
Apply the same shape to `internal/config`: stdlib first, then `github.com/pelletier/go-toml/v2` and `github.com/adrg/xdg` grouped separately, matching the existing two-group import convention seen throughout this codebase (stdlib blank-line-separated from third-party — see also `cmd/alturd/main.go` imports which group stdlib / bubbletea+termenv / internal packages in three blocks).

**XDG resolution pattern — MUST deviate from `internal/log`'s `xdg.StateFile`** (`internal/log/log.go` lines 30-33):
```go
func Init() (*os.File, error) {
	path, err := xdg.StateFile("alturd/alturd.log")
	if err != nil {
		return nil, fmt.Errorf("resolving log path: %w", err)
	}
```
`internal/log` uses `xdg.StateFile` (which *creates* directories — correct for logging, a write-path use case). **`internal/config` must use `xdg.SearchConfigFile` instead** (per RESEARCH.md Pattern 2 / Pitfall B) — this is a deliberate divergence from the closest analog, not an oversight. Do not copy `xdg.StateFile`/`xdg.ConfigFile` verbatim; use the read-only, non-creating variant.

**Startup-init entrypoint convention** (`internal/log/log.go` lines 21-27, doc comment):
```go
// Init must be called from cmd/alturd RunE only — never at package init or
// PersistentPreRunE — so --help/--version never create a log file (D-10).
```
Apply the same discipline to `config.Load`: call it from `run()` in `main.go`, not at package-level `var` or `init()`, so `--help`/`--version` never touch the filesystem (consistent with D-03's "no side effects on first run").

**Error wrapping pattern** (`internal/log/log.go` lines 31-33, 44-47):
```go
if err != nil {
	return nil, fmt.Errorf("resolving log path: %w", err)
}
...
f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
if err != nil {
	return nil, fmt.Errorf("opening log file: %w", err)
}
```
Use `fmt.Errorf("<verb phrase>: %w", err)` consistently — same convention `config.Load` should follow for TOML decode errors (see RESEARCH.md Pattern 1 for the `DisallowUnknownFields` + `StrictMissingError` extraction, which is the config-specific addition on top of this general wrapping style).

---

### `internal/config/keybindings.go` (config, transform)

**Analog:** `internal/git/args.go` — `ParseRefArgs`, a pure, fully-tested transform function with extensive doc-comment examples.

**Function-as-pure-transform pattern** (`internal/git/args.go` lines 1-24):
```go
// ParseRefArgs converts cobra's positional args (after flag parsing) to the
// git diff argument slice that ExecRunner.Run expects.
// ...
// Examples:
//
//	ParseRefArgs([], -1)                  → []
//	ParseRefArgs(["HEAD~1"], -1)          → ["HEAD~1"]
func ParseRefArgs(args []string, dashIdx int) []string {
```
Mirror this for `Merge(overrides map[string]string) error` — same doc-comment convention: explain inputs/outputs, then give concrete example invocations in a comment block. Keep the function pure (no I/O), matching `ParseRefArgs`'s testability profile — this is exactly what makes `args_test.go` a clean table-driven test to copy from (see Test pattern below).

**Pre-allocation / no-side-effect style** (`internal/git/args.go` lines 33-45): copy the "build result incrementally with clear inline comments referencing the decision ID" convention (e.g. `// Re-insert the "--" separator... (Pitfall 2)`), applying it to duplicate-key/unknown-action validation with inline `// D-02` comments.

---

### `internal/config/theme.go` (config, transform + bounded I/O)

**Analog:** `cmd/alturd/main.go` current OSC 11 call site (lines 79-83) is the thing being wrapped/replaced, not a pattern to copy verbatim — RESEARCH.md Pattern 3 is the authoritative source for the goroutine-race implementation:

```go
// cmd/alturd/main.go lines 79-83 (current — Phase 3 baseline, insufficient for THEME-01 per Pitfall A)
darkBg := termenv.NewOutput(os.Stdout).HasDarkBackground()
diff.SetDarkBackground(darkBg)
```

**Target pattern** (RESEARCH.md Pattern 3, to be placed in `internal/config/theme.go` as an exported `DetectDarkBackground(w io.Writer) bool` or similar, called from `main.go`):
```go
func detectDarkBackground(w io.Writer) bool {
	result := make(chan bool, 1)
	go func() {
		result <- termenv.NewOutput(w).HasDarkBackground()
	}()
	select {
	case dark := <-result:
		return dark
	case <-time.After(50 * time.Millisecond):
		return true // dark fallback (THEME-01, D-07)
	}
}
```

**Precedence resolution style** — follow `internal/git/errors.go`'s sentinel-typed-value convention for the `Theme` type (light/dark/auto as a small validated string type with an `Error()`-adjacent validation function), and follow `cmd/alturd/main.go`'s existing "resolve before `tea.NewProgram()`" sequencing constraint (main.go lines 79-93: detection happens strictly before `tui.NewModel`/`tea.NewProgram` calls) — the new `theme.Resolve(flag, config, difftoolMode)` call must be inserted at the same point in `run()`, immediately replacing the current `termenv.NewOutput(...).HasDarkBackground()` line.

---

### `internal/config/config_test.go` / `internal/config/theme_test.go` (test)

**Analog:** `internal/git/args_test.go`

**Table-driven test structure** (`internal/git/args_test.go` lines 1-20):
```go
package git_test

import (
	"slices"
	"testing"

	"github.com/alturd/alturd/internal/git"
)

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
		// ...
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := git.ParseRefArgs(tt.args, tt.dashIdx)
			if !slices.Equal(got, tt.wantArgs) {
				t.Errorf("ParseRefArgs(%v, %d) = %v, want %v", tt.args, tt.dashIdx, got, tt.wantArgs)
			}
		})
	}
}
```
Use `package config_test` (external test package — consistent with `git_test`), a `tests := []struct{...}{}` table with `name`/inputs/`want*` fields, and `t.Run(tt.name, ...)` subtests. This exact shape covers `TestLoad_UnknownField`, `TestKeybindings_PartialOverride`, `TestKeybindings_DuplicateRejected`, `TestTheme_Precedence` from RESEARCH.md's Phase Requirements → Test Map.

**For `TestLoad_NoSideEffectsOnFirstRun` (XDG env isolation)** — analog is `cmd/alturd/main_test.go`'s `t.TempDir()` + `XDG_STATE_HOME` env override pattern (lines 43-46):
```go
stateDir := t.TempDir()
cmd := exec.Command(alturdBin, "--version")
cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+stateDir)
```
Adapt directly: use `t.Setenv("XDG_CONFIG_HOME", t.TempDir())` (in-process, since `internal/config` tests don't need subprocess isolation like `main_test.go` does) then assert via `os.ReadDir` that no `alturd/` subdirectory was created — this is the Pitfall B regression test RESEARCH.md calls out.

**For `TestDetectDarkBackground_TimeoutFallback`** — analog is `internal/diff/highlight_test.go`'s pattern of testing a package-level style/behavior switch (`SetDarkBackground(bool)` toggling `chromaStyleName`) by asserting observable output changes; adapt to assert `elapsed < 100*time.Millisecond` (bounding check, not exact-timing) using a fake `io.Writer` that never responds.

---

### `cmd/alturd/main.go` (controller, request-response) — MODIFIED IN PLACE

**Analog:** itself — extend existing structure, do not rewrite.

**Current three-block import grouping** (lines 6-18):
```go
import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	tea "charm.land/bubbletea/v2"
	"github.com/muesli/termenv"

	"github.com/alturd/alturd/internal/diff"
	"github.com/alturd/alturd/internal/git"
	applog "github.com/alturd/alturd/internal/log"
	"github.com/alturd/alturd/internal/tui"
)
```
Add `"github.com/alturd/alturd/internal/config"` to the fourth (internal) import group; add cobra flag registration for `--config`, `--theme`, `--difftool-local`, `--difftool-remote`, `--difftool-path` in an `init()` or `rootCmd` flag block, matching RESEARCH.md's `installDifftoolCmd.Flags().String(...)` skeleton style (Code Examples section).

**Existing `run()` sequencing to preserve and extend** (lines 44-93): log init FIRST (unchanged) → git/diff pipeline (unchanged for standalone mode; branch for difftool mode per RESEARCH.md's dispatch diagram) → **insert new step here**: `cfg, err := config.Load(configFlag)` then `darkBg := theme.Resolve(...)` (replacing the bare `termenv` call) → existing empty-state guard → `tui.NewModel(files, darkBg, ...)` extended with new params (keymap, difftoolMode, counter, total) → unchanged `tea.NewProgram(m).Run()`.

**Exit-code dispatch pattern to reuse for `install-difftool` and config errors** (`main.go` lines 100-108):
```go
func main() {
	if err := rootCmd.Execute(); err != nil {
		var exitErr *git.ExitCodeError
		if errors.As(err, &exitErr) {
			fmt.Fprintln(os.Stderr, exitErr.Msg)
			os.Exit(exitErr.Code)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
```
Config validation errors and `install-difftool` errors should return plain `error` values from their `RunE`/`Load` functions and let this existing `main()` dispatcher print them — no new error-printing logic needed unless a config error needs a specific non-1 exit code (not indicated by any decision; default to falling through to `os.Exit(1)`).

---

### `cmd/alturd/difftool.go` (controller + service, event-driven) — NEW FILE

**Analog:** `internal/git/runner.go` (`ExecRunner.Run` — subprocess exec pattern) for the underlying `git config` calls; `internal/git/errors.go` (`ExitCodeError` sentinel type) for error surfacing; RESEARCH.md's `installDifftoolCmd` skeleton for cobra wiring.

**Subprocess exec pattern to adapt, NOT reuse verbatim** (`internal/git/runner.go` lines 38-44):
```go
func (ExecRunner) Run(args []string) (io.Reader, error) {
	cmd := exec.Command("git", args...) //nolint:gosec
	out, err := cmd.Output()
	if err != nil {
```
**Critical divergence (RESEARCH.md Pitfall C):** do NOT call `git.ExecRunner{}.Run(...)` unmodified for `git config --get`/`--set` calls — its error mapping hardcodes `"git diff: %s"` and treats exit 128 as always-fatal, which misclassifies `git config --get`'s normal "key unset" exit code 1. Write a small sibling helper in `difftool.go` (or extend `internal/git` with a second, config-aware error-interpretation function) that uses the same `exec.Command("git", args...)` argv-form-only primitive (same SECURITY comment convention as `runner.go` lines 27-30) but interprets exit codes per `git config`'s own conventions (0=success, 1=not found, 2=invalid file, 128=not a repo when `--local`).

**Cobra subcommand skeleton** (RESEARCH.md Code Examples, `install-difftool`):
```go
var installDifftoolCmd = &cobra.Command{
	Use:   "install-difftool",
	Short: "Register alturd as a git difftool",
	RunE:  runInstallDifftool,
}

func init() {
	installDifftoolCmd.Flags().String("scope", "global", "config scope: global|local")
	installDifftoolCmd.Flags().String("name", "alturd", "difftool name to register")
	installDifftoolCmd.Flags().Bool("force", false, "overwrite an existing diff.tool setting")
	rootCmd.AddCommand(installDifftoolCmd)
}
```
This matches `main.go`'s existing single-`rootCmd`-with-`RunE` convention (lines 26-38) — `installDifftoolCmd` is a sibling `*cobra.Command`, added via `rootCmd.AddCommand`, same `SilenceErrors`/`SilenceUsage` inheritance from the parent.

**Error type reuse** — for the `--scope local` outside-a-repo case (Pitfall D), reuse `git.ExitCodeError` exactly as `internal/git/errors.go` defines it:
```go
// internal/git/errors.go lines 12-19
type ExitCodeError struct {
	Code int
	Msg  string
}
func (e *ExitCodeError) Error() string { return e.Msg }
```
Construct a new sentinel (e.g. `ErrLocalScopeOutsideRepo`) in `internal/git/errors.go` following the exact `ErrNotGitRepo` pattern (lines 30-34), so `main()`'s existing `errors.As(err, &exitErr)` dispatch in `main.go` (line 102) picks it up with zero changes to `main()` itself.

---

### `cmd/alturd/difftool_test.go` (test) — NEW FILE

**Analog:** `cmd/alturd/main_test.go` — subprocess `TestMain` build-once pattern.

**Build-once binary pattern** (`main_test.go` lines 21-38):
```go
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "alturd-test-bin-*")
	...
	alturdBin = filepath.Join(dir, "alturd")
	buildCmd := exec.Command("go", "build", "-o", alturdBin, "./cmd/alturd")
	buildCmd.Dir = filepath.Join("..", "..")
	...
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
```
Reuse this `TestMain` as-is (same package `main_test`, same `alturdBin` var) — `difftool_test.go` lives in the same `cmd/alturd` test package as `main_test.go`, so no second `TestMain` is needed; just add new `Test*` functions to the same package.

**Env-isolated subprocess invocation pattern** (`main_test.go` lines 42-55, `TestVersionExitsZeroNoLog`):
```go
func TestVersionExitsZeroNoLog(t *testing.T) {
	stateDir := t.TempDir()
	cmd := exec.Command(alturdBin, "--version")
	cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+stateDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("--version exited non-zero: %v", err)
	}
}
```
For `TestInstallDifftool`, additionally isolate `HOME`/`GIT_CONFIG_GLOBAL` (or use `--scope local` inside a scratch repo, following `TestSmokeRunInRepoExitsZero`'s `gitSetup` helper at lines 90-105 for repo scaffolding) so the test never touches the real developer's `~/.gitconfig`.

**Exit-code assertion pattern** (`main_test.go` lines 145-153, `TestExitCodeNotGitRepo`):
```go
exitErr, ok := err.(*exec.ExitError)
if !ok {
	t.Fatalf("expected exec.ExitError, got %T: %v", err, err)
}
if code := exitErr.ExitCode(); code != 1 {
	t.Errorf("expected exit code 1, got %d", code)
}
```
Reuse verbatim for asserting `install-difftool --scope local` outside a repo exits with the correct code, and for `DIFFTOOL-01`'s difftool-mode smoke test exit codes.

---

### `internal/tui/model.go` (component, event-driven) — MODIFIED IN PLACE

**Analog:** itself — extend `NewModel` and `handleKey`, do not restructure.

**Current constructor signature to extend** (lines 78-105):
```go
func NewModel(files []*gitdiff.File, darkBg bool) model {
	...
	return model{
		files:       files,
		darkBg:      darkBg,
		focusedPane: diffFocused,
		...
	}
}
```
RESEARCH.md's architecture diagram specifies the target signature: `tui.NewModel(files, darkBg, keymap, difftoolMode, counter, total)`. Follow the existing flat-positional-args style (not an options struct) since that's what `NewModel` already does — add new fields to the `model` struct (lines 43-72) in the same grouped-by-concern style already present (files/darkBg together, tree fields together, diff fields together, search fields together) — add a new `difftool` field group (`difftoolMode bool`, `pathCounter`, `pathTotal int`) and a `keymap` field near the top with `files`/`darkBg`.

**Current hardcoded key-dispatch switch to refactor** (lines 416-439):
```go
switch msg.String() {
case "q":
	return m, tea.Quit
case "Q":
	os.Exit(1)
case "tab":
	m.toggleFocus()
	...
case "v":
	...
case "n":
	m.hunkNext()
case "N":
	m.hunkPrev()
case "]":
	m.handleFileCycle(true)
case "[":
	m.handleFileCycle(false)
...
case "/":
	...
case "a":
	m.toggleAllFiles()
```
CONFIG-02 requires this switch to become a keymap lookup: change `switch msg.String()` to first resolve `action := m.keymap.Lookup(msg.String())` (or equivalent) and `switch action` against named constants (`ActionQuit`, `ActionAbort`, `ActionToggleFocus`, etc.) rather than literal key strings. **Preserve the exact case bodies unchanged** — only the switch discriminant changes from raw key string to resolved action; this keeps the diff minimal and testable against the existing `model_test.go` suite. The search-mode sub-switches (lines 350-412, `n`/`N`/`]`/`[`/`esc`/`enter` inside search phases) are explicitly NOT part of the 10 rebindable global actions per D-04's action list — leave those literal (confirm against `03-UI-SPEC.md` Key Binding Contract during planning, per canonical_refs).

**Difftool-mode title bar** — analog is the existing `statusMarkerStyle(status string, darkBg bool) lipgloss.Style` pattern (theme-aware rendering helper) referenced in CONTEXT.md's Reusable Assets; locate this function (grep `statusMarkerStyle` in `model.go`) and follow its signature style for a new difftool-mode status-bar string builder consuming `pathCounter`/`pathTotal` — exact format from ROADMAP.md: `"alturd (difftool) — N of M — <filename>"`.

---

## Shared Patterns

### XDG Path Resolution (read-only vs write-capable)
**Source:** `internal/log/log.go` line 31 (`xdg.StateFile` — creates dirs, correct for logs) vs. RESEARCH.md Pattern 2 (`xdg.SearchConfigFile` — read-only, correct for config)
**Apply to:** `internal/config/config.go` only. Do NOT copy `internal/log`'s `xdg.StateFile` call as a template for config — the two packages need opposite XDG semantics (log always creates on write; config must never create on read-only lookup, per D-03).

### Argv-Form Subprocess Exec, Never Shell
**Source:** `internal/git/runner.go` lines 27-30 (SECURITY comment) and line 40 (`exec.Command("git", args...) //nolint:gosec`)
```go
// SECURITY: exec.Command uses argv form — each element is a separate argument.
// Shell metacharacters in ref or path strings are never interpreted.
cmd := exec.Command("git", args...) //nolint:gosec
```
**Apply to:** `cmd/alturd/difftool.go`'s `git config --get`/`--set` calls. The `//nolint:gosec` + explanatory SECURITY comment convention must be copied verbatim for any new `exec.Command("git", ...)` call site, consistent with the project's existing ASVS V5 justification already documented in `runner.go`.

### Sentinel Exit-Code Error Type
**Source:** `internal/git/errors.go` (`ExitCodeError` struct + `ErrGitNotFound`/`ErrNotGitRepo` package vars)
**Apply to:** Any new Phase 4 error condition that needs a specific process exit code (`install-difftool` scope-outside-repo error, potentially config validation errors if a distinct exit code is wanted — not specified by any decision, default to exit 1 via the existing generic branch in `main()`).

### Doc-Comment-with-Rationale-and-Decision-ID Convention
**Source:** pervasive across `internal/git/runner.go`, `internal/tui/model.go`, `cmd/alturd/main.go` — e.g. `// D-09: normalize CRLF→LF...`, `// D-13: open search...`
**Apply to:** All new Phase 4 code. Every non-obvious branch or design choice gets an inline `// D-XX: <rationale>` comment referencing the CONTEXT.md decision ID that justifies it — this is the dominant documentation convention in this codebase and downstream code review will expect it.

### Table-Driven External-Package Tests
**Source:** `internal/git/args_test.go` (full file — `package git_test`, `tests := []struct{...}`, `t.Run(tt.name, ...)`)
**Apply to:** `internal/config/*_test.go`. External test package (`config_test`, not `config`), forces the test to only exercise the public API, matching the project-wide convention (all `*_test.go` files sampled use `_test` package suffix).

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `.goreleaser.yaml` | config | batch | No existing build-config artifacts in this repo (Phase 4 is the first to introduce cross-compilation config); RESEARCH.md Code Examples section is the authoritative source, already verified against goreleaser v2 official docs this research session |
| `.github/workflows/ci.yml` | config | batch | No `.github/workflows/` directory exists yet; RESEARCH.md Code Examples is authoritative |
| `.github/workflows/release.yml` | config | batch | Same as above |
| `.golangci.yml` | config | batch | No lint config exists yet; RESEARCH.md Code Examples (v2 schema) is authoritative |
| `internal/config/theme.go`'s precedence-resolution logic specifically (as distinct from the OSC-11-detection sub-piece, which does have a call-site analog) | config | transform | No existing "resolve N-way precedence with typed enum + fallback" pattern exists in the codebase prior to this phase; closest structural cousin is `internal/git/errors.go`'s sentinel-value style, referenced above, but it's a partial match only |

## Metadata

**Analog search scope:** `cmd/alturd/`, `internal/git/`, `internal/log/`, `internal/tui/`, `internal/diff/` (all existing Go source and test files in the repository)
**Files scanned:** 12 non-test `.go` files, 10 `_test.go` files, `go.mod`
**Pattern extraction date:** 2026-08-03
</content>
