---
phase: 01-diff-model
verified: 2026-06-26T12:00:00Z
status: passed
score: 3/4 roadmap success criteria verified
behavior_unverified: 0
overrides_applied: 0
deferred:
  - truth: "Full-file mode and hunk-only mode each produce correct, independently testable output — full-file includes all unchanged lines; hunk-only includes only changed hunks"
    addressed_in: "Phase 3"
    evidence: "Phase 3 requirement DIFF-06: 'User can toggle between full-file and hunk-only view with v hotkey without reload'; Phase 3 SC #4: 'v toggles full-file/hunk-only view without reloading git data'. The library's HunkOnly behavioral distinction requires full-file access only available in Phase 3. align.go comment: 'HunkOnly: same as FullFile for Phase 1 ... the distinction matters in Phase 3 when the original file is available.'"
---

# Phase 1: Diff Model Verification Report

**Phase Goal:** Build the internal diff pipeline — parse, align, highlight, and render git diffs into side-by-side ANSI output
**Verified:** 2026-06-26
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| SC1 | `go test ./internal/diff/...` passes all table tests against 12+ Python fixture scenarios (binary patches, renames, mode-only changes, submodule bumps, no-newline-at-EOF) | ✓ VERIFIED | 37 tests pass: TestParse (14 subtests covering all 13 fixtures + malformed), TestAlign (11 subtests), TestHighlight (6 subtests), TestRender (8 subtests + 13 sub-subtests). `go vet ./internal/diff/` clean. |
| SC2 | A two-file diff produces side-by-side columns with Chroma syntax highlighting applied where a language can be detected; ANSI resets at every left-column boundary prevent color bleed | ✓ VERIFIED | `multi-file.diff` parses to 2 `*gitdiff.File` structs. `joinColumns()` in render.go appends `ansiReset` at left/right boundary. TestRender/rows_contain_ansi_codes and TestRender/reset_at_column_boundary both pass. TestHighlight/detectable_language_has_ansi passes. |
| SC3 | Modified lines carry intra-line character-level change markers produced by the LCS pass, subject to the 1000-char/200-token/100ms guards | ✓ VERIFIED | `applyIntraLine()` calls `dmp.DiffMain(old, new, false)` (checklines=false). `shouldSkipIntraLine()` checks `len > 1000 OR countTokens > 200`. `computeIntraLineWithTimeout()` has 100ms goroutine deadline. TestRender/intra_line_markers_on_modified_row passes. TestRender/large_line_guard_no_intra_markers and TestRender/many_tokens_guard_no_intra_markers both confirm guards suppress markers. |
| SC4 | Full-file mode and hunk-only mode each produce correct, independently testable output for the same input — full-file includes all unchanged lines; hunk-only includes only changed hunks | DEFERRED | Both modes produce identical output in Phase 1. align.go comment: "HunkOnly: same as FullFile for Phase 1; the distinction matters in Phase 3 when the original file is available. The mode parameter is wired here for forward compatibility." Plan 02 decision: "The mode parameter is wired for Plan 03 viewport toggle." Deferred to Phase 3/DIFF-06. See Deferred Items table. |

**Score:** 3/4 roadmap success criteria verified (1 deferred to Phase 3)

### Deferred Items

