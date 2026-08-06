---
status: diagnosed
phase: 04-config-theming-difftool-distribution
source: [04-VERIFICATION.md]
started: 2026-08-06T10:23:44Z
updated: 2026-08-06T18:41:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Live interactive git difftool render (post-G-04-2-fix re-test)
expected: |
  In a real interactive terminal, after `alturd install-difftool`, run `git difftool -t alturd <file>`
  on a repo with an uncommitted change. Confirm the alturd TUI single-file view appears with no
  repeating "git diff --no-index:" flood, no "fork/exec ... resource temporarily unavailable", and
  no "fatal: external diff died" lines. Then re-check the two assertions this test was previously
  blocked from reaching: a long filename's title bar ends in "…" on one row, and pressing 'Q'
  returns the shell prompt cleanly with `echo $?` reporting 1.
result: issue
reported: |
  Reproduced via scripted pty session against a real `git difftool -t alturd -- <file>` run on an
  uncommitted change (both a long-filename `--no-index` case and a genuine tracked-file change).
  Partial pass: no repeating "git diff --no-index:" flood, no fork/exec resource-unavailable error,
  and the long filename's title bar does end in "…" on one row. FAILS the other two assertions:
  pressing 'Q' does not return cleanly — git prints "fatal: external diff died, stopping at <file>"
  and the difftool session exits 128, not a clean 1. Root cause: `alturd install-difftool` writes
  `[difftool] prompt = false` and `trustExitCode = true` into git config. With trustExitCode=true,
  git-difftool--helper propagates alturd's exit code (1, alturd's signal for "user cancelled") back
  to git's core diff engine, which treats any nonzero exit from the external diff command as a
  fatal crash (finish_command() -> die("external diff died, stopping at %s")) regardless of whether
  the tool exited cleanly by design. This happens on every `git difftool -t alturd` invocation where
  the user presses Q, not just the --no-index case.
severity: major

## Summary

total: 1
passed: 0
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- gap_id: G-04-1
  truth: "Pressing 'Q' in the alturd difftool view returns the shell prompt cleanly with `echo $?` reporting 1, and no 'fatal: external diff died' line is printed."
  status: failed
  reason: "Reproduced via scripted pty test: `git difftool -t alturd -- <file>` then 'Q' prints 'fatal: external diff died, stopping at <file>' and exits 128. alturd itself exits cleanly with code 1 (confirmed by direct invocation and by a logging wrapper around the difftool.alturd.cmd), but `alturd install-difftool` sets `difftool.trustExitCode = true` locally, which causes git's diff engine to treat any nonzero exit from the external diff tool as fatal, regardless of exit code value."
  severity: major
  test: 1
  root_cause: |
    cmd/alturd/difftool.go:116 (runInstallDifftool) unconditionally writes difftool.trustExitCode=true
    to gitconfig. Combined with alturd's intentional exit-code-1-on-abort convention
    (cmd/alturd/main.go:211 errAborted; internal/tui/model.go:92,582 Aborted()/WasAborted(),
    triggered by the default 'Q' keybinding = config.ActionAbort), this makes git's core diff
    engine fatally die on every abort. Traced end-to-end against installed git 2.39.5 source:
    difftool.trustExitCode=true -> builtin/difftool.c's cmd_difftool() sets
    GIT_DIFFTOOL_TRUST_EXIT_CODE=true unconditionally on the `git diff` subprocess env ->
    inherited by git-difftool--helper (invoked per changed file as GIT_EXTERNAL_DIFF by diff.c)
    -> the helper's per-file loop propagates alturd's exit 1 as its own exit status (without this
    var it always falls through to `exit 0`, masking the tool's exit code) -> diff.c's
    run_external_diff()/finish_command() treats ANY nonzero exit from GIT_EXTERNAL_DIFF as an
    unconditional fatal crash -- there is no git-side mechanism to distinguish "intentional
    cancel" from "tool crashed" -- so it calls die("external diff died, stopping at %s"), and
    git difftool reports 128. This is a structural git limitation, not a simple code typo:
    difftool.trustExitCode=true and exit-1-on-abort can never coexist without triggering die().
    Two non-equivalent fix directions: (1) stop writing difftool.trustExitCode=true (delivers
    the UAT's literal single-file clean-exit expectation, but silently disables abort-driven
    early-stop across multi-file `git difftool` sessions -- pressing Q on file N would no longer
    prevent file N+1 from opening); or (2) keep trustExitCode=true for its multi-file
    abort-stopping value but change alturd's quit convention so an intentional abort doesn't
    reach git as a nonzero exit -- which resolves the fatal message but means alturd's own
    process exit code no longer distinguishes abort from normal quit for non-git callers.
  artifacts:
    - path: "cmd/alturd/difftool.go"
      issue: "line 116 (runInstallDifftool) unconditionally sets difftool.trustExitCode=true, which is incompatible with alturd's exit-1-on-abort convention under git's diff-engine error model"
    - path: "cmd/alturd/main.go"
      issue: "line 211: errAborted exit-code-1 convention (intentional, not itself buggy -- see root_cause)"
    - path: "internal/tui/model.go"
      issue: "lines 92, 582: Aborted()/WasAborted() -- 'Q' triggers config.ActionAbort, which surfaces as alturd's own exit 1"
  missing:
    - "Decide and implement one of the two fix directions in root_cause (drop/change trustExitCode, or change alturd's abort exit-code convention), since both cannot be satisfied simultaneously due to git's GIT_EXTERNAL_DIFF protocol"
  debug_session: .planning/debug/DEBUG-difftool-trustexitcode-fatal.md
