---
phase: "01"
plan: "01"
subsystem: internal/diff
status: complete
tags: [go-module, fixtures, type-model, diff-parsing]
completed: "2026-06-26"
duration: "~10 minutes"
tasks_completed: 3
tasks_total: 3

dependency_graph:
  requires: []
  provides:
    - go.mod with go-gitdiff/go-diff/chroma/v2 at locked versions
    - internal/diff/testdata/*.diff (13 scenario fixtures)
    - internal/diff/model.go (LineKind, RenderMode, RenderedLine, RowPair)
  affects:
    - 01-02-PLAN.md (parse.go, align.go depend on model.go and fixtures)
    - 01-03-PLAN.md (highlight.go, render.go depend on model.go)

tech_stack:
  added:
    - go 1.25 (module go directive — chroma/v2 v2.27.0 requires 1.25+, originally planned as 1.22)
    - github.com/bluekeyes/go-gitdiff v0.8.1
    - github.com/sergi/go-diff v1.4.0
    - github.com/alecthomas/chroma/v2 v2.27.0
    - github.com/dlclark/regexp2/v2 v2.2.1 (transitive dep of chroma/v2)
  patterns:
    - iota constants for typed enums (LineKind, RenderMode)
    - RenderedLine.ANSI populated in stages by Highlight and Render
    - RowPair.Left/Right model the two columns in side-by-side view
    - testdata/ Go convention for test fixtures (auto-available to go test)
    - .gitattributes eol=lf to prevent CRLF on Windows checkout (Pitfall 7)

key_files:
  created:
    - go.mod
    - go.sum
    - .gitattributes
    - internal/diff/model.go
    - internal/diff/testdata/simple.diff
    - internal/diff/testdata/binary.diff
    - internal/diff/testdata/rename.diff
    - internal/diff/testdata/mode-only.diff
    - internal/diff/testdata/submodule.diff
    - internal/diff/testdata/no-newline.diff
    - internal/diff/testdata/new-file.diff
    - internal/diff/testdata/deleted-file.diff
    - internal/diff/testdata/multi-file.diff
    - internal/diff/testdata/multi-hunk.diff
    - internal/diff/testdata/large-line.diff
    - internal/diff/testdata/many-tokens.diff
    - internal/diff/testdata/multiline-string.diff
  modified: []

decisions:
  - "Go module directive set to 1.25 (not 1.22) because chroma/v2 v2.27.0 requires go >= 1.25 in its own go.mod — the CLAUDE.md constraint is Go 1.22+, satisfied by 1.25"
  - "go.mod includes github.com/dlclark/regexp2/v2 v2.2.1 as a transitive dependency of chroma/v2; no action needed"
  - "go mod download used instead of go mod tidy for go.sum generation since no source files exist yet at Task 1 time (tidy removes unused deps)"
  - "FullFile=iota 0 confirmed as default zero value per DIFF-05 requirement"
  - "Fixtures hand-crafted per RESEARCH Open Question 1 (Python repo not available in this environment)"
---

# Phase 01 Plan 01: Go Module Foundation and Type Model Summary

**One-liner:** Go module initialized with go-gitdiff v0.8.1/go-diff v1.4.0/chroma v2.27.0, 13 LF-normalized fixture files covering all diff edge cases, and the RowPair/RenderedLine/LineKind/RenderMode type model.

## What Was Built

Plan 01-01 established the Wave 1 foundation that all other Phase 1 plans depend on:

**Task 1: Go Module Initialization** (commit a46fb1d)
- `go.mod` with module path `github.com/alturd/alturd` and three Phase 1 libraries at CLAUDE.md-locked versions
- `go.sum` with cryptographic checksums for all dependencies
- `.gitattributes` enforcing LF line endings on `internal/diff/testdata/*.diff`
- Note: go directive updated to 1.25 (chroma v2.27.0 minimum requirement; satisfies CLAUDE.md "Go 1.22+" constraint)

**Task 2: 13-Scenario Fixture Corpus** (commit 25ac4ee)
All 13 fixture files created in `internal/diff/testdata/`, each exercising one parse/render edge case per D-01/D-02:

| Fixture | Scenario | Key Assertion |
|---------|----------|---------------|
| simple.diff | Basic text change | NewName=README.md, context+delete+add lines |
| binary.diff | Binary file change | IsBinary=true |
| rename.diff | Pure rename 100% similarity | IsRename=true, no content |
| mode-only.diff | chmod+x, no text | OldMode!=NewMode, TextFragments empty |
| submodule.diff | Submodule bump | mode 160000, Subproject commit lines |
| no-newline.diff | Missing EOF newline | Line.NoEOL()=true on last line |
| new-file.diff | Added file | IsNew=true, all OpAdd lines |
| deleted-file.diff | Deleted file | IsDelete=true, all OpDelete lines |
| multi-file.diff | Two files in one diff | gitdiff.Parse returns 2 File structs |
| multi-hunk.diff | Two @@ hunks | Tests FullFile vs HunkOnly distinction (DIFF-05) |
| large-line.diff | 1100+ char changed line | Exercises 1000-char intra-line guard (D-07) |
| many-tokens.diff | 210-token changed line | Exercises 200-token guard (D-07) |
| multiline-string.diff | Go backtick multiline | Tests chroma per-line split color bleed (Pitfall 2) |

**Task 3: Core Type Model** (commit ea7df04)
`internal/diff/model.go` defines the shared type vocabulary:
- `LineKind` with 6 constants in iota order: Context, Added, Removed, ModifiedOld, ModifiedNew, Blank
- `RenderMode` with FullFile=0 (zero-value default) and HunkOnly=1
- `RenderedLine` with Kind, Content, ANSI fields (ANSI set in stages by Highlight/Render)
- `RowPair` with Left, Right RenderedLine (the DIFF-01 side-by-side column structure)
- No imports — pure type definitions

## Verification Results

All plan success criteria confirmed:
- `go version go1.25.11` >= 1.22 ✓
- `go build ./...` exits 0 ✓
- `go vet ./internal/diff/` exits 0 ✓
- `go.sum` contains checksums for go-gitdiff, go-diff, chroma/v2 ✓
- No Phase 3/4 deps (bubbletea/lipgloss/bubbles/termenv/go-toml/xdg) in go.mod ✓
- 13 LF-only fixtures under `internal/diff/testdata/` ✓
- `model.go` exports LineKind, RenderMode, RenderedLine, RowPair ✓

## Commits

| Task | Commit | Type | Description |
|------|--------|------|-------------|
| Task 1 | a46fb1d | chore | Initialize Go module with locked diff libraries |
| Task 2 | 25ac4ee | test | Create 13-scenario fixture corpus (D-01, D-02) |
| Task 3 | ea7df04 | feat | Define core type model for pipeline |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical Functionality] Go directive set to 1.25 instead of 1.22**
- **Found during:** Task 1 — `go get github.com/alecthomas/chroma/v2@v2.27.0`
- **Issue:** chroma/v2 v2.27.0 declares `go 1.25` minimum in its own go.mod. When `go get` runs, it automatically upgrades the module's go directive from 1.22 to 1.25 to satisfy the transitive requirement.
- **Fix:** Accepted the upgrade to `go 1.25`. The CLAUDE.md constraint is "Go 1.22+" which is satisfied. No functional difference for Phase 1 code; all target platforms (Linux/macOS/Windows) support Go 1.25 binaries.
- **Files modified:** go.mod
- **Impact:** Plan 02 and 03 will also use Go 1.25

**2. [Rule 3 - Blocking Issue] Go not installed on the execution host**
- **Found during:** Task 1 — `go version` returned command not found
- **Issue:** Go was not present in the system PATH.
- **Fix:** Downloaded Go 1.22.10 Linux binary from go.dev/dl and extracted to scratchpad. The `go get chroma` step then downloaded the Go 1.25 toolchain module. Both toolchains used from scratchpad only; not installed system-wide.
- **Files modified:** None (environment only)

**3. [Rule 3 - Blocking Issue] `go mod tidy` removes deps with no source files**
- **Found during:** Task 1 post-`go get`
- **Issue:** Running `go mod tidy` without any Go source files removes all declared dependencies since nothing imports them.
- **Fix:** Used `go mod download` instead to populate go.sum checksums without pruning unused dependencies. The `require` directives persist in go.mod because they were added via `go get`.
- **Files modified:** None (workflow adjustment)

## Known Stubs

None — this plan creates foundational types and fixtures, not application logic.

## Threat Flags

No new security-relevant surfaces introduced beyond what was in the threat model.

## Self-Check: PASSED

All 17 output files exist on disk. All 3 task commits verified in git history.

| Check | Result |
|-------|--------|
| go.mod | FOUND |
| go.sum | FOUND |
| .gitattributes | FOUND |
| internal/diff/model.go | FOUND |
| 13 fixture files | FOUND |
| Commit a46fb1d (Task 1) | FOUND |
| Commit 25ac4ee (Task 2) | FOUND |
| Commit ea7df04 (Task 3) | FOUND |
