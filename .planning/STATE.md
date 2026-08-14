---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
current_phase: 04.1
current_phase_name: address-tech-debt-phase-3-human-uat-requirements-md-doc-sync
status: executing
stopped_at: Completed 04.1-01-PLAN.md
last_updated: "2026-08-14T19:26:53.534Z"
last_activity: 2026-08-14
last_activity_desc: Phase 04.1 execution started
progress:
  total_phases: 5
  completed_phases: 4
  total_plans: 21
  completed_plans: 19
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-27)

**Core value:** A developer can download a single binary, run `alturd` in any git repository, and navigate every changed file and every individual diff hunk with fast keyboard-driven controls — no runtime dependencies required.
**Current focus:** Phase 04.1 — address-tech-debt-phase-3-human-uat-requirements-md-doc-sync

## Current Position

Phase: 04.1 (address-tech-debt-phase-3-human-uat-requirements-md-doc-sync) — EXECUTING
Plan: 2 of 3
Status: Ready to execute
Last activity: 2026-08-14 — Phase 04.1 execution started

Progress: [████████████████████] 5/5 plans ([█████████░] 90%)

## Performance Metrics

**Velocity:**

- Total plans completed: 18
- Average duration: —
- Total execution time: —

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 3 | - | - |
| 02 | 3 | - | - |
| 03 | 5 | - | - |
| 04 | 7 | - | - |

**Recent Trend:**

- Last 5 plans: —
- Trend: —

