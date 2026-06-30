---
phase: 02-git-layer-cli
verified: 2026-06-29T21:22:29Z
status: passed
score: 4/5 must-haves verified
behavior_unverified: 1
overrides_applied: 0
behavior_unverified_items:

  - truth: "Running `alturd` outside a git repo exits 1 with a single-line error; running it when git is not on PATH exits 127 with a single-line error"
    test: "Run the built alturd binary from a directory that is not a git repo; separately run it with a PATH that excludes git"
    expected: "Non-repo run exits 1 with the message 'not a git repository (run alturd inside a git working tree)'; no-git run exits 127 with the message 'git: command not found (is git installed and on PATH?)'"
    why_human: "No integration test exercises these two error paths through the compiled binary. Unit tests (TestRunnerExitCodes) verify the ExitCodeError sentinel values and errors.As routing via a fakeRunner, but the actual ExecRunner subprocess path — detecting exec.ErrNotFound vs ExitCode()==128 from a real git invocation — has not been exercised end-to-end."
human_verification:

  - test: "Run `./alturd` from a temp directory outside any git repo with XDG_STATE_HOME set to a temp dir"
    expected: "Process exits with code 1; stderr contains exactly one line: 'not a git repository (run alturd inside a git working tree)'; no alturd.log created"
    why_human: "TestSmokeRunInRepoExitsZero only tests the success path. No integration test invokes the binary in a non-repo context to exercise the ExecRunner exit-128 → ErrNotGitRepo → main() exit-1 chain."

  - test: "Run `./alturd` with PATH set to a directory containing no git binary"
    expected: "Process exits with code 127; stderr contains exactly one line: 'git: command not found (is git installed and on PATH?)'; no alturd.log created"
    why_human: "No integration test exercises the exec.ErrNotFound → ErrGitNotFound → main() exit-127 chain through the compiled binary."
---

# Phase 02: Git Layer + CLI Verification Report

**Phase Goal:** Build the git subprocess adapter, log infrastructure, and CLI entry point that wire Phase 1's diff model into a working `alturd` binary. Running `alturd` (no args) inside any git repository must invoke `git diff`, parse the output, and render side-by-side ANSI rows to stdout with a single-binary, zero-runtime-dependency CLI.
**Verified:** 2026-06-29T21:22:29Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `alturd --version` and `alturd --help` exit 0 and produce expected output; no log file or side effect is created | VERIFIED | TestVersionExitsZeroNoLog and TestHelpExitsZeroNoLog pass; each asserts exit 0 AND `os.IsNotExist` on `<stateDir>/alturd/alturd.log` under an isolated `XDG_STATE_HOME` |
| 2 | Running `alturd` outside a git repo exits 1 with a single-line error; running it when git is not on PATH exits 127 with a single-line error | PRESENT_BEHAVIOR_UNVERIFIED | ExitCodeError sentinels carry correct codes (127/1); `errors.As(*git.ExitCodeError)` routing in main() is present and wired; `fmt.Fprintln(os.Stderr, exitErr.Msg)` then `os.Exit(exitErr.Code)` confirmed. Unit tests (TestRunnerExitCodes) verify sentinel codes via fakeRunner. No integration test runs the binary outside a repo or with git absent from PATH. |
| 3 | All six invocation forms — no args, `<ref>`, `<ref1>..<ref2>`, `<ref1>...<ref2>`, `<ref1> <ref2>`, and `-- <paths>` filtering — each produce the correct `git diff` subprocess command | VERIFIED | TestParseRefArgs passes 7/7 subtests covering all six forms; wiring in main.go: `append([]string{"diff"}, git.ParseRefArgs(args, cmd.ArgsLenAtDash())...)` is correct; smoke test exercises the no-args path end-to-end |
| 4 | Git subprocess output is CRLF-normalized to LF immediately after `cmd.Output()` on all platforms before it reaches the diff parser | VERIFIED | `NormalizeCRLF` called at runner.go:61 immediately after `cmd.Output()`; TestCRLFNormalization verifies `\r\n` → `\n` and LF-only content unchanged |
| 5 | The log file is written to `$XDG_STATE_HOME/alturd/alturd.log` (never to stderr) and is truncated to 1 MB on startup if it exceeds that size | VERIFIED | TestInit passes 3/3 subtests: path_under_xdg_state_home (correct path), truncates_over_cap (tail retained, size <= maxLogSize), small_file_untouched (O_APPEND preserves content); `charmlog.SetOutput(f)` sends output to file, no reference to Stderr in log.go |

