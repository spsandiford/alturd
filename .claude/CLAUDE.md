<!-- GSD:project-start source:PROJECT.md -->

## Project

**Alturd (Go Port)**

Alturd is a terminal UI application for viewing `git diff` output as a navigable, side-by-side experience. It is a full port of the existing Python/Textual implementation to Go, producing a single self-contained binary that runs on Linux, macOS, and Windows. It is built for developers who live in the terminal and want a Meld/kdiff3-style diff review experience without leaving it or installing Python.

**Core Value:** A developer can download a single binary, run `alturd` in any git repository, and navigate every changed file and every individual diff hunk with fast keyboard-driven controls — no runtime dependencies required.

### Constraints

- **Target language**: Go — modern version (1.22+), chosen for single-binary cross-platform distribution
- **TUI framework**: Bubbletea + Lipgloss — the standard Go TUI stack, analogous to Textual
- **Cross-platform**: Must compile and run correctly on Linux (amd64/arm64), macOS (amd64/arm64), Windows (amd64)
- **Distribution**: GitHub Releases via GitHub Actions — no external registry or packaging system required
- **Behavior parity**: CLI grammar, exit codes, keybindings, config file location/format, and difftool integration must match the Python implementation so existing users can switch transparently
- **No runtime deps**: The final binary must be fully self-contained (CGO disabled where possible for portability)
- **Git compatibility**: Talks to git via its CLI/plumbing commands only, not via libgit2

<!-- GSD:project-end -->

<!-- GSD:stack-start source:research/STACK.md -->

## Technology Stack

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.22+ | Language runtime | Required for bubbletea v2; `range over integer` and other 1.22 features; specified in PROJECT.md |
| charm.land/bubbletea/v2 | v2.0.7 | TUI framework (Elm MVU) | The idiomatic Go TUI framework; Cursed Renderer is 10x faster than v1; battle-tested via Charm's Crush AI agent in production before Feb 2026 release; declarative View API; built-in color profile downsampling |
| charm.land/lipgloss/v2 | v2.x | Terminal styling and layout | Ships with bubbletea v2 as companion; `JoinHorizontal()`/`JoinVertical()` compose side-by-side panes without manual ANSI arithmetic; AdaptiveColor handles light/dark automatically |
| charm.land/bubbles/v2 | v2.x | Reusable TUI components | Provides Viewport (scrollable pane, high-performance mode for alternate screen) and TextInput (in-pane search); production-used in Crush |
| github.com/alecthomas/chroma/v2 | v2.27.0 | Syntax highlighting | Pure-Go port of Pygments by the same author (Alec Thomas); identical language database (200+ languages); terminal formatters for 8-color, 256-color, true-color ANSI; used by Hugo in production |
| github.com/bluekeyes/go-gitdiff | v0.8.1 | Unified git diff parsing | Handles the full git diff output including binary patches, renames/copies, extended headers, and no-newline markers — all edge cases present in the Python fixture corpus; produces typed FileDiff/TextFragment/BinaryFragment structs |
| github.com/sergi/go-diff | v1.4.0 | Intra-line character diff | Go port of diff-match-patch; `DiffMain(old, new, checklines)` returns `[]Diff{Type, Text}` for DiffDelete/DiffInsert/DiffEqual; applied pair-wise on changed lines to highlight exact changed characters |
| github.com/pelletier/go-toml/v2 | v2.4.2 | TOML configuration parsing | 5.8x faster than BurntSushi/toml; TOML 1.1.0 support; `DisallowUnknownFields()` for strict config validation; stdlib-like `Marshal`/`Unmarshal` API |
| github.com/muesli/termenv | v0.16.0 | Terminal background detection | `output.HasDarkBackground()` for auto light/dark theme selection; color profile detection and degradation (TrueColor → 256 → 16 → ASCII); Windows-compatible with `EnableVirtualTerminalProcessing` |
| goreleaser | v2.16 | Cross-platform binary distribution | Generates Linux/macOS/Windows binaries in one pass; automatic GitHub Release asset upload; checksums; CGO_ENABLED=0 for fully static binaries |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/adrg/xdg | v0.5.x | XDG base directory resolution | Resolving config file path on Linux (`$XDG_CONFIG_HOME/alturd/config.toml`); also provides Windows/macOS equivalents |
| github.com/muesli/reflow | v0.3.x | ANSI-aware text wrapping | Word-wrapping content inside Viewport without corrupting escape sequences; already depended on by bubbles |
| github.com/charmbracelet/log | v0.4.x | Structured logging | Debug-mode logging to a file (not stderr, which would corrupt TUI); optional dependency |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| goreleaser/goreleaser-action@v7 | GitHub Actions release automation | Pin `version: '~> v2'`; requires `fetch-depth: 0` for changelog; triggers on git tags |
| actions/setup-go@v5 | Go toolchain in CI | `go-version: stable` tracks current stable; use `cache: true` |
| golangci-lint | Lint in CI | Include `staticcheck`, `govet`, `errcheck`, `revive` |
| go test -race | Race detector | Enable in CI; bubbletea v2 is goroutine-safe but custom async code needs validation |

