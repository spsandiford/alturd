# Feature Research

**Domain:** Go TUI git diff viewer (port of Python/Textual alturd)
**Researched:** 2026-06-25
**Confidence:** MEDIUM (web sources cross-checked against multiple tool repositories; bubbletea/chroma APIs verified against pkg.go.dev documentation)

---

## Feature Landscape

### Table Stakes (Users Expect These)

These are the non-negotiable baseline — anything missing makes the port feel incomplete compared to the Python original or peer tools (gitui, delta).

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Side-by-side diff pane (old \| new) | Delta set this as the standard for terminal diff display; users coming from Meld/kdiff3 expect it | HIGH | lipgloss.JoinHorizontal with two viewports; synchronized scroll is non-trivial |
| Syntax highlighting per language | Both gitui and delta provide it; raw diffs feel archaic | MEDIUM | Chroma v2 (alecthomas/chroma) — direct Pygments port, 700+ lexers, terminal formatter outputs ANSI |
| Line-level diff colors (added/removed/modified) | Universal expectation for any diff tool | LOW | Chroma ANSI output layered with diff color spans; requires ANSI-aware string processing |
| `n`/`N` hunk navigation | Delta has it; keyboard users demand it | MEDIUM | Must track hunk positions and map them to viewport y_offset |
| `q` to quit | Universal TUI convention | LOW | tea.Quit message |
| `Tab` to switch focus between tree and diff | Established TUI pattern (gitui, lazygit) | LOW | Toggle focused component in model state, recalculate widths |
| `]`/`[` to cycle between changed files | Python alturd ships this; users expect continuity | LOW | Index into file list, update active diff |
| File tree with status markers ([A]/[M]/[D]/[R]) | gitui shows status in file list; alturd Python parity required | MEDIUM | List component with status-colored prefixes |
| Dirs-first sort with compact-folder layout | Python original behavior; expected for any file tree | MEDIUM | Sorting is trivial; compact-folder (collapse single-child dirs) requires tree traversal |
| Single self-contained binary | The entire value proposition of porting to Go | LOW | CGO_ENABLED=0 in goreleaser config; no dynamically linked C deps |
| `alturd [refs...] [-- paths...]` CLI grammar | Python parity — users who alias `alturd` must not break | LOW | Standard cobra/flag parsing; mirror Python's git-rev-parse ref resolution |
| Cross-platform (Linux/macOS/Windows) | Single-binary promise requires all three platforms | MEDIUM | goreleaser goos matrix; termenv handles ANSI on Windows 10+; path separators |

### Differentiators (Competitive Advantage)

These features make alturd distinct among Go TUI tools and justify porting the Python original rather than just using gitui or lazygit.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Full-file mode (entire file, changes inline) | No peer tool does this interactively — delta is a pager, gitui shows only the diff, lazygit delegates | HIGH | Entire file content must be pre-rendered with changes highlighted; requires virtual windowing for large files in bubbletea (rendering more lines than viewport height breaks the renderer) |
| Hunk-only mode with `v` toggle | Runtime toggle between views without reload — unique in Go TUI space | MEDIUM | Mode flag in model; full content string or hunk-only string swapped in viewport SetContent; viewport y_offset reset or translated |
| Adaptive pane width (45 focused / 24 unfocused) | Makes file tree unobtrusive when reading diffs — no Go TUI peer does this | MEDIUM | tea.WindowSizeMsg triggers width recalculation; tree pane width set to 45 or 24 on focus change; diff pane takes remainder |
| Intra-line word/character diff markers | Delta does this as a pager; alturd makes it interactive | HIGH | go-diff does NOT provide word-level diff. Requires secondary Myers/Levenshtein LCS pass over text content of adjacent remove+add line pairs. revdiff's worddiff sub-package is the Go blueprint. |
| `git difftool` integration with N of M counter | No other Go TUI exposes GIT_DIFF_PATH_COUNTER/GIT_DIFF_PATH_TOTAL natively | LOW | Read env vars at startup; display "N of M" in title bar; skip tree pane in difftool mode |
| `alturd install-difftool` subcommand | Zero-friction gitconfig setup — one command to wire alturd as git difftool | LOW | Writes difftool.alturd.cmd and diff.tool entries to gitconfig via `git config` calls |
| Changed-only vs full-repo tree toggle (`a` key) | Useful when reviewing partial-checkout or working-tree state | MEDIUM | Two list models pre-computed; swap on keypress; maintain selection index across swap |
| `/` in-pane search with match highlighting | gitui has file search; alturd brings it into the diff content itself | HIGH | ANSI-aware scanner required — must not break syntax-highlighting escape sequences. revdiff searchState pattern: track term, match spans, cursor; re-render with highlight markers on each keystroke |
| Light/dark/auto theming | delta supports themes statically; alturd detects at runtime | MEDIUM | muesli/termenv HasDarkBackground() on startup; fallback to dark if query fails (common in tmux/SSH); lipgloss.AdaptiveColor{} for per-color auto-selection |
| XDG config with configurable keybindings | Power-user feature missing from most Go TUI tools | LOW | adrg/xdg for path resolution; BurntSushi/toml for parsing; keybinding struct with defaults overridden by config |
| GitHub Releases binary distribution | goreleaser + GitHub Actions makes alturd installable with `curl \| tar \| mv` | LOW | Standard goreleaser setup; CGO_ENABLED=0 ensures truly portable static binaries |

