---
phase: 04
slug: config-theming-difftool-distribution
status: verified
threats_open: 0
asvs_level: 1
created: 2026-08-06
---

# Phase 04 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing | Plan |
|----------|-------------|---------------|------|
| filesystem → config decoder | A TOML file at an arbitrary path (`--config`) or the XDG default is read and decoded into process state before the TUI starts. | Config file bytes | 04-01 |
| config decoder → TUI dispatch | Decoded key strings become the dispatch table for every keystroke in the running program. | Keybinding strings | 04-01 |
| Go module proxy → build | `github.com/pelletier/go-toml/v2` is added to the dependency graph. | Third-party module code | 04-01 |
| pull-request author → CI runner | An untrusted fork PR causes workflow code to run on GitHub-hosted infrastructure with the repository's default token. | Workflow execution | 04-02 |
| CI runner → GitHub Releases API | The release job holds a write-scoped `GITHUB_TOKEN` and uploads user-downloadable executables. | Release assets, token | 04-02 |
| GitHub Actions marketplace → workflow | Third-party actions (setup-go, goreleaser-action, golangci-lint-action) execute with the job's permissions. | Action code | 04-02 |
| goreleaser → end user | Published binaries are executed directly by users who downloaded them. | Compiled binary | 04-02 |
| git (parent process) → alturd | `git difftool` invokes alturd as a subprocess, supplying `$LOCAL`/`$REMOTE`/`$MERGED` as argv values and `GIT_DIFF_PATH_COUNTER`/`GIT_DIFF_PATH_TOTAL`, `GIT_EXTERNAL_DIFF` in the environment. | Env vars, argv paths | 04-03, 04-05, 04-06 |
| environment → terminal renderer | Env-var values would reach the user's terminal as rendered title-bar text if passed through unvalidated. | Rendered text | 04-03 |
| alturd → `git diff --no-index` subprocess | User-supplied file paths are passed to a git subprocess. | File paths | 04-03, 04-06 |
| terminal → alturd | The OSC 11 background-color response is a byte sequence read back from the terminal device; terminal dimensions arrive via `tea.WindowSizeMsg`. | Terminal escape responses, size | 04-03, 04-05 |
| CLI arguments → gitconfig key namespace | The `--name` value is interpolated into the config key `difftool.<name>.cmd`. | Tool-name string | 04-04 |
| alturd → `git config` subprocess | Keys and values cross a subprocess boundary and are persisted to a file git also reads. | Config keys/values | 04-04, 04-07 |
| gitconfig `cmd` value → shell | Git evaluates the stored `cmd` string in a shell on every later `git difftool -t alturd` invocation. | Shell command string | 04-04 |
| alturd → the user's existing gitconfig (global scope) | An existing file containing unrelated keys, comments and ordering is modified in place by git. | Config file | 04-04, 04-07 |
| alturd → user's terminal device | On exit, alturd is responsible for handing back a terminal in raw-mode-off, primary-screen state. | Terminal mode state | 04-05 |
| alturd → git (parent process) | The process exit status is the only channel by which `difftool.trustExitCode` learns to stop/continue git's per-file loop. | Exit code | 04-05, 04-07 |
| user gitconfig → alturd's git subprocess | `diff.external` (global, system, or repo `.git/config`) redirects what any `git diff` alturd runs actually executes. | Config value naming an external program | 04-06 |
| alturd → OS process table | Every `difftoolDiff` call spawns a git subprocess; a self-referential dispatch could turn one invocation into unbounded process creation. | Subprocess spawns | 04-06 |
| test harness → filesystem/subprocess | The 04-07 end-to-end test writes an executable script and runs real `git`/`alturd` subprocesses. | Test-owned temp files | 04-07 |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-04-01-01 | Tampering | `config.Load` XDG discovery path | medium | mitigate | Read-only `xdg.SearchConfigFile`; `TestLoad_NoSideEffectsOnFirstRun` asserts zero fs entries created. Verified: `internal/config/config.go:60`, `internal/config/config_test.go:167`. | closed |
| T-04-01-02 | DoS | `Keymap.Merge` duplicate/shadow bindings | medium | mitigate | Merge-then-validate rejects two actions resolving to the same key. Verified: `internal/config/keybindings.go:187-199`. | closed |
| T-04-01-03 | DoS | go-toml/v2 decoder on pathological TOML | low | accept | Local user authors the file they run; no privilege boundary crossed. | closed |
| T-04-01-04 | Information Disclosure | `config:` error strings echoing resolved path | low | accept | Path is one the invoking user already supplied/owns; no secret material in keybinding/theme files. | closed |
| T-04-01-05 | Tampering | `--config <path>` arbitrary local file | low | accept | Standard `os.Open` error handling; user already has read access to any path they can name. | closed |
| T-04-01-06 | Elevation of Privilege | Unvalidated key strings reaching `Lookup` | low | mitigate | `validKeyString` constrains accepted values to `tea.KeyPressMsg.String()` forms. Verified: `internal/config/keybindings.go:89-101,172-177`. | closed |
| T-04-01-SC | Tampering | `go-toml/v2@v2.4.3` dependency | high | mitigate | Package-legitimacy audit OK; version pinned exactly, checksum recorded. Verified: `go.mod:15`, `go.sum`. | closed |
| T-04-02-01 | Elevation of Privilege | `ci.yml` trigger and token scope | high | mitigate | `pull_request` (not `pull_request_target`), top-level `permissions: contents: read`. Verified: `.github/workflows/ci.yml:4-6`. | closed |
| T-04-02-02 | Information Disclosure | `GITHUB_TOKEN` exposure across release steps | high | mitigate | Token only in goreleaser-action step `env:`; zero `run:` steps in release.yml. Verified: `.github/workflows/release.yml:23-24`. | closed |
| T-04-02-03 | Tampering | Third-party Actions resolving to a moving ref | medium | mitigate | All actions pinned to explicit major version; golangci-lint pinned to exact patch. Verified: `ci.yml:15,16,29,31`, `release.yml:12,15,18`. | closed |
| T-04-02-04 | Tampering | Published release assets substituted/truncated | medium | mitigate | `checksum.name_template: checksums.txt` covers every artifact. Verified: `.goreleaser.yaml:30-31`. | closed |
| T-04-02-05 | Spoofing | Non-semver/moving tag triggering unintended release | medium | mitigate | `on.push.tags` matches only `v*.*.*`. Verified: `release.yml:4-5`. | closed |
| T-04-02-06 | Tampering | CGO reintroduced producing a glibc-linked artifact | high | mitigate | Single `builds:` entry, `env: [CGO_ENABLED=0]`, no per-target override. Verified: `.goreleaser.yaml:3-8,20-22`. | closed |
| T-04-02-07 | Repudiation | Release provenance unclear for a downloaded binary | low | accept | `-X main.version` + `-trimpath` stamp version/strip paths; cryptographic signing out of DIST-01/02/03 scope. | closed |
| T-04-02-SC | Tampering | goreleaser/golangci-lint binaries installed at CI time | high | mitigate | Installed via pinned official actions from vendor release channels; no npm/pip/cargo. Verified: `release.yml:21`, `ci.yml:31`. | closed |
| T-04-03-01 | Tampering | `GIT_DIFF_PATH_COUNTER`/`GIT_DIFF_PATH_TOTAL` rendered into title bar | high | mitigate | `strconv.Atoi`; non-numeric/non-positive collapses to counter-less template; no raw env string reaches terminal. Verified: `cmd/alturd/main.go:345-352`, `internal/tui/model.go:313-315`. | closed |
| T-04-03-02 | Tampering | `exec.Command("git","diff","--no-index",...)` with user paths | medium | mitigate | Argv form only, `//nolint:gosec` + SECURITY comment, `--` separator before paths. Verified: `cmd/alturd/main.go:308-313`. | closed |
| T-04-03-03 | Tampering | Writes to `$LOCAL`/`$REMOTE`/`$MERGED` | high | mitigate | All three opened read-only; no write/truncate/remove code path. Verified: `main.go:262`; repo-wide grep confirms no such calls outside `internal/log/log.go`. | closed |
| T-04-03-04 | DoS | OSC 11 query hanging in subprocess/tmux/SSH context | high | mitigate | Query skipped outright in difftool mode; standalone path bounded by 50ms race. Verified: `internal/config/theme.go:32,72-77,108-110`. | closed |
| T-04-03-05 | Information Disclosure | OSC 11 query bytes in scrollback if terminal doesn't answer | medium | mitigate | Detection resolved before `tea.NewProgram` takes the TTY; timeout branch produces no further output. Verified: `main.go:125-127,175`, `theme.go:66-78`. | closed |
| T-04-03-06 | Information Disclosure | `--difftool-*` pointing at arbitrary local file | low | accept | Local paths the invoking user can already read; standard `os.ReadFile` error handling. | closed |
| T-04-03-07 | DoS | Very large `--difftool-remote` file read wholly into memory | low | accept | Large-file streaming explicitly out of scope per REQUIREMENTS.md; standalone path already reads whole files. | closed |
| T-04-03-SC | Tampering | New third-party dependencies (termenv promoted to direct) | low | accept | No new module added; termenv already vetted in Phase 3. | closed |
| T-04-04-01 | Tampering | `--name` interpolated into `difftool.<name>.cmd` | high | mitigate | `validateToolName` anchors to `^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`, runs before any subprocess. Verified: `cmd/alturd/difftool.go:46,102,107`. | closed |
| T-04-04-02 | Elevation of Privilege | Shell-evaluated `difftool.<name>.cmd` value | high | mitigate | Stored value is a static literal, no alturd-side interpolation. Verified: `difftool.go:58,145` (`const` template). | closed |
| T-04-04-03 | Tampering | Unintended modification of unrelated gitconfig entries | high | mitigate | Each of four keys set via its own `git config` call; `TestInstallDifftoolOnlyTouchesFourKeys` asserts count and survival of pre-seeded keys. Verified: `difftool.go:142,145,148,151`, `installdifftool_test.go:353,345,348`. | closed |
| T-04-04-04 | Tampering | `git config` invoked with a shell-interpreted command string | high | mitigate | Argv form only via `exec.Command`, `//nolint:gosec` + SECURITY comment; no shell spawned. Verified: `difftool.go:178-181`. | closed |
| T-04-04-05 | Spoofing | Silently hijacking a `diff.tool` configured for another tool | medium | mitigate | Read-then-write check blocks a differing existing value, requires explicit `--force`. Verified: `difftool.go:107-118`. | closed |
| T-04-04-06 | DoS | Partially written config leaving git unable to launch any difftool | medium | accept | Each key set via a separate atomic `git config` call; a re-run converges (idempotency contract). Transactional multi-key writes not offered by `git config`. | closed |
| T-04-04-07 | Information Disclosure | Test runs reading/mutating the developer's real `~/.gitconfig` | medium | mitigate | Every subprocess in the test file sets `GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM` isolation. Verified: `installdifftool_test.go:21-26` and 11 call sites; sibling files use `t.Setenv` equivalents. | closed |
| T-04-04-08 | Repudiation | Confusing raw git stderr dump replacing alturd's own error copy | low | mitigate | Git exit codes mapped to `git.ErrLocalScopeOutsideRepo`/typed errors instead of raw stderr. Verified: `difftool.go:206-207,228-237`. | closed |
| T-04-04-SC | Tampering | New third-party dependencies | low | accept | No module added. | closed |
| T-04-05-01 | DoS | `View()` separator-column arithmetic on unvalidated terminal height | high | mitigate | Separator repeat count and `handleResize` content height clamped at zero. Verified: `internal/tui/model.go:273-277,336-338`. | closed |
| T-04-05-02 | DoS | Terminal left in raw mode/alt screen after abort key | high | mitigate | Abort routed through `tea.Quit`, not `os.Exit`; terminal restored before process exit. Verified: `model.go:585-594`; repo-wide grep confirms no `os.Exit` in `internal/tui`. | closed |
| T-04-05-03 | Tampering | Exit status contract with parent `git difftool` process | high | mitigate | `errAborted = &git.ExitCodeError{Code: 1}` routed via `errors.As`; `TestReportError` pins code 1 + empty output. Verified: `cmd/alturd/main.go:185-187,221`, `main_internal_test.go:29-33`. **Note:** the mitigation's original second clause ("`trustExitCode=true` still stops git's loop") was superseded by 04-07's G-04-1 fix, which now writes `trustExitCode=false` — see T-04-07-04 for the accepted residual (multi-file abort-stop no longer occurs). The exit-code-1 contract itself remains intact and tested. | closed |
| T-04-05-04 | Information Disclosure | Hostile filename with ANSI escapes reaching title bar via `--difftool-path` | low | accept | Pre-existing, not widened; `ansi.Truncate` is escape-sequence aware so it cannot strand the terminal in an attacker-chosen SGR state. Full filename sanitization out of scope. | closed |
| T-04-05-05 | Repudiation | Debug log file abandoned unclosed on abort | low | mitigate | Abort returns through `run()` so `defer logFile.Close()` runs. Verified: `main.go:68-71,186`. | closed |
| T-04-05-SC | Tampering | Third-party dependency supply chain (`charmbracelet/x/ansi`) | low | accept | Already a direct, vetted dependency; no new module added by this plan. | closed |
| T-04-06-01 | DoS | `difftoolDiff`'s git subprocess inheriting `GIT_EXTERNAL_DIFF` (recursive fork bomb, G-04-2) | critical | mitigate | `--no-ext-diff` added to the diff primitive so it can never dispatch outward. `TestDifftoolDiffIgnoresExternalDiffConfiguration` passes. Verified: `cmd/alturd/main.go:313`, `cmd/alturd/difftooldiff_internal_test.go:29`. | closed |
| T-04-06-02 | Tampering | Program named by `GIT_EXTERNAL_DIFF`/`diff.external` executed by alturd's own subprocess | medium | mitigate | Both dispatch points (`main.go:313,203`) carry `--no-ext-diff`; call-site audit confirms no other `git diff` invocation is reachable from a difftool subprocess tree. | closed |
| T-04-06-03 | Tampering | Standalone `git diff` output shaped by `diff.external` before `diff.Parse` | medium | mitigate | `diffArgs` pins standalone argv to git's own diff output. `main_internal_test.go:78-93` pins all four argv shapes. Verified: `main.go:202-204`. | closed |
| T-04-06-04 | DoS | Absence of an explicit recursion-depth guard | low | accept | Dispatch vector removed at its only reachable call site (confirmed by debug-session call-site audit); a depth guard would defend against a cycle that no longer exists. | closed |
| T-04-06-05 | Information Disclosure | git's stderr surfaced verbatim through `difftoolDiff`'s error wrapping | low | accept | Unchanged by this plan; existing wrapping prints only paths the user supplied. | closed |
| T-04-06-SC | Tampering | Third-party dependency supply chain | low | accept | No module added; only standard-library imports. | closed |
| T-04-07-01 | Tampering | `gitConfigSet` call for the trust key in `runInstallDifftool` | low | mitigate | Key and value are compile-time literals in argv form; `validateScope` precedes any subprocess. Verified: `difftool.go:98,107,151`. | closed |
| T-04-07-02 | DoS | User's global gitconfig shared with other difftools | low | accept | `install-difftool` already owned and unconditionally overwrote this exact global key before this change; blast radius unchanged, reversible with one `git config` call. | closed |
| T-04-07-03 | Elevation of Privilege | End-to-end test's executable stub script | low | mitigate | Stub created inside `t.TempDir()`, referenced by absolute path from an isolated repo-local gitconfig; `GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM` isolation. Verified: `installdifftool_test.go:156-157,188-191,199`. | closed |
| T-04-07-04 | Repudiation | Difftool abort signalling — git can no longer observe an abort in a multi-file session | low | accept | User-locked decision (2026-08-06): direction (1) chosen over (2); alturd's own exit status 1 still carries the signal for non-git callers. Residual of T-04-05-03's superseded clause. | closed |
| T-04-07-SC | Tampering | npm/pip/cargo installs | low | accept | No package-manager installs, nothing added to `go.mod`. | closed |

