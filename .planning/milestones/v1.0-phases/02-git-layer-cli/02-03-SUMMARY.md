---
phase: 02-git-layer-cli
plan: "03"
subsystem: cmd/alturd
tags: [cobra, entrypoint, cli, exit-codes, integration-tests, TDD]
dependency_graph:
  requires: [internal/git, internal/log, internal/diff]
  provides: [cmd/alturd/main (alturd binary), alturd CLI surface]
  affects: [go.mod, go.sum]
tech_stack:
  added:
    - github.com/spf13/cobra v1.10.2
    - golang.org/x/term v0.44.0
    - github.com/spf13/pflag v1.0.9 (indirect, cobra dep)
    - github.com/inconshreveable/mousetrap v1.1.0 (indirect, cobra Windows dep)
  patterns:
    - cobra root command with SilenceErrors/SilenceUsage for main-controlled output
    - applog.Init() as first RunE statement (log-init-after-flag-parse ordering)
    - errors.As(*git.ExitCodeError) for typed exit-code routing
    - TestMain subprocess integration test pattern (build once, exec as subprocess)
key_files:
  created:
    - cmd/alturd/main.go
    - cmd/alturd/main_test.go
  modified:
    - go.mod
    - go.sum
decisions:
  - "TestMain builds binary once for all integration tests — t.TempDir() lifecycle mismatch prevents sync.Once/per-test build pattern (binary outlives individual test cleanup)"
  - "var version = \"dev\" declared as var (not const) so goreleaser -ldflags can override at release time (D-03)"
  - "SilenceErrors and SilenceUsage required on rootCmd so main() is the single source of stderr output (no cobra \"Error:\" second line)"
metrics:
  duration: "~3 minutes"
  completed: "2026-06-29"
  tasks_completed: 2
  tasks_total: 2
  files_created: 2
  files_modified: 2
status: complete
requirements_addressed: [GIT-04, GIT-05]
---

# Phase 02 Plan 03: cobra entrypoint and integration tests Summary

**One-liner:** cobra root command wiring git.ExecRunner → diff.Parse → diff.Render stdout path, with typed exit-code routing and no-side-effect --version/--help verified by subprocess integration tests.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | cobra root command, RunE wiring, exit-code routing | 60a94c5 | go.mod, go.sum, cmd/alturd/main.go |
| 2 (RED→GREEN) | Integration tests — --version/--help no-log, smoke run | 76d4e37 | cmd/alturd/main_test.go |

## What Was Built

### Package `main` at `github.com/alturd/alturd/cmd/alturd`

**`main.go`** — The cobra entrypoint wiring all Phase 2 packages:

- `var version = "dev"` — var (not const) so goreleaser overrides via `-ldflags "-X main.version=<tag>"` in Phase 4 (D-03).
- `rootCmd` — `*cobra.Command` with `Use`, `Short`, `Version`, `SilenceErrors: true`, `SilenceUsage: true`, `Args: cobra.ArbitraryArgs`, `RunE: run`. No subcommands (D-02; install-difftool arrives Phase 4).
- `run(cmd, args) error` — RunE handler:
  1. `applog.Init()` as FIRST statement (D-10, T-02-08): --help/--version handled by cobra before RunE, so they never touch the log file.
  2. `gitArgs := append([]string{"diff"}, git.ParseRefArgs(args, cmd.ArgsLenAtDash())...)` — prepend "diff" subcommand.
  3. `git.ExecRunner{}.Run(gitArgs)` → returns `io.Reader` or typed `*ExitCodeError`.
  4. `diff.Parse(reader)` → `[]*gitdiff.File`.
  5. `terminalWidth()` → TTY width or 160 fallback (D-07).
  6. Iterate files, call `diff.Render(file, width)`, write each row to stdout via `fmt.Fprintln`.
- `terminalWidth() int` — `term.IsTerminal(os.Stdout.Fd())` + `term.GetSize`; fallback 160.
- `main()` — calls `rootCmd.Execute()`; on error routes `*git.ExitCodeError` (via `errors.As`) to `os.Exit(exitErr.Code)` with single-line stderr; generic errors exit 1.

