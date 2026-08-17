# Phase 1: Diff Model - Context

**Gathered:** 2026-06-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Build `internal/diff` — a pure Go library that parses git diff output and produces aligned side-by-side ANSI output with syntax highlighting and intra-line change markers. No TUI code. No bubbletea. No terminal I/O. The library must pass table-driven Go tests against the Python fixture corpus before Phase 2 begins.

</domain>

<decisions>
## Implementation Decisions

### Fixture Corpus

- **D-01:** Copy the Python implementation's raw `.diff` fixture files into `internal/diff/testdata/` in this Go repo. No runtime dependency on the Python repo path.
- **D-02:** Fixture files live at `internal/diff/testdata/` (Go convention for `testdata/` — placed next to the package being tested, automatically available to `go test`).

### Renderer Output Contract

- **D-03:** The renderer produces `[]string` — one fully-composed ANSI string per rendered row (left column + right column joined). Phase 3 feeds this slice directly into a bubbletea viewport with no further transformation.
- **D-04:** The render function accepts a `width int` parameter: `Render(diff, width int) []string`. Tests pass a fixed width (e.g., 160). Phase 3 passes the actual terminal width. No globals or package-level defaults.

### Test Oracle Approach

- **D-05:** Tests use structural assertions only — no golden ANSI snapshot files. At minimum, each table test asserts:
  - Files parsed correctly (correct FileDiff count, filename, status marker)
  - Added/removed/unchanged line counts match expected values
  - Syntax highlighting applied for languages Chroma can detect (presence of ANSI color codes on relevant lines)
  - Intra-line character-level markers present on modified lines (when guards permit)
  - Edge-case files (binary patches, pure renames, mode-only changes, submodule bumps, no-newline-at-EOF) render the correct placeholder or diff content without panic
  - Guard thresholds: tests that exceed the 1000-char / 200-token guards exercise the degraded path and verify no intra-line markers are emitted

### Intra-line Diff Mode

- **D-06:** Default granularity is character-level — `go-diff DiffMain(old, new, checklines=false)`. Matches Python implementation behavior. Most precise.
- **D-07:** When any guard triggers (line > 1000 chars, token count > 200, or elapsed time > 100ms), skip intra-line entirely and show line-level diff color only. No fallback to word-level, no visual indicator. Graceful degradation.

### Locked Library Choices (from CLAUDE.md — do not re-litigate)

- **go-gitdiff v0.8.1** for diff parsing (handles binary patches, renames, no-newline markers — all present in fixture corpus)
- **go-diff v1.4.0** for intra-line character diff (`DiffMain`)
- **chroma/v2 v2.27.0** for syntax highlighting (same language database as Pygments)
- **CGO_ENABLED=0** — no CGO anywhere

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & Scope
- `.planning/REQUIREMENTS.md` §Diff Model — DIFF-01 through DIFF-07; guard thresholds (1000-char / 200-token / 100ms); full-file vs. hunk-only behavior spec
- `.planning/ROADMAP.md` §Phase 1 — Success criteria, ANSI reset requirement, fixture corpus expectations

### Library Choices & Rationale
- `.claude/CLAUDE.md` §Technology Stack — Full library selection table, alternatives considered, what NOT to use; all Phase 1 library choices are locked here

### Python Reference Implementation
- The Python implementation at v1.1 is the authoritative behavioral reference. Its fixture corpus (12+ scenarios) is the test oracle source. Fixture files must be copied from `tests/fixtures/diff/` in the Python repo into `internal/diff/testdata/` in this Go repo.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- None yet — this is a greenfield Go project. No source files exist.

### Established Patterns
- Go module layout not yet created; `internal/diff` package path is established by the ROADMAP success criteria (`go test ./internal/diff/...`)
- `testdata/` convention: Go test runner automatically makes `testdata/` available relative to the package; fixture files at `internal/diff/testdata/*.diff` are readable with `os.ReadFile("testdata/foo.diff")` inside tests

### Integration Points
- Phase 2 (Git Layer) will invoke `internal/diff.Parse()` on output from `git diff` subprocess
- Phase 3 (TUI) will call `internal/diff.Render(parsed, terminalWidth)` and feed the returned `[]string` into a bubbletea viewport
- Width parameter design must anticipate that Phase 3 will re-call Render when the terminal is resized

</code_context>

<specifics>
## Specific Ideas

No specific "I want it like X" references — open to standard Go patterns within the decisions above.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 1-Diff Model*
*Context gathered: 2026-06-26*
