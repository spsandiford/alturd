---
phase: 04-config-theming-difftool-distribution
plan: 02
subsystem: infra
tags: [github-actions, goreleaser, golangci-lint, ci-cd, distribution]

# Dependency graph
requires: []
provides:
  - "Three-OS (Linux/macOS/Windows) go test matrix on every push and pull_request, fail-fast: false"
  - "Independent golangci-lint gate (staticcheck, govet, errcheck, revive) on the same triggers"
  - "goreleaser pipeline publishing 5 CGO-disabled, trimpathed, version-stamped binaries on v*.*.* tag push"
  - "checksums.txt manifest covering every published release artifact"
affects: [distribution, release-process]

# Tech tracking
tech-stack:
  added: [goreleaser v2 (build tool, not a Go module), golangci-lint v2.7.2 (lint tool, not a Go module)]
  patterns:
    - "Single goreleaser builds: entry with matrix goos/goarch minus one ignore rule — structurally impossible for one artifact to diverge in CGO/trimpath settings from another"
    - "Independent CI jobs (test matrix + lint) with no needs: edge, so overall conclusion is the AND of all legs regardless of finish order"

key-files:
  created:
    - .github/workflows/ci.yml
    - .github/workflows/release.yml
    - .golangci.yml
    - .goreleaser.yaml
  modified:
    - cmd/alturd/main.go
    - cmd/alturd/main_test.go
    - internal/config/config.go
    - internal/diff/align.go
    - internal/diff/align_test.go
    - internal/diff/parse_test.go
    - internal/diff/render.go
    - internal/diff/render_test.go
    - internal/log/log.go
    - internal/log/log_test.go
    - internal/tui/model.go
    - internal/tui/model_test.go

key-decisions:
  - "golangci-lint v2.7.2 and goreleaser v2.17.1 were both installed locally via `go install` (network available in this environment) and used to actually validate both config files rather than relying on structural grep checks alone — DIST-02/DIST-03 are verified, not merely structurally plausible"
  - "All 20 golangci-lint findings against pre-existing Phase 1-3/04-01 code were fixed in Task 1's commit per the plan's explicit action instruction, so the new CI gate lands green on first run rather than red"
  - "tui.NewModel's unexported-return finding was suppressed with a targeted //nolint:revive rather than exporting the model type — this is a deliberate encapsulation choice (callers only need the tea.Model interface), not a bug, and renaming would have touched ~40 references across model.go/model_test.go well outside this plan's infra scope"

patterns-established:
  - "Pattern: fix defer f.Close() / os.Remove() errcheck findings via `defer func() { _ = f.Close() }()`, consistent across production and test code"
  - "Pattern: rely on Go 1.21+ builtin min/max instead of local redefinitions"

requirements-completed: [DIST-01, DIST-02, DIST-03]

coverage:
  - id: D1
    description: "CI runs go test ./... on ubuntu-latest, macos-latest, windows-latest for every push and pull_request, with fail-fast: false so one OS failure doesn't mask the others"
    requirement: "DIST-01"
    verification:
      - kind: other
        ref: "grep-verified .github/workflows/ci.yml matrix (os: [ubuntu-latest, macos-latest, windows-latest], fail-fast: false); go vet ./... and go test ./... run clean locally"
        status: pass
    human_judgment: false
  - id: D2
    description: "CI also runs golangci-lint (staticcheck, govet, errcheck, revive) as an independent gate on the same triggers"
    requirement: "DIST-01"
    verification:
      - kind: other
        ref: "golangci-lint v2.7.2 run ./... locally: 0 issues (after fixing 20 pre-existing findings)"
        status: pass
    human_judgment: false
  - id: D3
    description: "A v*.*.* tag push triggers goreleaser to publish Linux/macOS amd64+arm64 and Windows amd64 binaries (5 total) as GitHub Release assets, checksummed"
    requirement: "DIST-02"
    verification:
      - kind: other
        ref: "goreleaser check (exit 0) + goreleaser build --snapshot --clean produced exactly 5 binaries under dist/ (darwin_amd64, darwin_arm64, linux_amd64, linux_arm64, windows_amd64) matching the tar.gz/zip archive spec and checksums.txt template"
        status: pass
    human_judgment: false
  - id: D4
    description: "All published binaries are CGO_ENABLED=0 and -trimpath, version-stamped via -X main.version={{.Version}}"
    requirement: "DIST-03"
    verification:
      - kind: other
        ref: "ldd dist/alturd_linux_amd64_v1/alturd reports 'not a dynamic executable'; dist/alturd_linux_amd64_v1/alturd --version reports the injected snapshot version string"
        status: pass
    human_judgment: false

duration: ~20min
completed: 2026-08-03
status: complete
---

# Phase 4 Plan 2: CI/CD and Distribution Infrastructure Summary

**Three-OS CI test matrix plus golangci-lint gate, and a goreleaser pipeline validated end-to-end (goreleaser check + snapshot build + ldd) to publish 5 CGO-disabled, checksummed binaries on semver tag push**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-08-03T21:05:05Z
- **Tasks:** 2
- **Files created:** 4
- **Files modified:** 12 (pre-existing Phase 1-3/04-01 code, lint fixes only)

## Accomplishments

- `.github/workflows/ci.yml`: `test` job (3-OS matrix, `fail-fast: false`) + independent `lint` job, both triggered on bare `push:`/`pull_request:` with no branch or path filters, top-level `permissions: contents: read`
- `.golangci.yml`: v2 schema (`version: "2"`) enabling exactly staticcheck, govet, errcheck, revive
- `.goreleaser.yaml`: single `builds:` entry (`id: alturd`, `CGO_ENABLED=0`, `-trimpath`, `-X main.version={{.Version}}`) across linux/darwin/windows × amd64/arm64 minus the windows/arm64 `ignore` rule — 5 artifacts; `tar.gz` archives (`zip` on Windows); `checksums.txt` manifest
- `.github/workflows/release.yml`: `goreleaser` job triggered only on `v*.*.*` tag pushes, `contents: write` scoped to this workflow only, `GITHUB_TOKEN` supplied in exactly one step's `env:` block
- Fixed all 20 pre-existing golangci-lint findings across Phase 1-3 and 04-01 code so the new CI lint gate is green on first run, not red

## Task Commits

Each task was committed atomically:

1. **Task 1: CI matrix and lint gate (DIST-01, D-12, D-13)** - `614e9fb` (feat)
2. **Task 2: goreleaser config and tag-triggered release workflow (DIST-02, DIST-03, D-11)** - `b368638` (feat)

_This SUMMARY commit follows as the plan-metadata commit._

## Files Created/Modified

- `.github/workflows/ci.yml` - 3-OS `go test ./...` matrix + independent golangci-lint job
- `.golangci.yml` - golangci-lint v2 schema, staticcheck/govet/errcheck/revive enabled
- `.goreleaser.yaml` - single-build-entry CGO-disabled cross-platform matrix, archives, checksums
- `.github/workflows/release.yml` - goreleaser-action on `v*.*.*` tag push
- `cmd/alturd/main.go`, `cmd/alturd/main_test.go` - errcheck fixes (`defer logFile.Close()` → checked-close wrapper; `os.RemoveAll` return checked)
- `internal/config/config.go` - errcheck fix (`defer f.Close()` → checked-close wrapper)
- `internal/diff/align.go` - `mode` parameters explicitly marked reserved-for-forward-compatibility (revive unused-parameter)
- `internal/diff/align_test.go`, `internal/diff/parse_test.go` - errcheck fixes on fixture-file `Close()`
- `internal/diff/render.go` - removed dead `renderPair` function (unused linter); renamed `old`/`new` params to `oldText`/`newText` (revive redefines-builtin)
- `internal/diff/render_test.go` - removed local `min` helper, redundant with Go 1.21+ builtin (revive redefines-builtin)
- `internal/log/log.go`, `internal/log/log_test.go` - errcheck fixes on `tmp.Close()`/`os.Remove()`/`f.Close()`
- `internal/tui/model.go`, `internal/tui/model_test.go` - removed local `max` helper (builtin shadow); rewrote two empty-body decrement/increment loops with explicit bodies (revive empty-block); targeted `//nolint:revive` on `NewModel`'s intentional unexported return; errcheck fix on fixture-file `Close()`

