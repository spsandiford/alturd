---
phase: 04-config-theming-difftool-distribution
plan: 07
subsystem: infra
tags: [git, difftool, gitconfig, gap-closure]

# Dependency graph
requires:
  - phase: 04-config-theming-difftool-distribution
    provides: install-difftool subcommand (04-04), abort/quit exit-code convention (04-06)
provides:
  - difftool.trustExitCode written as false (was true) by install-difftool, closing G-04-1
  - end-to-end regression test reproducing the exact `git difftool` fatal-die symptom
  - stale-machine convergence test for the trust key
  - corrected exit-code rationale comments in cmd/alturd/main.go and internal/tui/model.go
  - corrected 04-UAT.md test 1 expectation (achievable exit-0 target, not exit-1 passthrough)
affects: [04-distribution, ship-readiness]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "gitconfig keys with cross-scope precedence must be written as explicit values, never removed, to converge machines already carrying a superseded value"

key-files:
  created: []
  modified:
    - cmd/alturd/difftool.go
    - cmd/alturd/installdifftool_test.go
    - cmd/alturd/main.go
    - internal/tui/model.go
    - .planning/phases/04-config-theming-difftool-distribution/04-UAT.md

key-decisions:
  - "difftool.trustExitCode written as false, not unset — config-scope precedence (system/global/local) means removing the key would still lose to a lower-precedence scope holding the superseded true"
  - "Accepted trade-off (user-locked, 2026-08-06): in a multi-file `git difftool` session with no pathspec, aborting on file N no longer prevents file N+1 from opening, since git's diff engine cannot distinguish an intentional cancel from a crash on any nonzero exit"
  - "Direction 2 (changing alturd's own abort exit-code convention) explicitly rejected by the user — only the gitconfig value changed; abort control flow and exit codes are untouched"

patterns-established: []

requirements-completed: [DIFFTOOL-01, DIFFTOOL-03]

coverage:
  - id: D1
    description: "install-difftool writes difftool.trustExitCode=false; a fresh machine and a machine carrying the superseded true value both converge to false"
    requirement: "DIFFTOOL-03"
    verification:
      - kind: unit
        ref: "cmd/alturd/installdifftool_test.go#TestInstallDifftoolWritesFourKeys"
        status: pass
      - kind: integration
        ref: "cmd/alturd/installdifftool_test.go#TestInstallDifftoolRewritesStaleTrustExitCode"
        status: pass
    human_judgment: false
  - id: D2
    description: "A difftool backend that exits non-zero (alturd's abort convention) under `git difftool -t alturd` now returns exit status 0 with no fatal external-diff line; a sensitivity control confirms the fatal path reappears when the superseded trust value is restored"
    requirement: "DIFFTOOL-01"
    verification:
      - kind: e2e
        ref: "cmd/alturd/installdifftool_test.go#TestGitDifftoolExitsCleanlyWhenBackendAborts"
        status: pass
    human_judgment: false
  - id: D3
    description: "04-UAT.md test 1 expectation corrected to the achievable target (git exit 0, no fatal line) while the ## Gaps block stays byte-identical for reconciliation"
    verification:
      - kind: other
        ref: "grep -c 'reporting 1' 04-UAT.md == 2 (only the two surviving Gaps-block mentions)"
        status: pass
    human_judgment: false

duration: ~5min
completed: 2026-08-06
status: complete
---

# Phase 04 Plan 07: Close UAT gap G-04-1 (difftool trustExitCode fatal) Summary

**`install-difftool` now writes `difftool.trustExitCode=false` instead of `true`, closing G-04-1: a `git difftool -t alturd` session with an aborting backend now exits 0 with no fatal external-diff line, proven by an end-to-end regression test with a sensitivity control.**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-08-06T19:04:00Z
- **Completed:** 2026-08-06T19:08:23Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments
- `runInstallDifftool` (cmd/alturd/difftool.go) writes `difftool.trustExitCode=false`, correcting the D-08 design intent that assumed a trusted non-zero exit could signal "user cancelled" to git — it cannot, git's external-diff protocol treats every non-zero exit as an unconditional fatal crash.
- Two new automated tests in `cmd/alturd/installdifftool_test.go`: `TestInstallDifftoolRewritesStaleTrustExitCode` (a machine already carrying the superseded `true` converges on re-run) and `TestGitDifftoolExitsCleanlyWhenBackendAborts` (an end-to-end `git difftool` invocation against a stub backend that exits 1, with a sensitivity control that seeds the stale value back and confirms the fatal path reappears).
- `errAborted`'s doc comment (cmd/alturd/main.go) and `WasAborted`'s doc comment (internal/tui/model.go) no longer claim a trusted-exit-code contract with git; both now explain alturd's exit-1-on-abort convention on its own merits for non-git callers and cite G-04-1.
- 04-UAT.md test 1's `expected` block now asks for the achievable target (git exit 0, no fatal line) instead of the unreachable exit-status-1 passthrough; the `## Gaps` block, `result`/`reported` blocks, summary counters and front-matter status are untouched.

