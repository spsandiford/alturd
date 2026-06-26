# Phase 1: Diff Model - Research

**Researched:** 2026-06-26
**Domain:** Pure Go diff parsing, alignment, syntax highlighting, and ANSI rendering — no TUI code
**Confidence:** MEDIUM

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**D-01:** Copy the Python implementation's raw `.diff` fixture files into `internal/diff/testdata/` in this Go repo. No runtime dependency on the Python repo path.

**D-02:** Fixture files live at `internal/diff/testdata/` (Go convention for `testdata/` — placed next to the package being tested, automatically available to `go test`).

**D-03:** The renderer produces `[]string` — one fully-composed ANSI string per rendered row (left column + right column joined). Phase 3 feeds this slice directly into a bubbletea viewport with no further transformation.

**D-04:** The render function accepts a `width int` parameter: `Render(diff, width int) []string`. Tests pass a fixed width (e.g., 160). Phase 3 passes the actual terminal width. No globals or package-level defaults.

**D-05:** Tests use structural assertions only — no golden ANSI snapshot files. At minimum, each table test asserts:
- Files parsed correctly (correct File count, filename, status marker)
- Added/removed/unchanged line counts match expected values
- Syntax highlighting applied for languages Chroma can detect (presence of ANSI color codes on relevant lines)
- Intra-line character-level markers present on modified lines (when guards permit)
- Edge-case files (binary patches, pure renames, mode-only changes, submodule bumps, no-newline-at-EOF) render the correct placeholder or diff content without panic
- Guard thresholds: tests that exceed the 1000-char / 200-token guards exercise the degraded path and verify no intra-line markers are emitted

**D-06:** Default granularity is character-level — `go-diff DiffMain(old, new, checklines=false)`. Matches Python implementation behavior.

**D-07:** When any guard triggers (line > 1000 chars, token count > 200, or elapsed time > 100ms), skip intra-line entirely and show line-level diff color only. No fallback to word-level, no visual indicator.

**Locked Library Choices (from CLAUDE.md — do not re-litigate):**
- `go-gitdiff v0.8.1` for diff parsing
- `go-diff v1.4.0` for intra-line character diff
- `chroma/v2 v2.27.0` for syntax highlighting
- `CGO_ENABLED=0` — no CGO anywhere

### Claude's Discretion

None specified — all choices locked via D-01 through D-07 and CLAUDE.md.

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DIFF-01 | User sees old and new file content rendered in aligned parallel side-by-side columns | `Align()` pairs old/new lines into `RowPair` structs; `Render()` joins left+right with explicit ANSI reset at column boundary |
| DIFF-02 | User sees syntax highlighting via Chroma (200+ languages, same lexer selection as Pygments) | `lexers.Match(filename)` + `formatters.Get("terminal16m")` + `formatter.Format(&sb, style, iter)` confirmed via pkg.go.dev |
| DIFF-03 | User sees line-level diff colors (added/removed/modified) layered with syntax highlighting | Background-color ANSI sequences applied to full line content after chroma tokenization; reset at line boundary |
| DIFF-04 | User sees intra-line word/character-level change markers on modified lines (with guards) | `diffmatchpatch.New().DiffMain(oldRaw, newRaw, false)` returns `[]Diff`; spans re-inserted into ANSI-encoded strings at rune positions |
| DIFF-05 | User sees full-file mode by default — entire file with all unchanged lines shown | Full-file mode: render all `TextFragment.Lines` across all hunks including context lines; fixtures generated with sufficient `-U` context |
| DIFF-07 | Binary files, pure renames, mode-only changes, submodule bumps, and no-newline-at-EOF all render correct placeholder or diff content | go-gitdiff v0.8.1 exposes `IsBinary`, `IsRename`, `OldMode/NewMode`, `Line.NoEOL()` — all edge cases are typed fields, not string parsing |
</phase_requirements>

---

## Summary

Phase 1 builds `internal/diff` — a pure Go library that parses git diff output and produces aligned side-by-side ANSI output with syntax highlighting and intra-line change markers. No TUI code, no bubbletea, no terminal I/O. The library validates against a fixture corpus of 12+ Python test scenarios before Phase 2 begins.

The technology stack for this phase is minimal: `go-gitdiff` parses the raw unified diff into typed structs; `go-diff` performs character-level LCS to identify intra-line change spans; `chroma/v2` tokenizes source code and formats it to ANSI terminal sequences. The output contract — `[]string` of pre-rendered rows — is intentionally narrow so Phase 3 can feed it directly into a bubbletea viewport.

The two hardest technical problems in this phase are: (1) composing three ANSI layers (syntax highlight foreground, line-level diff background, intra-line span highlight) without color state bleeding across columns or across layers; and (2) building an alignment algorithm that correctly pairs adjacent delete+add line sequences into `Modified` row pairs for intra-line diff processing. Both problems have well-understood solutions that this research documents.