## Installation

# Core framework (v2 vanity import paths)

# Syntax highlighting

# Diff parsing

# Config and environment

# goreleaser (dev tool, not a go dependency)

# .github/workflows/release.yml (key snippet)

- uses: goreleaser/goreleaser-action@v7

# .goreleaser.yaml (essential structure)

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| charm.land/bubbletea/v2 | github.com/rivo/tview | Only if you need a rich pre-built widget toolkit (forms, tables, modals) with zero MVU learning curve; tview is synchronous/imperative, which fits simple CRUD-style UIs but fights you on a reactive diff viewer |
| charm.land/bubbletea/v2 | github.com/gdamore/tcell/v2 | Only if building a bespoke renderer from scratch — tcell is the low-level terminal library that tview sits on; excessive complexity for this use case |
| github.com/bluekeyes/go-gitdiff | github.com/sourcegraph/go-diff | sourcegraph/go-diff (v0.8.0) works for simple cases but does not handle binary patches, rename extended headers, or the `\ No newline at end of file` sentinel as a typed field — all of which appear in the Python fixture corpus |
| github.com/alecthomas/chroma/v2 | github.com/alecthomas/chroma/v3 | Use v3 only after it reaches a stable (non-alpha) release; v3.0.0-alpha.1 replaces `Iterator` with `iter.Seq[Token]` (Go 1.23 required) — API is still changing |
| github.com/pelletier/go-toml/v2 | github.com/BurntSushi/toml | BurntSushi/toml is fine but 5.8x slower and does not have `DisallowUnknownFields` for typo detection; either works, but go-toml v2 is strictly better |
| goreleaser v2 | Custom Makefile + gh release | Use a Makefile only if your release matrix is a single platform; goreleaser handles checksums, GitHub Release API, and multi-arch archives that would otherwise require 100+ lines of shell |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| github.com/charmbracelet/bubbletea (v1) | v1 receives only bugfixes as of Feb 2026; new features (Cursed Renderer, declarative View, color auto-downsampling) are v2-only; import path is different so migration later is manual | charm.land/bubbletea/v2 |
| github.com/alecthomas/chroma/v3 | Alpha status (v3.0.0-alpha.1 as of Jun 2026); API is unstable — `Iterator` removed, requires Go 1.23; use when stable | github.com/alecthomas/chroma/v2 (v2.27.0) |
| github.com/nsf/termbox-go | Unmaintained; superseded by tcell and then by the Charm ecosystem; no color profile detection | charm.land/bubbletea/v2 + lipgloss |
| github.com/gizak/termui/v3 | Dashboard-only widget library (charts, gauges); not designed for keyboard-driven navigation or text-heavy pane UIs | charm.land/bubbletea/v2 |
| libgit2 / go-git | The PROJECT.md explicitly specifies talking to git via CLI/plumbing commands only; no git library at all | `exec.Command("git", "diff", ...)` |
| github.com/gdamore/tcell/v2 (directly) | Correct low-level library, but bubbletea v2 already wraps it; using tcell directly means rebuilding the event loop, renderer, and model that bubbletea provides | charm.land/bubbletea/v2 |
| CGO (enabled) | CGO_ENABLED=1 produces binaries that depend on glibc version on Linux and cannot easily cross-compile; PROJECT.md requires fully self-contained binaries | Set `CGO_ENABLED=0` in goreleaser build config |

## Stack Patterns by Variant

