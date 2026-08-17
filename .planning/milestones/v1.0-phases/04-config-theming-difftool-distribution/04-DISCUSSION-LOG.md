# Phase 4: Config + Theming + Difftool + Distribution - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-27
**Phase:** 4-Config + Theming + Difftool + Distribution
**Areas discussed:** Config file design, Theme behavior, Difftool setup & install-difftool, Release pipeline conventions

---

## Config file design

### Q: TOML keybinding schema shape

| Option | Description | Selected |
|--------|-------------|----------|
| Flat [keybindings] table | One flat table, e.g. `next_hunk = "n"`; matches lazygit/k9s conventions | |
| Grouped by pane | `[keybindings.tree]` / `[keybindings.diff]` / `[keybindings.global]` sub-tables | |
| You decide | Claude picks based on go-toml/v2 idioms and DisallowUnknownFields validation | ✓ |

**User's choice:** You decide.
**Notes:** Leaning documented in CONTEXT.md D-04 as a recommendation (flat table), not a lock.

### Q: Partial override vs. complete set

| Option | Description | Selected |
|--------|-------------|----------|
| Merge with defaults | Unspecified actions keep default key | ✓ |
| Require complete set | User must define every action or startup fails | |

**User's choice:** Merge with defaults.

### Q: Strictness of keybinding value validation

| Option | Description | Selected |
|--------|-------------|----------|
| Reject both cases at startup | Unknown field names AND unrecognized key strings AND duplicate bindings all fail fast | ✓ |
| Reject unknown fields only | TOML schema strictness only; key-string validity/duplicates unchecked | |

**User's choice:** Reject both cases at startup.

### Q: Missing config file behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Silently use defaults | No file required; nothing written to disk | ✓ |
| Create default config on first run | Writes a fully-commented template config.toml | |

**User's choice:** Silently use defaults.

---

## Theme behavior

### Q: Manual theme override vs. auto-only

| Option | Description | Selected |
|--------|-------------|----------|
| Add manual override | `theme = "light"\|"dark"\|"auto"` config key + `--theme` flag | ✓ |
| Auto-only, no override | Keep Phase 3's termenv auto-detect + dark fallback as-is | |

**User's choice:** Add manual override.

### Q: OSC 11 in difftool mode

| Option | Description | Selected |
|--------|-------------|----------|
| Skip OSC 11 in difftool mode | Use config/flag override or dark fallback; never attempt OSC 11 round-trip as a git subprocess | ✓ |
| Still attempt OSC 11 in difftool mode | Same detection logic regardless of mode, protected by existing 50ms timeout | |

**User's choice:** Skip OSC 11 in difftool mode.
**Notes:** Grounded in `.planning/research/PITFALLS.md` Pitfall 10 (OSC 11 hang/garbage risk in subprocess/tmux/SSH contexts).

---

## Difftool setup & install-difftool

### Q: Which four canonical gitconfig keys?

| Option | Description | Selected |
|--------|-------------|----------|
| diff.tool + difftool.<name>.cmd + difftool.prompt=false + difftool.trustExitCode=true | Standard git custom-difftool pattern | |
| You decide | Claude/researcher determines exact keys from git-scm.com docs | ✓ |

**User's choice:** You decide.
**Notes:** Python reference implementation unavailable in this repo to copy exact keys from (noted since Phase 1). The standard-pattern option's description was captured in CONTEXT.md D-08 as the working starting point pending researcher verification.

### Q: Default --scope and --name

| Option | Description | Selected |
|--------|-------------|----------|
| scope=global, name=alturd | Writes to ~/.gitconfig by default | ✓ |
| scope=local, name=alturd | Writes to current repo's .git/config by default | |

**User's choice:** scope=global, name=alturd.

### Q: --force semantics / idempotency contract

| Option | Description | Selected |
|--------|-------------|----------|
| Without --force: safe no-op/update own keys; --force only for conflicting existing diff.tool | Re-running always safely refreshes alturd's own keys | ✓ |
| Without --force: fail if any of the 4 keys exist at all; --force always required | More conservative, requires --force on every re-run | |

**User's choice:** Safe no-op/update own keys without --force; --force only for conflicts.

---

## Release pipeline conventions

### Q: Tag pattern to trigger release

| Option | Description | Selected |
|--------|-------------|----------|
| v*.*.* (semver) | Standard goreleaser convention; matches existing `var version` ldflags stub | ✓ |
| You decide | Claude picks the conventional pattern | |

**User's choice:** v*.*.* (semver).

### Q: CI trigger scope

| Option | Description | Selected |
|--------|-------------|----------|
| On push AND pull_request | Catches issues in PRs before merge | ✓ |
| Push only, per DIST-01 literal wording | Matches requirement text exactly | |

**User's choice:** On push AND pull_request.

### Q: golangci-lint as a CI gate

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, add golangci-lint as a CI gate | Matches CLAUDE.md tooling recommendation | ✓ |
| No, test-only per DIST-01 | Keep CI scope literally to go test ./... | |

**User's choice:** Yes, add golangci-lint as a CI gate.

---

## Claude's Discretion

- Exact TOML schema shape for keybindings (flat vs. grouped-by-pane table).
- Exact gitconfig keys written by `install-difftool` — verify against git-scm.com difftool docs before locking.
- CLI flag naming/spelling details not explicitly discussed (e.g. `--theme` flag short form).
- `.golangci.yml` ruleset specifics beyond the four tools named in CLAUDE.md (staticcheck, govet, errcheck, revive).

## Deferred Ideas

None — discussion stayed within phase scope.