**Primary recommendation:** Implement the five files in this order — `model.go` → `parse.go` → `align.go` → `highlight.go` → `render.go` — with tests written alongside each file rather than after. The fixture corpus gates the test assertions so each file can be independently validated before the next is written.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Diff parsing (unified format) | `internal/diff` | — | go-gitdiff wraps all parsing; diff package owns the typed model |
| Side-by-side alignment | `internal/diff` | — | Pure data transformation; no I/O or TUI involvement |
| Syntax highlighting | `internal/diff` | — | Chroma is pure computation; produces ANSI strings stored in model |
| Intra-line character diff | `internal/diff` | — | go-diff computation; guards enforced here |
| ANSI rendering / column join | `internal/diff` | — | The `Render()` function owns all terminal escape output |
| Git subprocess invocation | `internal/git` (Phase 2) | — | Not in scope for Phase 1; tests use fixture files |
| Viewport display | `internal/ui` (Phase 3) | — | Phase 3 feeds `[]string` from Render into bubbletea viewport |

---

## Standard Stack

### Core (Phase 1 Only)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/bluekeyes/go-gitdiff` | v0.8.1 | Unified git diff parsing | Handles binary patches, renames, extended headers, no-newline markers as typed fields — locked in CLAUDE.md [CITED: pkg.go.dev/github.com/bluekeyes/go-gitdiff/gitdiff] |
| `github.com/sergi/go-diff` | v1.4.0 | Intra-line character diff | Go port of diff-match-patch; `DiffMain(old, new, false)` for char-level — locked in CLAUDE.md [CITED: pkg.go.dev/github.com/sergi/go-diff/diffmatchpatch] |
| `github.com/alecthomas/chroma/v2` | v2.27.0 | Syntax highlighting | Pure-Go Pygments port; 200+ lexers; terminal ANSI formatters — locked in CLAUDE.md [CITED: pkg.go.dev/github.com/alecthomas/chroma/v2] |

### Not Used in Phase 1 (Introduced Later)

The following packages from CLAUDE.md are NOT installed in Phase 1 — they are TUI/CLI dependencies:
- `charm.land/bubbletea/v2` — Phase 3
- `charm.land/lipgloss/v2` — Phase 3
- `charm.land/bubbles/v2` — Phase 3
- `github.com/muesli/termenv` — Phase 4 (theme detection)
- `github.com/pelletier/go-toml/v2` — Phase 4 (config)
- `github.com/adrg/xdg` — Phase 4 (XDG paths)

**Phase 1 Installation:**
```bash
go mod init github.com/alturd/alturd
go get github.com/bluekeyes/go-gitdiff@v0.8.1
go get github.com/sergi/go-diff@v1.4.0
go get github.com/alecthomas/chroma/v2@v2.27.0
go mod tidy
```

---

## Package Legitimacy Audit

> The `gsd-tools query package-legitimacy` seam supports npm/PyPI/crates only — Go module legitimacy requires manual verification via `pkg.go.dev` and module registry.

| Package | Registry | Verified | Downloads Signal | Source Repo | Verdict | Disposition |
|---------|----------|----------|-----------------|-------------|---------|-------------|
| `github.com/bluekeyes/go-gitdiff` | pkg.go.dev | v0.8.1 [CITED: pkg.go.dev] | Established — used by Gitea, gitoxide-derived tools | github.com/bluekeyes/go-gitdiff | OK | Approved — locked in CLAUDE.md |
| `github.com/sergi/go-diff` | pkg.go.dev | v1.4.0 [CITED: pkg.go.dev] | High — Google's diff-match-patch Go port; used by protobuf, countless CLIs | github.com/sergi/go-diff | OK | Approved — locked in CLAUDE.md |
| `github.com/alecthomas/chroma/v2` | pkg.go.dev | v2.27.0 [CITED: pkg.go.dev] | Very high — used by Hugo, goldmark-highlighting, bat | github.com/alecthomas/chroma | OK | Approved — locked in CLAUDE.md |

**Packages removed due to SLOP verdict:** none
**Packages flagged as suspicious SUS:** none

---

## Architecture Patterns

### System Architecture Diagram

```
Fixture .diff file (or Phase 2 git subprocess output)
    │
    ▼ io.Reader
gitdiff.Parse(r)
    │  returns []*gitdiff.File + preamble
    │  each File: OldName, NewName, IsNew/IsDelete/IsRename/IsCopy
    │             IsBinary, BinaryFragment
    │             TextFragments []*TextFragment
    │             OldMode, NewMode (os.FileMode — 160000 = submodule)
    │
    ▼ []*gitdiff.File
diff.Align(file, mode)         ← mode = FullFile | HunkOnly
    │  walks TextFragment.Lines (OpContext / OpDelete / OpAdd)
    │  groups adjacent OpDelete+OpAdd sequences → LineModified pairs
    │  returns []RowPair{Left RenderedLine, Right RenderedLine}
    │
    ▼ []RowPair (with raw text content)
diff.Highlight(rowPairs, filename)
    │  lexer = lexers.Match(filename) || lexers.Fallback
    │  for each side: formatter.Format(&sb, style, lexer.Tokenise(content))
    │  splits by "\n" → per-line ANSI strings
    │  stores highlighted text in RowPair.{Left,Right}.Highlighted
    │
    ▼ []RowPair (with ANSI-highlighted content)
diff.Render(rowPairs, width int) []string
    │  for each pair:
    │    - apply line-level diff background color to left and right content
    │    - run intra-line diff on Modified pairs (with guards)
    │    - truncate left to (width/2 - 1) columns (ANSI-aware)
    │    - emit "\x1b[0m" at end of left column
    │    - join left + separator + right → single string
    │  returns []string (one element per row)
    │
    ▼ []string
Phase 3: viewport.SetContent(strings.Join(rows, "\n"))
```

