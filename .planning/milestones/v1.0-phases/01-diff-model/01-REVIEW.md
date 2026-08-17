---
phase: 01-diff-model
reviewed: 2026-06-26T00:00:00Z
depth: standard
files_reviewed: 10
files_reviewed_list:
  - internal/diff/align.go
  - internal/diff/align_test.go
  - internal/diff/highlight.go
  - internal/diff/highlight_test.go
  - internal/diff/model.go
  - internal/diff/parse.go
  - internal/diff/parse_test.go
  - internal/diff/render.go
  - internal/diff/render_test.go
  - internal/diff/testdata/binary.diff
  - internal/diff/testdata/deleted-file.diff
  - internal/diff/testdata/large-line.diff
  - internal/diff/testdata/many-tokens.diff
  - internal/diff/testdata/mode-only.diff
  - internal/diff/testdata/multi-file.diff
  - internal/diff/testdata/multi-hunk.diff
  - internal/diff/testdata/multiline-string.diff
  - internal/diff/testdata/new-file.diff
  - internal/diff/testdata/no-newline.diff
  - internal/diff/testdata/rename.diff
  - internal/diff/testdata/simple.diff
  - internal/diff/testdata/submodule.diff
findings:
  critical: 1
  warning: 4
  info: 5
  total: 10
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-06-26
**Depth:** standard
**Files Reviewed:** 10 source files + 12 test fixture files
**Status:** issues_found

## Summary

The `internal/diff` package implements a Parse → Align → Highlight → Render pipeline. The
`parse.go` and `align.go` layers are structurally sound: edge cases for binary, mode-only,
submodule, rename, and no-newline diffs are handled, and the test corpus covers the full fixture
set. The `highlight.go` tokenisation strategy is correct and the per-line reset anti-bleed approach
is good practice.

There is one blocker in `render.go`: the public `Render(file, width)` function accepts a terminal
width but never uses it — no column truncation is implemented. Four warnings follow: deleted files
silently lose syntax highlighting due to an incorrect filename lookup, Modified rows discard the
chroma ANSI output and lose syntax colouring, an empty deleted file can be misclassified as a
mode-only change, and the placeholder-detection heuristic in `highlight.go` relies on content
string prefixes that can match real source code.

---

## Critical Issues

### CR-01: `Render` accepts `width` but never uses it — no column truncation implemented

**File:** `internal/diff/render.go:54-68`

**Issue:** `Render(file *gitdiff.File, width int) []string` clamps `width` to a minimum of 4
(line 56) and then assigns the result to a local variable that is never read again. Every function
in the rendering chain — `renderPair`, `renderSide`, `applyIntraLine`, `joinColumns` — accepts no
width argument. No truncation, padding, or wrapping is applied to any column.

The function doc comment states "each column receives `width/2-1` characters," which is completely
unimplemented. In a real TUI at 80 columns, a single long minified line, a generated constant, or
any line longer than ~39 characters will overflow into the right column and corrupt the layout.

The test `width_independence` does not detect this bug because it only asserts that row *counts*
are equal for width 80 vs 160. Since width is unused, the row counts are always identical and the
test trivially passes.

**Fix:** Thread `colWidth := width/2 - 1` to the functions that produce per-column content and
truncate at that boundary. Because truncation must not break mid-ANSI-sequence, an ANSI-aware
truncation helper is required:

```go
// render.go — Render
func Render(file *gitdiff.File, width int) []string {
    if width < 4 {
        width = 4
    }
    colWidth := width/2 - 1

    pairs := Align(file, FullFile)
    _ = Highlight(pairs, effectiveName(file))

    rows := make([]string, 0, len(pairs))
    for _, p := range pairs {
        rows = append(rows, renderPairWidth(p, colWidth))
    }
    return rows
}

func renderPairWidth(p RowPair, colWidth int) string {
    var left, right string
    if p.Left.Kind == LineModifiedOld && p.Right.Kind == LineModifiedNew {
        left, right = applyIntraLine(p)
    } else {
        left = renderSide(p.Left)
        right = renderSide(p.Right)
    }
    return joinColumns(truncateANSI(left, colWidth), truncateANSI(right, colWidth))
}
```

`truncateANSI(s string, maxRunes int) string` must skip ANSI escape sequences when counting visible
width so that escape codes are not split.

---

## Warnings

### WR-01: Deleted files lose syntax highlighting — `Render` passes `file.NewName` which is empty

**File:** `internal/diff/render.go:61`

**Issue:** `Render` passes `file.NewName` to `Highlight` for lexer selection:

```go
_ = Highlight(pairs, file.NewName)
```

For deleted files (`IsDelete == true`), go-gitdiff sets `NewName` to an empty string or
`/dev/null`; the real path is in `OldName`. Chroma's `lexers.Match("")` and
`lexers.Match("/dev/null")` both return `nil`, causing a fallback to `lexers.Fallback`
(plaintext). Every deleted file — regardless of language — is rendered with no syntax colour.

