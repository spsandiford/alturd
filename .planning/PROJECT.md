# Alturd (Go Port)

## What This Is

Alturd is a terminal UI application for viewing `git diff` output as a navigable, side-by-side experience. It is a full port of the existing Python/Textual implementation to Go, producing a single self-contained binary that runs on Linux, macOS, and Windows. It ships as a standalone CLI/TUI and as a registered `git difftool` backend, giving developers who live in the terminal a Meld/kdiff3-style diff review experience without leaving it or installing Python.

## Current State

**Shipped: v1.0** (2026-08-17) — 5 phases, 21 plans, 30/30 v1 requirements satisfied. All phase verifications passed; Phase 3's outstanding human UAT checkpoint and TREE-01 colour confirmation were closed in Phase 04.1's tech-debt sweep, along with WR-07 (difftool deleted-file rendering), gofmt/lint CI enforcement, and REQUIREMENTS.md traceability sync. See `.planning/milestones/v1.0-ROADMAP.md` and `.planning/MILESTONES.md` for full detail.

## Core Value

A developer can download a single binary, run `alturd` in any git repository, and navigate every changed file and every individual diff hunk with fast keyboard-driven controls — no runtime dependencies required.

Still the right priority after shipping: v1.0 delivered exactly this (standalone mode + difftool mode, both fully keyboard-driven, single `CGO_ENABLED=0` binary). No shift in core value observed during the milestone.

## Requirements

### Validated

- ✓ **GIT-01**: No-args diff of working tree vs HEAD — v1.0 (Phase 2)
- ✓ **GIT-02**: `<ref>`, `<ref1>..<ref2>`, `<ref1>...<ref2>`, `<ref1> <ref2>` ref grammar — v1.0 (Phase 2)
- ✓ **GIT-03**: `-- <paths>` filtering — v1.0 (Phase 2)
- ✓ **GIT-04**: `--version`/`--help` exit cleanly, no log/side effects — v1.0 (Phase 2)
- ✓ **GIT-05**: Clear single-line error outside a git repo (exit 1) or git missing (exit 127) — v1.0 (Phase 2, confirmed 02-UAT.md)
- ✓ **DIFF-01**: Aligned parallel side-by-side old/new columns — v1.0 (Phase 1)
- ✓ **DIFF-02**: Chroma syntax highlighting (200+ languages) — v1.0 (Phase 1)
- ✓ **DIFF-03**: Line-level diff colors layered with syntax highlighting — v1.0 (Phase 1)
- ✓ **DIFF-04**: Intra-line character-level change markers (1000-char/200-token/100ms guards) — v1.0 (Phase 1)
- ✓ **DIFF-05**: Full-file mode by default — v1.0 (Phase 1, wired; toggle in Phase 3)
- ✓ **DIFF-06**: `v` toggles full-file/hunk-only without reload — v1.0 (Phase 3)
- ✓ **DIFF-07**: Binary/rename/mode-only/submodule/no-newline-at-EOF placeholder rendering — v1.0 (Phase 1)
- ✓ **NAV-01**: `n`/`N` jumps between hunks, centered — v1.0 (Phase 3)
- ✓ **NAV-02**: `]`/`[` cycles changed files with wraparound — v1.0 (Phase 3)
- ✓ **NAV-03**: `Tab` switches focus between tree and diff pane — v1.0 (Phase 3)
- ✓ **NAV-04**: `q` exits 0; `Q` aborts in difftool mode (exit 1) — v1.0 (Phase 3)
- ✓ **TREE-01**: Colored `[A]`/`[M]`/`[D]`/`[R]` status markers, dirs-first, compact-folder — v1.0 (Phase 3; colour rendering human-confirmed in Phase 04.1)
- ✓ **TREE-02**: 45 col focused / 24 col unfocused, instant resize on Tab — v1.0 (Phase 3; instant resize accepted over animated per PROJECT decision below)
- ✓ **TREE-03**: `a` toggles changed-only vs full-repo tree via `git ls-tree` — v1.0 (Phase 3)
- ✓ **SEARCH-01**: `/` in-pane search, ANSI-aware match highlighting, `n`/`N` navigation — v1.0 (Phase 3)
- ✓ **THEME-01**: Light/dark/auto theming, OSC 11 detection with 50ms timeout fallback to dark — v1.0 (Phase 4)
- ✓ **CONFIG-01**: TOML config at `$XDG_CONFIG_HOME/alturd/config.toml` (or `--config`) — v1.0 (Phase 4)
- ✓ **CONFIG-02**: Any default keybinding overridable via config file — v1.0 (Phase 4)
- ✓ **DIFFTOOL-01**: `git difftool -t alturd <file>` single-file view without tree — v1.0 (Phase 4; gaps G-04-1 abort-crash and G-04-2 recursive-spawn closed same phase)
- ✓ **DIFFTOOL-02**: TitleBar "alturd (difftool) — N of M — `<filename>`" via `GIT_DIFF_PATH_COUNTER`/`GIT_DIFF_PATH_TOTAL` — v1.0 (Phase 4)
- ✓ **DIFFTOOL-03**: `alturd install-difftool [--scope global|local] [--name NAME] [--force]` — v1.0 (Phase 4)
- ✓ **DIST-01**: GitHub Actions CI on every push (Linux/macOS/Windows) — v1.0 (Phase 4)
- ✓ **DIST-02**: goreleaser publishes binaries on git tag — v1.0 (Phase 4)
- ✓ **DIST-03**: `CGO_ENABLED=0` fully self-contained binaries — v1.0 (Phase 4)
- ✓ **LOG-01**: Log at `$XDG_STATE_HOME/alturd/alturd.log`, 1MB truncation, never stderr — v1.0 (Phase 2)

**30/30 v1 requirements validated. 0 unsatisfied, 0 orphaned** — confirmed by `.planning/v1.0-MILESTONE-AUDIT.md` 3-source cross-reference (REQUIREMENTS.md × VERIFICATION.md × SUMMARY.md).

### Active

(None — v1.0 requirements fully validated. Fresh requirements for the next milestone are defined via `/gsd-new-milestone`.)

### Out of Scope

- Python/Textual dependencies at runtime — the whole point of the port is a dependency-free binary
- Three-way merge / conflict resolution — was deferred in Python version; stays out of scope here
- PyPI/pip distribution — the Go port distributes via GitHub Releases binaries
- Large-file streaming or lazy loading — not a concern for this port; revisit only if a real user hits a wall
- Editing files in-place from the diff view — alturd is a reviewer, not an editor
- Mouse navigation in diff pane — keyboard-first; mouse support is a v2+ consideration
- In-app git operations (staging, commit, branch) — alturd is a diff viewer only; gitui/lazygit cover full git TUI
- libgit2 / go-git bindings — git CLI/plumbing only, per constraint
- Windows arm64 binary — no goreleaser cross-compilation support for this target currently
- Snapshot-based TUI tests (pytest-textual-snapshot equivalent) — Go TUI testing approach used table tests + human UAT instead; revisit if regressions slip through

All reasons re-audited at v1.0 close — still valid, no items invalidated during the milestone.

## Context

- **Current codebase**: ~7,500 LOC Go across 6 packages (`cmd/alturd`, `internal/config`, `internal/diff`, `internal/git`, `internal/log`, `internal/tui`). `go build ./...` and `go test ./...` clean.
- **Source application**: The Python/Textual implementation at v1.1 (shipped 2026-06-19) was the authoritative reference (11,629 LOC Python, 689 tests). The Go port is now the live implementation.
- **Timeline**: First commit 2026-06-25, v1.0 shipped 2026-08-17 (~53 days), 5 phases, 21 plans.
- **Known tech debt** (non-blocking, tracked in `.planning/WINDOWS.md` ledger and `.planning/STATE.md` Deferred Items):
  - `go test -race ./...` has never run in the dev sandbox (no C toolchain, `CGO_ENABLED=0` environment) — needs a run in an environment with gcc/cc (e.g. CI) before fully closing WR-07's verification bar. Non-race suite is fully green; the changed code path is synchronous.
  - Two debug session files (`DEBUG-difftool-trustexitcode-fatal`, `difftool-recursive-diff-loop`) are marked `status: diagnosed` rather than `fixed` — their root causes were actually fixed in Phase 4 (G-04-1, G-04-2) with regression tests, but the session files' status field was never flipped. Bookkeeping-only, no functional gap.
- **Security**: 49/49 STRIDE threats closed in Phase 4's security review (32 verified in implementation, 17 accepted risk). See `04-SECURITY.md`.
- **Human UAT coverage**: All 5 phases now have a recorded human UAT pass (02-UAT.md, 04-UAT.md, 04.1-UAT.md covering Phase 3's primary split-screen flow + TREE-01 colour + Phase 4's difftool live-terminal test).

## Constraints

- **Target language**: Go — modern version (1.22+), chosen for single-binary cross-platform distribution
- **TUI framework**: Bubbletea + Lipgloss — the standard Go TUI stack, analogous to Textual
- **Cross-platform**: Must compile and run correctly on Linux (amd64/arm64), macOS (amd64/arm64), Windows (amd64)
- **Distribution**: GitHub Releases via GitHub Actions — no external registry or packaging system required
- **Behavior parity**: CLI grammar, exit codes, keybindings, config file location/format, and difftool integration must match the Python implementation so existing users can switch transparently
- **No runtime deps**: The final binary must be fully self-contained (CGO disabled where possible for portability)
- **Git compatibility**: Talks to git via its CLI/plumbing commands only, not via libgit2

