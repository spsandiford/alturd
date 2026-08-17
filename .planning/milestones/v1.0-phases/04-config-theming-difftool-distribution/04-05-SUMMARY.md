---
phase: 04-config-theming-difftool-distribution
plan: 05
subsystem: ui
tags: [bubbletea, lipgloss, ansi, tui, difftool, exit-codes]

# Dependency graph
requires:
  - phase: 04-03
    provides: DifftoolInfo, difftoolTitleBar(), difftool mode layout
  - phase: 04-04
    provides: install-difftool subcommand, difftool.trustExitCode=true gitconfig contract
provides:
  - "Difftool title bar appends U+2026 ellipsis when a filename overflows m.termWidth (DIFFTOOL-02)"
  - "View() clamps the separator-column repeat count so it cannot panic on a degenerate terminal height (CR-01)"
  - "The abort key ('Q') routes through tea.Quit so bubbletea restores the terminal before the process exits (CR-02)"
  - "tui.WasAborted(tea.Model) bool — exported accessor cmd/alturd uses to detect an abort across the package boundary"
  - "cmd/alturd/main.go reportError(err, w) int — testable exit-code router extracted from main()"
affects: [tui-title-bar, tui-render-path, cli-exit-codes]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "ansi.Truncate(s, width, tail) for ANSI/wide-character-aware truncation with an explicit tail — lipgloss.Style.MaxWidth cannot express a tail (internally hardcodes an empty one)"
    - "Height/width clamp-at-zero before strings.Repeat, mirroring internal/diff/render.go's existing minimum-width clamp"
    - "model.aborted bool + exported WasAborted(tea.Model) bool interface-assertion accessor for crossing the unexported-model package boundary"

key-files:
  created:
    - cmd/alturd/main_internal_test.go
  modified:
    - internal/tui/model.go
    - internal/tui/model_test.go
    - cmd/alturd/main.go

key-decisions:
  - "Used github.com/charmbracelet/x/ansi.Truncate directly instead of lipgloss.Style.MaxWidth for difftool title bar truncation — MaxWidth hardcodes an empty tail internally and has no way to express an ellipsis"
  - "Routed abort through tea.Quit + a boolean model field + errAborted sentinel, rather than calling os.Exit after manually restoring terminal escape sequences — lets bubbletea's own unwind do the restore and keeps the fix in the existing tea.Quit code shape ActionQuit already used"
  - "reportError's empty-Msg suppression keeps the abort path silent on stderr, exactly matching the pre-fix os.Exit(1) path's observable behavior (no new blank line)"
  - "Tree pane (renderTree, same MaxWidth-no-ellipsis pattern) deliberately left untouched — Phase 3 contract, tracked as debt, not part of DIFFTOOL-02's scope"

patterns-established:
  - "TDD RED phase for a pre-existing behavior bug: revert the fix locally (git stash), write the test, confirm it fails/panics/crashes against the unmodified source, commit test-only, then reapply the fix (git stash pop) and commit the fix separately"

requirements-completed: [DIFFTOOL-02]

coverage:
  - id: D1
    description: "Difftool title bar truncates an overflowing filename to exactly m.termWidth display cells with U+2026 as the final visible glyph; a fitting title stays byte-identical"
    requirement: "DIFFTOOL-02"
    verification:
      - kind: unit
        ref: "internal/tui/model_test.go#TestDifftoolTitleBarTruncatesWithEllipsis"
        status: pass
      - kind: unit
        ref: "internal/tui/model_test.go#TestDifftoolTitleBar (5 subtests, non-regression)"
        status: pass
    human_judgment: false
  - id: D2
    description: "View() cannot panic via a negative strings.Repeat count at any terminal height 0-3 (search open or closed); above the degenerate boundary the separator column is unchanged"
    verification:
      - kind: unit
        ref: "internal/tui/model_test.go#TestViewNoPanicOnShortTerminal (9 subtests)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Abort key ('Q') returns control to bubbletea via tea.Quit (terminal restored before exit) instead of calling os.Exit directly; quit and abort stay distinguishable; exit status 1 and silent stderr are preserved end-to-end"
    verification:
      - kind: unit
        ref: "internal/tui/model_test.go#TestAbortKeyQuitsWithoutProcessExit (3 subtests)"
        status: pass
      - kind: unit
        ref: "cmd/alturd/main_internal_test.go#TestReportError (3 subtests)"
        status: pass
    human_judgment: true
    rationale: "The terminal-restore behavior itself (raw mode off, alternate screen exited, shell prompt returns cleanly with no reset/stty needed) can only be observed in a real interactive terminal — the plan's own <verification> block routes this to a manual carry-forward item into the phase's re-verification pass, same as 04-VERIFICATION.md already did for the live difftool render."

# Metrics
duration: 6min
completed: 2026-08-04
status: complete
---

# Phase 04 Plan 05: Ellipsis Truncation + CR-01/CR-02 Fixes Summary