The parse test confirms this: `deleted-file.diff` uses `wantNameIsOld: true`, meaning the authors
know `NewName` is not the real path for deleted files. No highlight test exercises a deleted-file
fixture, so the regression was not caught.

**Fix:**

```go
// effectiveName returns the best filename for lexer selection.
// For deleted files, NewName is empty; fall back to OldName.
func effectiveName(f *gitdiff.File) string {
    if f.NewName != "" && f.NewName != "/dev/null" {
        return f.NewName
    }
    return f.OldName
}

// In Render:
_ = Highlight(pairs, effectiveName(file))
```

---

### WR-02: `applyIntraLine` discards chroma ANSI — Modified rows lose syntax highlighting

**File:** `internal/diff/render.go:111-147`

**Issue:** `Highlight` populates `p.Left.ANSI` and `p.Right.ANSI` for every non-blank RowPair,
including Modified pairs. `renderSide` correctly prefers the `ANSI` field. However,
`applyIntraLine` rebuilds its output from `p.Left.Content` / `p.Right.Content` (raw text), never
reading the ANSI fields:

```go
func applyIntraLine(p RowPair) (left, right string) {
    oldRaw := p.Left.Content   // raw text; p.Left.ANSI (chroma output) is ignored
    newRaw := p.Right.Content  // raw text; p.Right.ANSI (chroma output) is ignored
    ...
    lb.WriteString(d.Text)     // raw, uncoloured characters
    ...
    left = bgModified + lb.String() + ansiReset  // chroma foreground absent
}
```

The documented ANSI layer order in the `Render` doc comment is:
1. Syntax highlighting foreground (chroma, via Highlight)
2. Line-level diff background
3. Intra-line bold spans on Modified rows

Layer 1 is missing for all Modified rows. Ironically, the guard fallback path (when
`shouldSkipIntraLine` trips) calls `renderSide`, which DOES use `p.Left.ANSI` — so syntax
highlighting is present only on rows where intra-line diffing is skipped. The most important rows
in any diff (changed lines) show no syntax colour.

**Fix:** The ANSI composition is non-trivial because `spanStart`/`spanEnd` must be injected into
already-highlighted text without breaking existing ANSI sequences. A tractable Phase 1 fix: use
the chroma-highlighted `p.Left.ANSI` as the base and inject bold spans at character-position
boundaries computed from `DiffMain`-on-raw-content, adjusting offsets to skip over ANSI escape
sequences. A simpler but acceptable interim: document explicitly that Modified rows suppress syntax
highlighting in favour of intra-line spans, and remove the misleading layer-order comment from the
`Render` doc block so the discrepancy is not hidden.

---

### WR-03: `isModeOnly` false-positive for empty deleted files

**File:** `internal/diff/align.go:24-26`

**Issue:** `isModeOnly` returns true when:
- `f.OldMode != f.NewMode` (modes differ)
- `f.OldMode != 0` (old mode is known)
- `len(f.TextFragments) == 0` (no text content)

An empty deleted file (e.g., a zero-byte `__init__.py` or an empty lock file) satisfies all three
conditions: go-gitdiff sets `IsDelete = true`, `OldMode = 0100644` (non-zero), `NewMode = 0`
(file is gone), and `TextFragments = []` (nothing to diff on an empty file). The `Align` function
checks `isModeOnly` before it inspects `IsDelete`, so the file is treated as a mode-only change and
returns `[Mode changed: 100644 → 0000]` as the placeholder row instead of rendering it as a deleted
empty file.

**Fix:** Guard with the explicit deletion and new-file flags:

```go
func isModeOnly(f *gitdiff.File) bool {
    return !f.IsDelete && !f.IsNew &&
        f.OldMode != f.NewMode && f.OldMode != 0 && len(f.TextFragments) == 0
}
```

---

### WR-04: `isPlaceholderSide` uses `"["` string prefix — fragile heuristic that can misclassify real source lines

**File:** `internal/diff/highlight.go:172-178`

**Issue:** `isPlaceholderSide` treats any line whose `Content` starts with `"["` as a placeholder
that bypasses chroma:

```go
return strings.HasPrefix(content, "[") ||
    strings.HasPrefix(content, "Subproject commit")
```

`isPlaceholderPairs` requires ALL rows to satisfy this check before the bypass fires. This is a
narrow risk, but it is reproducible: a TOML config file where all changed lines are section headers
(e.g., `-[database]` → `+[storage]`), a TOML file with only table-level renames, or an INI file
diff will have every diff line start with `"["`. The result is that chroma is skipped for the
entire file and raw text is shown with no colour; the user receives no error or indication.

**Fix:** Track placeholder origin structurally rather than by content inspection. Add a boolean to
`RowPair` and set it only in the three `Align` early-exit paths:

