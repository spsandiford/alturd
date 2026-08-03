---
phase: 04-config-theming-difftool-distribution
plan: 01
subsystem: config
tags: [go-toml, xdg, bubbletea, keybindings, tui]

# Dependency graph
requires:
  - phase: 03-tui-application
    provides: internal/tui/model.go's normal-mode key dispatch switch and the ten Phase 3 default keybindings (03-UI-SPEC.md Key Binding Contract)
provides:
  - internal/config package: Action constants, DefaultKeymap, Keymap.Lookup/Merge, Config, DefaultConfig, Load(explicitPath)
  - A keymap-driven internal/tui/model.go normal-mode dispatch (single Keymap.Lookup call, no hardcoded key literals)
  - --config flag on cmd/alturd's rootCmd wired through config.Load into tui.NewModel
  - Strict two-directional keybinding validation (unknown action, unrecognized key, duplicate/shadowed key) with the exact 04-UI-SPEC.md error templates
  - Read-only XDG config discovery proven to create zero files/directories on first run (D-03)
affects: [04-03-theming (same config.Load entrypoint decodes the theme key), 04-02-distribution (first tagged release publishes these action names as a user-facing contract)]

# Tech tracking
tech-stack:
  added: [github.com/pelletier/go-toml/v2 v2.4.3]
  patterns:
    - "Keymap.Merge validates in this fixed order: unknown action names -> unrecognized key strings -> merge -> duplicate scan over the merged map (merge-then-validate, not validate-then-merge)"
    - "All package-level action iteration goes through the canonicalActions slice, never Go map range order, so generated error messages are byte-identical across runs"
    - "config.Load is called from cmd/alturd's RunE only, mirroring internal/log's Init discipline, so --help/--version never touch the filesystem"

key-files:
  created:
    - internal/config/config.go
    - internal/config/keybindings.go
    - internal/config/config_test.go
    - internal/config/keybindings_test.go
  modified:
    - internal/tui/model.go
    - internal/tui/model_test.go
    - cmd/alturd/main.go
    - go.mod
    - go.sum

key-decisions:
  - "D-04 locked: flat [keybindings] TOML table, snake_case action names. Ten action names (verbatim, in canonical order): next_hunk, prev_hunk, next_file, prev_file, toggle_focus, toggle_render_mode, open_search, toggle_all_files, quit, abort."
  - "Merge-then-validate for duplicate detection: the merged map (not the override map alone) is scanned for two actions sharing a key, so a rebind that collides with a different, untouched action's default key is caught (e.g. rebinding next_hunk to the still-default quit key q)."
  - "Override inspection order is sorted (not map range order) for the unknown-action and unrecognized-key checks, and the duplicate scan walks canonicalActions, so every Merge error is deterministic across runs."

patterns-established:
  - "Pattern: strict TOML validation lives in application code (Keymap.Merge), not the decoder, because the config-file value being validated (keybinding actions) is a map[string]string, not a fixed struct DisallowUnknownFields can inspect."
  - "Pattern: read-only XDG discovery uses only xdg.SearchConfigFile, never a directory-creating helper from the same package; TestLoad_NoSideEffectsOnFirstRun is the regression guard, and was verified to actually fail when temporarily swapped to xdg.ConfigFile."

requirements-completed: [CONFIG-01, CONFIG-02]

