# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v1.0 — MVP

**Shipped:** 2026-08-17
**Phases:** 5 (01-diff-model, 02-git-layer-cli, 03-tui-application, 04-config-theming-difftool-distribution, 04.1-address-tech-debt) | **Plans:** 21 | **Timeline:** 2026-06-25 → 2026-08-17 (~53 days)

### What Was Built
- `internal/diff`: Go-native diff parsing/rendering engine (go-gitdiff, chroma, go-diff) validated against a 13-scenario fixture corpus
- `internal/git` + `cmd/alturd`: git subprocess layer and cobra CLI covering all six ref-grammar invocation forms
- `internal/tui`: full bubbletea v2 split-screen application — file tree, diff pane, hunk/file navigation, in-pane search
- `internal/config`: TOML config with strict validation, overridable keybindings, OSC 11 light/dark auto-theming
- `git difftool` integration (`install-difftool` + single-file mode) with two hardened production bugs (recursive process spawn, fatal exit on abort)
- Three-OS CI matrix + goreleaser publishing 5 `CGO_ENABLED=0` binaries on tag push

### What Worked
- Bottom-up phase ordering (diff model → git/CLI → TUI → config/theming/distribution) meant the highest-correctness-risk code (diff parsing) was validated in isolation before any TUI code existed, and later phases never had to revisit it.
- The `find_root_cause_only` debug pattern (used for both G-04-1 and G-04-2) let root causes get fully diagnosed via decomposed, safe experiments — e.g. avoiding an actual fork-bomb reproduction for the recursive-diff-loop bug by proving the mechanism through three smaller spy-script experiments instead.
- The TestMain subprocess pattern (persist a built binary for the whole test run via `os.MkdirTemp`/`defer os.RemoveAll` in `TestMain`, since `t.TempDir()` doesn't survive across subtests) was established in Phase 2 and reused without rediscovery in later phases.
- Security review (Phase 4) closed 49/49 STRIDE threats with a clear verified/accepted-risk split, giving a real signal rather than a checkbox exercise.

### What Was Inefficient
- Phase 3's blocking human UAT checkpoint (the primary split-screen visual smoke test) was never actually run when Phase 3 shipped — it sat unclosed through all of Phase 4 and only got executed in Phase 04.1, three phases later. A blocking checkpoint that can silently roll forward unexecuted is a process gap, not just an execution slip.
- REQUIREMENTS.md's traceability table drifted out of sync with its own checkboxes (8 Phase 4 rows said "Gaps Found" despite `[x]` and independent SATISFIED verdicts) and had to be resynced in a dedicated Phase 04.1 plan. Documentation-as-code discipline (auto-derive traceability status rather than hand-maintain it) would have prevented this class of debt entirely.
- Two debug sessions (`DEBUG-difftool-trustexitcode-fatal`, `difftool-recursive-diff-loop`) had their root causes fixed in Phase 4 but the debug session `.md` files were never flipped from `status: diagnosed` to `status: fixed` — surfaced again as open items at milestone close and had to be acknowledged/deferred rather than closed cleanly.
- Phase 4's Nyquist validation record sat at `status: draft` after the 04-07 gap-closure plan (post-dated the phase's original wave-based validation) and required a dedicated reconciliation run in Phase 04.1 rather than being caught immediately.
- `go test -race ./...` could never be run in the execution sandbox (no C toolchain, `CGO_ENABLED=0`, no root) — this constraint should be flagged earlier so race-sensitive changes get scheduled for CI verification rather than discovered as a gap at the tech-debt sweep.

### Patterns Established
- Debug-then-plan separation: diagnose-only debug sessions (`goal: find_root_cause_only`) followed by a dedicated gap-closure plan that applies and regression-tests the fix, rather than fixing inline during diagnosis.
- Every debug session records `Eliminated` hypotheses with evidence before reaching a confirmed root cause — kept false leads visible instead of silently discarded.
- Gap-closure plans consistently pair the fix with both a regression test and a "sensitivity control" test (seed the stale/broken value, confirm the old failure mode still reproduces) — proves the test would have caught the original bug, not just that the new code path is exercised.

### Key Lessons
1. When a plan defers a blocking human-verification checkpoint ("cannot be run by executor agent"), track it with the same urgency as an open requirement — it should block milestone close by default, not require a dedicated later phase to discover it was never run.
2. Traceability tables and Nyquist/validation status fields that are hand-set by a plan step will drift; prefer deriving them from VERIFICATION.md/SUMMARY.md at read time, or add a lint/audit step that runs automatically rather than being rediscovered at milestone audit.
3. When a debug session's root cause gets fixed by a later plan, that plan should update the debug session file's `status` field as part of its own completion — don't leave a second, disconnected artifact for milestone close to catch.

### Cost Observations
- Model mix: not tracked this milestone.
- Sessions: not tracked this milestone.
- Notable: the milestone required one full audit-and-reaudit cycle (2026-08-06 audit found 6 tech-debt items → Phase 04.1 inserted specifically to close them → 2026-08-17 re-audit confirmed closure). Inserting a dedicated tech-debt phase before archival, rather than accepting debt into the shipped baseline, kept the requirements/verification record honest at the cost of one extra phase and ~11 days.

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Sessions | Phases | Key Change |
|-----------|----------|--------|------------|
| v1.0 | — | 5 (incl. 1 inserted tech-debt phase) | Established debug-then-gap-closure-plan pattern; first use of a dedicated tech-debt sweep phase before milestone archival |

### Cumulative Quality

| Milestone | Tests | Coverage | Zero-Dep Additions |
|-----------|-------|----------|--------------------|
| v1.0 | `go test ./...` full pass across 6 packages | Not separately tracked | 8 direct deps (bubbletea/v2, lipgloss/v2, bubbles/v2, chroma/v2, go-gitdiff, go-diff, go-toml/v2, termenv) + supporting libs (xdg, reflow, charmbracelet/log) |

### Top Lessons (Verified Across Milestones)

1. Blocking human-verification checkpoints need enforcement, not just a flag in VERIFICATION.md — v1.0 saw one roll forward unexecuted for 3 phases.
2. Hand-maintained status/traceability fields (REQUIREMENTS.md Status column, VALIDATION.md `status`) drift from the underlying truth without an automated sync — v1.0 needed a dedicated doc-sync phase to close the gap.
