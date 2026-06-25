# Research Summary: alturd (Go TUI Git Diff Viewer)

**Synthesized:** 2026-06-25
**Sources:** STACK.md, FEATURES.md, ARCHITECTURE.md, PITFALLS.md
**Overall Confidence:** MEDIUM — all findings cross-verified against pkg.go.dev, official repos, and community issue trackers

---

## Executive Summary

alturd is a Go port of a Python/Textual interactive git diff viewer. The core value proposition is a feature-complete, single self-contained binary that delivers side-by-side diffs with syntax highlighting, full-file mode, intra-line word diffs, and `git difftool` integration — a feature combination that no existing Go TUI tool (gitui, lazygit, delta) currently offers in one interactive package. The stack is straightforward: bubbletea v2 (MVU TUI framework), lipgloss v2 (layout/styling), chroma v2 (syntax highlighting), go-gitdiff (diff parsing), and goreleaser (cross-platform distribution). The charm.land v2 ecosystem released together in February 2026 and is production-validated.

The recommended architecture is a strict four-layer dependency graph: `internal/git` (subprocess chokepoint) → `internal/diff` (parse + align + highlight) → `internal/config` (TOML/keybindings/theme) → `internal/ui` (bubbletea models). No cycles are possible because Go's `internal/` package visibility rules enforce the import direction. Build order should proceed bottom-up: diff model first (highest risk due to 12 edge-case fixture scenarios), then git layer, then config, then TUI components. This maximizes testability before any TUI code exists.

The principal technical risks are ANSI-layer composition (chroma output + line-level diff colors + intra-line word-diff markers must be stacked without state bleeding), bubbletea-specific constraints (View() must handle zero dimensions at startup; all I/O must go through tea.Cmd; large file content requires manual virtual windowing), and cross-platform reliability (Windows resize events are broken in bubbletea v2 and require a polling workaround; CRLF normalization must occur at the git exec boundary; OSC 11 background detection fails silently in tmux/SSH). All of these risks are well-understood with documented mitigations.

---

## Key Findings

### From STACK.md

| Technology | Version | Rationale |
|------------|---------|-----------|
| Go | 1.22+ | Required for bubbletea v2 |
| charm.land/bubbletea/v2 | v2.0.7 | Idiomatic Go TUI framework; Cursed Renderer 10x faster than v1; production-validated |
| charm.land/lipgloss/v2 | v2.x | Companion to bubbletea v2; JoinHorizontal/JoinVertical compose panes without manual ANSI arithmetic |
| charm.land/bubbles/v2 | v2.x | Viewport and TextInput components |
| github.com/alecthomas/chroma/v2 | v2.27.0 | Pure-Go Pygments port; 200+ language lexers; do not use v3 alpha |
| github.com/bluekeyes/go-gitdiff | v0.8.1 | Handles full git diff including binary patches, renames, extended headers |
| github.com/sergi/go-diff | v1.4.0 | Go diff-match-patch port for intra-line character diff |
| github.com/pelletier/go-toml/v2 | v2.4.2 | TOML config with DisallowUnknownFields |
| github.com/muesli/termenv | v0.16.0 | Terminal background detection for auto-theme |
| goreleaser | v2.16 | Cross-platform binary distribution |

Critical: Go 1.22+; bubbletea/lipgloss/bubbles must all be v2 (matching imports); chroma v2.27.0 not v3 alpha; goreleaser-action@v7.

Non-starters: bubbletea v1 (bugfix-only), chroma v3 (alpha), CGO-enabled builds, libgit2/go-git (CLI/plumbing only per PROJECT.md).

### From FEATURES.md

