---
phase: 04-config-theming-difftool-distribution
reviewed: 2026-08-06T19:30:00Z
depth: standard
files_reviewed: 25
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
  info: 8
  total: 15
status: issues_found
---

# Phase 04: Code Review Report

**Reviewed:** 2026-08-06T19:30:00Z
**Depth:** standard
**Files Reviewed:** 25
**Status:** issues_found

## Summary

This pass re-reviews the full Phase 04 scope (config/keybindings/theme resolution, `install-difftool`, difftool-mode wiring, CI/release workflows) as it stands after Plan 04-07 (the G-04-1 `difftool.trustExitCode` fix), which landed after an earlier review of this same file set. `go build ./...` and `go vet ./...` are both clean. `gofmt -l` was run directly against the reviewed files and still flags three of them (IN-01).

The security-sensitive surfaces remain sound: all `exec.Command` calls use argv form (never a shell string built from untrusted input), `--name` is regex-anchored before being interpolated into a gitconfig key, the published `difftool.<name>.cmd` template's `"$LOCAL"/"$REMOTE"/"$MERGED"` double-quoting is correct (a malicious working-tree filename containing `$(...)` cannot trigger command substitution through a plain parameter expansion), and `GIT_DIFF_PATH_COUNTER`/`GIT_DIFF_PATH_TOTAL` are validated as positive integers before ever reaching a `%d` format verb. No Critical/security-relevant defect was found. The two previously-fixed crash bugs (the `strings.Repeat` panic on short terminals, and the `os.Exit(1)` that bypassed bubbletea's terminal restore on abort) remain fixed in this tree.

Two WARNING findings are genuine, still-live correctness bugs, not style nits: **WR-02** (`diffW` going negative and silently blanking the diff pane on a narrow terminal with the tree pane focused) is explicitly flagged as a known-but-unaddressed gap in the code's own comments (`model.go:336`, `"diffW (WR-02) stays untouched"`), confirming this review's finding rather than introducing a new one. **WR-07** (difftool full-file mode reading the wrong side — `--difftool-remote` instead of `--difftool-local` — for deleted files) directly contradicts `AlignFull`'s own documented contract and has no test coverage exercising FullFile mode against a deletion in difftool mode to catch it; its effect is currently masked only because a whole-file deletion always parses as one contiguous hunk.

## Warnings

### WR-01: `refreshTreeContent` resizes the wrong viewport

**File:** `internal/tui/model.go:416-420`
**Issue:** A function named and documented as refreshing the tree pane instead sets `m.diffVP`'s width and never touches `m.treeVP`'s width:
```go
func (m *model) refreshTreeContent() {
	m.diffVP.SetWidth(m.termWidth - m.treeWidth - 1)
	content := m.renderTree()
	m.treeVP.SetContent(content)
}
```
`treeVP.SetWidth` is set only inside `handleResize` (`model.go:353`). Every current call site of `refreshTreeContent` (`handleResize` itself, `treeIdxMove`, `toggleAllFiles`, `treeToggleExpand`) either runs immediately after `handleResize` already set `treeVP`'s width correctly, or never changes `treeWidth` at all — so today the misdirected call is a harmless no-op that recomputes a value `diffVP` already had. A future call site that invokes `refreshTreeContent()` after changing `treeWidth` without going through `handleResize` first will render the tree pane at a stale width while this function pointlessly re-touches `diffVP`.
**Fix:**
```go
func (m *model) refreshTreeContent() {
	m.treeVP.SetWidth(m.treeWidth)
	content := m.renderTree()
	m.treeVP.SetContent(content)
}
```

### WR-02: `diffW` is never clamped and goes negative on a narrow terminal with the tree pane focused