### Anti-Features (Deliberately Excluded)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Staging hunks/lines from the diff view | gitui does it; users associate "diff viewer" with "staging tool" | Bloats scope; alturd is a reviewer, not a staging tool; adds state synchronization complexity | Use gitui or `git add -p` for staging |
| Branch/commit management UI | lazygit does it; "while I'm here" requests | Turns alturd into a git GUI; scope doubles; violates single-responsibility | Use lazygit or gitui for management operations |
| Three-way merge / conflict resolution | Present in Meld/kdiff3 which alturd resembles visually | Deferred in the Python original; adding it here invalidates the "1:1 port" constraint | Deferred to post-v1 at minimum; possibly never |
| Mouse click navigation | Modern TUI expectation | Adds a second interaction model; bubbletea mouse support works but interaction with viewport scrolling and column clicking is complex | Keyboard-first is the alturd identity |
| Streaming/lazy loading of very large files | Users with huge files may request it | Out of scope per PROJECT.md; bubbletea viewport with virtual windowing handles reasonable file sizes | Document the limit; recommend `git diff --stat` for huge repos |
| In-app git operations (commit, push, pull) | "While I'm here" expansion | Completely outside alturd's value proposition | Explicit non-goal since the Python original |
| Real-time filesystem watch / auto-refresh | "Show me changes as I edit" | Would require a background goroutine and viewport invalidation loop; conflicts with the diff-snapshot model | Run `alturd` again after editing |
| PyPI/pip distribution | Existing Python users may expect pip install | The whole point of the Go port is to eliminate the Python dependency | GitHub Releases binary only |

---

## Feature Dependencies