coverage:
  - id: D1
    description: "A TOML config file that rebinds one action changes which key the running TUI dispatches that action on; unspecified actions keep their default key (D-01)."
    requirement: "CONFIG-02"
    verification:
      - kind: unit
        ref: "internal/tui/model_test.go#TestKeymapOverrideDispatchesInModel"
        status: pass
      - kind: unit
        ref: "internal/config/config_test.go#TestLoad_PartialKeybindingOverride"
        status: pass
      - kind: unit
        ref: "internal/config/keybindings_test.go#TestKeybindings_PartialOverride"
        status: pass
    human_judgment: false
  - id: D2
    description: "--config <path> loads the named file; a missing --config target is a startup error, never a silent fallback to defaults (CONFIG-01)."
    requirement: "CONFIG-01"
    verification:
      - kind: unit
        ref: "internal/config/config_test.go#TestLoad_ExplicitPath"
        status: pass
      - kind: unit
        ref: "internal/config/config_test.go#TestLoad_MissingExplicitPath"
        status: pass
    human_judgment: false
  - id: D3
    description: "A clean XDG_CONFIG_HOME with no --config flag runs with all defaults and creates zero files/directories (D-03)."
    requirement: "CONFIG-01"
    verification:
      - kind: unit
        ref: "internal/config/config_test.go#TestLoad_NoSideEffectsOnFirstRun"
        status: pass
    human_judgment: false
  - id: D4
    description: "Unknown TOML field, unknown keybinding action, unrecognized key string, and duplicate/shadowed key each abort startup with their own single-line config: message, no silent last-one-wins (D-02)."
    requirement: "CONFIG-01"
    verification:
      - kind: unit
        ref: "internal/config/config_test.go#TestLoad_UnknownField"
        status: pass
      - kind: unit
        ref: "internal/config/keybindings_test.go#TestKeybindings_UnknownAction"
        status: pass
      - kind: unit
        ref: "internal/config/keybindings_test.go#TestKeybindings_UnrecognizedKey"
        status: pass
      - kind: unit
        ref: "internal/config/keybindings_test.go#TestKeybindings_DuplicateRejected"
        status: pass
      - kind: unit
        ref: "internal/config/keybindings_test.go#TestKeybindings_DuplicateErrorDeterministic"
        status: pass
      - kind: unit
        ref: "internal/config/config_test.go#TestLoad_ErrorsAreSingleLine"
        status: pass
    human_judgment: false
  - id: D5
    description: "TUI keypress dispatch was end-to-end verified live in a real terminal with a rebound quit key (Task 2 human-verify checkpoint)."
    verification:
      - kind: manual_procedural
        ref: "Task 2 checkpoint: user approved live terminal verification of `x` rebound to quit via XDG_CONFIG_HOME config"
        status: pass
    human_judgment: true
    rationale: "Real-terminal keypress dispatch was explicitly gated as a human-verify checkpoint per the plan; already confirmed by the user before this continuation began."

# Metrics
duration: ~35min (this continuation segment; Task 1-2 ran in a prior session)
completed: 2026-08-03
status: complete
---

# Phase 04 Plan 01: Config + Keybindings Foundation Summary

**TOML-driven keybinding overrides with strict two-directional validation (unknown action, unrecognized key, duplicate/shadowed key) and read-only XDG config discovery, wired end-to-end into bubbletea's keypress dispatch.**

## Performance

- **Duration:** ~35 min (this continuation segment covering Task 3; Tasks 1-2 completed in a prior session)
- **Completed:** 2026-08-03
- **Tasks:** 3 (1 decision checkpoint, 2 code tasks)
- **Files modified:** 9 (go.mod, go.sum, internal/config/{config,keybindings,config_test,keybindings_test}.go, internal/tui/{model,model_test}.go, cmd/alturd/main.go)

## Accomplishments
- Locked the D-04 TOML keybinding schema: flat `[keybindings]` table, snake_case action names, ten actions enumerated verbatim (`next_hunk`, `prev_hunk`, `next_file`, `prev_file`, `toggle_focus`, `toggle_render_mode`, `open_search`, `toggle_all_files`, `quit`, `abort`)
- Built `internal/config` from scratch: `Action`/`Keymap`/`Config` types, `DefaultKeymap`, `DefaultConfig`, `Keymap.Lookup`, `Keymap.Merge`, and `Load(explicitPath)` with strict-unknown-field TOML decode over a read-only XDG search path
- Refactored `internal/tui/model.go`'s entire normal-mode dispatch switch to resolve every keypress through one `Keymap.Lookup` call before switching on `config.Action*` constants — no hardcoded key literal (e.g. `case "q":`) remains for a rebindable action
- Wired `--config <path>` onto `cmd/alturd`'s `rootCmd`, calling `config.Load` before `tui.NewModel` and propagating a load error as a single-line stderr message with exit 1
- Implemented strict two-directional D-02 validation in `Keymap.Merge`: unknown action names, unrecognized key strings, and merge-then-validate duplicate/shadow detection, each producing the exact single-line `04-UI-SPEC.md` error template with deterministic (canonical-order) phrasing
- Proved D-03 with `TestLoad_NoSideEffectsOnFirstRun`, and manually confirmed the test actually fails when the read path is swapped to a directory-creating XDG helper (then reverted)

