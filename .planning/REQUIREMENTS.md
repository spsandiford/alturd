# Requirements: Alturd (Go Port)

**Defined:** 2026-06-26
**Core Value:** A developer can download a single binary, run `alturd` in any git repository, and navigate every changed file and every individual diff hunk with fast keyboard-driven controls — no runtime dependencies required.

## v1 Requirements

### Git Layer

- [x] **GIT-01**: User can run `alturd` in a git repo with no args to diff working tree vs HEAD
- [x] **GIT-02**: User can run `alturd <ref>`, `alturd <ref1>..<ref2>`, `alturd <ref1>...<ref2>`, `alturd <ref1> <ref2>` to diff specific ranges
- [x] **GIT-03**: User can run `alturd -- <paths>` to filter diff to specific paths
- [x] **GIT-04**: `alturd --version` and `alturd --help` exit cleanly without creating log files or side effects
- [x] **GIT-05**: User sees a clear single-line error message when not in a git repo (exit 1) or git not on PATH (exit 127)

### Diff Model

- [x] **DIFF-01**: User sees old and new file content rendered in aligned parallel side-by-side columns
- [x] **DIFF-02**: User sees syntax highlighting via Chroma (200+ languages, same lexer selection behavior as Pygments)
- [x] **DIFF-03**: User sees line-level diff colors (added/removed/modified) layered with syntax highlighting
- [x] **DIFF-04**: User sees intra-line word/character-level change markers on modified lines (with 1000-char/200-token/100ms guards)
- [x] **DIFF-05**: User sees full-file mode by default — entire file rendered with changes highlighted in place, unchanged lines shown in full
- [x] **DIFF-06**: User can toggle between full-file and hunk-only view with `v` hotkey without reload
- [x] **DIFF-07**: Binary files, pure renames, mode-only changes, submodule bumps, and no-newline-at-EOF all render correct placeholder or diff content

### Navigation

- [x] **NAV-01**: User can press `n`/`N` to jump between hunks; in full-file mode each change is centered with surrounding context visible
- [x] **NAV-02**: User can press `]`/`[` to cycle between changed files in the file tree
- [x] **NAV-03**: User can press `Tab` to switch focus between file tree pane and diff pane
- [x] **NAV-04**: User can press `q` to exit (exit 0); `Q` aborts in difftool mode (exit 1)

### File Tree

- [x] **TREE-01**: User sees a file tree of all changed files with [A]/[M]/[D]/[R] colored status markers, dirs-first sort, and compact-folder layout
- [x] **TREE-02**: Tree pane widens to 45 columns when focused and contracts to 24 columns (truncated filenames) when not focused, with animated transition
- [x] **TREE-03**: User can press `a` to toggle between changed-files-only view and full-repo tree view

### Search

- [x] **SEARCH-01**: User can press `/` to open in-pane search with match highlighting; `n`/`N` navigates between matches

### Theming & Config

- [ ] **THEME-01**: User sees light/dark/auto theming — terminal background detected via OSC 11 with 50ms timeout fallback to dark
- [ ] **CONFIG-01**: User can place a TOML config file at `$XDG_CONFIG_HOME/alturd/config.toml` (or pass `--config <path>`) to override defaults
- [ ] **CONFIG-02**: User can override any default keybinding via the config file

### Difftool Integration

- [ ] **DIFFTOOL-01**: User can invoke `alturd --difftool-local <path> --difftool-remote <path> --difftool-path <path>` to open a single-file side-by-side view without the file tree (for use by `git difftool`)
- [x] **DIFFTOOL-02**: TitleBar shows "alturd (difftool) — N of M — `<filename>`" when `GIT_DIFF_PATH_COUNTER`/`GIT_DIFF_PATH_TOTAL` env vars are set
- [ ] **DIFFTOOL-03**: User can run `alturd install-difftool [--scope global|local] [--name NAME] [--force]` to write the four canonical gitconfig keys idempotently

### Distribution

