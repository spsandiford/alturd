---
status: diagnosed
phase: 04-config-theming-difftool-distribution
source: [04-VERIFICATION.md]
started: 2026-08-04T20:34:49Z
updated: 2026-08-05T00:15:00Z
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
  root_cause: "difftoolDiff() in cmd/alturd/main.go runs `exec.Command(\"git\", \"diff\", \"--no-index\", \"--\", local, remote)` without `--no-ext-diff` and without overriding Env. When git's own `git difftool` builtin invokes alturd as `difftool.<name>.cmd`, it unconditionally sets GIT_EXTERNAL_DIFF=git-difftool--helper in that child's environment. difftoolDiff's exec.Command inherits the full parent environment, so its internal `git diff --no-index` call also honors GIT_EXTERNAL_DIFF and delegates to git-difftool--helper instead of computing its own diff — which re-dispatches to difftool.alturd.cmd = alturd again with fresh $LOCAL/$REMOTE/$MERGED, which calls difftoolDiff again, recursing unboundedly until fork()/exec() starts failing. Confirmed empirically: (a) GIT_EXTERNAL_DIFF is present in a difftool-cmd child's env even in a fresh scratch repo with only the four canonical install-difftool keys set — not a pre-existing user config; (b) `git diff --no-index` demonstrably delegates to GIT_EXTERNAL_DIFF when set; (c) adding --no-ext-diff suppresses the recursion and preserves the existing 0/1 exit-code contract."
  artifacts:
    - path: "cmd/alturd/main.go"
      issue: "difftoolDiff()'s `exec.Command(\"git\", \"diff\", \"--no-index\", \"--\", local, remote)` (~line 272) omits `--no-ext-diff`, so it recurses into git's own difftool dispatch via the inherited GIT_EXTERNAL_DIFF env var."
  missing:
    - "Add `--no-ext-diff` to the git diff --no-index argv in difftoolDiff so the internal diff computation never re-enters git's external-diff/difftool machinery."
  debug_session: ".planning/debug/difftool-recursive-diff-loop.md"
