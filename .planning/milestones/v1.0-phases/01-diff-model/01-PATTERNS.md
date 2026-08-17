# Phase 1: Diff Model - Pattern Map

**Mapped:** 2026-06-26
**Files analyzed:** 8 new files (5 source + 3 test)
**Analogs found:** 0 / 8 — greenfield project, no existing source files

## File Classification

| New File | Role | Data Flow | Closest Analog | Match Quality |
|----------|------|-----------|----------------|---------------|
| `internal/diff/model.go` | model | — | none | no analog |
| `internal/diff/parse.go` | utility | transform | none | no analog |
| `internal/diff/align.go` | utility | transform | none | no analog |
| `internal/diff/highlight.go` | utility | transform | none | no analog |
| `internal/diff/render.go` | utility | transform | none | no analog |
| `internal/diff/parse_test.go` | test | — | none | no analog |
| `internal/diff/align_test.go` | test | — | none | no analog |
| `internal/diff/render_test.go` | test | — | none | no analog |

## Pattern Assignments

This is a greenfield project. No existing codebase analogs exist. All patterns below are
sourced directly from RESEARCH.md library API documentation and Go standard conventions.

---

### `internal/diff/model.go` (model)

**Analog:** none — establish conventions here; all other files follow this file's type definitions.

**Imports pattern:**
```go
package diff
```

**Core type definitions:**
```go
type LineKind int

const (
    LineContext      LineKind = iota
    LineAdded                 // new side only
    LineRemoved               // old side only
    LineModifiedOld           // part of delete+add pair (old side)
    LineModifiedNew           // part of delete+add pair (new side)
    LineBlank                 // alignment filler
)

type RenderMode int

const (
    FullFile  RenderMode = iota // render all lines including unchanged context
    HunkOnly                    // render only changed hunks
)

type RenderedLine struct {
    Kind    LineKind
    Content string // raw text (before highlighting)
    ANSI    string // highlighted + diff-colored ANSI string (set by highlight.go)
}

type RowPair struct {
    Left  RenderedLine
    Right RenderedLine
}
```

---

### `internal/diff/parse.go` (utility, transform)

**Analog:** none — patterns sourced from RESEARCH.md Pattern 1 (go-gitdiff Parse API).

**Imports pattern:**
```go
package diff

import (
    "fmt"
    "io"

    "github.com/bluekeyes/go-gitdiff/gitdiff"
)
```

**Core parse pattern** (RESEARCH.md Pattern 1):
```go
// Parse wraps gitdiff.Parse, returning typed File structs.
// Phase 2 (internal/git) calls this with io.Reader from a git subprocess.
func Parse(r io.Reader) ([]*gitdiff.File, error) {
    files, _, err := gitdiff.Parse(r)
    if err != nil {
        return nil, fmt.Errorf("parsing diff: %w", err)
    }
    return files, nil
}
```

**Error handling:** wrap errors with `fmt.Errorf("...: %w", err)` — no panic.

**Key struct fields to use** (RESEARCH.md Pattern 1):
- `gitdiff.File`: `OldName`, `NewName`, `IsNew`, `IsDelete`, `IsRename`, `IsCopy`, `IsBinary`, `OldMode`, `NewMode`, `TextFragments`, `BinaryFragment`
- `gitdiff.Line`: `Op` (OpContext/OpDelete/OpAdd), `Line` (string), `NoEOL()` method

**Important:** The struct is `gitdiff.File`, NOT `gitdiff.FileDiff`. The function is `gitdiff.Parse()`, NOT `ParseMultiFileDiff()`.

---

### `internal/diff/align.go` (utility, transform)

**Analog:** none — patterns sourced from RESEARCH.md Pattern 2 (edge-case detection) and Pattern 3 (alignment algorithm).

**Imports pattern:**
```go
package diff

import (
    "fmt"
    "os"

    "github.com/bluekeyes/go-gitdiff/gitdiff"
)
```

**Edge-case detection pattern** (RESEARCH.md Pattern 2):
```go
const submoduleMode = os.FileMode(0160000)

func isSubmodule(f *gitdiff.File) bool {
    return f.OldMode == submoduleMode || f.NewMode == submoduleMode
}

func isModeOnly(f *gitdiff.File) bool {
    return f.OldMode != f.NewMode && f.OldMode != 0 && len(f.TextFragments) == 0
}
```