### Recommended Project Structure

```
alturd/                         ← module root (go.mod lives here)
├── go.mod                      ← module github.com/alturd/alturd, go 1.22
├── go.sum
├── internal/
│   └── diff/
│       ├── model.go            ← LineKind, RenderedLine, RowPair, RenderMode
│       ├── parse.go            ← Parse(r io.Reader) ([]*gitdiff.File, error)
│       ├── align.go            ← Align(file *gitdiff.File, mode RenderMode) []RowPair
│       ├── highlight.go        ← Highlight(pairs []RowPair, filename string)
│       ├── render.go           ← Render(file *gitdiff.File, width int) []string
│       ├── parse_test.go       ← table tests for parsing edge cases
│       ├── align_test.go       ← table tests for alignment logic
│       ├── render_test.go      ← table tests for render output assertions
│       └── testdata/
│           ├── simple.diff     ← basic text change
│           ├── binary.diff     ← binary patch
│           ├── rename.diff     ← pure rename no content change
│           ├── mode-only.diff  ← mode change only
│           ├── submodule.diff  ← submodule bump
│           ├── no-newline.diff ← no newline at EOF
│           ├── new-file.diff   ← added file
│           ├── deleted-file.diff ← deleted file
│           ├── multi-file.diff ← two files in one diff
│           ├── multi-hunk.diff ← single file multiple hunks
│           ├── large-line.diff ← line > 1000 chars (guard test)
│           └── many-tokens.diff ← > 200 token line (guard test)
```

### Pattern 1: go-gitdiff Parse API

**What:** Call `gitdiff.Parse(r)` to get `[]*gitdiff.File`. The package uses `File` not `FileDiff`.

**When:** Entry point for all diff processing.

```go
// Source: pkg.go.dev/github.com/bluekeyes/go-gitdiff/gitdiff
import "github.com/bluekeyes/go-gitdiff/gitdiff"

func Parse(r io.Reader) ([]*gitdiff.File, error) {
    files, _, err := gitdiff.Parse(r)
    if err != nil {
        return nil, fmt.Errorf("parsing diff: %w", err)
    }
    return files, nil
}
```

**Key struct fields:**
```go
// gitdiff.File — one per changed file
type File struct {
    OldName  string
    NewName  string
    IsNew    bool          // added file
    IsDelete bool          // deleted file
    IsCopy   bool          // copied from another path
    IsRename bool          // renamed from another path
    OldMode  os.FileMode   // 0 if unchanged; 0160000 (57344) = submodule
    NewMode  os.FileMode
    Score    int           // rename/copy similarity (0-100)
    TextFragments []*gitdiff.TextFragment
    IsBinary bool
    BinaryFragment         *gitdiff.BinaryFragment
    ReverseBinaryFragment  *gitdiff.BinaryFragment
}

// gitdiff.Line — one per line in a TextFragment
type Line struct {
    Op   gitdiff.LineOp  // OpContext, OpDelete, OpAdd
    Line string           // content including trailing "\n" (or not, if NoEOL)
}

func (l Line) NoEOL() bool  // true when "\ No newline at end of file"
func (l Line) Old() bool    // appears on old/left side
func (l Line) New() bool    // appears on new/right side
```

### Pattern 2: Edge-Case Detection

**What:** Detect all special file types from the parsed `File` struct fields.

```go
// Source: pkg.go.dev/github.com/bluekeyes/go-gitdiff/gitdiff
func fileStatus(f *gitdiff.File) string {
    switch {
    case f.IsNew:
        return "[A]"
    case f.IsDelete:
        return "[D]"
    case f.IsRename:
        return "[R]"
    case f.IsCopy:
        return "[C]"
    case f.IsBinary:
        return "[B]"
    case f.OldMode != f.NewMode && f.OldMode != 0:
        return "[M]" // mode-only change
    default:
        return "[M]"
    }
}

// Submodule detection: mode 0160000 (octal) == os.FileMode(57344)
const submoduleMode = os.FileMode(0160000)

func isSubmodule(f *gitdiff.File) bool {
    return f.OldMode == submoduleMode || f.NewMode == submoduleMode
}

// No-newline-at-EOF detection: check the last line in each fragment
func hasNoEOL(f *gitdiff.TextFragment) bool {
    if len(f.Lines) == 0 {
        return false
    }
    return f.Lines[len(f.Lines)-1].NoEOL()
}
```

### Pattern 3: Alignment Algorithm

**What:** Walk `TextFragment.Lines` to produce `[]RowPair`, grouping adjacent `OpDelete`+`OpAdd` sequences into `LineModified` pairs.

**When:** After parsing, before highlighting.

