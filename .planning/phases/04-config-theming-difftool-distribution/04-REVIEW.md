---
phase: 04-config-theming-difftool-distribution
reviewed: 2026-08-03T00:00:00Z
depth: standard
files_reviewed: 24
files_reviewed_list:
  - .github/workflows/ci.yml
  - .github/workflows/release.yml
  - cmd/alturd/difftool.go
  - cmd/alturd/difftool_internal_test.go
  - cmd/alturd/difftool_test.go
  - cmd/alturd/installdifftool_test.go
  - cmd/alturd/main.go
  - cmd/alturd/main_test.go
  - internal/config/config.go
  - internal/config/config_test.go
  - internal/config/keybindings.go
  - internal/config/keybindings_test.go
  - internal/config/theme.go
  - internal/config/theme_test.go
  - internal/diff/align.go
  - internal/diff/align_test.go
  - internal/diff/parse_test.go
  - internal/diff/render.go
  - internal/diff/render_test.go
  - internal/git/errors.go
  - internal/log/log.go
  - internal/log/log_test.go
  - internal/tui/model.go
  - internal/tui/model_test.go
findings:
  critical: 2
  warning: 5
  info: 2
  total: 9
status: issues_found
---

# Phase 04: Code Review Report

**Reviewed:** 2026-08-03T00:00:00Z
**Depth:** standard
**Files Reviewed:** 24
**Status:** issues_found

## Summary

Phase 04 layers config/keybinding parsing, theme resolution, difftool integration, and CI/release automation onto the existing diff/TUI core. The config, theme and install-difftool subsystems are carefully engineered — validation is strict, errors are single-line, and the accompanying tests are thorough and largely hermetic (isolated `GIT_CONFIG_GLOBAL`/`XDG_*` environments). `go build ./...`, `go vet ./...`, and `go test ./...` all pass cleanly on this tree.

However, `internal/tui/model.go` (a file this phase re-touches for difftool-mode layout, `04-03-PLAN.md` Task 2/3) has a reproducible crash: `View()` calls `strings.Repeat` with a count that goes negative whenever the effective content height is small, which panics the whole process on a resize to a small terminal. There is also a real terminal-corruption bug on the documented `Q` (abort) key, plus several lower-severity robustness gaps and one CI convention gap versus the project's own documented stack guidance (`go test -race`).

## Critical Issues

### CR-01: `View()` panics via negative `strings.Repeat` count on short terminals

**File:** `internal/tui/model.go:226-230`
**Issue:** In non-difftool mode, `View()` computes the pane-separator column with:
```go
contentH := m.termHeight - 1
if m.searchMode {
    contentH--
}
sep := strings.Repeat("│\n", contentH-1) + "│"
```
`m.termHeight` comes directly from `tea.WindowSizeMsg`/`resizePollMsg` in `handleResize` (`internal/tui/model.go:268-297`) with no lower-bound clamp anywhere in the resize path. Once `contentH <= 0` — i.e. a terminal height of 1 row (no search open) or 2 rows (search open), or any transient/degenerate size report — `contentH-1` is negative and `strings.Repeat` panics with `"strings: negative Repeat count"` (confirmed against the Go stdlib: `strings.Repeat(s, -1)` panics). This crashes the whole TUI process; there is no recover anywhere in the render path.

Contrast this with `internal/diff/render.go:61-63`, which explicitly clamps `width` to a minimum of 6 before doing arithmetic — the same defensive pattern is missing here for height.

**Fix:**
```go
contentH := m.termHeight - 1
if m.searchMode {
    contentH--
}
sepLines := contentH - 1
if sepLines < 0 {
    sepLines = 0
}
sep := strings.Repeat("│\n", sepLines) + "│"
```
(and/or clamp `m.termHeight`/`m.termWidth` to a sane minimum inside `handleResize` itself, since `diffW := w - m.treeWidth - 1` has the same unclamped-input problem — see WR-02).

### CR-02: `ActionAbort` calls `os.Exit(1)` directly, leaving the terminal in a broken state

**File:** `internal/tui/model.go:522-523`
**Issue:**
```go
case config.ActionAbort:
    os.Exit(1)
```
This runs inside `Update()`, on bubbletea's own event-loop goroutine, for the documented default key `Q` (`03-UI-SPEC.md` "Abort (difftool path)" row; `q`/`Q` are wired in every mode per `03-CONTEXT.md` D-18). `os.Exit` terminates the process immediately via the OS `exit()` syscall — it does not return control to `tea.Program.Run()`, so bubbletea never gets to restore the terminal (raw mode, alternate screen buffer, mouse tracking if any). It also bypasses every `defer` in `main()`, including `logFile.Close()` (`cmd/alturd/main.go:69-71`).

