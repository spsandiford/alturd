---
status: testing
phase: 04-config-theming-difftool-distribution
source: [04-VERIFICATION.md]
started: 2026-08-06T10:23:44Z
updated: 2026-08-06T10:23:44Z
---

## Current Test

number: 1
name: Live interactive git difftool render (post-G-04-2-fix re-test)
expected: |
  In a real interactive terminal, after `alturd install-difftool`, run `git difftool -t alturd <file>`
  on a repo with an uncommitted change. Confirm the alturd TUI single-file view appears with no
  repeating "git diff --no-index:" flood, no "fork/exec ... resource temporarily unavailable", and
  no "fatal: external diff died" lines. Then re-check the two assertions this test was previously
  blocked from reaching: a long filename's title bar ends in "…" on one row, and pressing 'Q'
  returns the shell prompt cleanly with `echo $?` reporting 1.
awaiting: user response

## Tests

### 1. Live interactive git difftool render (post-G-04-2-fix re-test)
expected: |
  In a real interactive terminal, after `alturd install-difftool`, run `git difftool -t alturd <file>`
  on a repo with an uncommitted change. Confirm the alturd TUI single-file view appears with no
  repeating "git diff --no-index:" flood, no "fork/exec ... resource temporarily unavailable", and
  no "fatal: external diff died" lines. Then re-check the two assertions this test was previously
  blocked from reaching: a long filename's title bar ends in "…" on one row, and pressing 'Q'
  returns the shell prompt cleanly with `echo $?` reporting 1.
result: [pending]

## Summary

total: 1
passed: 0
issues: 0
pending: 1
skipped: 0
blocked: 0

## Gaps
