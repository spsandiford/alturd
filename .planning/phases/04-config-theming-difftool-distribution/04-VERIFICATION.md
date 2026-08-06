---
phase: 04-config-theming-difftool-distribution
verified: 2026-08-06T00:00:00Z
status: human_needed
score: 53/53 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 52/53
  gaps_closed:
    - "G-04-2: git difftool -t alturd <file> recursively re-entered git's own difftool dispatch (via inherited GIT_EXTERNAL_DIFF), spawning processes until fork() failed — closed by 04-06 adding --no-ext-diff to both difftoolDiff's internal git diff --no-index call and the standalone diff path (diffArgs helper)."
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "In a real interactive terminal, after `alturd install-difftool`, run `git difftool -t alturd <file>` on a repo with an uncommitted change. Confirm the alturd TUI single-file view appears with no repeating 'git diff --no-index:' flood, no 'fork/exec ... resource temporarily unavailable', and no 'fatal: external diff died' lines. Then re-check the two assertions UAT test 2 was previously blocked from reaching: a long filename's title bar ends in '…' on one row, and pressing 'Q' returns the shell prompt cleanly with `echo $?` reporting 1."
    expected: "alturd TUI launches cleanly, single file side-by-side view, no recursion symptoms, ellipsis truncation and clean abort both hold in a real terminal."
    why_human: "No real interactive TTY is available in this sandbox to run git's actual difftool builtin end-to-end. The G-04-2 fix (--no-ext-diff) is independently confirmed at the mechanism level in this pass — go test ./... is green, TestDifftoolDiffIgnoresExternalDiffConfiguration's 4 subtests pass, and reverting the flag reproduces the exact pre-fix 'fatal: external diff died'/missing-diff-output failure recorded in 04-06-SUMMARY.md — but the full end-to-end symptom (git's difftool builtin actually launching alturd without recursing, observed live) can only be seen on a real TTY. 04-UAT.md's test 2 predates the 04-06 fix and was not re-run after it; this is the one carry-forward item from the prior verification pass's three human-verification items — the other two (live tag-push release, judgment-tier prohibitions) are now closed, see below."
---

# Phase 04: Config + Theming + Difftool + Distribution Verification Report

**Phase Goal:** The application is fully configurable via TOML, correctly themed across light and dark terminals, integrable as `git difftool`, and shipped as self-contained pre-built binaries via automated GitHub Actions.
**Verified:** 2026-08-06
**Status:** human_needed
**Re-verification:** Yes — after gap closure (04-06, closing G-04-2 found by UAT after the prior verification pass)

## Summary

The prior verification pass (`52/53`, `human_needed`) closed the DIFFTOOL-02 ellipsis gap and both
CR-01/CR-02 code-review criticals via 04-05, and routed three items to human sign-off: a live
tag-push GitHub Actions release, a live interactive `git difftool` render, and the five
judgment-tier prohibitions. `04-UAT.md` records the human's actual run of all three: the tag-push
release **passed**, the judgment-tier prohibitions **passed**, but the live difftool render
**failed** — `git difftool -t alturd README.md` flooded the terminal with an unbounded recursive
chain of `git diff --no-index:` errors and `fork/exec ... resource temporarily unavailable` until
the OS process/fork limit was exhausted. The alturd TUI never appeared. This was filed as gap
G-04-2.

