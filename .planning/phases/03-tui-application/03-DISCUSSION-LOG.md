# Phase 3: TUI Application - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-01
**Phase:** 3-TUI Application
**Areas discussed:** Tree pane width animation, Search mode key conflicts, Compact-folder tree layout, Startup blank-screen handling

---

## Tree pane width animation

| Option | Description | Selected |
|--------|-------------|----------|
| Instant resize | Width snaps 24↔45 in one frame. No tick commands, no animation state. | ✓ |
| Single intermediate step | 24→35→45 over 2 ticks (~30ms each). One extra model field. | |
| Smooth multi-frame | Gradual steps over 6-8 ticks. Full animation state machine. | |

**User's choice:** Instant resize

---

| Option | Description | Selected |
|--------|-------------|----------|
| Ellipsis at end | Long names as `src/internal/diff...` via `lipgloss.Style.MaxWidth()` | ✓ |
| Ellipsis in middle | Long names as `src/int...diff.go` — preserves extension | |
| You decide | Leave to researcher/planner | |

**User's choice:** Ellipsis at end

---

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, always fills remaining width | diff pane = terminal_width - tree_width - 1 | ✓ |
| Diff pane has a fixed minimum width | Won't shrink below N columns | |

**User's choice:** Always fills remaining width

---

| Option | Description | Selected |
|--------|-------------|----------|
| Single-character vertical bar │ | 1-col Unicode box-drawing separator | ✓ |
| Colored border via lipgloss | lipgloss Border style, changes color on focus | |
| No separator (flush) | No column consumed, panes touch | |

**User's choice:** Single-character vertical bar │

---

## Search mode key conflicts

| Option | Description | Selected |
|--------|-------------|----------|
| n/N navigates matches while open, hunks when closed | Vim-style mode dispatch | ✓ |
| Different keys for match navigation | Enter/Shift+Enter for matches, n/N always hunks | |
| n/N always means hunks | Search highlights but no n/N match navigation | |

**User's choice:** n/N navigates matches while open, hunks when closed

---

| Option | Description | Selected |
|--------|-------------|----------|
| Clear text and highlights | Esc closes input, removes all highlights | ✓ |
| Preserve text, hide input bar | Highlights stay visible after Esc | |
| You decide | Leave to planner to match Python behavior | |

**User's choice:** Clear text and highlights on Esc

---

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — ]/[ cycles files, closes search automatically | File nav closes search, no carry-over | ✓ |
| No — ]/[ suppressed while search is open | Search blocks all navigation except Esc and n/N | |
| Yes — cycles files, search query carries over | Same query applied to next file | |

**User's choice:** ]/[ cycles files, closes search automatically

---

| Option | Description | Selected |
|--------|-------------|----------|
| Bottom of the diff pane | 1-row input bar at bottom, viewport shrinks by 1 | ✓ |
| Top of the diff pane | Input bar above diff content | |
| Floating overlay at bottom of terminal | Full-width bar, not scoped to diff pane | |

**User's choice:** Bottom of the diff pane

---

## Compact-folder tree layout

| Option | Description | Selected |
|--------|-------------|----------|
| GitHub-style path collapsing | Single-child dir chains collapse to one node | ✓ |
| Flat list with path prefixes | All files as flat list, no collapsing | |
| You decide — match Python implementation exactly | Researcher replicates Python behavior | |

**User's choice:** GitHub-style path collapsing

---

| Option | Description | Selected |
|--------|-------------|----------|
| Dirs-first sort, dirs shown as expandable nodes | ▸/▾ markers, dirs before files | ✓ |
| Flat alphabetical, no expand/collapse | All entries alphabetical, no tree structure | |

**User's choice:** Dirs-first sort with expandable nodes

---

| Option | Description | Selected |
|--------|-------------|----------|
| Run `git ls-tree -r HEAD` for full-repo tree | Git-tracked files, .gitignore handled automatically | ✓ |
| Walk the filesystem from repo root | os.ReadDir walk with .gitignore filtering | |
| You decide | Researcher finds most reliable approach | |

**User's choice:** `git ls-tree -r HEAD`

---

| Option | Description | Selected |
|--------|-------------|----------|
| Inverted background on selected row | Reverse video on selected file | ✓ |
| Bold text with a > prefix | Selected shown as `> filename` in bold | |
| You decide | Planner picks consistent with Phase 4 theming | |

**User's choice:** Inverted background on selected row

---

## Startup blank-screen handling

| Option | Description | Selected |
|--------|-------------|----------|
| Defer render until WindowSizeMsg | `ready bool` in model, View() returns "" until set | ✓ |
| Use a sensible default size (80x24) | Initialize with defaults, reflow on WindowSizeMsg | |
| Show a loading spinner | bubbles/spinner until data + size ready | |

**User's choice:** Defer render until WindowSizeMsg (ready bool pattern)

---

| Option | Description | Selected |
|--------|-------------|----------|
| Before TUI starts — pass pre-parsed data into model | main.go runs git+parse before tea.NewProgram() | ✓ |
| Inside TUI as async tea.Cmd | Model fires tea.Cmd on Init(), loading state | |
| You decide | Planner chooses loading strategy | |

**User's choice:** Pre-load before TUI starts

---

| Option | Description | Selected |
|--------|-------------|----------|
| Researcher investigates bubbletea v2 issue #1601 | Current recommended workaround may have changed | ✓ |
| Poll terminal size on a 500ms tick | tea.Tick every 500ms, synthetic WindowSizeMsg if changed | |
| Accept initial size only on Windows | No polling, users restart to resize | |

**User's choice:** Researcher investigates and recommends

---

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — minimal status bar at top | `alturd — <filename> (N of M changed files)` + mode indicator | ✓ |
| No header in Phase 3, add in Phase 4 | Skip for now, Phase 4 adds it for difftool mode | |
| You decide | Leave to planner | |

**User's choice:** Include minimal status bar in Phase 3

---

## Claude's Discretion

- Exact bubbletea model architecture (single top-level vs. nested sub-models, message type design)
- ANSI-aware search scanner implementation details
- Hunk-centering context line count for n/N in full-file mode (default: 3 lines)
- `bubbles/viewport` vs. custom viewport (use `bubbles/viewport` per CLAUDE.md)
- Expand/collapse interaction for directory nodes in the tree

## Deferred Ideas

None — discussion stayed within phase scope.
