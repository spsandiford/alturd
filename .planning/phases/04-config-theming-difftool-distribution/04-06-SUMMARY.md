---
phase: 04-config-theming-difftool-distribution
plan: 06
subsystem: cli
tags: [git, difftool, exec, tdd, regression-test]

# Dependency graph
requires:
  - phase: 04-config-theming-difftool-distribution
    provides: "04-03 difftoolDiff/loadDifftoolFiles implementation; 04-05 title-bar truncation and abort-key handling that UAT test 2 also re-checks"
provides:
  - "difftoolDiff immune to GIT_EXTERNAL_DIFF and diff.external dispatch — G-04-2 recursion vector closed at its only reachable call site"
  - "diffArgs helper protecting the standalone alturd [ref] path from the same external-diff hijack class (GIT-01 non-regression)"
  - "TestDifftoolDiffIgnoresExternalDiffConfiguration and TestDiffArgsDisablesExternalDiff regression gates"
affects: [difftool-integration, distribution, phase-04-reverification]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "git diff-computation primitives always carry --no-ext-diff to stay immune to inherited/configured external-diff dispatch"

key-files:
  created:
    - cmd/alturd/difftooldiff_internal_test.go
  modified:
    - cmd/alturd/main.go
    - cmd/alturd/main_internal_test.go

key-decisions:
  - "difftoolDiff's exec.Command argv gets --no-ext-diff inserted between --no-index and --; no Env override, no cmd.Env rewriting (explicitly out of scope per plan) — the flag alone fully suppresses both env-sourced and config-sourced dispatch"
  - "Standalone run()'s inline gitArgs append extracted into a new diffArgs(refArgs) helper carrying the same --no-ext-diff flag, since ExecRunner.Run's documented contract is that the caller composes argv verbatim"
  - "Task 2's RED step is a compile failure (undefined: diffArgs), not a runtime test failure — confirmed via go build before adding the helper, per plan's explicit guidance for a new pure function"

patterns-established:
  - "Doc-comment-with-rationale-and-decision-ID convention extended: difftoolDiff and diffArgs both carry paragraph-length comments explaining --no-ext-diff is load-bearing and citing G-04-2 by name, so a future reader/reviewer cannot mistake the flag for optional hardening"

requirements-completed: [DIFFTOOL-01]

coverage:
  - id: D1
    description: "difftoolDiff(local, remote) computes its own diff under GIT_EXTERNAL_DIFF (env) or diff.external (config) poisoning, never executes the named external program, and preserves the identical/differs 0/1 exit-code contract — closing the G-04-2 recursion vector at its only reachable call site"
    requirement: "DIFFTOOL-01"
    verification:
      - kind: unit
        ref: "cmd/alturd/difftooldiff_internal_test.go#TestDifftoolDiffIgnoresExternalDiffConfiguration"
        status: pass
    human_judgment: true
    rationale: "The regression test proves the mechanism (difftoolDiff no longer dispatches to an external program) via real git subprocesses, but the full end-to-end symptom this closes — git difftool -t alturd <file> launching the alturd TUI instead of flooding the terminal and exhausting the process table — can only be observed in a real interactive terminal running git's actual difftool builtin. That is UAT test 2, deliberately not run by the debug session (to avoid exhausting process/fork resources on a shared sandbox) or by this executor, and is explicitly deferred to the phase's re-verification pass per this plan's <output> instructions."
  - id: D2
    description: "The standalone alturd [ref] path composes its git diff argv through a new diffArgs helper carrying --no-ext-diff, so a developer with diff.external configured for delta/difftastic gets a parseable unified diff instead of a third-party renderer's output fed into diff.Parse"
    requirement: "DIFFTOOL-01"
    verification:
      - kind: unit
        ref: "cmd/alturd/main_internal_test.go#TestDiffArgsDisablesExternalDiff"
        status: pass
    human_judgment: false

duration: 5min
completed: 2026-08-06
status: complete
---

