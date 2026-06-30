---
status: complete
phase: 02-git-layer-cli
source: [02-VERIFICATION.md]
started: "2026-06-29T00:00:00Z"
updated: "2026-06-30T00:00:00Z"
---

## Current Test

[testing complete]

## Tests

### 1. Exit code and message — outside a git repo
expected: |
  Build the binary first: `go build -o /tmp/alturd ./cmd/alturd`
  Then run from a non-git directory:
    cd /tmp && XDG_STATE_HOME=/tmp/alturd-test ./alturd
  Expected: exits 1, single line to stderr: "not a git repository (run alturd inside a git working tree)"
  Note: log file WILL exist. Just verify exit code 1 and the error message.
result: pass

### 2. Exit code and message — git not on PATH
expected: |
  Run with git removed from PATH:
    cd /tmp && XDG_STATE_HOME=/tmp/alturd-test PATH=/usr/bin/env ./alturd
  Expected: exits 127, single line to stderr: "git: command not found (is git installed and on PATH?)"
result: pass

## Summary

total: 2
passed: 2
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps
