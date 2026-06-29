# Phase 2: Git Layer + CLI - Research

**Researched:** 2026-06-29
**Domain:** Go CLI entrypoint (cobra), git subprocess invocation, ref-grammar parsing, error-code mapping, XDG log file initialization
**Confidence:** MEDIUM

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**D-01:** Use `cobra` as the CLI framework. Add it as a dependency in Phase 2.
**D-02:** Phase 2 root command has no subcommands — `install-difftool` is added in Phase 4. Cobra's help output shows only Phase 2 flags (no stubs).
**D-03:** Use cobra's built-in `--version` flag. Set `rootCmd.Version = "dev"` in Phase 2; goreleaser injects the real version string via `-ldflags` in Phase 4.
**D-04:** `internal/git` exposes a `Runner` interface: `type Runner interface { Run(args []string) (io.Reader, error) }`. The real implementation wraps `exec.Command("git", args...)`. Tests inject a fake Runner.
**D-05:** Error path tests (no-repo, no-git on PATH) use the injected fake Runner — no real git subprocess in any unit test.
**D-06:** On success, `alturd <args>` calls `diff.Render()` and writes the resulting `[]string` rows to `os.Stdout` as ANSI text (no TUI, no pager, no alternate screen).
**D-07:** Terminal width detection uses `golang.org/x/term` (or equivalent). Falls back to 160 columns if stdout is not a terminal. Phase 3 replaces this with `tea.WindowSizeMsg`.
**D-08:** Binary entrypoint lives at `cmd/alturd/main.go`. Go module: `github.com/alturd/alturd`. Build command: `go build ./cmd/alturd`.
**D-09:** CRLF normalization happens immediately after `cmd.Output()` in the Runner implementation, before bytes reach `diff.Parse()`.
**D-10:** `--version` and `--help` must produce no log file and no side effects (exit 0 before log initialization).
**D-11:** Exit codes: 1 for not-in-git-repo; 127 for git not on PATH; errors go to stderr as a single-line message.
**D-12:** Log file at `$XDG_STATE_HOME/alturd/alturd.log` (never stderr); truncated to 1 MB on startup if it exceeds that size. Log initialization happens after flag parsing.

### Claude's Discretion

None specified — all choices locked via D-01 through D-12 and CLAUDE.md.

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GIT-01 | User can run `alturd` in a git repo with no args to diff working tree vs HEAD | Runner.Run([]string{}) → `git diff`; no-args cobra command with cobra.ArbitraryArgs |
| GIT-02 | User can run `alturd <ref>`, `<ref1>..<ref2>`, `<ref1>...<ref2>`, `<ref1> <ref2>` to diff ranges | parseRefArgs() maps 1-arg and 2-arg forms to the correct `git diff` argument; `..` and `...` detected by string contains |
| GIT-03 | User can run `alturd -- <paths>` to filter diff to specific paths | cobra's ArgsLenAtDash() splits ref args from path args; paths appended after `--` in the git command |
| GIT-04 | `alturd --version` and `alturd --help` exit cleanly without creating log files or side effects | cobra's built-in --version/--help run before RunE; log init deferred to RunE body after these flags are consumed |
| GIT-05 | User sees a clear single-line error message when not in a git repo (exit 1) or git not on PATH (exit 127) | errors.Is(err,exec.ErrNotFound) → 127; ExitError.ExitCode()==128 → 1; SilenceErrors+SilenceUsage; custom ExitCodeError |
| LOG-01 | Log file written under `$XDG_STATE_HOME/alturd/alturd.log` (never to stderr); truncated at 1MB cap on startup | xdg.StateFile("alturd/alturd.log") returns correct cross-platform path; charmbracelet/log SetOutput(f) redirects to file |
</phase_requirements>

---

## Summary

Phase 2 layers a CLI entrypoint and git subprocess adapter on top of Phase 1's pure diff library. The work divides cleanly into four components: (1) `cmd/alturd/main.go` with a cobra root command, (2) `internal/git/runner.go` with the `Runner` interface and real subprocess implementation, (3) `internal/git/args.go` with the ref-grammar parser that maps CLI arguments to `git diff` argument slices, and (4) `internal/log/log.go` with XDG-based log file initialization.

The hardest design problem in Phase 2 is exit-code routing: cobra's `Execute()` returns a generic `error` with no exit-code metadata, but the requirements demand distinct codes for distinct failure modes (1 for not-in-repo, 127 for git-not-on-PATH). The clean solution is a sentinel error type `ExitCodeError{Code int, Msg string}` — `RunE` wraps all failures in it, `main()` type-asserts on it, prints `Msg` to stderr, and calls `os.Exit(Code)` directly. This avoids cobra's auto "Error:" prefix (which would appear as a second line) while keeping the exit code handling explicit and testable.

The second tricky area is the `--` path-filter separator. Cobra treats `--` as a flag terminator and provides `cmd.ArgsLenAtDash()` to find the split point in the args slice. Phase 2 must use this to correctly separate ref arguments from path-filter arguments and pass both through to `git diff` in the right order.

