---
status: diagnosed
trigger: |
  G-04-2: `git difftool -t alturd <file>` recursively re-invokes `git diff --no-index`
  (or itself) instead of launching the alturd TUI, causing runaway process spawning
  until the OS fork limit is exhausted.
created: 2026-08-05T00:00:00Z
updated: 2026-08-05T00:17:00Z
---

## Current Focus

reasoning_checkpoint:
  hypothesis: "difftoolDiff() in cmd/alturd/main.go calls `exec.Command(\"git\", \"diff\", \"--no-index\", \"--\", local, remote)` without `--no-ext-diff` and without overriding Env. When alturd is invoked as a git difftool backend, git's own builtin difftool machinery unconditionally sets GIT_EXTERNAL_DIFF=git-difftool--helper in the difftool.<name>.cmd child's environment. That env var is inherited into alturd's internal `git diff --no-index` subprocess, which itself honors GIT_EXTERNAL_DIFF and re-invokes git-difftool--helper instead of diffing directly — re-triggering the whole difftool cycle (fresh $LOCAL/$REMOTE/$MERGED, re-invoking difftool.alturd.cmd = alturd again), producing unbounded recursive process spawning until fork() fails."
  confirming_evidence:
    - "Spy-script experiment A: registered a non-alturd difftool.spytool.cmd in a fresh scratch repo (only the 4 canonical install-difftool keys, no other config) and confirmed GIT_EXTERNAL_DIFF=git-difftool--helper is present in that child's env unconditionally — not user-specific config."
    - "Spy-script experiment B: ran `GIT_EXTERNAL_DIFF=spy2.sh git diff --no-index -- a.txt b.txt` directly and confirmed git diff --no-index DOES honor GIT_EXTERNAL_DIFF when set (invokes spy2.sh instead of computing its own diff) — this is the exact mechanism difftoolDiff's inner git diff --no-index subprocess would hit."
    - "Spy-script experiment C: added --no-ext-diff to the same command with GIT_EXTERNAL_DIFF still set — external program was never invoked, correct built-in diff was produced. Confirms the fix direction."
    - "grep across the repo confirms difftoolDiff (main.go:272) is the ONLY exec.Command git invocation reachable from inside a git-difftool subprocess tree (gitConfigRun and internal/git/runner.go's ExecRunner are only used in code paths that never run as a difftool.cmd child)."
  falsification_test: "If git diff --no-index did NOT honor GIT_EXTERNAL_DIFF (i.e. spy2.sh was never invoked in experiment B), or if GIT_EXTERNAL_DIFF were not actually present in the difftool.cmd child's environment (experiment A), this hypothesis would be refuted. Both experiments came out confirming, not refuting."
  fix_rationale: "Adding --no-ext-diff (or equivalently stripping GIT_EXTERNAL_DIFF/diff.external influence via Env) to the single exec.Command in difftoolDiff addresses the root cause directly: it makes alturd's internal 'diff two temp files' computation immune to whatever external-diff configuration/environment it happens to be invoked under, which is correct because that internal git diff --no-index call is only ever meant to be a pure diff-computation primitive, never an actual external-diff dispatch."
  blind_spots: "Did not run the actual full end-to-end fork-bomb reproduction (git difftool -t alturd README.md) since doing so would exhaust process/fork resources on this shared sandbox; the mechanism was instead proven via two decomposed, safe sub-experiments (A/B/C) that together fully explain the observed symptom without needing the catastrophic full run. Have not verified behavior on macOS/Windows git builds, though GIT_EXTERNAL_DIFF handling is core git behavior, not platform-specific."
  candidate_causes:
    - "code: difftoolDiff()'s exec.Command omits --no-ext-diff / does not override Env, so it inherits GIT_EXTERNAL_DIFF from its parent difftool.cmd process"
    - "environment: GIT_EXTERNAL_DIFF is set by git-core's own difftool builtin (git-difftool--helper) unconditionally for every difftool invocation — confirmed NOT a pre-existing/user-specific gitconfig artifact (experiment A used a fresh scratch repo with only install-difftool's 4 canonical keys)"
  and_gate: "no — a single code fix (adding --no-ext-diff, or an equivalent Env override, to difftoolDiff's exec.Command) fully closes the loop regardless of the environment condition; the environment condition (GIT_EXTERNAL_DIFF being set) is not something alturd controls or should rely on being absent, so it is listed as a contributing category but not a second independent thing to fix — the fix in the code category alone is sufficient."

