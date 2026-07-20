---
plan: 03-01
phase: 03-tui-application
status: complete
completed: 2026-07-20
tasks_total: 2
tasks_completed: 2
key-files:
  created: []
  modified:
    - internal/diff/render.go
    - internal/diff/render_test.go
    - internal/diff/align.go
    - internal/diff/align_test.go
    - cmd/alturd/main.go
commits:
  - 995c570
  - f0a3568
---

# Plan 03-01: Diff Library Extensions for TUI — Summary

## What Was Built

Extended the `internal/diff` library with two capabilities the TUI needs before any bubbletea code exists:

1. **Mode-parameterized `Render()`** (DIFF-06): Changed signature from `Render(file, width)` to `Render(file, width, mode RenderMode)`. Updated all 5 call sites in `render_test.go` to pass `diff.FullFile` explicitly, plus the interim call in `cmd/alturd/main.go`. The TUI's 'v' key toggle can now request `FullFile` or `HunkOnly` mode without reparsing git data.

2. **`HunkStartRows()` + `countFragmentRows()`** (NAV-01): Appended to `align.go`. `HunkStartRows` returns ascending 0-based row indices for each TextFragment — the data source for 'n'/'N' hunk navigation in the TUI model. `countFragmentRows` mirrors `alignText`'s row-accounting loop exactly so viewport `SetYOffset` targets the correct line.

## Acceptance Criteria Verification

- `grep -q 'func Render(file \*gitdiff.File, width int, mode RenderMode) \[\]string' internal/diff/render.go` ✓
- `grep -q 'Align(file, mode)' internal/diff/render.go` ✓
- No remaining 2-arg `diff.Render` calls ✓
- `go build ./...` exits 0 ✓
- `go test ./internal/diff/... -run TestRender` passes ✓
- `func HunkStartRows` and `func countFragmentRows` in `align.go` ✓
- `go test ./internal/diff/... -run TestHunkStartRows` — all 4 subtests pass ✓
- Full diff test suite (`go test ./internal/diff/...`) green ✓

## Deviations

None. Implementation followed PATTERNS.md reference verbatim.

## Self-Check: PASSED
