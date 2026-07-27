---
phase: 4
slug: config-theming-difftool-distribution
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
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
| **Quick run command** | `go test ./internal/config/... ./cmd/alturd/... -run TestConfig -v` (once `internal/config` exists — Wave 0) |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~1s today (Phase 1-3 baseline, measured `go test -count=1 ./...`); expect ~2-4s once Phase 4's config/theme/difftool tests are added |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/config/... ./cmd/alturd/...` (or whichever package the task touched)
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green, plus `goreleaser check` passing against the new `.goreleaser.yaml`
- **Max feedback latency:** ~5 seconds

---

## Per-Task Verification Map

*Task ID / Plan / Wave are TBD — assigned by the planner as it creates tasks. Requirement → Test mapping below is seeded from `04-RESEARCH.md`'s Validation Architecture section; the planner fills in Task ID/Plan/Wave/Threat Ref columns as each task is authored.*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | CONFIG-01 | — | Unknown TOML key rejected at startup with clear error | unit | `go test ./internal/config/... -run TestLoad_UnknownField -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | CONFIG-01 | — | `--config <path>` overrides default XDG lookup | unit | `go test ./internal/config/... -run TestLoad_ExplicitPath -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | CONFIG-01 | V5 (input validation) | No config file present → defaults used, no directory created (uses `SearchConfigFile`, not `ConfigFile`) | unit | `go test ./internal/config/... -run TestLoad_NoSideEffectsOnFirstRun -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | CONFIG-02 | — | Config overrides one keybinding, others keep defaults | unit | `go test ./internal/config/... -run TestKeybindings_PartialOverride -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | CONFIG-02 | — | Duplicate/conflicting key bindings rejected | unit | `go test ./internal/config/... -run TestKeybindings_DuplicateRejected -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | THEME-01 | — | Auto-detect falls back to dark within 50ms when OSC 11 does not respond | unit | `go test ./cmd/alturd/... -run TestDetectDarkBackground_TimeoutFallback -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | THEME-01 | — | `--theme`/config/auto/dark-fallback precedence order (D-07) | unit | `go test ./internal/config/... -run TestTheme_Precedence -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | DIFFTOOL-01 | — | `--difftool-local/-remote/-path` renders single-file view without tree | integration (`TestMain` subprocess pattern) | `go test ./cmd/alturd/... -run TestDifftoolMode -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | DIFFTOOL-02 | — | Title bar shows "N of M" from `GIT_DIFF_PATH_COUNTER`/`_TOTAL` env vars | unit (model-level) | `go test ./internal/tui/... -run TestDifftoolTitleBar -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | DIFFTOOL-03 | Tampering (shell injection via `difftool.<name>.cmd`) | `install-difftool` writes 4 keys idempotently; `--force` semantics; static `cmd` string, no interpolation | integration (`TestMain` subprocess pattern) | `go test ./cmd/alturd/... -run TestInstallDifftool -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | DIST-01 | — | `go test ./...` runs on Linux/macOS/Windows | CI-only | `go test ./...` (in `ci.yml` matrix) | N/A — CI config | ⬜ pending |
| TBD | TBD | TBD | DIST-02, DIST-03 | — | `CGO_ENABLED=0` binaries build cleanly for the full platform matrix | manual / CI smoke | `goreleaser check && goreleaser build --snapshot --clean` | N/A — build tooling | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/config/config_test.go` — stubs for CONFIG-01, CONFIG-02
- [ ] `internal/config/theme_test.go` — stubs for THEME-01 (precedence + timeout-fallback)
- [ ] `cmd/alturd/difftool_test.go` (or extend `main_test.go`) — stubs for DIFFTOOL-01, DIFFTOOL-03 via the existing `TestMain` subprocess pattern (Phase 2 Plan 3 decision)
- [ ] `internal/tui/model_test.go` extension — stub for DIFFTOOL-02 title bar format
- [ ] No new test framework install needed — `go test` is already fully wired project-wide

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| CI matrix actually passes on real Linux/macOS/Windows runners | DIST-01 | GitHub Actions runners are the only true multi-OS environment; local dev machine is single-OS and `go test` passing locally does not prove the workflow YAML/matrix is correct | Push branch or open a PR, confirm all 3 OS jobs show green checks in the GitHub Actions tab |
| goreleaser produces real GitHub Release assets from a tag push | DIST-02, DIST-03 | Real Release creation requires a `GITHUB_TOKEN` and an actual tag push; `goreleaser build --snapshot` only proves local cross-compilation, not the release/upload path | Push a test tag (e.g. `v0.0.0-test`) or run `goreleaser release --skip=publish` as a dry run; confirm the release workflow triggers and the expected per-platform archives appear |
| Difftool side-by-side rendering looks correct in a real terminal | DIFFTOOL-01 | Terminal layout/ANSI rendering can't be fully asserted by `go test` string assertions alone | Run `git difftool -t alturd <file>` in an actual terminal against a real repo change; visually confirm layout matches normal mode minus the file tree, and the title bar format |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
