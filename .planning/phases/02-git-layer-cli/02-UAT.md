---
status: testing
phase: 02-git-layer-cli
source: [02-VERIFICATION.md]
started: "2026-06-29T00:00:00Z"
updated: "2026-06-29T00:00:00Z"
---

## Current Test

number: 1
name: Exit codes for non-repo and no-git scenarios
expected: |
  Running alturd outside a git repo exits 1 with single-line error to stderr.
  Running alturd with git not on PATH exits 127 with single-line error to stderr.
awaiting: user response

## Tests

### 1. Exit code and message — outside a git repo
expected: |
  Build the binary first: `go build -o /tmp/alturd ./cmd/alturd`
  Then run from a non-git directory:
    cd /tmp && XDG_STATE_HOME=/tmp/alturd-test ./alturd
  Expected: exits 1, single line to stderr: "not a git repository (run alturd inside a git working tree)"
  Expected: no log file created at /tmp/alturd-test/alturd/alturd.log (since RunE returns early)
  Note: log IS created because RunE runs before the git call — Init() is called first, then git fails.
  So log file WILL exist. Just verify exit code 1 and the error message.
result: [pending]

### 2. Exit code and message — git not on PATH
expected: |
  Run with git removed from PATH:
    cd /tmp && XDG_STATE_HOME=/tmp/alturd-test PATH=/usr/bin/env ./alturd
    (or set PATH to a dir without git)
  Expected: exits 127, single line to stderr: "git: command not found (is git installed and on PATH?)"
result: [pending]

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps
