---
phase: 04-config-theming-difftool-distribution
plan: 04
subsystem: difftool-install
tags: [git-config, cobra, difftool, gitconfig, subprocess]

# Dependency graph
requires:
  - phase: 04-config-theming-difftool-distribution (plan 03)
    provides: "--difftool-local/--difftool-remote/--difftool-path flags on rootCmd, which the difftool.<name>.cmd gitconfig value written by this plan must name exactly"
provides:
  - "cmd/alturd/difftool.go: install-difftool cobra subcommand, gitConfigRun/gitConfigGet/gitConfigSet git-config-aware subprocess helpers, validateScope/validateToolName validators"
  - "internal/git.ErrLocalScopeOutsideRepo sentinel *ExitCodeError"
  - "The published difftool.<name>.cmd gitconfig contract: alturd --difftool-local \"$LOCAL\" --difftool-remote \"$REMOTE\" --difftool-path \"$MERGED\""
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "gitConfigRun/gitConfigGet/gitConfigSet interpret git config's own exit-code contract (0=success, 1=key/section not found=normal, 2=invalid config file) instead of reusing internal/git.ExecRunner's git-diff-tuned error mapping"
    - "gitConfigRun always prepends the literal \"config\" subcommand itself — callers pass only the arguments that follow it (scopeFlag, \"--get\"/\"--set\", key, [value])"
    - "Local-scope-outside-a-repo detection matches the stderr substring \"git repository\" (covers both \"not a git repository\" and git 2.39.5's actual \"--local can only be used inside a git repository\" wording) rather than a single literal phrase"
    - "D-10 idempotency comparison between the existing diff.tool value and --name is an exact byte-string comparison — no case folding, no Unicode normalisation"

key-files:
  created:
    - cmd/alturd/difftool.go
    - cmd/alturd/difftool_internal_test.go
    - cmd/alturd/installdifftool_test.go
  modified:
    - internal/git/errors.go

key-decisions:
  - "gitConfigRun prepends \"config\" to the git argv itself (git config <args>), not just \"git <args>\" — the plan's action text and RESEARCH.md's illustrative snippet both omitted the literal \"config\" subcommand; every invocation would have failed with git's own usage error without this fix (Rule 1 auto-fix)."
  - "Local-scope-outside-repo stderr detection widened from the literal \"not a git repository\" (04-RESEARCH.md Pitfall D's assumed wording) to the substring \"git repository\", verified against the actual installed git 2.39.5 binary, which phrases the error as \"--local can only be used inside a git repository\" (Rule 1 auto-fix)."
  - "Task 1's cobra command declaration (installDifftoolCmd/init) was deferred to Task 2's commit exactly as the plan's task boundaries specify — Task 1's own commit contains only the git-config helpers and validators, keeping each task's commit buildable and scoped to its own action list."

patterns-established:
  - "difftool.go's package doc comment documents why it deliberately does not call internal/git.ExecRunner, citing 04-RESEARCH.md Pitfall C by name — the dominant doc-comment-with-rationale convention in this codebase."

requirements-completed: [DIFFTOOL-03]

