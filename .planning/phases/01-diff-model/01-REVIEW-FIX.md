---
phase: 01-diff-model
fixed_at: 2026-06-26T00:00:00Z
review_path: .planning/phases/01-diff-model/01-REVIEW.md
iteration: 1
findings_in_scope: 5
fixed: 5
skipped: 0
status: all_fixed
---

# Phase 01: Code Review Fix Report

**Fixed at:** 2026-06-26
**Source review:** .planning/phases/01-diff-model/01-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 5 (CR-01, WR-01, WR-02, WR-03, WR-04)
- Fixed: 5
- Skipped: 0

## Fixed Issues

### CR-01: `Render` accepts `width` but never uses it — no column truncation implemented

**Files modified:** `internal/diff/render.go`
**Commit:** 54c81ea
**Applied fix:** Added `truncateANSI(s string, maxRunes int) string` helper that skips ANSI CSI
escape sequences when counting visible rune positions so sequences are never split mid-escape.
Added `renderPairWidth(p RowPair, colWidth int) string` that truncates both columns to
`colWidth` before joining them. Updated `Render` to compute `colWidth := width/2 - 1` and
call `renderPairWidth` instead of `renderPair`. The `width` parameter is now used end-to-end.

Note: also incorporates WR-01 fix (see below) since both touch the same `Render` function.

### WR-01: Deleted files lose syntax highlighting — `Render` passes `file.NewName` which is empty

**Files modified:** `internal/diff/render.go`
**Commit:** 54c81ea (combined with CR-01)
**Applied fix:** Added `effectiveName(f *gitdiff.File) string` helper that returns `f.NewName`
when it is non-empty and not `/dev/null`, and falls back to `f.OldName` otherwise. Updated
`Render` to call `Highlight(pairs, effectiveName(file))` instead of `Highlight(pairs, file.NewName)`.
Deleted files now receive a valid lexer matched from their original filename.

### WR-02: `applyIntraLine` discards chroma ANSI — Modified rows lose syntax highlighting

**Files modified:** `internal/diff/render.go`
**Commit:** 86791a9
**Applied fix:** Applied the acceptable interim fix: updated the `Render` doc comment to
accurately describe the ANSI layer order — noting that syntax highlighting applies to
Added/Removed/Context rows, and that Modified rows use intra-line span markers on raw
Content instead (chroma is not applied). Updated the `applyIntraLine` doc comment to
explicitly document that it operates on `p.Left.Content` (not the chroma ANSI field) and
that full ANSI+intra-line composition is deferred to a future phase. The misleading "Layer 1:
Syntax highlighting foreground" claim for Modified rows is removed from the doc block.

Note: This finding is classified as `fixed: requires human verification` since the underlying
behaviour (no syntax highlighting on Modified rows) is unchanged — only the documentation
accurately reflects it now. The actual composition fix is deferred to a future phase.

### WR-03: `isModeOnly` false-positive for empty deleted files

**Files modified:** `internal/diff/align.go`
**Commit:** 7503501
**Applied fix:** Added `!f.IsDelete && !f.IsNew &&` guards at the start of `isModeOnly`.
An empty deleted file (`IsDelete=true`, `OldMode=0100644`, `NewMode=0`, no TextFragments)
previously satisfied all mode-only conditions and was misclassified. The explicit deletion
and new-file flag checks ensure only genuine pure-permission changes are treated as mode-only.

### WR-04: `isPlaceholderSide` uses `"["` string prefix — fragile heuristic

**Files modified:** `internal/diff/model.go`, `internal/diff/align.go`, `internal/diff/highlight.go`
**Commit:** a919e2e
**Applied fix:** Added `IsPlaceholder bool` field to `RowPair` in `model.go` with clear
documentation. Updated `Align` in `align.go` to set `IsPlaceholder: true` on the binary
placeholder row, the mode-only placeholder row, and all rows produced by `alignSubmodule`
(submodule commit SHA strings are not source code and bypass chroma). Updated
`isPlaceholderPairs` in `highlight.go` to check `p.IsPlaceholder` instead of using
`strings.HasPrefix(content, "[")` and `strings.HasPrefix(content, "Subproject commit")`.
The `isPlaceholderSide` function was removed entirely. A TOML config file with
table-header-only changed lines (all starting with `[`) is now correctly highlighted.

---

_Fixed: 2026-06-26_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
