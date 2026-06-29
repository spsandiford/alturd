---
phase: 02
slug: git-layer-cli
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-29
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
| **Full suite command** | `go test -race ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test -race ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | GIT-01 | — | N/A | unit | `go test ./internal/git/...` | ❌ W0 | ⬜ pending |
| 02-01-02 | 01 | 1 | GIT-02 | — | N/A | unit | `go test ./internal/git/...` | ❌ W0 | ⬜ pending |
| 02-01-03 | 01 | 1 | GIT-03 | — | N/A | unit | `go test ./internal/git/...` | ❌ W0 | ⬜ pending |
| 02-01-04 | 01 | 1 | GIT-04 | — | N/A | unit | `go test ./internal/git/...` | ❌ W0 | ⬜ pending |
| 02-02-01 | 02 | 1 | GIT-05 | — | N/A | unit | `go test ./cmd/alturd/...` | ❌ W0 | ⬜ pending |
| 02-02-02 | 02 | 1 | LOG-01 | — | log written only to file, never stderr | unit | `go test ./internal/log/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/git/runner_test.go` — stubs for GIT-01 through GIT-04 (Runner interface, arg capture, CRLF normalization, exit code mapping)
- [ ] `cmd/alturd/main_test.go` — stubs for GIT-05 (invocation forms, error exits)
- [ ] `internal/log/log_test.go` — stubs for LOG-01 (file path, truncation at 1 MB)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `--version` prints no log file | GIT-01 | Requires real filesystem check post-run | Run `alturd --version` and verify no `.log` file created in XDG_STATE_HOME |
| `--help` produces no side effects | GIT-01 | Requires real filesystem check post-run | Run `alturd --help` and verify no `.log` file created |
| Log truncated to 1 MB on startup | LOG-01 | Requires real filesystem + pre-seeded large log | Seed a log >1 MB, run `alturd`, check truncated size ≤ 1 MB |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
