# Phase 2: Git Layer + CLI - Context

**Gathered:** 2026-06-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Build `cmd/alturd/main.go` (cobra-based CLI entrypoint) and `internal/git/` (subprocess layer) so that `alturd <args>` parses the full ref grammar, executes the correct `git diff` subprocess, feeds its CRLF-normalized output to `internal/diff.Parse()`, renders the result to stdout with ANSI colors, and returns clean error codes for invalid invocations — all without a running TUI. The log file is initialized on startup (after flag parsing). Phase 3 replaces the stdout render path with the bubbletea TUI.

</domain>

<decisions>
## Implementation Decisions

### CLI Framework

- **D-01:** Use `cobra` as the CLI framework. It handles structured subcommand dispatch cleanly for Phase 4's `install-difftool` subcommand. Add it as a dependency in Phase 2.
- **D-02:** Phase 2 root command has no subcommands — `install-difftool` is added in Phase 4. Cobra's help output will show only Phase 2 flags (no stubs).
- **D-03:** Use cobra's built-in `--version` flag. Set `rootCmd.Version = "dev"` in Phase 2; goreleaser injects the real version string via `-ldflags` in Phase 4.

### Subprocess Test Design

- **D-04:** `internal/git` exposes a `Runner` interface for dependency injection: `type Runner interface { Run(args []string) (io.Reader, error) }`. The real implementation wraps `exec.Command("git", args...)`. Tests inject a fake Runner that captures the args slice without spawning a real subprocess.
- **D-05:** Error path tests (no-repo, no-git on PATH) also use the injected fake Runner — fake returns appropriate errors with the correct exit codes. No real git subprocess in any unit test.

### Phase 2 Binary Output

- **D-06:** On success, `alturd <args>` calls `diff.Render()` and writes the resulting `[]string` rows to `os.Stdout` as ANSI text (no TUI, no pager, no alternate screen). This gives a usable intermediate artifact while Phase 3 isn't built yet.
- **D-07:** Terminal width detection uses `golang.org/x/term` (or equivalent) to read `os.Stdout`'s column count. Falls back to 160 columns if stdout is not a terminal (e.g., piped output). Phase 3 replaces this with `tea.WindowSizeMsg`.

### main Package Layout

- **D-08:** Binary entrypoint lives at `cmd/alturd/main.go`. Go module: `github.com/alturd/alturd`. Build command: `go build ./cmd/alturd`. goreleaser will be configured to build from `cmd/alturd/` in Phase 4.

### Locked Behaviors (from ROADMAP / REQUIREMENTS — do not re-litigate)

- **D-09:** CRLF normalization happens immediately after `cmd.Output()` in the Runner implementation, before the bytes reach `diff.Parse()`.
- **D-10:** `--version` and `--help` must produce no log file and no side effects (exit 0 before log initialization).
- **D-11:** Exit codes: 1 for not-in-git-repo; 127 for git not on PATH; errors go to stderr as a single-line message.
- **D-12:** Log file at `$XDG_STATE_HOME/alturd/alturd.log` (never stderr); truncated to 1 MB on startup if it exceeds that size. Log initialization happens after flag parsing (so `--help`/`--version` do not trigger it).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & Scope

- `.planning/REQUIREMENTS.md` §Git Layer — GIT-01 through GIT-05; exact invocation forms and expected behavior
- `.planning/REQUIREMENTS.md` §Logging — LOG-01; log file path, size cap, truncation behavior
- `.planning/ROADMAP.md` §Phase 2 — Success criteria (5 items); exact exit code requirements; CRLF normalization spec; command-capture test requirement

### Library Choices & Architecture

- `.claude/CLAUDE.md` §Technology Stack — Full library selection table; what NOT to use; cobra is not listed but stdlib alternatives are assessed
- `.claude/CLAUDE.md` §Stack Patterns — XDG path resolution via `github.com/adrg/xdg` (add this dep for LOG-01 and CONFIG-01 in Phase 4)

### Phase 1 Integration Points

- `internal/diff/parse.go` — `Parse(io.Reader) ([]*gitdiff.File, error)` — the git layer feeds subprocess output to this
- `internal/diff/render.go` — `Render(files []*gitdiff.File, width int) []string` — Phase 2 calls this to produce stdout output

### Python Reference Implementation

- The Python implementation at v1.1 is the behavioral reference for CLI grammar, exit codes, and error message wording. Verify argument parsing parity against it for GIT-02 (ref forms) and GIT-05 (error messages).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- `internal/diff.Parse(io.Reader)` — accepts an `io.Reader`; the `Runner.Run()` return value feeds directly into this (designed with this in mind in Phase 1)
- `internal/diff.Render(files, width)` — `[]string` output; Phase 2 writes each row + `\n` to stdout
- `go.mod` — already at go 1.25 with chroma, go-gitdiff, go-diff declared; cobra and `golang.org/x/term` need to be added

### Established Patterns

- Phase 1 used table-driven tests with `testdata/` fixtures; Phase 2 tests should follow the same pattern (fake Runner captures args, compares against expected `[]string{"-diff", "--unified=…", ...}`)
- `CGO_ENABLED=0` is a project-wide constraint; any dependency added must be pure Go

### Integration Points

- Phase 3 (TUI) will replace the `os.Stdout` render path — it calls `internal/git` to get `[]*gitdiff.File`, then feeds them into bubbletea model state. Phase 2's stdout path is intentionally temporary.
- Phase 4 adds `install-difftool` as a cobra subcommand on the same root command built here; also adds `--config` flag and `github.com/adrg/xdg` for config path resolution.

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

*Phase: 2-Git Layer + CLI*
*Context gathered: 2026-06-29*
