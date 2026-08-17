---
phase: 01
slug: diff-model
status: verified
threats_open: 0
asvs_level: 1
created: 2026-06-29
---

# Phase 01 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| go module proxy → local build | Third-party module source and checksums cross into the build | Compiled code; cryptographic checksums |
| fixture file → test process | Fixture `.diff` bytes read by `go test` (test-controlled, not end-user input) | LF-normalized diff text; no user-supplied data |
| diff text (fixture, later git subprocess) → Parse | Untrusted/arbitrary diff bytes enter the library | Arbitrary byte sequences |
| parsed File → Align | Typed gitdiff.File structs walked to build row model | Typed struct fields; no raw bytes after Parse |
| RowPair content → DiffMain | Arbitrary line content fed to character-level diff | Line text strings (UTF-8) |
| RowPair content → Chroma | Arbitrary content fed to the tokenizer | Line text strings (UTF-8) |
| binary fragment → Render | Non-text bytes must not enter the text render path | Binary blob content (must be gated) |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-01-SC | Tampering | `go get` of go-gitdiff / go-diff / chroma | high | mitigate | Exact versions pinned (v0.8.1, v1.4.0, v2.27.0) in `go.mod`; cryptographic checksums in `go.sum` (2624 bytes); all three packages cleared in RESEARCH Package Legitimacy Audit | closed |
| T-01-01 | Tampering | Fixture file paths in tests | low | mitigate | Fixtures referenced only by hardcoded relative `testdata/` paths; never from user input | closed |
| T-01-02 | Information Disclosure | Placeholder module path `github.com/alturd/alturd` | low | accept | Non-secret placeholder; update to real path on publish — see Accepted Risks Log | closed |
| T-01-03 | Tampering | `Parse` on malformed diff input | medium | mitigate | `parse.go` wraps `gitdiff.Parse` with `fmt.Errorf("parsing diff: %w", err)`; never panics; `TestParseMalformed` asserts graceful error on adversarial input (ASVS V5) | closed |
| T-01-04 | Denial of Service | `Align` on empty/nil TextFragments (mode-only, binary, submodule) | low | mitigate | `align.go` explicit early-exit branches for `IsBinary`, `isModeOnly`, `isSubmodule` before any line walking; nil-safe throughout | closed |
| T-01-05 | Tampering | Submodule SHA lines mis-aligned as Modified pairs | low | mitigate | `isSubmodule(f)` branch passes raw lines through as one-sided rows; no Modified pairing on 40-char SHAs; verified by `submodule_raw_context_no_modified` test | closed |
| T-01-06 | Denial of Service | `DiffMain` on extremely long/complex lines | high | mitigate | `shouldSkipIntraLine` pre-guards (>1000 chars OR >200 tokens) plus `computeIntraLineWithTimeout` with 100ms goroutine deadline in `render.go`; verified by `large-line.diff` and `many-tokens.diff` tests (D-07; ASVS V5) | closed |
| T-01-07 | Tampering | Binary bytes rendered as text | medium | mitigate | `isPlaceholderPairs` early-return in `highlight.go`; `IsBinary` early-exit in `align.go` ensures binary content never reaches line-split or chroma tokenization | closed |
| T-01-08 | Information Disclosure | Chroma color bleed across side-by-side columns | low | mitigate | Explicit `ansiReset` (`\x1b[0m`) at left-column boundary in `joinColumns` (`render.go:196`); per-line reset appended by `splitAndReset` in `highlight.go` | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `high` count toward `threats_open`*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-01-01 | T-01-02 | Module path `github.com/alturd/alturd` is a non-secret placeholder per RESEARCH assumption A4. It carries no credentials, PII, or internal infrastructure information. Risk is informational only; update to real repo path before public release. | Patrick Sandiford | 2026-06-29 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-06-29 | 9 | 9 | 0 | gsd-security-auditor (L1 grep; asvs_level=1; short-circuit rule applied — register_authored_at_plan_time=true, threats_open=0) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log (AR-01-01)
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-06-29