status: root cause confirmed. goal is find_root_cause_only — stopping here, not applying a fix.
next_action: none — return ROOT CAUSE FOUND to caller

## Symptoms

expected: Running `git difftool -t alturd README.md` (after `alturd install-difftool`) in a real interactive terminal launches the alturd TUI single-file difftool view.
actual: The terminal was flooded with an unbounded, repeating chain of "git diff --no-index: git diff --no-index: ..." (thousands of repetitions on a single line), eventually hitting "fork/exec /usr/lib/git-core/git: resource temporarily unavailable" once the process/fork limit was exhausted, followed by hundreds of "fatal: external diff died, stopping at /tmp/git-blob-XXX/README.md" lines. The alturd TUI never appeared.
errors: "fork/exec /usr/lib/git-core/git: resource temporarily unavailable"; "fatal: external diff died, stopping at /tmp/git-blob-.../README.md" (git-core's own message, not found in alturd Go source)
reproduction: Run, in a real interactive terminal with a git repo containing README.md: `alturd install-difftool` then `git difftool -t alturd README.md`. (Test 2 in .planning/phases/04-config-theming-difftool-distribution/04-UAT.md)
started: Discovered during UAT of Phase 4, on fresh clone with difftool freshly installed via `alturd install-difftool`.

## Eliminated

- hypothesis: "The recursion is caused by a pre-existing GIT_EXTERNAL_DIFF/diff.external gitconfig on the user's machine, unrelated to alturd's own install-difftool config (i.e. environment-specific, not code-fixable)."
  evidence: |
    Reproduced the trigger condition (GIT_EXTERNAL_DIFF present in the difftool.cmd
    child's environment) in a completely fresh scratch repo containing ONLY the 4
    canonical keys install-difftool writes (diff.tool, difftool.<name>.cmd,
    difftool.prompt, difftool.trustExitCode) — no diff.external, no prior
    GIT_EXTERNAL_DIFF in the shell. git-core's own git-difftool builtin sets
    GIT_EXTERNAL_DIFF=git-difftool--helper unconditionally as part of how it
    dispatches to any registered difftool.cmd. This is inherent git-core behavior,
    not a user/environment artifact.
  timestamp: 2026-08-05T00:10:00Z

- hypothesis: "A shell alias/function/wrapper script named `alturd` on the user's system invokes git difftool again, causing the recursion (environment-specific, outside alturd's own code)."
  evidence: |
    Not needed as a live explanation: the recursion was fully reproduced and
    explained using a non-alturd spy script substituted for the difftool.cmd
    program, with no PATH-based alturd resolution involved at all. The mechanism
    (GIT_EXTERNAL_DIFF inherited into the internal `git diff --no-index`
    subprocess) is sufficient on its own and lives entirely in alturd's own
    difftoolDiff() code — no external alias/wrapper is required to reproduce it.
  timestamp: 2026-08-05T00:10:00Z

## Evidence

- timestamp: 2026-08-05T00:05:00Z
  checked: cmd/alturd/difftool.go and cmd/alturd/main.go (full read)
  found: |
    difftoolCmdTemplate registers `alturd --difftool-local "$LOCAL" --difftool-remote
    "$REMOTE" --difftool-path "$MERGED"` as difftool.<name>.cmd (no diff.external,
    no GIT_EXTERNAL_DIFF written anywhere by install-difftool — confirmed by grep,
    zero hits for GIT_EXTERNAL_DIFF/diff.external/no-ext-diff in the whole repo).
    difftoolDiff() (main.go ~251-295) runs exactly one subprocess:
    exec.Command("git", "diff", "--no-index", "--", local, remote) with NO Env
    override (inherits the full parent process environment) and NO --no-ext-diff
    flag. On any exit code other than 0/1 it wraps stderr/err as
    "git diff --no-index: %s"/"git diff --no-index: %w" (lines 291-294).
  implication: |
    difftoolDiff's git subprocess is unprotected against inherited external-diff
    configuration. Confirms the literal error-message prefix "git diff --no-index: "
    the user saw originates from this exact fmt.Errorf call, and that this call is
    reachable once per difftoolDiff invocation (i.e. once per recursion level, were
    recursion to occur).

- timestamp: 2026-08-05T00:12:00Z
  checked: |
    Empirical experiment A — registered a non-alturd "spytool" as
    difftool.spytool.cmd (dumps env to a log file, exits 0, no recursion) in a
    fresh scratch repo at /tmp/.../difftool-repro with only the 4 canonical
    install-difftool config keys. Ran `git difftool -t spytool README.md`.
  found: |
    The difftool.cmd child process's environment contained
    GIT_EXTERNAL_DIFF=git-difftool--helper, plus GIT_DIFF_TOOL=spytool,
    GIT_DIFFTOOL_TRUST_EXIT_CODE=true, GIT_DIFF_PATH_COUNTER=1,
    GIT_DIFF_PATH_TOTAL=1, GIT_PREFIX=., GIT_WORK_TREE=., GIT_DIR=<repo>/.git.
  implication: |
    git-core's own difftool builtin unconditionally sets GIT_EXTERNAL_DIFF in the
    environment of the process it invokes as difftool.<name>.cmd, for every
    difftool invocation, regardless of any prior gitconfig. This env var is
    inherited by any subprocess that difftool.cmd process itself spawns (Go's
    exec.Command inherits the parent's full environment unless Env is explicitly
    set) — including alturd's own internal `git diff --no-index` call.

- timestamp: 2026-08-05T00:14:00Z
  checked: |
    Empirical experiment B — ran `GIT_EXTERNAL_DIFF=<spy2.sh> git diff --no-index
    -- a.txt b.txt` directly (simulating the inherited env from experiment A)
    against a plain pair of files (no git repo needed for --no-index).
  found: |
    spy2.sh (a script with no relation to git-difftool--helper) WAS invoked by
    git diff --no-index, receiving the standard external-diff 7-argument calling
    convention (path old-file old-hex old-mode new-file new-hex new-mode). git
    diff --no-index produced NO diff output itself — it fully delegated to the
    external program instead of doing its own internal diff.
  implication: |
    git diff --no-index honors GIT_EXTERNAL_DIFF exactly like a normal git diff
    invocation would. Combined with experiment A, this proves: alturd's internal
    `git diff --no-index -- local remote` call, when run as part of the
    git-difftool subprocess tree, does NOT compute its own diff — it re-invokes
    git-difftool--helper (the real program GIT_EXTERNAL_DIFF points to in
    production, confirmed by experiment A), which re-dispatches to
    difftool.alturd.cmd = alturd again with fresh $LOCAL/$REMOTE/$MERGED temp
    files, which calls difftoolDiff again, which again inherits GIT_EXTERNAL_DIFF
    and recurses. This is unbounded: each level spawns a new alturd process and a
    new inner git diff --no-index process, until fork()/exec() starts failing
    ("fork/exec /usr/lib/git-core/git: resource temporarily unavailable" — matches
    reported error verbatim). git-difftool--helper's own invocation of a program
    that then dies/errors is exactly what produces git-core's own
    "fatal: external diff died, stopping at PATH" message per recursion level
    (also matches verbatim). The many concurrently-alive doomed alturd processes
    each independently print their own "git diff --no-index: <wrapped err>" line
    via main.go's reportError, producing the flood of repeated
    "git diff --no-index: git diff --no-index: ..." text the user observed.

- timestamp: 2026-08-05T00:16:00Z
  checked: |
    Empirical experiment C — repeated experiment B with --no-ext-diff added:
    `GIT_EXTERNAL_DIFF=<spy2.sh> git diff --no-index --no-ext-diff -- a.txt b.txt`.
  found: |
    spy2.sh was NEVER invoked (log file empty). git diff --no-index produced the
    correct built-in unified diff output and exited 1 (files differ), matching
    difftoolDiff's expected exit-code contract exactly.
  implication: |
    --no-ext-diff on the internal git diff --no-index invocation fully and
    correctly suppresses the external-diff dispatch regardless of inherited
    GIT_EXTERNAL_DIFF/diff.external, without changing the exit-code contract
    difftoolDiff already relies on (0=identical, 1=differs). This is a minimal,
    root-cause-targeted fix.

- timestamp: 2026-08-05T00:17:00Z
  checked: |
    grep -rn "exec.Command" across the whole repo (excluding _test.go) to confirm
    which call sites are actually reachable from inside a git-difftool subprocess
    tree.
  found: |
    Three exec.Command call sites total: (1) difftool.go:146 gitConfigRun — used
    only by install-difftool, run standalone, never as a difftool.cmd child;
    (2) main.go:272 difftoolDiff — the ONLY one invoked as part of the
    git-difftool subprocess tree (reached via --difftool-local/-remote/-path
    flags, which are only ever set by git itself when alturd runs as the
    registered difftool.cmd); (3) internal/git/runner.go:45 ExecRunner.Run — only
    used by the standalone (non-difftool) `alturd [ref]` code path in main.go's
    run(), never invoked from within a git-difftool subprocess tree.
  implication: |
    difftoolDiff's single exec.Command call at main.go:272 is the sole vulnerable
    site — no other git-spawning code in the repo is reachable from inside the
    GIT_EXTERNAL_DIFF-poisoned environment, so the fix is fully localized.

## Resolution

root_cause: |
  cmd/alturd/main.go's difftoolDiff() (called from loadDifftoolFiles, reached
  whenever alturd runs in difftool mode via --difftool-local/-remote/-path)
  invokes `exec.Command("git", "diff", "--no-index", "--", local, remote)`
  without the --no-ext-diff flag and without overriding Env. When git's own
  `git difftool` builtin invokes alturd as the configured difftool.<name>.cmd,
  it unconditionally sets GIT_EXTERNAL_DIFF=git-difftool--helper in that child
  process's environment (confirmed empirically, present with zero other
  difftool-related config beyond what install-difftool itself writes — not a
  pre-existing user/environment artifact). Because difftoolDiff's git
  subprocess inherits the full parent environment, its internal
  `git diff --no-index` call also honors GIT_EXTERNAL_DIFF (confirmed
  empirically: git diff --no-index treats GIT_EXTERNAL_DIFF exactly like a
  normal git diff invocation does) — so instead of computing its own diff, it
  re-invokes git-difftool--helper, which re-dispatches to difftool.alturd.cmd
  = alturd again with fresh $LOCAL/$REMOTE/$MERGED temp files. That new alturd
  process calls difftoolDiff again, which again inherits GIT_EXTERNAL_DIFF and
  recurses. This is unbounded recursive process spawning (alturd -> git diff
  --no-index -> git-difftool--helper -> alturd -> ...) that continues until
  fork()/exec() starts failing under resource pressure, producing exactly the
  reported symptoms: the "fork/exec ... resource temporarily unavailable"
  error, git-core's own "fatal: external diff died, stopping at PATH" message
  per failed recursion level, and the flood of repeated
  "git diff --no-index: ..." lines (each live doomed alturd process
  independently printing its own difftoolDiff error via main.go's
  reportError).
fix: (not applied — goal is find_root_cause_only)
verification: |
  Root cause confirmed via three decomposed, safe empirical experiments (A, B,
  C above) rather than the actual catastrophic full reproduction (which was
  deliberately avoided to prevent exhausting process/fork resources on the
  shared sandbox). Fix direction (--no-ext-diff) independently verified to
  suppress the exact mechanism in experiment C while preserving difftoolDiff's
  expected exit-code contract.
files_changed: []