- Use two `bubbles/viewport` instances, one for old content and one for new, sized with `lipgloss.Width()` and composed with `lipgloss.JoinHorizontal()`
- bubbletea v2's Cursed Renderer handles synchronized updates to both viewports without flicker
- Store focused/unfocused widths in model state
- On focus change, send a `tea.WindowSizeMsg`-equivalent to the viewport to resize
- Use lipgloss `Style.Width()` to truncate filenames cleanly at 24 or 45 cols
- Use `bubbles/textinput` for the search input, rendered as an overlay at pane bottom
- Match positions computed on content string; highlight via lipgloss `Style.Background()`
- Dismiss with `Escape`; cursor position maintained
- Call `termenv.NewOutput(os.Stdout).HasDarkBackground()` at startup
- Pass result to a theme selector that returns two lipgloss `Style` presets (dark/light)
- Use `lipgloss.AdaptiveColor{Light: "...", Dark: "..."}` in shared styles for automatic adaptation
- Detect `GIT_DIFF_PATH_COUNTER` env var at startup; if present, skip tree pane
- Start bubbletea in `tea.WithAltScreen()` mode
- Render only the diff viewport at full terminal width
- Set `CGO_ENABLED=0` and `-trimpath` in goreleaser `ldflags`
- Verify with `ldd alturd` on Linux — should print `not a dynamic executable`
- On macOS, static linking is partial by default; use `GOOS=linux CGO_ENABLED=0 go build` for the Linux artifacts specifically

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| charm.land/bubbletea/v2 v2.0.7 | charm.land/lipgloss/v2, charm.land/bubbles/v2 | All three released together Feb 2026; use matching v2 imports for all |
| github.com/alecthomas/chroma/v2 v2.27.0 | Go 1.22+ | v2.16.0+ added `Iterator.Stdlib()` for Go 1.23 iter.Seq; use v2.27.0 for latest language support |
| github.com/bluekeyes/go-gitdiff v0.8.1 | Go 1.17+ | No special compatibility constraints |
| github.com/sergi/go-diff v1.4.0 | Go 1.13+ | No special compatibility constraints |
| github.com/pelletier/go-toml/v2 v2.4.2 | Go 1.18+ | Generics used internally |
| goreleaser v2.16 | goreleaser-action@v7, Node.js 24 | Node.js 20 actions deprecated June 2026; use @v7 |
| charm.land/bubbletea/v2 | Windows | No resize events (SIGWINCH absent); works on Windows Terminal; CMD requires `EnableVirtualTerminalProcessing` |

## Windows TUI Support (Explicit Assessment)

| Capability | Windows Terminal | Windows CMD | Mitigation |
|------------|-----------------|-------------|------------|
| True color rendering | Yes (full RGB) | Partial (256-color) | lipgloss v2 auto-downgrades via colorprofile |
| Mouse support | Yes | Yes (fixed in v1.3.4) | Use `tea.WithMouseAllMotion()` |
| Terminal resize events | No (SIGWINCH absent) | No | Poll `tea.WindowSizeMsg` on a tick, or accept initial size only |
| Color scheme detection | Via env vars | Via env vars | termenv fallback to `COLORFGBG` env var |
| Alternate screen | Yes | Yes | Use `tea.WithAltScreen()` as normal |

## Sources

- [pkg.go.dev: charm.land/bubbletea/v2](https://pkg.go.dev/charm.land/bubbletea/v2) — version v2.0.7 confirmed
- [GitHub Discussion: Bubble Tea v2 What's New](https://github.com/charmbracelet/bubbletea/discussions/1374) — feature breakdown
- [pkg.go.dev: github.com/charmbracelet/lipgloss](https://pkg.go.dev/github.com/charmbracelet/lipgloss) — v1.1.0 (Mar 2025), v2 is highest major
- [pkg.go.dev: github.com/charmbracelet/bubbles](https://pkg.go.dev/github.com/charmbracelet/bubbles) — viewport and textinput components
- [pkg.go.dev: github.com/alecthomas/chroma/v2](https://pkg.go.dev/github.com/alecthomas/chroma/v2) — v2.27.0 confirmed stable
- [GitHub: alecthomas/chroma releases](https://github.com/alecthomas/chroma/releases) — v3.0.0-alpha.1 confirmed alpha
- [pkg.go.dev: github.com/bluekeyes/go-gitdiff](https://pkg.go.dev/github.com/bluekeyes/go-gitdiff/gitdiff) — v0.8.1
- [pkg.go.dev: github.com/sergi/go-diff](https://pkg.go.dev/github.com/sergi/go-diff/diffmatchpatch) — v1.4.0
- [pkg.go.dev: github.com/pelletier/go-toml/v2](https://pkg.go.dev/github.com/pelletier/go-toml/v2) — v2.4.2
- [pkg.go.dev: github.com/muesli/termenv](https://pkg.go.dev/github.com/muesli/termenv) — v0.16.0
- [GoReleaser builds docs](https://goreleaser.com/customization/builds/go/) — v2.16 confirmed
- [goreleaser-action GitHub](https://github.com/goreleaser/goreleaser-action) — @v7 recommended for Node.js 24
- [bubbletea Windows terminal support discussion](https://github.com/charmbracelet/bubbletea/discussions/312) — platform caveats

<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->

## Conventions

Conventions not yet established. Will populate as patterns emerge during development.
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->

## Architecture

Architecture not yet mapped. Follow existing patterns found in the codebase.
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->

## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, `.github/skills/`, or `.codex/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->

## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:

- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->

## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