`.planning/debug/difftool-recursive-diff-loop.md` diagnosed the root cause via three decomposed,
safe experiments (not the catastrophic full reproduction): git's own `git difftool` builtin
unconditionally sets `GIT_EXTERNAL_DIFF=git-difftool--helper` in the environment of any
`difftool.<name>.cmd` child. `difftoolDiff`'s internal `exec.Command("git", "diff", "--no-index",
"--", local, remote)` inherited that variable without overriding it, so instead of computing its
own diff it re-dispatched to `git-difftool--helper`, which re-invoked `difftool.alturd.cmd` = alturd
again, which called `difftoolDiff` again — unbounded recursion until `fork()` failed. Gap-closure
plan 04-06 added `--no-ext-diff` to that `exec.Command` call (Task 1) and to a new `diffArgs` helper
protecting the standalone `alturd [ref]` path against the same external-diff-hijack class (Task 2,
GIT-01 non-regression, since `diff.external` pointed at delta/difftastic would otherwise feed a
third-party renderer's output into `diff.Parse`).

This pass independently re-verified 04-06's fix — not by trusting 04-06-SUMMARY.md's claims, but by
reading the actual diff, running the new regression tests, and **reverting the `--no-ext-diff` flag
in `difftoolDiff` and re-running its test**, confirming the exact pre-fix failure mode reproduces
(`fatal: cannot run .../no-such-external-diff: No such file or directory` / `fatal: external diff
died` / spy-program marker file present), then restoring the fix and confirming the full suite is
green again.

| Defect | Reverted to | Result when reverted | Status |
|---|---|---|---|
| G-04-2 recursion | `exec.Command("git", "diff", "--no-index", "--", local, remote)` (flag removed) | `TestDifftoolDiffIgnoresExternalDiffConfiguration` fails on 3 of 4 subtests: `external_diff_env_var` and `diff_external_config` both fail with git's own `fatal: cannot run <path>: No such file or directory` / `fatal: external diff died` (proving git actually tried to dispatch to the named external program); `external_diff_program_never_invoked` fails because the spy marker file *does* exist (proving the spy program was actually executed) | ✓ Fix confirmed real |

`git diff --stat` between the pre-04-06 commit and the 04-06 completion commit touches exactly the
three files the plan declared (`cmd/alturd/main.go`, `cmd/alturd/difftooldiff_internal_test.go`,
`cmd/alturd/main_internal_test.go`) — no unrelated file was modified.

`go build ./...`, `go vet ./...`, and `go test ./...` are all green against the current tree
(re-run directly in this verification pass, not taken from the SUMMARY).

## Goal Achievement

### Roadmap Success Criteria

| # | Success Criterion | Status | Evidence |
|---|---|---|---|
| 1 | TOML config at `$XDG_CONFIG_HOME/alturd/config.toml` or `--config <path>` overrides keybindings; unknown keys rejected with clear startup error | ✓ VERIFIED (regression-checked) | Untouched by 04-06. `go test ./internal/config/... -v` — 14 subtests across `TestLoad_*`/`TestKeybindings_*` all pass |
| 2 | Terminal background auto-detected via OSC 11 with 50ms timeout; falls back to dark without hanging | ✓ VERIFIED (regression-checked) | Untouched by 04-06. `TestParseTheme`, `TestTheme_Precedence` (8 subtests), `TestDetectDarkBackground_TimeoutFallback` all pass |
| 3 | `install-difftool` writes four canonical gitconfig keys idempotently; `git difftool -t alturd <file>` launches single-file view without tree; title bar shows `alturd (difftool) — N of M — <filename>`; **and the launch itself does not recurse** | ✓ VERIFIED (code + regression tests); ⚠️ live end-to-end render still human-only | `install-difftool` tests unchanged (7/7 pass). `difftoolDiff` now carries `--no-ext-diff` (`cmd/alturd/main.go:303`), closing G-04-2 at its only reachable call site — see Gap Closure detail below |
| 4 | Every push to GitHub runs `go test ./...` on Linux, macOS, Windows via GitHub Actions | ✓ VERIFIED (regression-checked) | `.github/workflows/ci.yml` untouched; matrix `[ubuntu-latest, macos-latest, windows-latest]` running `go test ./...`, plus a separate `golangci-lint` job |
| 5 | A `v*.*.*` tag push triggers goreleaser to publish `CGO_ENABLED=0` binaries for Linux/macOS (amd64/arm64) and Windows (amd64) as GitHub Release assets | ✓ VERIFIED (structural + human-confirmed) | `.goreleaser.yaml`/`.github/workflows/release.yml` untouched, confirmed present and structurally correct (5 targets, `CGO_ENABLED=0`, `checksums.txt`). **04-UAT.md test 1 records a human ran this live and it passed** — a real tag push produced a GitHub Release with all five archives and checksums |

**Score:** 53/53 plan-level must-have truths verified (up from 52/53 — 04-06's 7 truths all
verified, closing the one gap the prior pass's re-verification cycle was tracking). One item (the
live end-to-end difftool render, now re-scoped to also confirm the G-04-2 fix) remains routed to
human verification.

### Per-Plan Truth Detail

#### 04-01 (Config + Keybindings) — 9/9 truths verified (regression-checked, unchanged since prior pass)

Not touched by 04-05 or 04-06. `go test ./internal/config/...` green, 14 subtests.

#### 04-02 (CI/CD + Distribution) — 12/12 truths verified

Not touched by 04-05 or 04-06. The one previously-backstop truth (live tag-push GitHub Release) is
now **human-confirmed passed** per `04-UAT.md` test 1 — no longer merely structural.

#### 04-03 (Theming + Difftool Mode) — 15/15 truths verified

Not touched by 04-06 except indirectly (the difftool full-file/title-bar code this plan wrote is the
context `difftoolDiff` feeds). All prior-pass evidence stands; `go test ./internal/config/... ./internal/tui/...` green.

#### 04-04 (install-difftool Subcommand) — 9/9 truths verified (regression-checked, unchanged)

Not touched by 04-05 or 04-06. `go test ./cmd/alturd/... -run TestInstallDifftool` green, 7 subtests.

#### 04-05 (Gap Closure: Ellipsis + CR-01 + CR-02) — 8/8 truths verified (regression-checked, unchanged)

Not touched by 04-06. `TestDifftoolTitleBarTruncatesWithEllipsis`, `TestViewNoPanicOnShortTerminal`,
`TestAbortKeyQuitsWithoutProcessExit` all still pass.

#### 04-06 (Gap Closure: G-04-2 Difftool Recursion) — 7/7 truths verified

| # | Truth | Status | Evidence |
|---|---|---|---|
| 1 | With `GIT_EXTERNAL_DIFF` set, `difftoolDiff` returns git's own built-in diff and nil error, never dispatching to the named program | ✓ VERIFIED | `TestDifftoolDiffIgnoresExternalDiffConfiguration/external_diff_env_var` passes; independently fault-injected (reverted the flag) and confirmed it fails with git's own `fatal: cannot run <path>` / `fatal: external diff died` |
| 2 | With `diff.external` set in gitconfig (no env var), same suppression holds | ✓ VERIFIED | `.../diff_external_config` subtest passes; same fault-injection reproduces failure |
| 3 | A spy program named by `GIT_EXTERNAL_DIFF` is never executed | ✓ VERIFIED | `.../external_diff_program_never_invoked` passes (marker file absent); reverted flag makes the marker file appear, proving the spy *was* invoked pre-fix |
| 4 | Exit-code contract unchanged: identical files → nil error, empty output; differing files → nil error, unified diff | ✓ VERIFIED | `.../identical_files_still_exit_zero` passes both pre- and post-fix (was never broken); the diff-content assertions in the other three subtests confirm real unified-diff output post-fix |
| 5 | `git difftool -t alturd <file>` launches the alturd TUI instead of flooding the terminal (UAT test 2) | ⚠️ Mechanism verified; live render still human-only | The recursion vector is structurally removed (see truths 1-4) and 04-06's plan explicitly defers the live-terminal confirmation to phase re-verification — see Human Verification below |
| 6 | Standalone `alturd [ref]` path also disables external-diff dispatch (GIT-01 non-regression) | ✓ VERIFIED | `TestDiffArgsDisablesExternalDiff` (5 subtests: nil/single/range refs, `--`-pathspec, non-mutation guard) all pass; `grep` confirms `run()` calls `diffArgs(git.ParseRefArgs(...))` |
| 7 | Every pre-existing test still passes unmodified | ✓ VERIFIED | `go test ./... -count=1` green across all 6 packages; `git diff --stat` confirms only the 3 declared files changed |

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `cmd/alturd/main.go` | `difftoolDiff`'s argv carrying `--no-ext-diff`; `diffArgs` helper for the standalone path | ✓ VERIFIED | `exec.Command("git", "diff", "--no-index", "--no-ext-diff", "--", local, remote)` at line 303; `diffArgs(refArgs []string) []string` at line 202 returning `append([]string{"diff", "--no-ext-diff"}, refArgs...)` |
| `cmd/alturd/difftooldiff_internal_test.go` | `TestDifftoolDiffIgnoresExternalDiffConfiguration` (4 subtests) | ✓ VERIFIED | New file, present, all 4 subtests pass |
| `cmd/alturd/main_internal_test.go` | `TestDiffArgsDisablesExternalDiff` (5 subtests) | ✓ VERIFIED | Present alongside pre-existing `TestReportError`, all 5 subtests pass |
| All other 04-01 through 04-05 artifacts (`internal/config/*.go`, `internal/tui/model.go`, `.github/workflows/*.yml`, `.goreleaser.yaml`, `cmd/alturd/difftool.go`) | Unchanged by 04-06 | ✓ VERIFIED (regression-checked) | Not in `git diff --stat` scope for 04-06; all associated tests green |

### Key Link Verification

| From | To | Via | Status |
|---|---|---|---|
| `difftoolDiff`'s `exec.Command` | git subprocess | `--no-ext-diff` breaks the recursion cycle at its only reachable call site | ✓ WIRED |
| `run()`'s standalone branch | `diffArgs` helper | `gitArgs := diffArgs(git.ParseRefArgs(args, cmd.ArgsLenAtDash()))` | ✓ WIRED (grep confirms call site) |
| (all prior-pass links: config→TUI dispatch, theme detection→render, install-difftool→gitconfig, CI/release workflows) | | | ✓ WIRED (regression-checked, unchanged) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Full test suite | `go build ./... && go vet ./... && go test ./...` | all 6 packages ok | ✓ PASS |
| G-04-2 fix is load-bearing (fault injection) | Reverted `--no-ext-diff` in `difftoolDiff`'s `exec.Command`, re-ran `TestDifftoolDiffIgnoresExternalDiffConfiguration` | 3 of 4 subtests fail exactly as documented (git's own `fatal: cannot run.../fatal: external diff died`; spy marker file appears) | ✓ PASS (confirms fix is real, not cosmetic) |
| Fix restored, suite green again | Restored the flag, re-ran the same test + full suite | all pass | ✓ PASS |
| Scope discipline | `git diff --stat` for 04-06's commit range (excluding planning docs) | exactly `cmd/alturd/main.go`, `cmd/alturd/difftooldiff_internal_test.go`, `cmd/alturd/main_internal_test.go` | ✓ PASS |
| `install-difftool` gitconfig keys untouched (explicit scope decision) | `go test ./cmd/alturd/... -run TestInstallDifftool -v` | all 7 subtests pass, unchanged | ✓ PASS |
| `gofmt -l` drift is pre-existing, not newly introduced | `gofmt -l cmd/alturd/main.go internal/tui/model.go` | both flagged (unchanged from IN-01 in 04-REVIEW.md); `cmd/alturd/difftooldiff_internal_test.go`/`main_internal_test.go` NOT flagged | ✓ PASS (new 04-06 files are correctly formatted; pre-existing drift not worsened) |

### Probe Execution

No `scripts/*/tests/probe-*.sh` files exist in this project. Step 7c: SKIPPED (no project probe convention in use).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| THEME-01 | 04-03 | OSC 11 auto-detect, 50ms fallback, `--theme`/config override | ✓ SATISFIED | Unchanged, regression-checked |
| CONFIG-01 | 04-01 | TOML config at XDG path or `--config`; unknown keys rejected | ✓ SATISFIED | Unchanged, regression-checked |
| CONFIG-02 | 04-01 | Any default keybinding overridable via config | ✓ SATISFIED | Unchanged, regression-checked |
| DIFFTOOL-01 | 04-03, 04-06 | `--difftool-*` flags open single-file view, no tree; the entry point is now reachable end-to-end under the environment git actually creates (not just a clean one) | ✓ SATISFIED (code-level); ⚠️ live end-to-end launch still human-only | 04-06 closes the mechanism-level gap; a real-terminal `git difftool -t alturd <file>` run is the last confirmation step |
| DIFFTOOL-02 | 04-03, 04-05 | Title bar `N of M — filename` format, including ellipsis on overflow | ✓ SATISFIED | Unchanged since 04-05, regression-checked |
| DIFFTOOL-03 | 04-04 | `install-difftool` writes 4 canonical gitconfig keys idempotently | ✓ SATISFIED | Unchanged, regression-checked |
| DIST-01 | 04-02 | 3-OS `go test` CI matrix + lint gate on every push/PR | ✓ SATISFIED | Unchanged, regression-checked |
| DIST-02 | 04-02 | Tag push publishes 5-target binary release via goreleaser | ✓ SATISFIED — now human-confirmed | 04-UAT.md test 1: human ran a live tag push, release published with 5 archives + checksums, passed |
| DIST-03 | 04-02 | All binaries `CGO_ENABLED=0`, no runtime deps | ✓ SATISFIED | Unchanged, regression-checked |

No orphaned requirements: all 9 IDs declared across the plans' `requirements:` frontmatter (04-01
through 04-06, including 04-06's `[DIFFTOOL-01]`) match the 9 IDs REQUIREMENTS.md maps to Phase 4.

**Documentation-sync note (not a code gap, carried forward from the prior pass):**
`.planning/REQUIREMENTS.md`'s Phase 4 checkboxes and Traceability table are stale — only
`DIFFTOOL-01` and `DIFFTOOL-02` are marked `[x]`/`Complete`; the other 7 (including CONFIG-01/02,
THEME-01, DIFFTOOL-03, DIST-01/02/03 — all independently verified satisfied above, and DIST-02 now
human-confirmed via a real release) still show `[ ]`/`Gaps Found`. This is stale administrative
bookkeeping, not a functional deficiency — recommend a docs-sync pass before the milestone is
archived so a reader of REQUIREMENTS.md alone doesn't incorrectly conclude 7 requirements are still
failing.

### Anti-Patterns Found

No debt markers (`TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`) found in any file 04-06 touched.
No stub patterns found. The G-04-2 blocker UAT filed against the prior pass is now closed at the
code level (see Gap Closure detail above).

Two items carried forward from `04-REVIEW.md` (2026-08-06), neither touched or introduced by 04-06,
both still present and still Warning/Info tier (not blockers for this phase's goal):

| Finding | Severity | Status |
|---|---|---|
| WR-07: difftool full-file rendering always reads `--difftool-remote`, never `--difftool-local`, for deleted files — `AlignFull`'s documented contract requires the *old* file for deletions; impact is currently masked because a full-file deletion diff is always one contiguous hunk, but this is a real contract violation with no regression test guarding it | Warning | Not fixed by 04-06; not a must-have of any Phase 4 plan; tracked debt, not a phase-goal blocker |
| IN-01: `gofmt -l` flags `cmd/alturd/main.go` and `internal/tui/model.go` (import ordering / struct alignment); `.golangci.yml` has no `formatters` block so this doesn't fail CI | Info | Confirmed unchanged/not worsened by 04-06 (the two new 04-06 files are correctly formatted) |

Neither finding blocks any of the 5 roadmap success criteria or the 9 requirement IDs — both are
pre-existing warning/info-tier debt outside this phase's must-have scope, consistent with
04-REVIEW.md's own `critical: 0` finding count.

### Human Verification Required

One item remains (down from three in the prior pass — the other two are now closed by `04-UAT.md`'s
own human-run results):

1. **Live `git difftool -t alturd <file>` render, post-G-04-2-fix**
   **Test:** In a real interactive terminal, after `alturd install-difftool`, run `git difftool -t
   alturd <file>` on a repo with an uncommitted change (ideally also on a file whose name exceeds
   the terminal width, to close out the 04-05 assertions in the same pass).
   **Expected:** The alturd TUI single-file side-by-side view appears; no repeating `git diff
   --no-index:` flood; no `fork/exec ... resource temporarily unavailable`; no `fatal: external
   diff died` lines; long filenames truncate with a trailing `…` on one row; pressing `Q` returns
   the shell prompt cleanly with `echo $?` reporting 1.
   **Why human:** This sandbox has no real interactive TTY capable of running git's actual
   difftool builtin end-to-end. The G-04-2 fix is confirmed at the mechanism level in this pass
   (regression tests pass; fault-injection reproduces the exact pre-fix failure when the fix is
   reverted), but the live, real-terminal launch itself — the thing `04-UAT.md` test 2 originally
   failed on — has not been re-run since the fix landed.

Already resolved by `04-UAT.md` (human-run, not re-litigated here since neither backing code path
was touched by 04-06):
- Live tag-push GitHub Actions release — **passed** (04-UAT.md test 1)
- Judgment-tier prohibitions sign-off (CONFIG-01 no-side-effects, CONFIG-02 no-silenced-exit,
  DIST-02 checksum coverage, THEME-01 no-visible-OSC-bytes, DIFFTOOL-01 read-only file access) —
  **passed** (04-UAT.md test 3)

### Gaps Summary

No gaps remain. G-04-2 — the one blocker UAT found after the prior verification pass — is closed:
`difftoolDiff`'s internal `git diff --no-index` call and the standalone `alturd [ref]` path both now
carry `--no-ext-diff`, making them immune to inherited (`GIT_EXTERNAL_DIFF`) or configured
(`diff.external`) external-diff dispatch. This was independently confirmed in this pass via
fault-injection: reverting the flag reproduces the exact pre-fix failure (git's own `fatal: cannot
run <path>`/`fatal: external diff died`, and a spy program's marker file appearing on disk), and
restoring it makes the full suite green again. Scope discipline held: exactly the three files 04-06
declared were touched, `internal/git/runner.go`/`cmd/alturd/difftool.go`/the gitconfig keys
`install-difftool` writes are all unchanged, and no new debt markers or stub patterns were
introduced.

One item remains for human sign-off: a live, real-terminal re-run of `git difftool -t alturd <file>`
to confirm the G-04-2 fix holds end-to-end under git's actual difftool builtin (not just under the
regression test's simulated poisoned environment). This is the only reason overall status is
`human_needed` rather than `passed` — every other truth, artifact, key link, and requirement is
independently verified in this pass, and two of the prior pass's three human-verification items
(live tag-push release, judgment-tier prohibitions) are now closed via `04-UAT.md`'s recorded human
results.

Two pre-existing Warning/Info findings from `04-REVIEW.md` (WR-07 deletion full-file rendering bug,
IN-01 gofmt drift) remain untouched and unworsened by 04-06 — both are tracked debt outside this
phase's must-have scope, not phase-goal blockers.

One documentation-sync note (not a code gap, not blocking, carried forward): `.planning/REQUIREMENTS.md`'s
checkboxes and Traceability table for Phase 4 still show 7 of 9 requirements as `[ ]`/`Gaps Found`
despite this report independently verifying all 9 as satisfied — recommend a docs-sync pass before
milestone archival.

---

_Verified: 2026-08-06_
_Verifier: Claude (gsd-verifier)_