```
CLI argument parsing (refs, paths)
    └──requires──> git plumbing calls (diff-tree, diff-files)
                       └──requires──> unified diff parser (go-diff)
                                          └──requires──> diff renderer

Diff renderer (side-by-side columns)
    ├──requires──> Chroma syntax highlighting (per-file language detection)
    ├──requires──> Line-level diff colors (layered ANSI spans)
    └──requires──> Intra-line word diff (secondary LCS pass)
                       └──requires──> ANSI-aware string processing

Full-file mode
    └──requires──> Diff renderer + virtual windowed viewport

Hunk-only mode
    └──requires──> Diff renderer (subset: just the hunks)

`v` toggle
    └──requires──> Both full-file mode AND hunk-only mode implemented

`n`/`N` hunk navigation
    └──requires──> Hunk position index (works in both modes)

File tree
    └──requires──> Changed file list from git
                       └──enhances──> `]`/`[` file cycling

In-pane search
    └──requires──> Diff renderer (search within rendered content)
    └──requires──> ANSI-aware marker insertion

Theming (light/dark/auto)
    └──requires──> termenv background detection
    └──enhances──> Chroma theme selection
    └──enhances──> Lipgloss AdaptiveColor styles

Difftool mode
    └──requires──> Diff renderer (without file tree)
    └──requires──> GIT_DIFF_PATH_COUNTER / GIT_DIFF_PATH_TOTAL env vars

`install-difftool` subcommand
    └──requires──> CLI entry point only (no TUI needed)

XDG config + keybindings
    └──enhances──> All keyboard navigation features
    └──requires──> adrg/xdg + BurntSushi/toml

Distribution (goreleaser)
    ──orthogonal to──> All TUI features (CI/CD concern only)
```

### Dependency Notes

- **Intra-line word diff requires ANSI-aware processing:** Chroma produces ANSI escape sequences; the word diff LCS pass must operate on the raw text content (stripping ANSI), compute spans, then re-insert highlight markers at the correct byte positions in the ANSI string without disrupting existing escape sequences.
- **Full-file mode requires virtual windowing:** bubbletea's renderer breaks if the content string has more lines than the terminal height. The viewport must track a visible window (y_offset to y_offset+height) and only pass that window to SetContent, recomputing on every scroll event.
- **Side-by-side scroll synchronization:** Both diff viewports (old and new) must have their y_offset kept equal. Tab focus switches between panes but scroll events on either pane should update both.
- **Theming must be resolved before first render:** termenv.HasDarkBackground() is a one-time call at startup; the result propagates into the Chroma style selection and all Lipgloss AdaptiveColor definitions.

---

## MVP Definition

This is a port, not a greenfield product. MVP = feature parity with the Python original at v1.1. The port is complete when every requirement in PROJECT.md Active section is satisfied.

### Launch With (v1 — Full Parity)

- [ ] File tree with [A]/[M]/[D]/[R] markers, dirs-first, compact-folder layout — TREE-01
- [ ] Adaptive tree width 45/24 columns on focus — TREE-02
- [ ] `]`/`[` file cycling, `a` changed-only toggle — TREE-03
- [ ] Side-by-side diff pane with old/new aligned — DIFF-01
- [ ] Chroma syntax highlighting — DIFF-02
- [ ] Line-level diff colors — DIFF-03
- [ ] Intra-line word/char diff markers — DIFF-04
- [ ] Full-file mode (default) — DIFF-05
- [ ] Hunk-only mode + `v` toggle — DIFF-06
- [ ] `n`/`N` hunk navigation — NAV-01
- [ ] `Tab` focus switch — NAV-02
- [ ] `/` in-pane search — SEARCH-01
- [ ] Light/dark/auto theming — THEME-01
- [ ] TOML config with configurable keybindings at XDG path — CONFIG-01
- [ ] `alturd [refs...] [-- paths...]` CLI — CLI-01
- [ ] `alturd install-difftool` subcommand — CLI-02
- [ ] Difftool mode (per-file launch, no tree) — HELPER-01
- [ ] N of M counter in title bar — HELPER-02
- [ ] GitHub Actions CI (Linux/macOS/Windows) — DIST-01
- [ ] goreleaser binary releases on tag — DIST-02
- [ ] Single self-contained binary — DIST-03

### Add After Validation (v1.x)

- [ ] Homebrew tap — triggered by user demand (goreleaser supports it natively)
- [ ] `apt`/`deb` package — triggered by Linux sysadmin audience requests
- [ ] Shell completions — triggered if CLI usage grows beyond difftool mode

### Future Consideration (v2+)

- [ ] Three-way merge view — only if Python original adds it first
- [ ] Git index / staging integration — would require explicit scope expansion decision

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Side-by-side diff pane | HIGH | HIGH | P1 |
| Syntax highlighting (Chroma) | HIGH | MEDIUM | P1 |
| Line-level diff colors | HIGH | LOW | P1 |
| File tree with markers | HIGH | MEDIUM | P1 |
| `n`/`N` hunk navigation | HIGH | MEDIUM | P1 |
| Full-file mode | HIGH | HIGH | P1 |
| Hunk-only mode + `v` toggle | MEDIUM | MEDIUM | P1 |
| Intra-line word diff | HIGH | HIGH | P1 — hardest single feature |
| Adaptive pane width | MEDIUM | MEDIUM | P1 |
| `Tab`/`]`/`[`/`q` navigation | HIGH | LOW | P1 |
| `/` in-pane search | MEDIUM | HIGH | P1 |
| Difftool mode + N of M | MEDIUM | LOW | P1 |
| `install-difftool` subcommand | LOW | LOW | P1 |
| Light/dark/auto theming | MEDIUM | MEDIUM | P1 |
| XDG TOML config + keybindings | LOW | LOW | P1 |
| goreleaser binary distribution | HIGH | LOW | P1 |
| GitHub Actions CI | HIGH | LOW | P1 |

All features are P1 because this is a 1:1 port — there is no MVP subset; all PROJECT.md Active requirements ship together.

---

## Competitor Feature Analysis

| Feature | gitui (Rust) | lazygit (Go) | delta (Rust pager) | alturd Go (target) |
|---------|--------------|--------------|--------------------|--------------------|
| Side-by-side diff | No | Via custom pager only | Yes (its signature feature) | Yes — native |
| Syntax highlighting | Partial | Via pager | Yes | Yes — Chroma |
| Intra-line word diff | No | Via delta pager | Yes — Levenshtein | Yes — custom LCS |
| Full-file mode | No (hunk only) | No | No (pager shows all) | Yes — toggle |
| Hunk-only mode | Yes (default) | Yes (default) | Yes (default) | Yes — toggle |
| `n`/`N` hunk nav | Yes | Via pager | Yes | Yes |
| File tree | Yes | Yes (full git UI) | No | Yes |
| Adaptive pane width | No | No | N/A | Yes |
| `git difftool` mode | No | No | No | Yes |
| In-pane search | File search only | No native | No | Yes |
| Light/dark auto theme | Via theme config | Via pager config | Via `--light` flag | Yes — auto-detect |
| TOML config + keybindings | Yes (keybindings) | Yes (YAML) | Yes (gitconfig) | Yes (TOML/XDG) |
| Single binary | Yes | Yes | Yes | Yes |
| Windows support | Yes | Yes (limited pager) | Yes | Yes |

**Key insight:** No existing Go TUI tool provides side-by-side diff + full-file mode + intra-line word diff as a single interactive TUI. Delta comes closest on features but is a pager (non-interactive). alturd fills a genuine gap.

---

## Port-Specific Considerations

These are the areas where Go/bubbletea differs enough from Python/Textual to require explicit design attention. Each represents a porting complexity that the Python implementation resolved in ways that do not translate directly.

### 1. Side-by-Side Synchronized Scroll (HIGH complexity)

**Python/Textual:** Two independent ScrollableContainer widgets linked by a shared reactive scroll event handler. Textual's reactive system propagates scroll offset changes automatically.

**Go/bubbletea:** No built-in widget linking. Both viewport models are updated manually in the Update function. When a KeyMsg scrolls the active pane, the Update handler must forward the same delta to the passive pane's y_offset. This is not complex logically but must be explicitly designed — it is invisible in Python, explicit in Go.

**Gotcha:** `bubbles/viewport` does not expose its internal y_offset for reading or direct mutation in a clean API. Custom viewport management or direct model field access may be needed.

### 2. Full-File Mode and the Bubbletea Renderer Limit (HIGH complexity)

**Python/Textual:** Full-file content is a first-class concept. Textual's virtual DOM renders only what is visible; the "content" of a widget can be arbitrarily large.

**Go/bubbletea:** The standard renderer passes the entire View() string to the terminal, which re-renders the changed portion. If View() produces more lines than the terminal height, rendering is corrupted. For large files in full-file mode, virtual windowing is **mandatory**: only the visible window (y_offset to y_offset+height lines) of the pre-rendered content should be passed to the renderer.

**Recommended approach:** Pre-render all lines of the file (with syntax highlighting and diff colors applied) into a `[]string` slice. In View(), slice `[y_offset : y_offset+height]` and join with newlines. SetContent() is used only for hunk-only mode where content is always bounded. For full-file mode, manage y_offset and height manually.

### 3. Intra-Line Word Diff — Not Provided by go-diff (HIGH complexity)

**Python implementation:** Uses the Python `difflib.SequenceMatcher` on word-tokenized line content to produce intra-line spans.

**Go:** `sourcegraph/go-diff` parses unified diffs but does NOT compute or expose word-level diff. A secondary pass is required:
1. Identify adjacent remove (`-`) and add (`+`) line pairs in a hunk.
2. Strip ANSI escapes from the raw text content of each pair.
3. Run a Myers or Levenshtein LCS diff on word/char tokens.
4. Map resulting change spans back to byte positions in the ANSI string.
5. Insert highlight markers (e.g., lipgloss reverse-video style) without breaking existing ANSI sequences.

**Blueprint:** revdiff's `worddiff` sub-package implements exactly this pattern and can be studied as a reference.

### 4. Layering Syntax Highlighting + Diff Colors + Intra-Line Markers (HIGH complexity)

**Python/Textual:** Textual's markup system handles layered styles via its CSS-like rich text model. Colors compose naturally.

**Go:** Three independent ANSI layers must be composed:
1. **Chroma** produces syntax highlight ANSI sequences.
2. **Line-level diff colors** (added/removed/modified) need to change the background color of entire lines.
3. **Intra-line word diff** needs to further modify specific spans within those lines.

The challenge is that ANSI SGR codes don't compose naturally — a background-color reset in one layer will clear the background set by another. The correct approach: apply syntax highlighting first, then overlay diff background colors on the entire line, then overlay intra-line highlight spans. Each step must reset and re-apply the appropriate SGR state. The `muesli/termenv` library's SGR tracking utilities help here.

### 5. Terminal Background Detection Reliability (MEDIUM complexity)

**Python/Textual:** Detects via the OS terminal API; reliable in all environments Textual supports.

**Go/termenv:** `termenv.HasDarkBackground()` issues an OSC 11 query to the terminal. This query:
- Works correctly in most terminals (iTerm2, Alacritty, foot, Windows Terminal)
- May time out or return incorrect results in multiplexers (tmux, screen, zellij) unless those multiplexers propagate OSC queries
- May not work over SSH sessions with certain terminal types

**Required fallback:** If `HasDarkBackground()` returns a timeout or ambiguous result (no response within ~100ms), default to dark theme. The TOML config should also expose `theme = "light" | "dark" | "auto"` so users can override.

### 6. ANSI-Safe Column Truncation for Side-by-Side (MEDIUM complexity)

**Python/Textual:** Textual handles truncation at the widget level; developers don't think about it.

**Go:** When a syntax-highlighted line is longer than the available column width, truncating at a fixed byte position may split an ANSI escape sequence, leaving the terminal in a broken color state. Lipgloss's `lipgloss.Width()` counts display width (ANSI-aware), and `ansi.Truncate()` from `muesli/reflow` is the correct primitive for ANSI-safe truncation. This must be applied to every line in every column before joining horizontally.

---

## Sources

- [GitHub: gitui-org/gitui](https://github.com/gitui-org/gitui) — feature list and keybinding reference
- [GitHub: jesseduffield/lazygit — Custom_Pagers.md](https://github.com/jesseduffield/lazygit/blob/master/docs/Custom_Pagers.md) — diff pager integration
- [GitHub: dandavison/delta](https://github.com/dandavison/delta) — side-by-side and word-diff reference implementation
- [GitHub: charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [GitHub: charmbracelet/bubbles — viewport.go](https://github.com/charmbracelet/bubbles/blob/master/viewport/viewport.go) — viewport component API
- [GitHub: charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) — styling and layout
- [pkg.go.dev: alecthomas/chroma/v2](https://pkg.go.dev/github.com/alecthomas/chroma/v2) — syntax highlighting API
- [pkg.go.dev: muesli/termenv](https://pkg.go.dev/github.com/muesli/termenv) — terminal background detection
- [GitHub: sourcegraph/go-diff](https://github.com/sourcegraph/go-diff) — unified diff parsing
- [pkg.go.dev: umputun/revdiff/app/ui](https://pkg.go.dev/github.com/umputun/revdiff/app/ui) — bubbletea diff viewer with search and word-diff blueprint
- [pkg.go.dev: adrg/xdg](https://pkg.go.dev/github.com/adrg/xdg) — XDG Base Directory implementation for Go
- [git-scm.com: git-difftool documentation](https://git-scm.com/docs/git-difftool) — GIT_DIFF_PATH_COUNTER/TOTAL protocol
- [GitHub: JanSmrcka/differ](https://github.com/JanSmrcka/differ) — Go bubbletea two-panel diff viewer (peer project)
- [goreleaser.com](https://goreleaser.com) — binary release toolchain

---

*Feature research for: Go TUI git diff viewer (alturd port)*
*Researched: 2026-06-25*
