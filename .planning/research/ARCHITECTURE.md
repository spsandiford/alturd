# Architecture Patterns

**Project:** alturd (Go port)
**Domain:** Terminal UI git diff viewer — four-layer architecture
**Researched:** 2026-06-25
**Confidence:** MEDIUM (community sources; verified against real-world bubbletea projects)

---

## Recommended Architecture

A four-layer dependency graph where data flows strictly downward: git subprocess produces raw bytes, the diff model parses and aligns them into structs, the config layer provides styling/keybinding settings, and the TUI layer consumes the model to render.

```
cmd/alturd/
    main.go            ← entry point; wires layers; dispatches standalone vs difftool mode

internal/git/          ← git subprocess chokepoint (no internal imports)
    runner.go          ← exec.CommandContext wrapper, repo-root detection
    diff.go            ← run_diff, resolve_scope, build_raw_entries
    errors.go          ← typed errors (NotARepo, NoChanges, etc.)

internal/diff/         ← pure diff middle layer (imports only internal/git)
    model.go           ← FileDiff, Hunk, Line, LineKind (Added/Removed/Context/Modified)
    parse.go           ← unified diff parser (wraps sourcegraph/go-diff or hand-ported)
    align.go           ← positional alignment of old/new sides → []RowPair
    highlight.go       ← chroma integration: tokenize + ANSI format per line

internal/config/       ← TOML config loader (no internal imports)
    config.go          ← ConfigFile struct, XDG path resolution
    keybindings.go     ← default keybindings + user overrides
    theme.go           ← light/dark/auto detection, no-color support, chroma style selection

internal/ui/           ← bubbletea TUI (imports diff, config)
    app.go             ← root AppModel: embeds FileTree + DiffView, routes messages
    filetree/
        model.go       ← FileTreeModel: cursor, selection, width adaptation
        view.go        ← render tree entries with [A]/[M]/[D]/[R] markers
    diffview/
        model.go       ← DiffViewModel: viewport.Model, current FileDiff, hunk nav
        view.go        ← render side-by-side columns via lipgloss.JoinHorizontal
        search.go      ← in-pane search state and match highlighting
    titlebar.go        ← TitleBar: "N of M" counter, filename, mode indicator
    hintbar.go         ← HintBar: contextual keybinding hints per focused pane
    messages.go        ← shared tea.Msg types: FileSelected, HunkJump, ModeToggle
    styles.go          ← lipgloss style definitions derived from config.Theme

.goreleaser.yaml       ← build matrix: linux/darwin/windows × amd64/arm64
.github/workflows/
    ci.yml             ← test + vet on every push (Linux, macOS, Windows)
    release.yml        ← goreleaser trigger on git tag push
```

---

## Component Boundaries

| Component | Responsibility | Imports | Communicates With |
|-----------|---------------|---------|-------------------|
| `cmd/alturd` | Entry point; argument parsing; mode dispatch | `internal/git`, `internal/config`, `internal/ui` | Passes initialized models into bubbletea |
| `internal/git` | All git subprocess invocations; repo-root detection | `os/exec`, `context` only | Returns raw diff bytes to `internal/diff` |
| `internal/diff` | Parse, model, align, highlight diff content | `internal/config` (for theme/style) | Produces `[]RowPair` consumed by `internal/ui` |
| `internal/config` | Load TOML, keybindings, theme detection | Standard library only | Read by `internal/diff` and `internal/ui` |
| `internal/ui` | All bubbletea models; TUI rendering | `internal/diff`, `internal/config` | Drives the bubbletea program loop |

**Import direction (enforced by Go compiler via internal/):**

```
cmd/alturd
    ├── internal/git        (no internal deps)
    ├── internal/config     (no internal deps)
    ├── internal/diff       (imports internal/config)
    └── internal/ui         (imports internal/diff, internal/config)
```

No cycles are possible if this direction is maintained. If `internal/git` ever needs something from `internal/config` (e.g., a git binary path), pass it as a parameter, not an import.

---

## Data Flow

### Standalone Mode (Full App)