Items not yet met but explicitly addressed in later milestone phases.

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | Full-file/hunk-only behavioral distinction (SC4) | Phase 3 | DIFF-06 in Phase 3 requirements: "User can toggle between full-file and hunk-only view with `v` hotkey without reload." Phase 3 SC #4 explicitly includes the toggle. The library's HunkOnly must produce fewer rows than FullFile for the toggle to work; this requires full-file content available in Phase 3. |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `go.mod` | Module `github.com/alturd/alturd`, `go 1.25`, 3 locked deps | ✓ VERIFIED | go-gitdiff v0.8.1, go-diff v1.4.0, chroma/v2 v2.27.0 present. No Phase 3/4 deps. |
| `go.sum` | Checksums for all deps | ✓ VERIFIED | Exists with checksums for all 4 modules (3 direct + 1 transitive). |
| `.gitattributes` | `internal/diff/testdata/*.diff text eol=lf` | ✓ VERIFIED | Exact rule present; prevents CRLF on Windows checkout. |
| `internal/diff/model.go` | LineKind, RenderMode, RenderedLine, RowPair | ✓ VERIFIED | All 4 types exported. LineKind constants in correct iota order. FullFile=iota 0 (default). No imports. |
| `internal/diff/testdata/*.diff` | 13 scenario fixtures, LF-only | ✓ VERIFIED | Exactly 13 files. No CRLF bytes. large-line.diff has 1121-char changed lines (>1000). many-tokens.diff has 212-token line (>200). submodule.diff has mode 160000 and Subproject commit lines. mode-only.diff has 0 @@ hunks. multi-file.diff has 2 `diff --git` headers. |
| `internal/diff/parse.go` | `Parse(io.Reader) ([]*gitdiff.File, error)` | ✓ VERIFIED | Wraps `gitdiff.Parse`, discards preamble, wraps error with `%w`. Never panics. |
| `internal/diff/parse_test.go` | Table test covering all 13 fixtures + malformed | ✓ VERIFIED | `TestParse` covers all 13 fixtures with file count, NewName, status flags, submodule mode, NoEOL. `TestParseMalformed` confirms no panic on garbage input. |
| `internal/diff/align.go` | `Align(*gitdiff.File, RenderMode) []RowPair` | ✓ VERIFIED | Binary/mode-only/submodule early-exit edge cases. Positional delete+add pairing. FullFile/HunkOnly wired (distinction deferred). |
| `internal/diff/align_test.go` | Structural RowPair assertions across fixtures | ✓ VERIFIED | 11 subtests covering Modified pairs, lone-sided rows, binary/mode-only/submodule placeholders, HunkOnly≤FullFile, no-panic on empty fragments. |
| `internal/diff/highlight.go` | `Highlight([]RowPair, string) error` | ✓ VERIFIED | Chroma monokai/terminal16m. Tokenise-once-per-side. `splitAndReset()` appends `\x1b[0m` per line. Placeholder detection bypasses chroma. |
| `internal/diff/highlight_test.go` | ANSI presence and placeholder passthrough | ✓ VERIFIED | 6 subtests: ANSI codes on detectable language, per-line reset on multiline-string, placeholder passthrough for binary/mode-only/submodule. |
| `internal/diff/render.go` | `Render(*gitdiff.File, int) []string` | ✓ VERIFIED | Package-level `dmp`. `lineBg()` maps LineKind to 256-colour backgrounds. `joinColumns()` inserts `ansiReset` at boundary. `applyIntraLine()` with guards. Width parameter wired (column-width truncation deferred to Phase 3 via lipgloss per plan). |
| `internal/diff/render_test.go` | Structural render assertions + guard tests | ✓ VERIFIED | 8 subtests + 13 no-panic sub-subtests. Guards verified for large-line and many-tokens. Column-boundary reset verified. Width-independence verified. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `parse.go` | `go-gitdiff/gitdiff` | `gitdiff.Parse(r)` call | ✓ WIRED | Direct import `github.com/bluekeyes/go-gitdiff/gitdiff`; Parse calls `gitdiff.Parse`. |
| `align.go` | `parse.go` | `*gitdiff.File` as input type | ✓ WIRED | `Align(file *gitdiff.File, mode RenderMode)` consumes Parse output. |
| `highlight.go` | `chroma/v2` | `lexers.Match`, `styles.Get`, `formatters.Get` | ✓ WIRED | All three chroma sub-packages imported and called. `Tokenise` + `Format` populate `RenderedLine.ANSI`. |
| `render.go` | `align.go` | `Align(file, FullFile)` | ✓ WIRED | Render calls Align with FullFile mode as default (DIFF-05 default). |
| `render.go` | `highlight.go` | `Highlight(pairs, file.NewName)` | ✓ WIRED | Highlight called between Align and row composition. |
| `render.go` | `go-diff/diffmatchpatch` | `dmp.DiffMain(old, new, false)` | ✓ WIRED | Package-level `var dmp = diffmatchpatch.New()`. DiffMain called in `computeIntraLineWithTimeout`. |
| `joinColumns()` | `ansiReset` const | `left + ansiReset + " " + right` | ✓ WIRED | ANSI reset inserted at left/right boundary. `ansiReset` defined in `highlight.go`, referenced in `render.go` (same package). |
| `.gitattributes` | `testdata/*.diff` | `text eol=lf` rule | ✓ WIRED | Rule guarantees LF on Windows checkout. |

### Data-Flow Trace (Level 4)

Not applicable — `internal/diff` is a pure library producing `[]string` (no dynamic data fetch; inputs are `io.Reader` / `*gitdiff.File`). All data flows from the caller (test or Phase 2 git subprocess) through the pipeline: Parse → Align → Highlight → Render.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All 37 tests pass | `go test ./internal/diff/... -v` | 37/37 PASS in 0.109s | ✓ PASS |
| Build clean | `go build ./...` | exit 0 | ✓ PASS |
| Vet clean | `go vet ./internal/diff/` | exit 0 | ✓ PASS |
| large-line guard: `len > 1000` suppresses intra-line | TestRender/large_line_guard_no_intra_markers | PASS — no `\x1b[1m` in output | ✓ PASS |
| many-tokens guard: `countTokens > 200` suppresses intra-line | TestRender/many_tokens_guard_no_intra_markers | PASS — no `\x1b[1m` in output | ✓ PASS |
| Modified row gets intra-line markers | TestRender/intra_line_markers_on_modified_row | PASS — `\x1b[1m` present on simple.diff Modified row | ✓ PASS |
| ANSI reset at column boundary | TestRender/reset_at_column_boundary | PASS — every row contains `\x1b[0m` | ✓ PASS |

