# Phase 3: TUI Application - Context

**Gathered:** 2026-07-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Build the full bubbletea v2 interactive terminal UI — a split-screen application with a file tree pane (left) and a diff pane (right) — that wires together the `internal/diff` library (Phase 1) and `internal/git` layer (Phase 2) into a keyboard-driven interactive experience. Replaces the `os.Stdout` render path in `cmd/alturd/main.go` with a bubbletea TUI running in alternate screen mode.

Requirements in scope: DIFF-06, NAV-01, NAV-02, NAV-03, NAV-04, TREE-01, TREE-02, TREE-03, SEARCH-01.

Out of scope for Phase 3: TOML config, OSC 11 theming, difftool mode (DIFFTOOL-01/02/03), CI/release (DIST-01/02/03) — all Phase 4.

</domain>

<decisions>
## Implementation Decisions

### Layout and Pane Structure

- **D-01:** Two-pane split: file tree pane on the left, diff pane on the right, separated by a single Unicode vertical bar `│` (1 column). No colored border, no lipgloss border on panes.
- **D-02:** Tree pane width: 24 columns when unfocused, 45 columns when focused. **Instant resize — no animation, no tick-based transition.** ("Animated transition" in REQUIREMENTS.md is satisfied by bubbletea's render speed; no explicit animation state machine needed.)
- **D-03:** Diff pane always fills remaining terminal width: `diff_width = terminal_width - tree_width - 1` (separator column). No minimum diff pane width enforcement.
- **D-04:** Filename truncation in the tree pane uses ellipsis at end: `src/internal/diffren...` via `lipgloss.Style.MaxWidth()`. Applied when the pane is at 24-col unfocused width.
- **D-05:** A 1-row status bar at the top of the screen showing current file name and mode: `alturd — <filename> (<N> of <M> changed files)`. Mode indicator shows `[SEARCH]` when in-pane search is open; otherwise empty. Phase 4 adds difftool mode indicator.

### Startup and Initialization

- **D-06:** `main.go` runs `internal/git` and `diff.Parse()` **before** `tea.NewProgram()`. The model is initialized with the complete `[]*gitdiff.File` slice. No async loading, no `tea.Cmd` for data fetch, no loading spinner.
- **D-07:** The bubbletea model tracks a `ready bool` field. `View()` returns an empty string until the first `tea.WindowSizeMsg` sets `ready = true`. This prevents blank/zero-width pane rendering on the first frame.
- **D-08:** Windows resize polling workaround (bubbletea v2 issue #1601): **researcher investigates** the current recommended approach and implements it. Do not hard-code a polling interval until the issue is reviewed.

### File Tree

- **D-09:** "Compact-folder layout" means **GitHub-style path collapsing**: single-child directory chains are collapsed into one node. A path `src/internal/diff/` where each directory has only one child is shown as a single `src/internal/diff` entry in the tree.
- **D-10:** Directories appear before files at each level (dirs-first sort, per TREE-01). Collapsed directory chains are shown with a `▸` marker; expanded with `▾`. (Expand/collapse interaction TBD by planner.)
- **D-11:** Pressing `a` toggles between changed-files-only view and full-repo tree view. Full-repo tree is populated via `git ls-tree -r HEAD` (not filesystem walk). Changed files in full-repo view retain their `[A]/[M]/[D]/[R]` status markers; unchanged files have no marker.
- **D-12:** The currently-selected file in the tree is highlighted with an inverted background on the selected row (reverse video or equivalent background color).

### Search Mode (SEARCH-01)

- **D-13:** `/` opens a 1-row text input bar at the **bottom of the diff pane**. The diff viewport shrinks by 1 row when the search bar is open and restores when closed. Uses `bubbles/textinput` for the input field.
- **D-14:** While search is open: `n`/`N` navigates forward/backward between matches. When search is closed: `n`/`N` jumps between hunks (NAV-01). Key dispatch is mode-based — `Update()` checks `searchMode bool` to route `n`/`N`.
- **D-15:** `Esc` closes search: clears the query text, removes all match highlights from the diff pane, closes the input bar. Diff pane restores to full height. `n`/`N` immediately returns to hunk navigation.
- **D-16:** While search is open, `]`/`[` (file cycling) still works and closes search automatically before switching files. No search-query carry-over to the new file.

### Navigation Key Dispatch

- **D-17:** Key dispatch summary for `Update()`:
  - `searchMode = true`: `n`/`N` → match nav; `]`/`[` → cycle files + close search; `Esc` → close search; all other nav keys ignored.
  - `searchMode = false`: `n`/`N` → hunk nav; `]`/`[` → cycle files; `Tab` → toggle focused pane; `v` → toggle full-file/hunk-only; `q` → exit 0; `Q` → exit 1 (difftool abort, even if difftool mode not active in Phase 3).
- **D-18:** `q` exits with code 0 (NAV-04). `Q` exits with code 1 (NAV-04 difftool abort path). Both are wired in Phase 3 even though full difftool mode is Phase 4.

### Claude's Discretion

- Exact bubbletea model architecture (single top-level model vs. nested sub-models, message type design) — standard bubbletea patterns apply.
- ANSI-aware search scanner implementation details — researcher to determine the approach (ROADMAP success criteria calls out "ANSI-aware scanner").
- Exact hunk-centering behavior for `n`/`N` in full-file mode (NAV-01 says "centered with surrounding context") — 3 lines of context above/below is a reasonable default; researcher/planner to confirm against Python implementation.
- `bubbles/viewport` vs. custom viewport — use `bubbles/viewport` per CLAUDE.md stack guidance.
- Expand/collapse interaction for directory nodes in the tree — planner decides; not in REQUIREMENTS.md.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & Scope

- `.planning/REQUIREMENTS.md` §Navigation — NAV-01 through NAV-04; exact hunk/file navigation behavior
- `.planning/REQUIREMENTS.md` §File Tree — TREE-01 through TREE-03; compact-folder, dirs-first, status markers, width behavior
- `.planning/REQUIREMENTS.md` §Search — SEARCH-01; in-pane search with ANSI-aware match highlighting
- `.planning/REQUIREMENTS.md` §Diff Model — DIFF-06; v-key toggle full-file/hunk-only without reload
- `.planning/ROADMAP.md` §Phase 3 — Success criteria (5 items); "ANSI-aware scanner" requirement for search; blank-screen handling requirement

### Library Choices & Architecture

- `.claude/CLAUDE.md` §Technology Stack — bubbletea v2 (charm.land/bubbletea/v2), lipgloss v2, bubbles v2; viewport + textinput patterns
- `.claude/CLAUDE.md` §Stack Patterns — Two-viewport split pattern, focus/resize handling, search input overlay pattern, Windows resize polling workaround
- `.claude/CLAUDE.md` §Version Compatibility — bubbletea v2 / lipgloss v2 / bubbles v2 must use matching v2 imports; Windows terminal support table

### Phase 1 + 2 Integration Points

- `internal/diff/model.go` — `RowPair`, `RenderedLine`, `LineKind`, `RenderMode` types; full type system for the diff model
- `internal/diff/render.go` — `Render(file *gitdiff.File, width int) []string` — Phase 3 calls this with actual terminal column width (width/2-1 per column); feeds result into `viewport.SetContent(strings.Join(rows, "\n"))`
- `internal/diff/align.go` — produces `[]RowPair` with hunk boundary information; researcher to check if it surfaces hunk positions (NAV-01 needs to know where hunks start)
- `internal/git/runner.go` — `ExecRunner{}` is the real git runner; Phase 3 calls it in `main.go` before starting bubbletea
- `cmd/alturd/main.go` — current Phase 2 stdout render path; Phase 3 replaces the `run()` body with bubbletea startup after git+parse succeed

### External References

- bubbletea v2 issue #1601 — Windows resize polling workaround; researcher must check current recommended approach
- Python v1.1 implementation — behavioral reference for compact-folder layout, hunk-centering context, and status bar format

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- `diff.Render(file, width) []string` — already designed for viewport consumption: `viewport.SetContent(strings.Join(rows, "\n"))`. Phase 3 uses this directly.
- `diff.RenderMode` enum (`FullFile`/`HunkOnly`) — already wired to `Render()`; DIFF-06 toggle just changes which enum value is passed on re-render.
- `git.ParseRefArgs(args, argsLenAtDash)` — already handles all 6 invocation forms; Phase 3 reuses exactly.
- `git.ExecRunner{}` — real git subprocess runner; Phase 3 calls `ExecRunner{}.Run(gitArgs)` in `main.go` before starting bubbletea.
- `applog.Init()` — log initialization; Phase 3 keeps the same `logFile, err := applog.Init()` call at the top of `run()`.
- `cobra` root command in `cmd/alturd/main.go` — Phase 3 keeps cobra; `RunE` body changes to start bubbletea instead of stdout render.

### Established Patterns

- Table-driven tests in `testdata/` — continue for any pure functions in Phase 3 (tree builder, path collapsing, search match positions).
- `CGO_ENABLED=0` project-wide — bubbletea v2 and lipgloss v2 are pure Go; no new CGO dependencies.
- `go 1.25` module directive — already set; no change needed.
- `ExecRunner` stateless + dependency injection — Phase 3 injects `ExecRunner{}` the same way Phase 2 did; don't introduce singletons.

### Integration Points

- Phase 3 replaces the `for _, file := range files { ... fmt.Fprintln(os.Stdout, row) }` loop in `main.go` with `tea.NewProgram(newModel(files), tea.WithAltScreen()).Run()`.
- `diff.Render()` is re-called when terminal resizes (new `tea.WindowSizeMsg`) or when `v` toggles `RenderMode`. Both triggers produce a new `[]string` fed into `viewport.SetContent()`. No git re-execution needed.
- Phase 4 replaces hardcoded colors in `render.go` (`bgAdded`, `bgRemoved`, `bgModified`) with theme-configurable values. Phase 3 keeps the hardcoded constants.
- Phase 4 adds `install-difftool` cobra subcommand and `--config` flag to the same root command. Phase 3 does not add stubs for these.

</code_context>

<specifics>
## Specific Ideas

No specific "I want it like X" references — user accepted recommended options throughout. Open to standard bubbletea/lipgloss patterns within the decisions above.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 3-TUI Application*
*Context gathered: 2026-07-01*