## Task Commits

Each task was committed atomically:

1. **Task 1: Pin the difftool trust setting off and prove a non-zero backend exit is no longer fatal** - `0f29617` (fix)
2. **Task 2: Correct the exit-code rationale comments without touching abort behaviour** - `0aa7ef8` (docs)
3. **Task 3: Correct 04-UAT.md test 1 to the achievable expectation** - `3030007` (docs)

**Plan metadata:** (this commit, pending)

## Files Created/Modified
- `cmd/alturd/difftool.go` - `difftool.trustExitCode` written as `"false"` (was `"true"`); header, `runInstallDifftool` doc comment, and the D-08 block above the four `gitConfigSet` calls all carry the corrected G-04-1 rationale (git's two-outcome exit-status model, why an explicit value beats unsetting the key, and the accepted multi-file abort trade-off)
- `cmd/alturd/installdifftool_test.go` - flipped `TestInstallDifftoolWritesFourKeys`'s trust-key assertion to `"false"`; added `TestInstallDifftoolRewritesStaleTrustExitCode` and `TestGitDifftoolExitsCleanlyWhenBackendAborts`
- `cmd/alturd/main.go` - `errAborted`'s doc comment rewritten: exit-1 stands on its own for non-git callers, cites G-04-1, no longer claims a trusted-exit-code contract
- `internal/tui/model.go` - `WasAborted`'s doc comment rewritten identically in substance; the surrounding explanation of the locally-declared method-interface boundary crossing is preserved unchanged
- `.planning/phases/04-config-theming-difftool-distribution/04-UAT.md` - test 1's `expected` block final clause rewritten to the achievable exit-0/no-fatal-line target, with one sentence explaining why, citing G-04-1 and the debug doc

## Decisions Made
- `difftool.trustExitCode` is written as the explicit string `"false"`, not unset, because git resolves this key across the system/global/local precedence chain (verified empirically against installed git 2.39.5): unsetting at the install scope would leave a lower-precedence scope's superseded `true` value in control, and the fatal would return. Writing an explicit value at the install scope overrides everything below it and converges any machine that already ran the previous version of `install-difftool` (D-10's "the command converges the config").
- Accepted trade-off (user-locked decision, recorded in 04-UAT.md's Gaps block and now also in code comments): in a multi-file `git difftool` session with no pathspec, pressing the abort key on file N no longer stops file N+1 from opening — git walks every remaining changed file, since turning `trustExitCode` off means `git-difftool--helper` always falls through to `exit 0` regardless of the backend's exit code.
- Direction 2 from the debug session (changing alturd's own quit-key exit-code convention instead of the gitconfig value) was explicitly rejected by the user. This plan only changes the gitconfig value and code comments; `errAborted`'s value, `Aborted()`/`WasAborted()` signatures and bodies, the abort keybinding, and all control flow are unchanged.

## Deviations from Plan

None - plan executed exactly as written. Task 1's tracer-task feedback gate (re-running the `<verify>` end-to-end) passed on the first attempt with all 9 install-difftool/GitDifftool tests green, so no Rule 1-4 fixes were needed before proceeding to Tasks 2-3.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- G-04-1 is closed: 04-UAT.md test 1 is now re-testable against an achievable expectation, and the full test suite (`go test ./... -count=1`) plus `go vet ./...` are green.
- A manual spot check (`install-difftool --scope global` against an isolated gitconfig) independently confirmed `git config --get difftool.trustExitCode` resolves to `false` after install, matching the automated tests.
- No blockers for the remaining phase 04 distribution work.

---
*Phase: 04-config-theming-difftool-distribution*
*Completed: 2026-08-06*

## Self-Check: PASSED

All 6 files confirmed present on disk (cmd/alturd/difftool.go, cmd/alturd/installdifftool_test.go, cmd/alturd/main.go, internal/tui/model.go, 04-UAT.md, 04-07-SUMMARY.md). All 3 task commit hashes (0f29617, 0aa7ef8, 3030007) confirmed present in `git log --oneline --all`.