```
User runs: alturd [refs] [-- paths]
    │
    ▼
cmd/alturd/main.go
    │  argparse → resolve mode (standalone)
    │  load internal/config (TOML + keybindings + theme)
    │
    ▼
internal/git.RunDiff(ctx, scope, paths)
    │  exec.CommandContext("git", "diff", "--unified=0", ...)
    │  returns: raw unified diff []byte
    │
    ▼
internal/diff.Parse(rawBytes)
    │  sourcegraph/go-diff ParseMultiFileDiff → []*diff.FileDiff structs
    │  returns: []FileDiff with Hunks and Lines
    │
    ▼
internal/diff.Align(fileDiff, mode)
    │  pairs old/new lines into []RowPair
    │  computes intra-line word diff spans on Modified lines
    │  returns: aligned []RowPair ready for rendering
    │
    ▼
internal/diff.Highlight(rowPairs, filename, theme)
    │  chroma lexer detection via filename/content
    │  ANSI terminal formatting per line segment
    │  returns: pre-rendered ANSI strings per side
    │
    ▼
internal/ui.AppModel (bubbletea)
    │
    ├── FileTreeModel.View()
    │       lipgloss.NewStyle().Width(treeWidth).Render(entries)
    │
    └── DiffViewModel.View()
            viewport.Model wrapping side-by-side block
            lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
            │
            └── TitleBar + HintBar (top/bottom chrome)
```

### Difftool Mode (Per-File, Skips Tree)

```
User runs: git difftool (env GIT_DIFF_PATH_COUNTER set)
    │
    ▼
cmd/alturd/main.go
    │  detects ORIGINAL/MODIFIED file args (no tree)
    │  reads files directly (no git subprocess needed)
    │
    ▼
internal/diff.ParseFile(orig, modified)
    │  produce synthetic FileDiff from two file reads
    │
    ▼
internal/ui.DiffOnlyModel (no FileTree)
    │  TitleBar shows "N of M" from env vars
```

---

## Bubbletea Multi-Pane Pattern

### Root App Model

```go
// internal/ui/app.go
type focusedPane int
const (
    paneFileTree focusedPane = iota
    paneDiffView
)

type AppModel struct {
    fileTree filetree.Model
    diffView diffview.Model
    titleBar TitleBarModel
    hintBar  HintBarModel
    focused  focusedPane
    width    int
    height   int
    cfg      *config.Config
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width, m.height = msg.Width, msg.Height
        // recalculate pane widths; propagate to children
        return m.propagateSize()

    case tea.KeyMsg:
        // route keybindings to focused pane, or handle global keys
        switch {
        case key.Matches(msg, m.cfg.Keys.Tab):
            return m.toggleFocus()
        default:
            return m.routeToFocused(msg)
        }

    case messages.FileSelected:
        // file tree selected a file; load diff into diffview
        return m.loadDiff(msg.Entry)
    }

    // delegate to both sub-models for non-key messages
    var cmds []tea.Cmd
    m.fileTree, cmd = m.fileTree.Update(msg); cmds = append(cmds, cmd)
    m.diffView, cmd = m.diffView.Update(msg); cmds = append(cmds, cmd)
    return m, tea.Batch(cmds...)
}

func (m AppModel) View() string {
    treePane := m.fileTree.View()
    diffPane := m.diffView.View()
    body := lipgloss.JoinHorizontal(lipgloss.Top, treePane, diffPane)
    return lipgloss.JoinVertical(lipgloss.Left,
        m.titleBar.View(),
        body,
        m.hintBar.View(),
    )
}
```

### Focus Width Adaptation

The Python implementation narrows the tree pane when unfocused (45 col focused, 24 col unfocused). Implement in the `focusedPane` transition handler:

```go
func (m *AppModel) toggleFocus() (tea.Model, tea.Cmd) {
    if m.focused == paneFileTree {
        m.focused = paneDiffView
        m.fileTree.SetWidth(24)
        m.diffView.SetWidth(m.width - 25) // account for separator
    } else {
        m.focused = paneFileTree
        m.fileTree.SetWidth(45)
        m.diffView.SetWidth(m.width - 46)
    }
    return m, nil
}
```

### DiffView Viewport Pattern

```go
// internal/ui/diffview/model.go
type Model struct {
    viewport viewport.Model
    rows     []diff.RowPair  // pre-rendered aligned pairs
    width    int
    height   int
    focused  bool
}

func New(width, height int) Model {
    vp := viewport.New(width, height)
    return Model{viewport: vp}
}

func (m *Model) SetContent(rows []diff.RowPair, halfWidth int) {
    var buf strings.Builder
    for _, row := range rows {
        left  := renderSide(row.OldLine, halfWidth)
        right := renderSide(row.NewLine, halfWidth)
        buf.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, right))
        buf.WriteString("\n")
    }
    m.viewport.SetContent(buf.String())
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
    if !m.focused {
        return m, nil  // swallow key events when unfocused
    }
    var cmd tea.Cmd
    m.viewport, cmd = m.viewport.Update(msg)
    return m, cmd
}
```

---

## Patterns to Follow

### Pattern 1: Pre-render Before Viewport, Not Inside View()

**What:** Compute the full ANSI-rendered string for all diff rows once (when a file is selected), store in viewport via `SetContent`. Do not call chroma inside `View()`.

**When:** Always. `View()` is called on every frame; tokenization is expensive.