**File:** `internal/tui/model.go:351-356`
**Issue:**
```go
diffW := w - m.treeWidth - 1

m.treeVP.SetWidth(m.treeWidth)
m.treeVP.SetHeight(contentH)
m.diffVP.SetWidth(diffW)
m.diffVP.SetHeight(contentH)
```
`m.treeWidth` is `treeWidthFocused` (45) whenever the tree pane has focus (`model.go:37`). On any terminal narrower than 46 columns — realistic in a tmux split or a resized terminal emulator — `diffW` goes negative. I traced this into `charm.land/bubbles/v2@v2.1.0/viewport/viewport.go`: `SetWidth` stores the raw (possibly negative) value with no clamp (`viewport.go:178-180`); `maxWidth()` clamps to `max(0, ...)` (`viewport.go:316-322`), and `visibleLines()` then early-returns `nil` the moment `maxWidth()` hits 0 (`viewport.go:336-338`) — so no panic occurs, but the entire diff pane silently renders blank with no indication to the user. This is not a hypothetical: the code's own comment at `model.go:336` already names this exact issue — `"Height only; diffW (WR-02) stays untouched."` — confirming it is a known, currently-unaddressed gap rather than new. `internal/diff/render.go:61-63` already establishes the correct pattern for this input class (clamp to a floor of 6 before use); this call site does not follow it.
**Fix:** Clamp `diffW` (and/or `m.treeWidth`) to a sane minimum before calling `SetWidth`, mirroring `internal/diff/render.go`'s width floor.

### WR-03: `fetchFileLines`'s working-tree fallback resolves a repo-relative path against the process's CWD

**File:** `internal/tui/model.go:965-970`
**Issue:**
```go
	// Last resort: read from the working tree.
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return splitFileBytes(data), nil
```
`name` is repo-root-relative — that's exactly why the two preceding attempts (`git show HEAD:name` at `model.go:953`, `git show :name` at `model.go:960`) work regardless of the caller's working directory. `os.ReadFile` resolves `name` relative to the process's actual working directory, not the repository root. When alturd is invoked from a subdirectory of the repo (an entirely normal way to run any git tool) and the target file is untracked/unstaged (so both `git show` calls legitimately fail), this fallback silently fails to find a file that does exist on disk. `refreshDiffContent` (`model.go:383-393`) then quietly degrades from full-file to hunk-only rendering with nothing surfaced to the user — DIFF-05 full-file mode simply stops working for that file with no error.
**Fix:** Resolve `name` against the repository root (e.g. a cached `git rev-parse --show-toplevel`) before this final `os.ReadFile` fallback.

### WR-04: `DetectDarkBackground`'s abandoned query goroutine can race the TUI for the first keystroke

**File:** `internal/config/theme.go:66-78`
**Issue:** When the OSC 11 round-trip exceeds `DetectTimeout` (50ms), `DetectDarkBackground` returns `true` (dark fallback) while its query goroutine is still running and still holds a read on the same terminal file descriptor `tea.NewProgram(m).Run()` (`cmd/alturd/main.go:175-176`) takes over immediately afterward. The function's own doc comment already flags this as unverified (`FA-04-02`); nothing in the reviewed files closes that gap. On a slow/unresponsive terminal — precisely the scenario the timeout exists for — the abandoned reader can consume the user's very first keystroke to the freshly-started bubbletea program, an intermittent, hard-to-reproduce "first key dropped" symptom.
**Fix:** Bind the OSC 11 query to a cancelable read, or explicitly drain/close the relevant fd path before `tea.NewProgram` starts, so the goroutine is provably finished rather than merely abandoned.

### WR-05: CI runs the test suite without the race detector

**File:** `.github/workflows/ci.yml:20`
**Issue:** `run: go test ./...` omits `-race`. This project's own stack guidance (project `CLAUDE.md`) states the race detector should be enabled in CI specifically because "bubbletea v2 is goroutine-safe but custom async code needs validation." This phase adds exactly that class of code: the goroutine/buffered-channel pattern in `DetectDarkBackground` (`internal/config/theme.go:67-70`, see WR-04) and the goroutine/timeout pattern in `computeIntraLineWithTimeout` (`internal/diff/render.go:311-323`) — neither is ever exercised under `-race` in CI.
**Fix:**
```yaml
      - run: go test -race ./...
```

### WR-06: The "close search and reset state" sequence is duplicated four times in `handleKey`

**File:** `internal/tui/model.go:505-512, 526-532, 539-546, 547-554`
**Issue:** The block that closes search mode —
```go
m.searchMode = false
m.searchInput.Reset()
m.searchMatches = nil
m.searchMatchIdx = 0
m.handleResize(m.termWidth, m.termHeight)
```
(plus `m.searchTyping = false` in the typing-phase `esc` variant) is copy-pasted at four call sites: typing-phase `esc`, navigation-phase `esc`, navigation-phase `]`, and navigation-phase `[`. A future change to search-close behaviour (e.g. clearing an added field, or fixing an ordering issue) is likely to land in some but not all four copies, silently producing inconsistent state between the `esc` path and the `[`/`]` paths.
**Fix:** Extract a `closeSearch()` helper (parameterised for the one variant that also clears `searchTyping`) and call it from all four sites.