```go
// Source: research synthesis [ASSUMED]
type LineKind int
const (
    LineContext      LineKind = iota
    LineAdded                 // new side only
    LineRemoved               // old side only
    LineModifiedOld           // part of delete+add pair (old side)
    LineModifiedNew           // part of delete+add pair (new side)
    LineBlank                 // alignment filler
)

type RenderedLine struct {
    Kind    LineKind
    Content string  // raw text (before highlighting)
    ANSI    string  // highlighted + diff-colored ANSI string (set later)
}

type RowPair struct {
    Left  RenderedLine
    Right RenderedLine
}

// align pairs adjacent delete+add lines as Modified; all others as one-sided
func alignLines(lines []gitdiff.Line) []RowPair {
    var result []RowPair
    i := 0
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
            // look ahead for matching OpAdd
            if i+1 < len(lines) && lines[i+1].Op == gitdiff.OpAdd {
                result = append(result, RowPair{
                    Left:  RenderedLine{Kind: LineModifiedOld, Content: line.Line},
                    Right: RenderedLine{Kind: LineModifiedNew, Content: lines[i+1].Line},
                })
                i += 2
            } else {
                result = append(result, RowPair{
                    Left:  RenderedLine{Kind: LineRemoved, Content: line.Line},
                    Right: RenderedLine{Kind: LineBlank, Content: ""},
                })
                i++
            }
        case gitdiff.OpAdd:
            result = append(result, RowPair{
                Left:  RenderedLine{Kind: LineBlank, Content: ""},
                Right: RenderedLine{Kind: LineAdded, Content: line.Line},
            })
            i++
        }
    }
    return result
}
```

**Note on multi-delete+multi-add runs:** The simple one-to-one pairing above works correctly when each delete is paired with exactly one add. For runs of multiple deletes followed by multiple adds, pair them positionally (first delete with first add, second with second, etc.) — this is the same strategy the Python implementation uses and is simpler than trying to compute the optimal pairing. [ASSUMED]

### Pattern 4: Intra-Line Character Diff

**What:** For `LineModifiedOld`+`LineModifiedNew` pairs, run `go-diff DiffMain` on the raw text to compute character-level change spans.

**When:** After alignment, before rendering, and only when guards permit.

```go
// Source: pkg.go.dev/github.com/sergi/go-diff/diffmatchpatch
import "github.com/sergi/go-diff/diffmatchpatch"

type IntraLineGuard struct {
    MaxChars  int           // 1000
    MaxTokens int           // 200
    MaxTime   time.Duration // 100ms
}

func computeIntraLine(oldRaw, newRaw string, guard IntraLineGuard) (oldSpans, newSpans []Span, skipped bool) {
    // Guard: line length
    if len(oldRaw) > guard.MaxChars || len(newRaw) > guard.MaxChars {
        return nil, nil, true
    }
    // Guard: token count (approximate via split on whitespace/punct)
    if countTokens(oldRaw) > guard.MaxTokens || countTokens(newRaw) > guard.MaxTokens {
        return nil, nil, true
    }
    // Guard: elapsed time
    start := time.Now()
    dmp := diffmatchpatch.New()
    diffs := dmp.DiffMain(oldRaw, newRaw, false) // checklines=false for char-level
    if time.Since(start) > guard.MaxTime {
        return nil, nil, true
    }
    // Walk diffs to extract spans for each side
    oldPos, newPos := 0, 0
    for _, d := range diffs {
        switch d.Type {
        case diffmatchpatch.DiffEqual:
            oldPos += len(d.Text)
            newPos += len(d.Text)
        case diffmatchpatch.DiffDelete:
            oldSpans = append(oldSpans, Span{Start: oldPos, End: oldPos + len(d.Text)})
            oldPos += len(d.Text)
        case diffmatchpatch.DiffInsert:
            newSpans = append(newSpans, Span{Start: newPos, End: newPos + len(d.Text)})
            newPos += len(d.Text)
        }
    }
    return oldSpans, newSpans, false
}
```

### Pattern 5: Chroma Syntax Highlighting Per-Line

**What:** Highlight one side's content with Chroma, write to a `strings.Builder`, then split on `"\n"` for per-line ANSI strings.

**When:** In `highlight.go` after alignment, before render.

```go
// Source: pkg.go.dev/github.com/alecthomas/chroma/v2
import (
    "github.com/alecthomas/chroma/v2/formatters"
    "github.com/alecthomas/chroma/v2/lexers"
    "github.com/alecthomas/chroma/v2/styles"
)

func highlightContent(content, filename string) ([]string, error) {
    lexer := lexers.Match(filename)
    if lexer == nil {
        lexer = lexers.Fallback
    }
    style := styles.Get("monokai")  // or from config in Phase 4
    if style == nil {
        style = styles.Fallback
    }
    formatter := formatters.Get("terminal16m") // true color; Phase 4 will auto-select
    if formatter == nil {
        formatter = formatters.Fallback
    }
    iterator, err := lexer.Tokenise(nil, content)
    if err != nil {
        return nil, err
    }
    var sb strings.Builder
    if err := formatter.Format(&sb, style, iterator); err != nil {
        return nil, err
    }
    // Split on newline to get per-line ANSI strings
    return strings.Split(sb.String(), "\n"), nil
}
```

### Pattern 6: ANSI Reset at Column Boundary

