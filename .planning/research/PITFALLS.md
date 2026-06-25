# Pitfalls Research

**Domain:** Go bubbletea TUI — cross-platform git diff viewer (port of Python/Textual)
**Researched:** 2026-06-25
**Confidence:** MEDIUM (web research, community issue trackers, verified against multiple sources)

---

## Critical Pitfalls

### Pitfall 1: Panic Inside tea.Cmd Leaves Terminal in Raw Mode

**What goes wrong:**
A panic occurring inside a `tea.Cmd` goroutine is NOT caught by bubbletea's `CatchPanics` recovery mechanism. The terminal is left in raw mode with the cursor hidden. The user's shell is effectively unusable until they run `reset` manually.

**Why it happens:**
`CatchPanics` only covers the main event loop goroutine. Commands run in separate goroutines that bubbletea does not wrap with panic recovery. The cleanup sequence (restoring terminal, showing cursor, exiting alt screen) is bypassed entirely.

**How to avoid:**
- Wrap every non-trivial `tea.Cmd` body with a `defer func() { if r := recover(); r != nil { /* send error msg */ } }()` pattern.
- Never let panics propagate out of command functions — return them as error messages to the event loop.
- Keep `CatchPanics` enabled (it is on by default — do not use `tea.WithoutCatchPanics()`).

**Warning signs:**
- Integration tests where a subprocess panics leave the test runner's terminal broken.
- Any `tea.Cmd` that does file I/O, git subprocess calls, or diff parsing has panic surface area.

**Phase to address:** Phase 1 (TUI scaffold) — establish the `tea.Cmd` error-return pattern before building any business logic on top.

---

### Pitfall 2: Windows Resize Events Never Fire (Bubbletea v2 Regression)

**What goes wrong:**
In bubbletea v2, `WindowSizeMsg` is sent once at startup but never again when the user resizes the Windows Terminal window. The layout never adapts. The diff pane and file tree remain at their startup dimensions regardless of window size changes.

