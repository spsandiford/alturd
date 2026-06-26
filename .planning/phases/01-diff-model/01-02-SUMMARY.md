---
phase: "01"
plan: "02"
subsystem: internal/diff
status: complete
tags: [parsing, alignment, go-gitdiff, tdd, diff-model]
completed: "2026-06-26"
duration: "~35 minutes"
tasks_completed: 2
tasks_total: 2

dependency_graph:
  requires:
    - 01-01 (go.mod with go-gitdiff, 13 fixture files, model.go types)
  provides:
    - internal/diff/parse.go — Parse(io.Reader) wrapping go-gitdiff
    - internal/diff/parse_test.go — table tests for all 13 fixtures
    - internal/diff/align.go — Align(file, mode) producing []RowPair
    - internal/diff/align_test.go — structural tests for all DIFF-07 edge cases
  affects:
    - 01-03-PLAN.md (highlight.go and render.go consume Align output)

tech_stack:
  added: []
  patterns:
    - TDD RED/GREEN with stub implementation to compile-and-fail
    - Parse wraps gitdiff.Parse(r) three-return form; discards preamble
    - Align early-exits for binary/mode-only/submodule before any line walking
    - Multi-delete+multi-add positional pairing with leftover one-sided rows (A1)
    - stripNewline removes trailing newline from Line.Line before storing Content
    - FileStatus helper returns [A]/[D]/[R]/[C]/[B]/[S]/[M] brackets

key_files:
  created:
    - internal/diff/parse.go
    - internal/diff/parse_test.go
    - internal/diff/align.go
    - internal/diff/align_test.go
  modified:
    - internal/diff/testdata/multi-file.diff (fixed @@ header count)
    - internal/diff/testdata/multi-hunk.diff (fixed both hunk header counts)
    - internal/diff/testdata/multiline-string.diff (fixed @@ header count)

decisions:
  - "FileStatus adds [S] for submodule instead of [M] to distinguish it from a regular modify"
  - "HunkOnly and FullFile produce identical output in Phase 1 because diff output is already hunk-local; the mode parameter is wired for Plan 03 viewport toggle"
  - "context_lines_populated test relaxed to allow empty Content on blank context lines — blank lines in the source become empty-string Content after stripNewline, which is correct behaviour"
  - "Submodule lines emitted as one-sided rows (LineRemoved/LineAdded) not context, because the old and new Subproject-commit lines are DIFFERENT values on each side"
---

# Phase 01 Plan 02: Parse and Align Pipeline Summary

**One-liner:** Parse wraps go-gitdiff with a no-panic contract; Align converts parsed Files into []RowPair with positional delete+add pairing and DIFF-07 placeholders for binary, mode-only, and submodule files.

## What Was Built

### Task 1: Parse wrapper with no-panic contract (TDD)

**RED commit:** 3ac1e3b — failing TestParse and TestParseMalformed written before implementation.

**GREEN commit:** 52ac5ed — `internal/diff/parse.go` implemented and three fixture files fixed.

`Parse(r io.Reader) ([]*gitdiff.File, error)` in package `diff`:
- Calls `gitdiff.Parse(r)` (three-return form), discards preamble
- Returns `fmt.Errorf("parsing diff: %w", err)` on error; never panics
- All 13 fixtures parse without error; `TestParseMalformed` confirms graceful error on adversarial input

`parse_test.go` (package `diff_test`):
- Table-driven `TestParse` covering all 13 fixtures with file count, name, IsNew/IsDelete/IsRename/IsBinary flags, submodule FileMode detection, NoEOL detection, and added/removed/context line counts
- `TestParseMalformed` feeds garbage and adversarial diff bytes; asserts no panic

### Task 2: Align into RowPairs with edge-case handling (TDD)

**RED commit:** 4f42ea8 — failing TestAlign with stub align.go returning nil.

**GREEN commit:** 892bec0 — full Align implementation.

`Align(file *gitdiff.File, mode RenderMode) []RowPair` in package `diff`:

Edge-case early-exits (DIFF-07):
- `file.IsBinary` → `[]RowPair{ {Left: "[Binary file changed — N bytes]", Right: LineBlank} }`
- `isModeOnly(file)` → `[]RowPair{ {Left: "[Mode changed: 0644 → 0755]", Right: LineBlank} }`
- `isSubmodule(file)` → raw lines as one-sided context rows; no Modified pairing

Normal text alignment:
- OpContext → RowPair with LineContext on both sides (content on both)
- Consecutive OpDelete + OpAdd run → collect all deletes, collect all adds, pair positionally: first delete+add as LineModifiedOld/LineModifiedNew, leftover deletes as LineRemoved+LineBlank, leftover adds as LineBlank+LineAdded
- Isolated OpAdd → LineBlank + LineAdded

Unexported helpers: `isSubmodule`, `isModeOnly`, `stripNewline`
Exported helpers: `FileStatus` (returns [A]/[D]/[R]/[C]/[B]/[S]/[M])