**What:** Emit `"\x1b[0m"` at the end of every left-column ANSI string before joining with the right column. This prevents chroma's foreground color state from bleeding into the right column's content.

**When:** In `render.go` when composing the final row string.

```go
// Source: research synthesis — ANSI SGR specification [ASSUMED]
const ansiReset = "\x1b[0m"

func joinColumns(left, right string, leftWidth, rightWidth int) string {
    // ANSI-aware truncation to column width (use lipgloss in Phase 3;
    // for Phase 1 tests use a fixed width that doesn't require truncation)
    paddedLeft := ansiPad(left, leftWidth)
    // CRITICAL: reset ANSI state at left column boundary
    return paddedLeft + ansiReset + " " + right
}
```

### Anti-Patterns to Avoid

- **Using `len(s)` or `len([]rune(s))` for column width:** ANSI escape sequences are invisible bytes that `len()` counts. Use `lipgloss.Width()` (Phase 3) or a dedicated ANSI-aware width function in Phase 1 tests. For Phase 1 tests with fixed widths, ensure test fixtures are short enough that truncation is not needed.

- **Running chroma tokenization inside the test assertion loop:** Tokenization is expensive. Pre-compute highlighted output in setup, store results, then assert on stored values.

- **Asserting exact ANSI string content:** Decision D-05 explicitly prohibits golden ANSI snapshots. Assert that ANSI escape codes are present (regex match `\x1b\[`), not what exact colors are used.

- **Misidentifying the go-gitdiff struct name:** The struct is `gitdiff.File`, not `FileDiff`. The parse function is `gitdiff.Parse()`, not `ParseMultiFileDiff()`.

- **Allocating a new `diffmatchpatch.DiffMatchPatch` per line:** `diffmatchpatch.New()` is cheap but idiomatic Go creates it once and reuses. Consider a package-level `var dmp = diffmatchpatch.New()`.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Unified diff parsing | Custom `--- +++ @@ hunk parser` | `github.com/bluekeyes/go-gitdiff v0.8.1` | Binary patches, renames, no-newline, extended headers — all typed fields; hand-rolled parsers miss all of these |
| Character-level LCS | Myers algorithm from scratch | `github.com/sergi/go-diff DiffMain` | Diff-match-patch has correctness, timeout, and boundary case handling built in; LCS is O(n²) in the worst case and needs line-end normalization |
| Syntax highlighting tokenization | Language detection regexes | `github.com/alecthomas/chroma/v2` | 200+ languages, correct handling of embedded languages (HTML in PHP, etc.); consistent with Python Pygments reference |
| ANSI escape sequence counting | Manual escape state machine | `lipgloss.Width()` or `runewidth.StringWidth()` (Phase 3+) | CJK double-width, combining marks, zero-width characters all break naive approaches |

**Key insight:** Every component listed above has been the subject of multiple serious bugs in hand-rolled implementations. The libraries exist specifically because getting these right requires weeks of edge-case work.

---

## Common Pitfalls

### Pitfall 1: ANSI Color Bleed Across Side-by-Side Columns

**What goes wrong:** The right column inherits the foreground/background color that chroma applied to the last token in the left column. All text in the right column appears in the wrong color.

**Why it happens:** Terminal ANSI SGR state is global within a rendered frame. The left column ends mid-color-sequence; the right column's text continues in that color.

**How to avoid:** Emit `"\x1b[0m"` at the end of every left-column ANSI string before appending the right column content. Do NOT rely on lipgloss or the TUI framework to insert this — the diff renderer owns it explicitly.

**Warning signs:** Right column text appears in a color matching the end of the last syntax-highlighted token on the left.

---

### Pitfall 2: Chroma Produces Multi-Line Tokens Across Line Boundaries

**What goes wrong:** When splitting chroma output on `"\n"`, some tokens span multiple lines. The resulting per-line slice may have incomplete ANSI sequences, causing the next line to inherit the previous line's unfinished color state.

**Why it happens:** Chroma's tokenizer (correctly) produces a multi-line token for multi-line string literals. The formatter writes a single ANSI sequence around the entire span. Splitting on `"\n"` cuts this mid-sequence.

**How to avoid:** After splitting chroma output on `"\n"`, append `"\x1b[0m"` to each line that does not already end with a reset. The formatter for `terminal16m` does not guarantee a reset at every `"\n"`. Alternatively, highlight line-by-line (call `Tokenise` per line), though this loses cross-line token context (affects multi-line strings). [ASSUMED — verify with a test using a multi-line string literal fixture]

**Warning signs:** Second and subsequent lines of a multi-line string literal in the left column appear with correct highlighting, but the right column immediately following is colored in the string literal color.

---

### Pitfall 3: go-diff DiffMain Checklines=True Produces Word-Level, Not Char-Level

**What goes wrong:** Passing `checklines=true` to `DiffMain` causes it to first diff on line boundaries (fast path) and then refine within lines. For our use case of comparing a single old line against a single new line, `checklines=true` has no benefit and may produce coarser spans than `checklines=false`.

**Why it happens:** `checklines=true` is the "high-performance mode" for large texts; it does a line-level diff first to quickly find matching regions, then refines. When the input is a single line (no `"\n"`), this heuristic degrades gracefully, but the API documentation is ambiguous. Decision D-06 explicitly requires `checklines=false`.

