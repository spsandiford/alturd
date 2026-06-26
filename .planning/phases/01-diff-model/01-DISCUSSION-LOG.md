# Phase 1: Diff Model - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-26
**Phase:** 1-Diff Model
**Areas discussed:** Fixture corpus strategy, Renderer output contract, Test oracle approach, Intra-line diff mode

---

## Fixture Corpus Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Copy into this repo | Copy the 12+ raw .diff files from the Python repo into testdata/fixtures/ in the Go project | ✓ |
| Git submodule / symlink to Python repo | Reference the Python repo's tests/fixtures/diff/ directory from the Go project | |
| Port scenarios, write new .diff files | Recreate the fixture scenarios as freshly written .diff files in the Go project | |

**User's choice:** Copy into this repo

**Follow-up — fixture location:**

| Option | Description | Selected |
|--------|-------------|----------|
| internal/diff/testdata/ | Fixtures live next to the package they test — Go convention | ✓ |
| testdata/ at project root | Top-level, accessible to multiple packages | |
| You decide | Let Claude and planner pick | |

**User's choice:** internal/diff/testdata/
**Notes:** None — straightforward preference for idiomatic Go layout.

---

## Renderer Output Contract

| Option | Description | Selected |
|--------|-------------|----------|
| ANSI strings per line ([]string) | Fully-composed ANSI string per rendered row; Phase 3 feeds directly to viewport | ✓ |
| Structured []RenderedLine{Left, Right string} | Separate left/right column strings; Phase 3 composes at display time | |
| [][]Token (semantic tokens, no ANSI) | Semantic data; ANSI applied by Phase 3 via Lipgloss | |

**User's choice:** ANSI strings per line ([]string)

**Follow-up — width handling:**

| Option | Description | Selected |
|--------|-------------|----------|
| Width parameter Render(diff, width int) | Tests pass fixed width; Phase 3 passes terminal width | ✓ |
| Default width constant (e.g., 160) | Package-level default, Phase 3 can override via setter | |
| You decide | Let planner determine width API | |

**User's choice:** Width parameter
**Notes:** None — clean contract, no globals.

---

## Test Oracle Approach

| Option | Description | Selected |
|--------|-------------|----------|
| Structural assertions only | Assert on parsed properties: line counts, syntax tokens, intra-line markers, edge-case placeholders | ✓ |
| Golden ANSI files | Snapshot full ANSI output; fail on any change | |
| Both: structural + golden | Structural for correctness, golden for regression | |

**User's choice:** Structural assertions only

**Follow-up — minimum assertion coverage:**

| Option | Description | Selected |
|--------|-------------|----------|
| Parser + line-level correctness | Parser correctness, line counts, syntax highlighting, intra-line markers, edge cases, guard behavior | ✓ |
| Parser correctness only | Only assert go-gitdiff struct output | |
| Full visual parity with Python | Byte-for-byte identical to Python implementation output | |

**User's choice:** Parser + line-level correctness
**Notes:** None.

---

## Intra-line Diff Mode

| Option | Description | Selected |
|--------|-------------|----------|
| Character-level (checklines=false) | Highlights individual changed characters; most precise; matches Python impl | ✓ |
| Word-level (checklines=true) | Groups changes by word boundaries; faster, less precise | |
| Adaptive: word-level by default, char-level for short lines | Branches based on line length; adds branching logic and new threshold | |

**User's choice:** Character-level

**Follow-up — guard fallback behavior:**

| Option | Description | Selected |
|--------|-------------|----------|
| Skip intra-line, show line-level color only | Drop to line-level diff color when any guard triggers; no visual indicator | ✓ |
| Show line-level color + visual indicator | Add dim marker (e.g., '…') to signal intra-line was skipped | |
| Fall back to word-level diff | Retry with checklines=true before giving up | |

**User's choice:** Skip intra-line, show line-level color only
**Notes:** None — graceful degradation preferred.

---

## Claude's Discretion

None — user made explicit choices for all presented options.

## Deferred Ideas

None — discussion stayed within phase scope.