**Score:** 4/5 truths verified (1 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/git/errors.go` | ExitCodeError type, ErrGitNotFound (127), ErrNotGitRepo (1) | VERIFIED | ExitCodeError{Code, Msg} with Error() returning Msg; both sentinels present with correct codes |
| `internal/git/runner.go` | Runner interface, ExecRunner with CRLF normalization and exit-code mapping | VERIFIED | Runner interface and ExecRunner struct confirmed; argv-form exec.Command; NormalizeCRLF called after cmd.Output(); no `sh -c` |
| `internal/git/args.go` | ParseRefArgs function covering all six invocation forms | VERIFIED | Pure function ParseRefArgs(args, dashIdx) exported; re-inserts `--` separator for path args |
| `internal/git/runner_test.go` | fakeRunner-based unit tests, external package | VERIFIED | package git_test; fakeRunner defined; TestRunnerExitCodes, TestExitCodeErrorMessage, TestFakeRunnerCapturesArgs, TestCRLFNormalization all pass |
| `internal/git/args_test.go` | 7-case table test for ParseRefArgs, external package | VERIFIED | package git_test; TestParseRefArgs with 7 subtests using slices.Equal; all pass |
| `internal/log/log.go` | applog.Init with XDG path, tail truncation, charmbracelet/log redirect | VERIFIED | package applog; Init resolves XDG path via xdg.StateFile; truncateLog keeps tail (ReadFile+WriteFile); charmlog.SetOutput(f); no os.Truncate() call |
| `internal/log/log_test.go` | White-box tests in package applog | VERIFIED | package applog; TestInit 3/3 pass; uses t.Setenv + xdg.Reload for isolation; t.TempDir prevents real user state dir writes |
| `go.mod (xdg + charmbracelet/log added)` | github.com/adrg/xdg v0.5.3, github.com/charmbracelet/log v1.0.0 | VERIFIED | Both require lines confirmed in go.mod |
| `cmd/alturd/main.go` | cobra root command wiring all packages; SilenceErrors/SilenceUsage; applog.Init first in RunE | VERIFIED | SilenceErrors:true, SilenceUsage:true confirmed; applog.Init at line 43 (first statement in run()); cobra.ArbitraryArgs; var version="dev"; no subcommands |
| `cmd/alturd/main_test.go` | Integration tests building binary as subprocess | VERIFIED | TestMain builds binary once; TestVersionExitsZeroNoLog, TestHelpExitsZeroNoLog, TestSmokeRunInRepoExitsZero all pass |
| `go.mod (cobra + x/term added)` | github.com/spf13/cobra v1.10.2, golang.org/x/term v0.44.0 | VERIFIED | Both require lines confirmed in go.mod |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `cmd/alturd/main.go run()` | `internal/git/args.go ParseRefArgs` | `git.ParseRefArgs(args, cmd.ArgsLenAtDash())` at main.go:50 | WIRED | `append([]string{"diff"}, git.ParseRefArgs(...))` confirms "diff" is prepended by caller |
| `cmd/alturd/main.go run()` | `internal/git/runner.go ExecRunner.Run` | `git.ExecRunner{}.Run(gitArgs)` at main.go:53 | WIRED | Returns io.Reader or typed *ExitCodeError |
| `cmd/alturd/main.go run()` | `internal/diff/parse.go diff.Parse` | `diff.Parse(reader)` at main.go:61 | WIRED | io.Reader from ExecRunner feeds directly into diff.Parse |
| `cmd/alturd/main.go run()` | `internal/diff/render.go diff.Render` | `diff.Render(file, width)` at main.go:71 in loop | WIRED | Iterates all files; rows written via `fmt.Fprintln(os.Stdout, row)` |
| `cmd/alturd/main.go run()` | `internal/log/log.go applog.Init` | `applog.Init()` at main.go:43 — FIRST statement | WIRED | defer logFile.Close() on success; non-fatal on error |
| `cmd/alturd/main.go main()` | `internal/git/errors.go ExitCodeError` | `errors.As(err, &exitErr)` at main.go:98 | WIRED | Routes to `os.Exit(exitErr.Code)` with `fmt.Fprintln(os.Stderr, exitErr.Msg)` |
| `internal/git/runner.go ExecRunner.Run` | `internal/git/errors.go ErrGitNotFound / ErrNotGitRepo` | `errors.As(err, &execErr)` and `errors.As(err, &exitErr)` at runner.go:48,52 | WIRED | exec.ErrNotFound → ErrGitNotFound; ExitCode()==128 → ErrNotGitRepo |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|-------------------|--------|
| `cmd/alturd/main.go` (run function) | `files []*gitdiff.File` | `diff.Parse(reader)` where reader comes from `git.ExecRunner{}.Run(gitArgs)` which shells out to `git diff` | Yes — ExecRunner executes real git subprocess; smoke test (TestSmokeRunInRepoExitsZero) verifies this returns exit 0 | FLOWING |
| `cmd/alturd/main.go` (run function) | `rows []string` per file | `diff.Render(file, width)` called per file from diff.Parse output | Yes — diff.Render produces ANSI rows from real parsed diff data (Phase 1 verified) | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `go test ./internal/git/... -run TestParseRefArgs` — all 6 invocation forms | `go test ./internal/git/... -run TestParseRefArgs -v` | 7/7 subtests PASS | PASS |
| `go test ./internal/git/... -run 'TestRunner\|TestExitCode\|TestCRLF'` — error codes and normalization | `go test ./internal/git/... -run 'TestRunner\|TestExitCode\|TestCRLF' -v` | 5/5 tests PASS | PASS |
| `go test ./internal/log/... -run TestInit` — XDG path, tail truncation, small file | `go test ./internal/log/... -run TestInit -v` | 3/3 subtests PASS | PASS |
| `go test ./cmd/alturd/... -v` — --version/--help no-log + smoke run | `go test ./cmd/alturd/... -v` | 3/3 tests PASS (TestVersionExitsZeroNoLog, TestHelpExitsZeroNoLog, TestSmokeRunInRepoExitsZero) | PASS |
| Full module suite | `go test ./...` | 4/4 packages PASS | PASS |
| Binary build | `go build ./cmd/alturd` | Exit 0, binary produced | PASS |
| go vet | `go vet ./internal/git/... ./internal/log/... ./cmd/alturd/...` | Clean | PASS |
| No shell injection | `grep "sh -c" internal/git/runner.go` | No matches | PASS |
| No os.Truncate call | `grep -n "os\.Truncate(" internal/log/log.go` | No call (only in comment) | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| GIT-01 | 02-01-PLAN.md | User can run `alturd` in a git repo with no args to diff working tree vs HEAD | SATISFIED | ParseRefArgs([], -1) returns []; "diff" prepended → ["diff"] passed to git; smoke test exits 0 |
| GIT-02 | 02-01-PLAN.md | User can run `alturd <ref>`, `alturd <ref1>..<ref2>`, `alturd <ref1>...<ref2>`, `alturd <ref1> <ref2>` | SATISFIED | TestParseRefArgs covers single_ref, two_dot_range, three_dot_range, two_args — all 4 forms pass |
| GIT-03 | 02-01-PLAN.md | User can run `alturd -- <paths>` to filter diff to specific paths | SATISFIED | TestParseRefArgs covers paths_only and ref_plus_paths; `--` separator re-inserted correctly |
| GIT-04 | 02-03-PLAN.md | `alturd --version` and `alturd --help` exit cleanly without creating log files or side effects | SATISFIED | TestVersionExitsZeroNoLog and TestHelpExitsZeroNoLog both assert exit 0 AND os.IsNotExist on log path |
| GIT-05 | 02-03-PLAN.md | User sees a clear single-line error message when not in a git repo (exit 1) or git not on PATH (exit 127) | NEEDS HUMAN | Error types carry correct codes; main() routing is wired; no integration test runs binary in error paths |
| LOG-01 | 02-02-PLAN.md | Log file written under `$XDG_STATE_HOME/alturd/alturd.log` (never to stderr); truncated at 1MB cap on startup | SATISFIED | TestInit 3/3 pass; charmlog.SetOutput(f) confirmed; no Stderr reference in log.go |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | No TBD/FIXME/XXX markers | — | Clean |
| — | — | No TODO/HACK/PLACEHOLDER markers in implementation files | — | Clean |
| — | — | No stub returns (return null / return []) | — | Clean |
| — | — | No `sh -c` invocation in runner.go | — | Security gate passed |
| — | — | No `os.Truncate()` call in log.go | — | Tail-retention contract preserved |

### Human Verification Required

#### 1. Not-In-Git-Repo Error Path (GIT-05 partial)

**Test:** Build the binary with `go build ./cmd/alturd`. Create a temp directory outside any git repo. Run `./alturd` from that directory with `XDG_STATE_HOME` set to a separate temp dir.

**Expected:** Process exits with code 1. Stderr contains exactly one line: `not a git repository (run alturd inside a git working tree)`. No `alturd.log` file is created under the XDG_STATE_HOME temp dir.

**Why human:** No integration test exercises this path through the compiled binary. Unit tests verify the ExitCodeError sentinel values via a fakeRunner, but the actual ExecRunner subprocess path (detecting git's exit code 128 and mapping to ErrNotGitRepo via `errors.As(*exec.ExitError)` + `exitErr.ExitCode() == 128`) has not been run end-to-end.

#### 2. Git-Not-On-PATH Error Path (GIT-05 partial)

**Test:** Run the binary with `PATH` set to a directory that does not contain a `git` binary (e.g., `PATH=/tmp ./alturd`).

**Expected:** Process exits with code 127. Stderr contains exactly one line: `git: command not found (is git installed and on PATH?)`. No `alturd.log` file is created.

**Why human:** The exec.ErrNotFound detection path (`errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound)`) has not been exercised through the compiled binary. Only the fakeRunner-based unit test (TestRunnerExitCodes/git_not_found) covers this branch.

### Gaps Summary

No gaps found. All artifacts exist, are substantive (not stubs), and are correctly wired. The full module test suite passes (`go test ./...`). The only item requiring human validation is the behavioral end-to-end exercise of the two error exit-code paths (GIT-05), which no integration test currently covers. The underlying code is correct and unit-tested via injected fakeRunners per the plan's D-05 constraint.

---

_Verified: 2026-06-29T21:22:29Z_
_Verifier: Claude (gsd-verifier)_