**How to avoid:** Always call `dmp.DiffMain(old, new, false)`. This is locked by D-06.

**Warning signs:** Intra-line markers span entire words rather than individual characters on simple substitutions like `"foo"` → `"fio"`.

---

### Pitfall 4: Mode-Only Diffs Have No TextFragments

**What goes wrong:** A file where only the mode changed (e.g., `chmod +x`) produces a `gitdiff.File` with `OldMode != NewMode` but `TextFragments` is empty (nil or zero-length). Code that iterates `file.TextFragments` without checking for empty produces correct output, but code that derives the rendered content from fragment count may skip the file entirely or render it as "no changes."

**Why it happens:** The go-gitdiff parser correctly produces a `File` struct with mode information but no line-level diff content (there is none). It's easy to conflate "no text fragments" with "skip this file."

**How to avoid:** Check `len(file.TextFragments) == 0` separately from `file.IsNew/IsDelete/IsBinary`. For mode-only changes, render a status line: `"Mode changed: 0644 → 0755"`.

**Warning signs:** Mode-only fixture test shows an empty rendered output instead of a status placeholder.

---

### Pitfall 5: Submodule Diffs Look Like Text Diffs

**What goes wrong:** A submodule bump diff has `OldMode == NewMode == os.FileMode(0160000)` and `TextFragments` containing lines like `-Subproject commit abc123` and `+Subproject commit def456`. If the renderer tries to syntax-highlight this as a source file (e.g., it matches no lexer), the output is correct but ugly. If it tries to apply intra-line diff between the two commit hash lines, the result is meaningless character-level markers.

**Why it happens:** go-gitdiff correctly parses submodule diffs as text fragments (the diff IS text). There is no explicit `IsSubmodule` flag.

**How to avoid:** Detect `OldMode == 0160000 || NewMode == 0160000` and skip syntax highlighting + intra-line diff for these files. Render a simple status placeholder: `"Submodule: abc123 → def456"` or pass the raw text content through without chroma.

**Warning signs:** Submodule fixture test shows diff markers inside the 40-char commit hash (intra-line diff on the SHA characters).

---

### Pitfall 6: Binary Diff Content Is Not Renderable as Text

**What goes wrong:** For `IsBinary == true` files, `BinaryFragment` contains base85-encoded data. Attempting to render this as a left/right column diff produces unreadable output or panics when the renderer tries to split on `"\n"`.

**Why it happens:** The renderer handles the "default" path (text content) first; binary files need an early-exit path.

**How to avoid:** Check `file.IsBinary` before entering the render pipeline and return a placeholder row: `"[Binary file — <N> bytes changed]"`.

**Warning signs:** Binary fixture test panics or produces garbled output.

---

### Pitfall 7: Fixture Files Must Be LF-Normalized

**What goes wrong:** If the `.diff` fixture files are committed with CRLF line endings (possible on Windows or if `.gitattributes` is not set), `gitdiff.Parse` may fail or produce lines with trailing `\r` in the `Line.Line` field. String comparisons in tests then fail intermittently depending on platform.

**Why it happens:** Git's core.autocrlf setting or a missing `.gitattributes` can convert line endings on checkout.

**How to avoid:** Add `.gitattributes` with `internal/diff/testdata/*.diff text eol=lf` before committing fixtures. Verify with `file testdata/*.diff | grep CRLF` returning no results.

**Warning signs:** Tests pass on Linux CI but fail on Windows CI with trailing `\r` in parsed line content.

---

## Code Examples

### Parsing a Diff File in a Test

```go
// Source: pkg.go.dev/github.com/bluekeyes/go-gitdiff/gitdiff + Go testing conventions
func TestParseSimpleDiff(t *testing.T) {
    f, err := os.Open("testdata/simple.diff")
    if err != nil {
        t.Fatalf("opening fixture: %v", err)
    }
    defer f.Close()

    files, err := diff.Parse(f)
    if err != nil {
        t.Fatalf("parsing diff: %v", err)
    }
    if len(files) != 1 {
        t.Errorf("expected 1 file, got %d", len(files))
    }
    file := files[0]
    if file.NewName != "README.md" {
        t.Errorf("expected NewName=README.md, got %q", file.NewName)
    }
    // Count added/removed/context lines
    var added, removed, ctx int
    for _, frag := range file.TextFragments {
        for _, line := range frag.Lines {
            switch line.Op {
            case gitdiff.OpAdd:    added++
            case gitdiff.OpDelete: removed++
            case gitdiff.OpContext: ctx++
            }
        }
    }
    // assert counts match expected values from the fixture
}
```

### Asserting ANSI Color Codes Are Present (Not Exact)