`align_test.go` (package `diff_test`):
- simple_modified_pair: asserts at least one LineModifiedOld+LineModifiedNew pair
- lone_delete_gets_blank_right: all rows from deleted-file.diff are LineRemoved+LineBlank
- lone_add_gets_blank_left: all rows from new-file.diff are LineBlank+LineAdded
- multi_delete_add_positional_pairing: multi-hunk.diff has both Modified pairs and standalone adds
- binary_placeholder: exactly 1 row containing "[Binary"
- mode_only_placeholder: exactly 1 row mentioning "Mode"
- submodule_raw_context_no_modified: no Modified kinds in submodule rows
- hunkonly_le_fullfile: HunkOnly ≤ FullFile row count on multi-hunk.diff
- no_panic_empty_fragments: rename.diff (no fragments) does not panic
- context_lines_populated: at least one non-blank LineContext with non-empty Content

## Verification Results

```
go test ./internal/diff/ -run 'TestParse|TestAlign' -v
--- PASS: TestParse (all 13 sub-tests)
--- PASS: TestParseMalformed
--- PASS: TestAlign (all 11 sub-tests)
go vet ./internal/diff/ → clean
go build ./... → clean
```

## Commits

| Task | Commit | Type | Description |
|------|--------|------|-------------|
| Task 1 RED | 3ac1e3b | test | add failing tests for Parse wrapper (RED) |
| Task 1 GREEN | 52ac5ed | feat | implement Parse wrapper with no-panic contract (GREEN) |
| Task 2 RED | 4f42ea8 | test | add failing tests for Align (RED) |
| Task 2 GREEN | 892bec0 | feat | implement Align with RowPair alignment and DIFF-07 edge cases (GREEN) |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Three fixture files had incorrect @@ header line counts**
- **Found during:** Task 1 GREEN — go-gitdiff returned parse errors for three fixtures
- **Issue:** Hand-crafted fixtures in Plan 01-01 had wrong line count values in `@@ -old,N +new,M @@` headers, causing go-gitdiff to miscount lines and fail
  - `multi-file.diff`: `@@ -1,5 +1,6 @@` should be `@@ -1,5 +1,8 @@` (4 adds, not 2)
  - `multi-hunk.diff` hunk 1: `@@ -1,12 +1,13 @@` should be `@@ -1,11 +1,12 @@` (bare blank lines = 9 context, not 10)
  - `multi-hunk.diff` hunk 2: `@@ -25,7 +26,8 @@` should be `@@ -25,7 +26,9 @@` (4 adds not 3)
  - `multiline-string.diff`: `@@ -1,18 +1,18 @@` should be `@@ -1,14 +1,14 @@` (bare blank lines not counted as 4 extra)
- **Fix:** Updated `@@` headers to match actual content
- **Files modified:** 3 fixture files in `internal/diff/testdata/`
- **Commit:** 52ac5ed

**2. [Rule 1 - Bug] context_lines_populated test was overly strict on blank content**
- **Found during:** Task 2 GREEN
- **Issue:** Test asserted `Content != ""` on all LineContext rows, but blank lines in the source file produce empty Content after `stripNewline("\n")` → `""` — which is correct
- **Fix:** Relaxed assertion to require at least one non-blank context line with non-empty Content
- **Files modified:** internal/diff/align_test.go
- **Commit:** 892bec0

**3. [Rule 2 - Missing Critical Functionality] Submodule lines emitted as one-sided rows not paired context**
- **Found during:** Task 2 implementation analysis
- **Issue:** Plan said "raw Subproject commit lines as context-kind RowPairs", but the old and new submodule SHA lines are DIFFERENT values — emitting both as context would show the same content on both sides, hiding the change
- **Fix:** Submodule delete lines → LineRemoved+LineBlank; add lines → LineBlank+LineAdded. This correctly shows the old SHA on the left and new SHA on the right without triggering Modified pairing (and thus intra-line diff). The test `submodule_raw_context_no_modified` verifies no Modified pairs appear.
- **Files modified:** internal/diff/align.go, internal/diff/align_test.go

## Known Stubs

None — both Parse and Align are fully implemented against the fixture corpus.

## Threat Flags

No new security-relevant surfaces beyond the plan's threat model. Parse wraps go-gitdiff and never panics on malformed input (T-01-03 mitigated). Align handles nil/empty TextFragments without panicking (T-01-04 mitigated). Submodule SHA lines pass through without intra-line pairing (T-01-05 mitigated).

## Self-Check: PASSED

| Check | Result |
|-------|--------|
| internal/diff/parse.go | FOUND |
| internal/diff/parse_test.go | FOUND |
| internal/diff/align.go | FOUND |
| internal/diff/align_test.go | FOUND |
| Commit 3ac1e3b (test RED parse) | FOUND |
| Commit 52ac5ed (feat GREEN parse) | FOUND |
| Commit 4f42ea8 (test RED align) | FOUND |
| Commit 892bec0 (feat GREEN align) | FOUND |
| go test TestParse all 13 sub-tests | PASS |
| go test TestParseMalformed | PASS |
| go test TestAlign all 11 sub-tests | PASS |