# Phase 4 Plan 6: Difftool Recursive-Diff-Loop Gap Closure Summary

**Both git-diff-computation call sites now carry `--no-ext-diff`, closing G-04-2's unbounded recursive process spawning and hardening the standalone path against the same external-diff hijack class, backed by two new regression tests exercising real git.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-08-06T10:01:46Z
- **Completed:** 2026-08-06T10:05:28Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Closed G-04-2: `difftoolDiff`'s internal `git diff --no-index` call now includes `--no-ext-diff`, so it can never re-enter git's own difftool dispatch (`git-difftool--helper`) no matter what `GIT_EXTERNAL_DIFF` or `diff.external` it inherits — the alturd → git diff → git-difftool--helper → alturd recursion is structurally impossible, not merely bounded.
- Extended the same protection to the standalone `alturd [ref]` path via a new `diffArgs(refArgs)` helper, closing the same root-cause family for developers with `diff.external` pointed at delta/difftastic (GIT-01 non-regression).
- Added `TestDifftoolDiffIgnoresExternalDiffConfiguration` (4 subtests: env-sourced dispatch, config-sourced dispatch, spy-program-never-invoked, identical-files-still-exit-zero) and `TestDiffArgsDisablesExternalDiff` (5 rows: nil/single/range refs, `--`-separated pathspec, non-mutation guard) as permanent regression gates against real git.
- Reproduced the pre-fix failure safely and decomposed (see below) — the actual catastrophic fork-bomb was never run, matching the debug session's own risk posture.

## Task Commits

Each task was committed atomically (TDD RED→GREEN):

1. **Task 1: Make difftoolDiff's internal diff immune to external-diff dispatch (closes G-04-2, DIFFTOOL-01)**
   - RED: `535f626` (test) — failing regression test added
   - GREEN: `3e653e8` (fix) — `--no-ext-diff` added to `difftoolDiff`'s argv
