---
phase: "02"
status: "issues"
files_reviewed: 9
files_reviewed_list:
  - internal/git/errors.go
  - internal/git/runner.go
  - internal/git/args.go
  - internal/git/runner_test.go
  - internal/git/args_test.go
  - internal/log/log.go
  - internal/log/log_test.go
  - cmd/alturd/main.go
  - cmd/alturd/main_test.go
findings:
  critical: 1
  warning: 5
  info: 2
  total: 8
reviewed_at: "2026-06-30"
---

# Phase 02: Code Review Report

**Reviewed:** 2026-06-30
**Depth:** standard
**Files Reviewed:** 9
**Status:** issues_found

## Summary

Reviewed the git subprocess adapter (`internal/git`), applog initializer (`internal/log`), and cobra entrypoint (`cmd/alturd`). The security posture is sound: `exec.Command` uses argv form throughout with no shell injection surface, log files are created with 0600 permissions, and XDG path resolution is correctly isolated in tests. The `ParseRefArgs` logic is correct for all six documented invocation forms and the test matrix is comprehensive.

One critical correctness bug exists: every git fatal error — including invalid refs and missing commits — is unconditionally mapped to "not a git repository", which gives users completely wrong diagnostic advice. Five warnings cover a dead-code `defer` that leaks the test binary on every CI run, a Windows broken-pipe silent-success path, lost git stderr on most error paths, overly broad CRLF stripping that corrupts diffs of Windows-formatted files, and a non-atomic log truncation that can silently destroy the log file. Two info-level issues cover mutable sentinel exports and a contradictory test comment.

## Critical Issues

### CR-01: All git fatal errors (exit 128) mapped to "not a git repository"

**File:** `internal/git/runner.go:52-55`

**Issue:** Git exits with code 128 for **any** fatal error, not exclusively for "not a git repository" situations. Examples:

- `git diff BADREF` inside a valid repo → `fatal: unknown revision or path not in the working tree` → exit 128
- `git diff main...nonexistent` → `fatal: unknown revision` → exit 128
- `git diff HEAD` with a corrupt object store → `fatal: object ... is missing` → exit 128

All of these hit the current branch and return `ErrNotGitRepo`, printing:

```
not a git repository (run alturd inside a git working tree)
```

This is incorrect diagnostic advice for a user who is inside a valid repo but mistyped a ref name. The correct fix is to inspect git's stderr before deciding which sentinel to return.

**Fix:**
```go
var exitErr *exec.ExitError
if errors.As(err, &exitErr) && exitErr.ExitCode() == 128 {
    stderr := strings.ToLower(strings.TrimSpace(string(exitErr.Stderr)))
    if strings.Contains(stderr, "not a git repository") {
        return nil, ErrNotGitRepo
    }
    // Other exit-128 fatals: return git's actual message.
    if len(exitErr.Stderr) > 0 {
        return nil, fmt.Errorf("git diff: %s", strings.TrimRight(string(exitErr.Stderr), "\n"))
    }
    return nil, fmt.Errorf("git diff: %w", err)
}
```

## Warnings

### WR-01: `defer os.RemoveAll` in `TestMain` is dead code — temp binary leaks every CI run

**File:** `cmd/alturd/main_test.go:25-36`

**Issue:** `os.Exit` does not unwind the call stack; deferred functions are never called. The pattern:

```go
defer os.RemoveAll(dir)   // line 25 — never executes
...
os.Exit(m.Run())           // line 36 — bypasses all defers
```

leaks the temp directory containing the compiled `alturd` binary on every invocation of `go test ./cmd/alturd/...`. On CI runners with many parallel jobs or frequent runs, this accumulates stale binary-sized artifacts.

**Fix:**
```go
code := m.Run()
os.RemoveAll(dir) // explicit cleanup before exit
os.Exit(code)
```

---

### WR-02: `fmt.Fprintln` error silently discarded in render loop — silent exit 0 on Windows broken pipe

**File:** `cmd/alturd/main.go:73-75`

**Issue:** The render loop discards the write error from `fmt.Fprintln`:

```go
for _, row := range rows {
    fmt.Fprintln(os.Stdout, row)  // return value ignored
}
return nil
```

On Unix, writing to a closed pipe sends SIGPIPE which terminates the process with non-zero status. On Windows there is no SIGPIPE: `fmt.Fprintln` returns `ERROR_BROKEN_PIPE`, the loop runs to completion across all files, and `run()` returns `nil`. A pipeline like `alturd | head -5` on Windows exits 0 even though output was truncated — callers that rely on the exit code for correctness checking get a false success.

**Fix:**
```go
for _, row := range rows {
    if _, err := fmt.Fprintln(os.Stdout, row); err != nil {
        return fmt.Errorf("writing output: %w", err)
    }
}
```

---

### WR-03: Git stderr discarded on non-sentinel exit codes — users see "exit status N" only

**File:** `internal/git/runner.go:44-56`

**Issue:** `cmd.Output()` stores git's stderr in `ExitError.Stderr` when the command fails, but the fallthrough error path discards it:

```go
return nil, fmt.Errorf("git diff: %w", err)
```

`err.Error()` on `*exec.ExitError` returns only `"exit status N"`. The user-facing message from `main()` becomes:

