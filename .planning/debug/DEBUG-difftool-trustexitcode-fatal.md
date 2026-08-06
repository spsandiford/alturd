---
status: diagnosed
trigger: "UAT gap G-04-1 (04-UAT.md): pressing 'Q' in the alturd difftool TUI should return the shell prompt cleanly with exit code 1 and no git fatal error, but git prints 'fatal: external diff died, stopping at <file>' and exits 128."
created: 2026-08-06T00:00:00Z
updated: 2026-08-06T00:00:00Z
---

## Current Focus

hypothesis: CONFIRMED — see Resolution.root_cause
test: traced git 2.39.5's actual C source (builtin/difftool.c) and shell
  helper (git-difftool--helper, git-mergetool--lib) installed at
  /usr/lib/git-core/ against the reporter's evidence chain
expecting: confirm/refute whether difftool.trustExitCode=true is what turns
  alturd's intentional exit-1-on-Q into git's fatal "external diff died"
next_action: none — goal is find_root_cause_only, return ROOT CAUSE FOUND

## Symptoms

expected: |
  In a real interactive terminal, after `alturd install-difftool`, running
  `git difftool -t alturd <file>` on a repo with an uncommitted change should
  show the alturd TUI, and pressing 'Q' should quietly return to the shell
  with exit code 1 — no git fatal error.
actual: |
  Pressing 'Q' causes git to print `fatal: external diff died, stopping at
  <file>` and the difftool session exits 128, not the expected clean 1.
  alturd's own exit code was independently confirmed clean (1) via a logging
  wrapper around difftool.alturd.cmd, both invoked directly and through the
  full git-difftool--helper chain.
errors: |
  `fatal: external diff died, stopping at <file>` (printed by git); git
  difftool overall process exits 128; alturd's own process exits 1 (confirmed
  clean, not the bug).
reproduction: |
  `alturd install-difftool --scope local` in a repo; make an uncommitted
  change to a tracked file; `git difftool -t alturd -- <file>` in a real
  terminal; press Q; `echo $?` reports 128 instead of 1, with the fatal line
  printed. Reproduced by reporter via scripted pty session against a real
  `git difftool -t alturd -- <file>` invocation (both --no-index long-filename
  case and a genuine tracked-file change).
started: |
  Discovered during UAT re-test of phase 04 (config-theming-difftool-distribution),
  re-testing after the prior G-04-2 (difftool recursion/flood) fix.

## Eliminated