coverage:
  - id: D1
    description: "install-difftool writes exactly the four canonical gitconfig keys (diff.tool, difftool.<name>.cmd, difftool.prompt=false, difftool.trustExitCode=true) and no others."
    requirement: "DIFFTOOL-03"
    verification:
      - kind: integration
        ref: "cmd/alturd/installdifftool_test.go#TestInstallDifftoolWritesFourKeys"
        status: pass
      - kind: integration
        ref: "cmd/alturd/installdifftool_test.go#TestInstallDifftoolOnlyTouchesFourKeys"
        status: pass
      - kind: manual
        ref: "phase gate: scratch repo install + git config --get difftool.alturd.cmd returned the exact published string"
        status: pass
    human_judgment: false
  - id: D2
    description: "--scope defaults to global, --name defaults to alturd (D-09)."
    requirement: "DIFFTOOL-03"
    verification:
      - kind: manual
        ref: "alturd install-difftool --help lists --scope (default global), --name (default alturd), --force (default false)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Re-running install-difftool when alturd's own four keys are already set is a safe no-op requiring no --force, exits 0, and re-converges the four keys (D-10)."
    requirement: "DIFFTOOL-03"
    verification:
      - kind: integration
        ref: "cmd/alturd/installdifftool_test.go#TestInstallDifftoolIsIdempotent"
        status: pass
    human_judgment: false
  - id: D4
    description: "--force is required only when diff.tool already names a different tool; without it exits 1 with the exact blocked-overwrite copy; with it, exits 0 with the exact overwrite copy (D-10)."
    requirement: "DIFFTOOL-03"
    verification:
      - kind: integration
        ref: "cmd/alturd/installdifftool_test.go#TestInstallDifftoolBlocksExistingTool"
        status: pass
      - kind: integration
        ref: "cmd/alturd/installdifftool_test.go#TestInstallDifftoolForceOverwrites"
        status: pass
    human_judgment: false
  - id: D5
    description: "install-difftool --scope local outside a git repository exits 1 with the exact single-line error, not a raw git stderr dump (04-RESEARCH.md Pitfall D)."
    requirement: "DIFFTOOL-03"
    verification:
      - kind: integration
        ref: "cmd/alturd/installdifftool_test.go#TestInstallDifftoolLocalScopeOutsideRepo"
        status: pass
      - kind: unit
        ref: "cmd/alturd/difftool_internal_test.go#TestGitConfigGetLocalScopeOutsideRepo"
        status: pass
    human_judgment: false
  - id: D6
    description: "--name '' and --scope '' are each rejected with a single-line error and exit 1, and no gitconfig key is ever written."
    requirement: "DIFFTOOL-03"
    verification:
      - kind: integration
        ref: "cmd/alturd/installdifftool_test.go#TestInstallDifftoolRejectsEmptyArgs"
        status: pass
      - kind: unit
        ref: "cmd/alturd/difftool_internal_test.go#TestValidateToolName, TestValidateScope"
        status: pass
    human_judgment: false
  - id: D7
    description: "git config --get diff.tool exiting 1 because the key is unset is a normal 'not configured' result, never surfaced as an error (04-RESEARCH.md Pitfall C)."
    requirement: "DIFFTOOL-03"
    verification:
      - kind: unit
        ref: "cmd/alturd/difftool_internal_test.go#TestGitConfigGetUnsetKeyIsNotAnError"
        status: pass
    human_judgment: false

# Metrics
duration: ~25min
completed: 2026-08-03
status: complete
---

# Phase 04 Plan 04: Install-Difftool Subcommand Summary

**`alturd install-difftool` writes the four canonical gitconfig keys (`diff.tool`, `difftool.<name>.cmd`, `difftool.prompt`, `difftool.trustExitCode`) idempotently via dedicated `git config`-exit-code-aware subprocess helpers, completing DIFFTOOL-03 and Phase 4.**

## Performance

- **Duration:** ~25 min
- **Completed:** 2026-08-03
- **Tasks:** 2 (both `type="auto" tdd="true"`)
- **Files modified:** 4 (3 created: `cmd/alturd/difftool.go`, `cmd/alturd/difftool_internal_test.go`, `cmd/alturd/installdifftool_test.go`; 1 modified: `internal/git/errors.go`)

## Accomplishments

- Added `git.ErrLocalScopeOutsideRepo`, a sentinel `*ExitCodeError` constructed exactly like the existing `ErrNotGitRepo`, so `main()`'s existing `errors.As` dispatcher routes it with zero changes to `main()`
- Implemented `gitConfigRun`/`gitConfigGet`/`gitConfigSet` in `cmd/alturd/difftool.go` — a dedicated subprocess boundary that interprets `git config`'s own exit-code contract (0=success, 1=key/section not found=normal outcome, 2=invalid config file) rather than reusing `internal/git.ExecRunner`'s `git diff`-tuned error mapping, per 04-RESEARCH.md Pitfall C
- Implemented `validateScope`/`validateToolName`, gating `--scope`/`--name` before any subprocess runs; `toolNamePattern` anchors `--name` to `^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$` to prevent gitconfig key injection via a crafted tool name (T-04-04-01)
- Implemented the `install-difftool` cobra subcommand (`--scope` default `global`, `--name` default `alturd`, `--force` default `false` — D-09) and `runInstallDifftool`, which reads the existing `diff.tool` value, applies the D-10 idempotency/`--force` contract, and writes all four canonical keys each through its own `gitConfigSet` call so git — not alturd — preserves unrelated entries, comments and ordering
- Published the exact `difftool.<name>.cmd` gitconfig value: `` alturd --difftool-local "$LOCAL" --difftool-remote "$REMOTE" --difftool-path "$MERGED" `` — this names the `--difftool-local`/`--difftool-remote`/`--difftool-path` flags 04-03 registered on `rootCmd`, confirmed byte-for-byte via `TestInstallDifftoolWritesFourKeys` and a manual scratch-repo phase-gate check

