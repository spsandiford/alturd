# Phase 4: Config + Theming + Difftool + Distribution - Research

**Researched:** 2026-07-27
**Domain:** Go TOML config parsing, terminal background detection (OSC 11), git custom-difftool integration, goreleaser cross-compilation, GitHub Actions CI
**Confidence:** HIGH

## Summary

Phase 4 has four independent sub-domains, each well-trodden but with one sharp landmine apiece. The two most important findings from this session, both verified by reading the **actual pinned dependency source code** (not just docs), change how the planner must sequence tasks:

1. **`termenv.OSCTimeout` is a hardcoded `const = 5*time.Second`** in `github.com/muesli/termenv@v0.16.0` (`termenv_unix.go`), not a configurable variable. The THEME-01 requirement ("OSC 11 with 50ms timeout") **cannot** be satisfied by configuring termenv — it must be satisfied by the caller racing `HasDarkBackground()` in a goroutine against an external `time.After(50*time.Millisecond)` and taking whichever returns first. The `cmd/alturd/main.go` code inherited from Phase 3 calls `termenv.NewOutput(os.Stdout).HasDarkBackground()` **synchronously with no external timeout at all** — it currently does not meet THEME-01 and must be wrapped in Phase 4, not merely reused.
2. **`xdg.ConfigFile()` creates the parent directory as a side effect**, even when only checking for existence. This directly conflicts with D-03 ("nothing is written to disk on first run"). Config-loading code MUST use `xdg.SearchConfigFile()` instead, which performs a read-only search and returns an error (no directory creation) when the file is absent.

On the difftool side, D-08's working assumption (`diff.tool`, `difftool.<name>.cmd`, `difftool.prompt = false`, `difftool.trustExitCode = true`) is **confirmed correct and complete** against both git-scm.com's official docs and git's own C source (`diff.c`, `builtin/difftool.c`, `git-difftool--helper.sh`). `GIT_DIFF_PATH_COUNTER`/`GIT_DIFF_PATH_TOTAL` are set on every `git difftool` invocation (confirmed by reading the exact code path), not only under the separate `GIT_EXTERNAL_DIFF` mechanism the docs describe them under — this was a real ambiguity in git's documentation structure that required source verification to resolve.

**Primary recommendation:** Build `internal/config` (TOML+keybindings+theme) and the difftool env-var/gitconfig plumbing as two independent, directly-testable packages with no `internal/tui` dependency; wire both into `cmd/alturd/main.go` behind explicit precedence logic (flag > config > auto-detect > dark). Ship distribution (`.goreleaser.yaml`, `.github/workflows/{ci,release}.yml`, `.golangci.yml`) as pure config/CI artifacts with no Go code dependencies, so they can be built and validated in parallel with the config/theme/difftool work.

## User Constraints (from CONTEXT.md)

<user_constraints>

### Locked Decisions

