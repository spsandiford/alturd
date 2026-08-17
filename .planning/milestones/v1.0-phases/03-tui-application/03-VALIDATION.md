---
phase: 3
slug: tui-application
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-07-01
audited: 2026-07-24
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
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~1 second |

> Note: `-race` requires CGO which is unavailable in the build environment. Use `go test ./...` for the full suite.

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/diff/... ./internal/tui/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~1 second

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 1 | DIFF-06 | — | N/A | unit | `go test ./internal/diff/... -run 'TestRender\|TestModel_ModeToggle'` | ✅ | ✅ green |
| 03-01-02 | 01 | 1 | NAV-01 | — | N/A | unit | `go test ./internal/diff/... -run 'TestHunkStartRows\|TestAlign'` | ✅ | ✅ green |
| 03-02-01 | 02 | 1 | TREE-01 | — | N/A | unit | `go test ./internal/tui/... -run 'TestBuildTree\|TestSortNode\|TestFlattenTree\|TestBuildStatusMap'` | ✅ | ✅ green |
| 03-02-02 | 02 | 1 | SEARCH-01 | — | N/A | unit | `go test ./internal/tui/... -run TestFindMatches` | ✅ | ✅ green |
| 03-03-01 | 03 | 2 | NAV-01 | — | N/A | unit | `go test ./internal/tui/... -run TestModel_HunkNav` | ✅ | ✅ green |
| 03-03-02 | 03 | 2 | NAV-02 | — | N/A | unit | `go test ./internal/tui/... -run TestModel_FileCycle` | ✅ | ✅ green |
| 03-03-03 | 03 | 2 | NAV-03 | — | N/A | unit | `go test ./internal/tui/... -run TestModel_FocusToggle` | ✅ | ✅ green |
| 03-03-04 | 03 | 2 | NAV-04 | — | N/A | unit | `go test ./internal/tui/... -run TestModel_Quit` | ✅ | ✅ green |
| 03-03-05 | 03 | 2 | TREE-02 | — | N/A | unit | `go test ./internal/tui/... -run 'TestModel_FocusToggle\|TestModel_TreeExpandCollapse'` | ✅ | ✅ green |
| 03-03-06 | 03 | 2 | DIFF-06 | — | N/A | unit | `go test ./internal/tui/... -run TestModel_ModeToggle` | ✅ | ✅ green |
| 03-03-07 | 03 | 2 | D-07 | — | N/A | unit | `go test ./internal/tui/... -run TestModel_NotReady` | ✅ | ✅ green |
| 03-04-01 | 04 | 3 | SEARCH-01 | — | N/A | unit | `go test ./internal/tui/... -run 'TestModel_SearchDispatch\|TestModel_SearchFindPositions'` | ✅ | ✅ green |
| 03-04-02 | 04 | 3 | TREE-03 | — | N/A | unit | `go test ./internal/tui/... -run TestModel_AllFilesToggle` | ✅ | ✅ green |
| 03-05-01 | 05 | 4 | All Phase 3 | — | N/A | integration | `go build ./... && go vet ./... && go test ./...` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/tui/model_test.go` — model dispatch tests (NAV-01 through NAV-04, SEARCH-01, D-07, DIFF-06, TREE-02, TREE-03)
- [x] `internal/tui/tree_test.go` — tree builder, flattenTree, sortNode, statusMap (TREE-01)
- [x] `internal/tui/search_test.go` — `findMatches()` with ANSI-escaped input (SEARCH-01)
- [x] `internal/diff/render_test.go` — render with mode parameter, all fixture corpus (DIFF-06)
- [x] `internal/diff/align_test.go` — `HunkStartRows` and full alignment tests (NAV-01)

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

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 15s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** 2026-07-24

---

## Validation Audit 2026-07-24

| Metric | Count |
|--------|-------|
| Gaps found | 3 |
| Resolved | 3 |
| Escalated | 0 |

**Gap details:**
- `TestRender_Modes` → corrected to `TestRender` (render_test.go) + `TestModel_ModeToggle` (model_test.go) for DIFF-06
- `TestCollapseChain` → corrected to `TestBuildTree/collapse_chain` subtest; command updated to `TestBuildTree` group for TREE-01
- `TestModel_SearchMode` → corrected to `TestModel_SearchDispatch|TestModel_SearchFindPositions` (model_test.go) for SEARCH-01

All 14 tests across 5 files pass: `go test ./internal/diff/... ./internal/tui/...` → ok (diff 0.14s, tui 0.40s)
