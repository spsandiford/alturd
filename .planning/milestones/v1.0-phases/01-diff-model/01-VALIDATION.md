---
phase: 01
slug: diff-model
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-29
---

# Phase 01 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) |
| **Config file** | none — go test requires no config |
| **Quick run command** | `go test ./internal/diff/ -run TestParse -v` |
| **Full suite command** | `go test ./internal/diff/...` |
| **Estimated runtime** | ~1 second |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/diff/ -run TestParse -v`
- **After every plan wave:** Run `go test ./internal/diff/...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 2 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-01-T1 | 01 | 1 | DIFF-01, DIFF-05 | T-01-SC | Supply-chain: exact versions pinned in go.sum | build | `go build ./... && grep -q 'go-gitdiff v0.8.1' go.mod && grep -q 'chroma/v2 v2.27.0' go.mod` | ✅ | ✅ green |
| 01-01-T2 | 01 | 1 | DIFF-01 | T-01-01 | Fixtures at hardcoded testdata/ paths only, never user input | integration | `test $(ls internal/diff/testdata/*.diff \| wc -l) -ge 13 && ! grep -lUr $'\r' internal/diff/testdata/` | ✅ | ✅ green |
| 01-01-T3 | 01 | 1 | DIFF-01, DIFF-05 | — | N/A — pure type definitions, no IO | unit | `go build ./internal/diff/ && go vet ./internal/diff/ && grep -q 'FullFile RenderMode = iota' internal/diff/model.go` | ✅ | ✅ green |
| 01-02-T1 | 02 | 2 | DIFF-07 | T-01-03 | Malformed diff → wrapped error, never panic | unit | `go test ./internal/diff/ -run 'TestParse\|TestParseMalformed' -v` | ✅ | ✅ green |
| 01-02-T2 | 02 | 2 | DIFF-05, DIFF-07 | T-01-04, T-01-05 | Binary/mode-only/submodule → placeholder rows; nil TextFragments handled | unit | `go test ./internal/diff/ -run TestAlign -v` | ✅ | ✅ green |
| 01-03-T1 | 03 | 3 | DIFF-02 | T-01-08 | Per-line ANSI reset prevents chroma color bleed across split boundary | unit | `go test ./internal/diff/ -run TestHighlight -v` | ✅ | ✅ green |
| 01-03-T2 | 03 | 3 | DIFF-01, DIFF-03, DIFF-04 | T-01-06, T-01-07, T-01-08 | DiffMain guards (1000-char/200-token/100ms) bound CPU; binary bytes bypass text path; ANSI reset at every column boundary | unit | `go test ./internal/diff/ -run TestRender -v && go vet ./internal/diff/` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

All test files (`parse_test.go`, `align_test.go`, `highlight_test.go`, `render_test.go`) were created during plan execution as part of TDD cycles. No Wave 0 bootstrapping needed.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Full-file mode includes ALL unchanged lines from the original file (not just diff-local context); hunk-only omits inter-hunk unchanged content | DIFF-05 (SC4) | Phase 1 `Align()` emits identical output for both modes because diff output is already hunk-local. The behavioral distinction requires full-file content available only in Phase 3 when git show is called. Deferred to Phase 3/DIFF-06. | In Phase 3: `v` hotkey toggles view; full-file row count must exceed hunk-only row count for a multi-hunk file with large unchanged regions. |

---

## Validation Audit 2026-06-29

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated to manual-only | 1 (DIFF-05 SC4 — already deferred in VERIFICATION.md) |

All 7 tasks have `<automated>` verify commands. All 37 tests pass (`go test ./internal/diff/...`). One item (DIFF-05 full-file/hunk-only behavioral distinction) is documented as a known Phase 3 deferral — not a missing test gap.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (none — all tests exist)
- [x] No watch-mode flags
- [x] Feedback latency < 2s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-06-29