### Probe Execution

Not applicable — no declared probes in PLAN files.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| DIFF-01 | 01-01, 01-03 | User sees old and new content in aligned parallel side-by-side columns | ✓ SATISFIED | `RowPair{Left, Right}` is the column model. `Render()` produces `[]string` with `joinColumns(left, right)` composition. TestRender/non_empty_rows_simple passes. |
| DIFF-02 | 01-03 | Syntax highlighting via Chroma (200+ languages) | ✓ SATISFIED | `Highlight()` uses `lexers.Match(filename)` with fallback. `terminal16m` formatter. TestHighlight/detectable_language_has_ansi verifies ANSI codes on README.md content. |
| DIFF-03 | 01-03 | Line-level diff colors (added/removed/modified) layered with syntax highlighting | ✓ SATISFIED | `lineBg()` maps LineKind to 256-colour backgrounds (bgAdded/bgRemoved/bgModified). Applied in `renderSide()` over chroma ANSI. TestRender/rows_contain_ansi_codes passes. |
| DIFF-04 | 01-03 | Intra-line word/character-level markers on modified lines with 1000-char/200-token/100ms guards | ✓ SATISFIED | `applyIntraLine()` with `shouldSkipIntraLine()` + `computeIntraLineWithTimeout()`. `DiffMain(old, new, false)` (checklines=false per D-06). All guard tests pass. |
| DIFF-05 | 01-01, 01-02 | Full-file mode by default — entire file rendered, unchanged lines shown in full | PARTIAL / DEFERRED | FullFile is iota 0 (default) in model.go; `Render()` calls `Align(file, FullFile)`. Context lines included in FullFile output. HunkOnly distinction deferred to Phase 3/DIFF-06. REQUIREMENTS.md marks as Pending (consistent with deferral). |
| DIFF-07 | 01-02, 01-03 | Binary files, pure renames, mode-only, submodule, no-newline-at-EOF render correctly | ✓ SATISFIED | align.go: binary→single placeholder RowPair; isModeOnly→mode notice; isSubmodule→passthrough without Modified pairing. parse_test.go verifies IsBinary, IsRename, OldMode≠NewMode, FileMode 0160000, NoEOL(). All tests pass. Note: REQUIREMENTS.md checkbox `[ ]` not updated — documentation discrepancy, not a code issue. |

**DIFF-07 documentation note:** REQUIREMENTS.md shows DIFF-07 as `[ ]` unchecked with traceability "Pending", while DIFF-01–04 are checked. The implementation fully satisfies DIFF-07 (all five edge cases are handled and tested). The checkbox was not updated in the state commit `0b8e2ec`. This is a documentation artifact only.

### Anti-Patterns Found

| File | Pattern | Severity | Notes |
|------|---------|----------|-------|
| Multiple files | "placeholder" word | ℹ️ Info | Used as domain concept (binary placeholder, mode-only placeholder row) in tests and comments. Not a stub indicator. |
| render.go | `width` parameter accepted but not used for column truncation | ℹ️ Info | Explicit Phase 1 design decision per Plan 03: "hard truncation is unnecessary... lipgloss-based truncation arrives in Phase 3." Width validated (min 4) and parameter is wired for API compatibility. |

No TBD, FIXME, or XXX markers found in any Go source file. No stub implementations. No empty-return stubs.

### Human Verification Required

None — all truths were verified programmatically. No visual/real-time/external-service checks required for this library-only phase.

### Gaps Summary

No gaps. One item deferred to Phase 3:

**SC4 — Full-file/hunk-only behavioral distinction** is deferred to Phase 3. In Phase 1, both `FullFile` and `HunkOnly` modes produce identical output because the diff output already contains only hunk-local context (no inter-hunk unchanged lines to omit). The align.go code documents this: `"HunkOnly: same as FullFile for Phase 1 ... the distinction matters in Phase 3 when the original file is available."` Phase 3's DIFF-06 requirement explicitly requires the toggle to work, which in turn requires the library distinction.

---

_Verified: 2026-06-26T12:00:00Z_
_Verifier: Claude (gsd-verifier)_