- hypothesis: alturd's own exit-code handling on quit is wrong (e.g. exits
    with the wrong code, or crashes)
  evidence: Reporter's own wrapper-script test (`alturd "$@"; echo "rc=$?" >>
    log`) confirmed alturd exits cleanly with code 1 on 'Q', both invoked
    directly with --difftool-local/-remote/-path flags and through the full
    git-difftool--helper chain. Source inspection of cmd/alturd/main.go
    (errAborted = &git.ExitCodeError{Code: 1}, reportError()) and
    internal/tui/model.go (WasAborted/Aborted()) confirms this is the
    documented, intentional convention (CR-02, D-08, NAV-04) — not a bug.
  timestamp: 2026-08-06T00:00:00Z

## Evidence

- timestamp: 2026-08-06T00:00:00Z
  checked: cmd/alturd/difftool.go (runInstallDifftool, lines 104-118)
  found: |
    install-difftool writes exactly four gitconfig keys, each via its own
    gitConfigSet call: diff.tool=<name>, difftool.<name>.cmd=<template>,
    difftool.prompt=false, difftool.trustExitCode=true (line 116). Comment
    block at top of file references 04-RESEARCH.md but does not itself
    re-derive the trustExitCode rationale.
  implication: |
    Confirms exactly what .git/config shows after install — trustExitCode is
    written unconditionally, with no scope/mode exception (e.g. not
    conditioned on whether alturd's own quit-key exit-code convention is
    compatible with it).

- timestamp: 2026-08-06T00:00:00Z
  checked: cmd/alturd/main.go (errAborted sentinel, reportError, WasAborted usage)
  found: |
    `errAborted = &git.ExitCodeError{Code: 1}` (line 211) with an explicit
    comment: "difftool.trustExitCode = true (D-08) reads only the process
    exit status, never any output." internal/tui/model.go's WasAborted doc
    comment (lines 99-104) says the same: "A true result means the caller
    must exit with status 1 (code review CR-02, D-08: difftool.trustExitCode
    = true reads only the process exit status)".
  implication: |
    The exit-1-on-abort convention and difftool.trustExitCode=true were
    co-designed together, on purpose, specifically so that pressing the
    default abort key ('Q', config.ActionAbort) would signal git to stop the
    difftool loop. The design intent is real and documented, not accidental
    — this rules out "alturd's exit code is a mistake" as a fix direction and
    frames the bug as a mismatch between that intent and git's actual
    trustExitCode semantics.

- timestamp: 2026-08-06T00:00:00Z
  checked: internal/config/keybindings.go
  found: |
    ActionQuit default key "q" (lowercase) vs ActionAbort default key "Q"
    (uppercase) are distinct actions (lines 119-120). Only ActionAbort sets
    model.aborted = true (internal/tui/model.go line 582), which is what
    drives the exit-1 path. A normal quit ("q") does not set aborted and
    exits 0 through the ordinary nil-error path in cmd/alturd/main.go's run().
  implication: |
    The bug is specific to the abort/'Q' path (exit 1), matching the UAT
    report exactly. A plain 'q' quit would exit 0 and NOT trigger git's
    fatal-die branch (0 is never treated as an external-diff failure) — so
    this bug's blast radius is precisely "any git difftool invocation of
    alturd where the user presses the abort key."

- timestamp: 2026-08-06T00:00:00Z
  checked: .planning/phases/04-config-theming-difftool-distribution/04-CONTEXT.md
    (D-08), 04-RESEARCH.md ("Difftool Setup" section, line 14 and line 31),
    04-UI-SPEC.md (line 181)
  found: |
    D-08 (04-CONTEXT.md line 37) explicitly states the intent: "difftool.
    trustExitCode = true (so Q -> exit 1 correctly signals git to abort the
    difftool loop, consistent with NAV-04's existing Q -> exit 1 behavior
    from Phase 3)." 04-RESEARCH.md's summary (line 14) claims this 4-key set
    was "confirmed correct and complete against both git-scm.com's official
    docs and git's own C source (diff.c, builtin/difftool.c, git-difftool--
    helper.sh)."
  implication: |
    The design intent for trustExitCode=true was to make 'Q' abort the
    difftool loop cleanly. RESEARCH.md's own claim to have verified this
    against git's C source is the crux to check next — if that verification
    was accurate, git's difftool loop-abort mechanism must somehow avoid the
    fatal die() path; if it was incomplete, that's the root-cause mismatch.

- timestamp: 2026-08-06T00:00:00Z
  checked: /usr/lib/git-core/git-difftool--helper (installed git 2.39.5,
    matches the version the reporter tested against), the shell script
    invoked as GIT_EXTERNAL_DIFF by git's core diff engine in difftool mode
  found: |
    The per-file loop (`while test $# -gt 6; do launch_merge_tool ...;
    status=$?; if test $status -ge 126 then exit $status fi; if test
    "$status" != 0 && test "$GIT_DIFFTOOL_TRUST_EXIT_CODE" = true then exit
    $status fi; shift 7; done; exit 0`) shows: launch_merge_tool's exit
    status (ultimately alturd's own exit status, 1, threaded through
    run_merge_tool -> run_diff_cmd -> diff_cmd) is only propagated as
    git-difftool--helper's OWN process exit status when
    GIT_DIFFTOOL_TRUST_EXIT_CODE=true. Without it, the script always falls
    through to `exit 0` at the end regardless of the tool's exit code
    (masking it entirely).
  implication: |
    Confirms exactly the mechanism the reporter hypothesized: trustExitCode
    is what makes alturd's exit 1 reach git-difftool--helper's own exit
    status, rather than being swallowed.

- timestamp: 2026-08-06T00:00:00Z
  checked: /usr/lib/git-core/git-mergetool--lib (run_merge_tool, run_diff_cmd,
    run_merge_cmd, trust_exit_code functions)
  found: |
    `run_merge_tool` branches on `merge_mode`: merge mode calls
    `run_merge_cmd`, which DOES consult a *different*, per-tool config key
    (`mergetool.<tool>.trustExitCode`, via the `trust_exit_code()` function)
    to decide whether to call `check_unchanged` as a fallback. Diff mode
    instead calls `run_diff_cmd`, which is simply `diff_cmd "$1"` — i.e. it
    invokes the tool directly and returns its exit status with NO
    trust_exit_code consultation at all in diff mode. `difftool.trustExitCode`
    (the key alturd writes) is a distinct, global, difftool-specific
    boolean — it never flows through this file at all; it only reaches the
    process as the GIT_DIFFTOOL_TRUST_EXIT_CODE env var (see next entry).
  implication: |
    Directly confirms reporter's evidence item 3: `mergetool.<tool>.
    trustExitCode` (the merge-mode key this library implements) is unrelated
    machinery from `difftool.trustExitCode` (the global diff-mode key alturd
    writes). Diff mode's own exit-status handling is controlled entirely by
    the outer git-difftool--helper script's GIT_DIFFTOOL_TRUST_EXIT_CODE
    check, not by anything in git-mergetool--lib.

- timestamp: 2026-08-06T00:00:00Z
  checked: /tmp/git-difftool.c (builtin/difftool.c source for the installed
    git 2.39.5, obtained for direct source verification), specifically
    difftool_config(), cmd_difftool(), run_file_diff()
  found: |
    `difftool_config()` (lines 49-63) parses the `difftool.trustexitcode`
    gitconfig key into `dt_options.trust_exit_code` (a plain bool, default 0
    — line 727). `cmd_difftool()` (line 804-805) does:
    `setenv("GIT_DIFFTOOL_TRUST_EXIT_CODE", dt_options.trust_exit_code ?
    "true" : "false", 1);` — this is the ONLY place that env var is set, and
    it is set unconditionally on the child process environment. `run_file_diff()`
    (lines 701-715) sets `GIT_EXTERNAL_DIFF=git-difftool--helper` and
    `GIT_PAGER=` on the child, then does `return run_command(child)` where
    child.args = ["diff", ..., "--", <file>]. child inherits the
    process env (including GIT_DIFFTOOL_TRUST_EXIT_CODE), so it flows
    straight through to the `git diff` subprocess and from there to whatever
    diff.c invokes as GIT_EXTERNAL_DIFF (git-difftool--helper) per changed
    file. cmd_difftool's own return value IS run_file_diff's return value IS
    run_command's return value on the `git diff` child — i.e. `git
    difftool`'s own exit status is exactly the `git diff` subprocess's exit
    status, nothing more is layered on top.
  implication: |
    Fully closes the causal chain end-to-end: difftool.trustExitCode=true
    (gitconfig, written by alturd install-difftool) -> GIT_DIFFTOOL_TRUST_EXIT_CODE=true
    (env var set unconditionally by cmd_difftool) -> inherited by the `git
    diff` subprocess -> inherited by git-difftool--helper (invoked per file
    as GIT_EXTERNAL_DIFF by diff.c's diff engine) -> git-difftool--helper
    propagates alturd's exit 1 as its OWN exit status (per the earlier
    finding) -> diff.c's run_external_diff()/finish_command() sees the
    external-diff command (git-difftool--helper) exit nonzero -> calls
    die(_("external diff died, stopping at %s"), name) -> the `git diff`
    subprocess itself terminates via die() (git's fatal-error path always
    exits 128 and prints exactly the observed line) -> that subprocess's
    exit code (128) is `git difftool`'s own final exit status, matching the
    UAT report exactly (128, not 1).

- timestamp: 2026-08-06T00:00:00Z
  checked: git's `die()` fatal-error semantics as exercised via
    run_external_diff()/finish_command() in diff.c (behavior confirmed
    through the observed 128 exit code + verbatim "external diff died"
    message, which is git's well-known, version-stable fatal-error idiom for
    any nonzero exit from a GIT_EXTERNAL_DIFF command — not something
    04-RESEARCH.md's own available source excerpt covered, since
    /tmp/git-difftool.c only contains builtin/difftool.c, not diff.c itself)
  found: |
    Git's external-diff protocol has exactly two outcomes for the invoked
    command's exit status: 0 = success (whatever the tool printed, if
    anything, is the diff), nonzero = fatal crash, unconditionally, for ANY
    nonzero value. There is no reserved nonzero value or side-channel that
    means "tool exited cleanly but wants to signal something to git" — this
    is true whether or not difftool.trustExitCode is set; trustExitCode only
    controls whether git-difftool--helper *forwards* the wrapped tool's exit
    code into this all-or-nothing signal at all, not how git's diff engine
    interprets a nonzero code once it arrives.
  implication: |
    This is the actual root-cause mismatch: D-08's design intent ("Q -> exit
    1 correctly signals git to abort the difftool loop") assumed
    trustExitCode=true lets git distinguish an intentional abort from a
    crash. It cannot — git's diff engine cannot represent that distinction at
    all; ANY nonzero exit is always fatal. So trustExitCode=true does achieve
    "stop processing" (die() halts the whole `git diff` subprocess
    immediately, which for a single-file `git difftool -t alturd -- <file>`
    invocation happens to look like "it stopped"), but only via an ugly crash
    path (fatal message + exit 128), never via the clean stop the UAT truth
    describes. 04-RESEARCH.md's "confirmed correct and complete" verdict (line
    14) verified the config keys existed and had the right names/values per
    git's docs and C source, but did not trace the further consequence of
    diff.c's unconditional die()-on-nonzero-exit behavior for the specific
    case of an *intentional* nonzero exit — that gap in the research is why
    the mismatch shipped.

- timestamp: 2026-08-06T00:00:00Z
  checked: whether removing/flipping trustExitCode loses real behavior (the
    tradeoff the debug_context asked to characterize) — reasoned from the
    git-difftool--helper script and cmd_difftool source already read
  found: |
    Without difftool.trustExitCode=true (i.e. default false, or the key
    simply absent), git-difftool--helper's per-file loop always falls
    through to `exit 0` regardless of alturd's exit code (masking it), so
    `git diff`'s external-diff invocation always reports success to diff.c
    and iteration continues normally through all changed files. Concretely:
    for `git difftool -t alturd -- <file>` on a SINGLE file, the practical
    effect of turning trustExitCode off is simply "no more fatal die() -
    clean exit 0" (there is nothing left to iterate). For a MULTI-file `git
    difftool -t alturd` invocation (no path pinned to one file), turning
    trustExitCode off means pressing the abort key on file N would no longer
    stop git from proceeding to open file N+1 — the abort signal would be
    silently swallowed for iteration-control purposes (though alturd would
    still exit 1 on its own side, that fact just never reaches git). The
    >=126 branch (command not found / not executable / killed by signal) is
    untouched by trustExitCode either way — genuine crashes still always
    propagate and still always die() fatally, which is arguably correct
    behavior to keep.
  implication: |
    trustExitCode=true was never able to deliver a "quiet future" for
    multi-file abort either — even the current true setting only stops
    remaining-file iteration via the identical ugly fatal/128 path (git's
    engine cannot tell clean-cancel from crash). So keeping it true buys
    "abort really does stop the remaining-file loop" at the cost of the fatal
    message + wrong exit code on every single quit; setting it false buys
    "clean exit, no fatal message" for the single-file case but silently
    disables abort-driven early-stop for the multi-file case (Q on file 1
    would not prevent file 2 from opening). Neither setting alone recovers
    both the UAT's literal expectation (exit 1, no fatal line) and the
    original multi-file abort-loop-stopping intent behind D-08, because git's
    diff engine fundamentally cannot honor "stop, but don't error" from a
    GIT_EXTERNAL_DIFF-style command — this constrains what a subsequent fix
    plan can choose between.

## Resolution

root_cause: |
  cmd/alturd/difftool.go:116 (runInstallDifftool) unconditionally writes
  `difftool.trustExitCode = true` to gitconfig. Combined with alturd's
  intentional exit-code-1-on-abort convention (cmd/alturd/main.go:211
  errAborted = &git.ExitCodeError{Code: 1}; internal/tui/model.go:92/582
  Aborted()/WasAborted(), triggered by the default 'Q' keybinding,
  config.ActionAbort), this causes git's core diff engine to fatally
  terminate on every abort. Mechanism, traced end-to-end against installed
  git 2.39.5 source (builtin/difftool.c, git-difftool--helper,
  git-mergetool--lib): difftool.trustExitCode=true -> cmd_difftool() sets
  GIT_DIFFTOOL_TRUST_EXIT_CODE=true unconditionally on the `git diff`
  subprocess environment (builtin/difftool.c line ~804) -> that env var is
  inherited by git-difftool--helper, which diff.c invokes as
  GIT_EXTERNAL_DIFF once per changed file -> git-difftool--helper's per-file
  loop propagates alturd's own exit code (1) as its OWN process exit status
  only because GIT_DIFFTOOL_TRUST_EXIT_CODE=true (without it, the script
  always exits 0, masking the tool's exit code) -> diff.c's
  run_external_diff()/finish_command() treats ANY nonzero exit from the
  GIT_EXTERNAL_DIFF command as an unconditional fatal crash (there is no
  git-side distinction between "tool exited nonzero because it crashed" and
  "tool exited nonzero because the user intentionally cancelled") -> git
  calls die(_("external diff died, stopping at %s"), name), which prints the
  observed fatal line and terminates the `git diff` subprocess with exit
  code 128 -> that becomes git difftool's own final exit status (verified in
  builtin/difftool.c: cmd_difftool()'s return value is exactly
  run_file_diff()'s run_command() result, nothing further is layered on
  top). This reproduces the reported 128 + fatal line exactly, and confirms
  alturd's own exit code (1) was never itself wrong — the reporter's own
  elimination of that hypothesis (via the logging wrapper) is corroborated
  by source: the bug is entirely in the D-08 gitconfig contract that
  install-difftool writes, not in the TUI's quit-key exit-code handling.

  This is a git-config-mismatch bug (single root cause, not a multi-cause
  AND-gate case): the code category is "config" (the trustExitCode=true
  value written by install-difftool), not "code" — alturd's own exit-code
  logic on the code side is correct and intentional per its own documented
  contract; the failure only manifests because that correct code-side
  behavior is paired with a config value whose actual git-side semantics
  (traced through git's own source) do not deliver what D-08 assumed they
  would.

fix: "" # not applied — goal is find_root_cause_only; a later
  /gsd-plan-phase --gaps step designs the fix.
verification: "" # not applicable in diagnose-only mode
files_changed: []
