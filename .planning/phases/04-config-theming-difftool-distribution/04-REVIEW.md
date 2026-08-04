---
phase: 04-config-theming-difftool-distribution
reviewed: 2026-08-04T00:00:00Z
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
  - cmd/alturd/main_internal_test.go
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
  critical: 0
  warning: 6
  info: 4
  total: 10
status: issues_found
---

# Phase 04: Code Review Report

**Reviewed:** 2026-08-04T00:00:00Z
**Depth:** standard
**Files Reviewed:** 24
**Status:** issues_found

## Summary

This is a fresh, independent re-review of the same 24-file scope as the prior `04-REVIEW.md`, performed after the 04-05 gap-closure plan (commits `4549e4b`..`2e9b843`). `go build ./...`, `go vet ./...`, and `go test ./...` all pass cleanly.

**Both previously-reported Critical findings are now fixed, and independently verified as fixed here, not merely assumed:**

- **CR-01** (`View()` panic via negative `strings.Repeat` count): `internal/tui/model.go`'s `View()` and `handleResize()` both now clamp their `contentH`/`sepLines` computations at zero (commit `bbfbb57`). I traced both call sites and confirmed the clamp is applied before the `strings.Repeat` call and before `SetHeight`. `TestViewNoPanicOnShortTerminal` (`internal/tui/model_test.go:492-524`) exercises heights 0-3 with search open and closed and asserts no panic, plus a non-regression check that the separator glyph count at 80×24 is unchanged. I re-ran this test in isolation and it passes.
- **CR-02** (`os.Exit(1)` bypassing bubbletea's terminal restore): `ActionAbort` now sets `m.aborted = true` and returns `(m, tea.Quit)` (`internal/tui/model.go:574-583`) instead of calling `os.Exit` directly. `cmd/alturd/main.go`'s `run()` now captures `p.Run()`'s final model, checks `tui.WasAborted(finalModel)`, and returns a new `errAborted` sentinel (`Code: 1`, empty `Msg`) only *after* `p.Run()` has returned (i.e., after bubbletea has already restored the terminal) — this also means the deferred `logFile.Close()` now runs on the abort path, which it did not before. The new `reportError()` helper (`cmd/alturd/main.go:338-348`) suppresses output for an empty-`Msg` `ExitCodeError`, preserving the pre-fix silent-abort/exit-1 observable behavior. `TestAbortKeyQuitsWithoutProcessExit` (`internal/tui/model_test.go:434-483`) and `TestReportError` (`cmd/alturd/main_internal_test.go`) both cover this and pass.
- The previously-verified **DIFFTOOL-02 ellipsis gap** (`04-VERIFICATION.md`) is also fixed: `difftoolTitleBar()` now calls `ansi.Truncate(title, m.termWidth, "…")` directly instead of `lipgloss.Style.MaxWidth` (which truncates with a hardcoded empty tail and cannot append an ellipsis — confirmed by reading `charm.land/lipgloss/v2@v2.0.5/style.go`). I traced `ansi.Truncate`'s implementation (`charmbracelet/x/ansi@v0.11.7/truncate.go`) through the degenerate widths 0 and 1 used in `TestDifftoolTitleBarTruncatesWithEllipsis` (`internal/tui/model_test.go:532-557`) and confirmed neither panics nor produces a mismatched display width.

**What is still open:** every Warning and Info item from the prior review (`WR-01` through `WR-05`, `IN-01`, `IN-02`) remains present in the current tree exactly as before — the 04-05 plan's scope was explicitly limited to CR-01/CR-02/the ellipsis gap (the `refreshTreeContent`/`diffW`-width fix, the CI `-race` gap, etc. were left as tracked debt per the plan and per in-code comments, e.g. `model.go:325 "Height only; diffW (WR-02) stays untouched"`). I independently re-verified each rather than assuming they're still accurate; findings below reflect that fresh verification, plus a small number of newly-observed items in this pass (duplicated search-close logic, a silently-swallowed file-read error, and a UX inconsistency introduced by the ellipsis fix itself).

## Warnings

### WR-01: `refreshTreeContent` mutates the diff viewport's width, not the tree viewport's

**File:** `internal/tui/model.go:405-409`
**Issue:**
```go
func (m *model) refreshTreeContent() {
	m.diffVP.SetWidth(m.termWidth - m.treeWidth - 1)
	content := m.renderTree()
	m.treeVP.SetContent(content)
}
```
A function named `refreshTreeContent` sets `m.diffVP`'s width, never `m.treeVP`'s. `m.treeVP.SetWidth` is set only inside `handleResize` (`model.go:342`). Today this is masked because every call site (`handleResize`, `toggleAllFiles`, `treeIdxMove`, `treeToggleExpand`) either runs immediately after `handleResize` already set the identical `diffVP` width, or never runs in difftool mode where `treeVP` doesn't matter. It is a latent bug: any future call site that invokes `refreshTreeContent()` without a preceding `handleResize` will silently fail to resize the tree pane while incorrectly re-deriving the diff pane's width as an unrelated side effect.
**Fix:**
```go
func (m *model) refreshTreeContent() {
	m.treeVP.SetWidth(m.treeWidth)
	content := m.renderTree()
	m.treeVP.SetContent(content)
}
```

### WR-02: `diffW` is not clamped and can go negative on narrow terminals

**File:** `internal/tui/model.go:340-345`
**Issue:**
```go
diffW := w - m.treeWidth - 1
m.treeVP.SetWidth(m.treeWidth)
m.treeVP.SetHeight(contentH)
m.diffVP.SetWidth(diffW)
m.diffVP.SetHeight(contentH)
```
`m.treeWidth` is `treeWidthFocused` (45) when the tree pane is focused. On any terminal narrower than 46 columns (real in tmux/split panes) `diffW` goes negative. I traced this through `charm.land/bubbles/v2@v2.1.0/viewport.go` and `charm.land/lipgloss/v2@v2.0.5`: it does **not** panic (both `maxWidth()`'s `max(0, ...)` guard and `alignTextHorizontal`'s `shortAmount > 0` guard prevent a crash), but the practical effect is that `diffVP.maxWidth()` clamps to 0 and `visibleLines()` returns `nil` — the entire diff pane silently renders as blank while the tree pane is focused on a narrow terminal, with no error or indication to the user. `diff.Render`/`diff.RenderFull` already establish the correct pattern (`internal/diff/render.go:61-63`: clamp to a minimum of 6) that this call site does not follow.
**Fix:** Clamp `diffW` (and `m.treeWidth` itself) to a sane minimum before calling `SetWidth`, matching `internal/diff/render.go`'s existing pattern.

### WR-03: Working-tree fallback in `fetchFileLines` uses a repo-root-relative path against the process's actual working directory

**File:** `internal/tui/model.go:954-959`
**Issue:**
```go
// Last resort: read from the working tree.
data, err := os.ReadFile(name)
```
`name` is repo-root-relative (used successfully by the two preceding `git show HEAD:name` / `git show :name` attempts). `os.ReadFile` resolves it relative to the process's actual CWD, not the repo root. If alturd is invoked from a subdirectory and the file is untracked/unstaged (so both `git show` attempts fail), this fallback silently fails to find a file that does exist, and `refreshDiffContent` quietly degrades from full-file to hunk-only rendering with no user-visible error.
**Fix:** Resolve `name` against the repository root (e.g. via `git rev-parse --show-toplevel`) before the `os.ReadFile` fallback, or explicitly document the CWD-dependence of this specific fallback path.

### WR-04: `DetectDarkBackground`'s abandoned goroutine can race the TUI for the first keystroke

**File:** `internal/config/theme.go:66-78`
**Issue:** When the OSC 11 query exceeds `DetectTimeout` (50ms), `DetectDarkBackground` returns `true` (dark fallback) while its query goroutine is still running and still holds a read on the terminal file descriptor that `tea.NewProgram(m).Run()` is about to take over immediately afterward (`cmd/alturd/main.go:174-176`). The function's own doc comment already flags this as an unresolved, unverified assumption (`FA-04-02`) — this is unchanged from the prior review; on a slow/unresponsive terminal (exactly the case this timeout exists to handle), the first keystroke sent to the freshly-started bubbletea program can be silently consumed by the abandoned OSC 11 response reader instead, producing an intermittent "first key is dropped" bug.
**Fix:** Bound the query with a context/cancelable read so the goroutine is provably done before `tea.NewProgram` takes the tty, or explicitly drain/flush stdin immediately before starting `tea.NewProgram`.

### WR-05: CI does not run tests with the race detector, despite documented project convention

**File:** `.github/workflows/ci.yml:20`
**Issue:** `run: go test ./...` has no `-race` flag. `CLAUDE.md`'s own stack guidance states: "go test -race — Race detector — Enable in CI; bubbletea v2 is goroutine-safe but custom async code needs validation." This phase's own code contains exactly that kind of custom async code — `DetectDarkBackground`'s goroutine (WR-04 above) and `computeIntraLineWithTimeout`'s goroutine-with-timeout pattern in `internal/diff/render.go:311-323` — yet CI never exercises either under the race detector.
**Fix:**
```yaml
      - run: go test -race ./...
```

### WR-06: Duplicated "close search and reset state" logic repeated four times in `handleKey`

**File:** `internal/tui/model.go:494-501, 515-521, 528-534, 536-542`
**Issue:** The five-statement sequence that closes search mode and resets its state —
```go
m.searchMode = false
m.searchInput.Reset()
m.searchMatches = nil
m.searchMatchIdx = 0
m.handleResize(m.termWidth, m.termHeight)
```
(plus `m.searchTyping = false` in the typing-phase variant) — appears four separate times: typing-phase `esc`, navigation-phase `esc`, navigation-phase `]`, and navigation-phase `[`. This is the same class of risk already demonstrated by WR-01: a future edit to search-close behavior (e.g. adding a new field to reset) is likely to update some but not all four copies, silently reintroducing inconsistent state between the `esc` path and the `]`/`[` paths.
**Fix:** Extract a `closeSearch()` helper method and call it from all four sites.

## Info

### IN-01: `gofmt` formatting violations persist in reviewed files

**File:** `cmd/alturd/main.go`, `internal/diff/render.go`, `internal/tui/model.go`
**Issue:** `gofmt -l` still flags all three files — import-group ordering (`charm.land/bubbletea/v2` not grouped with other third-party imports in `main.go`/`model.go`) and struct-field/comment alignment drift in `model.go`'s `model` struct and `render.go`'s `bg*` const block. This is unchanged from the prior review; `.golangci.yml` still does not enable `gofmt`/`goimports`, so it doesn't fail CI.
**Fix:** `gofmt -w cmd/alturd/main.go internal/diff/render.go internal/tui/model.go`, and/or enable the `gofmt`/`goimports` linter in CI.

### IN-02: `install-difftool` writes a literal `"alturd"` command, relying on PATH

**File:** `cmd/alturd/difftool.go:35-45`
**Issue:** `difftoolCmdTemplate` hardcodes the bare command name `alturd` rather than the currently-running executable's resolved path (`os.Executable()`). If a user runs `install-difftool` from a binary not on `PATH` (e.g. `./alturd install-difftool` from a freshly downloaded release archive), `git difftool` will later fail with "alturd: command not found" even though the tool that wrote the config is right there. Unchanged from the prior review; may be an accepted trade-off (PATH-relative commands survive machine/dotfile-sync moves better than a baked-in absolute path) but isn't documented as such anywhere in the reviewed files.
**Fix (if not an accepted trade-off):** Use `os.Executable()` (resolved/symlink-evaluated) as the default `cmd` value, or document the PATH requirement in the install-difftool success message.

### IN-03: `os.ReadFile(remote)` failure in `loadDifftoolFiles` is silently swallowed

**File:** `cmd/alturd/main.go:236-239`
**Issue:**
```go
var newFileLines []string
if data, readErr := os.ReadFile(remote); readErr == nil {
	newFileLines = splitDifftoolFileLines(data)
}
```
If reading git's own `--difftool-remote` temp file fails (permissions, race with git cleaning up the temp file, etc.), `readErr` is discarded entirely — no log entry, no stderr message. `internal/tui/model.go`'s `refreshDiffContent` then unconditionally takes the `RenderFull` branch for difftool mode regardless of whether `NewFileLines` is populated (it never checks an error there, since difftool mode has none to check), so the failure surfaces only as unexplained blank/empty context lines in the diff pane, with nothing for the user to diagnose. Low likelihood (the file is git's own freshly-materialized temp file) but worth at least a debug-log entry via `internal/log`.
**Fix:** Log `readErr` via `applog`'s log file (already wired for exactly this kind of non-fatal diagnostic) instead of discarding it silently.

### IN-04: Tree-pane truncation still lacks the ellipsis the difftool title bar now has

**File:** `internal/tui/model.go:472`
**Issue:** `renderTree()` still truncates long file/directory names via `lipgloss.NewStyle().MaxWidth(m.treeWidth).Render(line)`, which — as the DIFFTOOL-02 fix's own commit message and doc comment (`model.go:292-299`) point out — truncates with a hardcoded empty tail and cannot append an ellipsis. The difftool title bar was deliberately switched away from this exact pattern to `ansi.Truncate(..., "…")` to satisfy the Copywriting Contract, but the tree pane (same visual truncation problem, same underlying cause) was explicitly left as-is ("Tree pane (renderTree, same MaxWidth pattern) is untouched per the plan's explicit scope decision — tracked as debt", commit `b7740ae`). The result is a now-visible inconsistency within the same UI: overflowing filenames are marked with "…" in the difftool title bar but silently clipped with no indicator in the tree pane.
**Fix:** Apply the same `ansi.Truncate(line, m.treeWidth, "…")` pattern to `renderTree`'s per-row truncation for consistency (tracked debt per the 04-05 plan; flagging here so it isn't lost).

---

_Reviewed: 2026-08-04T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
