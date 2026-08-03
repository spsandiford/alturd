---
phase: 04-config-theming-difftool-distribution
plan: 03
subsystem: theming-difftool
tags: [termenv, bubbletea, lipgloss, difftool, osc11, theme]

# Dependency graph
requires:
  - phase: 04-config-theming-difftool-distribution (plan 01)
    provides: internal/config.Config/Load/Keymap and the --config flag; Config.Theme decoded (but not yet validated) as the field this plan retypes and validates
provides:
  - internal/config/theme.go: Theme type, ParseTheme, 50ms-bounded DetectDarkBackground, D-05/D-06/D-07 ResolveDarkBackground precedence resolver
  - --theme, --difftool-local, --difftool-remote, --difftool-path flags on rootCmd, dispatching to a difftool code path in cmd/alturd/main.go (difftoolDiff, loadDifftoolFiles, difftoolCounters, splitDifftoolFileLines)
  - tui.DifftoolInfo struct and the NewModel(files, darkBg, keys, dt) four-arg constructor signature
  - Difftool reduced-chrome layout in internal/tui/model.go: full-width diff pane, no tree pane, difftoolTitleBar() with the three DIFFTOOL-02 templates, Tab/'a' no-ops
affects: [04-04-distribution (writes a gitconfig `cmd` string that must name the --difftool-* flags exactly as registered here; also produces cmd/alturd/difftool.go and the install-difftool subcommand)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "50ms-bounded external race around termenv.HasDarkBackground: buffered result channel + select against time.After(DetectTimeout), because termenv.OSCTimeout is an unexported 5s const the caller cannot lower"
    - "ResolveDarkBackground is a pure function with an injectable detector func() bool, making the full D-05/D-06/D-07 precedence chain table-testable with zero real-terminal dependency"
    - "difftoolDiff interprets git diff --no-index's own exit-code contract (0=identical, 1=differs, other=failure) rather than reusing internal/git.ExecRunner's diff-runner-specific error mapping"
    - "GIT_DIFF_PATH_COUNTER/TOTAL are validated via strconv.Atoi and re-rendered with %d — the raw env string is never interpolated into rendered output"
    - "DifftoolInfo{} zero value means standalone mode; m.difftool.Enabled gates handleResize/refreshDiffContent/View/handleKey branches so no difftool-only code path executes for existing standalone callers"

key-files:
  created:
    - internal/config/theme.go
    - internal/config/theme_test.go
    - cmd/alturd/difftool_test.go
  modified:
    - internal/config/config.go
    - cmd/alturd/main.go
    - internal/tui/model.go
    - internal/tui/model_test.go

key-decisions:
  - "D-05/D-06/D-07 precedence implemented as ResolveDarkBackground(flagTheme, cfgTheme, difftoolMode, detect): --theme flag > config theme key > difftool-mode dark fallback (detect never called) > detect() > dark fallback when detect is nil."
  - "DetectDarkBackground bounds termenv's OSC 11 query to 50ms via an external goroutine race (buffered channel + select/time.After), since termenv.OSCTimeout is a const the importing package cannot override."
  - "tui.NewModel's DifftoolInfo struct parameter (not four positional params) was introduced in Task 2 rather than Task 3, because Task 2's own action list requires calling the four-arg NewModel and its own <verify> gate (go build ./...) cannot pass without it — documented as a Rule 3 auto-fix / task-ordering deviation."

patterns-established:
  - "Theme resolution and OSC 11 detection live entirely in internal/config; cmd/alturd/main.go no longer imports termenv directly."
  - "Difftool-mode-only fields are grouped into one DifftoolInfo struct on the model (mirrors config.Keymap's earlier struct-not-positional-params precedent from 04-01), keeping NewModel's signature stable as difftool functionality grows."

requirements-completed: [THEME-01, DIFFTOOL-01, DIFFTOOL-02]

coverage:
  - id: D1
    description: "ParseTheme validates light/dark/auto (and treats empty as auto), rejecting any other value with the exact 04-UI-SPEC.md error string (D-05)."
    requirement: "THEME-01"
    verification:
      - kind: unit
        ref: "internal/config/theme_test.go#TestParseTheme"
        status: pass
      - kind: e2e
        ref: "manual: alturd --theme purple exits 1 with 'config: invalid theme \"purple\" (must be \"light\", \"dark\", or \"auto\")'"
        status: pass
    human_judgment: false
  - id: D2
    description: "ResolveDarkBackground implements the full D-05/D-06/D-07 precedence chain (flag > config key > difftool-skip > auto-detect > dark fallback), asserted via detector call counts, not just return values."
    requirement: "THEME-01"
    verification:
      - kind: unit
        ref: "internal/config/theme_test.go#TestTheme_Precedence"
        status: pass
    human_judgment: false
  - id: D3
    description: "DetectDarkBackground bounds the OSC 11 query to 50ms with a dark fallback rather than blocking on termenv's internal 5s const."
    requirement: "THEME-01"
    verification:
      - kind: unit
        ref: "internal/config/theme_test.go#TestDetectDarkBackground_TimeoutFallback"
        status: pass
    human_judgment: false
  - id: D4
    description: "All three --difftool-* flags are required together; a missing one aborts startup with the exact single-line message (DIFFTOOL-01)."
    requirement: "DIFFTOOL-01"
    verification:
      - kind: integration
        ref: "cmd/alturd/difftool_test.go#TestDifftoolModeMissingFlags"
        status: pass
    human_judgment: false
  - id: D5
    description: "Identical --difftool-local/--difftool-remote files exit 0 and print 'No changes found.', mirroring the standalone empty-state guard."
    requirement: "DIFFTOOL-01"
    verification:
      - kind: integration
        ref: "cmd/alturd/difftool_test.go#TestDifftoolModeIdenticalFiles"
        status: pass
    human_judgment: false
  - id: D6
    description: "Difftool mode renders a full-width diff pane with no tree pane/separator column; Tab and 'a' are no-ops that never touch tree state (DIFFTOOL-01)."
    requirement: "DIFFTOOL-01"
    verification:
      - kind: unit
        ref: "internal/tui/model_test.go#TestDifftoolLayoutHasNoTree"
        status: pass
      - kind: unit
        ref: "internal/tui/model_test.go#TestDifftoolTabAndAllFilesAreNoOps"
        status: pass
    human_judgment: false
  - id: D7
    description: "The difftool title bar renders all three DIFFTOOL-02 templates exactly, including the equal (1 of 1) and adjacent (7 of 7) counter boundary cases rendered verbatim, never clamped or special-cased."
    requirement: "DIFFTOOL-02"
    verification:
      - kind: unit
        ref: "internal/tui/model_test.go#TestDifftoolTitleBar"
        status: pass
    human_judgment: false
  - id: D8
    description: "The full interactive difftool TUI (git difftool -t alturd invocation, real terminal, live rendering) was not run end-to-end in a real terminal during this session — only the subprocess/unit-test surface was exercised."
    human_judgment: true
    rationale: "No TTY available in this execution environment to launch the interactive bubbletea program; automated coverage (subprocess exit-code/output tests plus unit tests of layout, title bar, and key dispatch) proves every documented behavior, but a human should confirm the live rendering once via git difftool -t alturd during phase UAT."

# Metrics
duration: ~13min
completed: 2026-08-03
status: complete
---

# Phase 04 Plan 03: Theming + Difftool Mode Summary

**50ms-bounded OSC 11 theme auto-detection with `--theme`/config-key override, and a reduced-chrome single-file `--difftool-*` mode with a validated N-of-M title bar, wired end-to-end into `cmd/alturd/main.go` and `internal/tui/model.go`.**

## Performance

- **Duration:** ~13 min
- **Completed:** 2026-08-03
- **Tasks:** 3 (all `type="auto"`, two `tdd="true"`)
- **Files modified:** 7 (2 created new: `internal/config/theme.go`, `internal/config/theme_test.go`; 1 new test file: `cmd/alturd/difftool_test.go`; 4 modified: `internal/config/config.go`, `cmd/alturd/main.go`, `internal/tui/model.go`, `internal/tui/model_test.go`)

## Accomplishments
- Implemented THEME-01's full precedence chain: `--theme` flag → config `theme` key → difftool-mode dark fallback (OSC 11 never queried) → 50ms-bounded auto-detect → dark fallback, as a pure, injectable-detector `ResolveDarkBackground` function
- Bounded `termenv.HasDarkBackground()`'s OSC 11 terminal round-trip to 50ms via an external goroutine race, since `termenv.OSCTimeout` is an unexported 5-second `const` the importing package cannot lower
- Retyped `Config.Theme` from `string` to `config.Theme` and validated it through `ParseTheme` during `Load`, so an invalid `theme` value aborts startup with the exact 04-UI-SPEC.md error string
- Registered `--theme`, `--difftool-local`, `--difftool-remote`, `--difftool-path` on `rootCmd`; difftool mode activates when any `--difftool-*` flag is set and requires all three
- Implemented `difftoolDiff` (`git diff --no-index` in argv form, its own exit-code contract distinct from `internal/git.ExecRunner`'s) and `loadDifftoolFiles` (rewrites `OldName`/`NewName` to the `--difftool-path` basename for correct Chroma lexer selection, validates `GIT_DIFF_PATH_COUNTER`/`TOTAL` as positive integers before they ever reach the terminal, loads post-image lines from `--difftool-remote` without CRLF normalisation per Pitfall F)
- Introduced `tui.DifftoolInfo` and threaded it through `NewModel`'s signature; difftool mode renders a full-width single-file diff pane with no tree pane, no separator column, and `Tab`/`a` short-circuited to no-ops before any tree-viewport code runs
- Implemented `difftoolTitleBar()` covering all three DIFFTOOL-02 templates (with/without counters, with search open), including the equal (`1 of 1`) and adjacent (`7 of 7`) boundary cases rendered verbatim

## Task Commits

Each task was committed atomically:

1. **Task 1: Theme type, 50ms-bounded OSC 11 detection, and precedence resolver (THEME-01, D-05, D-06, D-07)** - `6a85437` (feat)
2. **Task 2: CLI flags, difftool mode dispatch, and theme wiring in main.go (THEME-01, DIFFTOOL-01, DIFFTOOL-02)** - `f0a8cda` (feat)
3. **Task 3: Difftool chrome in the TUI — no tree, full-width pane, title bar templates (DIFFTOOL-01, DIFFTOOL-02)** - `fa07674` (feat)

**Plan metadata:** pending (this commit)

## Files Created/Modified
- `internal/config/theme.go` - `Theme`/`ThemeLight`/`ThemeDark`/`ThemeAuto`, `DetectTimeout` (50ms), `ParseTheme`, `DetectDarkBackground`, `ResolveDarkBackground`
- `internal/config/theme_test.go` - `TestParseTheme`, `TestTheme_Precedence` (detector-call-count assertions), `TestDetectDarkBackground_TimeoutFallback`
- `internal/config/config.go` - `Config.Theme` retyped `string` → `config.Theme`; `Load` validates `raw.Theme` through `ParseTheme`
- `cmd/alturd/main.go` - `--theme`/`--difftool-*` flags; `difftoolDiff`, `loadDifftoolFiles`, `difftoolCounters`, `splitDifftoolFileLines` helpers; theme resolution now routed through `config.ResolveDarkBackground`; `termenv` import removed
- `cmd/alturd/difftool_test.go` - `TestDifftoolModeMissingFlags`, `TestDifftoolModeIdenticalFiles` (subprocess integration, reusing `main_test.go`'s `TestMain`)
- `internal/tui/model.go` - `DifftoolInfo` struct, `NewModel`'s fourth `dt DifftoolInfo` parameter, `difftool` model field, `difftoolTitleBar()`, difftool branches in `View`/`handleResize`/`refreshDiffContent`/`handleKey`
- `internal/tui/model_test.go` - `newModelWith`/existing `NewModel` call sites updated for the 4-arg signature; `TestDifftoolTitleBar`, `TestDifftoolLayoutHasNoTree`, `TestDifftoolTabAndAllFilesAreNoOps` added

## Decisions Made
- **D-05/D-06/D-07 as one pure resolver:** `ResolveDarkBackground(flagTheme, cfgTheme, difftoolMode, detect)` keeps the entire precedence chain in one testable function rather than scattering `if` branches across `main.go`; the detector is injected so tests never touch a real terminal.
- **External 50ms race, not a context/cancel:** `termenv.OSCTimeout` is an unexported `const`, so the only way to impose a lower bound is a `select` between a buffered result channel and `time.After` — the buffer is load-bearing, letting the abandoned goroutine's send complete once termenv's own internal bound expires instead of leaking forever (documented residual risk: FA-04-02, the abandoned goroutine briefly still holds the terminal fd bubbletea is about to claim).
- **`difftoolDiff` does not reuse `internal/git.ExecRunner`:** `ExecRunner`'s error mapping hardcodes a `git diff`-specific "not a git repository" branch and treats exit 128 specially; `git diff --no-index`'s contract is different (exit 1 = normal "files differ" outcome), so a dedicated helper with its own exit-code interpretation was required.
- **DifftoolInfo struct constructed in Task 2, not deferred to Task 3:** Task 2's action explicitly calls the 4-arg `tui.NewModel(...)`, and its own `<verify>` gate (`go build ./...`) cannot pass without the type/field/constructor change existing — this was pulled forward from Task 3's action list as a Rule 3 (blocking-issue) auto-fix; Task 3 then built the actual difftool chrome behavior (resize/view/action-switch branching, title bar, tests) on top of the already-present struct.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Introduced `tui.DifftoolInfo` and the 4-arg `NewModel` signature in Task 2 instead of Task 3**
- **Found during:** Task 2 (CLI flags, difftool mode dispatch, and theme wiring in main.go)
- **Issue:** Task 2's action text explicitly directs constructing `tui.NewModel(files, darkBg, cfg.Keys, tui.DifftoolInfo{...})`, but `tui.DifftoolInfo` and the 4-arg constructor are nominally Task 3's deliverable per Task 3's own action list ("declare an exported `type DifftoolInfo struct`... Extend the constructor..."). As literally sequenced, Task 2 cannot compile — and its own `<verify>` block runs `go build ./...` — without this type existing first.
- **Fix:** Added the minimal `DifftoolInfo` struct, the `difftool DifftoolInfo` model field, and the constructor's fourth parameter as part of Task 2's commit (touching `internal/tui/model.go` and `internal/tui/model_test.go`, which are not listed in Task 2's `<files>` block). Task 3 then added the full difftool chrome behavior (resize/view/action-switch branching, `difftoolTitleBar()`, and its three tests) on top of the already-present struct, without needing to redeclare it.
- **Files modified:** `internal/tui/model.go`, `internal/tui/model_test.go` (in Task 2's commit `f0a8cda`)
- **Verification:** `go build ./...` and `go test ./...` both green after Task 2's commit, satisfying Task 2's own `<verify>` gate; Task 3's `<verify>` gate also passed unchanged.
- **Committed in:** `f0a8cda` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking/task-ordering)
**Impact on plan:** Necessary to keep every task's own commit independently buildable and testable, per this executor's atomic-per-task-commit contract. No functional scope change — the same struct and behavior described across Task 2/3's action text landed in the same two commits either way, just with the type declaration one commit earlier than Task 3's prose implies.

## Issues Encountered
- The `<behavior>` spec for `TestDifftoolLayoutHasNoTree` describes asserting "no box-drawing separator column" in `View()` output. A naive `strings.Contains(view, "│")` check is a false positive: `internal/diff.Render`'s own old/new column separator (`" │ "`) is a legitimate part of every diff row in both standalone and difftool mode — it is unrelated to the *pane*-level tree/diff separator this test is meant to catch. Resolved by asserting the display width (via `lipgloss.Width`) of the first body row equals `diffVP.Width()` (200) exactly, which fails if a tree-pane column were still being concatenated but does not false-positive on the diff renderer's own internal separator.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `internal/tui.NewModel(files, darkBg, keys, dt tui.DifftoolInfo)` is the final signature; `tui.DifftoolInfo{Enabled, Counter, Total, Filename, NewFileLines}` is the final field list — 04-04 (distribution) writes the gitconfig `cmd` string invoking `alturd --difftool-local $LOCAL --difftool-remote $REMOTE --difftool-path $MERGED` and must name these three flags exactly as registered here (`--difftool-local`, `--difftool-remote`, `--difftool-path`).
- `config.ParseTheme`/`ResolveDarkBackground`/`DetectDarkBackground` are stable, tested exports; no further theming work is anticipated for this milestone.
- No blockers. `go build ./...`, `go vet ./...`, and `go test ./...` are all green; zero Phase 1-3 or 04-01/04-02 test regressions.
- Not yet human-verified in a real terminal: the live `git difftool -t alturd` end-to-end rendering (coverage D8). Recommended as a phase-level UAT checkpoint before shipping, since no TTY was available in this execution session.
- Flagged assumptions FA-04-01 (malformed/partial OSC 11 reply inside the 50ms window — unclassified, no code change needed) and FA-04-02 (abandoned detection goroutine briefly contends for the terminal fd bubbletea is about to claim) remain open per the plan frontmatter; both are non-fatal and documented in `internal/config/theme.go`'s doc comments.

---
*Phase: 04-config-theming-difftool-distribution*
*Completed: 2026-08-03*

## Self-Check: PASSED

All created/modified files verified present on disk (`internal/config/theme.go`, `internal/config/theme_test.go`, `cmd/alturd/difftool_test.go`, `internal/config/config.go`, `cmd/alturd/main.go`, `internal/tui/model.go`, `internal/tui/model_test.go`, and this SUMMARY.md); all three task commits (`6a85437`, `f0a8cda`, `fa07674`) verified present in `git log`.
