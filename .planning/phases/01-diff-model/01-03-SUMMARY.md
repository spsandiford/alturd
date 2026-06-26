---
phase: "01"
plan: "03"
subsystem: internal/diff
status: complete
tags: [chroma, syntax-highlighting, ansi-rendering, intra-line-diff, tdd, diff-model]
completed: "2026-06-26"
duration: "~7 minutes"
tasks_completed: 2
tasks_total: 2

dependency_graph:
  requires:
    - 01-01 (go.mod, model.go types, 13 fixture files)
    - 01-02 (parse.go, align.go — Align produces RowPairs consumed here)
  provides:
    - internal/diff/highlight.go — Highlight(pairs, filename) populates ANSI fields via Chroma
    - internal/diff/highlight_test.go — TestHighlight: ANSI presence + placeholder passthrough
    - internal/diff/render.go — Render(file, width) public []string API
    - internal/diff/render_test.go — TestRender: structure, colors, guards, no-panic corpus
  affects:
    - Phase 3 (TUI): Render is the direct feed into bubbletea viewport (D-03)
    - Phase 4 (config): style/formatter selection hooks are pre-wired with comments

tech_stack:
  added:
    - github.com/alecthomas/chroma/v2 v2.27.0 promoted to direct dependency (was indirect)
    - github.com/bluekeyes/go-gitdiff v0.8.1 promoted to direct
    - github.com/sergi/go-diff v1.4.0 promoted to direct
  patterns:
    - Tokenise-once-per-side strategy (not per render frame) via content assembly + split
    - splitAndReset() appends ansiReset to each split line after chroma output (Pitfall 2 guard)
    - isPlaceholderPairs() detects binary/mode-only/submodule and bypasses chroma
    - package-level var dmp = diffmatchpatch.New() (single instance, not per-call)
    - shouldSkipIntraLine(): len > 1000 char OR tokens > 200 (D-07 pre-guards, T-01-06)
    - computeIntraLineWithTimeout(): 100ms goroutine + time.After channel (D-07)
    - joinColumns(): ansiReset at left/right boundary (Pitfall 1, T-01-08)
    - DiffMain called with checklines=false (D-06, Pitfall 3)
    - TDD RED/GREEN cycle: render_test.go committed first (RED), render.go second (GREEN)

key_files:
  created:
    - internal/diff/highlight.go
    - internal/diff/highlight_test.go
    - internal/diff/render.go
    - internal/diff/render_test.go
  modified:
    - go.mod (direct dep promotion via go mod tidy)
    - go.sum (additional transitive dep checksums)

decisions:
  - "ansiReset const defined in highlight.go (not render.go) since both files use it; render.go references the package-level const without redeclaring"
  - "isPlaceholderPairs detects binary/mode-only via content prefix '[' and submodule via 'Subproject commit' prefix; avoids passing *gitdiff.File into Highlight"
  - "intra-line span markers use bold (\x1b[1m/\x1b[22m) for Phase 1; Phase 4 can switch to brighter background"
  - "For Modified rows, render uses raw Content for intra-line diff spans (not chroma ANSI); chroma ANSI is used for non-modified rows only — clean for Phase 1, composable in Phase 4"
  - "go mod tidy run after implementation promotes three libs from indirect to direct; transitive test deps of chroma/v2 added to go.sum"
  - "TDD RED commit (e1d0257) + GREEN commit (918326a) satisfy the plan's tdd=true contract"
---

# Phase 01 Plan 03: Highlight and Render Pipeline Summary

**One-liner:** Chroma syntax highlighting applied once per side with per-line reset guard; Render composes []string side-by-side rows with 256-colour diff backgrounds, intra-line bold spans on Modified rows, and ANSI reset at every column boundary.

## What Was Built

### Task 1: Chroma syntax highlighting per line (DIFF-02)

**Commit:** 145c453 — `feat(01-03): implement Chroma syntax highlighting per line`

`internal/diff/highlight.go` (`package diff`):
- `const ansiReset = "\x1b[0m"` — package-level, shared with render.go
- `func Highlight(pairs []RowPair, filename string) error` — populates `RenderedLine.ANSI` in-place
- `isPlaceholderPairs()` / `isPlaceholderSide()` — detect binary/mode-only (prefix "[") and submodule (prefix "Subproject commit") → bypass chroma, copy Content → ANSI unchanged
- Tokenise-once-per-side: left content assembled as `strings.Join`, tokenised once, split back per line
- `splitAndReset()` — appends `\x1b[0m` to every split line that lacks a trailing reset (RESEARCH Pitfall 2)
- Phase 4 note in comments: style ("monokai") and formatter ("terminal16m") are hardcoded here

`internal/diff/highlight_test.go` (`package diff_test`):
- `var ansiColorRe = regexp.MustCompile("\x1b\\[[0-9;]+m")` — package-level, reused in render_test.go
- `TestHighlight` with 6 sub-tests: ANSI presence on README.md, per-line reset on multiline-string.diff, placeholder passthrough for binary/mode-only/submodule, empty-pairs no-error

### Task 2: Render side-by-side rows (TDD — RED+GREEN)

**RED commit:** e1d0257 — `test(01-03): add failing tests for Render`
**GREEN commit:** 918326a — `feat(01-03): implement Render side-by-side output`