### WR-07: Difftool full-file rendering sources content from the wrong side for deleted files

**File:** `cmd/alturd/main.go:261-264` (`loadDifftoolFiles`); contract defined in `internal/diff/align.go:255-261` (`AlignFull` doc comment)
**Issue:** `loadDifftoolFiles` always reads `--difftool-remote` into `DifftoolInfo.NewFileLines`, with no branch on whether the file is a deletion:
```go
var newFileLines []string
if data, readErr := os.ReadFile(remote); readErr == nil {
	newFileLines = splitDifftoolFileLines(data)
}
```
`refreshDiffContent` (`internal/tui/model.go:376-383`) feeds this straight into `diff.RenderFull`/`diff.AlignFull` whenever difftool mode is enabled. But `AlignFull`'s own doc comment is explicit that `fileLines` must be *"the new file for modified/added files, **the old file for deleted files**"* (`align.go:255-257`), and `alignTextFull`'s inter-hunk-context logic (`align.go:295-321`) indexes into `fileLines` by old-file position specifically when `file.IsDelete`. For a deletion, `--difftool-remote` is git's `/dev/null` (or empty) — the wrong side — so `NewFileLines` ends up empty exactly in the one case (`IsDelete`) where `AlignFull` needs old-file content. The standalone (non-difftool) code path already gets this right: `fetchFileLines` (`internal/tui/model.go:934-940`) explicitly branches on `f.IsDelete` and reads `git show HEAD:OldName`. The difftool path has no equivalent branch.
Impact is currently masked: a whole-file deletion diff (`git diff --no-index local /dev/null`) is always exactly one hunk spanning the entire old file, so `alignTextFull`'s inter-hunk-gap-filling from `fileLines` never actually needs to run for a deletion, and the deleted lines themselves render correctly straight from hunk data regardless. This is correct by accident, not by design, and no test in the reviewed file list exercises FullFile mode against a deleted file in difftool mode to catch a regression.
**Fix:**
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
Consider also renaming `DifftoolInfo.NewFileLines` (e.g. to `RefFileLines`) since it no longer always holds "new" content once this is fixed.

## Info

### IN-01: `gofmt` formatting drift in three reviewed files, uncaught by CI

**File:** `cmd/alturd/main.go`, `internal/diff/render.go`, `internal/tui/model.go`
**Issue:** `gofmt -l` run directly against these files flags all three. `.golangci.yml` enables `staticcheck`, `govet`, `errcheck`, `revive` but has no `formatters` section (golangci-lint v2 moved gofmt/goimports checks under `formatters`), so this drift passes CI silently.
**Fix:** `gofmt -w cmd/alturd/main.go internal/diff/render.go internal/tui/model.go`; add a `formatters` block enabling `gofmt`/`goimports` to `.golangci.yml` so future drift is caught automatically.

### IN-02: `install-difftool` writes a bare `alturd` command, relying entirely on `PATH`

**File:** `cmd/alturd/difftool.go:58`
**Issue:** `difftoolCmdTemplate` hardcodes the literal command name `alturd`, not the resolved path of the currently-running executable. A user who runs `./alturd install-difftool` immediately after extracting a downloaded release archive — before it's on `PATH` — gets a gitconfig entry that fails with "alturd: command not found" on the next `git difftool` invocation, even though the exact binary that wrote the config is right there. This may be an intentional trade-off (PATH-relative survives the binary being moved/upgraded later; a baked-in absolute path would not), but nothing in the reviewed files documents that trade-off or warns the user.
**Fix (if not an accepted trade-off):** Resolve `os.Executable()` and use it as the default `cmd` value, or note the PATH requirement in the "Installed alturd as git difftool..." success message.

### IN-03: `os.ReadFile(remote)` failure in `loadDifftoolFiles` is silently discarded

**File:** `cmd/alturd/main.go:261-264`
**Issue:** `readErr` is checked only to decide whether `newFileLines` gets populated — never logged. If reading git's own `--difftool-remote` temp file fails (permissions, an unusual cleanup race), the only symptom is unexplained blank content in full-file mode, even though `internal/log` (`applog`) is already initialised in `run()` for exactly this kind of non-fatal diagnostic.
**Fix:** Log `readErr` via `applog`/`charmlog` instead of silently discarding it.