```
git diff: exit status 1
```

with no indication of what git actually complained about. After CR-01 is fixed (exit-128 narrowed), remaining non-sentinel exit codes (e.g., 1 from partial matches, 129 from unknown flags) still lose git's actual diagnostic.

**Fix:**
```go
// Fallthrough: unrecognised exit code — surface git's stderr if available.
var exitErr2 *exec.ExitError
if errors.As(err, &exitErr2) && len(exitErr2.Stderr) > 0 {
    return nil, fmt.Errorf("git diff: %s", strings.TrimRight(string(exitErr2.Stderr), "\n"))
}
return nil, fmt.Errorf("git diff: %w", err)
```

---

### WR-04: `NormalizeCRLF` strips content `\r` bytes — corrupts diffs of Windows-formatted files

**File:** `internal/git/runner.go:68-70`

**Issue:** In a unified diff of a Windows file with CRLF line endings, added/removed lines appear as:

```
+line content\r\n
```

where `\r` is **file content** and `\n` is the diff protocol line terminator. `bytes.ReplaceAll` treats every `\r\n` byte pair identically, stripping the content `\r`. A commit that intentionally preserves CRLF content would display as having no `\r` bytes, corrupting intra-line character highlighting and potentially masking real content changes.

The D-09 rationale targets Windows git's own line endings on the diff envelope, not the content inside diff lines. A narrower fix strips only `\r` at the line-final position (immediately preceding `\n` after a newline split) rather than globally.

**Fix (minimal):** Process line by line and strip only trailing `\r`:
```go
func NormalizeCRLF(b []byte) []byte {
    return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}
// ^^^ this is what the current code does; the fix requires awareness of context.
// Correct approach: strip \r only when it appears at position [len(line)-1]
// after splitting on \n, i.e. the diff transport's own CR, not content CR.
```

A simpler but still imperfect fix is to only normalize when git is confirmed to be running on Windows (check `runtime.GOOS`), since Unix git never emits CRLF in its diff output.

---

### WR-05: Non-atomic log truncation can silently destroy the log file

**File:** `internal/log/log.go:52-64`

**Issue:** `truncateLog` reads the entire file into memory, slices it, then overwrites it:

```go
data, err := os.ReadFile(path)   // step 1
...
os.WriteFile(path, data, 0600)   // step 2 — clobbers file even on partial failure
```

If the process is killed or OOM-killed between steps 1 and 2, the log file on disk is left empty (zero bytes): `os.WriteFile` truncates the file before writing. The user loses all historical log data with no error reported. A rename-based atomic replacement avoids this:

**Fix:**
```go
func truncateLog(path string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return fmt.Errorf("reading log for truncation: %w", err)
    }
    if int64(len(data)) > maxLogSize {
        data = data[int64(len(data))-maxLogSize:]
    }
    dir := filepath.Dir(path)
    tmp, err := os.CreateTemp(dir, ".alturd-log-*.tmp")
    if err != nil {
        return fmt.Errorf("creating temp log: %w", err)
    }
    tmpName := tmp.Name()
    if _, werr := tmp.Write(data); werr != nil {
        tmp.Close(); os.Remove(tmpName)
        return fmt.Errorf("writing truncated log: %w", werr)
    }
    if cerr := tmp.Close(); cerr != nil {
        os.Remove(tmpName)
        return fmt.Errorf("closing temp log: %w", cerr)
    }
    return os.Rename(tmpName, path)
}
```

## Info

### IN-01: Mutable exported fields on package-level sentinel errors allow silent corruption

**File:** `internal/git/errors.go:12-35`

**Issue:** `ExitCodeError.Code` and `ExitCodeError.Msg` are exported. Package-level sentinels `ErrGitNotFound` and `ErrNotGitRepo` are pointers to these mutable structs. `errors.As(err, &exitErr)` sets `exitErr` to the **same pointer** as the global singleton — not a copy. Any caller that writes `exitErr.Code = 0` after a successful `errors.As` match silently corrupts the sentinel for all subsequent uses in the process.

No current caller mutates these fields, but the hazard grows as the codebase adds callers. Standard library sentinels (e.g., `io.EOF`) use unexported fields via `errors.New` to prevent this pattern.

**Fix:** Expose read-only accessors instead of exported fields, or add a `//nolint` + documentation note that explicitly prohibits mutation.

---

### IN-02: Misleading comment in `TestCRLFNormalization` contradicts its own implementation

**File:** `internal/git/runner_test.go:108-111`

**Issue:** The opening sentence of the comment reads "This test uses ExecRunner directly with a real git subprocess" but the test never instantiates `ExecRunner`. The following sentence immediately retracts this: "we use a fakeRunner... we use a unit-testable helper." The opening sentence is a leftover from an earlier draft and will confuse future maintainers into thinking the ExecRunner subprocess path is covered here.

**Fix:** Replace the comment block:
```go
// TestCRLFNormalization verifies the NormalizeCRLF helper that ExecRunner applies
// to its output before returning. The helper is exported specifically to enable
// this unit test without spawning a subprocess; the integration path through
// ExecRunner is exercised in TestSmokeRunInRepoExitsZero.
```

---

_Reviewed: 2026-06-30_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
