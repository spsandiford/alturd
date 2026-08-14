# Alturd (Go Port)

## What This Is

Alturd is a terminal UI application for viewing `git diff` output as a navigable, side-by-side experience. It is a full port of the existing Python/Textual implementation to Go, producing a single self-contained binary that runs on Linux, macOS, and Windows. It is built for developers who live in the terminal and want a Meld/kdiff3-style diff review experience without leaving it or installing Python.

## Core Value

A developer can download a single binary, run `alturd` in any git repository, and navigate every changed file and every individual diff hunk with fast keyboard-driven controls — no runtime dependencies required.

## Requirements

### Validated

- ✓ **DIFF-02**: Chroma syntax highlighting — Phase 1 (Highlight() + per-line reset guard proven against 13-fixture corpus)
- ✓ **DIFF-03**: Line-level diff colors (Added/Removed/Modified 256-colour backgrounds) — Phase 1 (Render/lineBg() implemented and tested)
- ✓ **DIFF-04**: Intra-line character-level change markers on Modified rows — Phase 1 (applyIntraLine() + DiffMain with 100ms timeout guard)
- ✓ **DIFF-05**: Full-file and hunk-only render modes via RenderMode enum — Phase 1 (wired; TUI toggle in Phase 3)
- ✓ **CLI-01**: `alturd [<refs>...] [-- <paths>...]` ref grammar — Phase 2 (ParseRefArgs covers all six invocation forms; exit codes 0/1/127 verified)

### Validated (continued)

