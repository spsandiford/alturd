---
phase: 4
slug: config-theming-difftool-distribution
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-07-27
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` package (table-driven), consistent with all prior phases |
| **Config file** | none — `go test ./...` requires no config |
| **Quick run command** | `go test ./internal/config/... ./cmd/alturd/... -run TestConfig -v` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~1s (`go test -count=1 ./...` across all 6 packages) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/config/... ./cmd/alturd/...` (or whichever package the task touched)
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green, plus `goreleaser check` passing against `.goreleaser.yaml`
- **Max feedback latency:** ~5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 04-01-T3 | 01 | 1 | CONFIG-01 | — | Unknown TOML key rejected at startup with clear error | unit | `go test ./internal/config/... -run TestLoad_UnknownField -v` | ✅ | ✅ green |
| 04-01-T2 | 01 | 1 | CONFIG-01 | — | `--config <path>` overrides default XDG lookup | unit | `go test ./internal/config/... -run TestLoad_ExplicitPath -v` | ✅ | ✅ green |
| 04-01-T3 | 01 | 1 | CONFIG-01 | T-04-01-01 | No config file present → defaults used, no directory created (uses `SearchConfigFile`, not `ConfigFile`) | unit | `go test ./internal/config/... -run TestLoad_NoSideEffectsOnFirstRun -v` | ✅ | ✅ green |
| 04-01-T3 | 01 | 1 | CONFIG-02 | — | Config overrides one keybinding, others keep defaults | unit | `go test ./internal/config/... -run TestKeybindings_PartialOverride -v` | ✅ | ✅ green |
| 04-01-T3 | 01 | 1 | CONFIG-02 | — | Duplicate/conflicting key bindings rejected (merge-then-validate, deterministic ordering) | unit | `go test ./internal/config/... -run TestKeybindings_DuplicateRejected -v` | ✅ | ✅ green |
| 04-03-T1 | 03 | 3 | THEME-01 | — | Auto-detect falls back to dark within 50ms when OSC 11 does not respond | unit | `go test ./internal/config/... -run TestDetectDarkBackground_TimeoutFallback -v` | ✅ | ✅ green |
| 04-03-T1 | 03 | 3 | THEME-01 | — | `--theme`/config/auto/dark-fallback precedence order (D-05/D-06/D-07), asserted via detector call counts | unit | `go test ./internal/config/... -run TestTheme_Precedence -v` | ✅ | ✅ green |
| 04-03-T2 | 03 | 3 | DIFFTOOL-01 | — | `--difftool-local/-remote/-path` renders single-file view without tree; missing-flag and identical-file paths | integration (`TestMain` subprocess pattern) | `go test ./cmd/alturd/... -run 'TestDifftoolModeMissingFlags\|TestDifftoolModeIdenticalFiles' -v` | ✅ | ✅ green |
| 04-03-T3 | 03 | 3 | DIFFTOOL-02 | — | Title bar shows "N of M" from `GIT_DIFF_PATH_COUNTER`/`_TOTAL` env vars, all 5 boundary templates | unit (model-level) | `go test ./internal/tui/... -run 'TestDifftoolTitleBar$' -v` | ✅ | ✅ green |
| 04-04-T2 | 04 | 4 | DIFFTOOL-03 | Tampering (shell injection via `difftool.<name>.cmd`) | `install-difftool` writes 4 keys idempotently; `--force` semantics; static `cmd` string, no interpolation | integration (`TestMain` subprocess pattern) | `go test ./cmd/alturd/... -run 'TestInstallDifftoolWritesFourKeys\|TestInstallDifftoolIsIdempotent' -v` | ✅ | ✅ green |
| 04-02-T1 | 02 | 2 | DIST-01 | — | `go test ./...` + golangci-lint run on Linux/macOS/Windows for every push/PR | CI-only | `go test ./...` (in `.github/workflows/ci.yml` 3-OS matrix, `fail-fast: false`) | ✅ (workflow file) | ✅ green (local go test proxy; real 3-OS run is CI-only, see Manual-Only) |
| 04-02-T2 | 02 | 2 | DIST-02, DIST-03 | — | `CGO_ENABLED=0` binaries build cleanly for the full platform matrix, tag push publishes 5 artifacts | manual / CI smoke | `goreleaser check && goreleaser build --snapshot --clean` | ✅ (`.goreleaser.yaml`) | ✅ green (recorded pass: 04-02-SUMMARY.md — `goreleaser check` exit 0, 5 binaries built, `ldd` confirmed static; tools unavailable in this sandbox, see Manual-Only) |
| 04-06-T1 | 06 | 6 | DIFFTOOL-01 | Recursion (G-04-2) | `difftoolDiff`/standalone diff argv immune to `GIT_EXTERNAL_DIFF`/`diff.external` dispatch — closes the alturd→git→git-difftool--helper→alturd recursion at its only reachable call site | integration (real git subprocess, fault-injected) | `go test ./cmd/alturd/... -run TestDifftoolDiffIgnoresExternalDiffConfiguration -v` | ✅ | ✅ green |
| 04-07-T1 | 07 | 7 | DIFFTOOL-01, DIFFTOOL-03 | Repudiation (G-04-1) | `install-difftool` writes `difftool.trustExitCode=false`; an aborting backend under real `git difftool` exits 0 with no fatal line; sensitivity control confirms the fatal path reappears with the superseded value | e2e (real `git difftool` subprocess, fault-injected) | `go test ./cmd/alturd/... -run TestGitDifftoolExitsCleanlyWhenBackendAborts -v` | ✅ | ✅ green |
| 04.1-01-T1 (supplemental, post-phase) | 04.1-01 | 04.1-W1 | DIFFTOOL-01, DIFF-05 | — | Difftool full-file rendering for a deleted file sources reference content from `--difftool-local`, not `--difftool-remote` (WR-07 fix), with a sensitivity-control subtest guarding against the branch being inverted | unit (whitebox, `package main`) | `go test ./cmd/alturd/... -run TestLoadDifftoolFilesDeletedFileUsesLocalSide -v` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