## Task Commits

Each task was committed atomically:

1. **Task 1: Lock the TOML keybinding schema shape (D-04)** - decision only, no commit (recorded in Task 2's commit message and this SUMMARY)
2. **Task 2: End-to-end keybinding override — tracer** - `464853d` (feat) — human-verify checkpoint approved with no issues
3. **Task 3: Strict two-directional config validation and no-side-effect discovery (D-02, D-03)** - `55511bd` (feat), `7e0f68c` (test)

**Plan metadata:** pending (this commit)

## Files Created/Modified
- `internal/config/keybindings.go` - `Action` constants, `canonicalActions`, `DefaultKeymap`, `Keymap.Lookup`, `validKeyString` (single-rune / named-key allowlist / modifier-form regex), and the rewritten strict `Keymap.Merge`
- `internal/config/config.go` - `rawConfig`, `Config`, `DefaultConfig`, `Load(explicitPath)` with strict TOML decode and read-only XDG discovery
- `internal/config/config_test.go` - `TestLoad_ExplicitPath`, `TestLoad_PartialKeybindingOverride`, `TestLoad_MissingExplicitPath`, `TestLoad_NoConfigFileUsesDefaults`, `TestLoad_UnknownField`, `TestLoad_NoSideEffectsOnFirstRun`, `TestLoad_ErrorsAreSingleLine`
- `internal/config/keybindings_test.go` - `TestKeybindings_PartialOverride`, `TestKeybindings_UnknownAction`, `TestKeybindings_UnrecognizedKey`, `TestKeybindings_DuplicateRejected` (two subtests), `TestKeybindings_DuplicateErrorDeterministic`
- `internal/tui/model.go` - `keys config.Keymap` model field, `NewModel(files, darkBg, keys)`, keymap-driven normal-mode dispatch
- `internal/tui/model_test.go` - `newModelWith` updated for the new constructor signature; `TestKeymapOverrideDispatchesInModel` added
- `cmd/alturd/main.go` - `--config` persistent flag, `config.Load` call in `run()`
- `go.mod` / `go.sum` - `github.com/pelletier/go-toml/v2` promoted to direct; `github.com/adrg/xdg` and `github.com/muesli/termenv` promoted from indirect to direct

## Decisions Made
- **D-04 (locked):** flat-snake-case `[keybindings]` schema over grouped-by-pane or kebab-case, because nine of the ten actions are pane-independent (03-UI-SPEC.md) so there is nothing meaningful to group by, and flat decodes to a single `map[string]string` with one validation pass.
- **Merge-then-validate ordering:** duplicate detection scans the *merged* keymap, not the raw override map, so a rebind colliding with a different action's untouched default is caught — validating the override map alone would silently produce a shadowed binding, which D-02 forbids.
- **Deterministic error ordering:** override names are sorted before inspection and the duplicate scan walks `canonicalActions`, never Go's randomized map order, so re-running the same bad config produces a byte-identical error message every time.

## Deviations from Plan

None - plan executed as written. Task 3's action list was implemented verbatim; the one addition (`TestLoad_ErrorsAreSingleLine` and strengthening `TestLoad_MissingExplicitPath` to assert the `config: ` prefix) is in-scope test hardening for the plan's own `<verification>` requirement ("Every one of the six error templates ... is asserted verbatim by a test") and the acceptance criterion "No error returned by `config.Load` contains a newline character" — not a new feature, not Rule 4 territory.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `internal/config.Load` is the single entrypoint Plan 04-03 (theming) extends to validate the `theme` key (`config: invalid theme "{value}" (must be "light", "dark", or "auto")` — D-05) without re-shaping `rawConfig` or `Config`, since `Theme string` is already decoded and carried.
- The ten action names are now a published contract surface for Plan 04-02 (distribution) — the first tagged release ships user-facing config documentation against exactly this list.
- No blockers. Zero Phase 1-3 test regressions; `go build ./...`, `go test ./...`, and `go vet ./...` are all green.

---
*Phase: 04-config-theming-difftool-distribution*
*Completed: 2026-08-03*