*Updated after each plan completion*
| Phase 02 P01 | 128 | 2 tasks | 5 files |
| Phase 02-git-layer-cli P02 | 4m | 2 tasks | 3 files |
| Phase 02 P03 | 3m | 2 tasks | 4 files |
| Phase 03 P04 | 187 | 2 tasks | 2 files |
**Per-Plan Metrics:**

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| Phase 04 P01 | 35min | 3 tasks | 9 files |
| Phase 04 P02 | ~20min | 2 tasks | 16 files |
| Phase 04 P03 | 13min | 3 tasks | 7 files |
| Phase 04 P04 | 25min | 2 tasks | 4 files |
| Phase 04 P05 | 6min | 3 tasks | 4 files |
| Phase 04 P06 | 5min | 2 tasks | 3 files |
| Phase 04 P07 | 5min | 3 tasks | 5 files |
| Phase 04.1 P01 | ~5min active | 3 tasks | 10 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Phase 1: Go module directive set to 1.25 (chroma/v2 v2.27.0 minimum); CLAUDE.md "Go 1.22+" constraint satisfied
- Phase 1: Per-line ANSI reset guard required in Highlight() — splitAndReset() prevents Pitfall 2 color bleed
- Phase 1: intra-line diff uses bold spans (not background color) for Phase 1; Phase 4 can switch
- Phase 1: DiffMain called with checklines=false to prevent inaccurate diffs at line boundaries (Pitfall 3)
- Init: Windows resize polling workaround needed in Phase 3 (bubbletea v2 issue #1601)
- Init: OSC 11 background detection must use 50ms timeout with dark fallback (Phase 4)
- [Phase ?]: Phase 2 Plan 1: NormalizeCRLF exported as package-level helper to enable unit testing without real git subprocess (D-05 compliance)
- [Phase ?]: Phase 2 Plan 1: ExecRunner stateless (no package-level singleton) — callers inject via DI
- [Phase ?]: Phase 2 Plan 2: charmbracelet/log v1.0.0 SetOutput confirmed as SetOutput(w io.Writer)
- [Phase ?]: Phase 2 Plan 2: xdg.Reload used in tests with LIFO cleanup to isolate XDG global state between subtests
- [Phase ?]: Phase 2 Plan 3: TestMain subprocess pattern used for integration tests (t.TempDir lifecycle mismatch prevented sync.Once approach)
- Phase 2: var version declared as var not const so goreleaser -ldflags can override in Phase 4 (D-03)
- Phase 2: git diff exits 129 (not 128) on some git versions outside a repo — stderr message is the only reliable discriminator
- Phase 2: TestMain subprocess pattern for integration tests (t.TempDir lifecycle prevents per-test binary approach)
- [Phase ?]: D-04 locked: flat [keybindings] TOML table, snake_case action names (next_hunk, prev_hunk, next_file, prev_file, toggle_focus, toggle_render_mode, open_search, toggle_all_files, quit, abort)
- [Phase ?]: Keymap.Merge validates strictly two-directionally (D-02): unknown action -> unrecognized key -> merge -> duplicate scan over the merged map (merge-then-validate), producing deterministic canonical-order error messages
- [Phase ?]: Phase 4 Plan 2: golangci-lint v2.7.2 and goreleaser v2.17.1 installed locally to validate CI/release config with real tooling (goreleaser check, snapshot build, ldd) rather than structural checks alone
- [Phase ?]: Phase 4 Plan 2: fixed all 20 pre-existing golangci-lint findings in Phase 1-3/04-01 code (errcheck, revive) so the new CI lint gate lands green on first run
- [Phase ?]: Phase 4 Plan 3: THEME-01 D-05/D-06/D-07 precedence implemented as pure ResolveDarkBackground(flagTheme, cfgTheme, difftoolMode, detect) with injectable detector; 50ms OSC 11 bound imposed externally since termenv.OSCTimeout is an unexported const
- [Phase ?]: Phase 4 Plan 3: difftoolDiff uses git diff --no-index's own exit-code contract (0=identical, 1=differs), distinct from internal/git.ExecRunner's diff-runner-specific error mapping
- [Phase ?]: Phase 4 Plan 3: tui.DifftoolInfo struct and NewModel's 4-arg signature introduced in Task 2 (not Task 3 as plan text implied) so Task 2's own build/verify gate passes — Rule 3 auto-fix
- [Phase ?]: Phase 4 Plan 4: gitConfigRun must prepend the literal "config" git subcommand itself (git config <args>) — plan text/research examples omitted it, which would fail every invocation with git's own usage error (Rule 1 auto-fix)
- [Phase ?]: Phase 4 Plan 4: local-scope-outside-repo stderr detection widened to substring "git repository" (covers git 2.39.5's actual "--local can only be used inside a git repository" wording, not just the RESEARCH.md-assumed "not a git repository") (Rule 1 auto-fix)
- [Phase ?]: Phase 4 Plan 5: ansi.Truncate(title, m.termWidth, "…") replaces lipgloss.Style.MaxWidth for difftool title-bar truncation — MaxWidth hardcodes an empty tail internally and cannot express an ellipsis
- [Phase ?]: Phase 4 Plan 5: abort key routes through tea.Quit + model.aborted bool + exported WasAborted(tea.Model) accessor instead of os.Exit, so bubbletea restores the terminal before cmd/alturd applies exit status 1 via a silent errAborted sentinel (CR-02)
- [Phase ?]: Phase 4 Plan 6: difftoolDiff and diffArgs both carry --no-ext-diff (no cmd.Env rewriting) to close G-04-2's recursive difftool dispatch and its standalone-path analog; TestDifftoolDiffIgnoresExternalDiffConfiguration and TestDiffArgsDisablesExternalDiff pin the regression
- [Phase ?]: Phase 4 Plan 7 (G-04-1 gap closure): difftool.trustExitCode written as explicit false (not unset) so config-scope precedence converges machines already carrying the superseded true value
- [Phase ?]: Phase 4 Plan 7: accepted trade-off — with trustExitCode=false, aborting file N in a multi-file git difftool session no longer stops file N+1 from opening (direction 2, changing alturd's own exit convention, was explicitly rejected)
- Phase 4: UAT test 1 re-verified pass after G-04-1 gap closure (04-07) — git difftool exits 0 on abort with no fatal message, confirmed via scripted pty session
- Phase 4: Security review — 49/49 STRIDE threats closed (32 verified in implementation, 17 accepted risk), threats_open: 0 (04-SECURITY.md)
- [Phase ?]: Phase 04.1 Plan 1: DifftoolInfo.NewFileLines renamed to RefFileLines (WR-07 fix) since it no longer always holds post-image content
- [Phase ?]: Phase 04.1 Plan 1: gofmt swept 5 drifted files (not the 3 named by the milestone audit) so the new .golangci.yml formatters gate lands green on first run
- [Phase ?]: Phase 04.1 Plan 1: cleared 7 pre-existing golangci-lint findings (not named in 04.1-CONTEXT.md) as the enabling condition for the new formatters gate to be a readable CI signal

### Pending Todos

None.

### Blockers/Concerns

- Phase 3 research flag: in-pane search ANSI-aware marker insertion is non-trivial
- go test -race ./... cannot run in this execution sandbox (CGO_ENABLED=0, no gcc/cc, no root) — recorded in .planning/WINDOWS.md; needs re-run in an environment with a C toolchain before Phase 04.1 Plan 1's WR-07 fix is fully verified per the plan's own bar

### Roadmap Evolution

- Phase 04.1 inserted after Phase 4: Address tech debt: Phase 3 human UAT + REQUIREMENTS.md doc-sync (URGENT)

## Session Continuity

Last session: 2026-08-14T19:26:53.521Z
Stopped at: Completed 04.1-01-PLAN.md
Resume file: None