```go
// Source: Decision D-05 — structural assertions only
import "regexp"

var ansiColorRe = regexp.MustCompile(`\x1b\[[0-9;]+m`)

func TestHighlightApplied(t *testing.T) {
    rows := diff.Render(file, 160)
    // Find the first added/removed line in the output
    var coloredLine string
    for _, row := range rows {
        if strings.Contains(row, "some_function") { // known content from fixture
            coloredLine = row
            break
        }
    }
    if coloredLine == "" {
        t.Fatal("expected to find a line with 'some_function'")
    }
    if !ansiColorRe.MatchString(coloredLine) {
        t.Error("expected ANSI color codes in highlighted output, got none")
    }
}
```

### Binary File Render Placeholder

```go
// Source: research synthesis [ASSUMED]
func renderBinaryPlaceholder(f *gitdiff.File) []string {
    if f.BinaryFragment != nil {
        return []string{
            fmt.Sprintf("[Binary file: %d bytes]", f.BinaryFragment.Size),
        }
    }
    return []string{"[Binary file changed]"}
}
```

### Intra-Line Diff Guard Check

```go
// Source: Decision D-07 — guard thresholds
func shouldSkipIntraLine(oldLine, newLine string) bool {
    return len(oldLine) > 1000 || len(newLine) > 1000 ||
        countTokens(oldLine) > 200 || countTokens(newLine) > 200
}

// Time guard: run DiffMain under a context with 100ms deadline
func computeIntraLineWithTimeout(dmp *diffmatchpatch.DiffMatchPatch, old, new string) ([]diffmatchpatch.Diff, bool) {
    done := make(chan []diffmatchpatch.Diff, 1)
    go func() {
        done <- dmp.DiffMain(old, new, false)
    }()
    select {
    case diffs := <-done:
        return diffs, false
    case <-time.After(100 * time.Millisecond):
        return nil, true // timed out — skip intra-line
    }
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `ParseMultiFileDiff` (sourcegraph/go-diff) | `gitdiff.Parse()` (bluekeyes/go-gitdiff) | Library choice pre-locked | go-gitdiff handles binary patches and extended headers that sourcegraph/go-diff misses |
| `chroma/v2 v3.0.0-alpha.1` | `chroma/v2 v2.27.0` | v3 is alpha as of Jun 2026 | API stable; v3 changes `Iterator` to `iter.Seq` requiring Go 1.23 — not yet |
| Per-frame chroma tokenization | Pre-render on load | Research synthesis | Tokenization is expensive; do it once per file, not per render frame |

**Deprecated/outdated:**
- `github.com/sourcegraph/go-diff`: Does not handle binary patches or the `\ No newline at end of file` sentinel as a typed field — all of which appear in the Python fixture corpus. Do not use.
- `github.com/alecthomas/chroma/v3`: Alpha as of Jun 2026; API unstable. Use v2.27.0.
- `checklines=true` in `DiffMain`: Produces line-level diff, not character-level. Locked out by D-06.

---

## Open Questions

1. **Python fixture corpus availability**
   - What we know: The ROADMAP and CONTEXT.md reference "12+ Python fixture scenarios" from `tests/fixtures/diff/` in the Python repo.
   - What's unclear: The Python repo path is not checked into this Go repo. The fixtures must be copied (D-01) but the source location is not documented.
   - Recommendation: The planner must include a Wave 0 task: "Locate Python repo and copy fixture files, or create representative fixture files from scratch based on the 12 documented scenario types." If the Python repo is not accessible, all 12 scenario types are well-understood and can be hand-crafted from `git diff` output on a test repo.

2. **Full-file mode and diff context coverage**
   - What we know: DIFF-05 requires "entire file rendered with all unchanged lines shown." Phase 1 has no git subprocess (that's Phase 2).
   - What's unclear: How do Phase 1 tests verify full-file mode if the diff fixtures may only contain `-U3` context (3 lines)?
   - Recommendation: Fixture files for full-file mode tests should be generated with `git diff -U99999` to include all context. The planner should create fixtures with large `-U` values. Alternatively, the planner may decide that full-file mode in Phase 1 means "render all lines present in the diff output" (context + changed), which is testable without the original file. Clarify this with the user if needed.

3. **Chroma per-line multi-line token color bleed (Pitfall 2)**
   - What we know: Multi-line string literals produce tokens that span `"\n"` boundaries, which breaks per-line ANSI string splitting.
   - What's unclear: Whether `terminal16m` formatter emits a reset at every `"\n"` boundary (pkg.go.dev does not document this behavior).
   - Recommendation: The planner should include a task to write a micro-test with a two-line string literal fixture and verify the output. If color bleed occurs, the fix is to append `"\x1b[0m"` to each line after splitting.

4. **Module path for go.mod**
   - What we know: The binary is named `alturd`; the repo is `alturd/alturd` (or similar).
   - What's unclear: The exact GitHub org/repo name is not established.
   - Recommendation: Use `github.com/alturd/alturd` as a placeholder and update when the repo is published. This does not affect Phase 1 functionality.

---

## Environment Availability

> Phase 1 is code-only (no external services). Runtime dependencies are the Go toolchain and the three Go libraries installed via `go get`.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.22+ | Module build | [ASSUMED] — not verified in this session | Unknown | `go install golang.org/dl/go1.22@latest` if missing |
| git (CLI) | Fixture generation only | [ASSUMED] — standard dev tool | Unknown | Fixtures hand-crafted if git unavailable |

**Missing dependencies with no fallback:** None for Phase 1 execution (only code + fixtures needed).

**Verify Go version before starting:**
```bash
go version  # must be 1.22 or higher
```

---

## Security Domain

> `security_enforcement: true`, `security_asvs_level: 1`

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | Not applicable — library only, no user auth |
| V3 Session Management | No | Not applicable — stateless library |
| V4 Access Control | No | Not applicable — no access decisions |
| V5 Input Validation | Yes | Parse untrusted diff input without panic; go-gitdiff returns typed errors |
| V6 Cryptography | No | Not applicable — no cryptographic operations |

### Known Threat Patterns for Diff Parsing

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malformed diff causing panic | Tampering | go-gitdiff returns `error` on malformed input; wrap in `recover()` if used in a long-running server context; test with malformed fixture files |
| Extremely long lines exhausting memory | Denial of Service | The 1000-char intra-line guard limits go-diff memory usage on long lines; chroma has no built-in limit but tokenizes line-by-line |
| Binary diff with oversized payload | Denial of Service | Check `BinaryFragment.Size` before processing; `BinaryFragment.Data` contains raw bytes — do not attempt to render as text |
| Fixture file path traversal in tests | Tampering | Use `os.ReadFile("testdata/foo.diff")` with hardcoded relative paths only; never accept fixture paths from user input |

### Security Notes

Phase 1 is a library with no user-facing I/O, no network calls, and no file system writes. The primary security concern is that malformed diff input (e.g., from a malicious git repository) should not panic the application. Test malformed input in `parse_test.go` to ensure graceful error return.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Multi-delete+multi-add runs are paired positionally (first delete → first add) | Alignment algorithm | Incorrect intra-line diff pairing; fixable in align.go |
| A2 | `terminal16m` formatter does NOT emit `\x1b[0m` at every `\n` boundary | Pitfall 2 | If it does, the extra reset is harmless; if it doesn't, color bleed occurs on multi-line string literals |
| A3 | `\x1b[0m` is sufficient to reset all ANSI state at column boundary | ANSI reset pattern | Some terminal emulators with non-standard SGR stacks may not reset fully — use CSI `\x1b[m` (no params) as the most portable reset |
| A4 | Go module path `github.com/alturd/alturd` is acceptable placeholder | Pattern 1 | Module path change requires updating all internal imports — low cost but must be done before first external dependency |
| A5 | Full-file mode in Phase 1 tests can be satisfied by fixtures with large `-U` context | Open Question 2 | If the original file is required (not just the diff), the render API must accept it as an additional parameter — major design change |
| A6 | Submodule diffs have no `IsSubmodule` flag in go-gitdiff; detection via FileMode == 0160000 | Pattern 2 | If go-gitdiff adds explicit submodule detection, simpler check available; current approach is correct |

---

## Sources

### Primary (MEDIUM confidence)
- [pkg.go.dev: github.com/bluekeyes/go-gitdiff/gitdiff](https://pkg.go.dev/github.com/bluekeyes/go-gitdiff/gitdiff) — Full struct and function API, v0.8.1
- [pkg.go.dev: github.com/sergi/go-diff/diffmatchpatch](https://pkg.go.dev/github.com/sergi/go-diff/diffmatchpatch) — DiffMain signature, Diff struct, Operation constants, v1.4.0
- [pkg.go.dev: github.com/alecthomas/chroma/v2](https://pkg.go.dev/github.com/alecthomas/chroma/v2) — Lexer, formatter, style API, v2.27.0
- [pkg.go.dev: github.com/alecthomas/chroma/v2/formatters](https://pkg.go.dev/github.com/alecthomas/chroma/v2/formatters) — Formatter names (terminal, terminal8, terminal16, terminal256, terminal16m)

### Secondary (MEDIUM confidence)
- [CLAUDE.md §Technology Stack](/.claude/CLAUDE.md) — All Phase 1 library choices locked here; versions and rationale documented
- [.planning/research/STACK.md](/.planning/research/STACK.md) — Full stack with alternatives considered; researched 2026-06-25
- [.planning/research/ARCHITECTURE.md](/.planning/research/ARCHITECTURE.md) — Four-layer dependency graph; component responsibilities
- [.planning/research/PITFALLS.md](/.planning/research/PITFALLS.md) — ANSI color bleed, viewport ghost lines, Windows CRLF

### Tertiary (LOW confidence — marked [ASSUMED])
- ANSI SGR layering composition approach — from research synthesis; verify with multi-line string literal test
- Alignment algorithm for multi-delete+multi-add runs — Python parity assumed; verify against fixture corpus

---

## Metadata

**Confidence breakdown:**
- Standard stack: MEDIUM — all versions cited from pkg.go.dev; locked in CLAUDE.md
- go-gitdiff API: MEDIUM — struct fields and function signatures verified from pkg.go.dev
- go-diff API: MEDIUM — DiffMain signature and Operation constants verified from pkg.go.dev
- chroma API: MEDIUM — formatter and lexer API verified from pkg.go.dev
- Alignment algorithm: LOW — synthesized; reference implementation (Python) not inspected
- ANSI composition: LOW — requires empirical testing

**Research date:** 2026-06-26
**Valid until:** 2026-07-26 (stable libraries; 30-day horizon)

---

*Phase: 1 — Diff Model*
*Research completed: 2026-06-26*
