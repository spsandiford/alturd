---
phase: "02"
slug: git-layer-cli
status: verified
threats_open: 0
asvs_level: 1
created: 2026-07-01
---

# Phase 02 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| CLI args → ExecRunner | User-supplied ref/path strings cross into exec.Command argv form | ref strings, path filters (untrusted) |
| git subprocess stdout → diff parser | git subprocess output crosses back into the application | raw unified diff bytes (untrusted) |
| XDG_STATE_HOME env → log file path | Process-owner-controlled env var influences the log file location | file system path |
| RunE error → main()/os.Exit | Error type and message cross to the process exit code and stderr output | error strings |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-02-01 | Tampering | ExecRunner.Run subprocess invocation | high | mitigate | `exec.Command("git", args...)` argv form; no `sh -c`; each ref/path is a distinct argv element with no shell injection surface (ASVS V5) | closed |
| T-02-02 | Tampering | ParseRefArgs path-filter handling | medium | mitigate | Path args passed verbatim to git after the `--` separator (re-inserted by ParseRefArgs); no shell interpolation; git's own path validation applies | closed |
| T-02-07 | Information Disclosure | main() error output | medium | mitigate | `SilenceErrors: true` / `SilenceUsage: true` suppress cobra's verbose dump; main() prints exactly one line to stderr (ExitCodeError.Msg or err.Error()); raw git stderr never echoed | closed |
| T-02-08 | Tampering | RunE log-init ordering | medium | mitigate | `applog.Init()` is the first statement inside `run()`; `--version` / `--help` are handled by cobra before RunE fires, so they never create a log file (D-10); verified by no-log integration tests | closed |
| T-02-03 | Information Disclosure | ExecRunner error mapping | low | mitigate | All failures map to fixed single-line sentinel messages (`ErrGitNotFound` / `ErrNotGitRepo`); raw git stderr is discarded and never forwarded to the user | closed |
| T-02-05 | Information Disclosure | log file permissions | low | mitigate | Log file opened with `os.OpenFile(..., 0600)` — owner read/write only; other local users cannot read path-bearing log content | closed |
| T-02-06 | Denial of Service | unbounded log growth | low | mitigate | 1 MB tail-truncation cap enforced on every `applog.Init()` call at startup; `os.Truncate` not used — tail bytes (recent entries) are retained | closed |
| T-02-09 | Repudiation | exit-code routing | low | mitigate | `errors.As(err, &exitErr)` for `*git.ExitCodeError` routes typed exit codes (0 success / 1 not-a-repo / 127 git-not-found) deterministically; callers and CI can script on these codes | closed |
| T-02-04 | Elevation of Privilege | xdg.StateFile log-path resolution | low | accept | `xdg.StateFile` returns a path under the process owner's own state directory; an attacker controlling `XDG_STATE_HOME` already runs as that user and can only affect files they own — no privilege boundary crossed | closed |
| T-02-SC | Tampering | Go module dependencies | low | accept | All 5 new external modules pinned to audited versions: `github.com/adrg/xdg v0.5.3`, `github.com/charmbracelet/log v1.0.0`, `github.com/spf13/cobra v1.10.2`, `golang.org/x/term v0.44.0`; Plan 01 uses stdlib only; Package Legitimacy Audit in RESEARCH.md confirmed OK via proxy.golang.org | closed |

*Status: open · closed · open — below {block_on} threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `workflow.security_block_on` (`high`) count toward `threats_open`*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-02-01 | T-02-04 | `xdg.StateFile` resolves inside the process owner's own XDG state directory. An attacker controlling `XDG_STATE_HOME` already has equivalent access as that user — no privilege escalation possible. | gsd-security-auditor (L1 automated) | 2026-07-01 |
| AR-02-02 | T-02-SC | All five new module dependencies pinned to specific versions that passed the RESEARCH.md Package Legitimacy Audit. Supply-chain risk accepted at project bootstrap level; no new trust boundary introduced by these libraries. | gsd-security-auditor (L1 automated) | 2026-07-01 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-01 | 10 | 10 | 0 | gsd-security-auditor (L1, ASVS level 1, automated grep-depth) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-07-01