**Core alignment pattern** (RESEARCH.md Pattern 3):
```go
// Align converts a parsed gitdiff.File into []RowPair for rendering.
// Adjacent OpDelete+OpAdd sequences are paired as LineModified for intra-line diff.
// Multi-delete+multi-add runs are paired positionally (first delete → first add).
func Align(file *gitdiff.File, mode RenderMode) []RowPair {
    // ...walk TextFragments, group adjacent deletes+adds...
    // Return early for binary, mode-only, submodule with placeholder RowPairs
}
```

**Alignment loop pattern** (RESEARCH.md Pattern 3):
```go
for i < len(lines) {
    line := lines[i]
    switch line.Op {
    case gitdiff.OpContext:
        result = append(result, RowPair{
            Left:  RenderedLine{Kind: LineContext, Content: line.Line},
            Right: RenderedLine{Kind: LineContext, Content: line.Line},
        })
        i++
    case gitdiff.OpDelete:
        if i+1 < len(lines) && lines[i+1].Op == gitdiff.OpAdd {
            result = append(result, RowPair{
                Left:  RenderedLine{Kind: LineModifiedOld, Content: line.Line},
                Right: RenderedLine{Kind: LineModifiedNew, Content: lines[i+1].Line},
            })
            i += 2
        } else {
            result = append(result, RowPair{
                Left:  RenderedLine{Kind: LineRemoved, Content: line.Line},
                Right: RenderedLine{Kind: LineBlank},
            })
            i++
        }
    case gitdiff.OpAdd:
        result = append(result, RowPair{
            Left:  RenderedLine{Kind: LineBlank},
            Right: RenderedLine{Kind: LineAdded, Content: line.Line},
        })
        i++
    }
}
```

**Special-case patterns** (RESEARCH.md Pitfalls 4, 5, 6):
- Binary: check `file.IsBinary` first; return placeholder `[]RowPair` with `"[Binary file changed]"` content
- Mode-only: check `len(file.TextFragments) == 0`; return placeholder `"Mode changed: 0644 → 0755"`
- Submodule: check `OldMode == 0160000`; skip syntax highlight + intra-line, render raw text

---

### `internal/diff/highlight.go` (utility, transform)

**Analog:** none — patterns sourced from RESEARCH.md Pattern 5 (chroma syntax highlighting).

**Imports pattern:**
```go
package diff

import (
    "strings"

    "github.com/alecthomas/chroma/v2/formatters"
    "github.com/alecthomas/chroma/v2/lexers"
    "github.com/alecthomas/chroma/v2/styles"
)
```

**Core highlight pattern** (RESEARCH.md Pattern 5):
```go
// Highlight populates the ANSI field of each RenderedLine in pairs.
// Called once per file after Align, before Render.
func Highlight(pairs []RowPair, filename string) error {
    lexer := lexers.Match(filename)
    if lexer == nil {
        lexer = lexers.Fallback
    }
    style := styles.Get("monokai") // Phase 4 will make this configurable
    if style == nil {
        style = styles.Fallback
    }
    formatter := formatters.Get("terminal16m") // Phase 4: auto-select based on color depth
    if formatter == nil {
        formatter = formatters.Fallback
    }
    // Collect full content for one side, tokenise once, split on "\n"
    // Store per-line ANSI strings back into pairs[i].Left.ANSI / pairs[i].Right.ANSI
}
```

**ANSI per-line split pattern** (RESEARCH.md Pitfall 2):
```go
// After formatter.Format writes to sb, split and reset each line:
lines := strings.Split(sb.String(), "\n")
for i, line := range lines {
    if !strings.HasSuffix(line, "\x1b[0m") {
        lines[i] = line + "\x1b[0m"
    }
}
```

**Anti-pattern to avoid:** Do NOT run `lexer.Tokenise` inside a render loop per frame. Highlight once per file load; store results in `RenderedLine.ANSI`.

---

### `internal/diff/render.go` (utility, transform)

**Analog:** none — patterns sourced from RESEARCH.md Patterns 4 and 6.

**Imports pattern:**
```go
package diff

import (
    "fmt"
    "strings"
    "time"

    "github.com/bluekeyes/go-gitdiff/gitdiff"
    "github.com/sergi/go-diff/diffmatchpatch"
)
```