**Why it happens:**
Bubbletea v2 switched Windows input to VT input mode (`ENABLE_VIRTUAL_TERMINAL_INPUT`). The old `readConInputs` path in `key_windows.go` handled `coninput.WindowBufferSizeEventRecord` and converted it to `WindowSizeMsg`. That path was removed. Windows has no `SIGWINCH` equivalent, and `listenForResize` in `signals_windows.go` is a documented no-op. This is an active open regression (issue #1601, reported Feb 2026).

**How to avoid:**
Implement a polling workaround from day one on Windows:
```go
// In Init(), return a tick command that queries terminal size
func pollWindowSize() tea.Msg {
    w, h, _ := term.GetSize(int(os.Stdout.Fd()))
    return tea.WindowSizeMsg{Width: w, Height: h}
}
// Tick every 250ms on Windows only
```
Use build tags to keep the polling path Windows-only; Unix uses SIGWINCH via bubbletea normally.

**Warning signs:**
- Layout looks correct at startup but breaks after resizing on Windows.
- Side-by-side diff columns overflow or truncate incorrectly after a window resize.

**Phase to address:** Phase 1 (TUI scaffold) — test resize on Windows as part of the initial skeleton before the layout is complex.

---

### Pitfall 3: Using len() or len([]rune()) for String Width With ANSI Content

**What goes wrong:**
Side-by-side diff columns misalign when content contains: ANSI escape sequences (invisible bytes that len() counts), CJK ideographs (width 2 but len([]rune()) returns 1), or ambiguous-width Unicode characters (Cyrillic, Greek — classified as ambiguous in Unicode East Asian Width tables).

With chroma syntax highlighting, every line contains multiple invisible ANSI bytes for color codes. Using byte length or rune count to compute column padding produces completely wrong results. Lines appear shifted, columns bleed into each other, and the alignment breaks progressively worse as source code uses more Unicode.

**Why it happens:**
Go developers instinctively reach for `len(s)` (bytes) or `len([]rune(s))` (code points) to measure string length. Neither measures visual terminal width. ANSI escape sequences take 0 visual cells but multiple bytes. CJK characters take 2 visual cells but 1 rune. This is a pervasive Go string-handling trap.

**How to avoid:**
- Use `github.com/mattn/go-runewidth` (`runewidth.StringWidth()`) for all visual column width calculations.
- Use `lipgloss.Width()` for measuring lipgloss-styled strings (it is ANSI-aware).
- Use `ansi.StringWidth()` from `github.com/muesli/ansi` as an alternative.
- Never use `len(s)` for anything that will be placed in a terminal column.
- After chroma highlighting, run all lines through ANSI-aware padding/truncation before placing them in side-by-side columns.

**Warning signs:**
- Source files with Japanese, Chinese, Korean, Arabic, Cyrillic, or Greek characters cause diff column misalignment.
- Column borders visually shift when syntax highlighting is enabled vs. disabled.
- The right column starts at a different X position than expected.

**Phase to address:** Phase 2 (diff renderer) — enforce the runewidth convention as an invariant in the diff rendering layer; add test cases with multi-byte Unicode source files.

---

### Pitfall 4: Viewport Ghost Lines From Stale ANSI Content

**What goes wrong:**
When the content of a bubbletea viewport component is updated (e.g., switching to a different diff file), stale "ghost lines" from the previous content remain visible in the middle of the viewport. They do not appear at the top or bottom — always in the same position — and persist until the viewport is resized or the program restarts.

**Why it happens:**
Wide runes and ANSI sequences in the previous content disturb the cell count that the renderer uses to determine how many cells to erase. The renderer erases only what it thinks was rendered, leaving some cells untouched. This is a known open issue in bubbletea (issue #1477).

**How to avoid:**
- Use `viewport.SetContent()` rather than constructing custom viewport content through raw string mutations.
- After switching files, explicitly force a viewport repaint by sending a `tea.WindowSizeMsg` with the current dimensions.
- Alternatively, replace the viewport model with a fresh instance when switching to a new file.
- If using `HighPerformanceRendering`, test switching files specifically — this mode has more edge cases with stale content.

**Warning signs:**
- Visual artifacts remain when navigating between diff files.
- Lines from the previous file's diff appear "through" the new file's content.
- Artifacts are reproducible but position-dependent.

**Phase to address:** Phase 2 (diff renderer) — validate file-switch rendering in manual testing before declaring Phase 2 complete.

---

### Pitfall 5: Blocking the Update Loop With I/O

**What goes wrong:**
Running a git subprocess, reading config files, parsing a large diff, or computing intra-line diffs synchronously inside `Update()` freezes the TUI for the duration of the operation. The user sees an unresponsive screen. On slow disks or large repositories, this is 200–500ms or more per operation.

**Why it happens:**
The bubbletea event loop is single-threaded. `Update()` and `View()` run synchronously. Any blocking call inside `Update()` blocks the entire render pipeline. Developers porting from Textual (which uses async/await) often underestimate this because Textual's event handlers appear synchronous but actually run in an async event loop.

**How to avoid:**
- All git subprocess calls (loading file tree, loading diff content) must be wrapped in `tea.Cmd`.
- All file I/O (reading config, reading files for syntax highlighting) must be `tea.Cmd`.
- Intra-line diff computation (which can be up to 100ms for large hunks per the Python implementation's guards) must be `tea.Cmd`, not inline in `Update()`.
- The model should store a "loading" state and display a spinner while commands are running.

**Warning signs:**
- The cursor stops responding for any period when navigating between files.
- Key presses are dropped during file load transitions.
- `go tool pprof` shows `Update()` spending time in syscall or I/O.

**Phase to address:** Phase 1 (TUI scaffold) — establish the tea.Cmd pattern for all I/O before connecting real data sources.

---

### Pitfall 6: ANSI Color Bleed Across Side-by-Side Diff Columns

**What goes wrong:**
When syntax-highlighted lines from chroma are placed side-by-side in the diff view, ANSI color state from the left column "bleeds" into the right column. The right column inherits the last color set in the left column, producing incorrect colors throughout.

**Why it happens:**
Terminal ANSI color state is global within a render frame. When the left column ends mid-color (e.g., in the middle of a string literal that continues across the column boundary), the right column inherits that color. Each column is an independent logical unit but the terminal doesn't know that.

**How to avoid:**
- Emit an explicit ANSI reset sequence (`\x1b[0m`) at the end of every left-column line before rendering the separator and right column.
- Use chroma's `ANSI` formatter with explicit reset-at-EOL behavior or post-process output to inject resets.
- When joining left and right column strings with lipgloss `lipgloss.JoinHorizontal()`, verify that lipgloss inserts a reset between columns (it should, but validate this explicitly).

**Warning signs:**
- Right column text appears in a color that matches what the left column was using at its end.
- The effect is strongest in lines that end mid-token (e.g., end of line inside a multi-line string).
- Color bleed is absent when syntax highlighting is disabled.

**Phase to address:** Phase 2 (diff renderer) — validate with a test case that renders a syntax-highlighted file side-by-side and checks right-column colors.

---

## Moderate Pitfalls

### Pitfall 7: Windows VTP Disabled by cmd.exe When Spawning Subprocesses

**What goes wrong:**
`cmd.exe` disables Virtual Terminal Processing for child processes it launches. If alturd is launched via `cmd.exe` (e.g., from a batch script or a CI pipeline step that uses `cmd /C alturd`), VTP is toggled off. ANSI escape codes appear as literal characters in the terminal rather than being interpreted as formatting.

**Why it happens:**
This is Windows console behavior by design. `cmd.exe` re-enables VTP for itself but child processes start with VTP disabled unless they explicitly call `SetConsoleMode` to enable it. Go's TUI libraries (termenv, lipgloss, bubbletea) call `SetConsoleMode` via Windows API on startup, so launching alturd directly (`alturd.exe`) is fine. The problem surfaces only when launched via `cmd.exe` as an intermediary shell.

**How to avoid:**
- Document that users should run `alturd.exe` directly from PowerShell or Windows Terminal, not via `cmd /C alturd`.
- In CI, use `shell: pwsh` not `shell: cmd` in GitHub Actions workflow steps.
- Test specifically: launch from `cmd.exe`, `PowerShell 5.1`, `PowerShell 7`, `Windows Terminal`, and check output.

**Warning signs:**
- ANSI escape sequences appear as literal `^[[32m` text in CI output.
- Tests pass in Windows Terminal but fail in CI.

**Phase to address:** Phase 4 (Windows support and CI) — include cmd.exe launch in Windows test matrix.

---

### Pitfall 8: CGO_ENABLED Inconsistency Silently Breaks Cross-Compiled Targets

**What goes wrong:**
GoReleaser defaults `CGO_ENABLED=1` for the host platform and `CGO_ENABLED=0` for all cross-compiled targets. If any dependency has a CGO code path (even a stub that silently activates when CGO is disabled, like `go-sqlite3`), the host binary has full functionality while the cross-compiled Linux/macOS/Windows binaries silently lack it. The release builds succeed with zero errors.

**Why it happens:**
Go's build system silently swaps to stub implementations when `CGO_ENABLED=0`. There is no warning that a C dependency was replaced. GoReleaser does not make this explicit.

**How to avoid:**
- Explicitly set `env: [CGO_ENABLED=0]` in the GoReleaser build config for all targets.
- Audit all dependencies with `go list -m -json all | grep -i cgo` and verify none have CGO requirements.
- For alturd specifically: bubbletea, lipgloss, chroma, and go-diff are all pure Go — CGO_ENABLED=0 is straightforward. Confirm this remains true when adding any new dependency.
- Add a test in CI that explicitly builds with `CGO_ENABLED=0` and runs the binary.

**Warning signs:**
- Any new dependency that mentions `C.h` or `#cgo` in its source.
- Build error on cross-compilation targets after adding a new library.
- `go build -v` shows "cgo" in the build graph.

**Phase to address:** Phase 4 (CI and distribution) — set CGO_ENABLED=0 explicitly in `.goreleaser.yaml` from the start.

---

### Pitfall 9: CRLF in Git Subprocess Output on Windows

**What goes wrong:**
Git commands run via `exec.Command("git", ...)` on Windows produce output with `\r\n` line endings instead of `\n`. Code that splits on `\n` and parses fields gets trailing `\r` in every value, causing string comparison failures, incorrect path detection, and silently wrong behavior in the diff parser.

**Why it happens:**
On Windows, git's text output uses CRLF. Go's `exec.Command` does not apply any line-ending normalization. The diff output parser, which expects LF-terminated lines (as specified by the unified diff format and as the Python implementation assumes), silently receives CRLF-terminated lines.

**How to avoid:**
- Wrap all git output through a normalizer: `strings.ReplaceAll(output, "\r\n", "\n")` or `bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))`.
- Apply normalization at the lowest level — immediately after `cmd.Output()` or reading from `cmd.Stdout`, before any parsing.
- Pass `-c core.autocrlf=false` to git subprocesses to suppress CRLF conversion: `exec.Command("git", "-c", "core.autocrlf=false", "diff", ...)`.
- Add a Windows-specific fixture test that validates CRLF handling in the diff parser.

**Warning signs:**
- File paths in git output have trailing `\r` when printed in error messages.
- Diff hunk headers fail to parse correctly on Windows only.
- Any test that passes on Linux/macOS but fails on Windows around string comparison.

**Phase to address:** Phase 3 (git integration) — normalize output at the exec boundary before passing to the diff parser.

---

### Pitfall 10: OSC 11 Background Query Fails Silently in Common Environments

**What goes wrong:**
The OSC 11 terminal background color query (`\033]11;?\007`) used for auto-theming (light/dark detection) fails or produces garbage output in several common environments: old tmux (no OSC 11 support), new tmux with SSH (escape-time too short causes response to leak as visible text), pipes and non-TTY contexts (CI, git difftool subprocess), and some terminal emulators that simply ignore the query.

**Why it happens:**
OSC 11 requires a round-trip: the app writes the query, the terminal writes a response back to the app's stdin. This requires a real TTY. In tmux, the query passes through the multiplexer layer which may buffer, delay, or misroute the response. The Go `termenv` library has known limitations with tmux OSC sequences.

**How to avoid:**
- Use a short timeout (50–100ms) for the OSC 11 query; if no response arrives, fall back to dark theme.
- Check `COLORFGBG` environment variable first (set by many terminal emulators without a round-trip).
- Check `$TERM_PROGRAM` and known terminal identifiers as a heuristic before attempting OSC 11.
- In difftool mode (subprocess of git), skip OSC 11 entirely and use the config-specified or default theme.
- Never block program startup waiting for OSC 11 — do the query asynchronously via `tea.Cmd`.
- Use `termenv.HasDarkBackground()` from `github.com/muesli/termenv` as the primary implementation — it handles the fallback chain.

**Warning signs:**
- Program hangs briefly at startup in tmux-over-SSH environments.
- Garbage text like `11;rgb:1c1c/1c1c/1c1c` appears at the top of the terminal output.
- Color scheme detection works in Alacritty but fails in tmux.

**Phase to address:** Phase 3 (theming) — implement OSC 11 with explicit timeout and fallback before exposing auto-theme to users.

---

### Pitfall 11: Signal Handler Accumulation in Difftool Mode

**What goes wrong:**
Difftool mode launches one bubbletea program per changed file (the git difftool protocol calls the binary once per file). Each invocation installs a SIGINT/SIGTERM signal handler via bubbletea but does not clean it up after the program exits normally. After N files, pressing Ctrl+C requires N additional presses to exit, because each previous signal handler intercepted the signal.

**Why it happens:**
Bubbletea registers a global OS signal handler via `signal.Notify()` in each `tea.Program`. Signal handlers registered with `signal.Notify()` accumulate in a Go process's lifetime. In a process that starts and stops multiple `tea.Program` instances, each one adds another handler.

**How to avoid:**
- In difftool mode, use `tea.WithoutSignalHandler()` option and manage SIGINT/SIGTERM manually with a single global signal handler for the process.
- Or, since each difftool invocation is a fresh process (git spawns a new binary per file), this may be a non-issue if alturd does not itself spawn sub-programs.
- Verify: does difftool mode ever create multiple `tea.Program` instances within a single process lifetime? If yes, use `WithoutSignalHandler`.

**Warning signs:**
- Multiple Ctrl+C presses required to exit a difftool session.
- Signal-related test flakiness when running multiple TUI programs in sequence.

**Phase to address:** Phase 3 (difftool integration) — validate signal behavior in difftool mode during integration testing.

---

### Pitfall 12: View() Called Before Initial WindowSizeMsg Arrives

**What goes wrong:**
Bubbletea sends the initial `WindowSizeMsg` asynchronously. On program start, `View()` is called at least once before the window dimensions are known. If `View()` computes column widths, truncation, or viewport sizes based on `m.width` and `m.height` (which start at zero), the first render is completely broken — zero-width columns, empty viewports, or index-out-of-bounds panics.

**Why it happens:**
The window size query is intentionally asynchronous in bubbletea (many programs don't need it and delaying startup for it would add latency). `View()` must produce valid output at zero dimensions.

**How to avoid:**
- Add a `ready bool` field to the root model; set it to `true` when the first `WindowSizeMsg` arrives.
- In `View()`, return a simple loading string (e.g., `"Initializing..."`) when `!m.ready`.
- Guard all dimension-based calculations with `if m.width == 0 { return "" }`.
- Set reasonable non-zero defaults for width/height in the model's initial state as a fallback.

**Warning signs:**
- Index-out-of-bounds or division-by-zero panics on startup.
- Empty or visually broken first render before the layout settles.
- Race detector (`go test -race`) reports access to width/height fields at startup.

**Phase to address:** Phase 1 (TUI scaffold) — establish the `ready` pattern in the root model before any layout code is added.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Hard-code terminal width (e.g., 80) instead of using WindowSizeMsg | Simpler initial layout code | Layout is wrong on any terminal width; breaks on resize | Never |
| Use `len(s)` for visual width | Zero extra dependencies | Column misalignment with any Unicode or ANSI content | Never in a diff viewer |
| Synchronous git calls in Update() | No tea.Cmd boilerplate | TUI freezes on every file switch; unusable on slow repos | Never |
| Skip CRLF normalization on Windows | Simpler code path | Diff parser silently fails on Windows | Never — it takes 1 line |
| Skip OSC 11 timeout, block waiting | Simpler code | Program hangs in tmux/SSH; startup appears frozen | Never |
| Single hardcoded theme, no auto-detect | No theme complexity | Poor UX on light terminals | Acceptable for MVP if config override exists |
| No -c core.autocrlf=false in git calls | No extra git args | Possible CRLF surprises on Windows git configs | Acceptable initially if CRLF normalization is applied at output parse |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Chroma highlighter | Call `Format()` and use output directly in aligned column | Post-process output through ANSI-aware truncation/padding; verify no color bleed across column boundary |
| go-diff / unified diff parser | Parse raw git diff bytes including CRLF | Normalize to LF before parsing; handle missing-newline-at-EOF marker (`\ No newline at end of file`) |
| git subprocess | Use `exec.Command("sh", "-c", "git diff")` | Use `exec.Command("git", "diff", ...)` directly — never invoke via sh/cmd.exe |
| bubbletea viewport | Call `viewport.SetContent()` with raw ANSI | Ensure content width does not exceed viewport width; use ANSI-aware truncation |
| termenv / OSC 11 | Block on OSC 11 response with no timeout | Query with 50ms timeout via tea.Cmd; fall back to COLORFGBG or default dark |
| goreleaser release | Use `fetch-depth: 1` in GitHub Actions checkout | Use `fetch-depth: 0` — goreleaser needs full git history for changelogs and tag resolution |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Intra-line diff computed synchronously in Update() | TUI freezes 50–200ms when selecting modified lines | Run diff via tea.Cmd; cache result per hunk | Every time a modified line is selected |
| Syntax highlighting entire file on every render in View() | CPU spikes; frame rate drops | Highlight once when file is loaded; cache result in model | On any large file (>500 LOC) |
| Viewport.SetContent() called on every key press | Unnecessary full-viewport re-render | Only call SetContent when content changes; use scrolling for navigation | In any scroll-heavy interaction |
| git log/diff called in View() | Deadlock (View is synchronous) | All git calls are tea.Cmd in Init() or triggered by user actions | First render |
| go-diff parsing 10MB+ diff in Update() | Event loop blocks for seconds | Stream/chunk diff or run parsing as tea.Cmd | On monorepo commits with massive diffs |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| No loading state during file switch | Screen appears frozen; user presses keys that queue up | Show spinner or "Loading..." in diff pane while tea.Cmd runs |
| Keybindings consume keys during text search | User types search query but `n`/`N` navigate instead of entering text | Track focus mode: search input captures all printable keys, disables navigation keys |
| No pane indicator for current focus | User can't tell if Tab focus is on file tree or diff pane | Show visual indicator (border color change, "> " prefix) on focused pane |
| Status bar absent at zero terminal height | Crash or blank screen when terminal is very narrow | Guard all rendering on `m.ready && m.height > minimumHeight` |
| Cursor hidden after crash | User's terminal is unusable after alturd crashes | `defer` a cursor-restore sequence in main() before any TUI starts |

---

## "Looks Done But Isn't" Checklist

- [ ] **Windows resize handling**: Manually resize the Windows Terminal window mid-session — layout should adapt. Verify this works, not just startup dimensions.
- [ ] **Unicode source files**: Test a diff with Japanese, Chinese, or Cyrillic identifiers in source — columns must stay aligned.
- [ ] **ANSI color bleed**: Syntax-highlight a file where the left column ends mid-string-literal — right column must not inherit that color.
- [ ] **File switch rendering**: Switch between 5+ diff files rapidly — no ghost lines from previous file should remain.
- [ ] **Panic recovery**: Trigger a panic in a tea.Cmd (e.g., nil pointer in diff parser) — verify terminal is restored to usable state.
- [ ] **CRLF normalization**: Run on Windows and verify git output is parsed correctly (no trailing `\r` in paths or hunk headers).
- [ ] **OSC 11 in tmux**: Launch inside tmux — program should start without hanging and without leaking control sequences.
- [ ] **CGO_ENABLED=0**: Build with `CGO_ENABLED=0` explicitly and run all tests — confirm no silent stub activation.
- [ ] **goreleaser changelog**: Run goreleaser with `fetch-depth: 0` checkout — changelog must include recent commits.
- [ ] **Difftool signal handling**: Run `git difftool` on a repo with 10 changed files — Ctrl+C should exit immediately on first press.

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Ghost lines in viewport | LOW | Force viewport repaint by sending WindowSizeMsg; or replace viewport model on file switch |
| Panic leaves terminal raw | LOW | User runs `reset`; add defer cursor restore to main() to prevent recurrence |
| Windows resize broken | MEDIUM | Implement tick-based polling workaround; add build tag to isolate to Windows |
| Column misalignment from len() | MEDIUM | Replace all width calculations with runewidth.StringWidth; test with Unicode fixtures |
| CRLF in git output causes parse failure | LOW | Add LF normalization at exec boundary; add Windows-specific test |
| CGO_ENABLED inconsistency | LOW | Set explicit `env: [CGO_ENABLED=0]` in .goreleaser.yaml |
| OSC 11 hangs startup | LOW | Add 50ms timeout with COLORFGBG fallback; easy to add after the fact |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| View() before WindowSizeMsg | Phase 1: TUI scaffold | Add `ready` flag; test with simulated startup sequence |
| Blocking Update() with I/O | Phase 1: TUI scaffold | Run `go tool pprof` on a slow repo; check Update latency |
| tea.Cmd panic leaves terminal raw | Phase 1: TUI scaffold | Trigger deliberate panic in tea.Cmd; verify terminal restores |
| Signal handler accumulation | Phase 1: TUI scaffold | Run multiple programs in sequence; verify single Ctrl+C exits |
| Windows resize events | Phase 1: TUI scaffold | Test on Windows; implement polling workaround before Phase 2 |
| Column width with ANSI/Unicode | Phase 2: Diff renderer | Add Unicode fixture test cases; enforce runewidth throughout |
| ANSI color bleed across columns | Phase 2: Diff renderer | Add test: end mid-token in left column; check right column colors |
| Viewport ghost lines | Phase 2: Diff renderer | Rapid file-switch test; verify no rendering artifacts |
| CRLF in git output | Phase 3: Git integration | Add Windows-specific parse test using CRLF input fixtures |
| OSC 11 timeout/fallback | Phase 3: Theming | Test in tmux, test with no TTY; verify fallback to dark theme |
| Difftool signal handling | Phase 3: Difftool integration | Integration test: git difftool on 10+ changed files |
| CGO_ENABLED=0 explicit | Phase 4: CI/Distribution | CI step: `CGO_ENABLED=0 go build ./...`; verify on all targets |
| goreleaser fetch-depth | Phase 4: CI/Distribution | Check goreleaser changelog output; confirm recent commits appear |
| Windows VTP in cmd.exe | Phase 4: CI/Distribution | Add CI step that launches via cmd.exe; check ANSI rendering |

---

## Sources

- [Tips for building Bubble Tea programs (leg100.github.io)](https://leg100.github.io/en/posts/building-bubbletea-programs/)
- [Bubbletea issue #1477: Ghost lines in viewport](https://github.com/charmbracelet/bubbletea/issues/1477)
- [Bubbletea issue #1601: Windows terminal resize events not detected in v2](https://github.com/charmbracelet/bubbletea/issues/1601)
- [Bubbletea issue #282: View() called before WindowSizeMsg](https://github.com/charmbracelet/bubbletea/issues/282)
- [Bubbletea PR #412: Catch SIGTERM for cleanup](https://github.com/charmbracelet/bubbletea/pull/412)
- [Bubbletea PR #330: Race condition on repaint](https://github.com/charmbracelet/bubbletea/pull/330)
- [Bubbletea Concurrency and Goroutines (DeepWiki)](https://deepwiki.com/charmbracelet/bubbletea/5.1-concurrency-and-goroutines)
- [Commands in Bubble Tea (Charm blog)](https://charm.land/blog/commands-in-bubbletea/)
- [Loss of input in Charm's Bubbletea (dr-knz.net)](https://dr-knz.net/bubbletea-control-inversion.html)
- [Microsoft: Console Virtual Terminal Sequences](https://learn.microsoft.com/en-us/windows/console/console-virtual-terminal-sequences)
- [tmux issue #1919: OSC 11 support](https://github.com/tmux/tmux/issues/1919)
- [tmux issue #3582: OSC 11 caches response](https://github.com/tmux/tmux/issues/3582)
- [OSC 11 + SSH + Windows Terminal (zenn.dev)](https://zenn.dev/saitogo/articles/22bb1c5b8c7e70?locale=en)
- [Syntax highlighting misalignment with Cyrillic chars (anthropic/claude-code #23851)](https://github.com/anthropics/claude-code/issues/23851)
- [How to cross compile with CGO using GoReleaser and GitHub Actions (Bytebase)](https://www.bytebase.com/blog/how-to-cross-compile-with-cgo-use-goreleaser-and-github-action/)
- [Go issue #69709: os/exec encoding on Windows](https://github.com/golang/go/issues/69709)
- [termenv package (pkg.go.dev)](https://pkg.go.dev/github.com/muesli/termenv)
- [lipgloss package (pkg.go.dev)](https://pkg.go.dev/github.com/charmbracelet/lipgloss)
- [bubbletea viewport package (pkg.go.dev)](https://pkg.go.dev/github.com/charmbracelet/bubbles/viewport)

---
*Pitfalls research for: Go bubbletea TUI — alturd port from Python/Textual*
*Researched: 2026-06-25*
