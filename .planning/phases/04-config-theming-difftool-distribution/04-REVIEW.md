---
phase: 04-config-theming-difftool-distribution
reviewed: 2026-08-06T00:00:00Z
depth: standard
files_reviewed: 24
files_reviewed_list:
  - .github/workflows/ci.yml
  - .github/workflows/release.yml
  - cmd/alturd/difftool.go
  - cmd/alturd/difftool_internal_test.go
  - cmd/alturd/difftool_test.go
  - cmd/alturd/difftooldiff_internal_test.go
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
  warning: 7
  info: 6
  total: 13
status: issues_found
---

# Phase 04: Code Review Report

**Reviewed:** 2026-08-06T00:00:00Z
**Depth:** standard
**Files Reviewed:** 24
**Status:** issues_found

## Summary

This is an independent adversarial pass over the full 24-file Phase 04 scope (config, theming, difftool integration, and CI/release distribution). `go build ./...`, `go vet ./...`, and `go test ./...` all pass. `gofmt -l` was also run directly as part of this review (see IN-01).

The two previously-tracked Critical findings (a `strings.Repeat` panic on short terminals, and an `os.Exit(1)` call that bypassed bubbletea's terminal restore on the abort key) are confirmed fixed in the current tree: `View()`/`handleResize()` both clamp their height-derived counters at zero before `strings.Repeat`/`SetHeight`, and `ActionAbort` now routes through `tea.Quit` with the final model inspected via `tui.WasAborted` only after `tea.Program.Run()` returns. The difftool title bar's ellipsis truncation (`ansi.Truncate(title, m.termWidth, "…")`) is also present and correctly guards degenerate widths (0, 1). No Critical/Blocker-level defect was found in this pass — every remaining issue is a latent correctness gap, a robustness/logging gap, or a maintainability concern.

The most notable new finding from this pass (WR-07) is a genuine correctness bug in difftool full-file rendering for **deleted** files: `loadDifftoolFiles` unconditionally sources full-file content from `--difftool-remote`, never `--difftool-local`, even though `AlignFull`'s own documented contract requires the *old* file for deletions. The standalone (non-difftool) code path (`fetchFileLines`) already special-cases `f.IsDelete` correctly — this asymmetry between the two code paths is the tell. Its practical impact is currently masked because a full-file deletion diff is always a single hunk spanning the whole file (so no inter-hunk context gap ever needs filling), but it is a real contract violation that will silently produce blank content the moment that invariant doesn't hold.

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
A function named `refreshTreeContent` sets `m.diffVP`'s width and never touches `m.treeVP`'s width at all — `m.treeVP.SetWidth` is set only inside `handleResize` (`model.go:342`). All four call sites (`handleResize` itself, `toggleAllFiles`, `treeIdxMove`, `treeToggleExpand`) currently happen to run immediately after (or as part of) a `handleResize` call that already set the correct `diffVP` width via the identical formula, so the redundant assignment here is a no-op in practice today. But this is exactly the kind of latent bug that survives refactors silently: any future call site that invokes `refreshTreeContent()` without a preceding `handleResize` (e.g. a hypothetical "rename current node" or lazy-tree-reload feature) will fail to resize the tree pane while pointlessly recomputing an unrelated pane's width.
**Fix:**
```go
func (m *model) refreshTreeContent() {
	m.treeVP.SetWidth(m.treeWidth)
	content := m.renderTree()
	m.treeVP.SetContent(content)
}
```

### WR-02: `diffW` is never clamped and goes negative on narrow terminals

**File:** `internal/tui/model.go:340-345`
**Issue:**
```go
diffW := w - m.treeWidth - 1
m.treeVP.SetWidth(m.treeWidth)
m.treeVP.SetHeight(contentH)
m.diffVP.SetWidth(diffW)
m.diffVP.SetHeight(contentH)
```
`m.treeWidth` is `treeWidthFocused` (45) whenever the tree pane has focus. On any terminal narrower than 46 columns — a realistic width inside a tmux split or a resized terminal emulator — `diffW` goes negative. I traced this into `charm.land/bubbles/v2@v2.1.0/viewport.go`: `maxWidth()`'s `max(0, m.Width()-...)` guard prevents a panic, but `visibleLines()` early-returns `nil` once `maxWidth()` clamps to 0, so the entire diff pane silently renders blank with no indication to the user while the tree pane is focused on a narrow terminal. `internal/diff/render.go:61-63` (`Render`/`RenderFull`) already establishes the correct pattern for this exact class of input — clamp to a sane minimum (6) before use — that this call site does not follow. The in-code comment at `model.go:325` ("Height only; diffW (WR-02) stays untouched") confirms this is a known, currently-unaddressed gap rather than an oversight introduced by this review.
**Fix:** Clamp `diffW` (and/or `m.treeWidth`) to a sane minimum before calling `SetWidth`, matching `internal/diff/render.go`'s existing width-floor pattern.

### WR-03: `fetchFileLines`'s working-tree fallback resolves a repo-relative path against the process's actual CWD

**File:** `internal/tui/model.go:954-959`
**Issue:**
```go
// Last resort: read from the working tree.
data, err := os.ReadFile(name)
```
`name` is repo-root-relative — that's exactly why the two preceding attempts (`git show HEAD:name`, `git show :name`) work regardless of the user's current directory. `os.ReadFile`, however, resolves `name` relative to the process's actual working directory, not the repository root. If alturd is invoked from a subdirectory and the target file is untracked/unstaged (so both `git show` calls legitimately fail), this fallback silently fails to find a file that does exist on disk, and `refreshDiffContent` quietly degrades from full-file to hunk-only rendering with no error surfaced to the user.
**Fix:** Resolve `name` against the repository root (e.g. via a cached `git rev-parse --show-toplevel`) before this fallback `os.ReadFile` call, or explicitly document that this specific fallback is CWD-dependent.

### WR-04: `DetectDarkBackground`'s abandoned goroutine can race the TUI for the first keystroke

**File:** `internal/config/theme.go:66-78`
**Issue:** When the OSC 11 terminal round-trip exceeds `DetectTimeout` (50ms), `DetectDarkBackground` returns `true` (dark fallback) while its query goroutine is still running and still holds a read on the same terminal file descriptor that `tea.NewProgram(m).Run()` (`cmd/alturd/main.go:175-176`) takes over immediately afterward. The function's own doc comment already flags this as an open, unverified assumption (`FA-04-02: "that has not been independently verified here"`), which this review independently confirms is still accurate — nothing added since guards against it. On a slow or unresponsive terminal (precisely the scenario this timeout exists to handle), the abandoned reader can consume the very first keystroke the user sends to the freshly-started bubbletea program, producing an intermittent "first key is dropped" bug that would be very hard to reproduce and diagnose from a bug report.
**Fix:** Bind the OSC 11 query to a context/cancelable read (or an explicit stdin drain immediately before `tea.NewProgram`) so the goroutine is provably finished, not merely abandoned, before bubbletea takes over the TTY.

### WR-05: CI does not run the test suite with the race detector

**File:** `.github/workflows/ci.yml:20`
**Issue:** `run: go test ./...` has no `-race` flag. This project's own stack guidance (`CLAUDE.md`) states: *"go test -race — Race detector — Enable in CI; bubbletea v2 is goroutine-safe but custom async code needs validation."* This phase introduces exactly that class of custom async code — the goroutine-with-buffered-channel pattern in `DetectDarkBackground` (`internal/config/theme.go:67-70`, see WR-04) and the goroutine-with-timeout pattern in `computeIntraLineWithTimeout` (`internal/diff/render.go:311-323`) — yet CI never exercises either under `-race`.
**Fix:**
```yaml
      - run: go test -race ./...
```

### WR-06: The "close search and reset state" sequence is duplicated four times in `handleKey`

**File:** `internal/tui/model.go:494-501, 515-521, 528-534, 536-542`
**Issue:** The block that closes search mode and resets its state —
```go
m.searchMode = false
m.searchInput.Reset()
m.searchMatches = nil
m.searchMatchIdx = 0
m.handleResize(m.termWidth, m.termHeight)
```
(plus `m.searchTyping = false` in the typing-phase `esc` variant) is copy-pasted at four separate call sites: typing-phase `esc`, navigation-phase `esc`, navigation-phase `]`, and navigation-phase `[`. This is the same class of maintenance risk as WR-01: a future change to search-close behavior (e.g. resetting a new field, or fixing an ordering bug) is very likely to be applied to some but not all four copies, silently reintroducing inconsistent state between the `esc` path and the `[`/`]` paths.
**Fix:** Extract a `closeSearch()` helper method (with a `keepTyping bool` or similar parameter for the one variant that also clears `searchTyping`) and call it from all four sites.

### WR-07: Difftool full-file rendering sources content from the wrong side (`--difftool-remote`) for deleted files

**File:** `cmd/alturd/main.go:251-254` (`loadDifftoolFiles`), `internal/tui/model.go:371` (`refreshDiffContent`)
**Issue:** `loadDifftoolFiles` always reads `--difftool-remote` into `DifftoolInfo.NewFileLines`, regardless of whether the parsed file is a deletion:
```go
var newFileLines []string
if data, readErr := os.ReadFile(remote); readErr == nil {
	newFileLines = splitDifftoolFileLines(data)
}
```
`refreshDiffContent` then feeds this unconditionally into `diff.RenderFull`/`diff.AlignFull` as `fileLines` whenever `m.difftool.Enabled`. But `AlignFull`'s own doc comment (`internal/diff/align.go:255-257`) is explicit: `fileLines` must be *"the new file for modified/added files, **the old file for deleted files**"*. For a deletion, `--difftool-remote` is git's `/dev/null` (or an empty temp file) — the *new* (nonexistent) side — so `NewFileLines` ends up empty exactly when the file is deleted, i.e. exactly the case where `AlignFull` needs the *old* content instead. Tellingly, the standalone (non-difftool) code path already gets this right: `fetchFileLines` (`internal/tui/model.go:924-929`) explicitly special-cases `f.IsDelete` and reads `git show HEAD:OldName` instead of the new-file path. The difftool path has no equivalent branch — it only ever reads one side (`remote`), never `local`.
Impact today is masked: a full-file deletion diff (`git diff --no-index local /dev/null`) is always a single hunk spanning the entire old file, so `AlignFull`'s inter-hunk-context-gap logic (the only place `fileLines` is consulted) never needs to fill a gap, and the deleted lines themselves render correctly straight from the hunk data. But this is correct by accident, not by design: it silently breaks (blank inter-hunk context) the moment a deletion diff isn't representable as one contiguous hunk, and there is no test in this file list that exercises FullFile mode against a deleted file in difftool mode to catch a regression here.
**Fix:** Read `local` instead of `remote` when the parsed file is a deletion:
```go
files, err := diff.Parse(reader)
// ...
refPath := remote
if len(files) > 0 && files[0].IsDelete {
	refPath = local
}
var newFileLines []string
if data, readErr := os.ReadFile(refPath); readErr == nil {
	newFileLines = splitDifftoolFileLines(data)
}
```
(and consider renaming `DifftoolInfo.NewFileLines` to something contract-neutral, e.g. `RefFileLines`, since it no longer always holds "new" content).

## Info

### IN-01: `gofmt` formatting violations in three reviewed files

**File:** `cmd/alturd/main.go`, `internal/diff/render.go`, `internal/tui/model.go`
**Issue:** Running `gofmt -l` directly against the reviewed file set flags all three files. `.golangci.yml` enables `staticcheck`, `govet`, `errcheck`, `revive` but has no `formatters` section (golangci-lint v2 separates gofmt/goimports into `formatters`), so this drift does not fail CI or local `golangci-lint run`.
**Fix:** `gofmt -w cmd/alturd/main.go internal/diff/render.go internal/tui/model.go`, and add a `formatters` block enabling `gofmt`/`goimports` to `.golangci.yml` so future drift is caught automatically.

### IN-02: `install-difftool` writes a bare `alturd` command, relying entirely on `PATH`

**File:** `cmd/alturd/difftool.go:45`
**Issue:** `difftoolCmdTemplate` hardcodes the literal command name `alturd` rather than the currently-running executable's resolved path. A user who runs `install-difftool` from a binary that isn't on `PATH` yet (e.g. `./alturd install-difftool` immediately after extracting a downloaded release archive, before moving it into `/usr/local/bin`) will get a gitconfig entry that fails with "alturd: command not found" the next time `git difftool` is invoked — even though the very binary that wrote the config is sitting right there. This may be an intentional trade-off (a PATH-relative command survives the binary being moved/updated later, whereas a baked-in absolute path would not), but that trade-off isn't documented anywhere in the reviewed files or in the install-difftool success message.
**Fix (if not an accepted trade-off):** Resolve `os.Executable()` (following symlinks) and use it as the default `cmd` value, or at minimum note the PATH requirement in the "Installed alturd as git difftool..." success message.

### IN-03: `os.ReadFile(remote)` failure in `loadDifftoolFiles` is silently discarded

**File:** `cmd/alturd/main.go:251-254`
**Issue:**
```go
var newFileLines []string
if data, readErr := os.ReadFile(remote); readErr == nil {
	newFileLines = splitDifftoolFileLines(data)
}
```
`readErr` is checked only to decide whether to populate `newFileLines` — it is never logged or otherwise surfaced. If reading git's own `--difftool-remote` temp file fails (permissions, an unusual git cleanup race, etc.), the only symptom is unexplained blank content in full-file mode, with nothing recorded anywhere for a user or maintainer to diagnose, even though `internal/log` (`applog`) is already initialized in `run()` specifically for this kind of non-fatal diagnostic.
**Fix:** Log `readErr` via `applog` (e.g. `charmlog.Warn("reading difftool remote file", "path", remote, "err", readErr)`) instead of discarding it.

### IN-04: Tree-pane name truncation has no ellipsis, unlike the difftool title bar

**File:** `internal/tui/model.go:472`
**Issue:** `renderTree()` truncates long file/directory names via `lipgloss.NewStyle().MaxWidth(m.treeWidth).Render(line)`. `difftoolTitleBar()`'s own doc comment (`model.go:292-299`) explains in detail why `lipgloss.Style.MaxWidth` is unsuitable for this exact purpose: it truncates via a hardcoded empty tail internally and cannot append an ellipsis, which is why the title bar was switched to `ansi.Truncate(title, m.termWidth, "…")`. The tree pane has the identical truncation-without-indicator problem (an overflowing filename in a narrow/unfocused tree column is silently clipped) but was never migrated to the same fix, so the same UI now behaves inconsistently: the difftool title bar marks truncation with "…", the tree pane does not.
**Fix:** Apply the same `ansi.Truncate(line, m.treeWidth, "…")` pattern to the per-row truncation in `renderTree`.

### IN-05: `config.Load` treats every `xdg.SearchConfigFile` error as "no config file found"

**File:** `internal/config/config.go:60-64`
**Issue:**
```go
found, err := xdg.SearchConfigFile("alturd/config.toml")
if err != nil {
	// Not found — silently use defaults (D-03).
	return DefaultConfig(), nil
}
```
Any error from `xdg.SearchConfigFile` — not just "no matching file in any XDG config directory" — is swallowed and mapped to "use defaults, no error." If one of the XDG config directories on the search path is unreadable (e.g. a permissions problem on `$XDG_CONFIG_HOME` itself, or a broken symlink), the user gets silent default behavior with no indication that their real config was never even looked at, which could be confusing to debug ("my config changes aren't taking effect and there's no error").
**Fix:** If `adrg/xdg` exposes a way to distinguish "not found" from other I/O errors (or if a type assertion/sentinel is available), only swallow the specific "not found" case and surface anything else as a genuine `config:`-prefixed error.

### IN-06: `gitConfigRun`'s "outside a repository" detection is a broad substring match applied to every `git config` invocation

**File:** `cmd/alturd/difftool.go:161-175`
**Issue:**
```go
if strings.Contains(strings.ToLower(stderrStr), "git repository") {
	return "", exitErr.ExitCode(), git.ErrLocalScopeOutsideRepo
}
```
This check runs for *every* `git config` subprocess invocation `gitConfigRun` makes (both `--get` and write calls, for both `--global` and `--local` scope), not just the `--local`-outside-a-repo case it's documented to handle. It's a deliberate, documented deviation from a narrower literal-string match (per the in-code comment, chosen because git's exact wording varies by version), which is a reasonable trade-off — but it means any unrelated `git config` failure whose stderr happens to mention "git repository" (for example, wording changes in a future git version, or a `--global` write failing for a reason that happens to reference "repository") would be misreported as `ErrLocalScopeOutsideRepo` with its fixed "--scope local requires running inside a git repository." message, which would be actively misleading if the actual scope in play was `--global`.
**Fix:** At minimum, scope the substring check to only apply when `scopeFlag == "--local"`, so a `--global` failure can never be misclassified as the local-scope error.

---

_Reviewed: 2026-08-06T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
