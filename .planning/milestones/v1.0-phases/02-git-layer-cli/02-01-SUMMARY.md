---
phase: 02-git-layer-cli
plan: "01"
subsystem: internal/git
tags: [git, subprocess, runner, error-mapping, CRLF, ref-grammar, TDD]
dependency_graph:
  requires: []
  provides: [git.Runner, git.ExecRunner, git.ExitCodeError, git.ErrGitNotFound, git.ErrNotGitRepo, git.ParseRefArgs, git.NormalizeCRLF]
  affects: [cmd/alturd/main.go, internal/git/runner_test.go, internal/git/args_test.go]
tech_stack:
  added: []
  patterns: [Runner interface DI, ExitCodeError sentinel, table-driven TDD, argv form exec.Command]
key_files:
  created:
    - internal/git/errors.go
    - internal/git/runner.go
    - internal/git/args.go
    - internal/git/runner_test.go
    - internal/git/args_test.go
  modified: []
decisions:
  - "NormalizeCRLF exported as a package-level helper so tests can verify CRLF normalization logic directly without spawning a subprocess"
  - "ExecRunner is stateless (no package-level singleton) — callers inject it via dependency injection per PATTERNS.md Anti-Patterns"
  - "ErrGitNotFound maps to Code 127 (shell convention for command not found); ErrNotGitRepo maps to Code 1 per D-11"
  - "Runner.Run args slice does not include the 'git' program name — only the subcommand and its args"
metrics:
  duration: "2m"
  completed: "2026-06-29"
  tasks_completed: 2
  tasks_total: 2
  files_created: 5
  files_modified: 0
status: complete
---

# Phase 02 Plan 01: Git Layer - Runner + Args Summary

**One-liner:** `internal/git` package with Runner DI interface, ExitCodeError sentinels (127/1), ExecRunner with argv-form subprocess and CRLF normalization, and ParseRefArgs for all six invocation forms.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 (RED) | Failing tests for Runner, ExitCodeError, CRLF | 3574345 | internal/git/runner_test.go |
| 1 (GREEN) | ExitCodeError + ExecRunner implementation | 4a43bb9 | internal/git/errors.go, internal/git/runner.go |
| 2 (RED) | Failing tests for ParseRefArgs (7 cases) | afb7464 | internal/git/args_test.go |
| 2 (GREEN) | ParseRefArgs implementation | 399eb2b | internal/git/args.go |

## What Was Built

### Package `github.com/alturd/alturd/internal/git`

**`errors.go`** — Sentinel error type and two package-level vars:
- `type ExitCodeError struct { Code int; Msg string }` — carries exit code + user-facing message
- `Error() string` returns `e.Msg` unchanged (main() prints it directly to stderr)
- `ErrGitNotFound` — Code 127, triggered when "git" binary not on PATH
- `ErrNotGitRepo` — Code 1, triggered when git exits 128 (not in a git repo)

**`runner.go`** — Subprocess boundary:
- `type Runner interface { Run(args []string) (io.Reader, error) }` — DI interface
- `type ExecRunner struct{}` — production implementation; stateless
- `ExecRunner.Run` uses `exec.Command("git", args...)` — argv form, no shell (ASVS V5 / T-02-01)
- Exit code mapping: `exec.ErrNotFound` → `ErrGitNotFound`; `ExitCode() == 128` → `ErrNotGitRepo`
- CRLF→LF normalization applied immediately after `cmd.Output()` (D-09)
- `NormalizeCRLF([]byte) []byte` — exported helper for direct test coverage

**`args.go`** — Ref-grammar parser:
- `ParseRefArgs(args []string, dashIdx int) []string` — pure function, no side effects
- Re-inserts `"--"` separator between refs and paths (cobra strips it — Pitfall 2)
- Verbatim pass-through of ref tokens; git validates ref syntax
- Covers all six invocation forms: no-args, single-ref, two-dot, three-dot, two-args, paths-only, ref+paths

## TDD Gate Compliance

- RED commit `3574345` precedes GREEN commit `4a43bb9` for Task 1
- RED commit `afb7464` precedes GREEN commit `399eb2b` for Task 2
- All RED tests confirmed failing before implementation; all GREEN tests pass after

## Test Results

```
go test ./internal/git/... -v
--- PASS: TestParseRefArgs (0.00s)   [7 subtests]
--- PASS: TestRunnerExitCodes (0.00s) [2 subtests]
--- PASS: TestExitCodeErrorMessage (0.00s)
--- PASS: TestFakeRunnerCapturesArgs (0.00s)
--- PASS: TestCRLFNormalization (0.00s)
ok  github.com/alturd/alturd/internal/git  0.003s
```

`go vet ./internal/git/...` — clean.
`grep "sh -c" internal/git/runner.go` — no matches (shell injection surface absent).

## Deviations from Plan

### Auto-added: NormalizeCRLF exported helper (Rule 2 — missing test coverage path)

The plan specified CRLF normalization applied inside `ExecRunner.Run` after `cmd.Output()`. Testing this path without a real git subprocess (required by D-05) needed either:
1. Integration test spawning real git, or
2. Exported normalization helper testable in isolation.

Chose option 2: exported `NormalizeCRLF([]byte) []byte` as a package-level function. This satisfies D-05 (no real git subprocess in unit tests) while providing direct verification of the normalization logic. The function is small, pure, and fully documented. The plan's intent — verified CRLF normalization — is met.

## Threat Surface Scan

No new network endpoints, auth paths, or file access patterns beyond what the plan's threat model anticipated.

| Threat | Status |
|--------|--------|
| T-02-01: argv form subprocess invocation | Mitigated — `exec.Command("git", args...)` confirmed, no `sh -c` |
| T-02-02: path args passed verbatim after `--` separator | Mitigated — ParseRefArgs inserts `"--"` before path args; no interpolation |
| T-02-03: error messages are fixed sentinel strings | Mitigated — ErrGitNotFound/ErrNotGitRepo are literal constants; no raw git stderr |

## Self-Check: PASSED

Files exist:
- internal/git/errors.go — FOUND
- internal/git/runner.go — FOUND
- internal/git/args.go — FOUND
- internal/git/runner_test.go — FOUND
- internal/git/args_test.go — FOUND

Commits exist:
- 3574345 — FOUND
- 4a43bb9 — FOUND
- afb7464 — FOUND
- 399eb2b — FOUND