**Rationale:** Chroma tokenization + ANSI formatting for a large file takes milliseconds. Pre-rendering on `FileSelected` message means `View()` just calls `m.viewport.View()`.

### Pattern 2: Message-Passing for Cross-Pane Events

**What:** FileTree publishes `messages.FileSelected{Entry}`. DiffView subscribes. Both live in `AppModel.Update` which routes.

**When:** Any time one sub-model needs to trigger behavior in another.

```go
// internal/ui/messages/messages.go
type FileSelected struct { Entry diff.FileDiff }
type HunkJump    struct { Delta int }   // +1 next, -1 prev
type ModeToggle  struct{}               // full-file ↔ hunk-only
```

### Pattern 3: WindowSizeMsg Propagation

**What:** On `tea.WindowSizeMsg`, AppModel recalculates pane widths and calls `fileTree.SetWidth(n)` and `diffView.SetWidth(n)` and `diffView.SetHeight(n)` before forwarding the message to children.

**When:** Every terminal resize event.

**Why:** Viewport must know its dimensions to compute scroll percentage and clip content correctly. Lipgloss styles with `.Width(n)` are set per render, but viewport needs the dimensions explicitly.

### Pattern 4: Separate `model.go` and `view.go` per Sub-Model

**What:** `internal/ui/diffview/model.go` holds state and Update logic. `internal/ui/diffview/view.go` holds View() and all lipgloss rendering. `internal/ui/diffview/search.go` holds search state mutation.

**When:** For any sub-model with non-trivial rendering. Keeps files under ~300 lines.

**Reference:** `revdiff` uses this same model.go/view.go/handlers.go split per sub-model.

### Pattern 5: Typed Errors from git Layer

**What:** `internal/git` returns typed errors (`var ErrNotARepo = errors.New("not a git repo")`) not bare strings.

**When:** All git subprocess failures.

**Why:** `cmd/alturd/main.go` can provide clean user-facing messages by switching on error type, without parsing error strings.

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Calling Chroma in View()

**What goes wrong:** Per-frame syntax tokenization makes the TUI stutter on large diffs.

**Why it happens:** Natural to write, feels simple.

**Instead:** Pre-render on data load (`FileSelected` msg), store ANSI strings in model, call `viewport.SetContent` once.

### Anti-Pattern 2: Storing Rendered Strings in git Layer

**What goes wrong:** git layer grows UI concerns; becomes untestable without terminal.

**Instead:** git layer returns raw `[]byte` unified diff only. All rendering lives in `internal/diff` and `internal/ui`.

### Anti-Pattern 3: Import Cycles via Shared Types

**What goes wrong:** `internal/diff` imports from `internal/ui` (e.g., to use a shared `Style` type), creating a cycle the Go compiler rejects.

**Instead:** Define all shared types (`FileDiff`, `RowPair`, `LineKind`) in `internal/diff`. The ui layer imports diff, never the reverse. Config types live in `internal/config` which both diff and ui may import.

### Anti-Pattern 4: Single Giant AppModel Update Function

**What goes wrong:** `app.go` becomes 600+ lines handling all key events and state transitions for all panes inline.

**Instead:** Each sub-model handles its own key events when focused. AppModel only handles: focus switching (Tab), global quit, WindowSizeMsg propagation, and cross-pane messages (FileSelected, HunkJump).

### Anti-Pattern 5: Shell String Interpolation for Git Commands

**What goes wrong:** `exec.Command("sh", "-c", "git diff "+userInput)` enables command injection and shell quoting bugs.

**Instead:** `exec.CommandContext(ctx, "git", "diff", "--", path)` — args as separate strings, always.

---

## Build Order

Build in this order to maximize testability at each stage:

| Stage | Package | What to Build | Can Test With |
|-------|---------|---------------|---------------|
| 1 | `internal/diff` | FileDiff/Hunk/Line model structs, parser, alignment | Go table tests against 12 Python fixture files |
| 2 | `internal/git` | exec.CommandContext wrapper, repo detection | Integration tests with a temp git repo |
| 3 | `internal/config` | TOML loader, keybindings, theme detection | Unit tests with sample TOML |
| 4 | `internal/diff` (highlight) | Chroma integration, ANSI rendering | Snapshot tests of rendered output |
| 5 | `internal/ui/diffview` | DiffView model + viewport + lipgloss layout | Manual; bubbletea testing via teatest |
| 6 | `internal/ui/filetree` | FileTree model + rendering | Manual; teatest |
| 7 | `internal/ui` (app) | AppModel composition, focus routing | teatest integration |
| 8 | `cmd/alturd` | Entry point, arg parsing, mode dispatch | End-to-end test with real git repos |
| 9 | `.goreleaser.yaml` | Build matrix, GitHub Actions | `goreleaser release --snapshot --clean` |

