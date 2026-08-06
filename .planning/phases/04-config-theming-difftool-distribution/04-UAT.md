---
status: complete
phase: 04-config-theming-difftool-distribution
source: [04-VERIFICATION.md]
started: 2026-08-06T10:23:44Z
updated: 2026-08-06T18:26:18Z
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
  artifacts: []
  missing: []