**Difftool title bar now appends a U+2026 ellipsis via ansi.Truncate (closing 04-VERIFICATION.md's one FAILED truth), View() clamps its separator-row count so it cannot panic on a degenerate terminal height, and the abort key routes through tea.Quit so bubbletea restores the terminal before the process exits with status 1.**

## Performance

- **Duration:** 6 min (first commit 20:19:21Z, last commit 20:24:08Z)
- **Started:** 2026-08-04T20:19:21Z
- **Completed:** 2026-08-04T20:24:08Z
- **Tasks:** 3 (each executed as a TDD RED → GREEN pair)
- **Files modified:** 4 (`internal/tui/model.go`, `internal/tui/model_test.go`, `cmd/alturd/main.go`, `cmd/alturd/main_internal_test.go` — new)

## Accomplishments

- Replaced `lipgloss.NewStyle().MaxWidth(m.termWidth).Render(title)` with `ansi.Truncate(title, m.termWidth, "…")` in `difftoolTitleBar()`, closing the single FAILED truth in 04-VERIFICATION.md and satisfying 04-UI-SPEC.md's "ellipsis appended" Copywriting Contract. Discovered as a side effect: `lipgloss.Style.MaxWidth(0)` does not truncate at all (treats 0 as "unset") — `ansi.Truncate` correctly returns an empty string at width 0, an incidental second-defect fix.
- Clamped `View()`'s `sepLines` (the pane-separator `strings.Repeat` count) and `handleResize`'s independent `contentH` at zero, so a 1-row terminal (or 2 rows with search open, or any transient degenerate resize report) can no longer panic the process from inside the render path — fixes code review CR-01.
- Routed the abort key (`config.ActionAbort`, default `Q`) through `tea.Quit` instead of calling `os.Exit(1)` directly inside `Update()`, so bubbletea restores the terminal (raw mode, alternate screen) before the process exits — fixes code review CR-02. `cmd/alturd/main.go`'s `run()` now captures the final model, detects the abort via the new `tui.WasAborted` accessor, and returns a silent `*git.ExitCodeError{Code: 1}` sentinel so `run()`'s deferred `logFile.Close()` finally executes on the abort path.

## Task Commits

Each task followed a TDD RED → GREEN pair (test-only commit, then fix commit):

1. **Task 1: Append an ellipsis when the difftool title bar overflows**
   - `4549e4b` — test(04-05): add failing test reproducing missing title-bar ellipsis (RED)
   - `b7740ae` — feat(04-05): append ellipsis to truncated difftool title bar (GREEN)
2. **Task 2: Clamp the separator-column row count so View() cannot panic**
   - `fab3fd0` — test(04-05): add failing test reproducing CR-01 View() panic (RED)
   - `bbfbb57` — fix(04-05): clamp separator/content height so View() cannot panic (CR-01) (GREEN)
3. **Task 3: Route the abort key through bubbletea's quit path**
   - `c3b0754` — test(04-05): add failing test reproducing CR-02 abort process-exit (RED)
   - `28184b7` — fix(04-05): route abort key through tea.Quit, restore terminal (CR-02) (GREEN)

**Plan metadata:** (this commit, following)

## Files Created/Modified

- `internal/tui/model.go` — `ansi.Truncate` title-bar truncation; `sepLines`/`contentH` height clamps; `model.aborted` field, `(model).Aborted()`, `WasAborted(tea.Model) bool`; `ActionAbort` now sets `m.aborted = true` and returns `(m, tea.Quit)` instead of calling `os.Exit(1)`
- `internal/tui/model_test.go` — `TestDifftoolTitleBarTruncatesWithEllipsis`, `TestViewNoPanicOnShortTerminal`, `TestAbortKeyQuitsWithoutProcessExit`
- `cmd/alturd/main.go` — `run()` captures `p.Run()`'s final model and returns `errAborted` on `tui.WasAborted`; `reportError(err, w) int` extracted from `main()` with empty-`Msg` suppression for the silent-abort sentinel
- `cmd/alturd/main_internal_test.go` (new) — `TestReportError` (3 rows: silent abort, `ExitCodeError` with message, plain error)

## Decisions Made

- `ansi.Truncate` (already a direct `go.mod` require, already imported by `internal/tui/search.go`) replaces `lipgloss.Style.MaxWidth` for the difftool title bar specifically — `MaxWidth` has no user-facing tail parameter (it calls `ansi.Truncate(line, maxWidth, "")` internally with a hardcoded empty tail per `charm.land/lipgloss/v2@v2.0.5/style.go:510`), so it structurally cannot append an ellipsis. The tree pane (`renderTree`) uses the identical no-ellipsis pattern and is deliberately left untouched — it's a Phase 3 contract (03-UI-SPEC.md D-04) outside this plan's DIFFTOOL-02 scope, and changing the truncated width by a glyph would shift the reverse-highlight geometry of the selected row. Tracked as debt in 04-05-PLAN.md's `<tracked_debt>` table.
- Abort is now a `model.aborted` boolean surfaced through an exported `WasAborted(tea.Model) bool` that type-asserts against a locally declared `interface{ Aborted() bool }` — the idiomatic way for `cmd/alturd` to observe state on the unexported `model` type across the package boundary.
- `reportError`'s empty-`Msg` suppression is what keeps the abort path silent on stderr — without it, the sentinel would print a stray blank line that the pre-fix `os.Exit(1)` path never produced.

## Deviations from Plan

None — plan executed exactly as written. All three fixes landed exactly where the plan specified (`internal/tui/model.go`, `internal/tui/model_test.go`, `cmd/alturd/main.go`, `cmd/alturd/main_internal_test.go`); no other file was touched; the tree pane and all seven `<tracked_debt>` items (WR-01 through WR-05, IN-01, IN-02) were left untouched exactly as scoped.

## Issues Encountered

- **Task 1 RED reproduction, incidental finding:** while confirming the RED test against unmodified source, `lipgloss.NewStyle().MaxWidth(0).Render(title)` at `handleResize(0, 50)` returned the full 532-character unrendered title (`lipgloss.Width(got) == 532`, not `0`) — `lipgloss`'s `MaxWidth` treats `0` as "no limit" rather than "truncate to nothing." `ansi.Truncate(title, 0, "…")` correctly returns an empty string. This is a second latent defect the new test happens to catch alongside the primary ellipsis gap; both are fixed by the same one-line change. Not filed separately since the fix and the test coverage are already in place.
- **TDD RED phase mechanics for a pre-existing-behavior bug (Tasks 1 and 2):** since the plan's Task action already specified the fix inline, writing a genuine failing test against *unmodified* source required temporarily reverting the source change (`git stash push -- internal/tui/model.go`), adding the test, confirming the failure/panic, committing test-only, then `git stash pop` to reapply the fix and commit it separately. Documented here so the RED→GREEN commit pairs are understood as intentional TDD discipline, not accidental extra commits.
- **Task 3 RED phase, compile dependency:** `TestAbortKeyQuitsWithoutProcessExit` references `WasAborted`, which doesn't exist in unmodified source — a genuine "test fails to compile against pre-fix code" scenario. Resolved by adding the `model.aborted` field, `Aborted()`, and `WasAborted` as pure scaffolding in the RED commit (no behavior change — `ActionAbort` still called `os.Exit(1)` directly in that commit), confirmed the test still kills the whole package test binary (only the first subtest's `RUN` line printed, no `PASS`/`FAIL`, and no later test in the file — `TestViewNoPanicOnShortTerminal`, `TestDifftoolTitleBarTruncatesWithEllipsis` — got a chance to run), then implemented the real `ActionAbort` behavior change in the GREEN commit.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All 45 must-have truths from 04-VERIFICATION.md's original scoring are now satisfied: the one FAILED truth (DIFFTOOL-02 ellipsis) is closed by this plan, and the phase's `43/45 must-haves verified` becomes fully verified pending the two remaining human-verification items below.
- **Carried forward for the phase's re-verification pass (per this plan's `<verification>` manual item, matching 04-VERIFICATION.md's existing human-verification protocol):** in a real interactive terminal, run `git difftool -t alturd <file>` on a repo with a filename longer than the terminal is wide and confirm the title bar ends in `…` on a single row; then press `Q` and confirm the shell prompt returns normally with no garbled output and no need for `reset`/`stty sane`, and that `echo $?` reports `1`. This sandbox has no real interactive TTY, so this remains unautomatable here exactly as 04-VERIFICATION.md already documented for the live difftool render (04-03-SUMMARY.md coverage item D8).
- Tracked debt carried forward unchanged from 04-REVIEW.md, per this plan's explicit scope decision: tree-pane ellipsis (03-UI-SPEC.md D-04), WR-01 (`refreshTreeContent` sets the wrong viewport's width), WR-02 (`diffW` unclamped — the width twin of CR-01), WR-03 (working-tree fallback path resolution), WR-04 (`DetectDarkBackground` abandoned-goroutine race, already FA-04-02), WR-05 (CI missing `-race`), IN-01 (`gofmt` drift, confirmed pre-existing and unchanged by this plan's edits), IN-02 (`install-difftool`'s PATH-relative command).
- Phase 04 is otherwise ready for milestone completion once the live-tag-push GitHub Actions release confirmation (04-VERIFICATION.md human-verification item 1) and the live interactive difftool render (item 2, restated above) are performed by a human with real infrastructure.

---
*Phase: 04-config-theming-difftool-distribution*
*Completed: 2026-08-04*

## Self-Check: PASSED

All 6 task commits (`4549e4b`, `b7740ae`, `fab3fd0`, `bbfbb57`, `c3b0754`, `28184b7`) confirmed present in `git log`. All 4 modified/created source files and this SUMMARY.md confirmed present on disk.