```go
// model.go
type RowPair struct {
    Left          RenderedLine
    Right         RenderedLine
    IsPlaceholder bool // set by Align for binary/mode-only/submodule placeholder rows
}

// align.go — binary case
return []RowPair{{
    Left:          RenderedLine{Kind: LineContext, Content: notice},
    Right:         RenderedLine{Kind: LineBlank},
    IsPlaceholder: true,
}}

// highlight.go — isPlaceholderPairs
func isPlaceholderPairs(pairs []RowPair) bool {
    for _, p := range pairs {
        if !p.IsPlaceholder {
            return false
        }
    }
    return true
}
```

---

## Info

### IN-01: `Highlight` errors silently discarded with no logging path

**File:** `internal/diff/render.go:61`

**Issue:** `_ = Highlight(pairs, file.NewName)` suppresses all errors from chroma tokenisation and
formatting. The ANSI fallback to raw `Content` is a reasonable degradation, but in a debug session
the developer has no signal that highlighting failed. The recommended stack includes
`charmbracelet/log` for debug-mode file logging (CLAUDE.md). This call site should log the error
once that infrastructure is in place.

**Fix:**
```go
if err := Highlight(pairs, effectiveName(file)); err != nil {
    // log.Debug("highlight failed, falling back to raw content", "file", effectiveName(file), "err", err)
    _ = err
}
```

---

### IN-02: `computeIntraLineWithTimeout` parameter `new` shadows Go builtin

**File:** `internal/diff/render.go:189`

**Issue:**

```go
func computeIntraLineWithTimeout(old, new string) ([]diffmatchpatch.Diff, bool) {
```

The parameter `new` shadows the Go builtin allocator `new()` within the function scope. While
`new()` is not called here, `revive` (listed as a required linter in CLAUDE.md's CI config) flags
this as a builtin shadow.

**Fix:** Rename to `newLine` to match the naming used in `shouldSkipIntraLine`:

```go
func computeIntraLineWithTimeout(oldLine, newLine string) ([]diffmatchpatch.Diff, bool) {
```

---

### IN-03: `min` helper in `render_test.go` shadows Go 1.21+ builtin

**File:** `internal/diff/render_test.go:192-197`

**Issue:** The file defines `func min(a, b int) int` in the `diff_test` package. With `go 1.25`
declared in `go.mod`, the language-builtin `min` is available and performs the same operation.
`golangci-lint` with `revive` will flag this as an unnecessary shadow of a builtin.

**Fix:** Delete lines 192-197 and use the builtin `min` directly at the one call site (line 88).

---

### IN-04: `shouldSkipIntraLine` uses byte length instead of rune count for the 1000-char guard

**File:** `internal/diff/render.go:176`

**Issue:**

```go
return len(oldLine) > 1000 || len(newLine) > 1000 || ...
```

`len(s)` in Go returns the byte length of a UTF-8 string. A line containing 350 CJK characters
occupies ~1050 bytes and trips the guard even though it has only 350 code points — well under the
intended 1000-character threshold. Intra-line highlighting is suppressed for these lines even though
`DiffMain` could handle them efficiently.

**Fix:**
```go
import "unicode/utf8"

func shouldSkipIntraLine(oldLine, newLine string) bool {
    return utf8.RuneCountInString(oldLine) > 1000 ||
        utf8.RuneCountInString(newLine) > 1000 ||
        countTokens(oldLine) > 200 ||
        countTokens(newLine) > 200
}
```

---

### IN-05: `alignText` condition `if mode == HunkOnly || mode == FullFile` is always true

**File:** `internal/diff/align.go:163`

**Issue:** `RenderMode` has exactly two defined values (`FullFile = 0`, `HunkOnly = 1`). The
condition covers both, so context lines are always emitted regardless of `mode`. The comment
documents this as intentional Phase 1 behaviour ("In Phase 1 this is equivalent to FullFile"),
but the dead conditional will mislead a future contributor adding a third `RenderMode` value who
might assume HunkOnly already has a suppression branch.

Additionally, `HunkOnly` mode is inaccessible through `Render`, which calls `Align(file, FullFile)`
unconditionally. Phase 3 cannot request HunkOnly rendering via the public API.

**Fix (deferred to Phase 3):** Replace the dead conditional with a plain unconditional append and
add a Phase 3 TODO comment indicating where the HunkOnly branch must go:

```go
case gitdiff.OpContext:
    content := stripNewline(lines[i].Line)
    // TODO(Phase 3): In HunkOnly mode, omit inter-hunk context lines when the
    // full file source is available. For Phase 1, all context is included.
    result = append(result, RowPair{
        Left:  RenderedLine{Kind: LineContext, Content: content},
        Right: RenderedLine{Kind: LineContext, Content: content},
    })
    i++
```

---

_Reviewed: 2026-06-26_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
