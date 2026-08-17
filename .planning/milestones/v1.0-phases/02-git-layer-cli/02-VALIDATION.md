---
phase: 02
slug: git-layer-cli
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-29
audited: 2026-07-01
---

# Phase 02 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) |
| **Config file** | none — standard Go test runner |
| **Quick run command** | `go test ./...` |
| **Full suite command** | `go test ./...` (CGO disabled; -race unavailable in this env) |
| **Estimated runtime** | ~1 second |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|------|--------|
| 02-01-01 | 01 | 1 | GIT-01 | T-02-01 | argv form, no shell injection | unit | `go test ./internal/git/... -run TestRunner\|TestExitCode\|TestCRLF\|TestFake` | internal/git/runner_test.go | ✅ green |
| 02-01-02 | 01 | 1 | GIT-02 | — | N/A | unit | `go test ./internal/git/... -run TestParseRefArgs` | internal/git/args_test.go | ✅ green |
| 02-01-03 | 01 | 1 | GIT-03 | — | N/A | unit | `go test ./internal/git/... -run TestParseRefArgs` | internal/git/args_test.go | ✅ green |
| 02-01-04 | 03 | 2 | GIT-04 | T-02-08 | no log file on --version/--help | integration | `go test ./cmd/alturd/... -run TestVersion\|TestHelp` | cmd/alturd/main_test.go | ✅ green |
| 02-02-01 | 03 | 2 | GIT-05 | T-02-09 | exit 127/1 on error, single-line stderr | integration | `go test ./cmd/alturd/... -run TestExitCode\|TestSmoke` | cmd/alturd/main_test.go | ✅ green |
| 02-02-02 | 02 | 1 | LOG-01 | T-02-05 | log written only to file, never stderr | unit | `go test ./internal/log/... -run TestInit` | internal/log/log_test.go | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/git/runner_test.go` — TestRunnerExitCodes, TestExitCodeErrorMessage, TestFakeRunnerCapturesArgs, TestCRLFNormalization
- [x] `internal/git/args_test.go` — TestParseRefArgs (7 invocation-form subtests)
- [x] `internal/log/log_test.go` — TestInit (path, truncation, small-file-untouched)
- [x] `cmd/alturd/main_test.go` — TestVersionExitsZeroNoLog, TestHelpExitsZeroNoLog, TestSmokeRunInRepoExitsZero, TestExitCodeNotGitRepo, TestExitCodeGitNotFound

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Log truncated to 1 MB on startup in real install | LOG-01 | Unit test uses XDG override; real install path depends on user's XDG_STATE_HOME | Seed `$XDG_STATE_HOME/alturd/alturd.log` >1 MB, run `alturd`, check truncated size ≤ 1 MB |

---

## Validation Audit 2026-07-01

| Metric | Count |
|--------|-------|
| Gaps found | 1 |
| Resolved | 1 |
| Escalated | 0 |

Gap resolved: GIT-05 error-path exit code routing — added `TestExitCodeNotGitRepo` (exit 1) and `TestExitCodeGitNotFound` (exit 127) as subprocess integration tests in `cmd/alturd/main_test.go`.

---

## Validation Sign-Off

- [x] All tasks have automated verify
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all requirements
- [x] No watch-mode flags
- [x] Feedback latency < 5s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** 2026-07-01 — `go test ./...` green (4 packages, 14 tests)
