# Roadmap: Alturd (Go Port)

## Overview

Port the Python/Textual alturd diff viewer to a self-contained Go binary. Four horizontal layers
build bottom-up: the diff model library is validated in isolation first (highest correctness risk),
then the git subprocess and CLI layer, then the full bubbletea TUI that consumes both, and finally
the configuration, theming, difftool integration, and release infrastructure that ship the product.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Diff Model** - Pure Go diff parsing and rendering engine validated against the Python fixture corpus
- [ ] **Phase 2: Git Layer + CLI** - Git subprocess chokepoint, ref-grammar argument parsing, error handling, and logging
- [ ] **Phase 3: TUI Application** - Full bubbletea v2 interactive app with file tree, diff pane, and all navigation
- [ ] **Phase 4: Config + Theming + Difftool + Distribution** - TOML config, OSC 11 theming, difftool mode, CI, goreleaser release

## Phase Details

### Phase 1: Diff Model

**Goal**: The internal/diff library correctly parses all valid git diff formats and produces aligned side-by-side output with syntax highlighting and intra-line markers, verified by table tests against the Python fixture corpus — before any TUI code exists.
**Depends on**: Nothing (first phase)
**Requirements**: DIFF-01, DIFF-02, DIFF-03, DIFF-04, DIFF-05, DIFF-07
**Success Criteria** (what must be TRUE):

  1. `go test ./internal/diff/...` passes all table tests against the 12+ Python fixture scenarios (binary patches, renames, mode-only changes, submodule bumps, no-newline-at-EOF)
  2. A two-file diff produces side-by-side columns with Chroma syntax highlighting applied where a language can be detected; ANSI resets at every left-column boundary prevent color bleed
  3. Modified lines carry intra-line character-level change markers produced by the LCS pass, subject to the 1000-char/200-token/100ms guards
  4. Full-file mode and hunk-only mode each produce correct, independently testable output for the same input — full-file includes all unchanged lines; hunk-only includes only changed hunks

**Plans**: 2/3 plans executed
**Wave 1**

- [x] 01-01-PLAN.md — Module foundation: go.mod + 3 locked diff libs, 13-fixture corpus, core type model (RowPair/RenderMode)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 01-02-PLAN.md — Parse (no-panic go-gitdiff wrapper) + Align (RowPairs, full-file/hunk modes, edge-case placeholders)

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 01-03-PLAN.md — Highlight (Chroma) + Render (side-by-side []string, layered diff colors, guarded intra-line markers)

### Phase 2: Git Layer + CLI

**Goal**: The `alturd` binary can be invoked with any valid command-line form, executes the correct git plumbing commands, and returns clean errors for invalid invocations — all without a running TUI.
**Depends on**: Phase 1
**Requirements**: GIT-01, GIT-02, GIT-03, GIT-04, GIT-05, LOG-01
**Success Criteria** (what must be TRUE):

  1. `alturd --version` and `alturd --help` exit 0 and produce expected output; no log file or side effect is created
  2. Running `alturd` outside a git repo exits 1 with a single-line error; running it when git is not on PATH exits 127 with a single-line error
  3. All six invocation forms — no args, `<ref>`, `<ref1>..<ref2>`, `<ref1>...<ref2>`, `<ref1> <ref2>`, and `-- <paths>` filtering — each produce the correct `git diff` subprocess command (verified via command-capture tests)
  4. Git subprocess output is CRLF-normalized to LF immediately after `cmd.Output()` on all platforms before it reaches the diff parser
  5. The log file is written to `$XDG_STATE_HOME/alturd/alturd.log` (never to stderr) and is truncated to 1 MB on startup if it exceeds that size

**Plans**: TBD

### Phase 3: TUI Application

**Goal**: A developer can run `alturd` in a git repository and navigate every changed file and every hunk interactively using keyboard controls in a split-screen bubbletea v2 terminal UI.
**Depends on**: Phase 2
**Requirements**: DIFF-06, NAV-01, NAV-02, NAV-03, NAV-04, TREE-01, TREE-02, TREE-03, SEARCH-01
**Success Criteria** (what must be TRUE):

  1. `alturd` in a git repo displays a split-screen with the file tree pane (24 cols, truncated filenames) on the left and the diff pane on the right; the UI does not crash or show blank output before the first WindowSizeMsg arrives
  2. The file tree lists all changed files with colored [A]/[M]/[D]/[R] markers, dirs-first sort, and compact-folder layout; pressing `a` toggles between changed-files-only and full-repo tree view
  3. `Tab` switches focus between the two panes — the tree widens to 45 cols when focused and contracts to 24 when unfocused; `]`/`[` cycles between changed files
  4. `n`/`N` jumps between hunks; in full-file mode each hunk is centered with surrounding context; `v` toggles full-file/hunk-only view without reloading git data; `q` exits with code 0
  5. `/` opens in-pane search, typed text highlights matching substrings in the diff pane, and `n`/`N` navigates between matches using an ANSI-aware scanner

**Plans**: TBD
**UI hint**: yes

### Phase 4: Config + Theming + Difftool + Distribution

**Goal**: The application is fully configurable via TOML, correctly themed across light and dark terminals, integrable as `git difftool`, and shipped as self-contained pre-built binaries via automated GitHub Actions.
**Depends on**: Phase 3
**Requirements**: THEME-01, CONFIG-01, CONFIG-02, DIFFTOOL-01, DIFFTOOL-02, DIFFTOOL-03, DIST-01, DIST-02, DIST-03
**Success Criteria** (what must be TRUE):

  1. A TOML config file at `$XDG_CONFIG_HOME/alturd/config.toml` (or `--config <path>`) overrides default keybindings; unknown keys are rejected with a clear error message at startup
  2. Terminal background is auto-detected via OSC 11 with a 50ms timeout; when detection fails or times out the app falls back to dark theme without hanging
  3. `alturd install-difftool [--scope global|local] [--name NAME] [--force]` writes the four canonical gitconfig keys idempotently; `git difftool -t alturd <file>` launches a single-file side-by-side view without the file tree, and the title bar shows "alturd (difftool) — N of M — `<filename>`" when `GIT_DIFF_PATH_COUNTER`/`GIT_DIFF_PATH_TOTAL` env vars are present
  4. Every push to GitHub runs `go test ./...` on Linux, macOS, and Windows via GitHub Actions CI
  5. A git tag push triggers goreleaser to publish `CGO_ENABLED=0` binaries for Linux (amd64/arm64), macOS (amd64/arm64), and Windows (amd64) as GitHub Release assets

**Plans**: TBD
**UI hint**: yes

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Diff Model | 2/3 | In Progress|  |
| 2. Git Layer + CLI | 0/TBD | Not started | - |
| 3. TUI Application | 0/TBD | Not started | - |
| 4. Config + Theming + Difftool + Distribution | 0/TBD | Not started | - |
