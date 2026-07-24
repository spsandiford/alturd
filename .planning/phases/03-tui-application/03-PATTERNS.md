# Phase 3: TUI Application - Pattern Map

**Mapped:** 2026-07-01
**Files analyzed:** 11 (3 new packages, 3 modified existing files, 5 test files)
**Analogs found:** 11 / 11 (all have either exact self-analogs or role-match analogs)

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `cmd/alturd/main.go` | entrypoint | request-response | itself (Phase 2) | self-modify |
| `internal/diff/render.go` | utility | transform | itself (Phase 2) | self-modify |
| `internal/diff/align.go` | utility | transform | itself (Phase 2) | self-modify |
| `internal/diff/render_test.go` | test | — | itself (Phase 2) | self-modify |
| `internal/diff/align_test.go` | test | — | itself (Phase 2) | self-modify |
| `internal/tui/model.go` | provider (tea.Model) | event-driven | `cmd/alturd/main.go` (wiring pattern) | partial-match |
| `internal/tui/tree.go` | utility | transform | `internal/diff/align.go` (pure transform) | role-match |
| `internal/tui/search.go` | utility | transform | `internal/diff/render.go` (string processing) | role-match |
| `internal/tui/model_test.go` | test | — | `internal/diff/align_test.go` | role-match |
| `internal/tui/tree_test.go` | test | — | `internal/diff/align_test.go` | role-match |
| `internal/tui/search_test.go` | test | — | `internal/diff/align_test.go` | role-match |

---

## Pattern Assignments

### `cmd/alturd/main.go` (entrypoint, request-response — MODIFY)

**Analog:** itself — current Phase 2 file

**Current imports pattern** (`cmd/alturd/main.go` lines 1-17):
```go
package main

import (
    "errors"
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "golang.org/x/term"

    "github.com/alturd/alturd/internal/diff"
    "github.com/alturd/alturd/internal/git"
    applog "github.com/alturd/alturd/internal/log"
)
```

**Phase 3 additional imports** (add to import block):
```go
tea "charm.land/bubbletea/v2"
"github.com/alturd/alturd/internal/tui"
```

**Current `run()` body to REPLACE** (`cmd/alturd/main.go` lines 41-80):

The section that runs git, parses diff, and renders stays — only the render loop
at lines 70-77 is replaced. Keep everything up to and including `diff.Parse()`.
Remove the `terminalWidth()` call and the `for _, file := range files` loop.

**Replacement block** (replaces lines 67-79 of current main.go):
```go
// Phase 3: launch bubbletea TUI in alternate screen mode (D-06).
// Data is pre-loaded; no async loading inside the model.
m := tui.NewModel(files)
p := tea.NewProgram(m, tea.WithAltScreen())
if _, err := p.Run(); err != nil {
    return err
}
return nil
```

**Keep unchanged:**
- `var version = "dev"` (line 21) — goreleaser ldflags target
- `rootCmd` definition (lines 25-36) — cobra config unchanged
- `applog.Init()` call (lines 43-46) — first statement in `run()`
- `gitArgs` construction (line 50)
- `git.ExecRunner{}.Run(gitArgs)` (line 53)
- `diff.Parse(reader)` (line 61)
- `main()` function (lines 97-107) — ExitCodeError handling unchanged

**Remove:**
- `terminalWidth()` function (lines 84-91) — no longer needed; terminal width comes from `tea.WindowSizeMsg`
- The `for _, file := range files` render loop (lines 70-77)

---

### `internal/diff/render.go` (utility, transform — MODIFY)

**Analog:** itself — current Phase 2 file

**Signature change** (`render.go` line 60):

Current:
```go
func Render(file *gitdiff.File, width int) []string {
```

Phase 3:
```go
func Render(file *gitdiff.File, width int, mode RenderMode) []string {
```

**Internal call change** (`render.go` line 68):

Current:
```go
pairs := Align(file, FullFile)
```

Phase 3:
```go
pairs := Align(file, mode)
```

No other changes to `render.go`. The file's existing patterns (package-level `dmp`, ANSI constants, pure functions, no global state) all carry forward unchanged.

---

### `internal/diff/align.go` (utility, transform — MODIFY)

**Analog:** itself — current Phase 2 file

**New function to ADD** (append after line 233, after the existing `alignText` function):

```go
// HunkStartRows returns the 0-based row indices (in the Render/Align output) where
// each TextFragment begins. Used by the TUI for hunk navigation (NAV-01).
//
// For edge-case files (binary, mode-only, submodule) there is always exactly one
// hunk at row 0 — the placeholder or raw-line output starts there.
//
// The mode parameter must match the mode passed to Render() so that row counts
// agree with the actual viewport content.
func HunkStartRows(file *gitdiff.File, mode RenderMode) []int {
    if file.IsBinary || isModeOnly(file) || isSubmodule(file) {
        return []int{0}
    }
    var starts []int
    row := 0
    for _, frag := range file.TextFragments {
        starts = append(starts, row)
        row += countFragmentRows(frag, mode)
    }
    return starts
}

// countFragmentRows returns the number of Align output rows produced by frag.
// The pairing logic mirrors alignText: delete+add runs produce max(dels,adds) rows.
func countFragmentRows(frag *gitdiff.TextFragment, mode RenderMode) int {
    count := 0
    lines := frag.Lines
    i := 0
    for i < len(lines) {
        switch lines[i].Op {
        case gitdiff.OpContext:
            if mode == FullFile || mode == HunkOnly {
                count++
            }
            i++
        case gitdiff.OpDelete:
            dels := 0
            for i < len(lines) && lines[i].Op == gitdiff.OpDelete {
                dels++
                i++
            }
            adds := 0
            for i < len(lines) && lines[i].Op == gitdiff.OpAdd {
                adds++
                i++
            }
            n := dels
            if adds > n {
                n = adds
            }
            count += n
        case gitdiff.OpAdd:
            count++
            i++
        }
    }
    return count
}
```

**Import addition required** — `gitdiff` is already imported in `align.go` (line 8). No new imports needed.

---

### `internal/diff/render_test.go` (test — MODIFY)

**Analog:** itself — current Phase 2 file

**Pattern change:** All calls to `diff.Render(file, width)` must become `diff.Render(file, width, diff.FullFile)`.

**Locate via:** `renderFile` helper at line 47-52 — this is the single call site:
```go
// Current (line 51):
rows := diff.Render(file, width)

// Phase 3:
rows := diff.Render(file, width, diff.FullFile)
```

Only `renderFile()` needs updating; it is the single call site for all `TestRender` subtests.

---

### `internal/diff/align_test.go` (test — MODIFY)

**Analog:** itself — current Phase 2 file

**New tests to ADD** (append after line 236, after `TestAlign`):

Follow the existing table-driven test style shown in `TestAlign`. Use `parseFirst()` helper already defined at lines 15-30. Pattern:

```go
func TestHunkStartRows(t *testing.T) {
    t.Run("simple_diff_one_fragment_at_row_zero", func(t *testing.T) {
        file := parseFirst(t, "simple.diff")
        rows := diff.HunkStartRows(file, diff.FullFile)
        if len(rows) == 0 {
            t.Fatal("HunkStartRows(simple.diff): got 0 entries, want >= 1")
        }
        if rows[0] != 0 {
            t.Errorf("HunkStartRows(simple.diff): rows[0] = %d, want 0", rows[0])
        }
    })

    t.Run("multi_hunk_ascending_rows", func(t *testing.T) {
        file := parseFirst(t, "multi-hunk.diff")
        rows := diff.HunkStartRows(file, diff.FullFile)
        for i := 1; i < len(rows); i++ {
            if rows[i] <= rows[i-1] {
                t.Errorf("HunkStartRows: rows[%d]=%d not > rows[%d]=%d (must be ascending)",
                    i, rows[i], i-1, rows[i-1])
            }
        }
    })

    t.Run("binary_always_row_zero", func(t *testing.T) {
        file := parseFirst(t, "binary.diff")
        rows := diff.HunkStartRows(file, diff.FullFile)
        if len(rows) != 1 || rows[0] != 0 {
            t.Errorf("HunkStartRows(binary.diff): got %v, want [0]", rows)
        }
    })

    t.Run("hunkonly_row_count_le_fullfile", func(t *testing.T) {
        file := parseFirst(t, "multi-hunk.diff")
        fullRows := diff.HunkStartRows(file, diff.FullFile)
        // HunkOnly doesn't reduce the number of fragments, just context rows within them.
        // The count of fragment starts must be the same; the row indices may differ.
        hunkRows := diff.HunkStartRows(file, diff.HunkOnly)
        if len(hunkRows) != len(fullRows) {
            t.Errorf("HunkStartRows fragment count: HunkOnly=%d FullFile=%d (must match)",
                len(hunkRows), len(fullRows))
        }
    })
}
```

---

### `internal/tui/model.go` (provider/tea.Model, event-driven — NEW)

**Analog:** `cmd/alturd/main.go` (wiring/init pattern) + RESEARCH.md Patterns 1–3, 7

**Package declaration pattern** (from all existing `internal/` packages):
```go
// Package tui implements the bubbletea v2 terminal UI for alturd.
// It owns all interactive state: pane layout, file selection, search mode,
// and hunk navigation. Data is pre-loaded by cmd/alturd/main.go before
// tea.NewProgram is called (D-06).
package tui
```

**Imports pattern** (derive from existing packages' import style):
```go
import (
    "fmt"
    "os"
    "runtime"
    "strings"
    "time"

    tea "charm.land/bubbletea/v2"
    "charm.land/bubbles/v2/textinput"
    "charm.land/bubbles/v2/viewport"
    "charm.land/lipgloss/v2"
    "github.com/bluekeyes/go-gitdiff/gitdiff"
    "golang.org/x/term"

    "github.com/alturd/alturd/internal/diff"
)
```

**Model struct pattern** (RESEARCH.md Pattern 1, lines 229-258):
```go
type pane int

const (
    treeFocused pane = iota
    diffFocused
)

const (
    treeWidthUnfocused = 24
    treeWidthFocused   = 45
    windowsPollInterval = time.Second / 4
)

type model struct {
    files       []*gitdiff.File

    ready       bool
    termWidth   int
    termHeight  int
    focusedPane pane
    treeWidth   int

    treeVP    viewport.Model
    treeNodes []*TreeNode
    treeFlat  []flatRow
    treeIdx   int
    allFiles  bool
    allFilePaths []string  // lazily populated on first 'a' press

    diffVP      viewport.Model
    currentFile int
    renderMode  diff.RenderMode
    hunkRows    []int
    currentHunk int

    searchMode    bool
    searchInput   textinput.Model
    searchMatches [][]int
}
```

**Constructor pattern** (RESEARCH.md Pattern 1; mirrors `cmd/alturd/main.go` `run()` init style):
```go
// NewModel creates the initial bubbletea model. files must be non-nil (may be empty).
// Called from cmd/alturd/main.go after git+parse complete (D-06).
func NewModel(files []*gitdiff.File) model {
    ti := textinput.New()
    ti.Prompt = "/"
    ti.Placeholder = "search..."

    statusMap := buildStatusMap(files)
    nodes := buildTree(filePaths(files), statusMap)
    flat := flattenTree(nodes, 0)

    return model{
        files:       files,
        treeWidth:   treeWidthUnfocused,
        searchInput: ti,
        treeNodes:   []*TreeNode{nodes},
        treeFlat:    flat,
    }
}
```

**Init pattern** (RESEARCH.md Pattern 7):
```go
func (m model) Init() tea.Cmd {
    if runtime.GOOS == "windows" {
        return tea.Tick(windowsPollInterval, func(_ time.Time) tea.Msg {
            return resizePollMsg{}
        })
    }
    return nil
}

type resizePollMsg struct{}
```

**View() pattern** (RESEARCH.md Pattern 3 — critical: returns `tea.View` not `string`):
```go
func (m model) View() tea.View {
    if !m.ready {
        return tea.NewView("") // D-07: blank until WindowSizeMsg
    }

    fileName := ""
    if len(m.files) > 0 {
        f := m.files[m.currentFile]
        if f.NewName != "" && f.NewName != "/dev/null" {
            fileName = f.NewName
        } else {
            fileName = f.OldName
        }
    }

    statusBar := fmt.Sprintf("alturd — %s (%d of %d changed files)",
        fileName, m.currentFile+1, len(m.files))
    if m.searchMode {
        statusBar += " [SEARCH]"
    }
    statusBar = lipgloss.NewStyle().Width(m.termWidth).Render(statusBar)

    treeStr := m.treeVP.View()
    diffStr  := m.diffVP.View()

    var searchBar string
    if m.searchMode {
        searchBar = "\n" + m.searchInput.View()
    }

    body := lipgloss.JoinHorizontal(
        lipgloss.Top,
        treeStr,
        "│",
        diffStr+searchBar,
    )

    return tea.NewView(statusBar + "\n" + body)
}
```

**Update() dispatch skeleton** (RESEARCH.md key dispatch table D-17; use `tea.KeyPressMsg` not `tea.KeyMsg`):
```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.handleResize(msg.Width, msg.Height)
        return m, nil

    case resizePollMsg:
        w, h, err := term.GetSize(int(os.Stdout.Fd()))
        if err == nil && (w != m.termWidth || h != m.termHeight) {
            m.handleResize(w, h)
        }
        if runtime.GOOS == "windows" {
            return m, tea.Tick(windowsPollInterval, func(_ time.Time) tea.Msg {
                return resizePollMsg{}
            })
        }
        return m, nil

    case tea.KeyPressMsg: // v2 type — NOT tea.KeyMsg
        return m.handleKey(msg)
    }

    if m.searchMode {
        var cmd tea.Cmd
        m.searchInput, cmd = m.searchInput.Update(msg)
        m.recomputeSearch()
        return m, cmd
    }

    return m, nil
}
```

**handleResize helper** (RESEARCH.md Pattern 2):
```go
func (m *model) handleResize(w, h int) {
    m.termWidth  = w
    m.termHeight = h
    m.ready      = true

    contentH := h - 1 // status bar
    if m.searchMode {
        contentH-- // search bar
    }
    diffW := w - m.treeWidth - 1 // D-03

    m.treeVP.SetWidth(m.treeWidth)
    m.treeVP.SetHeight(contentH)
    m.diffVP.SetWidth(diffW)
    m.diffVP.SetHeight(contentH)

    m.refreshDiffContent()
    m.refreshTreeContent()
}

func (m *model) refreshDiffContent() {
    if len(m.files) == 0 {
        return
    }
    diffW := m.termWidth - m.treeWidth - 1
    rows := diff.Render(m.files[m.currentFile], diffW, m.renderMode)
    m.hunkRows = diff.HunkStartRows(m.files[m.currentFile], m.renderMode)
    m.diffVP.SetContent(strings.Join(rows, "\n"))
}
```

**Key dispatch** (D-17/D-18; note `msg.Code` not string comparison for special keys):
```go
func (m model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
    if m.searchMode {
        switch msg.String() {
        case "esc":
            m.searchMode = false
            m.searchInput.Reset()
            m.searchMatches = nil
            m.diffVP.SetHighlights(nil)
            m.handleResize(m.termWidth, m.termHeight) // restore height
        case "n":
            m.diffVP.HighlightNext()
        case "N":
            m.diffVP.HighlightPrevious()
        case "]", "[":
            m.searchMode = false
            m.searchInput.Reset()
            m.handleFileCycle(msg.String() == "]")
        }
        return m, nil
    }

    switch msg.String() {
    case "q":
        return m, tea.Quit()
    case "Q":
        os.Exit(1)
    case "tab":
        m.toggleFocus()
    case "v":
        if m.renderMode == diff.FullFile {
            m.renderMode = diff.HunkOnly
        } else {
            m.renderMode = diff.FullFile
        }
        m.refreshDiffContent()
    case "n":
        m.hunkNext()
    case "N":
        m.hunkPrev()
    case "]":
        m.handleFileCycle(true)
    case "[":
        m.handleFileCycle(false)
    case "/":
        m.searchMode = true
        m.handleResize(m.termWidth, m.termHeight) // shrink viewport
        return m, m.searchInput.Focus()
    case "a":
        m.toggleAllFiles()
    }

    var cmd tea.Cmd
    m.diffVP, cmd = m.diffVP.Update(msg)
    return m, cmd
}
```

---

### `internal/tui/tree.go` (utility, transform — NEW)

**Analog:** `internal/diff/align.go` — same role: pure functions operating on domain types

**Package + imports pattern** (mirror `internal/diff/align.go` lines 1-9):
```go
package tui

import (
    "sort"
    "strings"

    "github.com/bluekeyes/go-gitdiff/gitdiff"
    "github.com/alturd/alturd/internal/diff"
)
```

**Core types** (RESEARCH.md Pattern 6):
```go
// TreeNode is one node in the file tree displayed in the left pane.
// A node may represent a single directory, a GitHub-style collapsed chain
// (e.g. "src/internal/diff"), or a file leaf.
type TreeNode struct {
    Name     string       // display name; may contain "/" for collapsed chains (D-09)
    Children []*TreeNode
    IsDir    bool
    Path     string       // full path; non-empty only for file leaves
    Status   string       // "[A]", "[M]", etc.; empty for unchanged files (D-11)
    expanded bool         // for expand/collapse interaction
}

// flatRow is one displayable row in the tree viewport after flattening.
type flatRow struct {
    node  *TreeNode
    depth int
}
```

**Tree builder functions** (RESEARCH.md Pattern 6):
```go
// buildTree builds a collapsed TreeNode hierarchy from a flat list of file paths.
// statusMap maps path → status marker (from diff.FileStatus); paths absent from
// statusMap are unchanged files (status = "").
func buildTree(paths []string, statusMap map[string]string) *TreeNode {
    root := &TreeNode{IsDir: true}
    for _, p := range paths {
        insertPath(root, strings.Split(p, "/"), p, statusMap[p])
    }
    collapseChain(root)
    sortNode(root)
    return root
}

func insertPath(node *TreeNode, parts []string, fullPath, status string) {
    if len(parts) == 0 {
        return
    }
    if len(parts) == 1 {
        // File leaf.
        node.Children = append(node.Children, &TreeNode{
            Name:   parts[0],
            Path:   fullPath,
            Status: status,
        })
        return
    }
    // Directory node — find or create.
    for _, c := range node.Children {
        if c.IsDir && c.Name == parts[0] {
            insertPath(c, parts[1:], fullPath, status)
            return
        }
    }
    child := &TreeNode{Name: parts[0], IsDir: true}
    node.Children = append(node.Children, child)
    insertPath(child, parts[1:], fullPath, status)
}

// collapseChain merges single-child directory chains into one node (D-09).
func collapseChain(node *TreeNode) {
    for _, c := range node.Children {
        collapseChain(c)
    }
    if node.IsDir && len(node.Children) == 1 && node.Children[0].IsDir {
        child := node.Children[0]
        node.Name = node.Name + "/" + child.Name
        node.Children = child.Children
        collapseChain(node) // may qualify again after merge
    }
}

// sortNode sorts children dirs-first, then files, each group alphabetical (TREE-01).
func sortNode(node *TreeNode) {
    sort.SliceStable(node.Children, func(i, j int) bool {
        a, b := node.Children[i], node.Children[j]
        if a.IsDir != b.IsDir {
            return a.IsDir // dirs first
        }
        return a.Name < b.Name
    })
    for _, c := range node.Children {
        sortNode(c)
    }
}

// flattenTree produces an ordered []flatRow for viewport rendering.
// Only expanded directories have their children included.
func flattenTree(node *TreeNode, depth int) []flatRow {
    var rows []flatRow
    for _, c := range node.Children {
        rows = append(rows, flatRow{node: c, depth: depth})
        if c.IsDir && c.expanded {
            rows = append(rows, flattenTree(c, depth+1)...)
        }
    }
    return rows
}
```

**Status map helper** (bridges `internal/diff.FileStatus` to the tree builder):
```go
// buildStatusMap builds a path→status map from the pre-loaded diff files.
// Used by NewModel and by the 'a' toggle to mark changed files in full-repo tree.
func buildStatusMap(files []*gitdiff.File) map[string]string {
    m := make(map[string]string, len(files))
    for _, f := range files {
        path := f.NewName
        if path == "" || path == "/dev/null" {
            path = f.OldName
        }
        m[path] = diff.FileStatus(f)
    }
    return m
}

// filePaths returns the display paths of all diff files, in order.
func filePaths(files []*gitdiff.File) []string {
    paths := make([]string, 0, len(files))
    for _, f := range files {
        p := f.NewName
        if p == "" || p == "/dev/null" {
            p = f.OldName
        }
        paths = append(paths, p)
    }
    return paths
}
```

---

### `internal/tui/search.go` (utility, transform — NEW)

**Analog:** `internal/diff/render.go` — same role: pure string-processing functions

**Package + imports pattern**:
```go
package tui

import (
    "strings"

    "github.com/charmbracelet/x/ansi"
)
```

**Core function** (RESEARCH.md Pattern 4):
```go
// findMatches returns plain-text (ANSI-stripped) byte positions of all
// non-overlapping occurrences of query in content.
//
// The returned positions are in the same coordinate space that
// viewport.SetHighlights() expects: character offsets in the ANSI-stripped
// string (Assumption A2). If query is empty, nil is returned.
func findMatches(content, query string) [][]int {
    if query == "" {
        return nil
    }
    plain := ansi.Strip(content)
    var matches [][]int
    for i := 0; i < len(plain); {
        j := strings.Index(plain[i:], query)
        if j < 0 {
            break
        }
        start := i + j
        end   := start + len(query)
        matches = append(matches, []int{start, end})
        i = end
    }
    return matches
}
```

**recomputeSearch helper** (called from `model.go`'s `Update()` when search input changes):
```go
// recomputeSearch recomputes m.searchMatches from the current searchInput value
// and feeds the results to the diff viewport. Called whenever the search query changes.
func (m *model) recomputeSearch() {
    query := m.searchInput.Value()
    content := m.diffVP.Content() // GetContent equivalent in bubbles v2
    m.searchMatches = findMatches(content, query)
    m.diffVP.SetHighlights(m.searchMatches)
}
```

---

### Test Files (NEW)

**Analog for all test files:** `internal/diff/align_test.go` — exact pattern match

**Package convention** (`align_test.go` line 1):
```go
package tui_test  // black-box external test package; mirrors diff_test pattern
```

**Test helper pattern** (mirrors `parseFirst()` in `align_test.go` lines 15-30):
```go
// newModelWith is the test-model factory: creates a model from synthetic files
// and immediately calls handleResize to set ready=true.
func newModelWith(t *testing.T, files []*gitdiff.File) model {
    t.Helper()
    m := NewModel(files)
    m.handleResize(200, 50)
    return m
}
```

**Model dispatch test pattern** (mirrors `TestAlign` subtests in `align_test.go`):
```go
func TestModel_NotReady(t *testing.T) {
    m := NewModel(nil)
    v := m.View()
    if string(v) != "" {
        t.Errorf("View() before WindowSizeMsg: got %q, want empty string (D-07)", string(v))
    }
}

func TestModel_Quit(t *testing.T) {
    m := newModelWith(t, nil)
    _, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyRune, Rune: 'q'})
    // tea.Quit() returns a non-nil Cmd; use cmd != nil as proxy
    if cmd == nil {
        t.Error("'q' key: expected non-nil Cmd (tea.Quit), got nil")
    }
}
```

**Tree test pattern** (mirrors table-driven subtests in `align_test.go`):
```go
func TestBuildTree(t *testing.T) {
    t.Run("dirs_before_files", func(t *testing.T) {
        paths := []string{"src/main.go", "src/internal/foo.go", "README.md"}
        root := buildTree(paths, nil)
        // First child of root should be a directory (src), not README.md
        ...
    })
    t.Run("collapse_chain", func(t *testing.T) {
        paths := []string{"a/b/c/file.go"}
        root := buildTree(paths, nil)
        // Root's single child should be "a/b/c" (collapsed), not "a"
        ...
    })
}
```

---

## Shared Patterns

### Package-Level Comments
**Source:** Every existing `internal/` package (`internal/diff/model.go` line 1-10, `internal/git/runner.go` line 1-10)
**Apply to:** `internal/tui/model.go`, `internal/tui/tree.go`, `internal/tui/search.go`

Pattern: `// Package X does Y. \n// \n// Key design decisions: ...`

### Import Path Convention
**Source:** `internal/diff/align.go` line 8, `cmd/alturd/main.go` lines 14-17
**Apply to:** All new tui package files

Pattern: stdlib first, blank line, third-party ordered by module path, blank line, project-internal imports last (`github.com/alturd/alturd/internal/...`).

### Pure Functions, No Global State
**Source:** `internal/diff/render.go` (exception: `var dmp` at line 15 — noted as intentional package-level reuse)
**Apply to:** `internal/tui/tree.go`, `internal/tui/search.go`

All exported functions in `tree.go` and `search.go` must be pure (no package-level state). Methods on `*model` may mutate model fields.

### Error Handling in `run()`
**Source:** `cmd/alturd/main.go` lines 53-65
**Apply to:** `cmd/alturd/main.go` Phase 3 modification

Pattern: return errors directly; let `main()` handle `*git.ExitCodeError` via `errors.As`. Do not call `os.Exit()` inside `run()` — only in `main()` and the `'Q'` key handler.

### Dependency Injection (Stateless Runners)
**Source:** `internal/git/runner.go` — `ExecRunner{}` is stateless struct
**Apply to:** `git ls-tree` call in `model.go` `toggleAllFiles()`

Pattern: `git.ExecRunner{}.Run([]string{"ls-tree", "-r", "--full-tree", "--name-only", "HEAD"})` — same pattern as the diff invocation in `main.go` line 53.

### `//nolint` Comments
**Source:** `internal/git/runner.go` line 45: `//nolint:gosec`
**Apply to:** Any `exec.Command` calls in tui package (unlikely; git calls go through ExecRunner)

### Test Fixture Access
**Source:** `internal/diff/align_test.go` lines 15-30 (`parseFirst` reads `testdata/` files)
**Apply to:** `internal/tui/tree_test.go`, `internal/tui/search_test.go`

For tui tests: construct `*gitdiff.File` structs inline rather than using fixture files, since tree/search logic does not need full parse. Use the pattern from `align_test.go` only when a real parse is needed.

---

## No Analog Found

None — all files have sufficient analogs (self-analogs for modifications; role-match analogs for new files). Where bubbletea v2 API specifics are needed, the RESEARCH.md Patterns 1–7 provide the code to copy directly.

---

## Key Anti-Patterns to Enforce (from RESEARCH.md)

| Anti-Pattern | Correct Pattern | Source |
|---|---|---|
| `tea.KeyMsg` (v1 type) | `tea.KeyPressMsg` (v2 type) | RESEARCH.md Pitfall 1 |
| `viewport.New(w, h)` (v1 constructor) | `viewport.New(viewport.WithWidth(w), viewport.WithHeight(h))` | RESEARCH.md Pitfall 2 |
| `View() string` return type | `View() tea.View` + `return tea.NewView(s)` | RESEARCH.md Pitfall 3 |
| Rendering before `WindowSizeMsg` | Guard with `if !m.ready { return tea.NewView("") }` | RESEARCH.md Pitfall 4 / D-07 |
| ANSI byte positions in `SetHighlights` | Strip ANSI first, use plain-text positions | RESEARCH.md Pitfall 5 |
| Calling `diff.Render()` on every `Update()` | Cache in viewport; re-render only on resize/mode/file change | RESEARCH.md Anti-patterns |
| `lipgloss.AdaptiveColor{}` | Not available in lipgloss v2; Phase 4 concern | RESEARCH.md State of the Art |
| Windows polling on all platforms | Wrap in `if runtime.GOOS == "windows"` | RESEARCH.md Pitfall 8 |

---

## Metadata

**Analog search scope:** `/src/alturd/cmd/`, `/src/alturd/internal/`
**Files scanned:** 18 Go files (all project files; project is pre-Phase 3 with no TUI yet)
**Pattern extraction date:** 2026-07-01
