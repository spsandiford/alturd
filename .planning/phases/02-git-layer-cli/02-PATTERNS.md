# Phase 2: Git Layer + CLI - Pattern Map

**Mapped:** 2026-06-29
**Files analyzed:** 8 new files
**Analogs found:** 3 / 8 (all from internal/diff; codebase has no CLI or subprocess files yet)

---

## File Classification

| New File | Role | Data Flow | Closest Analog | Match Quality |
|----------|------|-----------|----------------|---------------|
| `cmd/alturd/main.go` | entrypoint | request-response | `internal/diff/parse.go` (package structure) | partial — same module, different role |
| `internal/git/runner.go` | service | request-response | `internal/diff/parse.go` (io.Reader boundary) | partial — both wrap an external I/O call |
| `internal/git/args.go` | utility | transform | `internal/diff/align.go` (pure function over slice) | partial — same pure-function pattern |
| `internal/git/errors.go` | utility | — | none | no analog |
| `internal/git/runner_test.go` | test | — | `internal/diff/parse_test.go` | role-match — table-driven, `_test` package |
| `internal/git/args_test.go` | test | — | `internal/diff/parse_test.go` | role-match — table-driven, `_test` package |
| `internal/log/log.go` | utility | file-I/O | none | no analog |
| `internal/log/log_test.go` | test | — | `internal/diff/parse_test.go` | role-match |

---

## Pattern Assignments

### `cmd/alturd/main.go` (entrypoint, request-response)

**Analog:** `internal/diff/parse.go` — use for package declaration style, import block layout, and error wrapping idiom. RESEARCH.md §Code Examples contains the authoritative cobra structure for this file.

**Package and import pattern** (`internal/diff/parse.go` lines 1–8):
```go
package diff

import (
	"fmt"
	"io"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
)
```
Apply same structure: stdlib imports in one group, third-party in a second group, project-internal in a third group. `goimports` enforces this.

**Error wrapping pattern** (`internal/diff/parse.go` lines 17–20):
```go
files, _, err := gitdiff.Parse(r)
if err != nil {
    return nil, fmt.Errorf("parsing diff: %w", err)
}
```
Wrap all errors with `%w` for `errors.Is`/`errors.As` chain compatibility. In `main.go`, `RunE` returns wrapped errors; `main()` type-asserts via `errors.As(err, &exitErr)`.

