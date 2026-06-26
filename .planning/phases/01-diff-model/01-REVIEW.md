---
phase: 01-diff-model
reviewed: 2026-06-26T00:00:00Z
depth: standard
files_reviewed: 9
files_reviewed_list:
  - internal/diff/model.go
  - internal/diff/parse.go
  - internal/diff/parse_test.go
  - internal/diff/align.go
  - internal/diff/align_test.go
  - internal/diff/highlight.go
  - internal/diff/highlight_test.go
  - internal/diff/render.go
  - internal/diff/render_test.go
findings:
  critical: 3
  warning: 4
  info: 4
  total: 11
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-06-26
**Depth:** standard
**Files Reviewed:** 9
**Status:** issues_found

## Summary

The diff model pipeline (Parse → Align → Highlight → Render) is largely well-structured. The
`Parse` and `Align` layers are sound for the normal case. However three blockers were found in
`render.go` and `highlight.go` that undermine the stated feature contract: the `width` parameter
passed to `Render` has no effect on output, syntax highlighting is silently absent for all deleted
files, and Chroma highlighting is discarded entirely for Modified rows (the most common case in any
diff). Four warnings cover an edge-case misclassification in `isModeOnly`, a fragile placeholder
heuristic in `isPlaceholderPairs`, a goroutine that outlives its timeout, and a `HunkOnly` mode
that is unreachable through the public `Render` API.

---

## Critical Issues

### CR-01: `Render` accepts `width` parameter but never uses it

**File:** `internal/diff/render.go:54-68`