`internal/diff/render.go` (`package diff`):
- `var dmp = diffmatchpatch.New()` — package-level, not per-call
- `func Render(file *gitdiff.File, width int) []string` — public phase API (D-03/D-04)
- `lineBg()` — maps LineKind → 256-colour background (`\x1b[48;5;Nm`)
- `renderSide()` — applies line bg over chroma ANSI; falls back to raw Content
- `applyIntraLine()` — DiffMain(old, new, false) + span markers for Modified rows; falls back to renderSide on any guard trip
- `shouldSkipIntraLine()` — len > 1000 char OR Fields > 200 token (D-07, T-01-06)
- `computeIntraLineWithTimeout()` — 100ms deadline via goroutine + time.After (D-07)
- `joinColumns()` — `left + ansiReset + " " + right` (Pitfall 1, T-01-08)
- `countTokens()` — `strings.Fields` whitespace proxy

`internal/diff/render_test.go` (`package diff_test`):
- `const testWidth = 160`, `ansiResetStr`, `intraLineRe = regexp.MustCompile("\x1b\\[1m")`
- `TestRender` with 8 sub-tests: non-empty output, ANSI presence, column reset boundary, intra-line markers on simple.diff, guard suppression for large-line.diff and many-tokens.diff, binary no-panic, no-panic across all 13 fixtures, width independence

**go mod tidy commit:** 0a75d60 — promotes three Phase 1 libraries from `// indirect` to direct

## Verification Results

```
go test ./internal/diff/... — PASS (all 37 tests across TestParse/TestParseMalformed/TestAlign/TestHighlight/TestRender)
go vet ./internal/diff/ — clean
go build ./... — clean
```

Guard-path verification:
- large-line.diff: `len > 1000` → `shouldSkipIntraLine` returns true → no `\x1b[1m` in output ✓
- many-tokens.diff: `countTokens > 200` (210 tokens) → same guard → no span markers ✓
- simple.diff Modified row: guard passes → DiffMain fires → `\x1b[1m` present ✓

Pitfall mitigations verified:
- Pitfall 1 (column colour bleed): `strings.Contains(row, "\x1b[0m")` true for every row ✓
- Pitfall 2 (multi-line token bleed): every ANSI field ends with `\x1b[0m` ✓
- Pitfall 3 (checklines): `dmp.DiffMain(old, new, false)` — hard-coded false ✓

## Commits

| Task | Commit | Type | Description |
|------|--------|------|-------------|
| Task 1 | 145c453 | feat | Chroma syntax highlighting per line (DIFF-02) |
| Task 2 RED | e1d0257 | test | Failing tests for Render (RED) |
| Task 2 GREEN | 918326a | feat | Render side-by-side output with diff colors (GREEN) |
| go mod tidy | 0a75d60 | chore | Promote direct deps; add transitive sums |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking Issue] `const ansiReset` must be defined in one file**
- **Found during:** Task 1 implementation — both highlight.go (used in splitAndReset) and render.go (used in joinColumns) need ansiReset
- **Issue:** Plan specified `const ansiReset = "\x1b[0m"` as a render.go artifact, but highlight.go also requires it for per-line reset. Defining in two files in the same package causes a compile error (duplicate const).
- **Fix:** Defined `ansiReset` in highlight.go (written first in Task 1). render.go references it without redeclaration.
- **Impact:** Minor package organisation difference from plan; no functional change.

**2. [Rule 2 - Missing Critical Functionality] go mod tidy after source implementation**
- **Found during:** After Task 2 GREEN
- **Issue:** Prior plans used `go mod download` (no source files to resolve direct deps). Now that source files import the three Phase 1 libraries directly, `// indirect` markers are incorrect and future `go mod tidy` by contributors would cause churn.
- **Fix:** Ran `go mod tidy` to promote direct deps and populate transitive sums.
- **Files modified:** go.mod, go.sum
- **Commit:** 0a75d60

### Non-Deviations (Plan-Aligned)

- Highlight uses fallback lexer for unrecognised filenames (as specified)
- intra-line spans use bold (`\x1b[1m`) for Phase 1; Phase 4 note added in code comment
- For Modified rows, render uses raw Content for intra-line span computation (not chroma ANSI); this is clean for Phase 1 and composable in Phase 4 when lipgloss ANSI-aware operations are available

## TDD Gate Compliance

| Gate | Commit | Type | Status |
|------|--------|------|--------|
| RED | e1d0257 | test | PRESENT — fails with `diff.Render undefined` |
| GREEN | 918326a | feat | PRESENT — all TestRender sub-tests pass |
| REFACTOR | — | — | Not needed; implementation clean |

## Known Stubs

None — Highlight and Render are fully implemented against the 13-fixture corpus. Phase 4 style/formatter selection is documented via code comments but the current hardcoded values ("monokai", "terminal16m") are functionally complete for Phase 1.

## Threat Flags

No new security-relevant surfaces beyond the plan's threat model:
- T-01-06 (DiffMain DoS): mitigated by shouldSkipIntraLine + computeIntraLineWithTimeout ✓
- T-01-07 (binary bytes as text): mitigated by isPlaceholderPairs early-exit in Highlight + Align early-exit ✓
- T-01-08 (chroma column bleed): mitigated by ansiReset at column boundary in joinColumns ✓

## Self-Check: PASSED

| Check | Result |
|-------|--------|
| internal/diff/highlight.go | FOUND |
| internal/diff/highlight_test.go | FOUND |
| internal/diff/render.go | FOUND |
| internal/diff/render_test.go | FOUND |
| Commit 145c453 (feat highlight) | FOUND |
| Commit e1d0257 (test render RED) | FOUND |
| Commit 918326a (feat render GREEN) | FOUND |
| Commit 0a75d60 (chore go mod tidy) | FOUND |
| go test ./internal/diff/... | PASS |
| go vet ./internal/diff/ | clean |