## Task Commits

Each task was TDD-cycled (RED test commit, then GREEN implementation commit):

1. **Task 1: git-config-aware subprocess helpers, argument validation, local-scope sentinel** — `bb5581d` (test, RED), `15e0cab` (feat, GREEN)
2. **Task 2: install-difftool subcommand — four keys, idempotency, --force** — `581794e` (test, RED), `ab1d93f` (feat, GREEN)

**Plan metadata:** pending (this commit)

## Files Created/Modified

- `cmd/alturd/difftool.go` — `gitConfigRun`/`gitConfigGet`/`gitConfigSet`, `validateScope`/`validateToolName`, `toolNamePattern`, `difftoolCmdTemplate`, `installDifftoolCmd`, `runInstallDifftool`
- `cmd/alturd/difftool_internal_test.go` — `TestValidateToolName`, `TestValidateScope`, `TestGitConfigGetUnsetKeyIsNotAnError`, `TestGitConfigGetLocalScopeOutsideRepo` (package `main`, whitebox)
- `cmd/alturd/installdifftool_test.go` — `TestInstallDifftoolWritesFourKeys`, `TestInstallDifftoolIsIdempotent`, `TestInstallDifftoolBlocksExistingTool`, `TestInstallDifftoolForceOverwrites`, `TestInstallDifftoolOnlyTouchesFourKeys`, `TestInstallDifftoolLocalScopeOutsideRepo`, `TestInstallDifftoolRejectsEmptyArgs` (package `main_test`, subprocess integration, reuses `TestMain`/`alturdBin` from `main_test.go`)
- `internal/git/errors.go` — added `ErrLocalScopeOutsideRepo`

## Decisions Made

- **`gitConfigRun` prepends `"config"` to the git argv itself:** both the plan's action text and 04-RESEARCH.md's Pattern 5 code snippet describe calling `gitConfigRun(scopeFlag, "--get", key)` without ever showing the literal `"config"` argument reaching `exec.Command`. Taken literally, every invocation would run `git --global --get diff.tool` (missing the `config` subcommand) and fail with git's own usage error rather than returning a real value or exit code. Fixed by having `gitConfigRun` always prepend `"config"` to the args slice it passes to `exec.Command`, so callers only ever specify the arguments that follow `git config`.
- **Local-scope-outside-repo detection widened to `"git repository"`:** 04-RESEARCH.md Pitfall D and the plan's `<action>` text both specify matching stderr for the literal phrase `"not a git repository"`. The git binary actually installed in this environment (2.39.5) phrases the error as `"--local can only be used inside a git repository"` — which does not contain `"not a git repository"` as a substring. Verified this directly against the real git CLI before writing the detection logic, then matched on the broader (but still narrow-enough-to-be-safe, since it only fires inside the `*exec.ExitError` branch of a `git config` invocation) substring `"git repository"`, which covers both phrasings.
- **D-10 idempotency comparison is an exact byte-string comparison:** `existing != name` in `runInstallDifftool` performs no case folding or normalisation, matching git's own case-sensitive subsection semantics for `difftool.<name>.*` keys.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `gitConfigRun` was missing the literal `"config"` git subcommand**
- **Found during:** Task 1, first RED→GREEN verification pass (`TestGitConfigGetUnsetKeyIsNotAnError`/`TestGitConfigGetLocalScopeOutsideRepo` both failed with exit code 129 instead of the expected exit 1 / sentinel error)
- **Issue:** The plan's action text for `gitConfigGet`/`gitConfigSet` describes calling `gitConfigRun(scopeFlag, "--get", key)` / `gitConfigRun(scopeFlag, key, value)`, and 04-RESEARCH.md's Pattern 5 illustrative snippet never shows the underlying `exec.Command` construction including the `"config"` subcommand. A literal `exec.Command("git", scopeFlag, "--get", key)` call — i.e. `git --global --get diff.tool` — is not a valid git invocation.
- **Fix:** `gitConfigRun` now always prepends `"config"` to the args slice before constructing `exec.Command("git", fullArgs...)`. Callers (`gitConfigGet`, `gitConfigSet`) are unchanged from the plan's described call shape — they pass only the arguments after `config`.
- **Files modified:** `cmd/alturd/difftool.go`
- **Verification:** `TestGitConfigGetUnsetKeyIsNotAnError` and `TestGitConfigGetLocalScopeOutsideRepo` both pass after the fix; the full Task 2 subprocess suite (which depends on this helper) also passes.
- **Committed in:** `15e0cab` (Task 1 GREEN commit)