**Primary recommendation:** Build in three waves — (1) `internal/git` Runner + args parser with full unit tests using fake Runner, (2) `internal/log` initializer, (3) `cmd/alturd/main.go` integration — matching Phase 1's plan structure. The Runner interface with dependency injection is the cornerstone; every other component depends on having it right.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CLI flag/arg parsing | `cmd/alturd` (cobra root cmd) | — | cobra owns the UX surface; no business logic here |
| Ref-grammar interpretation | `internal/git` (args.go) | — | Pure function mapping `[]string` args → git diff args; testable in isolation |
| Git subprocess execution | `internal/git` (runner.go) | — | Runner interface is the subprocess chokepoint; all tests mock this boundary |
| CRLF normalization | `internal/git` (runner.go) | — | Happens immediately after `cmd.Output()`, before bytes leave internal/git |
| Exit-code mapping | `cmd/alturd/main.go` | `internal/git` (error types) | git layer defines sentinel errors with codes; main() calls os.Exit |
| Log file initialization | `internal/log` | — | XDG path resolution + charmbracelet/log setup; no other layer touches logging |
| Width detection | `cmd/alturd/main.go` | — | `golang.org/x/term` at startup; passed as parameter to diff.Render |
| Diff rendering (stdout) | `cmd/alturd/main.go` | `internal/diff` | Phase 2's temporary stdout path; Phase 3 replaces with TUI |

---

## Standard Stack

### Core (Phase 2 additions)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/spf13/cobra` | v1.10.2 | CLI framework — flags, subcommands, --help/--version | D-01; the de-facto standard Go CLI framework; Phase 4 adds `install-difftool` subcommand on same root [CITED: proxy.golang.org] |
| `golang.org/x/term` | v0.44.0 | Terminal width detection (`GetSize`) and `IsTerminal` check | D-07; official Go extended library; CGO-free on all platforms [CITED: proxy.golang.org] |
| `github.com/adrg/xdg` | v0.5.3 | XDG base directory resolution — `StateFile("alturd/alturd.log")` for LOG-01 | Listed in CLAUDE.md §Supporting Libraries; handles Linux/macOS/Windows differences [CITED: proxy.golang.org] |
| `github.com/charmbracelet/log` | v1.0.0 | Structured logging to file (never stderr) | Listed in CLAUDE.md §Supporting Libraries; `log.SetOutput(f)` redirects all output to file [CITED: proxy.golang.org] |

### Already Present (from Phase 1, unchanged)

| Library | Version | Role in Phase 2 |
|---------|---------|-----------------|
| `github.com/bluekeyes/go-gitdiff` | v0.8.1 | `diff.Parse()` consumes Runner output |
| `github.com/alecthomas/chroma/v2` | v2.27.0 | `diff.Render()` called from main |
| `github.com/sergi/go-diff` | v1.4.0 | Used inside `diff.Render()` |

### Alternatives Considered

| Recommended | Alternative | Why Not |
|-------------|-------------|---------|
| `github.com/spf13/cobra` | `flag` (stdlib) | Locked by D-01; cobra needed for Phase 4 subcommand dispatch |
| `golang.org/x/term` | `github.com/mattn/go-isatty` + `golang.org/x/sys` | x/term is a single official package that combines IsTerminal + GetSize; no extra dependency |
| `github.com/charmbracelet/log` | `log/slog` (stdlib) | charmbracelet/log is listed in CLAUDE.md; slog is also fine but charmbracelet/log is project-selected |

**Installation:**
```bash
go get github.com/spf13/cobra@v1.10.2
go get golang.org/x/term@v0.44.0
go get github.com/adrg/xdg@v0.5.3
go get github.com/charmbracelet/log@v1.0.0
go mod tidy
```

---

## Package Legitimacy Audit

> Go module proxy used for verification; `gsd-tools package-legitimacy` supports npm/PyPI/crates only — Go packages verified via `proxy.golang.org` and CLAUDE.md authority.

| Package | Registry | Age | Origin | Source Repo | Verdict | Disposition |
|---------|----------|-----|--------|-------------|---------|-------------|
| `github.com/spf13/cobra` | proxy.golang.org | v1.10.2 Dec 2025 | github.com/spf13/cobra | github.com/spf13/cobra | OK | Approved — project decision D-01 [CITED: proxy.golang.org] |
| `golang.org/x/term` | proxy.golang.org | v0.44.0 Jun 2026 | go.googlesource.com/term | golang.org/x/term | OK | Approved — official Go extended lib [CITED: proxy.golang.org] |
| `github.com/adrg/xdg` | proxy.golang.org | v0.5.3 Oct 2024 | github.com/adrg/xdg | github.com/adrg/xdg | OK | Approved — listed in CLAUDE.md [CITED: proxy.golang.org] |
| `github.com/charmbracelet/log` | proxy.golang.org | v1.0.0 Mar 2026 | github.com/charmbracelet/log | github.com/charmbracelet/log | OK | Approved — listed in CLAUDE.md [CITED: proxy.golang.org] |

**Packages removed due to SLOP verdict:** none
**Packages flagged as suspicious SUS:** none

---

## Architecture Patterns

### System Architecture Diagram