**Issue:** `Render(file, width)` clamps `width` to a minimum of 4 then never passes it to any
downstream function. `renderPair`, `renderSide`, `applyIntraLine`, and `joinColumns` take no width
argument, so every column is rendered at the raw content length with no truncation or padding.
The public contract documented in the function comment ("each column receives `width/2-1`
characters") is unimplemented. In the TUI, lines longer than `width/2-1` will overflow into the
opposing column and corrupt the side-by-side layout.

The test `width_independence` passes because it only compares row *counts* (which are
width-independent), not column widths. The comment "width changes column widths, not row count"
is aspirational, not tested.

**Fix:** Thread `colWidth := width/2 - 1` through to `renderSide` and `joinColumns`, truncating
(or wrapping) content that exceeds `colWidth`. Example signature change:

```go
func renderSide(line RenderedLine, colWidth int) string { ... }
func joinColumns(left, right string, colWidth int) string {
    // pad/truncate left to colWidth, then join
}
```

---

### CR-02: Syntax highlighting disabled for all deleted files

**File:** `internal/diff/render.go:61`

**Issue:** `Render` calls `Highlight(pairs, file.NewName)`. For deleted files (`IsDelete == true`),
go-gitdiff sets `NewName` to `/dev/null` (the `+++` header in the diff) while the real filename
is in `OldName`. Chroma's `lexers.Match("/dev/null")` returns nil, so the fallback plaintext
lexer is used — no syntax coloring is produced for any deleted file regardless of its actual
language. The parse tests confirm this: `deleted-file.diff` is tested with `wantNameIsOld: true`,
which means authors know `NewName` is not the real path for deleted files.

The highlight tests never exercise a deleted-file fixture through `Highlight`, so this gap was not
caught.

**Fix:** Pass the most informative filename available:

```go
name := file.NewName
if name == "" || name == "/dev/null" {
    name = file.OldName
}
_ = Highlight(pairs, name)
```

---

### CR-03: `applyIntraLine` discards Chroma syntax highlighting for Modified rows

**File:** `internal/diff/render.go:111-147`

**Issue:** `applyIntraLine` builds its output from `p.Left.Content` / `p.Right.Content` (raw
text), completely ignoring the `p.Left.ANSI` / `p.Right.ANSI` fields that `Highlight` populated.
The character-diff (`dmp.DiffMain`) is computed on raw strings, and the result is wrapped with
`bgModified + plaintext + ansiReset`. Chroma foreground colors are absent from every Modified row.

The documented ANSI layer order (§Render doc comment) states layer 1 is "Syntax highlighting
foreground (chroma, via Highlight)". That layer is bypassed for the most common row type in any
diff. The guard fallback path (`shouldSkipIntraLine` trips) correctly calls `renderSide`, which
does use `p.Left.ANSI` — so syntax highlighting is paradoxically present only when the intra-line
diff is *skipped*.

**Fix:** Integrate the ANSI content into the character-span reconstruction, or apply the diff
against the raw text and then compose the result with the ANSI rendering. One practical approach:
apply the intra-line `spanStart`/`spanEnd` markers as an overlay onto the already-highlighted
ANSI string using character-position mapping. A simpler but still correct approach: run the diff
on `Content` to identify changed character ranges, then annotate `p.Left.ANSI` by injecting span
markers at the same byte offsets (adjusting for ANSI sequences already present).

---

## Warnings

### WR-01: `isModeOnly` false-positive for empty deleted files

**File:** `internal/diff/align.go:24-26`

**Issue:** `isModeOnly` fires when `OldMode != NewMode && OldMode != 0 && len(TextFragments) == 0`.
For an empty deleted file (e.g., an empty `__init__.py` or zero-byte lock file being removed),
go-gitdiff sets `IsDelete = true`, `OldMode = 0100644`, `NewMode = 0`, `TextFragments = []`.
All three conditions are satisfied, so `Align` returns a `[Mode changed: 0644 → 0000]` placeholder
instead of treating the file as a deleted empty file. The check runs before `IsDelete` is
considered. A pure mode change on a non-deleted file correctly sets both OldMode and NewMode to
non-zero values (as in `mode-only.diff`), so the guard `OldMode != 0` is insufficient to
distinguish the two cases.

**Fix:** Add explicit exclusion of deletion and new-file events:

```go
func isModeOnly(f *gitdiff.File) bool {
    return !f.IsDelete && !f.IsNew &&
        f.OldMode != f.NewMode && f.OldMode != 0 && len(f.TextFragments) == 0
}
```

---

### WR-02: `isPlaceholderPairs` uses fragile content-prefix heuristic

**File:** `internal/diff/highlight.go:159-178`

**Issue:** `isPlaceholderSide` classifies a non-blank line as a placeholder if its `Content`
starts with `"["` or `"Subproject commit"`. If *every* line in the diff of a real source file
happens to start with `"["` — e.g., a TOML file where the only changed lines are section headers
like `[section1]` → `[section2]`, or an INI file, or a diff showing only `[]interface{}` type
annotations in Go — `isPlaceholderPairs` returns `true` and Chroma is skipped for the entire
file. The failure mode is silent: output falls back to unhighlighted `Content`.

A TOML diff of only section renames would consistently reproduce this:

```toml
-[database]
+[storage]
```

Both lines start with `"["`, `isPlaceholderPairs` returns `true`, no highlighting applied.

**Fix:** Track placeholder origin at the structural level rather than by content inspection.
Add a boolean field to `RowPair` or use a sentinel `LineKind` for placeholder rows emitted by
`Align`:

```go
type RowPair struct {
    Left        RenderedLine
    Right       RenderedLine
    IsPlaceholder bool  // set by Align for binary/mode-only/submodule markers
}
```

Then `isPlaceholderPairs` reduces to a single-field check with no content inspection.

---

### WR-03: Goroutine leak when `computeIntraLineWithTimeout` fires

**File:** `internal/diff/render.go:189-201`

**Issue:** When `time.After(100ms)` fires, the function returns `(nil, true)` but the goroutine
running `dmp.DiffMain(old, new, false)` continues executing in the background without any
cancellation signal. The buffered channel (`done := make(chan result, 1)`) prevents the goroutine
from blocking permanently once it completes, but for pathological inputs that slip past the
`shouldSkipIntraLine` guards (e.g., 999-character strings with many differences), DiffMain can
take several seconds. If `Render` is invoked concurrently on many such pairs, leaked goroutines
accumulate.

**Fix:** `diffmatchpatch` has no context-cancellation support, so true cancellation is not
possible. Document the limitation and add a comment that the goroutine will eventually unblock.
As a stronger mitigation, extend the guards so that inputs likely to cause slow DiffMain runs are
caught before the goroutine is spawned (e.g., lower the char threshold or add an edit-distance
estimate):

```go
// NOTE: the spawned goroutine cannot be cancelled; it will complete and send to
// the buffered channel. The channel will be GC'd after the goroutine exits.
```

---

### WR-04: `HunkOnly` mode is unreachable through the public `Render` API

**File:** `internal/diff/render.go:60`

**Issue:** `Render` calls `Align(file, FullFile)` unconditionally. The `RenderMode` parameter
(`HunkOnly` / `FullFile`) is correctly threaded through `Align` → `alignText`, but is never
exposed as a parameter on `Render`. Phase 3 (TUI) cannot request `HunkOnly` rendering via the
published API. The `HunkOnly` mode implemented in `Align` is therefore inaccessible to callers
today.

Additionally, the `alignText` condition `if mode == HunkOnly || mode == FullFile` is always true
(both defined enum values satisfy it), making HunkOnly and FullFile produce identical output.
When Phase 3 needs real HunkOnly behavior (omitting between-hunk context), the condition will
need to be corrected as well.

**Fix:** Add a `mode RenderMode` parameter to `Render`:

```go
func Render(file *gitdiff.File, width int, mode RenderMode) []string {
    ...
    pairs := Align(file, mode)
    ...
}
```

---

## Info

### IN-01: `splitAndReset` appends `ansiReset` to empty strings

**File:** `internal/diff/highlight.go:140-148`

**Issue:** The reset loop appends `\x1b[0m` to every line that doesn't already end with it,
including empty strings. A blank line in the diff (empty `Content`) becomes `"\x1b[0m"` in the
`ANSI` field. In `renderSide`, this non-empty ANSI value is used in preference to the empty
`Content`, producing a spurious but visually harmless reset sequence. The fix is a single guard:

```go
if line != "" && !strings.HasSuffix(line, ansiReset) {
    lines[i] = line + ansiReset
}
```

---

### IN-02: `TestParseMalformed` only asserts no panic, not error return

**File:** `internal/diff/parse_test.go:265-281`

**Issue:** The malformed-input test discards both the result and the error (`_, _ = diff.Parse(...)`).
The test validates that `Parse` does not panic, but does not confirm that a truly malformed diff
header returns a non-nil error. The comment acknowledges go-gitdiff is lenient, but the adversarial
input (`@@ INVALID @@`) should trigger a parse error that the test could assert on:

```go
_, err := diff.Parse(bytes.NewReader(adversarial))
if err == nil {
    t.Error("Parse(adversarial): expected error for invalid hunk header, got nil")
}
```

---

### IN-03: `min` function at package level shadows Go 1.21+ builtin

**File:** `internal/diff/render_test.go:193-198`

**Issue:** The package defines `func min(a, b int) int` at package scope. With `go 1.25` (as
declared in `go.mod`), `min` is a language builtin. The user-defined function shadows the builtin;
this is valid Go but `golangci-lint` with `revive` (listed in CLAUDE.md as a required linter)
will flag it as `shadow of builtin`. The function can be removed entirely since the builtin
performs the same operation.

---

### IN-04: `Highlight` error silently discarded in `Render`

**File:** `internal/diff/render.go:61`

**Issue:** `_ = Highlight(pairs, file.NewName)` discards the error. The "non-fatal" fallback to
`Content` is a reasonable degradation strategy for a TUI, but errors go completely unlogged. When
the debug-mode logging infrastructure is available (Phase 4: `charmbracelet/log` to a file), this
call site should log the error at debug level so that Chroma tokeniser failures are diagnosable:

```go
if err := Highlight(pairs, name); err != nil {
    // log.Debug("highlight failed, falling back to plain content", "file", name, "err", err)
    _ = err
}
```

---

_Reviewed: 2026-06-26_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