*Status: open · closed · open — below `high` threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `workflow.security_block_on` (high) count toward `threats_open`*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-04-01 | T-04-01-03 | go-toml/v2 pathological-input DoS: local user authors own config, no privilege boundary. | 04-01-PLAN.md | 2026-08-03 |
| AR-04-02 | T-04-01-04 | Config error strings echo a path the user already owns/supplied; no secrets in keybinding/theme files. | 04-01-PLAN.md | 2026-08-03 |
| AR-04-03 | T-04-01-05 | `--config <path>` arbitrary local file: user already has read access to any path they can name. | 04-01-PLAN.md | 2026-08-03 |
| AR-04-04 | T-04-02-07 | No cryptographic signing/attestation for releases; out of DIST-01/02/03 scope. | 04-02-PLAN.md | 2026-08-03 |
| AR-04-05 | T-04-03-06 | `--difftool-*` arbitrary local file: user already has read access. | 04-03-PLAN.md | 2026-08-03 |
| AR-04-06 | T-04-03-07 | Whole-file reads for large `--difftool-remote`: streaming explicitly out of scope. | 04-03-PLAN.md | 2026-08-03 |
| AR-04-07 | T-04-03-SC | `termenv` promoted to direct dependency; no new module, already vetted Phase 3. | 04-03-PLAN.md | 2026-08-03 |
| AR-04-08 | T-04-04-06 | Partial multi-key config write on interruption: each `git config` call atomic, re-run converges. | 04-04-PLAN.md | 2026-08-03 |
| AR-04-09 | T-04-04-SC | No new dependency added by 04-04. | 04-04-PLAN.md | 2026-08-03 |
| AR-04-10 | T-04-05-04 | ANSI-escape filename in title bar: pre-existing, `ansi.Truncate` prevents SGR-state stranding; full sanitization out of scope. | 04-05-PLAN.md | 2026-08-04 |
| AR-04-11 | T-04-05-SC | No new dependency added by 04-05 (`charmbracelet/x/ansi` already direct/vetted). | 04-05-PLAN.md | 2026-08-04 |
| AR-04-12 | T-04-06-04 | No explicit recursion-depth guard: dispatch vector removed at its only reachable call site. | 04-06-PLAN.md | 2026-08-05 |
| AR-04-13 | T-04-06-05 | git's stderr surfaced verbatim: unchanged by this plan, contains only user-supplied paths. | 04-06-PLAN.md | 2026-08-05 |
| AR-04-14 | T-04-06-SC | No new dependency added by 04-06. | 04-06-PLAN.md | 2026-08-05 |
| AR-04-15 | T-04-07-02 | Global gitconfig trust key shared across difftools: pre-existing unconditional overwrite behavior, unchanged blast radius. | 04-07-PLAN.md | 2026-08-06 |
| AR-04-16 | T-04-07-04 | `difftool.trustExitCode` set to `false` (G-04-1 fix): multi-file `git difftool` sessions no longer stop on abort. User-locked decision, direction (1) over (2); alturd's own exit code still signals abort for non-git callers. | User (2026-08-06), 04-07-PLAN.md | 2026-08-06 |
| AR-04-17 | T-04-07-SC | No new dependency added by 04-07. | 04-07-PLAN.md | 2026-08-06 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-06 | 49 | 49 | 0 | gsd-security-auditor (opus) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-06