```
User invokes: alturd [flags] [refs] [-- paths]
    │
    ▼ os.Args
cobra rootCmd.Execute()
    │  parses flags (--version exits 0 here, before log init)
    │  (--help exits 0 here, before log init)  ← D-10 satisfied
    │
    ▼ RunE(cmd, args []string)
initLog()                          ← D-12: after flag parse
    │  xdg.StateFile("alturd/alturd.log")
    │  if file.Size > 1MB → truncateLog()
    │  charmbracelet/log.SetOutput(logFile)
    │
parseRefArgs(args, cmd.ArgsLenAtDash())
    │  returns gitArgs []string   ← the args after "git diff"
    │
    ▼ gitArgs
Runner.Run(gitArgs)               ← internal/git interface
    │  real: exec.Command("git", "diff", gitArgs...)
    │         cmd.Output() → []byte
    │         bytes.ReplaceAll(out, "\r\n", "\n")  ← D-09
    │  test:  fake captures gitArgs, returns fixture bytes
    │
    ▼ io.Reader (bytes.NewReader of normalized output)
diff.Parse(r)                     ← Phase 1 integration point
    │  []*gitdiff.File
    │
    ▼
diff.Render(file, width)          ← per file, Phase 2 stdout path
    │  []string ANSI rows
    │
    ▼ stdout
fmt.Println(row) per row
    │
    ▼ process exit 0

Error paths:
  exec.ErrNotFound → ExitCodeError{Code:127, Msg:"git not found on PATH"}
  ExitError{ExitCode:128} → ExitCodeError{Code:1, Msg:"not a git repository"}
  ExitCodeError → main() prints Msg to stderr, os.Exit(Code)  ← D-11
```

### Recommended Project Structure

```
alturd/
├── cmd/
│   └── alturd/
│       └── main.go              # cobra root cmd, RunE, main()
├── internal/
│   ├── diff/                    # Phase 1 (unchanged)
│   │   ├── model.go
│   │   ├── parse.go
│   │   ├── align.go
│   │   ├── highlight.go
│   │   └── render.go
│   ├── git/
│   │   ├── runner.go            # Runner interface + ExecRunner (real impl)
│   │   ├── args.go              # parseRefArgs(args []string, dashIdx int) []string
│   │   ├── errors.go            # ExitCodeError type
│   │   ├── runner_test.go       # table tests: fake runner, arg capture
│   │   └── args_test.go         # table tests: all 6 invocation forms
│   └── log/
│       ├── log.go               # InitLog() — xdg path, truncation, SetOutput
│       └── log_test.go          # truncation threshold test
├── go.mod
└── go.sum
```

### Pattern 1: cobra Root Command Setup

**What:** Configure cobra with SilenceErrors+SilenceUsage so the error printing is entirely controlled by main().

**When:** `cmd/alturd/main.go` init.

```go
// Source: pkg.go.dev/github.com/spf13/cobra@v1.10.2 [CITED: pkg.go.dev/github.com/spf13/cobra]
var rootCmd = &cobra.Command{
    Use:          "alturd [ref] [ref1..ref2] [-- paths]",
    Short:        "Side-by-side terminal diff viewer",
    Version:      version, // overridden by goreleaser -ldflags in Phase 4
    SilenceErrors: true,   // suppress cobra's "Error: " prefix
    SilenceUsage:  true,   // suppress usage dump on error
    Args:          cobra.ArbitraryArgs,
    RunE:          run,
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        var exitErr *git.ExitCodeError
        if errors.As(err, &exitErr) {
            fmt.Fprintln(os.Stderr, exitErr.Msg)
            os.Exit(exitErr.Code)
        }
        fmt.Fprintln(os.Stderr, err.Error())
        os.Exit(1)
    }
}
```

### Pattern 2: ExitCodeError — Sentinel Error Type

**What:** A named error type that carries both a human message and an OS exit code. RunE wraps all failure paths in it; main() type-asserts and calls os.Exit.

**When:** `internal/git/errors.go`.

```go
// Source: standard Go error pattern [ASSUMED]
package git

// ExitCodeError is returned by Runner.Run when the error maps to a
// specific OS exit code requirement.
type ExitCodeError struct {
    Code int    // OS exit code to use (1 = no-repo, 127 = git-not-found)
    Msg  string // single-line message printed to stderr
}

func (e *ExitCodeError) Error() string { return e.Msg }

// ErrGitNotFound is returned when git is not on PATH (exit 127).
var ErrGitNotFound = &ExitCodeError{Code: 127, Msg: "git: command not found (is git installed and on PATH?)"}

// ErrNotGitRepo is returned when cwd is not inside a git repository (exit 1).
var ErrNotGitRepo = &ExitCodeError{Code: 1, Msg: "not a git repository (run alturd inside a git working tree)"}
```

### Pattern 3: Runner Interface and ExecRunner

**What:** The subprocess chokepoint. The real implementation normalizes CRLF; the test fake captures args without running git.

**When:** `internal/git/runner.go`.