The design intent (`03-CONTEXT.md` D-17/D-18) is that `Q` exits immediately without a confirmation dialog and with exit code 1 — that is a legitimate product decision. But "immediate, no confirmation" does not require "skip terminal cleanup"; as implemented, every user who presses the documented abort key is left with a corrupted terminal (blank/garbled screen still in the alternate buffer, no visible input echo) until they run `reset`/`stty sane` or close the terminal tab. This is a guaranteed, 100%-reproducible regression for the one key path this feature exists to support (aborting a `git difftool` session cleanly).

**Fix:** Let bubbletea release the terminal before terminating, e.g. return a `tea.Quit`-style command carrying the desired exit code and have `main()` call `os.Exit(1)` only *after* `p.Run()` returns, or explicitly release the terminal (`p.ReleaseTerminal()` equivalent in bubbletea v2, if exposed) immediately before calling `os.Exit(1)`:
```go
case config.ActionAbort:
    // restore terminal state before hard-exiting
    fmt.Print(tea.ExitAltScreen /* or the v2 equivalent restore sequence */)
    os.Exit(1)
```
or, preferably, thread an exit-code field through a normal `tea.Quit`-returning path so `Run()` unwinds normally and `main()` performs the `os.Exit(1)` after the terminal has been restored.

## Warnings

### WR-01: `refreshTreeContent` sets the diff viewport's width, not the tree viewport's

**File:** `internal/tui/model.go:353-357`
**Issue:**
```go
func (m *model) refreshTreeContent() {
	m.diffVP.SetWidth(m.termWidth - m.treeWidth - 1)
	content := m.renderTree()
	m.treeVP.SetContent(content)
}
```
A function named `refreshTreeContent` mutates `m.diffVP`'s width instead of `m.treeVP`'s. Today this is masked because every call site (`handleResize`, `toggleAllFiles`, `treeIdxMove`, `treeToggleExpand`) either (a) is preceded in the same call by `handleResize`, which already sets `m.diffVP`'s width to the identical value, or (b) never runs in difftool mode where `treeVP` doesn't matter. The `treeVP`'s own width is never set inside this function at all — it only gets set inside `handleResize`. This is very likely a copy/paste error (compare with the symmetrical, correctly-named `refreshDiffContent`) and is a latent bug: any future call site that invokes `refreshTreeContent()` without a preceding `handleResize` (e.g. a new feature that only changes `treeWidth`) will silently fail to resize the tree pane while incorrectly re-deriving the diff pane's width as a side effect.

**Fix:**
```go
func (m *model) refreshTreeContent() {
	m.treeVP.SetWidth(m.treeWidth)
	content := m.renderTree()
	m.treeVP.SetContent(content)
}
```

### WR-02: `diffW` is not clamped and can go negative on narrow terminals

**File:** `internal/tui/model.go:288-296`
**Issue:**
```go
diffW := w - m.treeWidth - 1
m.treeVP.SetWidth(m.treeWidth)
m.treeVP.SetHeight(contentH)
m.diffVP.SetWidth(diffW)
```
`m.treeWidth` is `treeWidthFocused` (45) when the tree pane is focused. For any terminal narrower than 46 columns (a real scenario in tmux/split panes, or a resize mid-drag), `diffW` goes negative and is passed straight into `viewport.SetWidth`, unlike `diff.Render`/`diff.RenderFull`, which explicitly clamp `width` to a minimum of 6 (`internal/diff/render.go:61-63,87-89`). Behavior of `bubbles/viewport` with a negative width is unvalidated here.

**Fix:** Clamp `diffW` (and ideally `m.treeWidth` itself, and `contentH`) to a sane minimum before calling `SetWidth`/`SetHeight`, matching the pattern already used in `internal/diff/render.go`.

### WR-03: Working-tree fallback in `fetchFileLines` uses a repo-root-relative path against the process's actual working directory