- ✓ **TREE-01**: File tree with [A]/[M]/[D]/[R] status markers (256-colour, light/dark adaptive), dirs-first, compact-folder — Phase 3
- ✓ **TREE-02**: Tree pane 45 col focused / 24 col unfocused, instant resize on Tab — Phase 3
- ✓ **TREE-03**: `]`/`[` cycles changed files; `a` toggles changed-only vs full-repo tree via git ls-tree — Phase 3
- ✓ **DIFF-06**: `v` toggles full-file ↔ hunk-only without reload; hunk-only shows diff context lines — Phase 3
- ✓ **NAV-01**: `n`/`N` jumps between hunks, centered via SetYOffset — Phase 3
- ✓ **NAV-02**: `]`/`[` cycles files with wraparound — Phase 3
- ✓ **NAV-03**: `Tab` switches focus between tree and diff pane — Phase 3
- ✓ **NAV-04**: `q` exits 0; `Q` exits 1 — Phase 3
- ✓ **SEARCH-01**: `/` opens in-pane search with ANSI-aware match highlighting and n/N navigation — Phase 3
- ✓ **THEME-01**: Light/dark/auto theming, OSC 11 detection with 50ms timeout fallback to dark — Phase 4
- ✓ **CONFIG-01**: TOML config file at `$XDG_CONFIG_HOME/alturd/config.toml` (or `--config <path>`) overrides defaults — Phase 4
- ✓ **CONFIG-02**: Any default keybinding overridable via the config file — Phase 4
- ✓ **DIFFTOOL-01**: `git difftool -t alturd <file>` launches a single-file side-by-side view without the file tree — Phase 4 (gap G-04-1 closed in 04-07: abort key no longer triggers git's fatal external-diff crash)
- ✓ **DIFFTOOL-02**: TitleBar shows "alturd (difftool) — N of M — `<filename>`" via `GIT_DIFF_PATH_COUNTER`/`GIT_DIFF_PATH_TOTAL` — Phase 4
- ✓ **DIFFTOOL-03**: `alturd install-difftool [--scope global|local] [--name NAME] [--force]` writes the four canonical gitconfig keys idempotently — Phase 4
- ✓ **DIST-01**: GitHub Actions CI builds and tests on every push (Linux, macOS, Windows) — Phase 4
- ✓ **DIST-02**: GitHub Actions releases pre-built binaries for Linux/macOS/Windows on git tag — Phase 4
- ✓ **DIST-03**: Single self-contained binary (`CGO_ENABLED=0`) — no runtime dependencies — Phase 4

### Active

- [ ] **CLI-01**: `alturd [<refs>...] [-- <paths>...]` ref grammar matches Python implementation exactly

### Out of Scope

- Python/Textual dependencies at runtime — the whole point of the port is a dependency-free binary
- Three-way merge / conflict resolution — was deferred in Python version; stays out of scope here
- PyPI/pip distribution — the Go port distributes via GitHub Releases binaries
- Large-file streaming or lazy loading — explicitly not a concern for this port
- Editing files in-place from the diff view — alturd is a reviewer, not an editor
- Snapshot-based TUI tests (pytest-textual-snapshot equivalent) — Go TUI testing approach to be determined during planning

## Context

- **Source application**: The Python/Textual implementation at v1.1 (shipped 2026-06-19) is the authoritative reference. It has 11,629 LOC Python source, 689 passing tests, and a well-documented architecture in `.planning/` including fixture corpus and detailed phase artifacts.
- **Why Go**: Single binary distribution across Linux/macOS/Windows without a Python runtime. Go's cross-compilation makes this trivial; the result is a drop-in replacement users can just download and run.
- **TUI ecosystem**: Bubbletea (Charm) is the idiomatic Go TUI framework — reactive, composable, actively maintained. Lipgloss handles styling. This is the Go analogue of Textual.
- **Syntax highlighting**: Chroma is the Go port of Pygments (same author, same language database). Direct replacement.
- **Diff parsing**: The Python implementation uses `unidiff`; Go has `sourcegraph/go-diff` for unified diff parsing, or the diff parser can be ported directly from the well-tested Python implementation given its 12-scenario fixture corpus.
- **Two distinct entry points**: Standalone mode (file tree + diff pane) and difftool mode (per-file, skips tree). Both share the same diff renderer. This architecture maps directly to Go.
- **Existing test fixtures**: The Python implementation has a rich fixture corpus under `tests/fixtures/diff/` (binary pairs, renames, no-newline edges, etc.) that should be reused to validate the Go diff parser.

## Constraints

- **Target language**: Go — modern version (1.22+), chosen for single-binary cross-platform distribution
- **TUI framework**: Bubbletea + Lipgloss — the standard Go TUI stack, analogous to Textual
- **Cross-platform**: Must compile and run correctly on Linux (amd64/arm64), macOS (amd64/arm64), Windows (amd64)
- **Distribution**: GitHub Releases via GitHub Actions — no external registry or packaging system required
- **Behavior parity**: CLI grammar, exit codes, keybindings, config file location/format, and difftool integration must match the Python implementation so existing users can switch transparently
- **No runtime deps**: The final binary must be fully self-contained (CGO disabled where possible for portability)
- **Git compatibility**: Talks to git via its CLI/plumbing commands only, not via libgit2

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Port to Go (not Rust/other) | Go produces small cross-platform binaries with trivial cross-compilation; bubbletea/lipgloss are mature; chroma is an exact Pygments analogue | — Pending |
| Bubbletea + Lipgloss for TUI | The standard Go TUI stack; most comparable to Textual's reactive model; actively maintained by Charm | — Pending |
| Chroma for syntax highlighting | Direct Go port of Pygments by same author — same language database, same API surface | Confirmed — Phase 1; per-line ANSI reset guard required (Pitfall 2) |
| Reuse Python fixture corpus | 12-scenario diff fixture corpus in `tests/fixtures/diff/` validates edge cases; porting the fixtures eliminates re-discovering parser bugs | Confirmed — Phase 1; hand-crafted 13-scenario Go corpus (Python repo unavailable); 3 fixtures needed @@ count fixes |
| GitHub Actions + goreleaser for distribution | goreleaser handles cross-compilation matrix and GitHub Release artifact management cleanly | — Pending |
| Go module go directive set to 1.25 | chroma/v2 v2.27.0 requires `go 1.25` minimum; CLAUDE.md "Go 1.22+" constraint is satisfied by 1.25 | Confirmed — Phase 1 |
| `git diff` exit code unreliable for "not a git repo" | Exits 129 (not 128) on Git 2.39+ on Linux; stderr message "not a git repository" is the only portable discriminator | Confirmed — Phase 2 UAT |
| `var version` not `const` for goreleaser override | goreleaser `-ldflags "-X main.version=<tag>"` requires a `var` (linker cannot set a `const`) | Confirmed — Phase 2 |
| TestMain subprocess pattern for integration tests | `t.TempDir()` is cleaned up when the first subtest finishes, invalidating a shared binary path; `os.MkdirTemp` + `defer os.RemoveAll` in `TestMain` persists the binary for the full test run | Confirmed — Phase 2 |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-08-14 after Phase 04.1 — tech-debt sweep closed: WR-07 (difftool deleted-file rendering) fixed with regression coverage, gofmt/golangci-lint drift now CI-enforced, REQUIREMENTS.md traceability table resynced, Phase 4 Nyquist record reconciled out of draft, and Phase 3's long-outstanding human UAT checkpoint executed and recorded (11/11 pass, including TREE-01 colour rendering). All 30 v1 requirements remain Validated; none were newly satisfied by this phase.*
