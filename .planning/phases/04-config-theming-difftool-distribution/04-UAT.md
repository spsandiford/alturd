---
status: complete
phase: 04-config-theming-difftool-distribution
source: [04-VERIFICATION.md]
started: 2026-08-04T20:34:49Z
updated: 2026-08-05T00:05:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Live tag-push GitHub Actions release
expected: A GitHub Release is created carrying five binary archives (linux amd64/arm64, darwin amd64/arm64, windows amd64) plus checksums.txt, with no other assets.
result: pass

### 2. Live interactive git difftool render
expected: |
  In a real interactive terminal, run `git difftool -t alturd <file>` (after `alturd
  install-difftool`) on a repository containing a file whose name is longer than the
  terminal is wide. Confirm: (a) the title bar ends in "…" on a single row rather than
  being cut off mid-word; (b) pressing 'Q' returns the shell prompt cleanly with no
  garbled output and no need for `reset`/`stty sane`; (c) `echo $?` reports 1 after the
  'Q' abort.
result: issue
reported: |
  Ran `git difftool -t alturd README.md` from ~/src/jig. Instead of launching the alturd
  TUI, the terminal was flooded with an unbounded, repeating chain of
  "git diff --no-index: git diff --no-index: ..." (thousands of repetitions on one line),
  eventually hitting "fork/exec /usr/lib/git-core/git: resource temporarily unavailable"
  once the process/fork limit was exhausted, followed by hundreds of
  "fatal: external diff died, stopping at /tmp/git-blob-.../README.md" lines. The alturd
  TUI never appeared. This looks like the configured difftool command is recursively
  invoking `git diff --no-index` (or itself) rather than launching the alturd binary,
  causing runaway process spawning until the OS fork limit was hit.
severity: blocker

### 3. Judgment-tier prohibitions sign-off
expected: |
  Confirm the five judgment-tier prohibitions (CONFIG-01 no-side-effects, CONFIG-02
  no-silenced-exit, DIST-02 checksum coverage, THEME-01 no-visible-OSC-bytes, DIFFTOOL-01
  read-only file access) against the shipped code. Each prohibition holds with no
  counter-example.
result: pass

## Summary

total: 3
passed: 2
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- gap_id: G-04-2
  truth: "Running `git difftool -t alturd <file>` in a real interactive terminal launches the alturd TUI difftool render."
  status: failed
  reason: "User reported: git difftool floods the terminal with an unbounded repeating chain of \"git diff --no-index: git diff --no-index: ...\" until fork/exec fails with \"resource temporarily unavailable\", followed by hundreds of \"fatal: external diff died\" lines. The alturd TUI never launches — looks like the configured difftool command recursively invokes `git diff --no-index` (or itself) instead of the alturd binary, causing runaway process spawning until the OS fork limit is hit."
  severity: blocker
  test: 2
  artifacts: []
  missing: []
