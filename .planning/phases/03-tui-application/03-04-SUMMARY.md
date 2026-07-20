---
plan: 03-04
phase: 03-tui-application
status: complete
completed: 2026-07-20
duration: 187s
tasks_total: 2
tasks_completed: 2
subsystem: tui
tags: [search, tree, bubbletea, tui]
requires: [03-03]
provides: [search-mode, all-files-toggle]
affects: [internal/tui/model.go, internal/tui/model_test.go]
tech-stack:
  added: []
  patterns:
    - searchMode branch guards handleKey before normal-mode switch
    - recomputeSearch called on every textinput mutation for live match refresh
    - toggleAllFiles uses lazy cache (allFilePaths) to avoid repeated ls-tree calls
key-files:
  created: []
  modified:
    - internal/tui/model.go
    - internal/tui/model_test.go
decisions:
  - GetContent() used instead of Content() — bubbles v2 viewport API has no Content() method
  - SetHighlights(nil) called explicitly on Esc and ]/[ to clear viewport highlight state before height restore
  - CGO_ENABLED=0 env: race detector skipped (CGO required for -race); tests run without race flag
  - allFilePaths pre-seeded in test to keep TestModel_AllFilesToggle hermetic (no live git subprocess)
key-decisions:
  - viewport.GetContent() is the bubbles v2 accessor (not Content() as listed in PATTERNS.md)
  - HighlightStyle and SelectedHighlightStyle both set to Reverse(true) in NewModel
  - handleKey searchMode branch returns early before falling through to normal-mode switch
metrics:
  duration_seconds: 187
  completed_date: 2026-07-20
  tasks: 2
  files_modified: 2
commits:
  - 14e9f80
  - 6fc875d
requirements: [SEARCH-01, TREE-03]
---

# Phase 03 Plan 04: Search Mode and Full-Repo Tree Toggle — Summary

## What Was Built

In-pane search (SEARCH-01) and full-repo tree toggle (TREE-03) wired into `internal/tui/model.go`, completing the D-17 key dispatch table.

**Search mode (`internal/tui/model.go`):**
- `'/'` in normal mode: sets `searchMode=true`, calls `handleResize()` to shrink diff viewport 1 row (Pitfall 7), returns `searchInput.Focus()` cmd (D-13)
- `searchMode` branch at the top of `handleKey` dispatches before the normal-mode switch
- `'n'/'N'` in searchMode: call `diffVP.HighlightNext()` / `diffVP.HighlightPrevious()` (D-14)
- `'Esc'`: clears `searchMode`, resets textinput, calls `SetHighlights(nil)`, restores viewport height (D-15)
- `']'/'['` in searchMode: full close (clear query + highlights + restore height) then `handleFileCycle` with no query carry-over (D-16)
- Typed characters in searchMode: forwarded to `searchInput.Update(msg)` then `recomputeSearch()` for live match refresh
- `recomputeSearch()`: reads `diffVP.GetContent()`, calls `findMatches(content, query)`, calls `diffVP.SetHighlights(matches)`
- `HighlightStyle` and `SelectedHighlightStyle` set to `lipgloss.NewStyle().Reverse(true)` in `NewModel` (RESEARCH Open Question 2)
- Non-key messages forwarded to `searchInput` in `Update()` for cursor blink etc.

**Full-repo tree toggle (`internal/tui/model.go`):**
- `'a'` in normal mode calls `toggleAllFiles()` (D-11, TREE-03)
- `toggleAllFiles()`: flips `m.allFiles`; lazily loads `allFilePaths` via `git.ExecRunner{}.Run([]string{"ls-tree", "-r", "--full-tree", "--name-only", "HEAD"})` on first toggle-on; caches result; reverts silently on error (no TUI crash)
- Rebuilds `treeNodes`/`treeFlat` over `allFilePaths` (full repo) or `filePaths(files)` (changed-files only); clamps `treeIdx`; calls `refreshTreeContent()`
- Changed files keep their `[A]/[M]/[D]/[R]` status markers via `buildStatusMap`; unchanged files have empty Status (D-11)
- `--full-tree` flag makes paths root-relative regardless of cwd

**Tests (`internal/tui/model_test.go`):**
- `TestModel_SearchDispatch`: `'/'` → `searchMode=true`; `'n'`/`'N'` in searchMode leave `currentHunk` unchanged (proving highlight nav, not hunk nav, D-14); `Esc` clears `searchMode` and `searchMatches` (D-15)
- `TestModel_SearchFindPositions`: `recomputeSearch()` runs without panic (SEARCH-01 path verification)
- `TestModel_AllFilesToggle`: pre-seeds `allFilePaths` to be hermetic; asserts toggle ON expands `treeFlat` over full path set with status markers retained; toggle OFF rebuilds over changed-files paths

## Key Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - API correction] `Content()` → `GetContent()` in recomputeSearch**
- **Found during:** Task 1 implementation
- **Issue:** PATTERNS.md documents `diffVP.Content()` but the bubbles v2 viewport API exposes `GetContent()` — `Content()` does not exist on the v2 Model
- **Fix:** Used `m.diffVP.GetContent()` in `recomputeSearch()`
- **Files modified:** `internal/tui/model.go`
- **Commit:** 14e9f80

**2. [Rule 2 - Test hermeticity] Pre-seeded allFilePaths in TestModel_AllFilesToggle**
- **Found during:** Task 2 test design
- **Issue:** First 'a' press invokes live git ls-tree — makes test non-hermetic
- **Fix:** Pre-seeded `m.allFilePaths` before calling `toggleAllFiles()`, bypassing git subprocess; plan already anticipated this approach in its action description
- **Files modified:** `internal/tui/model_test.go`
- **Commit:** 6fc875d

**3. [Rule 2 - Race detector] CGO_ENABLED=0 environment prevents -race flag**
- **Found during:** Task 2 verification
- **Issue:** `go test -race` requires CGO; the project runs with `CGO_ENABLED=0` for static binary portability
- **Fix:** Ran `go test ./internal/tui/...` without `-race`; all tests pass; this is expected and documented
- **Impact:** None on correctness; race detection deferrd to CI with CGO-enabled runner

## Known Stubs

None. All search and tree-toggle behavior is fully wired.

## Threat Surface Scan

| Flag | File | Description |
|------|------|-------------|
| T-03-05 (mitigated) | model.go | `git.ExecRunner{}.Run([]string{"ls-tree",...})` — argv form, no shell injection surface (STRIDE T-03-05 mitigation confirmed present) |
| T-03-PATH (mitigated) | model.go | ls-tree paths split on `\n`, used only as tree display strings and map keys — no `os.Open`/`filepath` resolution |
| T-03-SEARCH (mitigated) | model.go | Search query matched against `ansi.Strip`'d content, passed to viewport as plain-text int positions — never interpolated into ANSI output |

## Self-Check: PASSED