2. **Task 2: Disable external-diff dispatch on the standalone git diff argv too (GIT-01 non-regression)**
   - RED+GREEN: `2c8794d` (feat) — test (compile-failure RED, per plan's explicit guidance for a new pure function) and `diffArgs` helper landed together since a standalone compile-failure commit would break the build at that commit

**Plan metadata:** (this commit, immediately following)

## Files Created/Modified
- `cmd/alturd/difftooldiff_internal_test.go` - New file; `TestDifftoolDiffIgnoresExternalDiffConfiguration`, the G-04-2 regression gate (4 subtests against real git with a poisoned environment and a poisoned gitconfig)
- `cmd/alturd/main.go` - `difftoolDiff`'s `exec.Command` argv gains `--no-ext-diff`; new `diffArgs(refArgs []string) []string` helper; `run()`'s standalone branch now calls `diffArgs` instead of an inline `append`; both call sites' doc comments extended with G-04-2 rationale
- `cmd/alturd/main_internal_test.go` - `TestDiffArgsDisablesExternalDiff` added alongside the existing `TestReportError`

## Decisions Made
- `--no-ext-diff` inserted between the existing `--no-index` and the `--` separator in `difftoolDiff`'s argv — no other change to that function's exit-0/exit-1 branches, stdout/stderr buffers, or error wrapping (all already correct, now covered by the fourth subtest).
- No `cmd.Env` rewriting anywhere: the flag alone suppresses both the environment-sourced (`GIT_EXTERNAL_DIFF`) and configuration-sourced (`diff.external`) dispatch vectors, avoiding the risk of dropping an environment variable git actually needs (explicitly out of scope per plan).
- `diffArgs` placed immediately after `run()` and before `errAborted`, matching the plan's specified location; `internal/git/runner.go` and `internal/git/args.go` left untouched, preserving `ExecRunner`'s "caller composes argv" contract.

## Deviations from Plan

None - plan executed exactly as written.

## Pre-fix Failure Reproduction (recorded per plan's <output> instructions)

Running `go test ./cmd/alturd/ -count=1 -run TestDifftoolDiffIgnoresExternalDiffConfiguration -v` against the unmodified source produced, verbatim:

```
=== RUN   TestDifftoolDiffIgnoresExternalDiffConfiguration/external_diff_env_var
    difftooldiff_internal_test.go:75: difftoolDiff() error = git diff --no-index: fatal: cannot run /tmp/.../no-such-external-diff: No such file or directory
        fatal: external diff died, stopping at /tmp/.../local.go, want nil
=== RUN   TestDifftoolDiffIgnoresExternalDiffConfiguration/diff_external_config
    difftooldiff_internal_test.go:90: difftoolDiff() error = git diff --no-index: fatal: cannot run /tmp/.../no-such-external-diff-2: No such file or directory
        fatal: external diff died, stopping at /tmp/.../local.go, want nil
=== RUN   TestDifftoolDiffIgnoresExternalDiffConfiguration/external_diff_program_never_invoked
    difftooldiff_internal_test.go:113: diff output missing "@@"; got:
    difftooldiff_internal_test.go:113: diff output missing "-func old() {}"; got:
    difftooldiff_internal_test.go:113: diff output missing "+func new() {}"; got:
    difftooldiff_internal_test.go:116: marker file exists — spy.sh was invoked, want it never invoked (stat err = <nil>)
--- FAIL: TestDifftoolDiffIgnoresExternalDiffConfiguration/external_diff_env_var (0.01s)
--- FAIL: TestDifftoolDiffIgnoresExternalDiffConfiguration/diff_external_config (0.01s)
--- FAIL: TestDifftoolDiffIgnoresExternalDiffConfiguration/external_diff_program_never_invoked (0.02s)
--- PASS: TestDifftoolDiffIgnoresExternalDiffConfiguration/identical_files_still_exit_zero (0.00s)
```

This is the safe, decomposed reproduction of the fork-bomb: git's own `fatal: external diff died` message fires because the spy/nonexistent program was actually dispatched to, and (in the third subtest) the marker file existing on disk directly proves the spy program ran. The fourth subtest already passed pre-fix, confirming the identical-files exit-0 path was never the problem.

## Final argv of both git invocations (recorded per plan's <output> instructions)

- `difftoolDiff`: `git diff --no-index --no-ext-diff -- <local> <remote>`
- Standalone `run()` (via `diffArgs`): `git diff --no-ext-diff [refArgs...]` (e.g. `git diff --no-ext-diff HEAD`, `git diff --no-ext-diff -- internal/tui`)

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- G-04-2 is closed at the code level with automated regression coverage; `go build ./...`, `go vet ./...`, and `go test ./... -count=1` are all green across every package.
- **Deferred to phase re-verification (UAT test 2, 04-UAT.md):** a real interactive terminal must still confirm `alturd install-difftool` + `git difftool -t alturd <file>` launches the alturd TUI cleanly — no repeating `git diff --no-index:` output, no `fork/exec ... resource temporarily unavailable`, no `fatal: external diff died` — and that the two assertions UAT test 2 was previously blocked from reaching (DIFFTOOL-02 ellipsis truncation, NAV-04/D-08 `Q` exit code 1) now pass. This is the only verification step this plan's automated tooling could not perform, matching the debug session's own decision not to run the catastrophic full reproduction.
- No other blockers. Phase 04 has all 6 plans executed; ready for phase-level re-verification pass.

---
*Phase: 04-config-theming-difftool-distribution*
*Completed: 2026-08-06*

## Self-Check: PASSED

All claimed files exist on disk (cmd/alturd/difftooldiff_internal_test.go, cmd/alturd/main.go, cmd/alturd/main_internal_test.go, .planning/phases/04-config-theming-difftool-distribution/04-06-SUMMARY.md). All claimed commits (535f626, 3e653e8, 2c8794d, 1590d07) found in git log.
