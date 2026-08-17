---
phase: 04-config-theming-difftool-distribution
verified: 2026-08-06T19:24:51Z
status: passed
score: 5/5 must-haves verified (04-07 gap-closure scope); 53/53 plan-level truths carried forward as regression-checked
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 53/53 (1 item routed to human verification)
  gaps_closed:
    - "G-04-1: `install-difftool` wrote `difftool.trustExitCode = true`, which made git's diff engine treat alturd's intentional abort exit (1) as an unconditional fatal crash (`fatal: external diff died, stopping at <file>`, exit 128) instead of a clean stop. Closed by 04-07: `runInstallDifftool` now writes `difftool.trustExitCode = false`, restoring git's default masking of the backend's exit status. Proven by a real end-to-end test (`TestGitDifftoolExitsCleanlyWhenBackendAborts`) that shells out to an actual `git difftool -t alturd` invocation against a stub backend that exits 1, plus a sensitivity control that reseeds the superseded `true` value and confirms the fatal path reappears."
  gaps_remaining: []
  regressions: []
---

# Phase 04: Config + Theming + Difftool + Distribution Verification Report

**Phase Goal:** The application is fully configurable via TOML, correctly themed across light and dark terminals, integrable as `git difftool`, and shipped as self-contained pre-built binaries via automated GitHub Actions.
**Verified:** 2026-08-06
**Status:** passed
**Re-verification:** Yes — after gap closure (04-07, closing G-04-1 found by UAT after the prior 53/53 verification pass)

## Summary