No constraints changed during v1.0 development.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Port to Go (not Rust/other) | Go produces small cross-platform binaries with trivial cross-compilation; bubbletea/lipgloss are mature; chroma is an exact Pygments analogue | ✓ Confirmed — v1.0 shipped, CGO_ENABLED=0 binaries for 5 platform targets |
| Bubbletea + Lipgloss for TUI | The standard Go TUI stack; most comparable to Textual's reactive model; actively maintained by Charm | ✓ Confirmed — v1.0 Phase 3, full interactive split-screen app |
| Chroma for syntax highlighting | Direct Go port of Pygments by same author — same language database, same API surface | ✓ Confirmed — Phase 1; per-line ANSI reset guard required (Pitfall 2) |
| Reuse Python fixture corpus | 12-scenario diff fixture corpus in `tests/fixtures/diff/` validates edge cases; porting eliminates re-discovering parser bugs | ✓ Confirmed — Phase 1; hand-crafted 13-scenario Go corpus (Python repo unavailable); 3 fixtures needed @@ count fixes |
| GitHub Actions + goreleaser for distribution | goreleaser handles cross-compilation matrix and GitHub Release artifact management cleanly | ✓ Confirmed — Phase 4, 3-OS CI matrix + 5-target release on tag |
| Go module go directive set to 1.25 | chroma/v2 v2.27.0 requires `go 1.25` minimum; CLAUDE.md "Go 1.22+" constraint is satisfied by 1.25 | ✓ Confirmed — Phase 1 |
| `git diff` exit code unreliable for "not a git repo" | Exits 129 (not 128) on Git 2.39+ on Linux; stderr message "not a git repository" is the only portable discriminator | ✓ Confirmed — Phase 2 UAT |
| `var version` not `const` for goreleaser override | goreleaser `-ldflags "-X main.version=<tag>"` requires a `var` (linker cannot set a `const`) | ✓ Confirmed — Phase 2 |
| TestMain subprocess pattern for integration tests | `t.TempDir()` is cleaned up when the first subtest finishes, invalidating a shared binary path; `os.MkdirTemp` + `defer os.RemoveAll` in `TestMain` persists the binary for the full test run | ✓ Confirmed — Phase 2 |
| TREE-02 resize is instant, not animated | Bubbletea's synchronous render model makes animated resize a significant added-complexity item with no clear UX win over instant resize | ✓ Confirmed — Phase 3, accepted via PROJECT decision |
| Abort key routes through `tea.Quit` + `model.aborted` bool, not `os.Exit` | Lets bubbletea restore the terminal before `cmd/alturd` applies exit status 1 via a silent `errAborted` sentinel (CR-02) | ✓ Confirmed — Phase 4 |
| `difftool.trustExitCode` written `false`, not `true` | Git's external-diff protocol cannot express "user cancelled" — any trusted non-zero exit is treated as an unconditional fatal crash (G-04-1). Trade-off accepted: multi-file abort no longer stops file N+1 from opening | ✓ Confirmed — Phase 4 (04-07), regression test with sensitivity control |
| `difftoolDiff`/`diffArgs` always carry `--no-ext-diff` | Without it, alturd's internal `git diff --no-index` inherits `GIT_EXTERNAL_DIFF` from git's own difftool dispatch and recurses into itself until the process table is exhausted (G-04-2) | ✓ Confirmed — Phase 4 (04-06), regression tests pin it |
| `DifftoolInfo.NewFileLines` renamed to `RefFileLines` | WR-07: the field no longer always holds post-image content once deleted files use the local (pre-image) side | ✓ Confirmed — Phase 04.1 |
| Live human session (not scripted pty) for the blocking Phase 3 UAT checkpoint | D-02: some checks (colour rendering) cannot be automated; scripted pty sessions can't confirm actual terminal color output | ✓ Confirmed — Phase 04.1, 11/11 items passed |

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

## Next Milestone Goals

Candidates for the next milestone (from REQUIREMENTS.md v2 section — none committed yet, to be finalized via `/gsd-new-milestone`):

- **V2-01**: Three-way merge / conflict resolution mode
- **V2-02**: Staged-only diff view (`git diff --cached`)
- **V2-03**: Multi-terminal HUMAN-UAT breadth (alacritty, kitty, GNOME Terminal, iTerm2, tmux passthrough)
- **V2-04**: Homebrew tap or other package manager distribution beyond GitHub Releases

Plus housekeeping carried from v1.0's tech debt: reconcile the two stale debug session statuses, and re-run `go test -race ./...` in an environment with a C toolchain (CI already has one).

---
*Last updated: 2026-08-17 after v1.0 milestone shipped — full requirements audit against REQUIREMENTS.md's canonical 30 IDs (corrected a stale `CLI-01` placeholder that never matched the real `GIT-01..05` IDs), all 30 v1 requirements moved to Validated, Key Decisions reconciled with outcomes, Current State and Next Milestone Goals sections added.*