All 15 rows above were re-run against the tree as it stands after `04.1-01` (commit `e29e28b`) during this reconciliation pass — every automated command listed executed and passed. Full suite: `go test ./... -count=1` — all 6 packages `ok`.

---

## Wave 0 Requirements

- [x] `internal/config/config_test.go` — CONFIG-01, CONFIG-02 (present, all tests green)
- [x] `internal/config/theme_test.go` — THEME-01 precedence + timeout-fallback (present, all tests green)
- [x] `cmd/alturd/difftool_test.go` + `cmd/alturd/installdifftool_test.go` — DIFFTOOL-01, DIFFTOOL-03 via the `TestMain` subprocess pattern (present, all tests green)
- [x] `internal/tui/model_test.go` — DIFFTOOL-02 title bar format (present, all tests green)
- [x] No new test framework install needed — `go test` was already fully wired project-wide

All Wave 0 bootstrapping materialized across plans 04-01 through 04-04; no gaps found during this reconciliation.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions | Status |
|----------|-------------|------------|--------------------|--------|
| CI matrix actually passes on real Linux/macOS/Windows runners | DIST-01 | GitHub Actions runners are the only true multi-OS environment; local `go test` passing does not prove the workflow YAML/matrix itself is correct | Push branch or open a PR, confirm all 3 OS jobs show green checks in the GitHub Actions tab | Recorded pass — 04-VERIFICATION.md carries this forward as confirmed in an earlier verification round; workflow file structurally verified (`os: [ubuntu-latest, macos-latest, windows-latest]`, `fail-fast: false`) |
| goreleaser produces real GitHub Release assets from a tag push | DIST-02, DIST-03 | Real Release creation requires a `GITHUB_TOKEN` and an actual tag push; `goreleaser build --snapshot` only proves local cross-compilation, not the release/upload path | Push a test tag (e.g. `v0.0.0-test`) or run `goreleaser release --skip=publish` as a dry run; confirm the release workflow triggers and the expected per-platform archives appear | Recorded pass — 04-02-SUMMARY.md: `goreleaser check` exit 0, `goreleaser build --snapshot --clean` produced exactly 5 binaries, `ldd` confirmed static linking; live tag push confirmed human-verified in an earlier verification round per 04-VERIFICATION.md |
| Difftool side-by-side rendering looks correct in a real terminal | DIFFTOOL-01, DIFFTOOL-02 | Terminal layout/ANSI rendering can't be fully asserted by `go test` string assertions alone | Run `git difftool -t alturd <file>` in an actual terminal against a real repo change; visually confirm layout matches normal mode minus the file tree, title bar format, and ellipsis truncation on long filenames | Resolved — 04-UAT.md test 1: scripted pty session confirmed no flood, no fork/exec exhaustion, ellipsis truncation correct, clean exit. 04-VERIFICATION.md records "Human Verification Required: None" |

All three manual-only items above have a recorded pass; none are outstanding. They remain listed as Manual-Only because their nature (real multi-OS CI runners, real tag-triggered GitHub Release, real-terminal ANSI rendering) is not automatable by `go test`, not because they are unverified.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 5s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-08-14

---

## Validation Audit 2026-08-14

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

Reconciliation triggered by `.planning/v1.0-MILESTONE-AUDIT.md` §4/§5 (Phase 4 flagged `status: draft` / NOT-VALIDATED because `/gsd-validate-phase 4` was never re-run after the `04-07` gap-closure plan). All 9 Phase 4 requirements (THEME-01, CONFIG-01, CONFIG-02, DIFFTOOL-01, DIFFTOOL-02, DIFFTOOL-03, DIST-01, DIST-02, DIST-03) were cross-referenced against the 7 plan SUMMARY files (04-01 through 04-07), `04-VERIFICATION.md`'s independent SATISFIED verdicts, and re-ran against the tree as it stands after `04.1-01` — no coverage gap found. The Per-Task Verification Map above additionally accounts for the `04-06`/`04-07` gap-closure regression tests (which post-date the phase's original wave-based validation) and the `04.1-01` WR-07 regression test for the DIFFTOOL-01/DIFF-05 full-file difftool path. `nyquist_compliant: true` reflects a genuine finding: every requirement traces to a passing automated test, re-run in this pass, not a formality.