**Cobra root command pattern** (RESEARCH.md §Pattern 1, lines 214–236):
```go
var rootCmd = &cobra.Command{
    Use:           "alturd [ref] [ref1..ref2] [-- paths]",
    Short:         "Side-by-side terminal diff viewer",
    Version:       version,
    SilenceErrors: true,
    SilenceUsage:  true,
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

**Terminal width helper** (RESEARCH.md §Pattern 6):
```go
func terminalWidth() int {
    if term.IsTerminal(int(os.Stdout.Fd())) {
        if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
            return w
        }
    }
    return 160
}
```

**initLog placement rule:** Call `applog.Init()` as the FIRST statement inside `RunE` — never in `PersistentPreRunE` or `init()`. `--help` and `--version` never reach `RunE`.

---

### `internal/git/runner.go` (service, request-response)

**Analog:** `internal/diff/parse.go` — both wrap an external call behind an `io.Reader` boundary.

**io.Reader boundary pattern** (`internal/diff/parse.go` lines 16–22):
```go
func Parse(r io.Reader) ([]*gitdiff.File, error) {
    files, _, err := gitdiff.Parse(r)
    if err != nil {
        return nil, fmt.Errorf("parsing diff: %w", err)
    }
    return files, nil
}
```
`Runner.Run()` produces `io.Reader`; `diff.Parse()` consumes `io.Reader`. The two functions compose directly — no intermediary buffer is needed beyond `bytes.NewReader`.

**Runner interface + ExecRunner pattern** (RESEARCH.md §Pattern 3, lines 271–309):
```go
type Runner interface {
    Run(args []string) (io.Reader, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(args []string) (io.Reader, error) {
    cmd := exec.Command("git", args...)
    out, err := cmd.Output()
    if err != nil {
        var execErr *exec.Error
        if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
            return nil, ErrGitNotFound
        }
        var exitErr *exec.ExitError
        if errors.As(err, &exitErr) && exitErr.ExitCode() == 128 {
            return nil, ErrNotGitRepo
        }
        return nil, err
    }
    // D-09: normalize CRLF immediately after cmd.Output()
    out = bytes.ReplaceAll(out, []byte("\r\n"), []byte("\n"))
    return bytes.NewReader(out), nil
}
```

**Security rule:** NEVER use `exec.Command("sh", "-c", "git diff "+ref)`. Always pass args as separate slice elements to avoid shell injection (ASVS V5).

---

### `internal/git/args.go` (utility, transform)

**Analog:** `internal/diff/align.go` — a pure function over a slice input, no side effects, easily table-tested. Read `internal/diff/align.go` for package-level comment style and pure-function structure.

**Pure function pattern** (RESEARCH.md §Pattern 4, lines 335–355):
```go
// ParseRefArgs converts cobra args (after flag parsing) to git diff arguments.
// dashIdx is cmd.ArgsLenAtDash() — the index in args where "--" appeared, or -1.
func ParseRefArgs(args []string, dashIdx int) []string {
    var refs []string
    var paths []string

    if dashIdx >= 0 {
        refs = args[:dashIdx]
        paths = args[dashIdx:]
    } else {
        refs = args
    }

    result := make([]string, 0, len(refs)+len(paths))
    result = append(result, refs...)
    if len(paths) > 0 {
        result = append(result, "--")
        result = append(result, paths...)
    }
    return result
}
```

**Critical:** cobra strips `--` from `args`; `ArgsLenAtDash()` returns the split index. `ParseRefArgs` must re-insert `"--"` before path args when `dashIdx >= 0`.

---

### `internal/git/errors.go` (utility, no analog)

No codebase analog. Use RESEARCH.md §Pattern 2 directly.

**Sentinel error type pattern** (RESEARCH.md §Pattern 2, lines 247–263):
```go
package git

type ExitCodeError struct {
    Code int
    Msg  string
}

func (e *ExitCodeError) Error() string { return e.Msg }

var ErrGitNotFound = &ExitCodeError{Code: 127, Msg: "git: command not found (is git installed and on PATH?)"}
var ErrNotGitRepo  = &ExitCodeError{Code: 1,   Msg: "not a git repository (run alturd inside a git working tree)"}
```

---

### `internal/log/log.go` (utility, file-I/O, no analog)

No codebase analog. Use RESEARCH.md §Pattern 5 directly.

**XDG log init pattern** (RESEARCH.md §Pattern 5, lines 367–413):
```go
package applog

import (
    "os"
    charmlog "github.com/charmbracelet/log"
    "github.com/adrg/xdg"
)

const maxLogSize = 1 << 20 // 1 MB

func Init() (*os.File, error) {
    path, err := xdg.StateFile("alturd/alturd.log")
    if err != nil {
        return nil, err
    }
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

// truncateLog keeps the LAST maxLogSize bytes (tail, not head).
// Do NOT use os.Truncate(path, maxLogSize) — that keeps the head.
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

**Verify before coding:** Run `go doc github.com/charmbracelet/log SetOutput` after `go get github.com/charmbracelet/log@v1.0.0` — CLAUDE.md lists v0.4.x but research confirms v1.0.0 is latest; API may differ (RESEARCH.md Open Question 2).

---

### `internal/git/runner_test.go` and `internal/git/args_test.go` (test, table-driven)

**Analog:** `internal/diff/parse_test.go` — exact role match. Copy these structural patterns:

**External test package declaration** (`parse_test.go` line 1):
```go
package diff_test
```
Use `package git_test` (external black-box test package), not `package git`.

**Struct-based table test pattern** (`parse_test.go` lines 13–31):
```go
type parseCase struct {
    name          string
    fixture       string
    wantFileCount int
    // ... fields named wantXxx
}

var parseCases = []parseCase{ ... }
```
For `args_test.go`, use `[]struct{ name, args, dashIdx, wantArgs }`. For `runner_test.go`, use `[]struct{ name, fakeOutput, fakeErr, wantArgs, wantErr }`.

**t.Run sub-test pattern** (`parse_test.go` lines 160–165):
```go
for _, tc := range parseCases {
    t.Run(tc.name, func(t *testing.T) {
        // ... assertions
    })
}
```

**Fake Runner pattern** (RESEARCH.md §Code Examples, lines 588–600):
```go
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
```
`capturedArgs` does NOT include `"git"` — only the args after `"git"` (e.g., `["diff"]`, not `["git", "diff"]`).

**Error assertion pattern** (`parse_test.go` lines 265–282):
```go
defer func() {
    if r := recover(); r != nil {
        t.Errorf("panicked: %v", r)
    }
}()
```
For exit code tests, use `errors.As`:
```go
var exitErr *git.ExitCodeError
if !errors.As(err, &exitErr) {
    t.Fatalf("expected ExitCodeError, got %T", err)
}
if exitErr.Code != 127 {
    t.Errorf("expected exit code 127, got %d", exitErr.Code)
}
```

---

### `internal/log/log_test.go` (test, file-I/O)

**Analog:** `internal/diff/parse_test.go` — package structure and t.Run pattern. No file-I/O test analog in codebase.

**Test setup pattern for file-I/O** (synthesized — no existing analog):
- Use `t.TempDir()` to create the test log file location; set `XDG_STATE_HOME` env var via `t.Setenv` to redirect xdg path into the temp dir.
- Pre-populate a file larger than 1 MB, call `applog.Init()`, assert `fi.Size() <= maxLogSize` and that the returned `*os.File` is non-nil.
- Use `defer f.Close()` immediately after `Init()` returns.

---

## Shared Patterns

### Import Group Ordering
**Source:** `internal/diff/parse.go` lines 3–8, `internal/diff/render.go` lines 3–11
**Apply to:** All new files

```go
import (
    // Group 1: stdlib
    "bytes"
    "errors"
    "fmt"
    "io"
    "os"
    "os/exec"

    // Group 2: third-party
    charmlog "github.com/charmbracelet/log"
    "github.com/adrg/xdg"
    "github.com/spf13/cobra"

    // Group 3: project-internal
    "github.com/alturd/alturd/internal/diff"
    "github.com/alturd/alturd/internal/git"
)
```
`goimports` enforces this. Blank lines between groups, not within.

### Error Wrapping with %w
**Source:** `internal/diff/parse.go` line 19
**Apply to:** All new files that return errors

```go
return nil, fmt.Errorf("context describing what failed: %w", err)
```

### Package-Level Doc Comment
**Source:** `internal/diff/model.go` lines 1–10
**Apply to:** Each new package's primary file (`runner.go`, `args.go`, `log.go`)

```go
// Package git provides the git subprocess adapter for alturd.
// It exposes a Runner interface for dependency injection in tests,
// and maps CLI argument forms to git diff argument slices.
package git
```

### Table-Driven Tests with Named Subtests
**Source:** `internal/diff/parse_test.go` lines 160–165
**Apply to:** `runner_test.go`, `args_test.go`, `log_test.go`

All test cases use `t.Run(tc.name, func(t *testing.T) { ... })` with descriptive snake_case names matching the invocation form (e.g., `"no_args"`, `"two_dot_range"`, `"paths_only"`).

### Module Path
**Source:** `go.mod` line 1
**Apply to:** All import paths in new files

```
module github.com/alturd/alturd
```
Internal packages import as `github.com/alturd/alturd/internal/git`, etc.

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/git/errors.go` | utility | — | No custom error type files in codebase yet |
| `internal/log/log.go` | utility | file-I/O | No XDG/logging infrastructure in codebase yet |

For these files, RESEARCH.md §Pattern 2 and §Pattern 5 are the primary implementation references.

---

## Anti-Patterns (Do Not Copy)

These are codebase patterns that do NOT apply to Phase 2:

- `internal/diff/render.go` uses package-level `var dmp = diffmatchpatch.New()` (singleton). Do NOT apply this pattern to `ExecRunner` — the Runner should be stateless; callers pass it in via dependency injection.
- `internal/diff/render_test.go` uses `package diff_test` with a `parseFirst` helper that opens fixture files. Phase 2 tests do NOT use fixture files — they use a `fakeRunner` that returns inline bytes.

---

## Metadata

**Analog search scope:** `/src/alturd/internal/diff/`
**Go files scanned:** 9 (all existing source files)
**Pattern extraction date:** 2026-06-29