**File:** `internal/tui/model.go:863-900`
**Issue:** `fetchFileLines`'s doc comment states path 1 ("`git show HEAD:path`") "works for any CWD" because the name is repo-root-relative. That's true for the two `git show` attempts. But the final fallback,
```go
// Last resort: read from the working tree.
data, err := os.ReadFile(name)
```
uses that same repo-root-relative `name` directly with `os.ReadFile`, which resolves relative to the process's actual current working directory, not the repo root. If alturd is invoked from a subdirectory of the repository (a normal thing to do) and the file being viewed is untracked/unstaged so both `git show` attempts fail, this fallback will silently fail to find the file (wrong relative path) even though the file exists, and `refreshDiffContent` will quietly degrade to hunk-only rendering instead of full-file rendering.

**Fix:** Resolve the path against the repository root before falling back to `os.ReadFile` (e.g. via `git rev-parse --show-toplevel`, or `git show :name`'s companion `git rev-parse --show-prefix`), or document/accept the degradation explicitly rather than implying CWD-independence for all three strategies.

### WR-04: `DetectDarkBackground`'s abandoned goroutine can race the TUI for the first keystroke

**File:** `internal/config/theme.go:50-78`
**Issue:** When the OSC 11 query exceeds `DetectTimeout` (50ms), `DetectDarkBackground` returns `true` (dark fallback) while its query goroutine is still running and still holds a read on the terminal file descriptor that `tea.NewProgram(m).Run()` is about to take over (`cmd/alturd/main.go:125-175`). The function's own doc comment already flags this as an unresolved, unverified assumption (`FA-04-02`): "this exits cleanly without consuming a keystroke destined for the TUI, but that has not been independently verified here." Confirming this review's job is to surface exactly this kind of un-mitigated risk: on a slow/unresponsive terminal (exactly the case this timeout exists to handle), the very first keystroke the user sends to the freshly-started bubbletea program can be silently consumed by the OSC 11 response reader instead, producing an intermittent "first key is dropped" bug that would be extremely hard to reproduce/diagnose from a bug report.

**Fix:** Either bound the query with a context/cancelable read so the goroutine is provably done (not just "likely done") before `tea.NewProgram` takes the tty, or explicitly drain/flush stdin immediately before starting `tea.NewProgram`.

### WR-05: CI does not run tests with the race detector, despite documented project convention

**File:** `.github/workflows/ci.yml:20`
**Issue:** `run: go test ./...` has no `-race` flag. `CLAUDE.md`'s own stack guidance table states: `go test -race — Race detector — Enable in CI; bubbletea v2 is goroutine-safe but custom async code needs validation`. This phase introduces exactly the kind of custom async code that guidance is warning about (`DetectDarkBackground`'s goroutine race in WR-04, `computeIntraLineWithTimeout`'s goroutine-with-timeout pattern in `internal/diff/render.go`), yet CI never exercises them under the race detector.

**Fix:**
```yaml
      - run: go test -race ./...
```

## Info

### IN-01: `gofmt` formatting violations

**File:** `internal/tui/model.go`, `cmd/alturd/main.go`, `internal/diff/render.go`
**Issue:** `gofmt -l` flags all three files (import-group ordering in `model.go`/`main.go` — `charm.land/bubbletea/v2` is not grouped/sorted with the other `charm.land`/third-party imports — and struct-field/comment alignment drift in `model.go`'s `model` struct and `render.go`'s `bg*` const block). `.golangci.yml` does not currently enable the `gofmt`/`goimports` linter, so this doesn't fail CI today, but it is a real, mechanically-verifiable deviation from standard Go formatting.
**Fix:** `gofmt -w internal/tui/model.go cmd/alturd/main.go internal/diff/render.go` (or add `gofmt`/`goimports` to `.golangci.yml`'s enabled linters so this class of drift is caught automatically going forward).

### IN-02: `install-difftool` writes a literal `"alturd"` command, relying on PATH

**File:** `cmd/alturd/difftool.go:35-45`
**Issue:** `difftoolCmdTemplate` hardcodes the bare command name `alturd` rather than the currently-running executable's resolved path (`os.Executable()`). If a user runs `install-difftool` from a binary that isn't on `PATH` (e.g. a freshly downloaded release binary invoked as `./alturd install-difftool`), `git difftool` will later fail with "alturd: command not found" even though the tool that wrote the config is right there. This may be an intentional trade-off (a PATH-relative command is more portable across machines/dotfile syncs than an absolute path baked in at install time), but it's worth an explicit note/decision since it's not obviously documented as such in the reviewed files.
**Fix (if not already an accepted trade-off):** consider `os.Executable()` (resolved+symlink-evaluated) as the default `cmd` value, or at least document the PATH requirement in the `install-difftool` success message.

---

_Reviewed: 2026-08-03T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