**Rationale for this order:**

The diff parser (`internal/diff`) is the highest-risk component — it has 12 edge-case scenarios in the Python fixture corpus (binary pairs, renames, no-newline at EOF, empty diffs, etc.). Build and validate it in complete isolation before any TUI code exists. The git layer can then be built knowing the parser it feeds is correct. The TUI is the most complex to test and iterate, so it comes last.

---

## Scalability Considerations

| Concern | Approach | Notes |
|---------|----------|-------|
| Very large diffs (10K+ lines) | Pre-render once to ANSI string; viewport clips to visible lines only | chroma tokenizes entire file — acceptable for typical files; may need chunking for huge monorepo diffs |
| Terminal resize | WindowSizeMsg recalculates widths and calls viewport.SetWidth/Height; content re-rendered | Viewport preserves scroll percent on resize |
| Multiple entry points | `cmd/alturd/main.go` single binary; mode detected from args and env vars | Standalone vs difftool share all of `internal/` |
| Cross-platform ANSI | Bubbletea uses `x/term` for terminal detection; lipgloss respects NO_COLOR | Windows terminal support is good in modern Windows Terminal via bubbletea's Windows renderer |
| CGO disabled | `CGO_ENABLED=0` in goreleaser build; chroma and go-diff are pure Go | No CGO dependency in the entire stack |

---

## Repo Structure (Complete)

```
alturd/
├── go.mod                          ← module github.com/your-org/alturd
├── go.sum
├── .goreleaser.yaml                ← goreleaser build matrix
├── Makefile                        ← dev targets: build, test, lint, snapshot
├── cmd/
│   └── alturd/
│       └── main.go                 ← entry point
├── internal/
│   ├── git/
│   │   ├── runner.go
│   │   ├── diff.go
│   │   └── errors.go
│   ├── diff/
│   │   ├── model.go
│   │   ├── parse.go
│   │   ├── align.go
│   │   └── highlight.go
│   ├── config/
│   │   ├── config.go
│   │   ├── keybindings.go
│   │   └── theme.go
│   └── ui/
│       ├── app.go
│       ├── messages/
│       │   └── messages.go
│       ├── filetree/
│       │   ├── model.go
│       │   └── view.go
│       ├── diffview/
│       │   ├── model.go
│       │   ├── view.go
│       │   └── search.go
│       ├── titlebar.go
│       ├── hintbar.go
│       └── styles.go
├── tests/
│   └── fixtures/
│       └── diff/                   ← ported from Python fixture corpus (12 scenarios)
└── .github/
    └── workflows/
        ├── ci.yml
        └── release.yml
```

---

## Goreleaser Configuration Sketch

```yaml
# .goreleaser.yaml
version: 2
before:
  hooks:
    - go mod tidy
    - go test ./...

builds:
  - env:
      - CGO_ENABLED=0
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ignore:
      - goos: windows
        goarch: arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

archives:
  - format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: "checksums.txt"

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
```

---

## Sources

- [Organizing a Go module (go.dev official)](https://go.dev/doc/modules/layout) — MEDIUM confidence
- [golang-standards/project-layout](https://github.com/golang-standards/project-layout) — MEDIUM confidence
- [Structuring Go Code for CLI Applications (bytesizego)](https://www.bytesizego.com/blog/structure-go-cli-app) — MEDIUM confidence
- [charmbracelet/bubbletea GitHub](https://github.com/charmbracelet/bubbletea) — MEDIUM confidence
- [charmbracelet/lipgloss GitHub](https://github.com/charmbracelet/lipgloss) — MEDIUM confidence
- [charmbracelet/bubbles viewport (pkg.go.dev)](https://pkg.go.dev/github.com/charmbracelet/bubbles/viewport) — MEDIUM confidence
- [sourcegraph/go-diff GitHub](https://github.com/sourcegraph/go-diff) — MEDIUM confidence
- [alecthomas/chroma GitHub](https://github.com/alecthomas/chroma) — MEDIUM confidence
- [GoReleaser GitHub Actions docs](https://goreleaser.com/customization/ci/actions/) — MEDIUM confidence
- [revdiff TUI package (real-world reference)](https://pkg.go.dev/github.com/umputun/revdiff/app/ui) — MEDIUM confidence
- [Multi-View Interfaces in Bubble Tea (shi.foo)](https://shi.foo/weblog/multi-view-interfaces-in-bubble-tea) — LOW confidence (403 on fetch, title suggestive)
- [Go exec.Command best practices (Medium)](https://medium.com/@caring_smitten_gerbil_914/running-external-programs-in-go-the-right-way-38b11d272cd1) — LOW confidence