```go
// Source: D-04, D-09 from CONTEXT.md
package git

import (
    "bytes"
    "errors"
    "io"
    "os/exec"
)

// Runner abstracts git subprocess execution for dependency injection in tests.
type Runner interface {
    Run(args []string) (io.Reader, error)
}

// ExecRunner is the real Runner that invokes exec.Command("git", args...).
type ExecRunner struct{}

func (ExecRunner) Run(args []string) (io.Reader, error) {
    cmd := exec.Command("git", args...)
    out, err := cmd.Output()
    if err != nil {
        // git binary not on PATH
        var execErr *exec.Error
        if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
            return nil, ErrGitNotFound
        }
        // git ran but exited non-zero (e.g., not a git repo → exit 128)
        var exitErr *exec.ExitError
        if errors.As(err, &exitErr) && exitErr.ExitCode() == 128 {
            return nil, ErrNotGitRepo
        }
        return nil, err
    }
    // D-09: normalize CRLF immediately after cmd.Output(), before diff.Parse
    out = bytes.ReplaceAll(out, []byte("\r\n"), []byte("\n"))
    return bytes.NewReader(out), nil
}
```

### Pattern 4: Ref-Grammar Parser

**What:** Maps cobra's `args []string` + `dashIdx int` to the correct `[]string` of git diff arguments.

**When:** `internal/git/args.go`. This is a pure function — trivially table-tested.

```go
// Source: CONTEXT.md GIT-02/GIT-03 + cobra ArgsLenAtDash docs [ASSUMED synthesis]
package git

import "strings"

// ParseRefArgs converts cobra args (after flag parsing) to git diff arguments.
// dashIdx is cmd.ArgsLenAtDash() — the index in args where "--" appeared, or -1.
//
// Invocation forms:
//   alturd                      → git diff
//   alturd <ref>                → git diff <ref>
//   alturd <ref1>..<ref2>       → git diff <ref1>..<ref2>   (single arg)
//   alturd <ref1>...<ref2>      → git diff <ref1>...<ref2>  (single arg)
//   alturd <ref1> <ref2>        → git diff <ref1> <ref2>    (two args)
//   alturd -- <paths>           → git diff -- <paths>
//   alturd <ref> -- <paths>     → git diff <ref> -- <paths>
func ParseRefArgs(args []string, dashIdx int) []string {
    var refs []string
    var paths []string

    if dashIdx >= 0 {
        refs = args[:dashIdx]
        paths = args[dashIdx:] // includes the "--" sentinel as paths[0]
    } else {
        refs = args
    }

    // Build the git diff argument slice.
    // refs is passed as-is; git validates ref syntax.
    result := make([]string, 0, len(refs)+len(paths))
    result = append(result, refs...)
    if len(paths) > 0 {
        result = append(result, "--")
        result = append(result, paths...)
    }
    return result
}
```

Note: cobra's `args` slice after `--` does NOT include the `--` itself. `ArgsLenAtDash()` returns the split point. The `--` must be re-inserted into the git command manually.

### Pattern 5: XDG Log File Initialization

**What:** Resolve the platform-correct log path, truncate if oversized, open for append, and redirect charmbracelet/log output.

**When:** `internal/log/log.go`, called from `RunE` after flag parse (D-10, D-12).

```go
// Source: pkg.go.dev/github.com/adrg/xdg@v0.5.3 [CITED], charmbracelet/log v1.0.0 [CITED]
package applog

import (
    "os"

    charmlog "github.com/charmbracelet/log"
    "github.com/adrg/xdg"
)

const maxLogSize = 1 << 20 // 1 MB

// Init opens the log file at $XDG_STATE_HOME/alturd/alturd.log,
// truncates it if it exceeds 1 MB, and redirects charmbracelet/log output to it.
// Returns the open *os.File so the caller can defer f.Close().
func Init() (*os.File, error) {
    path, err := xdg.StateFile("alturd/alturd.log")
    if err != nil {
        return nil, err
    }
    // Truncate if needed before opening for append.
    if fi, err := os.Stat(path); err == nil && fi.Size() > maxLogSize {
        if err := truncateLog(path); err != nil {
            return nil, err
        }
    }
    f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
    if err != nil {
        return nil, err
    }
    charmlog.SetOutput(f)
    return f, nil
}

// truncateLog keeps the last maxLogSize bytes of the log file.
// This retains recent entries while dropping old ones.
func truncateLog(path string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    if int64(len(data)) > maxLogSize {
        data = data[int64(len(data))-maxLogSize:]
    }
    return os.WriteFile(path, data, 0600)
}
```

### Pattern 6: Terminal Width Detection

**What:** Read stdout terminal width at startup; fall back to 160 if non-terminal or error.

**When:** `cmd/alturd/main.go` inside `RunE`, before calling `diff.Render`.

```go
// Source: pkg.go.dev/golang.org/x/term@v0.44.0 [CITED]
import (
    "os"
    "golang.org/x/term"
)

func terminalWidth() int {
    if term.IsTerminal(int(os.Stdout.Fd())) {
        if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
            return w
        }
    }
    return 160 // D-07: fallback for non-terminal stdout (piped output)
}
```

### Anti-Patterns to Avoid