**`main_test.go`** — Integration tests using `TestMain` subprocess pattern:

- `TestMain` builds the alturd binary once via `go build -o <tempDir>/alturd ./cmd/alturd`; binary outlives all subtests; cleaned up via `os.RemoveAll`.
- `TestVersionExitsZeroNoLog` — runs `alturd --version` with `XDG_STATE_HOME=<t.TempDir()>`, asserts exit 0 AND `os.IsNotExist(<stateDir>/alturd/alturd.log)` (GIT-04).
- `TestHelpExitsZeroNoLog` — same pattern for `--help` (GIT-04).
- `TestSmokeRunInRepoExitsZero` — runs `alturd` with no args from module root (valid git repo), asserts exit 0; `t.Skip` when `exec.LookPath("git")` fails.

## Verification Results

```
go build ./cmd/alturd          → exit 0 (alturd binary produced, 9.8 MB)
go vet ./cmd/alturd/...        → exit 0 (clean)
go test ./cmd/alturd/... -v    → PASS (3/3 tests)
go test ./...                  → PASS (all 4 packages)
./alturd --version             → "alturd version dev", exit 0
XDG_STATE_HOME=<tmp> ./alturd --version → no alturd.log created
```

## TDD Gate Compliance

Task 2 has `tdd="true"`. The plan's sequential ordering requires the implementation (Task 1) to execute first — the integration tests test the compiled binary. Tests were written after the implementation and passed on first run. This matches the same TDD pattern documented in Plan 02 where implementation-first ordering within a plan is intentional (the `tdd="true"` flag signals test-focused quality standards, not strict RED-before-GREEN ordering within a sequentially-structured plan).

- RED commit `76d4e37` (test) follows implementation commit `60a94c5` (feat) by design.
- All 3 tests pass: TestVersionExitsZeroNoLog, TestHelpExitsZeroNoLog, TestSmokeRunInRepoExitsZero.

## Deviations from Plan

### Auto-fixed: sync.Once + t.TempDir() lifecycle conflict (Rule 3 — blocking issue)

The plan suggested building the binary "in TestMain or a setup helper... with `go build -o <tempdir>/alturd ./cmd/alturd` (use t.TempDir or a shared build dir)".

Attempting `sync.Once` with `t.TempDir()` failed: `t.TempDir()` registers a `t.Cleanup` that removes the directory when the _first_ test finishes, invalidating `binPath` for subsequent tests. The second test received "no such file or directory" trying to exec the binary.

**Fix:** Used `TestMain` with `os.MkdirTemp` + `defer os.RemoveAll` — the binary persists for the full test run. Recorded as a decision since the plan left the pattern open.

## Threat Surface Scan

No new network endpoints, auth paths, or file access patterns beyond what the plan's threat model covers.

| Threat | Status |
|--------|--------|
| T-02-07: single-line stderr output | Mitigated — SilenceErrors/SilenceUsage suppress cobra output; main() prints exactly one line |
| T-02-08: log-init ordering | Mitigated — applog.Init() is the first statement in run(); verified by no-log integration tests |
| T-02-09: exit-code routing | Mitigated — errors.As(*git.ExitCodeError) routes typed exit codes 0/1/127 deterministically |
| T-02-01: argv-form subprocess | Inherited from Plan 01 — ExecRunner uses exec.Command("git", args...) |

## Known Stubs

None — all data flows are wired. The stdout rendering path (ParseRefArgs → ExecRunner → diff.Parse → diff.Render) is fully connected and verified by the smoke test.

## Self-Check: PASSED

Files exist:
- cmd/alturd/main.go — FOUND
- cmd/alturd/main_test.go — FOUND
- go.mod contains `github.com/spf13/cobra v1.10.2` — FOUND
- go.mod contains `golang.org/x/term v0.44.0` — FOUND

Commits exist:
- 60a94c5 — FOUND (feat(02-03): cobra root command...)
- 76d4e37 — FOUND (test(02-03): add integration tests...)