**Public API pattern** (CONTEXT.md D-03, D-04):
```go
// Render produces one ANSI string per rendered row (left column + right column joined).
// width is the total terminal width; each column gets width/2 - 1 characters.
// No globals. No package-level defaults. Re-callable on terminal resize.
func Render(file *gitdiff.File, width int) []string
```

**Package-level DMP instance** (RESEARCH.md anti-pattern note):
```go
// Create once at package level, not per-call.
var dmp = diffmatchpatch.New()
```

**Intra-line guard pattern** (RESEARCH.md Pattern 4, D-07):
```go
func shouldSkipIntraLine(oldLine, newLine string) bool {
    return len(oldLine) > 1000 || len(newLine) > 1000 ||
        countTokens(oldLine) > 200 || countTokens(newLine) > 200
}

func computeIntraLineWithTimeout(old, new string) ([]diffmatchpatch.Diff, bool) {
    done := make(chan []diffmatchpatch.Diff, 1)
    go func() { done <- dmp.DiffMain(old, new, false) }()
    select {
    case diffs := <-done:
        return diffs, false
    case <-time.After(100 * time.Millisecond):
        return nil, true // timed out
    }
}
```

**ANSI reset at column boundary** (RESEARCH.md Pattern 6, Pitfall 1):
```go
const ansiReset = "\x1b[0m"

// joinColumns resets ANSI state at left column boundary before appending right.
// CRITICAL: without this, chroma foreground color bleeds into right column.
func joinColumns(left, right string, leftWidth int) string {
    return left + ansiReset + " " + right
}
```

**DiffMain call** (RESEARCH.md D-06):
```go
// Always checklines=false for character-level diff (locked by D-06).
diffs := dmp.DiffMain(oldRaw, newRaw, false)
```

---

### `internal/diff/parse_test.go` (test)

**Analog:** none — establish test conventions here.

**Imports pattern:**
```go
package diff_test

import (
    "os"
    "testing"

    "github.com/bluekeyes/go-gitdiff/gitdiff"

    "github.com/alturd/alturd/internal/diff"
)
```

**Test file open pattern** (RESEARCH.md Code Examples):
```go
// Fixture files read with hardcoded relative path from testdata/.
// go test sets cwd to the package directory; testdata/ is always available.
f, err := os.Open("testdata/simple.diff")
if err != nil {
    t.Fatalf("opening fixture: %v", err)
}
defer f.Close()
```

**Table test structure:**
```go
func TestParse(t *testing.T) {
    tests := []struct {
        name          string
        fixture       string
        wantFileCount int
        wantNewName   string
        wantIsNew     bool
        wantIsBinary  bool
    }{
        {name: "simple", fixture: "simple.diff", wantFileCount: 1, wantNewName: "README.md"},
        {name: "binary", fixture: "binary.diff", wantFileCount: 1, wantIsBinary: true},
        // ... all 12 fixture scenarios
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            // open fixture, call diff.Parse, assert fields
        })
    }
}
```

**Line count assertion pattern** (RESEARCH.md Code Examples):
```go
var added, removed, ctx int
for _, frag := range file.TextFragments {
    for _, line := range frag.Lines {
        switch line.Op {
        case gitdiff.OpAdd:     added++
        case gitdiff.OpDelete:  removed++
        case gitdiff.OpContext: ctx++
        }
    }
}
```

---

### `internal/diff/align_test.go` (test)

**Analog:** none.

**Imports pattern:**
```go
package diff_test

import (
    "os"
    "testing"

    "github.com/alturd/alturd/internal/diff"
)
```

**Core assertion pattern** (D-05):
```go
// Assert structural properties, not ANSI content.
// For alignment: assert RowPair counts and LineKind values.
if len(rows) != tc.wantRowCount {
    t.Errorf("Align: got %d rows, want %d", len(rows), tc.wantRowCount)
}
if rows[tc.modifiedIdx].Left.Kind != diff.LineModifiedOld {
    t.Errorf("row[%d].Left.Kind = %v, want LineModifiedOld", tc.modifiedIdx, rows[tc.modifiedIdx].Left.Kind)
}
```

---

### `internal/diff/render_test.go` (test)

**Analog:** none.