**2. [Rule 1 - Bug] Local-scope-outside-repo stderr detection widened from a literal phrase to a substring**
- **Found during:** Task 1, same verification pass as above (once the missing-`config` bug was fixed, the outside-repo test still failed because the actual git stderr text did not match the plan-specified literal)
- **Issue:** 04-RESEARCH.md Pitfall D and the plan's Task 1 action text both specify detecting `"not a git repository"` in stderr. Verified directly against the git binary in this environment (`git 2.39.5`) that `git config --local --get diff.tool` outside a repository actually prints `"fatal: --local can only be used inside a git repository"` — a different phrasing that does not contain `"not a git repository"` as a substring.
- **Fix:** Detection logic in `gitConfigRun` now matches the substring `"git repository"` (case-insensitive), which covers both the RESEARCH.md-assumed phrasing and the actual installed git's phrasing, scoped narrowly to only fire inside the already-failed `*exec.ExitError` branch of a `git config` call.
- **Files modified:** `cmd/alturd/difftool.go`
- **Verification:** `TestInstallDifftoolLocalScopeOutsideRepo` and `TestGitConfigGetLocalScopeOutsideRepo` both pass; manual phase-gate check confirmed the real git binary's message is correctly caught.
- **Committed in:** `15e0cab` (Task 1 GREEN commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — bugs found via failing tests before any implementation was assumed correct)
**Impact on plan:** Both deviations were caught by the plan's own TDD RED/GREEN discipline before Task 2 began — Task 2's subprocess integration tests exercise the fixed `gitConfigRun` and pass cleanly with no further changes needed to the detection logic. No scope change: the four-key contract, D-09 defaults, and D-10 idempotency/`--force` semantics are implemented exactly as specified.

## Issues Encountered

None beyond the two Rule 1 auto-fixes documented above — both were caught immediately by the TDD RED phase (tests failed with a clear diagnostic exit code / message before any implementation code was trusted) rather than surfacing later as silent misbehavior.

## User Setup Required

None — no external service configuration required. `alturd install-difftool` is itself the user-facing setup step this plan implements; a real user still needs to run it once (documented behavior, not a gap).

## Next Phase Readiness

- **Phase 4 is now complete.** All four requirements addressed by this phase's plans (THEME-01, CONFIG-01/02, DIFFTOOL-01/02/03, DIST-01/02/03) have landed across 04-01 through 04-04.
- `go build ./...`, `go vet ./...`, and `go test ./...` are all green with zero regressions across Phases 1-4.
- The published `difftool.<name>.cmd` contract (`alturd --difftool-local "$LOCAL" --difftool-remote "$REMOTE" --difftool-path "$MERGED"`) is now locked and tested byte-for-byte — any future change to the `--difftool-*` flag names in `cmd/alturd/main.go` must be mirrored here, since nothing statically ties the two together beyond this test coverage.
- `difftool.<name>.path` (the optional per-tool absolute-binary-path key) remains explicitly out of scope per 04-RESEARCH.md Open Question 1 / Assumption A2 — `install-difftool` currently assumes `alturd` is on `PATH`. Not a blocker; flagged as a documented future enhancement if a real user's `PATH` doesn't include the install location.
- Not yet human-verified: an actual end-to-end `git difftool -t alturd <file>` invocation with a real terminal after running `install-difftool` for real (this session verified the gitconfig write correctness and the flag-name contract match, but did not launch the interactive TUI via the installed difftool path, mirroring 04-03's same open TUI human-verification gap, D8). Recommended as a phase-level UAT checkpoint.

---
*Phase: 04-config-theming-difftool-distribution*
*Completed: 2026-08-03*

## Self-Check: PASSED

All created/modified files verified present on disk (`cmd/alturd/difftool.go`, `cmd/alturd/difftool_internal_test.go`, `cmd/alturd/installdifftool_test.go`, `internal/git/errors.go`, and this SUMMARY.md); all five commits (`bb5581d`, `15e0cab`, `581794e`, `ab1d93f`, `926f92a`) verified present in `git log`.
