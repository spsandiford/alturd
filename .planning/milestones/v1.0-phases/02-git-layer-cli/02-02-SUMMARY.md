---
phase: 02-git-layer-cli
plan: "02"
subsystem: internal/log
tags: [logging, xdg, applog, charmbracelet-log]
dependency_graph:
  requires: []
  provides: [applog.Init, applog.truncateLog, applog.maxLogSize]
  affects: [cmd/alturd (Plan 03 caller)]
tech_stack:
  added:
    - github.com/adrg/xdg v0.5.3
    - github.com/charmbracelet/log v1.0.0
  patterns:
    - XDG state file resolution via xdg.StateFile
    - Tail-retaining log truncation (read/slice/write, not os.Truncate)
    - charmbracelet/log.SetOutput for file redirection
    - White-box tests with t.Setenv + xdg.Reload for isolated XDG state
key_files:
  created:
    - internal/log/log.go
    - internal/log/log_test.go
  modified:
    - go.mod
    - go.sum
decisions:
  - "charmbracelet/log v1.0.0 SetOutput API confirmed as func SetOutput(w io.Writer) — same as pattern"
  - "xdg.Reload() used in tests to refresh package globals after t.Setenv; registered via LIFO cleanup so env var is restored before final Reload"
  - "os.Truncate not used; keep-tail algorithm uses ReadFile+WriteFile to retain recent entries"
metrics:
  duration: "~4 minutes"
  completed: "2026-06-29T20:44:33Z"
  tasks_completed: 2
  files_created: 3
  files_modified: 2
status: complete
requirements_addressed: [LOG-01]
---

# Phase 02 Plan 02: applog.Init with XDG Path and Tail-Truncation Summary

**One-liner:** XDG-based log file init with 1 MB tail-truncation cap, redirecting charmbracelet/log to file (never stderr), implemented as package applog with full white-box test coverage.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add xdg + charmbracelet/log deps and implement applog.Init | 6ea478a | go.mod, go.sum, internal/log/log.go |
| 2 | Truncation-threshold and XDG-path tests (TDD) | 02171d4 | internal/log/log_test.go |

## What Was Built

**Package `internal/log` (identifier `applog`)** implements the XDG-based log initialization for LOG-01:

- `applog.Init() (*os.File, error)` — resolves `$XDG_STATE_HOME/alturd/alturd.log` via `xdg.StateFile`, tail-truncates if >1 MB, opens with mode 0600, redirects `charmbracelet/log` output to the file, returns the open `*os.File` for the caller to defer-close.
- `applog.truncateLog(path string) error` — reads the file, slices the trailing `maxLogSize` bytes, writes back with mode 0600. Does NOT use `os.Truncate` (which would retain the head/oldest entries).
- `const maxLogSize = 1 << 20` — 1 MB threshold.

**Threat mitigations implemented:**
- T-02-05: File mode 0600 (owner-only), preventing other local users from reading log content.
- T-02-06: 1 MB tail-truncation cap enforced on every Init (startup), preventing unbounded log growth.

**Tests (`internal/log/log_test.go`, package `applog`):**
- `TestInit/path_under_xdg_state_home` — verifies Init returns a non-nil `*os.File` whose path is `tmpDir/alturd/alturd.log`; file exists on disk after Init.
- `TestInit/truncates_over_cap` — pre-populates file with maxLogSize+10000 bytes including a tail marker, calls Init, asserts final size ≤ maxLogSize AND tail marker is still at end (proving tail retention, not head).
- `TestInit/small_file_untouched` — pre-populates small file, calls Init, asserts original content is preserved at file head (O_APPEND mode).

All tests use `t.Setenv("XDG_STATE_HOME", t.TempDir())` + `xdg.Reload()` with LIFO cleanup to isolate real user state dir.

## Verification Results

```
go test ./internal/log/... -run TestInit -v   → PASS (3/3 subtests)
go build ./internal/log/...                    → exit 0
go vet ./internal/log/...                      → exit 0
go mod tidy                                    → no diff
grep os\.Truncate\( internal/log/log.go        → no call (only in comment)
```

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written.

### Informational Notes

1. **charmbracelet/log API verification (Open Question 2 resolved):** Ran `go doc github.com/charmbracelet/log SetOutput` after installing v1.0.0 — API is `func SetOutput(w io.Writer)`, identical to the pattern in PATTERNS.md. No code change needed.

2. **TDD task ordering:** Task 2 has `tdd="true"` but its implementation (Task 1) executes first per the plan's sequential ordering. Since the implementation was complete before tests were written, there was no RED phase (tests passed immediately on first run). This matches the plan's intent — the `tdd="true"` flag indicates test-focused quality standards, not strict red-before-green ordering within this plan.

3. **xdg.Reload() cleanup pattern:** The plan noted that xdg reads XDG_STATE_HOME at init and may cache it. Added `xdg.Reload()` call in tests with LIFO-ordered `t.Cleanup` registration to ensure correct env restoration after each subtest. This is documented as a decision since it was not explicitly specified in PATTERNS.md.

## Known Stubs

None — package is complete and fully wired to its dependencies.

## Threat Flags

None — no new security-relevant surface beyond what the plan's threat model covers (T-02-04, T-02-05, T-02-06 all addressed).

## Self-Check: PASSED

- `internal/log/log.go` — FOUND
- `internal/log/log_test.go` — FOUND
- `go.mod` contains `github.com/adrg/xdg v0.5.3` — FOUND
- `go.mod` contains `github.com/charmbracelet/log v1.0.0` — FOUND
- Commit `6ea478a` — FOUND (feat(02-02): add applog.Init...)
- Commit `02171d4` — FOUND (test(02-02): add white-box tests...)
- All tests pass — VERIFIED (`go test ./internal/log/...` exit 0)
