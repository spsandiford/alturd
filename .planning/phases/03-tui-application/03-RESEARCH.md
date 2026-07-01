# Phase 3: TUI Application - Research

**Researched:** 2026-07-01
**Domain:** bubbletea v2 TUI, lipgloss v2 layout, bubbles v2 viewport/textinput, ANSI-aware search, git file-tree
**Confidence:** HIGH (stack decisions), MEDIUM (API details from pkg.go.dev), LOW (Windows polling workaround pattern)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Layout and Pane Structure**
- D-01: Two-pane split: file tree pane left, diff pane right, separated by a single Unicode vertical bar `│` (1 col). No lipgloss border on panes.
- D-02: Tree pane width: 24 cols unfocused, 45 cols focused. Instant resize — no animation state machine.
- D-03: `diff_width = terminal_width - tree_width - 1`. No minimum diff pane width enforcement.
- D-04: Filename truncation uses ellipsis at end via `lipgloss.Style.MaxWidth()`. Applied at 24-col unfocused width.
- D-05: 1-row status bar at top: `alturd — <filename> (<N> of <M> changed files)`. `[SEARCH]` appended when search is open.

**Startup and Initialization**
- D-06: `main.go` runs `internal/git` and `diff.Parse()` BEFORE `tea.NewProgram()`. Model initialized with complete `[]*gitdiff.File`. No async loading.
- D-07: `ready bool` field in model. `View()` returns empty string until first `tea.WindowSizeMsg` sets `ready = true`.
- D-08: Windows resize polling workaround (issue #1601) — researcher determines recommended approach.

**File Tree**
- D-09: GitHub-style path collapsing — single-child dir chains collapsed into one node.
- D-10: Dirs-first sort at each level. Collapsed chains shown with `▸`; expanded with `▾`.
- D-11: `a` toggles changed-files-only / full-repo tree. Full-repo tree via `git ls-tree -r HEAD`. Changed files retain `[A]/[M]/[D]/[R]` markers; unchanged files have no marker.
- D-12: Selected file row highlighted with inverted background.

**Search Mode**
- D-13: `/` opens 1-row text input bar at bottom of diff pane. Diff viewport shrinks by 1 row when search open. Uses `bubbles/textinput`.
- D-14: Search open: `n`/`N` = match nav. Search closed: `n`/`N` = hunk nav. Mode-based dispatch.
- D-15: `Esc` closes search: clears query, removes highlights, restores diff pane height. `n`/`N` returns to hunk nav.
- D-16: `]`/`[` while search open: cycle files + close search automatically. No carry-over of query.

**Navigation Key Dispatch**
- D-17: `Update()` branches on `searchMode`:
  - `searchMode=true`: `n`/`N`→match nav; `]`/`[`→cycle+close; `Esc`→close; other nav keys ignored.
  - `searchMode=false`: `n`/`N`→hunk nav; `]`/`[`→cycle files; `Tab`→toggle pane; `v`→toggle full-file/hunk-only; `q`→exit 0; `Q`→exit 1.
- D-18: `q` exits code 0; `Q` exits code 1. Both wired in Phase 3.

### Claude's Discretion

- Exact bubbletea model architecture (single top-level model vs. nested sub-models, message type design) — standard bubbletea patterns apply.
- ANSI-aware search scanner implementation details.
- Hunk-centering behavior (3 lines of context above/below is reasonable default).
- `bubbles/viewport` vs. custom viewport — use `bubbles/viewport`.
- Expand/collapse interaction for directory nodes in the tree.

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.

</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DIFF-06 | Toggle full-file/hunk-only with `v` key, no reload | `diff.Render()` signature extension to accept `RenderMode`; re-call with current mode |
| NAV-01 | `n`/`N` jumps between hunks; full-file mode hunks centered with surrounding context | New `diff.HunkStartRows()` function; viewport `SetYOffset` for centering |
| NAV-02 | `]`/`[` cycles between changed files in file tree | Model `currentFile int` field; re-render diff with new file |
| NAV-03 | `Tab` switches focus between file tree and diff panes | `focusedPane` toggle; instant `treeWidth` change; viewport resize |
| NAV-04 | `q` exits 0; `Q` exits 1 | `tea.Quit()` in Update(); `os.Exit(1)` for Q |
| TREE-01 | File tree with colored `[A]/[M]/[D]/[R]` markers, dirs-first, compact-folder | `diff.FileStatus()` already exists; custom `TreeNode` builder; viewport for tree pane |
| TREE-02 | Tree pane 24→45 cols on focus, instant resize | Width stored in model state; viewport `SetWidth()` on Tab |
| TREE-03 | `a` toggles changed-files-only/full-repo tree | `git ls-tree -r --name-only HEAD` via ExecRunner |
| SEARCH-01 | `/` search with ANSI-aware match highlighting; `n`/`N` between matches | `ansi.Strip()` + viewport `SetHighlights()`/`HighlightNext()`/`HighlightPrevious()` |

</phase_requirements>

---

## Summary

Phase 3 wires together the existing `internal/diff` library (Phase 1) and `internal/git` layer (Phase 2) into a full bubbletea v2 interactive terminal UI. The core technical challenge is composing two `bubbles/v2` viewport instances (tree + diff) with correct ANSI-aware content management, hunk-position tracking, and search highlighting, while handling platform-specific resize events on Windows.

The key discovery from research is that bubbles/v2's `viewport.SetHighlights()` accepts positions in plain-text (ANSI-stripped) coordinates — making the ANSI-aware search scanner a straightforward: strip ANSI via `ansi.Strip()`, find string matches in plain text, call `SetHighlights()` with those positions. The viewport handles the visual injection using `lipgloss.StyleRanges` internally.

The tree builder is pure Go — no library needed for the GitHub-style path collapsing algorithm. The algorithm is about 40 lines: build a raw trie from file paths, then recursively merge nodes that have exactly one child directory. The existing `diff.FileStatus()` function already produces the `[A]/[M]/[D]/[R]` markers TREE-01 requires.

Three changes to existing `internal/diff` code are required: (1) add a `mode RenderMode` parameter to `Render()`, (2) add a `HunkStartRows()` function for NAV-01, (3) update existing `render_test.go` callers for the signature change.

**Primary recommendation:** Single top-level bubbletea model (`internal/tui/model.go`), two viewport instances, mode-based key dispatch, pre-loaded data (D-06), and the built-in bubbles/v2 `SetHighlights` API for search.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Git data loading | `cmd/alturd/main.go` | `internal/git` | D-06: runs before bubbletea starts; no async cmd |
| Diff rendering | `internal/diff` | `internal/tui/model.go` | `Render()` is pure, re-called on resize or mode toggle |
| Hunk position extraction | `internal/diff` | — | Pure function over gitdiff.File; unit-testable |
| TUI state machine | `internal/tui/model.go` | — | Single bubbletea model owns all interaction state |
| Tree building / collapsing | `internal/tui/tree.go` | — | Pure Go; no library; deterministic, unit-testable |
| ANSI-aware search | `internal/tui/search.go` | `charm.land/bubbles/v2/viewport` | Strip ANSI for matching; viewport handles highlight display |
| Viewport layout | `charm.land/bubbles/v2/viewport` | `charm.land/lipgloss/v2` | Two viewport instances joined via `JoinHorizontal` |
| Search input | `charm.land/bubbles/v2/textinput` | `internal/tui/model.go` | `textinput.Model` owned by top-level model |
| Windows resize polling | `internal/tui/model.go` | `golang.org/x/term` | Platform-conditional tick-based poll |
| Full-repo file listing | `internal/git` (ExecRunner) | `internal/tui/tree.go` | `git ls-tree -r --name-only HEAD` via existing runner |
| Status bar | `internal/tui/model.go` `View()` | — | Simple string format; no state beyond existing model fields |

---

## Standard Stack

### Core (Phase 3 additions — not yet in go.mod)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `charm.land/bubbletea/v2` | v2.0.7 [VERIFIED: Go proxy, confirmed 2026-07-01] | TUI framework (Elm MVU) | CLAUDE.md mandated; Cursed Renderer; Feb 2026 production release |
| `charm.land/lipgloss/v2` | v2.0.4 [VERIFIED: Go proxy, confirmed 2026-07-01] | Terminal styling and layout | Ships with bubbletea v2; `JoinHorizontal`, `Style.MaxWidth` |
| `charm.land/bubbles/v2` | v2.1.0 [VERIFIED: Go proxy, confirmed 2026-07-01] | viewport + textinput components | Provides both pane components; built-in `SetHighlights` for search |

### Already in go.mod (indirect — now used directly)

| Library | Version | Purpose | Why Used |
|---------|---------|---------|----------|
| `github.com/charmbracelet/x/ansi` | v0.8.0 [VERIFIED: go.mod] | ANSI stripping for search | `ansi.Strip()` needed for plain-text search coordinate extraction |
| `golang.org/x/term` | v0.44.0 [VERIFIED: go.mod] | Terminal size query | Windows resize polling via `term.GetSize()` |

### Version Verification

```bash
# Verified against Go module proxy (2026-07-01):
# charm.land/bubbletea/v2  → v2.0.7 (2026-06-01), source: github.com/charmbracelet/bubbletea
# charm.land/bubbles/v2    → v2.1.0 (2026-03-25), source: github.com/charmbracelet/bubbles
# charm.land/lipgloss/v2   → v2.0.4 (2026-06-12), source: github.com/charmbracelet/lipgloss
```

**Installation:**
```bash
go get charm.land/bubbletea/v2@v2.0.7
go get charm.land/lipgloss/v2@v2.0.4
go get charm.land/bubbles/v2@v2.1.0
```

Note: `charmbracelet/lipgloss v1.1.0` remains in go.mod as an indirect dependency of `charmbracelet/log v1.0.0`. The v1 and v2 lipgloss modules coexist without conflict (different module paths).

---

## Package Legitimacy Audit

> The npm legitimacy checker cannot assess Go module packages (vanity import paths like `charm.land/...` do not exist on the npm registry). Go packages are verified via the Go module proxy and authoritative CLAUDE.md project instructions.

| Package | Registry | Verified | Source Repository | Verdict | Disposition |
|---------|----------|----------|-------------------|---------|-------------|
| `charm.land/bubbletea/v2` | Go proxy | v2.0.7, 2026-06-01 | github.com/charmbracelet/bubbletea | OK | Approved — mandated in CLAUDE.md |
| `charm.land/lipgloss/v2` | Go proxy | v2.0.4, 2026-06-12 | github.com/charmbracelet/lipgloss | OK | Approved — mandated in CLAUDE.md |
| `charm.land/bubbles/v2` | Go proxy | v2.1.0, 2026-03-25 | github.com/charmbracelet/bubbles | OK | Approved — mandated in CLAUDE.md |
| `github.com/charmbracelet/x/ansi` | Go proxy | v0.8.0 | github.com/charmbracelet/x | OK | Already in go.mod; same Charmbracelet team |

**Packages removed due to SLOP verdict:** none
**Packages flagged as SUS:** none (npm checker inapplicable to Go modules; all verified via Go proxy)

---

## Architecture Patterns

### System Architecture Diagram

```
 git repo on disk
        │
        ▼
 [cmd/alturd/main.go]
  ├── ExecRunner.Run("diff") → diff.Parse() → []*gitdiff.File
  ├── ExecRunner.Run("ls-tree") → full-repo paths (when 'a' pressed)
  └── tea.NewProgram(newModel(files), tea.WithAltScreen()).Run()
                │
                ▼
       [internal/tui/model.go]  ←── tea.WindowSizeMsg (resize or Windows tick)
        │
        ├── View()
        │    ├── status bar (1 row, lipgloss string)
        │    └── lipgloss.JoinHorizontal(Top, treeStr, "│", diffStr)
        │         ├── treeVP.View()  ← treeVP.SetContent(renderTree(nodes))
        │         └── diffVP.View()  ← diffVP.SetContent(diff.Render(file, diffWidth, mode))
        │                              diffVP.SetHighlights(searchMatches) when search active
        │
        └── Update(msg tea.Msg)
             ├── tea.KeyPressMsg → dispatch on searchMode+focused pane
             │    ├── '/': open search, textinput.Focus()
             │    ├── 'n'/'N' (searchMode): diffVP.HighlightNext/Previous()
             │    ├── 'n'/'N' (hunk nav): diffVP.SetYOffset(hunkRows[i] - halfHeight)
             │    ├── ']'/'[': currentFile±1, re-render diff, recompute hunkRows
             │    ├── 'Tab': toggle focusedPane, instant treeWidth change, resize viewports
             │    ├── 'v': toggle renderMode, re-call diff.Render(), re-feed diffVP
             │    ├── 'q': tea.Quit()
             │    └── 'Q': os.Exit(1)
             ├── tea.WindowSizeMsg → set termWidth/termHeight, set ready=true, resize viewports
             ├── tickMsg (Windows) → poll term.GetSize(), emit synthetic WindowSizeMsg if changed
             └── textinput msgs (search open) → update searchInput, recompute matches
```

### Recommended Project Structure

```
cmd/alturd/
└── main.go              # replace stdout loop with tea.NewProgram(...)

internal/
├── diff/
│   ├── model.go         # existing — RowPair, RenderMode, LineKind (no change)
│   ├── align.go         # extend: add HunkStartRows() function
│   ├── render.go        # extend: add mode RenderMode param to Render()
│   ├── highlight.go     # existing — no change
│   └── parse.go         # existing — no change
├── git/
│   └── runner.go        # extend: add ls-tree invocation helper OR call directly
├── log/
│   └── log.go           # existing — no change
└── tui/
    ├── model.go         # top-level tea.Model: state, Init, Update, View
    ├── tree.go          # TreeNode, buildTree, collapseChain, renderTree, flattenTree
    └── search.go        # ANSI-aware match finder: ansi.Strip + match, returns [][]int
```

### Pattern 1: Top-Level Model Initialization

```go
// Source: bubbletea v2 Elm Architecture pattern (pkg.go.dev/charm.land/bubbletea/v2)
type model struct {
    // Pre-loaded data (D-06)
    files       []*gitdiff.File

    // Layout state
    ready       bool
    termWidth   int
    termHeight  int
    focusedPane pane // treeFocused | diffFocused
    treeWidth   int  // 24 or 45

    // Tree pane
    treeVP      viewport.Model
    treeNodes   []*TreeNode  // collapsed tree nodes
    treeFlat    []flatRow    // flattened for rendering
    treeIdx     int          // selected row
    allFiles    bool         // D-11 toggle

    // Diff pane
    diffVP      viewport.Model
    currentFile int
    renderMode  diff.RenderMode
    hunkRows    []int  // row indices of hunk starts in diffVP content
    currentHunk int

    // Search
    searchMode    bool
    searchInput   textinput.Model
    searchMatches [][]int  // plain-text position pairs for SetHighlights
}

func newModel(files []*gitdiff.File) model {
    ti := textinput.New()
    ti.Prompt = "/"
    ti.Placeholder = "search..."
    return model{
        files:     files,
        treeWidth: 24,
        searchInput: ti,
    }
}

func (m model) Init() tea.Cmd {
    if runtime.GOOS == "windows" {
        return tea.Tick(time.Second/4, func(t time.Time) tea.Msg {
            return resizePollMsg{}
        })
    }
    return nil
}
```

### Pattern 2: Viewport Construction and Layout

```go
// Source: pkg.go.dev/charm.land/bubbles/v2/viewport
// Called when tea.WindowSizeMsg arrives (first resize sets ready=true)
func (m *model) handleResize(w, h int) {
    m.termWidth = w
    m.termHeight = h
    m.ready = true

    contentH := h - 1 // minus 1-row status bar
    if m.searchMode {
        contentH-- // minus 1-row search bar
    }

    diffW := w - m.treeWidth - 1 // D-03

    m.treeVP.SetWidth(m.treeWidth)
    m.treeVP.SetHeight(contentH)
    m.diffVP.SetWidth(diffW)
    m.diffVP.SetHeight(contentH)

    // Re-render diff with new width
    m.refreshDiffContent()
}

func (m *model) refreshDiffContent() {
    if len(m.files) == 0 {
        return
    }
    diffW := m.termWidth - m.treeWidth - 1
    rows := diff.Render(m.files[m.currentFile], diffW, m.renderMode) // Phase 3 signature
    m.hunkRows = diff.HunkStartRows(m.files[m.currentFile], m.renderMode)
    m.diffVP.SetContent(strings.Join(rows, "\n"))
}
```

### Pattern 3: Layout Rendering

```go
// Source: pkg.go.dev/charm.land/lipgloss/v2
func (m model) View() tea.View {
    if !m.ready {
        return tea.NewView("")
    }

    statusBar := fmt.Sprintf("alturd — %s (%d of %d changed files)",
        currentFileName(m), m.currentFile+1, len(m.files))
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
        diffStr + searchBar,
    )

    return tea.NewView(statusBar + "\n" + body)
}
```

### Pattern 4: ANSI-Aware Search

```go
// Source: pkg.go.dev/github.com/charmbracelet/x/ansi#Strip
// Source: pkg.go.dev/charm.land/bubbles/v2/viewport — SetHighlights uses plain-text positions
func findMatches(content, query string) [][]int {
    if query == "" {
        return nil
    }
    plain := ansi.Strip(content) // removes all ANSI escape sequences
    var matches [][]int
    for i := 0; ; {
        j := strings.Index(plain[i:], query)
        if j < 0 {
            break
        }
        start := i + j
        matches = append(matches, []int{start, start + len(query)})
        i = start + len(query)
    }
    return matches
}

// Called in Update when search query changes:
// m.searchMatches = findMatches(m.diffVP.GetContent(), m.searchInput.Value())
// m.diffVP.SetHighlights(m.searchMatches)  ← SetHighlights accepts plain-text positions
// m.diffVP.HighlightNext() / m.diffVP.HighlightPrevious() for n/N
```

### Pattern 5: Hunk Navigation

```go
// New function to add to internal/diff/align.go
// Returns 0-based row indices in the Align output where each TextFragment begins.
func HunkStartRows(file *gitdiff.File, mode RenderMode) []int {
    // For edge-case files (binary, mode-only, submodule), there is one hunk at row 0
    if file.IsBinary || isModeOnly(file) || isSubmodule(file) {
        return []int{0}
    }
    var starts []int
    row := 0
    for _, frag := range file.TextFragments {
        starts = append(starts, row)
        // Count rows this fragment produces (same logic as alignText)
        row += countFragmentRows(frag)
    }
    return starts
}

// Centering in Update() for 'n':
// targetOffset := max(0, m.hunkRows[m.currentHunk] - m.diffVP.Height()/2)
// m.diffVP.SetYOffset(targetOffset)
```

### Pattern 6: Tree Builder

```go
// Source: [ASSUMED] — standard trie + collapse algorithm; no library needed
type TreeNode struct {
    Name     string      // display name; may be "src/internal" for collapsed chain
    Children []*TreeNode
    IsDir    bool
    Path     string      // full path for file leaf nodes
    Status   string      // "[A]", "[M]", etc.; empty for unchanged files in full-repo mode
}

func buildTree(paths []string, statusMap map[string]string) *TreeNode {
    root := &TreeNode{IsDir: true}
    for _, p := range paths {
        insertPath(root, strings.Split(p, "/"), p, statusMap[p])
    }
    collapseChain(root)
    sortNode(root) // dirs first, then files, each group alphabetical
    return root
}

func collapseChain(node *TreeNode) {
    for _, child := range node.Children {
        collapseChain(child) // bottom-up
    }
    // Collapse if this dir has exactly one child that is also a dir
    if node.IsDir && len(node.Children) == 1 && node.Children[0].IsDir {
        child := node.Children[0]
        node.Name = node.Name + "/" + child.Name
        node.Children = child.Children
        // recurse: the merged node might still qualify for further collapse
        collapseChain(node)
    }
}
```

### Pattern 7: Windows Resize Polling

```go
// Source: [ASSUMED] — community workaround for bubbletea v2 issue #1601
// Only compiled/activated on Windows; golang.org/x/term is already in go.mod.
type resizePollMsg struct{}

func (m model) Init() tea.Cmd {
    if runtime.GOOS == "windows" {
        return tea.Tick(time.Second/4, func(_ time.Time) tea.Msg {
            return resizePollMsg{}
        })
    }
    return nil
}

// In Update():
case resizePollMsg:
    w, h, err := term.GetSize(int(os.Stdout.Fd()))
    if err == nil && (w != m.termWidth || h != m.termHeight) {
        m.handleResize(w, h)
    }
    return m, tea.Tick(time.Second/4, func(_ time.Time) tea.Msg {
        return resizePollMsg{}
    })
```

### Anti-Patterns to Avoid

- **Starting bubbletea before loading data:** D-06 mandates git+parse runs in `run()` before `tea.NewProgram()`. Never use a `tea.Cmd` for the initial data load — this avoids the complexity of loading states and nil-file guards throughout the model.
- **Calling `diff.Render()` on every `Update()` tick:** Render only when `renderMode` changes, a new file is selected, or terminal width changes. Cache the result in a model field or viewport content.
- **Using `tea.KeyMsg` (bubbletea v1):** In v2 it is `tea.KeyPressMsg`. Using the v1 type produces a type mismatch; no compiler error because `tea.Msg` is `interface{}`.
- **Using viewport v1 constructor `viewport.New(w, h int)`:** v2 uses `viewport.New(viewport.WithWidth(w), viewport.WithHeight(h))`. Fields `Width` and `Height` are now setter/getter methods.
- **Setting `HighPerformanceRendering`:** Removed in v2. No-op at best; compile error more likely.
- **Passing ANSI byte positions to `SetHighlights()`:** The viewport's `SetHighlights()` uses plain-text character positions (after ANSI strip). Always call `ansi.Strip(content)`, find matches there, and pass those positions.
- **Blocking `View()` on data loading:** `View()` must be instant; pre-compute `hunkRows` and tree flat list in `Update()`, not in `View()`.
- **Rendering viewport before `WindowSizeMsg`:** Guard with `ready bool`; return `tea.NewView("")` until dimensions are known (D-07).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Scrollable text panes | Custom scroll buffer | `charm.land/bubbles/v2/viewport` | Handles scroll position, resize, FillHeight, gutter — all edge-cases covered |
| Search text input | Custom key handler | `charm.land/bubbles/v2/textinput` | Cursor, backspace, paste, width truncation — platform quirks handled |
| Search match highlighting | Custom ANSI injection | `viewport.SetHighlights()` + `HighlightNext()` | Built-in: applies `HighlightStyle` via `lipgloss.StyleRanges`; navigates with `EnsureVisible` |
| ANSI-aware string width | Manual byte counting | `ansi.StringWidth()` from `charmbracelet/x/ansi` | Already handles multi-byte UTF-8, wide characters, ANSI escape sequences |
| ANSI stripping | Regex | `ansi.Strip()` from `charmbracelet/x/ansi` | Correct CSI, OSC, DCS sequence handling; already in go.mod |
| Side-by-side layout | Manual ANSI column math | `lipgloss.JoinHorizontal()` | Handles multi-line string alignment, fills missing lines with spaces |
| Filename truncation | String slicing | `lipgloss.NewStyle().MaxWidth(n).Render(s)` | ANSI-aware; works correctly with non-ASCII filenames |

**Key insight:** Every problem in this list has an edge-case tail that makes custom implementations fail in practice (Unicode, ANSI, resize events, cursor blink). The Charm ecosystem covers all of them.

---

## Existing Code Integration Points

### Functions Consumed by Phase 3 (no changes to existing signature)

| Function | Location | Phase 3 Use |
|----------|----------|-------------|
| `diff.FileStatus(f)` | `internal/diff/align.go` | TREE-01 `[A]/[M]/[D]/[R]` markers — already correct |
| `diff.Parse(reader)` | `internal/diff/parse.go` | Called in `main.go` before bubbletea starts |
| `git.ParseRefArgs(args, lenAtDash)` | `internal/git/args.go` | Unchanged; still called in `main.go` |
| `git.ExecRunner{}.Run(args)` | `internal/git/runner.go` | Used for `git diff` AND `git ls-tree` in Phase 3 |
| `applog.Init()` | `internal/log/log.go` | First call in `run()`; unchanged |
| `diff.RenderMode` (FullFile / HunkOnly) | `internal/diff/model.go` | Toggle via `v` key; already defined |

### Functions Requiring Signature Changes

| Function | Current Signature | Phase 3 Signature | Reason |
|----------|------------------|-------------------|--------|
| `diff.Render()` | `Render(file, width int) []string` | `Render(file, width int, mode RenderMode) []string` | `v` key needs to pass current mode |
| Internal call in `Render()` | `Align(file, FullFile)` | `Align(file, mode)` | Pass through the mode parameter |

### Functions to Add

| Function | Location | Purpose |
|----------|----------|---------|
| `diff.HunkStartRows(file, mode RenderMode) []int` | `internal/diff/align.go` | Returns 0-based row indices of hunk starts for NAV-01 |
| `tui.newModel(files)` | `internal/tui/model.go` | Constructor; initializes viewports + textinput |
| `tui.buildTree(paths, statusMap)` | `internal/tui/tree.go` | Builds + collapses TreeNode hierarchy |
| `tui.flattenTree(root)` | `internal/tui/tree.go` | Produces `[]flatRow` for viewport rendering |
| `tui.findMatches(content, query)` | `internal/tui/search.go` | Strips ANSI, returns `[][]int` plain-text positions |

### Files Replacing Existing Behavior

`cmd/alturd/main.go` — the `run()` function body changes from:
```go
// Phase 2: stdout render loop (REPLACED by Phase 3)
for _, file := range files {
    rows := diff.Render(file, width)
    for _, row := range rows { fmt.Fprintln(os.Stdout, row) }
}
```
to:
```go
// Phase 3: bubbletea TUI
m := tui.NewModel(files)
p := tea.NewProgram(m, tea.WithAltScreen())
if _, err := p.Run(); err != nil {
    return err
}
```

---

## Common Pitfalls

### Pitfall 1: bubbletea v1 vs v2 Key Type

**What goes wrong:** `case tea.KeyMsg:` matches nothing in v2; all key handling is silently skipped. **Why it happens:** v2 renames `tea.KeyMsg` to `tea.KeyPressMsg`. **How to avoid:** Use `case tea.KeyPressMsg:` everywhere. Use `msg.Code == tea.KeyEscape` not string comparisons. **Warning signs:** Key presses produce no model updates; UI appears frozen to keyboard.

### Pitfall 2: viewport Constructor Changed in v2

**What goes wrong:** `viewport.New(80, 24)` compiles but panics at runtime or produces incorrect dimensions. **Why it happens:** The v1 constructor signature accepted `(width, height int)`; v2 uses functional options. **How to avoid:** Use `viewport.New(viewport.WithWidth(w), viewport.WithHeight(h))` or set after construction via `SetWidth()`/`SetHeight()`. **Warning signs:** Viewport renders at wrong size.

### Pitfall 3: `View()` Must Return `tea.View`, Not `string`

**What goes wrong:** Compile error if `View()` returns `string` — breaks `tea.Model` interface. **Why it happens:** bubbletea v2 changed `View() string` to `View() tea.View` (a distinct struct type). **How to avoid:** Use `return tea.NewView(s)` where `s string` is the rendered UI string. **Warning signs:** Compile error: `cannot use string as tea.View`.

### Pitfall 4: Rendering Before `WindowSizeMsg` Arrives

**What goes wrong:** `View()` renders with zero-width viewports on the first frame, producing corrupted output. **Why it happens:** bubbletea delivers `WindowSizeMsg` asynchronously after program start. **How to avoid:** Check `m.ready` at top of `View()`; return `tea.NewView("")` until `ready = true`. Set `ready = true` in `Update()` when first `WindowSizeMsg` is handled. (D-07)

### Pitfall 5: ANSI Byte Positions vs Plain-Text Positions in SetHighlights

**What goes wrong:** `viewport.SetHighlights(positions)` highlights wrong characters or produces garbled output. **Why it happens:** Positions must be in plain-text (ANSI-stripped) character space; if you pass byte offsets from the ANSI-escaped content, positions are off by the length of all preceding escape sequences. **How to avoid:** Always call `ansi.Strip(content)` first, find matches in the stripped text, pass those positions to `SetHighlights()`. **Warning signs:** Highlights appear at wrong locations in the pane.

### Pitfall 6: Forgetting to Update Hunk Rows on Mode Toggle or File Change

**What goes wrong:** `n`/`N` jumps to wrong hunk row after toggling `v` or cycling files. **Why it happens:** `hunkRows []int` must be recomputed whenever `renderMode` changes or `currentFile` changes — not just on resize. **How to avoid:** Centralize hunk row recomputation in a `refreshDiffContent()` helper that is always called when file or mode changes.

### Pitfall 7: Search Bar Height Not Subtracted from Viewport

**What goes wrong:** Search bar overlaps the last line of the diff pane content. **Why it happens:** When `searchMode = true`, the diff viewport must be `termHeight - 1 (status) - 1 (search) = termHeight - 2` rows tall. **How to avoid:** Always compute `contentH := termHeight - 1; if m.searchMode { contentH-- }` and call `m.diffVP.SetHeight(contentH)` whenever searchMode toggles.

### Pitfall 8: Windows Resize Polling Runs on All Platforms

**What goes wrong:** Unnecessary CPU usage on macOS/Linux (SIGWINCH already works there). **Why it happens:** Polling tick re-queues itself unconditionally. **How to avoid:** Wrap the polling tick in `if runtime.GOOS == "windows"`. The `resizePollMsg` handler should also guard the re-queue.

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `tea.KeyMsg` for key events | `tea.KeyPressMsg` | bubbletea v2 (Feb 2026) | All key dispatch code must use the new type |
| `viewport.New(w, h int)` | `viewport.New(viewport.WithWidth(w), viewport.WithHeight(h))` | bubbles v2 (Feb 2026) | Constructor call syntax change |
| `View() string` | `View() tea.View` + `tea.NewView(s)` | bubbletea v2 (Feb 2026) | Interface method return type changed |
| `HighPerformanceRendering` field | Removed — Cursed Renderer is default | bubbletea v2 (Feb 2026) | Do not set this field; it may cause compile error |
| `vp.Width` field (v1) | `vp.Width()` / `vp.SetWidth(w)` methods | bubbles v2 (Feb 2026) | Exported fields replaced with getter/setter |
| `textinput.NewModel()` | `textinput.New()` | bubbles v2 (Feb 2026) | Constructor alias removed |
| `textinput.DefaultKeyMap` (variable) | `textinput.DefaultKeyMap()` (function) | bubbles v2 (Feb 2026) | Variable became function |
| No built-in search highlighting | `SetHighlights()` / `HighlightNext()` / `HighlightPrevious()` | bubbles v2 (Mar 2026) | Eliminates need for custom highlight injection |
| `lipgloss.AdaptiveColor{}` | Removed — explicit theme selection required | lipgloss v2 (Feb 2026) | Phase 4 concern; Phase 3 uses hardcoded colors |

**Deprecated/outdated:**
- `github.com/charmbracelet/bubbletea` (v1): receives bugfixes only; import path differs from v2; use `charm.land/bubbletea/v2`.
- `HighPerformanceRendering`: removed; do not reference.

---

## Windows Resize Polling Detail (D-08)

**Root cause (issue #1601):** bubbletea v2 switched from Windows Console API (ConInput reader, which delivered `WINDOW_BUFFER_SIZE_EVENT`) to VT input mode (`ENABLE_VIRTUAL_TERMINAL_INPUT`). VT input mode does not deliver window resize events, and Windows has no `SIGWINCH` equivalent. The `listenForResize` in bubbletea's `signals_windows.go` is a no-op on Windows.

**Status:** Issue #1601 is closed. The Go proxy confirms v2.0.7 released 2026-06-01. Whether the fix landed in v2.0.7 is unconfirmed. [ASSUMED: polling is still needed in v2.0.7 based on issue analysis; verify at implementation time by testing on Windows without polling.]

**Recommended workaround pattern:**
```go
// In model.Init(): only activate on Windows
if runtime.GOOS == "windows" {
    return tea.Tick(time.Second/4, func(_ time.Time) tea.Msg { return resizePollMsg{} })
}
// In Update() case resizePollMsg:
w, h, _ := term.GetSize(int(os.Stdout.Fd()))
if w != m.termWidth || h != m.termHeight {
    // handle same as tea.WindowSizeMsg
}
if runtime.GOOS == "windows" {
    return m, tea.Tick(time.Second/4, func(_ time.Time) tea.Msg { return resizePollMsg{} })
}
```
**Polling interval:** `time.Second/4` (250ms, 4 Hz) — imperceptible delay, negligible CPU. Use a named constant `windowsPollInterval = time.Second / 4`.

---

## git ls-tree Integration (D-11)

**Command:** `git ls-tree -r --name-only HEAD` [CITED: https://git-scm.com/docs/git-ls-tree]

- `-r`: recurse into subtrees (required for nested dirs)
- `--name-only`: one filepath per line (no hash/mode noise)
- `HEAD`: current commit tree

**Runner call:** `ExecRunner{}.Run([]string{"ls-tree", "-r", "--name-only", "HEAD"})`

**Output parsing:** split by `\n`, filter empty strings. NormalizeCRLF is applied inside `ExecRunner.Run()` on Windows — output is already clean.

**When invoked:** lazily on first press of `a` in the TUI (not at startup). Cache the full-repo path list in `model.allFilePaths []string` after first load — `git ls-tree` can be slow on very large repos (50k+ files).

**Merging changed status:** `statusMap := map[string]string{"src/foo.go": "[M]", ...}` built from `m.files` at model init. Used in `buildTree(allPaths, statusMap)` — unchanged files get `Status: ""`.

---

## HunkStartRows Implementation Guidance

`align.go`'s `alignText()` already walks `file.TextFragments` sequentially. The row count per fragment is deterministic and can be computed without re-running Highlight or Render.

**Algorithm:**
```go
func countFragmentRows(frag *gitdiff.TextFragment) int {
    count := 0
    i := 0
    for i < len(frag.Lines) {
        switch frag.Lines[i].Op {
        case gitdiff.OpContext:
            count++; i++
        case gitdiff.OpDelete:
            dels, adds := 0, 0
            for i < len(frag.Lines) && frag.Lines[i].Op == gitdiff.OpDelete { dels++; i++ }
            for i < len(frag.Lines) && frag.Lines[i].Op == gitdiff.OpAdd    { adds++; i++ }
            count += max(dels, adds) // paired + leftover rows
        case gitdiff.OpAdd:
            count++; i++
        }
    }
    return count
}
```

**Note:** Edge-case files (binary, mode-only, submodule) always return `[]int{0}` from `HunkStartRows()` — there is exactly one hunk at row 0 for those. `Align()` already handles these correctly.

**Hunk centering math:**
```go
targetOffset := m.hunkRows[m.currentHunk] - m.diffVP.Height()/2
if targetOffset < 0 { targetOffset = 0 }
m.diffVP.SetYOffset(targetOffset)
```
This shows the hunk's first changed line at the vertical center with context above and below. The 3-lines-of-context guideline is achieved naturally when the hunk does not start near line 0.

---

## Validation Architecture

> `nyquist_validation = true` in config.json — this section is required.

### Test Framework

| Property | Value |
|----------|-------|
| Framework | `go test` (standard library; no additional framework needed) |
| Config file | none — existing `go test` conventions |
| Quick run command | `go test ./internal/diff/... ./internal/tui/...` |
| Full suite command | `go test -race ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DIFF-06 | `Render()` with `HunkOnly` mode produces fewer rows than `FullFile` | unit | `go test ./internal/diff/... -run TestRender_Modes` | ❌ Wave 0 — extend existing render_test.go |
| NAV-01 | `HunkStartRows()` returns correct row indices for known gitdiff.File | unit | `go test ./internal/diff/... -run TestHunkStartRows` | ❌ Wave 0 |
| NAV-01 | Hunk centering: `SetYOffset` after n/N produces correct YOffset | unit | `go test ./internal/tui/... -run TestModel_HunkNav` | ❌ Wave 0 |
| NAV-02 | `]`/`[` increments `currentFile` and calls `refreshDiffContent` | unit | `go test ./internal/tui/... -run TestModel_FileCycle` | ❌ Wave 0 |
| NAV-03 | Tab toggles `focusedPane` and changes `treeWidth` from 24 to 45 | unit | `go test ./internal/tui/... -run TestModel_FocusToggle` | ❌ Wave 0 |
| NAV-04 | `q` msg causes model to emit `tea.Quit()` cmd | unit | `go test ./internal/tui/... -run TestModel_Quit` | ❌ Wave 0 |
| TREE-01 | Tree builder: dirs-first, collapsed chains, correct status markers | unit | `go test ./internal/tui/... -run TestBuildTree` | ❌ Wave 0 |
| TREE-01 | `collapseChain`: single-child dir chains collapse into one display node | unit | `go test ./internal/tui/... -run TestCollapseChain` | ❌ Wave 0 |
| TREE-02 | Tree width: `treeWidth=24` unfocused, `treeWidth=45` focused | unit | `go test ./internal/tui/... -run TestModel_FocusToggle` | ❌ Wave 0 (shared test) |
| TREE-03 | `a` toggles `allFiles` mode; model accepts `allFilePaths` after ls-tree | unit | `go test ./internal/tui/... -run TestModel_AllFilesToggle` | ❌ Wave 0 |
| SEARCH-01 | `findMatches()` returns correct plain-text positions for ANSI-escaped content | unit | `go test ./internal/tui/... -run TestFindMatches` | ❌ Wave 0 |
| SEARCH-01 | Search mode: `n`/`N` routes to `HighlightNext()`/`HighlightPrevious()` not hunk nav | unit | `go test ./internal/tui/... -run TestModel_SearchMode` | ❌ Wave 0 |
| D-07 | `View()` returns empty string before first WindowSizeMsg | unit | `go test ./internal/tui/... -run TestModel_NotReady` | ❌ Wave 0 |

**Manual-only tests (cannot automate):**
- Full terminal rendering visual quality (requires real TTY)
- Windows resize polling behavior (requires Windows Terminal)
- Performance under large diff files (>10k lines)

### Sampling Rate

- **Per task commit:** `go test ./internal/diff/... ./internal/tui/...`
- **Per wave merge:** `go test -race ./...`
- **Phase gate:** `go test -race ./...` green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/tui/model_test.go` — model dispatch tests (NAV-01 through NAV-04, SEARCH-01, D-07)
- [ ] `internal/tui/tree_test.go` — tree builder, collapseChain, flattenTree (TREE-01 through TREE-03)
- [ ] `internal/tui/search_test.go` — `findMatches()` with ANSI-escaped input (SEARCH-01)
- [ ] `internal/diff/render_test.go` — update existing callers for new `mode` parameter (DIFF-06)
- [ ] `internal/diff/align_test.go` — add `HunkStartRows` tests (NAV-01)
- [ ] Framework install: none — `go test` is built-in

---

## Security Domain

> `security_enforcement = true`, `security_asvs_level = 1` in config.json.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | No auth in TUI diff viewer |
| V3 Session Management | No | No sessions; single-process in-memory state |
| V4 Access Control | No | No ACLs; reads only what git exposes |
| V5 Input Validation | Yes (partial) | Search query: used only for `strings.Index()` — no exec, no SQL, no template rendering. ANSI output: already sanitized by `ansi.Strip()` before search. |
| V6 Cryptography | No | No cryptographic operations |

### Known Threat Patterns for This Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Shell injection via git ref args | Tampering | `exec.Command("git", args...)` — argv form, not shell. Already implemented in Phase 2. No change needed. |
| ANSI injection via diff content (terminal escape injection) | Tampering | Diff content is rendered via `diff.Render()` which produces known ANSI. Search query (`strings.Index`) is not interpolated into ANSI sequences. No risk. |
| Path traversal via `git ls-tree` output | Spoofing | `git ls-tree` output is parsed as plain paths split by newline; used only as display strings + tree keys. No filesystem access from these paths in Phase 3. |
| Denial of service via very large diff | Denial of Service | `diff.Render()` limits intra-line diff to 1000 chars / 200 tokens / 100ms (DIFF-04 guards from Phase 1). Full-file viewport scrolling is bounded by terminal height. |
| Sensitive data in log file | Information Disclosure | `applog.Init()` from Phase 2 writes to `$XDG_STATE_HOME/alturd/alturd.log`. No diff content is written to the log — log level is debug for structural events only. |

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All builds | ✓ | go1.25.11 | — |
| `git` on PATH | `git ls-tree` (D-11) | ✓ (assumed in dev env) | any | Error message already handled by Phase 2 ErrGitNotFound |
| `charm.land/bubbletea/v2` | TUI framework | ✗ (not in go.mod yet) | v2.0.7 | `go get charm.land/bubbletea/v2@v2.0.7` |
| `charm.land/lipgloss/v2` | Layout | ✗ (v1 present as indirect) | v2.0.4 | `go get charm.land/lipgloss/v2@v2.0.4` |
| `charm.land/bubbles/v2` | viewport + textinput | ✗ (not in go.mod yet) | v2.1.0 | `go get charm.land/bubbles/v2@v2.1.0` |
| `github.com/charmbracelet/x/ansi` | ANSI stripping | ✓ (indirect dep v0.8.0) | v0.8.0 | Already installed |
| `golang.org/x/term` | Windows resize poll | ✓ (v0.44.0 in go.mod) | v0.44.0 | Already installed |

**Missing dependencies with no fallback:**
- charm.land packages (bubbletea v2, lipgloss v2, bubbles v2) — must be installed via `go get` as Wave 0 step

**Missing dependencies with fallback:**
- None

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Windows resize polling is still needed in bubbletea v2.0.7 (issue #1601 may have been fixed) | Windows Resize Polling | If fixed, polling still works (just redundant); no harm in keeping it |
| A2 | `viewport.SetHighlights([][]int)` positions are in plain-text character coordinates (ANSI-stripped) | Pattern 4, Pitfall 5 | If positions are raw bytes, search highlights appear at wrong locations; need custom ANSI position mapper |
| A3 | `collapseChain` algorithm (pure Go) correctly replicates GitHub-style collapse | Pattern 6 | Edge cases (files at root with no parent dir, empty dir chains) may need additional guards |
| A4 | `git ls-tree -r --name-only HEAD` works correctly when run from subdirectory of repo | git ls-tree section | May need `--full-tree` flag if alturd is invoked from a subdirectory |
| A5 | `tea.View` is a struct wrapping string (not a type alias `= string`) | Pattern 3 | If `View` is a type alias, `return tea.NewView(s)` still works; no risk |
| A6 | `diffVP.GetContent()` returns the same string that was passed to `SetContent()` (no transformation) | Pattern 4 | If content is line-wrapped or modified by viewport internals, `ansi.Strip(diffVP.GetContent())` positions won't match `diffVP.SetHighlights()` positions |

**If this table is empty:** It is not empty — see above.

---

## Open Questions

1. **Is bubbletea v2 issue #1601 fixed in v2.0.7?**
   - What we know: Issue closed; v2.0.7 released 2026-06-01; root cause was VT input mode dropping ConInput.
   - What's unclear: Whether a fix was merged before v2.0.7, or the issue was closed as "by design".
   - Recommendation: Implement polling unconditionally for safety; test on Windows before Phase 4. The poll is cheap (250ms interval, single `term.GetSize` call).

2. **Does `viewport.SetHighlights` work correctly when content has ANSI sequences?**
   - What we know: `SetHighlights` uses plain-text positions; `lipgloss.StyleRanges` is ANSI-aware.
   - What's unclear: Whether `HighlightStyle` correctly overrides existing ANSI background colors in the diff (e.g., dark green diff background) vs. composing on top.
   - Recommendation: Set `HighlightStyle = lipgloss.NewStyle().Reverse(true)` (reverse video) which inverts fg/bg — visible on any background. Fall back to custom ANSI insertion if visual results are unacceptable.

3. **`git ls-tree` behavior from subdirectory**
   - What we know: `-r --name-only HEAD` returns paths relative to tree root.
   - What's unclear: Whether running from `src/subdir/` changes the paths returned.
   - Recommendation: Add `--full-tree` flag: `git ls-tree -r --full-tree --name-only HEAD`. This ensures root-relative paths regardless of cwd. [CITED: git-scm.com/docs/git-ls-tree]

---

## Sources

### Primary (MEDIUM confidence — Go proxy + pkg.go.dev)
- [pkg.go.dev/charm.land/bubbles/v2/viewport](https://pkg.go.dev/charm.land/bubbles/v2/viewport) — viewport.Model API, SetHighlights semantics
- [pkg.go.dev/charm.land/bubbletea/v2](https://pkg.go.dev/charm.land/bubbletea/v2) — tea.Model interface, tea.View type, tea.NewProgram
- [pkg.go.dev/charm.land/lipgloss/v2](https://pkg.go.dev/charm.land/lipgloss/v2) — JoinHorizontal, Style.MaxWidth
- [pkg.go.dev/charm.land/bubbles/v2/textinput](https://pkg.go.dev/charm.land/bubbles/v2/textinput) — textinput.Model API
- [github.com/charmbracelet/bubbles/blob/main/UPGRADE_GUIDE_V2.md](https://github.com/charmbracelet/bubbles/blob/main/UPGRADE_GUIDE_V2.md) — v1→v2 breaking changes
- [Go module proxy](https://proxy.golang.org) — version and source verification for all charm.land packages
- [git-scm.com/docs/git-ls-tree](https://git-scm.com/docs/git-ls-tree) — flags: -r, --name-only, --full-tree

### Secondary (LOW confidence — web search)
- [github.com/charmbracelet/bubbletea/issues/1601](https://github.com/charmbracelet/bubbletea/issues/1601) — Windows resize regression analysis
- [github.com/charmbracelet/bubbletea/discussions/661](https://github.com/charmbracelet/bubbletea/discussions/661) — polling workaround pattern
- [pkg.go.dev/github.com/charmbracelet/x/ansi](https://pkg.go.dev/github.com/charmbracelet/x/ansi) — Strip, StringWidth functions

### Codebase (HIGH confidence — verified by Read)
- `/src/alturd/internal/diff/align.go` — `FileStatus()` confirmed; `Align()` signature confirmed; hunk iteration pattern confirmed
- `/src/alturd/internal/diff/render.go` — `Render(file, width)` confirmed; `truncateANSI` pattern usable for position mapping
- `/src/alturd/internal/diff/model.go` — `RenderMode`, `RowPair`, `RenderedLine`, `LineKind` all confirmed
- `/src/alturd/cmd/alturd/main.go` — Phase 2 stdout render loop confirmed; replacement point identified
- `/src/alturd/go.mod` — versions confirmed; `charmbracelet/x/ansi v0.8.0` confirmed as indirect dep

---

## Metadata

**Confidence breakdown:**
- Standard stack: MEDIUM — charm.land packages verified on Go proxy; API details from pkg.go.dev (official source)
- Architecture: HIGH — derived from existing codebase read + locked CONTEXT.md decisions
- Pitfalls: MEDIUM — v1→v2 migration changes confirmed from official UPGRADE_GUIDE_V2.md
- Windows polling: LOW — community workaround; issue status unclear

**Research date:** 2026-07-01
**Valid until:** 2026-08-01 (charm.land packages are stable; bubbles v2.1.0 released 2026-03-25 with no known breaking changes since)