The prior verification pass (`53/53`, `human_needed`) closed G-04-2 (difftool recursion) via 04-06 and
routed exactly one item to human sign-off: a live, real-terminal `git difftool -t alturd <file>`
run. `04-UAT.md` records that human run. Three of its four assertions passed (no `git diff
--no-index:` flood, no `fork/exec` resource exhaustion, long-filename ellipsis truncation). The
fourth failed: pressing the abort key (`Q`) did not return the shell cleanly — git printed `fatal:
external diff died, stopping at <file>` and the difftool session exited 128. This was filed as gap
G-04-1, with a full root-cause trace in `.planning/debug/DEBUG-difftool-trustexitcode-fatal.md`:
`install-difftool` writes `difftool.trustExitCode = true`; git's diff engine treats *any* non-zero
exit from a trusted external-diff command as an unconditional fatal crash — there is no channel to
say "the user cancelled, stop quietly." The user locked the fix direction: stop writing
`trustExitCode = true` (accepting that a multi-file `git difftool` session no longer stops early on
abort), and explicitly rejected the alternative (changing alturd's own abort exit convention).

This pass independently re-verified 04-07's fix — not by trusting 04-07-SUMMARY.md's claims, but by
reading the actual code, running the new/changed tests directly, and **reverting the fix in
`difftool.go` and re-running the new end-to-end test**, confirming the exact pre-fix failure mode
reproduces verbatim (`git difftool exit = 128`, `fatal: external diff died, stopping at file.txt`),
then restoring the fix and confirming the full suite is green again.

| Defect | Reverted to | Result when reverted | Status |
|---|---|---|---|
| G-04-1 fatal abort | `gitConfigSet(scopeFlag, "difftool.trustExitCode", "true")` (fix undone) | `TestGitDifftoolExitsCleanlyWhenBackendAborts` fails: `git difftool exit = 128, want 0; output="fatal: external diff died, stopping at file.txt\n"` — the exact symptom the original bug report and 04-UAT.md's `reported:` block describe | ✓ Fix confirmed real |

`git diff --stat` for 04-07's declared source-file scope (`2decb1a..aeb5f0c`, excluding
`.planning/`) touches exactly `cmd/alturd/difftool.go`, `cmd/alturd/installdifftool_test.go`,
`cmd/alturd/main.go`, `internal/tui/model.go` — no unrelated file was modified. A diff of the
comment-only commit (`0aa7ef8`) confirms every changed line in `main.go`/`model.go` is a `//` doc
comment — zero functional/control-flow changes, matching the plan's explicit constraint that
direction 2 (changing alturd's abort exit convention) was rejected.

`go build ./...`, `go vet ./...`, and `go test ./... -count=1` are all green against the current
tree (re-run directly in this verification pass, not taken from the SUMMARY).

## Goal Achievement

### Roadmap Success Criteria

| # | Success Criterion | Status | Evidence |
|---|---|---|---|
| 1 | TOML config at `$XDG_CONFIG_HOME/alturd/config.toml` or `--config <path>` overrides keybindings; unknown keys rejected with clear startup error | ✓ VERIFIED (regression-checked) | Untouched by 04-07. `go test ./internal/config/...` green |
| 2 | Terminal background auto-detected via OSC 11 with 50ms timeout; falls back to dark without hanging | ✓ VERIFIED (regression-checked) | Untouched by 04-07. `go test ./internal/tui/...` green |
| 3 | `install-difftool` writes four canonical gitconfig keys idempotently; `git difftool -t alturd <file>` launches single-file view without tree; title bar shows `alturd (difftool) — N of M — <filename>`; launch does not recurse (G-04-2); **and pressing the abort key returns cleanly with no fatal external-diff line (G-04-1)** | ✓ VERIFIED | `difftool.trustExitCode` now written as `"false"` (`cmd/alturd/difftool.go:151`, confirmed by direct read); `TestGitDifftoolExitsCleanlyWhenBackendAborts` runs a real `git difftool -t alturd -- <file>` against a stub backend that exits 1 and asserts exit 0 + no `external diff died` — independently re-run in this pass and passing; independently fault-injected (reverted the value) and confirmed it reproduces the exact reported failure |
| 4 | Every push to GitHub runs `go test ./...` on Linux, macOS, Windows via GitHub Actions | ✓ VERIFIED (regression-checked) | `.github/workflows/ci.yml` untouched by 04-07 (`git diff --stat` over `.github` for the 04-07 commit range is empty); matrix `[ubuntu-latest, macos-latest, windows-latest]` running `go test ./...` |
| 5 | A `v*.*.*` tag push triggers goreleaser to publish `CGO_ENABLED=0` binaries for Linux/macOS (amd64/arm64) and Windows (amd64) as GitHub Release assets | ✓ VERIFIED (structural, carried forward as human-confirmed by an earlier UAT round per the prior verification pass) | `.goreleaser.yaml`/`.github/workflows/release.yml` untouched by 04-07; a prior verification round recorded a human-run live tag push producing a GitHub Release with all five archives and checksums — not re-litigated here since neither backing file changed |

**Score:** 5/5 roadmap success criteria verified. All 53 plan-level must-have truths from 04-01
through 04-06 carry forward as regression-checked (untouched by 04-07's 3-task, 4-file diff). 04-07
contributes 7 new/updated truths (see Per-Plan detail below), all independently verified — including
the specific one the prior human-verification item was blocked on.

### Per-Plan Truth Detail

#### 04-01 (Config + Keybindings) — 9/9 truths verified (regression-checked, unchanged)
Not touched by 04-07. `go test ./internal/config/...` green.

#### 04-02 (CI/CD + Distribution) — 12/12 truths verified (regression-checked, unchanged)
Not touched by 04-07. `.github/workflows/*.yml` and `.goreleaser.yaml` confirmed byte-unchanged
across the 04-07 commit range.

#### 04-03 (Theming + Difftool Mode) — 15/15 truths verified (regression-checked, unchanged)
Not touched by 04-07. `go test ./internal/config/... ./internal/tui/...` green.

#### 04-04 (install-difftool Subcommand) — 9/9 truths verified (regression-checked except the trust key)
Seven of the pre-existing `TestInstallDifftool*` tests unmodified and passing. The eighth
(`TestInstallDifftoolWritesFourKeys`) had its trust-key assertion deliberately flipped by 04-07 to
match the corrected value — that flip is itself part of the fix, not a regression.

#### 04-05 (Gap Closure: Ellipsis + CR-01 + CR-02) — 8/8 truths verified (regression-checked, unchanged)
Not touched by 04-07. `TestDifftoolTitleBarTruncatesWithEllipsis`, `TestViewNoPanicOnShortTerminal`,
`TestAbortKeyQuitsWithoutProcessExit` all still pass.

#### 04-06 (Gap Closure: G-04-2 Difftool Recursion) — 7/7 truths verified (regression-checked, unchanged)
Not touched by 04-07. `TestDifftoolDiffIgnoresExternalDiffConfiguration` (4 subtests),
`TestDiffArgsDisablesExternalDiff` (5 subtests) all still pass.

#### 04-07 (Gap Closure: G-04-1 trustExitCode Fatal Abort) — 7/7 truths verified

| # | Truth | Status | Evidence |
|---|---|---|---|
| 1 | `install-difftool` writes `difftool.trustExitCode = false` on a fresh machine | ✓ VERIFIED | `TestInstallDifftoolWritesFourKeys` asserts `v == "false"`; independently re-run, passes |
| 2 | A machine already carrying the superseded `true` converges to `false` on re-run | ✓ VERIFIED | `TestInstallDifftoolRewritesStaleTrustExitCode` seeds `true` via a real `git config --global` call, runs `install-difftool`, asserts the resolved value is `false`; independently re-run, passes |
| 3 | A backend that exits non-zero (alturd's abort convention) under `git difftool -t alturd` now returns exit status 0 with no fatal external-diff line | ✓ VERIFIED | `TestGitDifftoolExitsCleanlyWhenBackendAborts` — a real end-to-end `git difftool -t alturd -- file.txt` invocation against a stub script that exits 1; asserts exit 0 and absence of `external diff died`; independently re-run, passes; independently fault-injected (reverted the fix) and reproduces exit 128 / `fatal: external diff died, stopping at file.txt` verbatim |
| 4 | Sensitivity control: reseeding the stale value reproduces the failure, so the test above is not vacuous | ✓ VERIFIED | Same test's final block reseeds `difftool.trustExitCode=true` at local scope and asserts the exit code is non-zero; passes (matches independent fault-injection above) |
| 5 | Alturd's own abort exit status is still 1 with no stderr output; no abort control flow changed | ✓ VERIFIED | `git show 0aa7ef8` diff of `main.go`/`model.go` contains only `//` comment-line changes — zero functional diff; `errAborted`'s value, `Aborted()`/`WasAborted()` bodies untouched |
| 6 | `install-difftool`'s four-key contract, idempotency, `--force` behaviour and unrelated-key preservation are unchanged | ✓ VERIFIED | `TestInstallDifftoolIsIdempotent`, `TestInstallDifftoolBlocksExistingTool`, `TestInstallDifftoolForceOverwrites`, `TestInstallDifftoolOnlyTouchesFourKeys` all still pass unmodified |
| 7 | 04-UAT.md test 1's `expected` block states the achievable target (exit 0, no fatal line) instead of the unreachable exit-1 passthrough, while the `## Gaps` block stays untouched | ✓ VERIFIED | `expected:` block line reads "...reporting 0 and no 'fatal: external diff died' line printed... (Expectation corrected...per G-04-1...)"; `grep -c 'reporting 1'` over the file returns exactly 2, both inside the untouched `## Gaps` block (`truth:` and `missing:` entries) — matches the plan's own verify gate |

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `cmd/alturd/difftool.go` | `difftool.trustExitCode` written as `"false"`; corrected G-04-1 rationale in header/D-08/doc comments | ✓ VERIFIED | `gitConfigSet(scopeFlag, "difftool.trustExitCode", "false")` at line 151; three rationale-comment locations updated and consistent with the fix |
| `cmd/alturd/installdifftool_test.go` | Flipped key assertion + `TestInstallDifftoolRewritesStaleTrustExitCode` + `TestGitDifftoolExitsCleanlyWhenBackendAborts` | ✓ VERIFIED | All three present, all pass; fault-injection confirms the third is load-bearing |
| `cmd/alturd/main.go` | `errAborted` doc comment no longer claims a trusted-exit-code contract; cites G-04-1 | ✓ VERIFIED | Comment rewritten as specified; `grep -c G-04-1` = 1; zero functional diff |
| `internal/tui/model.go` | `WasAborted` doc comment no longer claims a trusted-exit-code contract; cites G-04-1 | ✓ VERIFIED | Comment rewritten as specified; `grep -c G-04-1` = 1; zero functional diff |
| `.planning/phases/04-config-theming-difftool-distribution/04-UAT.md` | Test 1 `expected` corrected; `## Gaps` block untouched | ✓ VERIFIED | Confirmed by direct read and the grep-count check above |
| All other 04-01 through 04-06 artifacts | Unchanged by 04-07 | ✓ VERIFIED (regression-checked) | Not in `git diff --stat` scope for 04-07's commit range; all associated tests green |

### Key Link Verification

| From | To | Via | Status |
|---|---|---|---|
| `runInstallDifftool`'s fourth `gitConfigSet` call | git's resolved `difftool.trustExitCode` value | writes the literal string `"false"` at the install scope, overriding any lower-precedence scope | ✓ WIRED |
| git's resolved trust value | `git-difftool--helper`'s exit-forwarding branch | `helper` only forwards the backend's exit status when the resolved value is exactly `"true"`; `"false"` restores the default masking | ✓ WIRED (proven by the end-to-end test + fault injection, not just argued) |
| (all prior-pass links: config→TUI dispatch, theme detection→render, `difftoolDiff`'s `--no-ext-diff`, standalone `diffArgs`, CI/release workflows) | | | ✓ WIRED (regression-checked, unchanged) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Full test suite | `go build ./... && go vet ./... && go test ./... -count=1` | all 6 packages ok | ✓ PASS |
| Targeted install-difftool/GitDifftool tests | `go test ./cmd/alturd/ -run 'InstallDifftool\|GitDifftool' -count=1 -v` | all 9 subtests PASS (2 new, 1 flipped, 6 unmodified) | ✓ PASS |
| G-04-1 fix is load-bearing (independent fault injection) | Reverted `difftool.trustExitCode` value to `"true"` in `difftool.go`, re-ran `TestGitDifftoolExitsCleanlyWhenBackendAborts` alone | Fails exactly as the original bug report and 04-UAT.md describe: `git difftool exit = 128, want 0; output="fatal: external diff died, stopping at file.txt\n"` | ✓ PASS (confirms the fix is real, not cosmetic) |
| Fix restored, suite green again | Restored the value, `git diff` shows zero delta, re-ran the same test + full suite | all pass; `git status --short` clean | ✓ PASS |
| Scope discipline | `git diff --stat 2decb1a..aeb5f0c` (source tree, excluding `.planning`) | exactly `cmd/alturd/difftool.go`, `cmd/alturd/installdifftool_test.go`, `cmd/alturd/main.go`, `internal/tui/model.go` | ✓ PASS |
| Comment-only diff claim | `git show 0aa7ef8 -- cmd/alturd/main.go internal/tui/model.go` filtered to non-`//`, non-`+++`/`---` lines | empty — every changed line is a comment | ✓ PASS |
| CI/release workflows untouched | `git diff --stat 0005fc0..aeb5f0c -- .github .goreleaser.yaml` | empty | ✓ PASS |
| `gofmt -l` drift is pre-existing, not newly introduced | `gofmt -l cmd/alturd/main.go internal/tui/model.go` (before and after this diff) | both flagged in both states — unchanged from IN-01 in 04-REVIEW.md; the two new/changed test files are correctly formatted | ✓ PASS |
| Code review re-run after the fix | `04-REVIEW.md` header | `critical: 0` | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` files exist in this project. Step 7c: SKIPPED (no project probe convention in use).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| THEME-01 | 04-03 | OSC 11 auto-detect, 50ms fallback, `--theme`/config override | ✓ SATISFIED | Unchanged, regression-checked |
| CONFIG-01 | 04-01 | TOML config at XDG path or `--config`; unknown keys rejected | ✓ SATISFIED | Unchanged, regression-checked |
| CONFIG-02 | 04-01 | Any default keybinding overridable via config | ✓ SATISFIED | Unchanged, regression-checked |
| DIFFTOOL-01 | 04-03, 04-06, 04-07 | `--difftool-*` flags open single-file view, no tree; entry point reachable end-to-end under git's real invocation environment, including a clean abort | ✓ SATISFIED | 04-06 closed the recursion mechanism; 04-07 closes the fatal-abort mechanism; both independently confirmed with real subprocess tests + fault injection in this pass |
| DIFFTOOL-02 | 04-03, 04-05 | Title bar `N of M — filename` format, including ellipsis on overflow | ✓ SATISFIED | Unchanged since 04-05, regression-checked |
| DIFFTOOL-03 | 04-04, 04-07 | `install-difftool` writes 4 canonical gitconfig keys idempotently, with the corrected trust value | ✓ SATISFIED | `TestInstallDifftoolWritesFourKeys` + `TestInstallDifftoolRewritesStaleTrustExitCode` both pass; four-key/idempotency contract unchanged |
| DIST-01 | 04-02 | 3-OS `go test` CI matrix + lint gate on every push/PR | ✓ SATISFIED | Unchanged, regression-checked |
| DIST-02 | 04-02 | Tag push publishes 5-target binary release via goreleaser | ✓ SATISFIED | Unchanged, regression-checked; carried forward from an earlier verification round's recorded human-confirmed live tag push |
| DIST-03 | 04-02 | All binaries `CGO_ENABLED=0`, no runtime deps | ✓ SATISFIED | Unchanged, regression-checked |

No orphaned requirements: the 9 IDs declared across 04-01 through 04-07's `requirements:`
frontmatter (04-07 declares `[DIFFTOOL-01, DIFFTOOL-03]`, both already tracked to this phase) match
the 9 IDs REQUIREMENTS.md maps to Phase 4.

**Documentation-sync note (not a code gap, carried forward from prior passes):**
`.planning/REQUIREMENTS.md`'s Phase 4 checkboxes and Traceability table are still stale.
`git show 954e63e -- .planning/REQUIREMENTS.md` shows 04-07 flipped only `DIFFTOOL-03` to `[x]` (per
its own `requirements-completed` frontmatter); `DIFFTOOL-01` and `DIFFTOOL-02` were already `[x]`
from an earlier commit; the remaining 6 IDs (CONFIG-01, CONFIG-02, THEME-01, DIST-01, DIST-02,
DIST-03 — all independently verified satisfied above) still show `[ ]`/`Gaps Found` in the
Traceability table. This is stale administrative bookkeeping, not a functional deficiency —
recommend a full docs-sync pass before the milestone is archived so a reader of REQUIREMENTS.md
alone doesn't incorrectly conclude 6 requirements are still failing.

**Minor test-coverage observation (not a gap, not a blocker):** 04-07's first `must_haves.truths`
entry claims the fix holds "even when a lower-precedence config scope sets it to `true`." This
specific three-scope-precedence scenario (e.g., a global-scope install overriding a system-scope
`true`) is not exercised by a dedicated automated test — `TestInstallDifftoolRewritesStaleTrustExitCode`
proves reinstalling at the *same* scope converges a stale value, which is the actual write-path
mechanism (explicit value, never unset) the fix relies on. The broader claim rests on git's own
well-documented config-scope precedence semantics (verified empirically against the installed git
2.39.5 per the plan's own text) rather than alturd-specific logic, so this is a documentation/test
polish gap rather than a functional risk — noted for completeness, not filed as a blocker.

### Anti-Patterns Found

No debt markers (`TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`) found in any file 04-07 touched.
No stub patterns found. G-04-1 — the one blocker UAT found after the prior verification pass — is
closed at the code level and independently confirmed via fault injection in this pass.

Two items carried forward from `04-REVIEW.md` (re-reviewed 2026-08-06 after the G-04-1 fix, still
`critical: 0`), neither touched or introduced by 04-07:

| Finding | Severity | Status |
|---|---|---|
| WR-07: difftool full-file rendering always reads `--difftool-remote`, never `--difftool-local`, for deleted files — a real contract violation with no regression test guarding it | Warning | Not fixed by 04-07; not a must-have of any Phase 4 plan; tracked debt, not a phase-goal blocker |
| IN-01: `gofmt -l` flags `cmd/alturd/main.go` and `internal/tui/model.go` (import ordering / struct alignment); `.golangci.yml` has no `formatters` block so this doesn't fail CI | Info | Confirmed unchanged/not worsened by 04-07 (the changed test file is correctly formatted) |

Neither finding blocks any of the 5 roadmap success criteria or the 9 requirement IDs.

### Human Verification Required

None. The single item the prior verification pass routed to human sign-off — a live, real-terminal
`git difftool -t alturd <file>` run — was executed (recorded in `04-UAT.md`'s `reported:` block via
a scripted pty session against a real repo) and found the G-04-1 defect. That defect is now closed
with a genuine end-to-end automated regression test that invokes real `git`/`git difftool`
subprocesses under an isolated gitconfig, carries a sensitivity control, and was independently
fault-injected in this pass (reverting the fix reproduces the exact reported failure verbatim: exit
128, `fatal: external diff died, stopping at file.txt`). The other three assertions from that same
human/pty run (no `--no-index` flood, no fork/exec exhaustion, long-filename ellipsis) were already
confirmed passing and are unaffected by 04-07's diff (scope-checked: 04-07 touches only the trust
gitconfig value and doc comments, nothing in the recursion-prevention or title-bar-truncation code
paths).

### Gaps Summary

No gaps remain. G-04-1 — the one blocker UAT found after the prior verification pass — is closed:
`install-difftool` now writes `difftool.trustExitCode = false` instead of `true`, restoring git's
default behaviour of masking an external diff tool's exit status rather than treating any non-zero
exit as an unconditional fatal crash. This was independently confirmed in this pass via
fault-injection: reverting the value reproduces the exact pre-fix failure (`git difftool exit = 128`,
`fatal: external diff died, stopping at file.txt`), and restoring it makes the full suite green
again. Scope discipline held: exactly the four declared source files were touched, the doc-comment
changes are functionally inert (comment-only diff confirmed by filtering the commit), and no new
debt markers or stub patterns were introduced. `04-UAT.md`'s test 1 `expected` block and its `##
Gaps` block are internally consistent with the fix: the expected clause now asks for the achievable
target (exit 0, no fatal line) that the fix delivers, while the `## Gaps` block's historical record
of the original (now-superseded) exit-1-passthrough expectation is preserved untouched for
reconciliation, exactly as the plan required.

All 5 roadmap success criteria are verified. All 9 Phase 4 requirement IDs are satisfied with no
orphans. No human verification items remain — every item the prior pass deferred to a human has
either already been human-run (tag-push release, judgment-tier prohibitions, the pty-based difftool
session that found G-04-1) or is now covered by a genuine end-to-end automated test that this pass
independently re-ran and fault-injected.

Two pre-existing Warning/Info findings from `04-REVIEW.md` (WR-07 deletion full-file rendering bug,
IN-01 gofmt drift) remain untouched and unworsened by 04-07 — both are tracked debt outside this
phase's must-have scope, not phase-goal blockers.

One documentation-sync note (not a code gap, not blocking, carried forward): `.planning/REQUIREMENTS.md`'s
checkboxes and Traceability table for Phase 4 still show 6 of 9 requirements as `[ ]`/`Gaps Found`
despite this report independently verifying all 9 as satisfied — recommend a docs-sync pass before
milestone archival.

---

_Verified: 2026-08-06_
_Verifier: Claude (gsd-verifier)_
