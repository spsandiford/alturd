# Milestones

## v1.0 MVP (Shipped: 2026-08-17)

**Phases completed:** 5 phases (01-diff-model, 02-git-layer-cli, 03-tui-application, 04-config-theming-difftool-distribution, 04.1-address-tech-debt), 21 plans, 32 tasks

**Delivered:** A single self-contained `alturd` binary that renders `git diff` output as a navigable side-by-side TUI, with syntax highlighting, full keyboard navigation, TOML config, light/dark theming, `git difftool` integration, and automated cross-platform releases — no Python runtime required.

**Key accomplishments:**

- `internal/diff`: Go-native diff parsing/rendering engine (go-gitdiff + chroma + go-diff) validated against a 13-scenario fixture corpus, with side-by-side alignment, syntax highlighting, and intra-line change markers.
- `internal/git` + `cmd/alturd`: git subprocess chokepoint and cobra CLI supporting all six ref-grammar invocation forms, with typed exit codes and CRLF normalization.
- `internal/tui`: full bubbletea v2 interactive application — file tree, split-screen diff pane, hunk/file navigation, in-pane search, all keyboard-driven.
- `internal/config`: TOML config with strict validation and fully overridable keybindings; OSC 11-based light/dark theme auto-detection.
- `git difftool` integration (`install-difftool` + single-file mode), plus two production-grade bugs found and fixed during Phase 4 hardening: an unbounded recursive process spawn (G-04-2, `--no-ext-diff` fix) and a fatal git exit on abort (G-04-1, `trustExitCode=false` fix), both with end-to-end regression coverage.
- Three-OS CI matrix + goreleaser pipeline publishing 5 `CGO_ENABLED=0` binaries (Linux/macOS/Windows) on tag push.
- Phase 04.1 closed all 6 tech-debt items from the prior audit, including the previously-unrun blocking human UAT checkpoint for Phase 3's core interactive flow.

**Known non-blocking debt at ship time** (see `.planning/STATE.md` Deferred Items and `.planning/WINDOWS.md`): `go test -race ./...` has not been run in an environment with a C toolchain; two debug session files record root causes that were fixed in code but never had their status field flipped to `fixed`. Both are bookkeeping-only, verified via the milestone audit to have zero functional impact.

---
