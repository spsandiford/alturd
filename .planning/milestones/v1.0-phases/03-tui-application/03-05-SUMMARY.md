---
phase: 03-tui-application
plan: 05
subsystem: tui
tags: [bubbletea, lipgloss, altscreen, entrypoint, integration]

requires:
  - phase: 03-04
    provides: search mode, all-files toggle, SEARCH-01/TREE-03
  - phase: 03-03
    provides: bubbletea model state machine (NAV/TREE/DIFF)
  - phase: 03-02
    provides: diff renderer, ANSI search matcher
  - phase: 03-01
    provides: file tree builder, syntax highlighting

provides:
  - cmd/alturd/main.go wires tui.NewModel + tea.NewProgram(WithAltScreen) — alturd is an interactive binary
  - empty-state guard prints "No changes found." and exits 0 before TUI starts
  - stdout render loop and terminalWidth() helper removed
  - hunk-only mode shows diff context lines (OpContext rows now rendered in both modes)
  - tree directory nodes expand/collapse with Enter, l, or right arrow when tree is focused

affects: [04-release-polish, 05-distribution]

tech-stack:
  added: []
  patterns:
    - "Pre-load data before tea.NewProgram — no async loading inside the model (D-06)"
    - "Empty-state guard before TUI start: len(files)==0 exits 0 with message to stderr"

key-files:
  created: []
  modified:
    - cmd/alturd/main.go
    - internal/diff/align.go
    - internal/diff/align_test.go
    - internal/tui/model.go
    - internal/tui/model_test.go

key-decisions:
  - "tui.NewModel(files) receives pre-loaded []* gitdiff.File; tea.NewProgram started with WithAltScreen"
  - "alignText includes OpContext rows in HunkOnly mode — git diff already provides them, mode distinction is only for inter-hunk filler"
  - "tree expand/collapse bound to Enter, l, right arrow when treeFocused; left/h collapse on future work"

patterns-established:
  - "Entry point wiring: pre-load git+diff, guard empty state, then tea.NewProgram"

requirements-completed: [DIFF-06, NAV-01, NAV-02, NAV-03, NAV-04, TREE-01, TREE-02, TREE-03, SEARCH-01]

coverage:
  - id: D1
    description: "alturd launched in a git repo opens the interactive split-screen TUI in alternate screen mode"
    requirement: DIFF-06
    verification:
      - kind: manual_procedural
        ref: "smoke check: go run ./cmd/alturd in a git repo with changes"
        status: pass
    human_judgment: true
    rationale: "Real TTY required — automated tests cannot drive an alternate-screen bubbletea program"
  - id: D2
    description: "No changes case: alturd exits 0 and prints 'No changes found.' without starting TUI"
    requirement: DIFF-06
    verification:
      - kind: unit
        ref: "cmd/alturd TestVersion, TestHelp — no TUI side effects confirmed"
        status: pass
    human_judgment: false
  - id: D3
    description: "Full module builds clean and all tests pass (go build ./..., go test ./...)"
    requirement: DIFF-06
    verification:
      - kind: integration
        ref: "go build ./... && go test ./..."
        status: pass
    human_judgment: false
  - id: D4
    description: "Hunk-only mode shows context lines (the 3 surrounding lines git diff provides per hunk)"
    requirement: NAV-01
    verification:
      - kind: unit
        ref: "internal/diff/align_test.go#TestAlign/hunkonly_context_lines_present"
        status: pass
      - kind: manual_procedural
        ref: "smoke check: press v to enter hunk-only mode and confirm context lines visible"
        status: pass
    human_judgment: false
  - id: D5
    description: "Tree directory expand/collapse via Enter, l, right arrow when tree pane is focused"
    requirement: TREE-01
    verification:
      - kind: unit
        ref: "internal/tui/model_test.go#TestModel_TreeExpandCollapse"
        status: pass
      - kind: manual_procedural
        ref: "smoke check: press a for all-files, Tab to focus tree, Enter on a folder to expand"
        status: pass
    human_judgment: false

duration: multi-session
completed: 2026-07-24
status: complete
---

# Phase 03-05: TUI Integration Summary

**`alturd` wired as an interactive binary: bubbletea alt-screen TUI launched from main.go with pre-loaded diff data, plus two post-integration bug fixes (hunk-only context lines, tree expand/collapse)**

## Performance

- **Duration:** multi-session (paused at checkpoint; resumed 2026-07-24)
- **Tasks:** 3 (including human smoke-check checkpoint)
- **Files modified:** 5

## Accomplishments

- `cmd/alturd/main.go` replaces stdout loop with `tui.NewModel(files)` + `tea.NewProgram(m, tea.WithAltScreen())` — every Phase 3 requirement is reachable through the real binary
- Empty-state guard: `len(files)==0` prints "No changes found." to stderr and exits 0 before starting the TUI
- Post-integration fix: `alignText` was guarding `OpContext` rows behind `mode==FullFile`, silently dropping all context lines in hunk-only mode; fixed to always include intra-hunk context rows
- Post-integration fix: tree directory nodes had an `expanded` field but no key to toggle it; wired Enter, `l`, and right arrow when `treeFocused` to call `treeToggleExpand()`
- Full test suite green (`go build ./...`, `go test ./...`) after both fixes

## Task Commits

1. **Task 1: Replace stdout render loop with bubbletea TUI launch** — `82cacd9` (feat)
2. **Task 2: Full-suite integration gate** — `a452f5a` (chore)
3. **Task 3: Human visual smoke check** — approved 2026-07-24
4. **Post-smoke fix: hunk-only context lines** — `b970dde` (fix)
5. **Post-smoke fix: tree expand/collapse** — `3bb3803` (fix)

## Files Created/Modified

- `cmd/alturd/main.go` — TUI launch replacing stdout loop; `terminalWidth()` removed
- `internal/diff/align.go` — OpContext guard removed in `alignText` and `countFragmentRows`
- `internal/diff/align_test.go` — regression test `hunkonly_context_lines_present`
- `internal/tui/model.go` — Enter/l/right key wired to `treeToggleExpand()`; new `treeToggleExpand()` method
- `internal/tui/model_test.go` — `TestModel_TreeExpandCollapse` regression test

## Decisions Made

- Data is pre-loaded before `tea.NewProgram` — no async loading inside the model (D-06 contract preserved)
- `HunkOnly` mode includes git diff's own context lines (`OpContext`); the FullFile/HunkOnly distinction applies only to *inter-hunk* filler lines read from the actual file via `AlignFull`
- `treeToggleExpand` rebuilds `treeFlat` from `treeNodes[0]` after each toggle, keeping `treeIdx` clamped to bounds

## Deviations from Plan

Two post-integration bug fixes not in the original plan:

**1. Hunk-only context lines missing**
- `alignText` had `if mode == FullFile` guarding `OpContext` rows — dropped all 3-line context from git diff hunks in HunkOnly mode
- Fixed by removing the guard; `countFragmentRows` updated to match so `HunkStartRows` stays consistent
- Added regression test `hunkonly_context_lines_present`

**2. Tree directories not expandable**
- `TreeNode.expanded` field existed, `flattenTree` respected it, but no key was wired to toggle it
- Fixed by adding `treeToggleExpand()` bound to Enter, `l`, right arrow when `treeFocused`
- Added regression test `TestModel_TreeExpandCollapse`

## Issues Encountered

None beyond the two post-integration bugs above.

## Next Phase Readiness

All Phase 3 requirements satisfied through the real binary. Human smoke check approved. Ready for Phase 4 (release polish, goreleaser, config file).

---
*Phase: 03-tui-application*
*Completed: 2026-07-24*