### IN-04: Tree-pane name truncation has no ellipsis, unlike the difftool title bar

**File:** `internal/tui/model.go:483`
**Issue:** `renderTree()` truncates long names via `lipgloss.NewStyle().MaxWidth(m.treeWidth).Render(line)`. `difftoolTitleBar()`'s own doc comment (`model.go:303-306`) explains in detail why `lipgloss.Style.MaxWidth` is the wrong tool for this: it truncates via a hardcoded empty tail internally and cannot append an ellipsis — which is why the title bar was switched to `ansi.Truncate(title, m.termWidth, "…")`. The tree pane has the identical truncation problem (an overflowing filename in a narrow/unfocused tree column is silently clipped) but was never migrated, so the same application now truncates inconsistently: the difftool title bar marks truncation with "…", the tree pane does not.
**Fix:** Apply the same `ansi.Truncate(line, m.treeWidth, "…")` pattern in `renderTree`.

### IN-05: `config.Load` treats every `xdg.SearchConfigFile` error as "no config file found"

**File:** `internal/config/config.go:59-64`
**Issue:**
```go
found, err := xdg.SearchConfigFile("alturd/config.toml")
if err != nil {
	// Not found — silently use defaults (D-03).
	return DefaultConfig(), nil
}
```
Any error — not just "no file found on the search path" — is swallowed into "use defaults, no error." A permissions problem on `$XDG_CONFIG_HOME` itself, or some other I/O failure during the search, produces silent default behaviour with no indication the user's real config was never consulted.
**Fix:** If `adrg/xdg` exposes a way to distinguish "not found" from other I/O errors, only swallow the former and surface anything else as a `config:`-prefixed error.

### IN-06: `gitConfigRun`'s "outside a repository" substring match applies to every invocation, not just `--local`

**File:** `cmd/alturd/difftool.go:198-210`
**Issue:**
```go
if strings.Contains(strings.ToLower(stderrStr), "git repository") {
	return "", exitErr.ExitCode(), git.ErrLocalScopeOutsideRepo
}
```
This runs for every `git config` subprocess `gitConfigRun` makes — both `--get` and write calls, for both `--global` and `--local` scope — not only the `--local`-outside-a-repo case it exists to handle. It's a documented, deliberate deviation (git's exact wording for this case varies by version), but as written any unrelated `git config` failure whose stderr happens to mention "git repository" would be misreported as `ErrLocalScopeOutsideRepo`, printing "--scope local requires running inside a git repository." even for a `--global` invocation.
**Fix:** Guard the substring check with `scopeFlag == "--local"` so a `--global` failure can never be misclassified as the local-scope error.

### IN-07: Duplicated file-line-splitting helper across two packages

**File:** `cmd/alturd/main.go:359-368` (`splitDifftoolFileLines`), `internal/tui/model.go:983-994` (`splitFileBytes`)
**Issue:** Both functions implement identical logic (CRLF normalisation, trailing-newline strip, empty-to-nil, `strings.Split`); `main.go`'s comment explicitly notes this is a deliberate duplication rather than an exported shared function. A future bug fix applied to one copy (e.g. handling of a lone trailing `\r`) is unlikely to be applied to the other, silently reintroducing the same class of bug in the other call path.
**Fix:** Export one implementation and call it from both packages.

### IN-08: Test env-var filter in `TestExitCodeGitNotFound` drops more than intended

**File:** `cmd/alturd/main_test.go:163-171`
**Issue:**
```go
for _, e := range os.Environ() {
	if len(e) > 0 && e[0:1] != "P" { // Skip PATH* vars
		if !strings.HasPrefix(e, "PATH") {
			env = append(env, e)
		}
	}
}
```
The outer condition excludes every env var whose name starts with `P` (e.g. `PWD`, `PROMPT`, `PYTHONPATH`), not just `PATH*` as the comment claims; the inner `!strings.HasPrefix(e, "PATH")` check is then unreachable in effect, since nothing that survives the outer filter could ever start with `"PATH"`. Doesn't currently break the test, but it silently strips more environment state than intended and could mask an environment-dependent failure in an unusual CI shell.
**Fix:**
```go
for _, e := range os.Environ() {
	if !strings.HasPrefix(e, "PATH") {
		env = append(env, e)
	}
}
```

---

_Reviewed: 2026-08-06T19:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
