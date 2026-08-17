# Phase 4: Config + Theming + Difftool + Distribution - Context

**Gathered:** 2026-07-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Make alturd fully configurable via TOML (keybinding overrides), correctly themed across light/dark terminals (with a manual override escape hatch), integrable as a `git difftool` backend via a self-installing subcommand, and shipped as self-contained pre-built binaries via automated GitHub Actions CI + goreleaser.

Requirements in scope: THEME-01, CONFIG-01, CONFIG-02, DIFFTOOL-01, DIFFTOOL-02, DIFFTOOL-03, DIST-01, DIST-02, DIST-03.

**Carrying forward from Phase 3:** Light/dark background auto-detection is already partially wired — `main.go` calls `termenv.NewOutput(os.Stdout).HasDarkBackground()` synchronously before `tea.NewProgram()`, and `diff.SetDarkBackground(bool)` / `statusMarkerStyle(status, darkBg)` already consume the result. Phase 4 adds the config/CLI override layer and difftool-mode-specific handling on top of this, not a rebuild.

Out of scope for Phase 4: everything already validated in Phases 1-3 (diff rendering, git layer, TUI navigation) — not re-litigated here.

</domain>

<decisions>
## Implementation Decisions

### Config File Design (CONFIG-01, CONFIG-02)

- **D-01:** Keybinding overrides merge with defaults — a config file only needs to specify the actions it changes; unspecified actions keep their default key. Do not require a complete action set.
- **D-02:** Config validation is strict in both directions: unknown TOML field names are rejected at startup (via `go-toml/v2` `DisallowUnknownFields`, per CONFIG-01), AND unrecognized key-string values or duplicate/conflicting key bindings across actions are also rejected at startup with a clear single-line error. No silent "last one wins" behavior.
- **D-03:** If no config file exists at `$XDG_CONFIG_HOME/alturd/config.toml` and `--config` wasn't passed, alturd silently runs with all defaults. Nothing is written to disk on first run — no auto-generated template config.
- **D-04 (Claude's discretion):** Exact TOML schema shape for keybindings (flat `[keybindings]` table vs. grouped-by-pane sub-tables) — user deferred to whatever is idiomatic for `go-toml/v2` and easiest to validate with `DisallowUnknownFields`. Recommendation leaning: flat `[keybindings]` table (action = key string), since most of the 10 rebindable actions (`n`/`N`/`]`/`[`/`Tab`/`v`/`/`/`a`/`q`/`Q`) are global regardless of focused pane — see `03-UI-SPEC.md` Key Binding Contract for the full action list and current hardcoded defaults.

### Theme Behavior (THEME-01)

- **D-05:** Add a manual theme override on top of the existing auto-detect: a `theme = "light" | "dark" | "auto"` config key, plus a matching `--theme` CLI flag. `auto` (the default) preserves Phase 3's OSC-11-via-termenv + dark-fallback behavior; `light`/`dark` bypass detection entirely. — **Reversibility:** reversible — additive config surface, no behavior change when unset.
- **D-06:** In difftool mode specifically, skip the OSC 11 query entirely — never attempt the terminal round-trip when alturd is invoked as a `git difftool` subprocess. Use the config/flag `theme` override if set, otherwise fall back to dark. Rationale: `.planning/research/PITFALLS.md` Pitfall 10 documents OSC 11 hangs/garbage-output risk in subprocess/tmux/SSH contexts, which difftool invocation is especially prone to (git is the parent process, not a plain interactive shell).
- **D-07:** Precedence order (standard CLI-over-config convention, not explicitly asked but implied by D-05): `--theme` flag > config `theme` key > auto-detect > dark fallback. Planner should confirm this is the only sane order — no alternative was discussed.

### Difftool Setup (DIFFTOOL-01, DIFFTOOL-02, DIFFTOOL-03)

- **D-08 (Claude's discretion, research required):** The exact "four canonical gitconfig keys" DIFFTOOL-03 must write were NOT copied from the Python reference implementation (unavailable in this repo — see Phase 1 decision log). User accepted the standard git custom-difftool pattern as a starting point: `diff.tool`, `difftool.<name>.cmd`, `difftool.prompt = false` (skip git's per-file "Hit return to continue" prompt — alturd IS the interactive viewer), `difftool.trustExitCode = true` (so `Q` → exit 1 correctly signals git to abort the difftool loop, consistent with NAV-04's existing `Q` → exit 1 behavior from Phase 3). Researcher/planner should verify against git-scm.com difftool docs before locking this in — user did not want to block on it.
- **D-09:** Default `--scope` is `global` (writes to `~/.gitconfig`) and default `--name` is `alturd`. Most users want alturd available in every repo without re-running per-repo.
- **D-10:** Idempotency contract for re-running `install-difftool` without `--force`: safely no-op / re-write alturd's own 4 keys (this is what "idempotent" means per DIFFTOOL-03's wording). `--force` is only required when `diff.tool` is already set to something else (e.g. `vimdiff`) and the user wants alturd to take over that slot. Re-running install-difftool to simply refresh/confirm alturd's own config should never require `--force` or error.

### Release Pipeline Conventions (DIST-01, DIST-02, DIST-03)

- **D-11:** goreleaser release workflow triggers on semver tags matching `v*.*.*` (e.g. `v1.0.0`) — standard goreleaser convention, and what `-ldflags "-X main.version=<tag>"` (the `var version` stub already in place since Phase 2) expects to inject as a clean version string.
- **D-12:** CI (`go test ./...` on Linux/macOS/Windows, DIST-01) runs on both `push` and `pull_request` triggers, not push-only — catches issues in PRs before merge, which is the GitHub Actions norm even though DIST-01's literal wording only says "every push."
- **D-13:** CI also runs `golangci-lint` (staticcheck, govet, errcheck, revive — per `.claude/CLAUDE.md` §Development Tools) as a gate, even though DIST-01's acceptance criteria only literally requires `go test ./...`. User wants the CLAUDE.md-recommended lint tooling actually wired in, not deferred indefinitely.

### Claude's Discretion

- Exact TOML schema shape for keybindings (D-04) — flat vs. grouped table.
- Exact gitconfig keys for install-difftool (D-08) — verify against git-scm.com docs before locking.
- CLI flag naming/spelling details not explicitly discussed (e.g. `--theme` flag short form, if any).
- Whether `golangci-lint` config (`.golangci.yml`) needs a specific ruleset beyond the four tools named in CLAUDE.md — planner/researcher to decide reasonable defaults.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & Scope

- `.planning/REQUIREMENTS.md` §Theming & Config — THEME-01, CONFIG-01, CONFIG-02; exact override/rejection wording
- `.planning/REQUIREMENTS.md` §Difftool Integration — DIFFTOOL-01, DIFFTOOL-02, DIFFTOOL-03; CLI flag names, titlebar format, gitconfig subcommand contract
- `.planning/REQUIREMENTS.md` §Distribution — DIST-01, DIST-02, DIST-03; CI/goreleaser acceptance criteria
- `.planning/ROADMAP.md` §Phase 4 — Success criteria (5 items); exact titlebar string format `"alturd (difftool) — N of M — <filename>"`

### Library Choices & Architecture

- `.claude/CLAUDE.md` §Technology Stack — `pelletier/go-toml/v2` (config parsing, `DisallowUnknownFields`), `muesli/termenv` (background detection — already an indirect dep via bubbletea, main.go already imports it directly), `adrg/xdg` (already a direct dependency since Phase 2/3), goreleaser v2.16, golangci-lint tool list
- `.claude/CLAUDE.md` §Stack Patterns — `termenv.NewOutput(os.Stdout).HasDarkBackground()` pattern, `AdaptiveColor` usage, `CGO_ENABLED=0` + `-trimpath` goreleaser ldflags pattern
- `.claude/CLAUDE.md` §What NOT to Use — CGO must stay disabled; no libgit2/go-git

### Research & Pitfalls

- `.planning/research/PITFALLS.md` Pitfall 9 — CRLF normalization (already handled in `internal/git`, verify difftool mode's file-reading path doesn't reintroduce this)
- `.planning/research/PITFALLS.md` Pitfall 10 — OSC 11 hang/garbage-output risk; directly informs D-06 (skip OSC 11 in difftool mode)
- `.planning/research/PITFALLS.md` Pitfall 11 — signal handler accumulation across multiple `tea.Program` instances; likely a non-issue since git spawns a fresh alturd process per difftool file, but planner should verify no in-process loop is introduced
- `.planning/research/ARCHITECTURE.md` — difftool mode sketch (`internal/diff.ParseFile(orig, modified)` idea); written before Phases 1-3 existed, so treat as inspiration only — actual package layout (`internal/tui`, not `internal/ui`; no `internal/config` yet) has diverged
- `.planning/research/FEATURES.md` — install-difftool description, GIT_DIFF_PATH_COUNTER/TOTAL protocol summary

### Phase 1-3 Integration Points

- `.planning/phases/03-tui-application/03-UI-SPEC.md` §Key Binding Contract (around line 254-268) — full list of the 10 rebindable actions and their current hardcoded key/behavior, needed for CONFIG-02
- `.planning/phases/03-tui-application/03-CONTEXT.md` D-05 — status bar format `alturd — <filename> (<N> of <M> changed files)`; Phase 4 adds the difftool mode indicator per DIFFTOOL-02's exact format
- `.planning/phases/03-tui-application/03-CONTEXT.md` D-18 — `q`/`Q` exit codes (0/1) already wired in Phase 3's `tui/model.go`, informs D-08's `difftool.trustExitCode` reasoning
- `.planning/phases/02-git-layer-cli/02-CONTEXT.md` D-03 — `rootCmd.Version` set via `var version` (not `const`), ready for goreleaser `-ldflags` injection (D-11)
- `.planning/phases/02-git-layer-cli/02-CONTEXT.md` D-08 — module `github.com/alturd/alturd`, binary built from `cmd/alturd/` — goreleaser build config target

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- `cmd/alturd/main.go` — already imports `github.com/muesli/termenv` and calls `termenv.NewOutput(os.Stdout).HasDarkBackground()` synchronously before `tea.NewProgram()`; already calls `diff.SetDarkBackground(darkBg)`. Phase 4 wraps this with the config/flag override (D-05) and the difftool-mode skip (D-06) — does not replace it.
- `internal/diff/highlight.go` `SetDarkBackground(bool)` — switches the Chroma style between dark/light variants; already tested (`highlight_test.go`).
- `internal/tui/model.go` `statusMarkerStyle(status string, darkBg bool) lipgloss.Style` — already theme-aware for tree status markers ([A]/[M]/[D]/[R] colors); reuse for any new difftool-mode chrome.
- `internal/tui/model.go` `NewModel(files []*gitdiff.File, darkBg bool) model` — constructor signature Phase 4 will likely need to extend (e.g. adding difftool-mode flag, N-of-M counter, keybinding map) — check current signature before adding parameters vs. an options struct.
- `rootCmd` in `cmd/alturd/main.go` — cobra root command with `SilenceErrors`/`SilenceUsage` already set (D-02 from Phase 2); `install-difftool` and `--config`/`--theme` flags attach to this same command per Phase 2/3 decisions.
- `var version = "dev"` in `main.go` — already a `var` (not `const`) specifically so goreleaser `-ldflags "-X main.version=<tag>"` can override it (D-03 from Phase 2). Ready to use as-is for D-11.

### Established Patterns

- `internal/git.Runner` interface + `ExecRunner{}` DI pattern (stateless, injected, not a singleton) — any new git subprocess calls Phase 4 needs (e.g. `git config` calls for install-difftool) should follow the same interface-for-testability pattern.
- Table-driven tests with `testdata/` fixtures — continue for config parsing tests (valid/invalid TOML fixtures) and keybinding validation tests.
- `CGO_ENABLED=0` project-wide constraint — `pelletier/go-toml/v2` is pure Go, compatible.
- go.mod currently lists `github.com/muesli/termenv` and `github.com/adrg/xdg` as indirect — Phase 4 should verify/promote these to direct requires now that `main.go` explicitly imports termenv and config code will explicitly import xdg.

### Integration Points

- No `.github/workflows/` directory exists yet — DIST-01/DIST-02 create it from scratch (`ci.yml`, `release.yml`).
- No `.goreleaser.yaml` exists yet — DIST-02/DIST-03 create it from scratch.
- No `internal/config` package exists yet — CONFIG-01/02 create it from scratch; needs to expose a keybinding lookup that `internal/tui/model.go`'s `Update()` key-dispatch switch (currently hardcoded string literals like `case "q":`, `case "tab":`) can consume instead of literal key strings.
- `install-difftool` and `--difftool-local`/`--difftool-remote`/`--difftool-path` flags are new cobra surface on the existing `rootCmd` — no stubs exist yet (confirmed: Phase 2/3 deliberately left these out per their CONTEXT.md decisions).

</code_context>

<specifics>
## Specific Ideas

No specific "I want it like X" references — user accepted recommended options throughout, with two explicit "you decide" deferrals (TOML keybinding schema shape, exact gitconfig key names for install-difftool).

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 4-Config + Theming + Difftool + Distribution*
*Context gathered: 2026-07-27*
