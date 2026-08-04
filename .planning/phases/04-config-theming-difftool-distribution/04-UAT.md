---
status: testing
phase: 04-config-theming-difftool-distribution
source: [04-VERIFICATION.md]
started: 2026-08-04T20:34:49Z
updated: 2026-08-04T20:34:49Z
---

## Current Test

number: 1
name: Live tag-push GitHub Actions release
expected: |
  Push a real v0.1.0-style tag (matching v*.*.*) to a fresh GitHub repository with GitHub
  Actions enabled and confirm the release workflow runs to completion. A GitHub Release is
  created carrying five binary archives (linux amd64/arm64, darwin amd64/arm64, windows
  amd64) plus checksums.txt, with no other assets, per the backstop truth in 04-02-PLAN.md
  must_haves ("A tag pushed when no prior tag exists still produces a complete GitHub
  Release...").
awaiting: user response

## Tests

### 1. Live tag-push GitHub Actions release
expected: A GitHub Release is created carrying five binary archives (linux amd64/arm64, darwin amd64/arm64, windows amd64) plus checksums.txt, with no other assets.
result: [pending]

### 2. Live interactive git difftool render
expected: |
  In a real interactive terminal, run `git difftool -t alturd <file>` (after `alturd
  install-difftool`) on a repository containing a file whose name is longer than the
  terminal is wide. Confirm: (a) the title bar ends in "…" on a single row rather than
  being cut off mid-word; (b) pressing 'Q' returns the shell prompt cleanly with no
  garbled output and no need for `reset`/`stty sane`; (c) `echo $?` reports 1 after the
  'Q' abort.
result: [pending]

### 3. Judgment-tier prohibitions sign-off
expected: |
  Confirm the five judgment-tier prohibitions (CONFIG-01 no-side-effects, CONFIG-02
  no-silenced-exit, DIST-02 checksum coverage, THEME-01 no-visible-OSC-bytes, DIFFTOOL-01
  read-only file access) against the shipped code. Each prohibition holds with no
  counter-example.
result: [pending]

## Summary

total: 3
passed: 0
issues: 0
pending: 3
skipped: 0
blocked: 0

## Gaps