- **Running git via a shell string:** Never `exec.Command("sh", "-c", "git diff "+ref)`. Always use `exec.Command("git", "diff", refArg)` with separate args — prevents shell injection (ASVS V5).
- **Detecting not-in-repo by stderr string parsing:** `"fatal: not a git repository"` message text varies by git version and locale. Detect by ExitCode() == 128 instead.
- **Initializing the log file before flag parsing:** This would create the log file on `--help` or `--version`, violating D-10. `initLog()` must be the first statement inside `RunE`, never in `PersistentPreRunE` or `init()`.
- **Using `os.Truncate(path, 1MB)`:** This keeps the first 1MB (head) not the last 1MB (tail). To keep recent log entries, read the file, slice the tail, and write back. (See Pattern 5's `truncateLog`.)
- **Relying on cobra's `Execute()` return code for exit code routing:** `cobra.Execute()` always returns the error from `RunE`; there is no metadata about which exit code to use. Custom error type is required.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Flag and subcommand dispatch | Custom `os.Args` parser | `github.com/spf13/cobra v1.10.2` | Handles `--help`, `--version`, subcommands, flag errors, usage formatting — getting all these right by hand takes weeks and breaks in edge cases |
| XDG base directory resolution | Platform-detection switch | `github.com/adrg/xdg v0.5.3` | XDG on Linux, Application Support on macOS, LOCALAPPDATA on Windows — three OSes with special-cased paths each; the library has tests for all |
| Terminal width detection | `os.Getenv("COLUMNS")` | `golang.org/x/term.GetSize` | COLUMNS env var is rarely set; GetSize uses the terminal's actual column count via ioctl/syscall — works in tmux, resize events, etc. |
| Log structured output | `fmt.Fprintf(logFile,...)` | `github.com/charmbracelet/log` | Timestamps, log levels, structured fields, and `SetOutput(f)` redirection — free-form format strings produce unreadable logs |

**Key insight:** The git CLI layer is thin by design — its value is in the interface boundary (Runner) that isolates the real subprocess from all tests. Anything that goes into the Runner is framework code; anything that wraps it is business logic.

---

## Runtime State Inventory

> Not applicable — Phase 2 is a new binary entrypoint, not a rename/refactor phase.

---

## Common Pitfalls

### Pitfall 1: Log File Created on `--help` / `--version`

**What goes wrong:** If `initLog()` is called in `PersistentPreRunE` or at package `init()` time, the log file is created whenever cobra starts — including for `--help` and `--version`. This violates D-10 (success criterion 1).

**Why it happens:** cobra's `PersistentPreRunE` runs before every subcommand including help. The `init()` function runs on import. Both are too early.

**How to avoid:** Call `initLog()` as the first statement inside `RunE` — the function that only runs for actual diff invocations. `--help` and `--version` never reach `RunE`.

**Warning signs:** `alturd --help` creates `~/.local/state/alturd/alturd.log`.

---

### Pitfall 2: cobra's `--` Handling vs. Re-inserting `--` for Git

**What goes wrong:** After cobra splits args at `--`, the `args` slice contains only the post-`--` items without the `--` itself. If the git command is built as `git diff refArgs...` + `pathArgs...` (without re-inserting `--`), git may interpret path args as ref args and produce wrong output or errors.

**Why it happens:** cobra consumes `--`; it does not appear in `args`.

**How to avoid:** `ParseRefArgs` must detect `dashIdx >= 0` and explicitly append `"--"` before path args when constructing the git command args slice. (See Pattern 4.)

**Warning signs:** `alturd HEAD -- internal/` runs `git diff HEAD internal/` instead of `git diff HEAD -- internal/`, potentially diffing against a branch named `internal/` instead of filtering by path.

---

### Pitfall 3: Exit Code 128 vs. Exit Code 1 Confusion

**What goes wrong:** Git exits with code 128 for "not a git repository" errors. The requirements say `alturd` should exit 1 in this case. Code that passes git's exit code through directly violates the spec.

**Why it happens:** Developers assume alturd should re-emit git's exit code.

**How to avoid:** In `ExecRunner.Run()`, detect `ExitCode() == 128` and map it to `ErrNotGitRepo` (exit 1). Map `exec.ErrNotFound` to `ErrGitNotFound` (exit 127). Pass through other non-zero exit codes unchanged.

**Warning signs:** `alturd` (outside a git repo) exits 128 instead of 1.

---

### Pitfall 4: CRLF in Git Output on Windows

**What goes wrong:** On Windows, `git` may output `\r\n` line endings in diff output. `go-gitdiff.Parse()` reads lines that end in `\r` — the `Line.Line` field includes the trailing `\r`, breaking string assertions and rendering.

**Why it happens:** Windows git installations often have `core.autocrlf=true` set globally.

**How to avoid:** In `ExecRunner.Run()`, apply `bytes.ReplaceAll(out, []byte("\r\n"), []byte("\n"))` immediately after `cmd.Output()`, before returning the reader. This is D-09.

**Warning signs:** Windows CI test failures with lines ending in `\r` in the parsed content.

---

### Pitfall 5: Fake Runner Arg Slice Comparison

**What goes wrong:** Tests assert `fakeRunner.capturedArgs == []string{"diff", "--unified=3", "HEAD"}` but the Runner interface passes only the args after `"git"` (not including `"git"` itself). Tests that include `"git"` in the expected slice will always fail.

**Why it happens:** `exec.Command("git", args...)` takes `args` as the arguments to git, not including `"git"` itself.

**How to avoid:** The `Runner.Run(args []string)` method receives the post-`"git"` args. Tests compare against slices that do NOT include `"git"`. For example, `["diff"]` for no-args, `["diff", "HEAD~1..HEAD"]` for a ref range.

**Warning signs:** Fake runner captures `["git", "diff"]` and comparison to `["diff"]` fails.

---

## Code Examples

### Verified: cobra root command structure

```go
// Source: pkg.go.dev/github.com/spf13/cobra@v1.10.2 [CITED]
package main

import (
    "errors"
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "github.com/alturd/alturd/internal/git"
    "github.com/alturd/alturd/internal/log"
)

var version = "dev" // overridden by goreleaser -ldflags in Phase 4

var rootCmd = &cobra.Command{
    Use:           "alturd",
    Short:         "Side-by-side terminal diff viewer",
    Version:       version,
    SilenceErrors: true,
    SilenceUsage:  true,
    Args:          cobra.ArbitraryArgs,
    RunE:          run,
}

func run(cmd *cobra.Command, args []string) error {
    logFile, err := applog.Init()
    if err != nil {
        // Log init failure is non-fatal: continue without logging
        _ = err
    } else {
        defer logFile.Close()
    }
    // ... parse refs, run git, render diff ...
    return nil
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        var exitErr *git.ExitCodeError
        if errors.As(err, &exitErr) {
            fmt.Fprintln(os.Stderr, exitErr.Msg)
            os.Exit(exitErr.Code)
        }
        fmt.Fprintln(os.Stderr, err.Error())
        os.Exit(1)
    }
}
```

### Verified: fake Runner for testing

```go
// Source: D-04, D-05 from CONTEXT.md + Go testing conventions [ASSUMED synthesis]
type fakeRunner struct {
    capturedArgs []string
    output       []byte
    err          error
}

func (f *fakeRunner) Run(args []string) (io.Reader, error) {
    f.capturedArgs = args
    if f.err != nil {
        return nil, f.err
    }
    return bytes.NewReader(f.output), nil
}

// Table test: verify each invocation form produces the correct git args
func TestParseRefArgs(t *testing.T) {
    tests := []struct {
        name     string
        args     []string
        dashIdx  int
        wantArgs []string
    }{
        {"no args", []string{}, -1, []string{}},
        {"single ref", []string{"HEAD~1"}, -1, []string{"HEAD~1"}},
        {"two-dot range", []string{"HEAD~1..HEAD"}, -1, []string{"HEAD~1..HEAD"}},
        {"three-dot range", []string{"main...feature"}, -1, []string{"main...feature"}},
        {"two args", []string{"HEAD~1", "HEAD"}, -1, []string{"HEAD~1", "HEAD"}},
        {"paths only", []string{"internal/"}, 0, []string{"--", "internal/"}},
        {"ref + paths", []string{"HEAD~1", "internal/"}, 1, []string{"HEAD~1", "--", "internal/"}},
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            got := git.ParseRefArgs(tc.args, tc.dashIdx)
            if !slices.Equal(got, tc.wantArgs) {
                t.Errorf("ParseRefArgs(%v, %d) = %v, want %v", tc.args, tc.dashIdx, got, tc.wantArgs)
            }
        })
    }
}
```

### Verified: error path tests using fake Runner

```go
// Source: D-05 from CONTEXT.md [ASSUMED synthesis]
func TestRunnerErrNotFound(t *testing.T) {
    // Fake that returns the ErrGitNotFound sentinel
    r := &fakeRunner{err: git.ErrGitNotFound}
    _, err := r.Run([]string{"diff"})
    var exitErr *git.ExitCodeError
    if !errors.As(err, &exitErr) {
        t.Fatalf("expected ExitCodeError, got %T", err)
    }
    if exitErr.Code != 127 {
        t.Errorf("expected exit code 127, got %d", exitErr.Code)
    }
}

func TestRunnerErrNotRepo(t *testing.T) {
    r := &fakeRunner{err: git.ErrNotGitRepo}
    _, err := r.Run([]string{"diff"})
    var exitErr *git.ExitCodeError
    if !errors.As(err, &exitErr) {
        t.Fatalf("expected ExitCodeError, got %T", err)
    }
    if exitErr.Code != 1 {
        t.Errorf("expected exit code 1, got %d", exitErr.Code)
    }
}
```

---

## Validation Architecture

> `workflow.nyquist_validation: true` in config.json — this section is required.

### Test Framework

| Property | Value |
|----------|-------|
| Framework | `go test` (stdlib) |
| Config file | none — `go test ./...` from module root |
| Quick run command | `go test ./internal/git/... ./internal/log/...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GIT-01 | No args → `git diff` (no extra args) | unit | `go test ./internal/git/... -run TestParseRefArgs/no_args` | Wave 1 |
| GIT-02 | All 6 invocation forms produce correct git args | unit | `go test ./internal/git/... -run TestParseRefArgs` | Wave 1 |
| GIT-03 | `-- <paths>` appended correctly with `--` separator | unit | `go test ./internal/git/... -run TestParseRefArgs/paths` | Wave 1 |
| GIT-04 | `--version` / `--help` exit 0; no log file created | integration | `go test ./cmd/alturd/... -run TestVersionHelp` | Wave 3 |
| GIT-05 | Not-in-repo → exit 1; git-not-found → exit 127, single-line stderr | unit + integration | `go test ./internal/git/... -run TestExitCodes` | Wave 1 + 3 |
| LOG-01 | Log file at XDG_STATE_HOME path; truncated when >1MB | unit | `go test ./internal/log/... -run TestInitLog` | Wave 2 |

### Success Criteria → Test Map

| Success Criterion | Test Type | What It Checks |
|-------------------|-----------|----------------|
| SC-1: `--version`/`--help` exit 0, no side effects | integration smoke | Build binary, run `./alturd --version`, assert exit 0, assert log file absent |
| SC-2: Outside repo → exit 1; git not on PATH → exit 127 | unit (fake runner) | `fakeRunner{err: ErrNotGitRepo}` → ExitCodeError{Code:1}; `fakeRunner{err: ErrGitNotFound}` → ExitCodeError{Code:127} |
| SC-3: All 6 invocation forms produce correct git command | unit (arg table test) | `TestParseRefArgs` table with 7 cases covering all forms |
| SC-4: CRLF → LF normalization before diff.Parse | unit | `fakeRunner` returns `"line\r\nother\r\n"`, assert `diff.Parse` receives `"line\nother\n"` |
| SC-5: Log file at XDG path, truncated to 1MB | unit | `TestInitLog` pre-populates file >1MB, calls Init(), asserts file size ≤ 1MB and path matches xdg.StateFile("alturd/alturd.log") |

### Sampling Rate

- **Per-task commit:** `go test ./internal/git/... ./internal/log/...`
- **Per-wave merge:** `go test ./...`
- **Phase gate:** `go test ./...` green + `go build ./cmd/alturd` succeeds before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/git/runner_test.go` — covers GIT-01 through GIT-05 via fake runner
- [ ] `internal/git/args_test.go` — covers all 6 invocation forms of GIT-02/GIT-03
- [ ] `internal/log/log_test.go` — covers LOG-01 truncation + path correctness
- [ ] `cmd/alturd/main_test.go` — integration tests for GIT-04 (no log file on --help/--version) and SC-1

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.22+ | Module build | Yes | go1.25.11 linux/amd64 | N/A |
| git (CLI) | Integration tests (real subprocess); fake runner for unit tests | Yes | 2.39.5 | Fake runner covers unit tests without git |

**Missing dependencies with no fallback:** none

**All unit tests use fake runner (D-05) — no git subprocess required for `go test ./...`.**

---

## Security Domain

> `security_enforcement: true`, `security_asvs_level: 1`

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | No user authentication in this phase |
| V3 Session Management | No | Stateless CLI invocation |
| V4 Access Control | No | No access decisions |
| V5 Input Validation | Yes | Never interpolate user-supplied ref strings into a shell command; always use exec.Command args slice (no `sh -c`) |
| V6 Cryptography | No | No cryptographic operations |

### Known Threat Patterns for Git CLI Wrappers

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Shell injection via ref arg | Tampering | Use `exec.Command("git", "diff", refArg)` — args are passed directly to execve, never through a shell; no `sh -c` string concatenation [VERIFIED: Go exec docs] |
| Path traversal via `-- <path>` | Tampering | git itself validates paths; alturd passes them verbatim to git via exec args (no shell) — git's own path validation applies |
| Log file path traversal (XDG_STATE_HOME override) | Elevation of Privilege | `xdg.StateFile()` returns a path under the process owner's state directory; an attacker controlling XDG_STATE_HOME can only affect files they own |
| Oversized git output filling memory | Denial of Service | `cmd.Output()` buffers all output in memory; `git diff` output is bounded by repository size — acceptable for a developer tool. If a future user reports OOM, switch to `cmd.Stdout = pipeReader` streaming. |
| Stderr injection from git subprocess | Information Disclosure | Capture stderr via `ExitError.Stderr` for debug logging only; never echo it to the user verbatim (stderr may contain sensitive path information) |

**Security note:** The strongest protection here is argument-vector subprocess invocation. `exec.Command("git", args...)` passes each arg as a separate element in the `argv[]` array — there is no shell involved and no injection surface, regardless of what characters appear in ref names.

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `os/exec` with `sh -c "git diff " + ref` | `exec.Command("git", "diff", ref)` | Long-established best practice | Eliminates shell injection surface entirely |
| Manual XDG path construction | `github.com/adrg/xdg.StateFile()` | xdg library exists since 2016 | Cross-platform correctness without hand-rolling Windows/macOS special cases |
| cobra v0.x (github.com/spf13/cobra) | cobra v1.10.2 | v1.0 released 2021 | Stable API; v1.10.2 adds CompletionOptions improvements |

**Deprecated/outdated:**
- `github.com/urfave/cli`: Second-most-common Go CLI library, but D-01 locks cobra for Phase 4 subcommand reasons.
- `flag` (stdlib): Insufficient for Phase 4 subcommand structure; D-01 decision.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | cobra `args` after `--` does NOT include `--` itself; `ArgsLenAtDash()` returns the split index | Pattern 4 | If cobra includes `--` in args, `ParseRefArgs` inserts a duplicate `--` in the git command; easy to fix once verified |
| A2 | `exec.Command("git", ...).Output()` returns `*exec.Error` (not `*exec.ExitError`) when git binary is absent from PATH | Pattern 3 (runner.go) | If Go wraps the LookPath error differently, the `errors.As(err, &execErr)` branch is skipped and git-not-found may appear as exit 1 instead of 127 |
| A3 | git exits with code 128 (not 1, 2, or other) specifically for "not a git repository" | Pattern 3 (runner.go) | Different git error conditions also use 128; the error message on stderr is needed to disambiguate if other 128 cases need different handling |
| A4 | `os.Truncate(path, 0)` followed by `os.WriteFile(path, tail, 0600)` is safe for log file truncation (no concurrent writers) | Pattern 5 | In a concurrent server context this would race; for a single CLI process startup it is safe |
| A5 | `charmbracelet/log v1.0.0` exposes `SetOutput(w io.Writer)` as a package-level function | Pattern 5 | If the API changed between v0.4.x (CLAUDE.md) and v1.0.0, the call site needs updating; verify against pkg.go.dev |

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed. (Table is not empty; A1–A5 require verification during implementation.)

---

## Open Questions

1. **cobra ArgsLenAtDash() behavior with no `--` in command line**
   - What we know: The cobra docs say it returns -1 when `--` was not provided
   - What's unclear: Whether this is the exact return value in cobra v1.10.2 (vs. an older version)
   - Recommendation: Add a unit test `alturd HEAD` (no `--`) and assert `dashIdx == -1` produces `git diff HEAD` with no trailing `--`

2. **charmbracelet/log package-level SetOutput signature in v1.0.0**
   - What we know: CLAUDE.md lists `github.com/charmbracelet/log v0.4.x`; proxy.golang.org returns v1.0.0 as latest
   - What's unclear: Whether the SetOutput API changed between v0.4.x and v1.0.0
   - Recommendation: Run `go get github.com/charmbracelet/log@v1.0.0 && go doc github.com/charmbracelet/log SetOutput` before writing log.go

3. **git diff arg form for two-space-separated refs**
   - What we know: `git diff ref1 ref2` is equivalent to `git diff ref1..ref2` for most cases
   - What's unclear: edge cases with merge bases; not a Phase 2 concern since alturd passes refs verbatim to git
   - Recommendation: Document in args.go that alturd is a pass-through — git itself handles the semantic difference

---

## Sources

### Primary (MEDIUM confidence — verified via Go module proxy or official pkg.go.dev)

- [pkg.go.dev/github.com/spf13/cobra@v1.10.2](https://pkg.go.dev/github.com/spf13/cobra@v1.10.2) — SilenceErrors, SilenceUsage, Version field, PersistentPreRunE, ArgsLenAtDash [CITED: pkg.go.dev/github.com/spf13/cobra]
- [pkg.go.dev/golang.org/x/term@v0.44.0](https://pkg.go.dev/golang.org/x/term@v0.44.0) — GetSize, IsTerminal [CITED: pkg.go.dev/golang.org/x/term]
- [pkg.go.dev/github.com/adrg/xdg@v0.5.3](https://pkg.go.dev/github.com/adrg/xdg@v0.5.3) — StateFile, StateHome, platform equivalents [CITED: pkg.go.dev/github.com/adrg/xdg]
- [pkg.go.dev/os/exec](https://pkg.go.dev/os/exec) — LookPath, ErrNotFound, ExitError, ExitCode(), cmd.Err [CITED: pkg.go.dev/os/exec]
- [pkg.go.dev/os#Truncate](https://pkg.go.dev/os#Truncate) — Truncate semantics (truncates from end), Stat for size check [CITED: pkg.go.dev/os]
- [proxy.golang.org](https://proxy.golang.org) — Version confirmations: cobra v1.10.2, x/term v0.44.0, adrg/xdg v0.5.3, charmbracelet/log v1.0.0

### Secondary (MEDIUM confidence — from CLAUDE.md project authority)

- [.claude/CLAUDE.md §Supporting Libraries](/.claude/CLAUDE.md) — adrg/xdg and charmbracelet/log listed as project-selected dependencies

### Tertiary (LOW confidence — synthesis, tagged [ASSUMED])

- ExitCodeError pattern — standard Go error design; not from a specific authoritative source
- ParseRefArgs algorithm — synthesized from cobra ArgsLenAtDash docs + git diff man page; verify in unit tests
- Log truncation keep-tail algorithm — synthesized from os.Truncate docs + standard patterns

---

## Metadata

**Confidence breakdown:**
- Standard stack: MEDIUM — all packages verified via Go module proxy; cobra/xdg/charmbracelet/log listed in CLAUDE.md
- Architecture: MEDIUM — component responsibilities follow directly from CONTEXT.md decisions D-01 through D-12
- Runner interface: HIGH — identical pattern specified in D-04; standard Go dependency injection
- Exit code mapping: MEDIUM — exec.ErrNotFound and ExitError well-documented; exact mapping assumes git's 128 code (A3)
- Log truncation: LOW — keep-tail algorithm synthesized from docs; needs implementation verification

**Research date:** 2026-06-29
**Valid until:** 2026-07-29 (stable libraries; 30-day horizon)

---

*Phase: 2 — Git Layer + CLI*
*Research completed: 2026-06-29*