**Table stakes (must have for Python parity):** side-by-side diff pane with synchronized scroll; syntax highlighting; line-level diff colors; n/N hunk navigation; q/Tab/]/[ keyboard navigation; file tree with [A]/[M]/[D]/[R] markers, dirs-first, compact-folder layout; single self-contained binary; alturd CLI grammar; cross-platform Linux/macOS/Windows.

**Differentiators (make alturd unique):**
- Full-file mode with v toggle to hunk-only — no peer Go TUI tool offers this interactively
- Adaptive tree pane width (45 focused / 24 unfocused) — unique in Go TUI space
- Intra-line word/character diff markers — hardest single feature; secondary LCS pass over ANSI-stripped text
- git difftool integration with N of M counter
- alturd install-difftool subcommand for zero-friction gitconfig setup
- / in-pane search with match highlighting — ANSI-aware scanner required
- Light/dark/auto theming via runtime OSC 11 detection

**Anti-features (explicitly excluded):** staging hunks, branch/commit management, three-way merge, mouse navigation, in-app git operations.

**This is a 1:1 port — all 21 features in the Launch checklist are P1. There is no MVP subset.**

### From ARCHITECTURE.md

Four-layer structure enforced by Go internal/ visibility:

```
cmd/alturd/          — entry point; mode dispatch
internal/git/        — subprocess chokepoint (no internal imports)
internal/diff/       — parse, align, highlight (imports config)
internal/config/     — TOML, keybindings, theme (no internal imports)
internal/ui/         — bubbletea models and rendering (imports diff + config)
```

Key patterns:
1. Pre-render ANSI strings on FileSelected message, never inside View() — chroma tokenization is expensive per frame
2. Message-passing for cross-pane events (FileSelected, HunkJump, ModeToggle)
3. WindowSizeMsg propagation — AppModel recalculates pane widths and propagates to children
4. Separate model.go / view.go per sub-model — keeps files under 300 lines
5. Typed errors from git layer — main.go switches on error type for clean UX messages

Build order (risk-driven): diff parser → git layer → config → chroma highlight → diffview UI → filetree UI → AppModel → cmd entry point → goreleaser

### From PITFALLS.md

**Critical pitfalls (Phase 1):**
- View() before WindowSizeMsg: add ready bool; return "Initializing..." until first WindowSizeMsg
- Blocking Update() with I/O: all git calls and diff computation in tea.Cmd
- Panic in tea.Cmd leaves terminal in raw mode: defer recover() in every non-trivial tea.Cmd
- Windows resize never fires (bubbletea v2 regression #1601): polling workaround via 250ms tick behind Windows build tag

**Critical pitfalls (Phase 2):**
- len() for ANSI string width causes column misalignment: use runewidth.StringWidth() or lipgloss.Width() everywhere
- ANSI color bleed across side-by-side columns: emit explicit reset at end of every left-column line
- Viewport ghost lines on file switch (issue #1477): force repaint via WindowSizeMsg after SetContent

**Critical pitfalls (Phase 3):**
- CRLF in git output on Windows: normalize \r\n → \n immediately after cmd.Output()
- OSC 11 hangs in tmux/SSH: 50–100ms timeout; fallback to COLORFGBG; default to dark

**Moderate pitfalls (Phase 4):**
- Windows VTP disabled in cmd.exe: document PowerShell requirement; use shell: pwsh in CI
- CGO_ENABLED inconsistency: explicitly set env: [CGO_ENABLED=0] in .goreleaser.yaml
- goreleaser needs fetch-depth: 0 in GitHub Actions checkout

---

## Implications for Roadmap

### Suggested Phase Structure

**Phase 1: TUI Scaffold and Foundation**
Rationale: Establishes bubbletea v2 patterns that everything else builds on; contains the highest-risk TUI gotchas before complex rendering is added.
Delivers: Compilable multi-pane TUI shell with AppModel + FileTreeModel + DiffViewModel skeletons; WindowSizeMsg propagation; focus switching; tea.Cmd error-return pattern; Windows resize polling workaround; ready flag guard.
Features: NAV-02 (Tab), q-to-quit, window size handling, project scaffold.
Must avoid: Pitfalls 1, 2, 5, 12 (View before WindowSizeMsg; Windows resize; blocking Update; panic in tea.Cmd).
Research flag: STANDARD — bubbletea v2 multi-pane patterns are well-documented.

**Phase 2: Diff Parser and Renderer**
Rationale: The diff parser is the highest-correctness-risk component (12 edge-case fixture scenarios); must be fully validated before TUI code consumes it; ANSI layer composition is the hardest technical challenge.
Delivers: internal/diff (model + parse + align + highlight) with table tests against Python fixture corpus; side-by-side column rendering with ANSI-safe truncation, color bleed prevention, runewidth-correct alignment; full-file virtual windowing; hunk-only mode; v toggle; intra-line word diff.
Features: DIFF-01 through DIFF-06.
Must avoid: Pitfalls 3, 6, 4 (len() width; ANSI color bleed; viewport ghost lines).
Research flag: NEEDS RESEARCH — intra-line word diff ANSI composition; revdiff worddiff sub-package is the known blueprint but implementation details need spiking.

**Phase 3: Git Integration, Navigation, and Theming**
Rationale: Once the renderer is validated, connect real git data sources and add all interactive navigation.
Delivers: internal/git exec wrapper; CRLF normalization; CLI argument parsing; file tree with markers and compact-folder layout; ]/[ file cycling; a toggle; n/N hunk navigation; / in-pane search; light/dark/auto theming with OSC 11 timeout+fallback; git difftool mode; install-difftool subcommand; XDG TOML config with configurable keybindings.
Features: TREE-01 through TREE-03, NAV-01, SEARCH-01, THEME-01, CONFIG-01, CLI-01, CLI-02, HELPER-01, HELPER-02.
Must avoid: Pitfalls 9, 10, 11 (CRLF; OSC 11 hang; difftool signal handling).
Research flag: STANDARD for git integration and navigation; NEEDS RESEARCH for in-pane search ANSI-aware marker insertion.

**Phase 4: Cross-Platform CI and Distribution**
Rationale: Cap the project with production-ready release infrastructure; validates all platforms.
Delivers: .goreleaser.yaml with CGO_ENABLED=0 and linux/darwin/windows × amd64/arm64 matrix; GitHub Actions CI (test + vet + lint on all three platforms); release trigger on tag push; cross-platform end-to-end tests including Windows cmd.exe launch test and Unicode fixture tests.
Features: DIST-01, DIST-02, DIST-03.
Must avoid: Pitfalls 7, 8, goreleaser fetch-depth (Windows VTP in cmd.exe; CGO_ENABLED inconsistency).
Research flag: STANDARD — goreleaser v2 + GitHub Actions is standard configuration.

### Research Flags

| Phase | Research Needed | Reason |
|-------|----------------|--------|
| Phase 2 | YES | Intra-line word diff ANSI composition is novel; revdiff worddiff is the only known Go blueprint; needs spiking before scoping |
| Phase 3 | YES (partial) | In-pane search ANSI marker insertion is non-trivial byte-level manipulation in already-encoded chroma output |
| Phase 1 | No | bubbletea v2 multi-pane scaffold is well-documented |
| Phase 4 | No | goreleaser v2 + GitHub Actions is standard configuration |

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | MEDIUM-HIGH | All versions verified against pkg.go.dev as of 2026-06-25; one active regression (bubbletea v2 #1601 Windows resize) has a workaround but no upstream fix |
| Features | MEDIUM | Cross-referenced Python original + PROJECT.md + peer tool analysis; all 21 launch features have clear implementation paths |
| Architecture | MEDIUM | Four-layer structure well-grounded in Go conventions; bubbletea sub-model patterns from revdiff are real-world validated; viewport virtual windowing for full-file mode is design-derived |
| Pitfalls | MEDIUM-HIGH | 12 pitfalls sourced from bubbletea issue tracker, community posts, and OS documentation; Windows-specific pitfalls are especially well-evidenced with GitHub issue numbers |

### Gaps to Address During Planning

1. **Intra-line word diff spike:** The algorithm is clear but the exact API for re-inserting spans into ANSI-encoded strings needs a prototype before Phase 2 task scoping.
2. **Viewport y_offset access:** ARCHITECTURE.md notes bubbles/viewport may not expose y_offset cleanly for synchronized scroll. Needs investigation before DiffView is planned.
3. **Python fixture corpus location:** ARCHITECTURE.md references "12 Python fixture files." Confirm these exist in the repo before Phase 2 planning.
4. **Windows CI cost:** GitHub Actions Windows runners are 3–5x slower than Linux. Phase 4 planning should decide on full vs targeted Windows test suite.

---

## Sources (Aggregated)

- pkg.go.dev: charm.land/bubbletea/v2, lipgloss/v2, bubbles/v2, alecthomas/chroma/v2, bluekeyes/go-gitdiff, sergi/go-diff, pelletier/go-toml/v2, muesli/termenv
- GitHub: charmbracelet/bubbletea issues #1477 (ghost lines), #1601 (Windows resize), #282 (View before WindowSizeMsg)
- GitHub: charmbracelet/bubbletea discussions #1374 (v2 What's New)
- GitHub: gitui-org/gitui, jesseduffield/lazygit, dandavison/delta (competitor analysis)
- GitHub: umputun/revdiff (real-world bubbletea diff viewer blueprint)
- goreleaser.com: v2 build and CI/Actions documentation
- go.dev: official Go module layout guide
- leg100.github.io: Tips for building Bubble Tea programs
- Microsoft: Console Virtual Terminal Sequences documentation
- tmux issues #1919, #3582: OSC 11 support and caching
- git-scm.com: git-difftool documentation

---

*Research synthesis for: alturd — Go TUI git diff viewer port from Python/Textual*
*Synthesized: 2026-06-25*
