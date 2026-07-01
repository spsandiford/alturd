---
phase: 3
slug: tui-application
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-01
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` (standard library) |
| **Config file** | none — existing `go test` conventions |
| **Quick run command** | `go test ./internal/diff/... ./internal/tui/...` |
| **Full suite command** | `go test -race ./...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/diff/... ./internal/tui/...`
- **After every plan wave:** Run `go test -race ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 03-??-01 | W0 | 1 | DIFF-06 | — | N/A | unit | `go test ./internal/diff/... -run TestRender_Modes` | ❌ Wave 0 | ⬜ pending |
| 03-??-02 | W0 | 1 | NAV-01 | — | N/A | unit | `go test ./internal/diff/... -run TestHunkStartRows` | ❌ Wave 0 | ⬜ pending |
| 03-??-03 | W0 | 1 | NAV-01 | — | N/A | unit | `go test ./internal/tui/... -run TestModel_HunkNav` | ❌ Wave 0 | ⬜ pending |
| 03-??-04 | W0 | 1 | NAV-02 | — | N/A | unit | `go test ./internal/tui/... -run TestModel_FileCycle` | ❌ Wave 0 | ⬜ pending |
| 03-??-05 | W0 | 1 | NAV-03 | — | N/A | unit | `go test ./internal/tui/... -run TestModel_FocusToggle` | ❌ Wave 0 | ⬜ pending |
| 03-??-04 | W0 | 1 | NAV-04 | — | N/A | unit | `go test ./internal/tui/... -run TestModel_Quit` | ❌ Wave 0 | ⬜ pending |
| 03-??-06 | W0 | 1 | TREE-01 | — | N/A | unit | `go test ./internal/tui/... -run TestBuildTree` | ❌ Wave 0 | ⬜ pending |
| 03-??-07 | W0 | 1 | TREE-01 | — | N/A | unit | `go test ./internal/tui/... -run TestCollapseChain` | ❌ Wave 0 | ⬜ pending |
| 03-??-08 | W0 | 1 | TREE-02 | — | N/A | unit | `go test ./internal/tui/... -run TestModel_FocusToggle` | ❌ Wave 0 | ⬜ pending |
| 03-??-09 | W0 | 1 | TREE-03 | — | N/A | unit | `go test ./internal/tui/... -run TestModel_AllFilesToggle` | ❌ Wave 0 | ⬜ pending |
| 03-??-10 | W0 | 1 | SEARCH-01 | — | N/A | unit | `go test ./internal/tui/... -run TestFindMatches` | ❌ Wave 0 | ⬜ pending |
| 03-??-11 | W0 | 1 | SEARCH-01 | — | N/A | unit | `go test ./internal/tui/... -run TestModel_SearchMode` | ❌ Wave 0 | ⬜ pending |
| 03-??-12 | W0 | 1 | D-07 | — | N/A | unit | `go test ./internal/tui/... -run TestModel_NotReady` | ❌ Wave 0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/tui/model_test.go` — model dispatch tests (NAV-01 through NAV-04, SEARCH-01, D-07)
- [ ] `internal/tui/tree_test.go` — tree builder, collapseChain, flattenTree (TREE-01 through TREE-03)
- [ ] `internal/tui/search_test.go` — `findMatches()` with ANSI-escaped input (SEARCH-01)
- [ ] `internal/diff/render_test.go` — update existing callers for new `mode` parameter (DIFF-06)
- [ ] `internal/diff/align_test.go` — add `HunkStartRows` tests (NAV-01)

*Framework install: none — `go test` is built-in.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Full terminal rendering visual quality | All TUI reqs | Requires real TTY | Run `alturd` in a git repo with 3+ changed files; verify panes, colors, and layout visually |
| Windows resize polling behavior | D-08 | Requires Windows Terminal + SIGWINCH absent | Run `alturd` on Windows Terminal; resize window; verify diff pane reflows |
| Performance under large diff (>10k lines) | NAV-01 | Requires real large diff | Run `alturd` on a large repo; verify hunk nav is responsive |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