**Imports pattern:**
```go
package diff_test

import (
    "os"
    "regexp"
    "strings"
    "testing"

    "github.com/alturd/alturd/internal/diff"
)
```

**ANSI presence assertion** (RESEARCH.md Code Examples, D-05 — no golden snapshots):
```go
var ansiColorRe = regexp.MustCompile(`\x1b\[[0-9;]+m`)

// Assert ANSI codes are present (not exact colors).
if !ansiColorRe.MatchString(coloredLine) {
    t.Error("expected ANSI color codes in highlighted output, got none")
}
```

**Width parameter usage** (D-04):
```go
// Tests always pass a fixed width. Phase 3 passes actual terminal width.
const testWidth = 160
rows := diff.Render(file, testWidth)
```

**Guard path assertion** (D-07):
```go
// Large-line fixture must produce output with NO intra-line markers:
// verify by asserting that lines from large-line.diff do NOT contain
// the bright-highlight escape sequence used for intra-line spans.
```

---

## Shared Patterns

### Error Handling
**Source:** Go standard library convention (no existing project analog)
**Apply to:** `parse.go`, `align.go`, `highlight.go`, `render.go`
```go
// Wrap errors with context; never panic on bad input.
return nil, fmt.Errorf("<function name>: %w", err)
```

### ANSI Reset
**Source:** RESEARCH.md Pattern 6 / Pitfall 1
**Apply to:** `render.go` (column join), `highlight.go` (per-line split)
```go
const ansiReset = "\x1b[0m"
// Emit at: (1) end of every left column string, (2) end of every highlighted line after "\n" split
```

### No-Panic Contract
**Source:** RESEARCH.md §Security Domain
**Apply to:** `parse.go`, `align.go`, `render.go`
- `Parse` returns `error` on malformed input — never panics
- `Align` handles nil/empty `TextFragments` — never panics on edge cases
- `Render` checks `IsBinary` / `len(TextFragments) == 0` before entering render pipeline

### Test Fixture Access
**Source:** Go `testdata/` convention
**Apply to:** `parse_test.go`, `align_test.go`, `render_test.go`
```go
// go test sets cwd to the package dir; access fixtures with relative path:
os.Open("testdata/foo.diff")
// Never: os.Open("internal/diff/testdata/foo.diff")
```

### Package External Test Style
**Apply to:** all test files
```go
// Use package diff_test (external test package), not package diff.
// This enforces testing via the public API, not internals.
package diff_test
```

## No Analog Found

All files in this phase have no close match in the codebase because this is a greenfield project.
The planner must establish conventions from scratch using RESEARCH.md patterns and CLAUDE.md stack choices.

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/diff/model.go` | model | — | No source files exist in the codebase |
| `internal/diff/parse.go` | utility | transform | No source files exist in the codebase |
| `internal/diff/align.go` | utility | transform | No source files exist in the codebase |
| `internal/diff/highlight.go` | utility | transform | No source files exist in the codebase |
| `internal/diff/render.go` | utility | transform | No source files exist in the codebase |
| `internal/diff/parse_test.go` | test | — | No source files exist in the codebase |
| `internal/diff/align_test.go` | test | — | No source files exist in the codebase |
| `internal/diff/render_test.go` | test | — | No source files exist in the codebase |

## Additional Files Required (Not in RESEARCH.md file list)

| File | Role | Reason |
|------|------|--------|
| `go.mod` | config | Module root required before any Go source; `module github.com/alturd/alturd`, `go 1.22` |
| `go.sum` | config | Generated by `go mod tidy` after `go get` of three libraries |
| `.gitattributes` | config | `internal/diff/testdata/*.diff text eol=lf` — prevents CRLF on Windows (RESEARCH.md Pitfall 7) |
| `internal/diff/testdata/*.diff` | test fixture | 12 scenario files — see RESEARCH.md §Recommended Project Structure for full list |

## Metadata

**Analog search scope:** entire repository (only `.claude/CLAUDE.md` exists — no Go source files)
**Files scanned:** 1 (CLAUDE.md only)
**Pattern extraction date:** 2026-06-26
**Greenfield note:** All patterns in this document are sourced from RESEARCH.md library API documentation and Go standard conventions. The first file written (`model.go`) establishes the type definitions that all other files import. Implementation order: `model.go` → `parse.go` → `align.go` → `highlight.go` → `render.go`, with tests alongside each file.
