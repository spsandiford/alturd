---
plan: 03-02
phase: 03-tui-application
status: complete
completed: 2026-07-20
tasks_total: 2
tasks_completed: 2
key-files:
  created:
    - internal/tui/tree.go
    - internal/tui/tree_test.go
    - internal/tui/search.go
    - internal/tui/search_test.go
  modified:
    - go.mod
    - go.sum
commits:
  - 71bad9a
  - 2871e84
---

# Plan 03-02: Tree + Search Pure Functions — Summary

## What Was Built

Created `internal/tui/tree.go` and `internal/tui/search.go` — pure functions with no bubbletea dependency.

### tree.go

- `TreeNode` + `flatRow` types
- `buildTree(paths, statusMap)` — dirs-first sorted hierarchy with GitHub-style single-child chain collapsing
- `insertPath`, `collapseChain`, `sortNode`, `flattenTree` — tree manipulation
- `buildStatusMap(files)` — bridges `diff.FileStatus` into the tree
- `filePaths(files)` — returns display paths in order

**Key fix:** `collapseChain` guards `node.Name != ""` to skip the invisible root sentinel — the reference implementation in PATTERNS.md would collapse the root into its single dir child, producing a node named "/a/b/c" instead of the expected "a/b/c".

### search.go

- `findMatches(content, query)` — ANSI-strips content first, then scans with `strings.Index`, returning non-overlapping `[start, end]` positions in the stripped coordinate space. This aligns with `viewport.SetHighlights` expectations (RESEARCH Pitfall 5).

### go.mod

`charmbracelet/x/ansi` promoted from `// indirect` to a direct dependency.

## Acceptance Criteria Verification

- `buildTree` / `collapseChain` / `flattenTree` all implemented ✓
- collapse_chain subtest: root child Name is "a/b/c" ✓
- dirs_before_files subtest: first root child IsDir=true ✓
- `go test ./internal/tui/... -run TestBuildTree` passes ✓
- `findMatches` implemented with `ansi.Strip` ✓
- `charmbracelet/x/ansi` not marked indirect in go.mod ✓
- `go test ./internal/tui/... -run TestFindMatches` — all 5 subtests pass ✓
- `go build ./...` stays green ✓

## Deviations

`collapseChain` root guard (node.Name != "") — PATTERNS.md reference omits this, causing the root to collapse incorrectly. Fix applied, rationale documented in commit.

## Self-Check: PASSED