- **D-01:** Keybinding overrides merge with defaults — a config file only needs to specify the actions it changes; unspecified actions keep their default key. Do not require a complete action set.
- **D-02:** Config validation is strict in both directions: unknown TOML field names are rejected at startup (via `go-toml/v2` `DisallowUnknownFields`, per CONFIG-01), AND unrecognized key-string values or duplicate/conflicting key bindings across actions are also rejected at startup with a clear single-line error. No silent "last one wins" behavior.
- **D-03:** If no config file exists at `$XDG_CONFIG_HOME/alturd/config.toml` and `--config` wasn't passed, alturd silently runs with all defaults. Nothing is written to disk on first run — no auto-generated template config.
- **D-04 (Claude's discretion):** Exact TOML schema shape for keybindings (flat `[keybindings]` table vs. grouped-by-pane sub-tables) — user deferred to whatever is idiomatic for `go-toml/v2` and easiest to validate with `DisallowUnknownFields`. Recommendation leaning: flat `[keybindings]` table (action = key string), since most of the 10 rebindable actions (`n`/`N`/`]`/`[`/`Tab`/`v`/`/`/`a`/`q`/`Q`) are global regardless of focused pane.
- **D-05:** Add a manual theme override on top of the existing auto-detect: a `theme = "light" | "dark" | "auto"` config key, plus a matching `--theme` CLI flag. `auto` (the default) preserves Phase 3's OSC-11-via-termenv + dark-fallback behavior; `light`/`dark` bypass detection entirely. Reversibility: additive config surface, no behavior change when unset.
- **D-06:** In difftool mode specifically, skip the OSC 11 query entirely — never attempt the terminal round-trip when alturd is invoked as a `git difftool` subprocess. Use the config/flag `theme` override if set, otherwise fall back to dark.
- **D-07:** Precedence order: `--theme` flag > config `theme` key > auto-detect > dark fallback.
- **D-08 (Claude's discretion, research required):** The four canonical gitconfig keys DIFFTOOL-03 must write: `diff.tool`, `difftool.<name>.cmd`, `difftool.prompt = false`, `difftool.trustExitCode = true`. **Research verdict: confirmed correct and complete — see Difftool Setup section below.**
- **D-09:** Default `--scope` is `global` (writes to `~/.gitconfig`) and default `--name` is `alturd`.
- **D-10:** Idempotency contract for re-running `install-difftool` without `--force`: safely no-op / re-write alturd's own 4 keys. `--force` is only required when `diff.tool` is already set to something else and the user wants alturd to take over that slot.
- **D-11:** goreleaser release workflow triggers on semver tags matching `v*.*.*`.
- **D-12:** CI (`go test ./...` on Linux/macOS/Windows, DIST-01) runs on both `push` and `pull_request` triggers.
- **D-13:** CI also runs `golangci-lint` (staticcheck, govet, errcheck, revive per `.claude/CLAUDE.md` §Development Tools) as a gate.

### Claude's Discretion

- Exact TOML schema shape for keybindings (D-04) — flat vs. grouped table.
- Exact gitconfig keys for install-difftool (D-08) — verify against git-scm.com docs before locking.
- CLI flag naming/spelling details not explicitly discussed (e.g. `--theme` flag short form, if any).
- Whether `golangci-lint` config (`.golangci.yml`) needs a specific ruleset beyond the four tools named in CLAUDE.md — planner/researcher to decide reasonable defaults.

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| THEME-01 | Light/dark/auto theming; OSC 11 with 50ms timeout, dark fallback | `termenv.OSCTimeout` const-vs-var finding (Theme Detection Architecture section); goroutine-race pattern in Code Examples |
| CONFIG-01 | TOML config at `$XDG_CONFIG_HOME/alturd/config.toml` or `--config <path>`; unknown keys rejected at startup | go-toml/v2 `DisallowUnknownFields` API (Standard Stack, Code Examples); `xdg.SearchConfigFile` vs `xdg.ConfigFile` finding (Common Pitfalls) |
| CONFIG-02 | Override any default keybinding via config file | Flat `[keybindings]` TOML schema (Architecture Patterns); merge-with-defaults + duplicate-detection pattern (Code Examples) |
| DIFFTOOL-01 | `alturd --difftool-local/-remote/-path` single-file view, no tree | `$LOCAL`/`$REMOTE`/`$MERGED` variable mapping (Difftool Setup section) |
| DIFFTOOL-02 | Title bar "N of M" from `GIT_DIFF_PATH_COUNTER`/`GIT_DIFF_PATH_TOTAL` | Verified env var propagation through git's `run_external_diff()` → `git-difftool--helper` chain (Difftool Setup section) |
| DIFFTOOL-03 | `alturd install-difftool` writes 4 gitconfig keys idempotently | Confirmed 4-key set + `--tool-help` convention + idempotency read-before-write pattern (Difftool Setup section, Code Examples) |
| DIST-01 | CI runs `go test ./...` on Linux/macOS/Windows on push+PR | GitHub Actions matrix pattern (Architecture Patterns, Code Examples) |
| DIST-02 | Tag push triggers goreleaser binary publish | `.goreleaser.yaml` structure + `goreleaser-action@v7` workflow (Standard Stack, Code Examples) |
| DIST-03 | Binaries are `CGO_ENABLED=0`, self-contained | goreleaser `env: [CGO_ENABLED=0]` per-build setting (Code Examples); Pitfall 8 cross-reference |

</phase_requirements>

## Architectural Responsibility Map

Alturd is a terminal CLI/TUI application, not a web app — the standard Browser/API/CDN tier model doesn't apply. The relevant tiers for a single-binary Go CLI are: **CLI Entry** (`cmd/alturd`), **Config/Theme Layer** (new `internal/config`), **TUI Runtime** (`internal/tui`), **Git Subprocess Boundary** (`internal/git`), and **Build/Distribution** (CI/goreleaser, no runtime code).

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| TOML config parsing + validation | Config/Theme Layer (new `internal/config`) | CLI Entry (wires `--config` flag) | Parsing/validation is pure and testable in isolation; must not import `internal/tui` |
| Keybinding override + merge-with-defaults | Config/Theme Layer | TUI Runtime (consumes the resolved map) | `internal/tui/model.go`'s `handleKey` switch must be refactored to look up actions via a keybinding map instead of literal `case "q":` strings — this is a TUI-tier consumer of a config-tier artifact |
| Theme resolution (flag > config > auto-detect > dark) | CLI Entry (`cmd/alturd/main.go`) | Config/Theme Layer (theme value type + validation) | Precedence resolution must happen before `tea.NewProgram()` is called, same constraint Phase 3 already established for `darkBg` |
| OSC 11 background detection with 50ms external timeout | CLI Entry | — | Must remain synchronous-before-TUI-start (same reason as today), but wrapped in a goroutine+select timeout owned by `main.go`, not by termenv |
| Difftool single-file mode (no tree) | CLI Entry (mode dispatch on `--difftool-*` flags) | TUI Runtime (a reduced-chrome model variant) | `main.go` must branch before constructing the model; the model itself needs a "difftool mode" flag threaded through `NewModel` |
| `GIT_DIFF_PATH_COUNTER`/`TOTAL` → title bar | CLI Entry (reads env vars) | TUI Runtime (renders the string) | Env vars read once at startup in `main.go`, passed into the model as plain ints — no need for the model to know about git plumbing |
| `install-difftool` gitconfig writes | CLI Entry (new cobra subcommand) | Git Subprocess Boundary (`internal/git.Runner` for `git config` calls) | Follows the existing `git.Runner` DI pattern already used for `diff` and `ls-tree`; needs either a new runner method or careful reuse (see Common Pitfalls) |
| CI test matrix | Build/Distribution | — | Zero runtime code; pure YAML + Go standard toolchain |
| goreleaser cross-compilation | Build/Distribution | — | Zero runtime code; consumes the existing `var version` in `main.go` (already `var`, ready since Phase 2) |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/pelletier/go-toml/v2 | v2.4.3 (confirmed current via Go module proxy `@latest`; CLAUDE.md lists v2.4.2, one patch behind) | TOML config parsing with strict unknown-field rejection | `Decoder.DisallowUnknownFields()` is exactly the CONFIG-01 requirement; no alternative TOML library in the Go ecosystem has this as a first-class streaming-decoder feature |
| github.com/adrg/xdg | v0.5.3 (already in go.mod, currently indirect) | XDG path resolution for config file | Already a direct dependency of `internal/log`; reuse the same library rather than introducing a second XDG resolver. **Use `xdg.SearchConfigFile`, not `xdg.ConfigFile`, for the read path — see Common Pitfalls.** |
| github.com/muesli/termenv | v0.16.0 (already in go.mod, currently indirect; already imported directly by `main.go` since Phase 3) | OSC 11 background detection | Already wired in Phase 3; Phase 4 does not replace it, but must wrap `HasDarkBackground()` with an external timeout — see Theme Detection Architecture |
| goreleaser | v2.16+ (`version: '~> v2'` pin in workflow, not a Go module) | Cross-platform binary build + GitHub Release publish | Already named in CLAUDE.md; confirmed current `.goreleaser.yaml` schema requires top-level `version: 2` |
| golangci-lint | v2.4.0+ required for Go 1.25 support; v2.7.2 was latest stable as of Dec 2025 [CITED: github.com/golangci/golangci-lint issues #5873] | Lint gate (staticcheck, govet, errcheck, revive) | Go 1.25 compatibility requires v2.4.0 minimum — installing an older v1-line binary via a package manager will silently fail on this module's `go 1.25.0` directive |

**Version verification performed this session:**
```bash
go list -m -versions github.com/pelletier/go-toml/v2   # → ... v2.4.1 v2.4.2 v2.4.3
curl -s https://proxy.golang.org/github.com/pelletier/go-toml/v2/@latest
# → {"Version":"v2.4.3","Time":"2026-07-05T02:25:11Z", ...}
go version   # → go1.25.11 linux/amd64 (matches go.mod's `go 1.25.0` directive)
```

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| golangci-lint-action | v9 current, v7/v8 still supported (`v7` requires golangci-lint v2 only; `v8` requires golangci-lint >= v2.1.0) [CITED: github.com/golangci/golangci-lint-action README] | Wires golangci-lint into GitHub Actions | Pin an explicit `version:` for the golangci-lint binary (e.g. `v2.4.0` or later) regardless of action major version, since the action does not itself guarantee Go 1.25 support |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `pelletier/go-toml/v2` | `BurntSushi/toml` | No `DisallowUnknownFields`-equivalent strict mode as a first-class API; CLAUDE.md already rejects this |
| Custom OSC 11 query loop | Rely on termenv's default 5s timeout as-is | Fails THEME-01's literal 50ms requirement; would leave a 5-second hang risk in tmux/SSH environments exactly as PITFALLS.md Pitfall 10 warns |
| Flat `[keybindings]` table | Grouped `[keybindings.tree]` / `[keybindings.diff]` sub-tables | Grouped tables would require duplicating global actions (`q`, `Tab`, `]`, `[`) across sub-tables since the UI-SPEC Key Binding Contract shows most actions are pane-independent; flat table matches the data shape better and is simpler to validate |

**Installation:**
```bash
go get github.com/pelletier/go-toml/v2@v2.4.3
go mod tidy   # promotes github.com/adrg/xdg and github.com/muesli/termenv from indirect to direct
```

## Package Legitimacy Audit

> The `gsd-tools package-legitimacy check` seam only supports `npm`/`pypi`/`crates` ecosystems; it does not cover Go modules. This audit was performed manually against the Go module proxy (an authoritative registry) and GitHub repository metadata.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| github.com/pelletier/go-toml/v2 | Go module proxy (proxy.golang.org) | v2 line since 2021; latest patch v2.4.3 (Jul 2026) | Not applicable to Go modules (no npm-style download counter); widely imported — used by Hugo, Traefik, and 1000s of other modules per pkg.go.dev "Imported By" | github.com/pelletier/go-toml | OK | Approved — this is the only genuinely new dependency this phase introduces |
| github.com/adrg/xdg | Already in go.mod (Phase 2) | Pre-existing dependency | — | github.com/adrg/xdg | OK | No action — already vetted in a prior phase |
| github.com/muesli/termenv | Already in go.mod (Phase 3, indirect) | Pre-existing dependency | — | github.com/muesli/termenv | OK | No action — promote to direct require only |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
                     ┌─────────────────────────────┐
                     │   cmd/alturd/main.go (CLI)   │
                     └──────────────┬───────────────┘
                                    │
              ┌─────────────────────┼─────────────────────┐
              │                     │                     │
       standalone mode        difftool mode         install-difftool
   (no --difftool-* flags)  (--difftool-local/       (new subcommand,
              │              -remote/-path set)        no TUI at all)
              │                     │                     │
              ▼                     ▼                     ▼
   ┌──────────────────┐  ┌──────────────────────┐  ┌───────────────────┐
   │ 1. Load config     │  │ 1. Skip OSC 11 (D-06) │  │ 1. git config --get│
   │    (--config flag  │  │ 2. Read GIT_DIFF_PATH_│  │    diff.tool        │
   │    or XDG search,  │  │    COUNTER/TOTAL env  │  │ 2. If set & != name │
   │    D-03: no side   │  │ 3. Build synthetic     │  │    & !force → error │
   │    effects if      │  │    single-file diff    │  │ 3. git config --set │
   │    absent)          │  │    from -local/-remote │  │    ×4 keys (D-08)   │
   │ 2. Validate         │  │    /-path              │  └───────────────────┘
   │    (DisallowUnknown │  └──────────┬─────────────┘
   │    Fields, D-02)    │             │
   └─────────┬────────────┘             │
             │                          │
             ▼                          ▼
   ┌───────────────────────────────────────────┐
   │  2/3. Resolve theme (D-05/D-07 precedence): │
   │  --theme flag > config theme key >          │
   │  auto-detect (goroutine-raced OSC 11,       │
   │  50ms external timeout, standalone only) >  │
   │  dark fallback                              │
   └──────────────────┬───────────────────────────┘
                       │
                       ▼
   ┌───────────────────────────────────────────┐
   │  4. Resolve keybinding map: defaults        │
   │  overlaid with config [keybindings] table   │
   │  (D-01), duplicate/unknown-key validation   │
   │  (D-02) — fails startup with single-line    │
   │  error before any TUI code runs             │
   └──────────────────┬───────────────────────────┘
                       │
                       ▼
   ┌───────────────────────────────────────────┐
   │  5. tui.NewModel(files, darkBg, keymap,     │
   │     difftoolMode, counter, total)           │
   │  tea.NewProgram(m).Run()                    │
   └───────────────────────────────────────────┘
```

### Recommended Project Structure

```
internal/
├── config/                 # NEW — no dependency on internal/tui
│   ├── config.go            # Config struct, Load(path string) (*Config, error)
│   │                        # xdg.SearchConfigFile for the no-flag case (D-03)
│   ├── keybindings.go       # DefaultKeymap(), Merge(overrides), duplicate/unknown validation (D-01, D-02)
│   └── theme.go             # Theme type ("light"|"dark"|"auto"), Resolve(flag, config, detectFn) (D-05/D-06/D-07)
cmd/alturd/
├── main.go                  # adds --config, --theme flags; --difftool-local/-remote/-path flags;
│                             # install-difftool subcommand; mode dispatch
internal/tui/
├── model.go                 # NewModel signature extended: keymap, difftoolMode, pathCounter, pathTotal
.golangci.yml                # NEW
.goreleaser.yaml             # NEW
.github/workflows/
├── ci.yml                   # NEW — go test ./... matrix + golangci-lint
└── release.yml               # NEW — goreleaser on tag push
```

### Pattern 1: Strict TOML Decode with Merge-onto-Defaults

**What:** Decode user config into a struct with `DisallowUnknownFields`, then apply only the fields that were actually present onto a pre-built defaults struct — because go-toml/v2 has no built-in "merge partial config over defaults" primitive.
**When to use:** CONFIG-01/CONFIG-02 — config only needs to specify the keys it overrides (D-01).
**Example:**
```go
// Source: pkg.go.dev/github.com/pelletier/go-toml/v2 (verified this session)
type rawConfig struct {
    Theme       string            `toml:"theme"`
    Keybindings map[string]string `toml:"keybindings"`
}

func Load(path string) (*Config, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    var raw rawConfig
    dec := toml.NewDecoder(f)
    dec.DisallowUnknownFields()
    if err := dec.Decode(&raw); err != nil {
        var strictErr *toml.StrictMissingError
        if errors.As(err, &strictErr) {
            // strictErr.String() gives line/column-annotated context —
            // surface strictErr.Error() as the single-line startup error (D-02).
            return nil, fmt.Errorf("config: %s", strictErr.Error())
        }
        return nil, fmt.Errorf("config: %w", err)
    }

    cfg := DefaultConfig() // pre-populated with all defaults
    if raw.Theme != "" {
        if err := validateTheme(raw.Theme); err != nil {
            return nil, err
        }
        cfg.Theme = raw.Theme
    }
    if err := cfg.Keybindings.Merge(raw.Keybindings); err != nil { // D-01, D-02
        return nil, err
    }
    return cfg, nil
}
```

### Pattern 2: D-03-Compliant Config Discovery (No Side Effects)

**What:** Distinguish the read-only "does a config file exist" check from the write-capable "give me a path to create a file at" helper `adrg/xdg` exposes — they are different functions with different side effects.
**When to use:** CONFIG-01's default-path resolution when `--config` was not passed.
**Example:**
```go
// Source: direct inspection of github.com/adrg/xdg@v0.5.3 source (go mod download + read),
// confirmed this session — xdg.go: ConfigFile() creates parent dirs; SearchConfigFile() does not.
path, err := xdg.SearchConfigFile("alturd/config.toml")
if err != nil {
    // Not found — this is NOT an error condition. Silently use defaults (D-03).
    return DefaultConfig(), nil
}
return Load(path)
```
**Do NOT use `xdg.ConfigFile("alturd/config.toml")` for this check** — it creates `$XDG_CONFIG_HOME/alturd/` on disk as a side effect of merely resolving the path, violating D-03 even though no file content is written.

### Pattern 3: 50ms-Bounded OSC 11 Detection (Goroutine Race)

**What:** Since `termenv.OSCTimeout` is a hardcoded 5-second `const`, the caller must impose its own shorter timeout externally by racing the blocking call in a goroutine.
**When to use:** THEME-01's auto-detect path, standalone mode only (D-06 skips this entirely in difftool mode).
**Example:**
```go
// Source: verified against github.com/muesli/termenv@v0.16.0 termenv_unix.go this session —
// OSCTimeout = 5*time.Second is a const, not a var; cannot be reconfigured.
func detectDarkBackground(w io.Writer) bool {
    result := make(chan bool, 1)
    go func() {
        result <- termenv.NewOutput(w).HasDarkBackground()
        // If the 50ms case below fires first, this goroutine is abandoned but not leaked
        // forever: termenv's own internal waitForData(OSCTimeout) bounds it to 5s, after
        // which it returns and the goroutine exits normally (buffered channel absorbs the send).
    }()
    select {
    case dark := <-result:
        return dark
    case <-time.After(50 * time.Millisecond):
        return true // dark fallback (THEME-01, D-07)
    }
}
```

### Pattern 4: Difftool Mode Env-Var Read + Cmdline Mapping

**What:** Map git's `$LOCAL`/`$REMOTE`/`$MERGED` shell variables (substituted by `git-difftool--helper`'s `eval`) onto alturd's own `--difftool-local`/`--difftool-remote`/`--difftool-path` flags.
**When to use:** DIFFTOOL-01/02.
**Example:**
```go
// Source: git-scm.com/docs/git-difftool + git/git source (git-difftool--helper.sh) — verified this session.
// $LOCAL  = temp file with pre-image content  → --difftool-local
// $REMOTE = temp file with post-image content → --difftool-path is NOT $REMOTE — see below
// $MERGED = the real filename being compared (NOT a temp file) → --difftool-path
counter := os.Getenv("GIT_DIFF_PATH_COUNTER") // 1-based, set by diff.c's run_external_diff()
total := os.Getenv("GIT_DIFF_PATH_TOTAL")
// Title bar: fmt.Sprintf("alturd (difftool) — %s of %s — %s", counter, total, filepath.Base(mergedPath))
```

### Pattern 5: Idempotent `install-difftool` Read-Then-Write

**What:** Check the current `diff.tool` value before writing, to implement D-10's idempotency/`--force` contract.
**When to use:** DIFFTOOL-03.
**Example:**
```go
// Uses internal/git.Runner — but NOT ExecRunner.Run() unmodified; see Common Pitfalls.
scopeFlag := "--global" // or "--local" per --scope
existing, err := getConfigValue(scopeFlag, "diff.tool") // git config --get exits 1 if unset — not an error
if err == nil && existing != "" && existing != name && !force {
    return fmt.Errorf("diff.tool is already set to %q; pass --force to overwrite", existing)
}
setConfigValue(scopeFlag, "diff.tool", name)
setConfigValue(scopeFlag, fmt.Sprintf("difftool.%s.cmd", name),
    fmt.Sprintf(`alturd --difftool-local "$LOCAL" --difftool-remote "$REMOTE" --difftool-path "$MERGED"`))
setConfigValue(scopeFlag, "difftool.prompt", "false")
setConfigValue(scopeFlag, "difftool.trustExitCode", "true")
```

### Anti-Patterns to Avoid

- **Reusing `git.ExecRunner.Run()` verbatim for `git config` calls:** its error-mapping is tuned specifically for `git diff` exit-code semantics (hardcodes `"git diff: %s"` in wrapped error messages, and treats exit 128 as always-fatal). `git config --get` legitimately exits 1 when a key is simply unset — that is not an application error and must not be surfaced as one.
- **Calling `xdg.ConfigFile()` to check config existence:** creates directories as a side effect; violates D-03.
- **Trusting `termenv.OSCTimeout` as configurable:** it is a `const`; attempting `termenv.OSCTimeout = 50*time.Millisecond` will not compile.
- **Running OSC 11 detection in difftool mode:** D-06 explicitly forbids this; git's own subprocess environment for `git-difftool--helper` is not a plain interactive TTY context and is exactly the "subprocess/non-TTY" hang risk PITFALLS.md Pitfall 10 describes.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| TOML parsing + unknown-key rejection | A custom line-based TOML-ish parser | `go-toml/v2` `Decoder.DisallowUnknownFields()` | TOML 1.0 has real edge cases (multi-line strings, inline tables, array-of-tables); a hand parser will silently mis-handle them |
| XDG path resolution across OSes | `os.UserConfigDir()` + manual `$XDG_CONFIG_HOME` env checks | `adrg/xdg` (already a dependency) | `adrg/xdg` already correctly implements the XDG spec's full fallback chain (`XDG_CONFIG_HOME` → `~/.config` on Linux, `~/Library/Application Support` on macOS, `%LOCALAPPDATA%` on Windows) and is already used by `internal/log` — a second implementation would risk inconsistent path resolution between config and log |
| OSC 11 terminal query byte-parsing | Hand-rolled `\x1b]11;?\x07` write + response parse | `termenv.HasDarkBackground()` (already wired in Phase 3) | termenv already implements the `COLORFGBG` fallback and TTY/CI-environment short-circuit correctly; only the external timeout wrapper is missing, not the whole detection mechanism |
| Cross-platform binary build matrix | Hand-written `GOOS=... GOARCH=... go build` loop in a Makefile or shell script | `goreleaser` | goreleaser handles archive naming conventions, checksums, GitHub Release API upload, and the CGO_ENABLED-per-target footgun (Pitfall 8) that a hand-rolled script would need to reimplement from scratch |

**Key insight:** Every "don't hand-roll" item in this phase already has its library wired into the project (`go-toml/v2` is new but drop-in; `adrg/xdg` and `termenv` are already dependencies) — the actual engineering risk in Phase 4 is not "which library" but "which exact method/option on the already-chosen library," which is why the source-level verification in this research session (const vs var, `ConfigFile` vs `SearchConfigFile`) mattered more than library selection.

## Common Pitfalls

### Pitfall A: `termenv.OSCTimeout` Cannot Satisfy THEME-01's 50ms Requirement As-Is

**What goes wrong:** A naive Phase 4 implementation might assume "Phase 3 already calls `HasDarkBackground()`, so THEME-01's 50ms timeout is already handled." It is not — the call in `cmd/alturd/main.go` today blocks for up to termenv's internal 5-second `OSCTimeout` const in the worst case (e.g., a terminal that ignores OSC 11 entirely in a way that doesn't immediately error).
**Why it happens:** `OSCTimeout` is defined as `const OSCTimeout = 5 * time.Second` in `termenv_unix.go` — despite being capitalized (exported), Go constants cannot be reassigned by importing code the way exported `var`s can.
**How to avoid:** Wrap the call in a goroutine raced against `time.After(50*time.Millisecond)` (Pattern 3 above). This must ship as part of Phase 4, not be treated as already-done carryover from Phase 3.
**Warning signs:** Startup feels sluggish specifically over SSH or inside tmux/screen; a manual test with `ssh` piping through a slow link would reveal multi-second startup delay if this pitfall is not addressed.

### Pitfall B: `xdg.ConfigFile()` Creates Directories on First Run

**What goes wrong:** Calling `xdg.ConfigFile("alturd/config.toml")` merely to check "does a config exist" creates `$XDG_CONFIG_HOME/alturd/` on disk, even if the function's error return is discarded and no file is ever written into that directory. This silently violates D-03 ("nothing is written to disk on first run").
**Why it happens:** `adrg/xdg`'s `ConfigFile`/`DataFile`/`StateFile`/`CacheFile` family is designed for the *write* case (a caller who is about to create a file there) and creates parent directories as a documented convenience. The *read* case has a separate, non-creating function (`SearchConfigFile`) that is easy to overlook since it's not the first result alphabetically in the package's godoc.
**How to avoid:** Use `xdg.SearchConfigFile` for the default-path lookup (Pattern 2 above); reserve `xdg.ConfigFile` for a hypothetical future "alturd config init" feature that explicitly writes a template (out of scope per D-03).
**Warning signs:** An integration test that runs `alturd` with a clean `$XDG_CONFIG_HOME` and then asserts no directories were created under it — this test would need to exist for D-03 to be verifiable at all (see Validation Architecture).

### Pitfall C: Reusing `internal/git.ExecRunner.Run()`'s Error Mapping for `git config`

**What goes wrong:** `ExecRunner.Run()`'s error-handling branch is hand-tuned for `git diff`'s exit-code conventions: it special-cases exit 128 as always-fatal and hardcodes the string `"git diff: %s"` into every wrapped error message. Calling it unmodified for `git config --get diff.tool` (which exits 1, not 128, when the key is simply absent — a normal, expected outcome, not a failure) will either mis-classify a successful "key unset" result as an error, or produce a confusing `"git diff: exit status 1"` message for what is actually a `git config` call.
**Why it happens:** The `Runner` interface's single `Run([]string) (io.Reader, error)` method is generic, but `ExecRunner`'s current *implementation* of error interpretation is not — it was written when the only caller was the `diff` subcommand.
**How to avoid:** Either (a) add a second, `git config`-aware error-interpretation path (e.g. a small wrapper function in the new `install-difftool` code that calls the same `exec.Command("git", args...)` primitive but interprets exit codes per `git config`'s own conventions: 0=success, 1=key not found/section not found, 2=invalid config file, 128=not a git directory when writing `--local`), or (b) extend `ExecRunner.Run` to accept a caller-supplied error-classification strategy. Do not call the existing `Run()` and assume its error semantics generalize.
**Warning signs:** `install-difftool --scope local` run outside a repo produces a stack-trace-flavored `"git diff: ..."` error instead of a clear `"--scope local requires running inside a git repository"` message.

### Pitfall D: `--scope local` Requires Being Inside a Git Repository

**What goes wrong:** `git config --local` writes to `.git/config` and fails outside a repository. If `install-difftool --scope local` is run from a non-repo directory, the failure must be surfaced clearly (per the project's existing GIT-05 pattern for other commands), not as a raw git stderr dump.
**Why it happens:** Unlike `--global` (writes to `~/.gitconfig`, always available), `--local` scope is inherently repo-dependent.
**How to avoid:** Detect this case explicitly (reuse the existing "not a git repository" detection idiom already present in `git.ExecRunner.Run`, adapted per Pitfall C) and emit a single-line error consistent with the rest of alturd's CLI error conventions.
**Warning signs:** Confusing/verbose git stderr surfaces to the user instead of alturd's own clear single-line error format.

### Pitfall E: `difftool.<name>.cmd` Is Evaluated By a Shell — Windows Included

**What goes wrong:** Assuming `difftool.<name>.cmd` behaves like an argv-form `exec.Command` call (as `internal/git.ExecRunner` does elsewhere in this codebase) rather than a shell-evaluated string. If the `alturd` binary path or any argument needs quoting, it must be shell-quoted, not argv-escaped.
**Why it happens:** Git's own docs state the `cmd` value "is evaluated in shell." On Windows, this shell is the `sh.exe` bundled with Git for Windows (POSIX-style), not `cmd.exe` — so POSIX-style double-quoting (`"$LOCAL"`) works consistently across all three target platforms, which is fortunate but non-obvious.
**How to avoid:** Write the `cmd` value as a single POSIX-shell-syntax string: `` alturd --difftool-local "$LOCAL" --difftool-remote "$REMOTE" --difftool-path "$MERGED" `` — this is safe on Linux/macOS (native `sh`) and Windows (Git for Windows' bundled `sh`) alike, so long as the `alturd` binary itself is on `PATH` (document this as an install prerequisite, or embed an absolute path via `difftool.<name>.path` if a more robust variant is wanted later).
**Warning signs:** A gitconfig entry that works when tested with `git difftool -x` (extcmd, which bypasses shell quoting differently) but fails when invoked via `-t alturd`.

### Pitfall F (cross-checked against `.planning/research/PITFALLS.md`): CRLF Normalization Does Not Need Re-Verification for Difftool Mode

**What was checked:** Whether difftool mode's file-reading path (reading `$LOCAL`/`$REMOTE` temp files, or the real file at `$MERGED`) reintroduces Pitfall 9 (CRLF from git subprocess output on Windows).
**Finding:** Pitfall 9's fix (`git.NormalizeCRLF`, applied inside `ExecRunner.Run` immediately after `cmd.Output()`) is specific to *parsing git's own diff/show output*. Difftool mode does not call `git diff`/`git show` at all — it reads two already-materialized local files (`$LOCAL`/`$REMOTE` are temp files git itself writes to disk, and `$MERGED` is the real working-tree file) directly via `os.ReadFile`. This is a different code path than the one Pitfall 9 patches, and CRLF content inside those files is legitimate file content (not a git-output artifact) that should NOT be stripped — stripping it would corrupt the diff of any CRLF-formatted source file. **No action needed; do not apply `NormalizeCRLF` to difftool-mode file reads.**

### Pitfall G (cross-checked against `.planning/research/PITFALLS.md`): Signal Handler Accumulation Is a Non-Issue for Difftool Mode

**What was checked:** Whether Pitfall 11 (bubbletea `signal.Notify()` accumulation across multiple `tea.Program` instances) applies to difftool mode.
**Finding:** `git difftool` invokes the configured `cmd` (i.e., the `alturd` binary) as a **fresh OS process per file** — confirmed by reading `git-difftool--helper.sh`'s `eval $cmd` inside its per-path loop driven by `run_external_diff()`'s per-call `run_command(&cmd)` in `diff.c`. Each invocation is a new process with a fresh Go runtime and a fresh signal-handler table; there is no in-process loop across files within a single `alturd` invocation. **No action needed** — `tea.WithoutSignalHandler()` is unnecessary as long as `main.go` never itself loops to spawn multiple `tea.Program` instances within one process (it currently does not, and difftool mode's single-file design gives no reason to add one).

## Code Examples

### Flat `[keybindings]` TOML Schema (D-04)

```toml
# Source: derived from 03-UI-SPEC.md Key Binding Contract (10 actions) + go-toml/v2 struct-tag conventions verified this session
theme = "auto"   # "light" | "dark" | "auto" — D-05

[keybindings]
next_hunk = "n"
prev_hunk = "N"
next_file = "]"
prev_file = "["
toggle_focus = "tab"
toggle_render_mode = "v"
open_search = "/"
toggle_all_files = "a"
quit = "q"
abort = "Q"
```

```go
// Source: pattern derived from go-toml/v2 verified API this session
type rawConfig struct {
    Theme       string            `toml:"theme"`
    Keybindings map[string]string `toml:"keybindings"`
}
```
Using `map[string]string` for `[keybindings]` (rather than a fixed struct with 10 named fields) means `DisallowUnknownFields()` cannot catch a typo'd action name (e.g. `nex_hunk`) at the TOML-decode layer — that check must happen explicitly in application code against the list of known action names, consistent with D-02's requirement that unrecognized key-string *values* are also rejected (the same validation pass can cover both unknown action names and unrecognized key strings).

### `install-difftool` Cobra Subcommand Skeleton

```go
// Source: pattern consistent with existing rootCmd conventions in cmd/alturd/main.go
var installDifftoolCmd = &cobra.Command{
    Use:   "install-difftool",
    Short: "Register alturd as a git difftool",
    RunE:  runInstallDifftool,
}

func init() {
    installDifftoolCmd.Flags().String("scope", "global", "config scope: global|local")
    installDifftoolCmd.Flags().String("name", "alturd", "difftool name to register")
    installDifftoolCmd.Flags().Bool("force", false, "overwrite an existing diff.tool setting")
    rootCmd.AddCommand(installDifftoolCmd)
}
```

### GitHub Actions CI Matrix (DIST-01, D-12, D-13)

```yaml
# Source: goreleaser.com + golangci-lint.run official docs, GitHub Actions default-shell
# behavior confirmed this session (windows-latest run steps default to pwsh — Pitfall 7
# from PITFALLS.md is already mitigated by GH Actions' own default, no extra shell: config needed)
name: ci
on:
  push:
  pull_request:
jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-go@v5   # pinned per CLAUDE.md — search results suggested v6 exists,
        with:                        # but CLAUDE.md's stated version takes precedence
          go-version: stable
          cache: true
      - run: go test ./...
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-go@v5
        with:
          go-version: stable
          cache: true
      - uses: golangci/golangci-lint-action@v7
        with:
          version: v2.7.2   # >= v2.4.0 required for Go 1.25 support
```

### `.golangci.yml` (v2 Schema, D-13)

```yaml
# Source: golangci-lint.run official v2 configuration docs, verified this session
version: "2"
linters:
  default: standard
  enable:
    - staticcheck
    - govet
    - errcheck
    - revive
```

### `.goreleaser.yaml` (DIST-02, DIST-03, D-11)

```yaml
# Source: goreleaser.com official builds docs, verified this session
version: 2

builds:
  - id: alturd
    main: ./cmd/alturd
    binary: alturd
    env:
      - CGO_ENABLED=0
    flags:
      - -trimpath
    ldflags:
      - -s -w -X main.version={{.Version}}
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64   # matches REQUIREMENTS.md Out of Scope: "Windows arm64 binary"

archives:
  - formats: [tar.gz]
    format_overrides:
      - goos: windows
        formats: [zip]

checksum:
  name_template: "checksums.txt"
```

### `.github/workflows/release.yml` (DIST-02, D-11)

```yaml
# Source: goreleaser.com CI docs + goreleaser-action README, verified this session
name: release
on:
  push:
    tags:
      - "v*.*.*"   # D-11
permissions:
  contents: write
jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with:
          fetch-depth: 0   # required for changelog generation
      - uses: actions/setup-go@v5
        with:
          go-version: stable
      - uses: goreleaser/goreleaser-action@v7
        with:
          distribution: goreleaser
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `golangci-lint` v1 `.golangci.yml` schema (`linters-settings`, no `formatters` section) | `golangci-lint` v2 schema (`version: "2"`, top-level `linters`/`formatters`/`exclusions`) | v2.0 release (2025) | The many v1-style config examples still circulating in blog posts/search results will fail v2's schema validation; use the `version: "2"` shape shown in Code Examples |
| `goreleaser` v1 `.goreleaser.yml` (no `version:` key) | `goreleaser` v2 requires explicit top-level `version: 2` | goreleaser v2.0 | Omitting `version: 2` in a fresh config either defaults incorrectly or errors depending on goreleaser release — always include it explicitly |
| `golangci-lint-action@v4`–`v6` (implicit Go install, various removed options) | `golangci-lint-action@v7`+ requires an explicit prior `actions/setup-go` step | v4.0.0 breaking change | Confirmed still true for v7/v8/v9 — the workflow in Code Examples includes the explicit `setup-go` step |

**Deprecated/outdated:**
- golangci-lint v1 binaries: will not lint a `go 1.25.0` module correctly even if installed; must be v2.4.0+.
- `goreleaser-action@v4` and earlier `skip-go-installation`/`skip-pkg-cache` options: removed in later action majors; not referenced in this research's recommended workflow.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|----------------|
| A1 | `golangci-lint-action@v7` is compatible with `golangci-lint v2.7.2` (the version pinned in the CI example) | Code Examples, Standard Stack | Low — action-vs-tool version compatibility is documented by the action itself and will fail loudly (action install step errors) rather than silently, so this is self-correcting during first CI run; confirm during Wave 0 |
| A2 | `difftool.<name>.cmd` invoked via `-t <name>` (not `-x`/`--extcmd`) resolves `alturd` purely via `PATH` with no `difftool.<name>.path` key needed | Difftool Setup, Pitfall E | Medium — if a user's `PATH` doesn't include the alturd install location (e.g. installed to a non-standard directory), `install-difftool` would need to also offer writing an absolute path via `difftool.<name>.path`; not discussed in CONTEXT.md and not in the locked 4-key set, so this is intentionally out of scope unless a real user hits it (matches project's general "revisit only if a real user hits a wall" philosophy already used elsewhere in FEATURES.md) |
| A3 | goreleaser v2.16 (the version named in CLAUDE.md) uses the same `.goreleaser.yaml` schema shown in Code Examples — the WebFetch source used to construct this schema did not report an exact goreleaser point-version alongside the schema shown | Code Examples (.goreleaser.yaml) | Low — the `version: 2` schema has been stable across the v2.x line since goreleaser 2.0; a schema-breaking change mid-v2 would be unusual and the planner should still run `goreleaser check` against the generated config as a Wave 0/verification step |

**If this table is empty:** N/A — see entries above. All three assumptions are low-to-medium risk and self-verifying (CI will surface a version mismatch immediately rather than silently misbehaving).

## Open Questions (RESOLVED)

1. **Should `install-difftool` also offer `difftool.<name>.path`?**
   - What we know: The 4-key set (D-08, confirmed) covers the minimum viable setup assuming `alturd` is on `PATH`.
   - What's unclear: Whether the planner wants a `--path-to-binary` flag or `os.Executable()`-derived absolute path as a defensive addition.
   - Recommendation: Ship the locked 4-key set only for Phase 4 (matches D-08 exactly); leave this as a documented future enhancement, not a Phase 4 task, since CONTEXT.md's decisions did not request it and adding it unrequested would be scope creep.

2. **Exact wording of the CONFIG-02 duplicate-keybinding error message.**
   - What we know: D-02 requires "a clear single-line error" for duplicate/conflicting key bindings, with no further wording specified.
   - What's unclear: Exact phrasing.
   - Recommendation: Planner/executor discretion — follow the existing single-line error convention already established in `internal/git.ExecRunner` (e.g. `"config: key %q is bound to both %q and %q"`).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|--------------|-----------|---------|----------|
| Go toolchain | All of Phase 4 | ✓ | go1.25.11 linux/amd64 (matches go.mod `go 1.25.0`) | — |
| git | Difftool setup/testing | ✓ | present in this environment (used for all research verification above) | — |
| goreleaser CLI | Local `goreleaser build --snapshot` smoke test (recommended Wave 0 step, not installed in this research environment) | ✗ (not checked — not installed in this sandbox) | — | CI (`goreleaser-action`) installs it automatically; local dev can install via `go install github.com/goreleaser/goreleaser/v2@latest` or skip local testing and rely on CI's first tag-triggered dry run |
| golangci-lint CLI | Local lint runs (recommended dev-loop step) | ✗ (not checked — not installed in this sandbox) | — | CI (`golangci-lint-action`) installs its own pinned version; not a blocker for planning or execution |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** goreleaser CLI and golangci-lint CLI are not installed in this research sandbox, but both have GitHub-Actions-managed installation as the actual execution path for DIST-01/02/03 — local installation is a developer-experience nicety, not a phase blocker.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package (table-driven), consistent with all prior phases |
| Config file | none — `go test ./...` requires no config |
| Quick run command | `go test ./internal/config/... ./cmd/alturd/... -run TestConfig -v` (once `internal/config` exists) |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|---------------------|-------------|
| CONFIG-01 | Unknown TOML key rejected at startup with clear error | unit | `go test ./internal/config/... -run TestLoad_UnknownField -v` | ❌ Wave 0 |
| CONFIG-01 | `--config <path>` overrides default XDG lookup | unit | `go test ./internal/config/... -run TestLoad_ExplicitPath -v` | ❌ Wave 0 |
| CONFIG-01 | No config file present → defaults used, **no directory created** | unit (uses `t.Setenv("XDG_CONFIG_HOME", t.TempDir())` + post-assert dir tree) | `go test ./internal/config/... -run TestLoad_NoSideEffectsOnFirstRun -v` | ❌ Wave 0 — this test is the only reliable way to catch Pitfall B (`xdg.ConfigFile` vs `SearchConfigFile`) regressions |
| CONFIG-02 | Config overrides one keybinding, others keep defaults | unit | `go test ./internal/config/... -run TestKeybindings_PartialOverride -v` | ❌ Wave 0 |
| CONFIG-02 | Duplicate/conflicting key bindings rejected | unit | `go test ./internal/config/... -run TestKeybindings_DuplicateRejected -v` | ❌ Wave 0 |
| THEME-01 | Auto-detect falls back to dark within 50ms when OSC 11 does not respond | unit (fake `io.Writer` that never writes a response, assert elapsed time bound) | `go test ./cmd/alturd/... -run TestDetectDarkBackground_TimeoutFallback -v` | ❌ Wave 0 |
| THEME-01 | `--theme`/config precedence order (D-07) | unit | `go test ./internal/config/... -run TestTheme_Precedence -v` | ❌ Wave 0 |
| DIFFTOOL-01 | `--difftool-local/-remote/-path` renders single-file view without tree | integration (subprocess, following existing `TestMain` pattern from Phase 2) | `go test ./cmd/alturd/... -run TestDifftoolMode -v` | ❌ Wave 0 |
| DIFFTOOL-02 | Title bar shows "N of M" from env vars | unit (model-level, following `internal/tui/model_test.go` conventions) | `go test ./internal/tui/... -run TestDifftoolTitleBar -v` | ❌ Wave 0 |
| DIFFTOOL-03 | `install-difftool` writes 4 keys idempotently; `--force` semantics | integration (subprocess against a scratch `HOME`/repo, following `TestMain` pattern) | `go test ./cmd/alturd/... -run TestInstallDifftool -v` | ❌ Wave 0 |
| DIST-01 | `go test ./...` runs on 3 OSes | CI-only (not locally automatable beyond running the suite itself) | `go test ./...` (in `ci.yml` matrix) | N/A — CI config, not a Go test |
| DIST-02/03 | goreleaser produces `CGO_ENABLED=0` binaries for the correct platform matrix | manual / CI smoke (recommended: `goreleaser check` + `goreleaser build --snapshot --clean` as a Wave 0 or pre-tag verification step, not a `go test`) | `goreleaser check && goreleaser build --snapshot --clean` | N/A — build tooling, not a Go test |

### Sampling Rate

- **Per task commit:** `go test ./internal/config/... ./cmd/alturd/...` (or the touched package)
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd-verify-work`, plus `goreleaser check` passing against the new `.goreleaser.yaml`

### Wave 0 Gaps

- [ ] `internal/config/config_test.go` — covers CONFIG-01, CONFIG-02
- [ ] `internal/config/theme_test.go` — covers THEME-01 precedence and timeout-fallback
- [ ] `cmd/alturd/difftool_test.go` (or extend `main_test.go`) — covers DIFFTOOL-01, DIFFTOOL-03 via the existing `TestMain` subprocess pattern (`Phase 2 Plan 3` decision log)
- [ ] `internal/tui/model_test.go` extension — covers DIFFTOOL-02 title bar format
- [ ] No new test framework install needed — `go test` is already fully wired

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|-------------------|
| V2 Authentication | no | N/A — alturd has no auth surface |
| V3 Session Management | no | N/A |
| V4 Access Control | no | N/A — single-user local CLI |
| V5 Input Validation | yes | TOML config parsing (`DisallowUnknownFields` + explicit action/key-string validation, D-02); CLI flag validation for `--difftool-*` paths and `install-difftool --scope`/`--name` |
| V6 Cryptography | no | N/A — no crypto operations in this phase |

### Known Threat Patterns for This Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-----------------------|
| Shell injection via `difftool.<name>.cmd` string construction | Tampering | The `cmd` value written by `install-difftool` is a **static string** (`alturd --difftool-local "$LOCAL" ...`) with no user-supplied interpolation — `$LOCAL`/`$REMOTE`/`$MERGED` are git's own shell variables substituted by git itself, not by alturd's Go code, so there is no string-formatting injection surface in the write path. The existing project convention (argv-form `exec.Command`, never shell interpolation of untrusted input — already documented in `internal/git/runner.go`'s SECURITY comment) should be followed for the `git config --set` calls themselves |
| Path traversal via `--config <path>` or `--difftool-path <path>` | Tampering | These are user-supplied local paths for a local CLI tool the user is already running with their own privileges — no privilege boundary is crossed by reading an arbitrary local file the invoking user can already read; standard `os.Open`/`os.ReadFile` error handling (not a custom path-sanitization layer) is sufficient, consistent with how `internal/git` already handles user-supplied paths (ASVS V5, referenced in `runner.go`'s existing SECURITY comment for ref/path arguments) |
| TOML "billion laughs"-style resource exhaustion via deeply nested/duplicated structures | Denial of Service | Not a meaningful threat for a local single-user CLI processing a config file the user themselves authored; no mitigation beyond go-toml/v2's own parser limits is warranted — flagging only for completeness, no action needed |
| Config file world-readable/writable permissions leaking keybinding preferences | Information Disclosure | Not sensitive data (no secrets in a keybinding/theme TOML file) — no special file-permission handling needed, unlike `internal/log`'s `0600` log file mode (which was chosen because logs could contain path/diff content, a different threat model) |

## Sources

### Primary (HIGH confidence)

- Direct source inspection of `github.com/muesli/termenv@v0.16.0` (`go mod download` + local read) — `OSCTimeout` const finding
- Direct source inspection of `github.com/adrg/xdg@v0.5.3` (`go mod download` + local read) — `ConfigFile` vs `SearchConfigFile` finding
- Direct source inspection of `git/git` master (`diff.c`, `builtin/difftool.c`, `git-difftool--helper.sh`, fetched via raw.githubusercontent.com and grepped locally) — `GIT_DIFF_PATH_COUNTER`/`GIT_DIFF_PATH_TOTAL` propagation through `git difftool`
- [git-scm.com/docs/git-difftool](https://git-scm.com/docs/git-difftool) — official docs for `diff.tool`, `difftool.<tool>.cmd`, `difftool.prompt`, `difftool.trustExitCode`, `--tool-help`, `$LOCAL`/`$REMOTE`/`$MERGED`/`$BASE`
- `go list -m -versions github.com/pelletier/go-toml/v2` + `proxy.golang.org/.../@latest` — version currency confirmation

### Secondary (MEDIUM confidence)

- [pkg.go.dev/github.com/pelletier/go-toml/v2](https://pkg.go.dev/github.com/pelletier/go-toml/v2) — `Decoder.DisallowUnknownFields()` API shape (WebFetch-summarized, cross-checked against the package's documented behavior)
- [goreleaser.com/customization/builds/go/](https://goreleaser.com/customization/builds/go/) — `.goreleaser.yaml` schema
- [golangci-lint.run/docs/configuration/file/](https://golangci-lint.run/docs/configuration/file/) — v2 config schema
- [github.com/golangci/golangci-lint issues #5873](https://github.com/golangci/golangci-lint/issues/5873) — Go 1.25 support version threshold
- [github.com/golangci/golangci-lint-action README](https://github.com/golangci/golangci-lint-action) — action version compatibility table

### Tertiary (LOW confidence)

- Community wiki examples of `difftool.<tool>.cmd` Windows quoting (SourceGear/WinMerge gitconfig snippets) — used only to corroborate that git evaluates `cmd` via shell on Windows too, not as a source of the actual command string used in this project (which is derived directly from the official docs' `$LOCAL`/`$REMOTE`/`$MERGED` variable names instead)
- General WebSearch summaries for `goreleaser-action@v7` and `golangci-lint-action` workflow YAML shape — cross-checked against goreleaser's own official docs page where possible, but the exact `actions/checkout`/`actions/setup-go` point versions in search results (`v6`) were overridden in this document's Code Examples with CLAUDE.md's explicitly pinned `v5` per project constraint precedence

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every library either already vetted in a prior phase or confirmed current via the Go module proxy directly
- Architecture: HIGH — patterns derived from direct source-code verification of the exact pinned dependency versions, not general knowledge
- Pitfalls: HIGH — the two most consequential pitfalls (Pitfall A, Pitfall B) were discovered by reading actual dependency source, not inferred from documentation gaps

**Research date:** 2026-07-27
**Valid until:** 2026-08-26 (30 days — dependency versions and goreleaser/golangci-lint schemas are stable but not guaranteed unchanged beyond this window; re-verify `go list -m -versions` for `go-toml/v2` before execution if this research is consumed significantly later)
