# Phase 2: Git Layer + CLI - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-29
**Phase:** 2-Git Layer + CLI
**Areas discussed:** CLI framework, Subprocess test design, Phase 2 binary output, main package layout

---

## CLI Framework

| Option | Description | Selected |
|--------|-------------|----------|
| cobra | Adds ~500KB dep but gives structured subcommands for free — install-difftool in Phase 4 is just cobra.Command{}. The standard Go CLI library. | ✓ |
| stdlib flag | Zero deps. Subcommand dispatch requires manual os.Args[1] check in Phase 4 — works but more boilerplate. | |
| urfave/cli | Another structured option, lighter than cobra but less common. | |

**User's choice:** cobra

---

| Option | Description | Selected |
|--------|-------------|----------|
| Show only Phase 2 flags now | Root command has no subcommands in Phase 2; Phase 4 adds install-difftool as a subcommand then. Cleanest incremental build. | ✓ |
| Stub install-difftool in Phase 2 | Add the subcommand as a stub that prints 'not yet implemented' — preserves CLI shape but wastes effort that Phase 4 will redo. | |

**User's choice:** Show only Phase 2 flags now

---

| Option | Description | Selected |
|--------|-------------|----------|
| cobra's built-in --version | Set rootCmd.Version = "dev" in Phase 2; goreleaser injects the real version via ldflags in Phase 4. Zero extra code. | ✓ |
| Custom --version flag | Hand-roll the flag to get exact output format control. Only needed if Python --version output is unusual. | |

**User's choice:** cobra's built-in --version

---

## Subprocess Test Design

| Option | Description | Selected |
|--------|-------------|----------|
| Interface injection | type Runner interface { Run(args []string) (io.Reader, error) }. Real impl uses exec.Command; tests inject a fake that captures args. | ✓ |
| Function variable | var runGit = func(args []string) ([]byte, error) { ... }. Tests swap it. Simpler, but package-level var is a testing smell. | |
| Real temp git repo | TestMain creates a git repo, runs real git commands. Most realistic but slower. | |

**User's choice:** Interface injection

---

| Option | Description | Selected |
|--------|-------------|----------|
| Mix: interface for command-capture, real git for error paths | Error paths are hard to fake accurately; use real git for those tests. | |
| Interface for everything | Inject fake errors too. Simpler test setup. | ✓ |

**User's choice:** Interface for everything

---

## Phase 2 Binary Output

| Option | Description | Selected |
|--------|-------------|----------|
| Rendered diff to stdout (ANSI, no TUI) | Call diff.Render() and write the []string rows to os.Stdout. Gives a usable intermediate artifact. | ✓ |
| Exit 0, no output | Pure wiring/testing phase — no visible output. Verified entirely via unit tests. | |
| Debug dump to stdout | Print parsed FileDiff metadata as plain text for development debugging. | |

**User's choice:** Rendered diff to stdout (ANSI, no TUI)

---

| Option | Description | Selected |
|--------|-------------|----------|
| os.Stdout terminal width via os.Getterm size | Use golang.org/x/term to get terminal columns; falls back to 160 for piped output. | ✓ |
| Hard-coded fallback (e.g. 160 cols) | No terminal detection in Phase 2 — just use a default. Simpler; Phase 3 replaces it anyway. | |

**User's choice:** Terminal width detection via golang.org/x/term, fallback 160

---

## main Package Layout

| Option | Description | Selected |
|--------|-------------|----------|
| cmd/alturd/main.go | Canonical Go multi-binary layout. goreleaser detects it automatically. go build ./cmd/alturd. Scales cleanly if a second binary is ever added. | ✓ |
| main.go at repo root | go build . works. More common in small single-binary CLIs. | |

**User's choice:** cmd/alturd/main.go

---

## Claude's Discretion

None — all areas had explicit user selection.

## Deferred Ideas

None — discussion stayed within phase scope.