- [ ] **DIST-01**: Every push to GitHub runs `go test ./...` on Linux, macOS, and Windows via GitHub Actions CI
- [ ] **DIST-02**: Every git tag push triggers GitHub Actions to build and publish pre-built binaries for Linux (amd64/arm64), macOS (amd64/arm64), and Windows (amd64) as GitHub Release assets
- [ ] **DIST-03**: All released binaries are fully self-contained (`CGO_ENABLED=0`) with no runtime dependencies

### Logging

- [x] **LOG-01**: Log file written under `$XDG_STATE_HOME/alturd/alturd.log` (never to stderr); truncated at 1MB cap on startup

## v2 Requirements

### Future Enhancements

- **V2-01**: Three-way merge / conflict resolution mode (explicitly deferred; was v2 candidate in Python version)
- **V2-02**: Staged-only diff view (`git diff --cached`)
- **V2-03**: Multi-terminal HUMAN-UAT breadth (alacritty, kitty, GNOME Terminal, iTerm2, tmux passthrough)
- **V2-04**: Homebrew tap or other package manager distribution beyond GitHub Releases

## Out of Scope

| Feature | Reason |
|---------|--------|
| Python runtime dependency | The entire point of the Go port is a dependency-free binary |
| PyPI/pip distribution | Go port distributes via GitHub Releases binaries only |
| Three-way merge / conflict resolution | Explicitly deferred in Python version; remains out of scope |
| Editing files in-place from diff view | alturd is a reviewer, not an editor |
| Large-file streaming / lazy loading | Not a concern for this port; revisit only if a real user hits a wall |
| Mouse navigation in diff pane | Keyboard-first; mouse support is a v2+ consideration |
| In-app git operations (staging, commit, branch) | alturd is a diff viewer only; gitui/lazygit cover full git TUI |
| libgit2 / go-git bindings | Git CLI/plumbing only, per PROJECT.md constraint |
| CGO-dependent builds | All binaries must be CGO-disabled for portability |
| Windows arm64 binary | No goreleaser cross-compilation support for this target currently |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| DIFF-01 | Phase 1 | Complete |
| DIFF-02 | Phase 1 | Complete |
| DIFF-03 | Phase 1 | Complete |
| DIFF-04 | Phase 1 | Complete |
| DIFF-05 | Phase 1 | Complete |
| DIFF-07 | Phase 1 | Complete |
| GIT-01 | Phase 2 | Complete |
| GIT-02 | Phase 2 | Complete |
| GIT-03 | Phase 2 | Complete |
| GIT-04 | Phase 2 | Complete |
| GIT-05 | Phase 2 | Complete |
| LOG-01 | Phase 2 | Complete |
| DIFF-06 | Phase 3 | Complete |
| NAV-01 | Phase 3 | Complete |
| NAV-02 | Phase 3 | Complete |
| NAV-03 | Phase 3 | Complete |
| NAV-04 | Phase 3 | Complete |
| TREE-01 | Phase 3 | Complete |
| TREE-02 | Phase 3 | Complete |
| TREE-03 | Phase 3 | Complete |
| SEARCH-01 | Phase 3 | Complete |
| THEME-01 | Phase 4 | Gaps Found |
| CONFIG-01 | Phase 4 | Gaps Found |
| CONFIG-02 | Phase 4 | Gaps Found |
| DIFFTOOL-01 | Phase 4 | Gaps Found |
| DIFFTOOL-02 | Phase 4 | Gaps Found |
| DIFFTOOL-03 | Phase 4 | Gaps Found |
| DIST-01 | Phase 4 | Gaps Found |
| DIST-02 | Phase 4 | Gaps Found |
| DIST-03 | Phase 4 | Gaps Found |

**Coverage:**

- v1 requirements: 30 total
- Mapped to phases: 30 (Phase 1: 6, Phase 2: 6, Phase 3: 9, Phase 4: 9)
- Unmapped: 0 ✓

---
*Requirements defined: 2026-06-26*
*Last updated: 2026-06-26 after roadmap creation — traceability populated*