## Decisions Made

- Installed `golangci-lint@v2.7.2` and `goreleaser@latest` (resolved to v2.17.1) locally via `go install` since network access was available, enabling real tool-backed verification (`golangci-lint run ./...` → 0 issues; `goreleaser check` → exit 0; `goreleaser build --snapshot --clean` → 5 binaries; `ldd` confirmed static linking) rather than falling back to the plan's structural-grep-only contingency.
- All 20 lint findings in pre-existing code were fixed as directed by Task 1's `<action>` block, using minimal, behavior-preserving changes (checked-close wrappers, builtin shadowing removal, explicit loop bodies) rather than broader refactors.
- `tui.NewModel`'s `unexported-return` finding was suppressed with a targeted `//nolint:revive` comment instead of exporting the `model` type, since that rename would have touched ~40 references in a TUI file well outside this plan's declared file scope (`.github/workflows/*`, `.golangci.yml`, `.goreleaser.yaml`) for a stylistic (not correctness) finding.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1/Task-directed - Bug/Style] Fixed all pre-existing golangci-lint findings in Phase 1-3/04-01 code**
- **Found during:** Task 1 (running `golangci-lint run ./...` per the plan's explicit action instruction)
- **Issue:** 20 findings across 12 files: 8 errcheck (unchecked `Close`/`Remove` return values), 8 revive (2 unused-parameter, 3 redefines-builtin, 2 empty-block, 1 unexported-return), 1 unused (dead `renderPair` function)
- **Fix:** Checked-close wrapper closures for errcheck; builtin-shadowing helpers removed (`min`/`max` redundant since Go 1.21+); `old`/`new` params renamed off builtin names; empty decrement/increment loops rewritten with explicit bodies; dead `renderPair` removed (confirmed unused anywhere via grep); `NewModel`'s intentional unexported return suppressed with a scoped `//nolint:revive`
- **Files modified:** cmd/alturd/main.go, cmd/alturd/main_test.go, internal/config/config.go, internal/diff/align.go, internal/diff/align_test.go, internal/diff/parse_test.go, internal/diff/render.go, internal/diff/render_test.go, internal/log/log.go, internal/log/log_test.go, internal/tui/model.go, internal/tui/model_test.go
- **Verification:** `go build ./...`, `go vet ./...`, `go test ./...` all pass; `golangci-lint run ./...` reports 0 issues
- **Committed in:** `614e9fb` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed batch (task-directed lint cleanup, explicitly called for by the plan's own action text)
**Impact on plan:** No scope creep — every change was a minimal, behavior-preserving lint fix with no logic changes; all existing tests still pass unchanged.

## Issues Encountered

None. Both `goreleaser` and `golangci-lint` binaries were successfully installed locally (network was available in this execution environment), so both tasks were verified with the actual tools rather than falling back to the plan's "record as first-run risk" contingency.

## User Setup Required

None - no external service configuration required. (Publishing a real release still requires pushing a `v*.*.*` git tag to a repository with GitHub Actions enabled and a `GITHUB_TOKEN` — standard repo-level setup, not a Phase 4 blocker.)

## Next Phase Readiness

- CI and release infrastructure is complete and locally validated; the first real tag push (e.g. `v0.1.0`) will exercise the actual GitHub Actions runners for the first time, which is the only remaining unverified surface (this is expected — GitHub-hosted runner behavior cannot be fully simulated locally).
- No blockers for sibling plans 04-01 (config/keybindings, already complete), 04-03 (theming), or 04-04 (difftool) — this plan shares no source files with any of them.

---
*Phase: 04-config-theming-difftool-distribution*
*Completed: 2026-08-03*

## Self-Check: PASSED

All created files verified present on disk (.github/workflows/ci.yml, .github/workflows/release.yml, .golangci.yml, .goreleaser.yaml, this SUMMARY.md). Both task commits (614e9fb, b368638) verified present in git log.
