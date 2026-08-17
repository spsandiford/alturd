---
phase: 03
slug: tui-application
status: verified
threats_open: 0
asvs_level: 1
created: 2026-07-24
---

# Phase 03 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| git subprocess → diff parser | Untrusted repo content already crosses here; mitigated in Phase 1/2 (typed gitdiff structs, no shell). No new boundary added by Plan 01. | Unified diff text; typed via go-gitdiff |
| user search query → matcher | The typed query is untrusted text; used only as a needle for strings.Index over ANSI-stripped content; never interpolated into ANSI sequences, shell, or templates | Plaintext string; length-bounded by TUI input |
| diff content → terminal viewport | Rendered ANSI comes from diff.Render (known, Phase-1-controlled escapes); the model does not synthesize new escapes from untrusted input | ANSI escape sequences; source-controlled vocab |
| CLI args → git subprocess | Ref/path args parsed by git.ParseRefArgs and passed as argv to ExecRunner (Phase 2 chokepoint, unchanged) | git rev/path strings; exec.Command argv form |
| git ls-tree output → tree model | File paths returned by git ls-tree are used as display strings only; never shell-interpolated | Repo path strings; display-only |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-03-SEARCH | Tampering | findMatches / search query → ANSI output | low | mitigate | Query is a needle for `strings.Index` over `ansi.Strip`'d content (`internal/tui/search.go:26,32`); never written back into ANSI/escape sequences; L1 grep confirmed | closed |
| T-03-03 | Tampering | Terminal escape injection via diff content in viewport | low | accept | See Accepted Risks Log | closed |
| T-03-05 | Tampering | Shell injection via git ls-tree invocation | medium | mitigate | `model.go:612–614` uses `git.ExecRunner{}.Run([]string{"ls-tree","-r","--full-tree","--name-only","HEAD"})` — argv form (exec.Command), no user input in argument list; L1 grep confirmed | closed |
| T-03-06 | Tampering | Shell injection via CLI ref/path args | medium | mitigate | `main.go:52–55` uses `ParseRefArgs` + `ExecRunner{}.Run(gitArgs)` — argv form; no shell interpolation; unchanged from Phase 2 chokepoint; L1 grep confirmed | closed |

*Status: open · closed · open — below {block_on} threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-03-01 | T-03-03 | diffVP content is produced by `diff.Render` (internal, Phase-1-controlled) whose ANSI vocabulary is fixed and reset-guarded. No untrusted string is turned into escape sequences inside the viewport; the risk surface is entirely within the controlled diff rendering pipeline. | Plan author (03-03-PLAN.md) | 2026-07-24 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-24 | 4 | 4 | 0 | gsd-secure-phase (Claude, ASVS L1) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-07-24
