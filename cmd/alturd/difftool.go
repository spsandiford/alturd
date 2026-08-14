// difftool.go implements the `install-difftool` subcommand: it registers
// alturd as a git difftool backend by writing four canonical gitconfig keys
// (DIFFTOOL-03, D-08).
//
// G-04-1 correction (see .planning/debug/DEBUG-difftool-trustexitcode-fatal.md):
// the fourth key, difftool.trustExitCode, is written as false, not true.
// git's external-diff protocol has exactly two outcomes for the invoked
// command's exit status — zero means success, and every non-zero value is an
// unconditional fatal crash. It cannot represent "the user intentionally
// cancelled", so a trusted non-zero exit — which is exactly what alturd
// produces on its documented abort convention — can never produce the clean
// stop D-08 originally assumed it would; it instead makes git's diff engine
// die() with "external diff died, stopping at <file>". Accepted trade-off:
// in a multi-file `git difftool` session with no pathspec, aborting on file N
// no longer prevents file N+1 from opening — git walks every remaining
// changed file.
//
// Its git invocations deliberately do NOT go through internal/git.ExecRunner.
// Per 04-RESEARCH.md Pitfall C, ExecRunner's error interpretation is tuned to
// `git diff`'s exit-code conventions (it treats exit 128 as always-fatal and
// hardcodes a diff-specific message prefix) and would misreport a
// `git config --get` lookup whose key is simply absent — that call legitimately
// exits 1 for "key not found", which is a normal outcome here, not a failure.
// gitConfigRun below interprets `git config`'s own exit-code contract instead.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alturd/alturd/internal/git"
)

// toolNamePattern anchors --name to a safe character set. This bound is
// security-relevant, not cosmetic: name is interpolated into the gitconfig
// key difftool.<name>.cmd, so permitting whitespace, quotes, brackets or
// newlines would let a crafted name inject an unrelated key or section into
// the user's gitconfig (T-04-04-01).
var toolNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// difftoolCmdTemplate is the published gitconfig `cmd` value written by
// install-difftool. Git evaluates this string in a shell on every platform —
// on Windows that shell is the POSIX sh bundled with Git for Windows, not
// cmd.exe — so POSIX double-quoting is correct and portable across all three
// targets (04-RESEARCH.md Pitfall E). $LOCAL/$REMOTE/$MERGED are substituted
// by git itself, never by alturd, so nothing user-supplied is interpolated
// into this value. $MERGED maps to --difftool-path because it is the real
// working-tree filename, while $LOCAL and $REMOTE are git's pre-image and
// post-image temp files (matches the --difftool-* flags registered in
// cmd/alturd/main.go by 04-03).
const difftoolCmdTemplate = `alturd --difftool-local "$LOCAL" --difftool-remote "$REMOTE" --difftool-path "$MERGED"`

// installDifftoolCmd registers alturd as a git difftool backend (DIFFTOOL-03).
var installDifftoolCmd = &cobra.Command{
	Use:   "install-difftool",
	Short: "Register alturd as a git difftool",
	RunE:  runInstallDifftool,
}

// init registers --scope/--name/--force (D-09 defaults: scope=global,
// name=alturd, force=false) and attaches the subcommand to the existing
// rootCmd, inheriting SilenceErrors/SilenceUsage so main() remains the
// single place errors are printed.
func init() {
	installDifftoolCmd.Flags().String("scope", "global", "config scope: global|local")
	installDifftoolCmd.Flags().String("name", "alturd", "difftool name to register")
	installDifftoolCmd.Flags().Bool("force", false, "overwrite an existing diff.tool setting")
	rootCmd.AddCommand(installDifftoolCmd)
}

// runInstallDifftool implements the D-08/D-09/D-10 install-difftool
// contract. --scope and --name are validated before any subprocess runs, so
// an invalid argument never causes a partial gitconfig write. The
// difftool.trustExitCode key is written as false (G-04-1 correction, see the
// D-08 comment below and .planning/debug/DEBUG-difftool-trustexitcode-fatal.md)
// rather than the superseded true.
func runInstallDifftool(cmd *cobra.Command, _ []string) error {
	scope, err := cmd.Flags().GetString("scope")
	if err != nil {
		return err
	}
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}

	scopeFlag, err := validateScope(scope)
	if err != nil {
		return err
	}
	if err := validateToolName(name); err != nil {
		return err
	}

	// D-10: read the existing diff.tool value before writing anything.
	existing, present, err := gitConfigGet(scopeFlag, "diff.tool")
	if err != nil {
		return err
	}

	// existing/name is an exact byte-string comparison — no case folding, no
	// Unicode normalisation — matching git's own case-sensitive subsection
	// semantics for the idempotency check (DIFFTOOL-03 encoding edge).
	overwritingForeignTool := present && existing != name
	if overwritingForeignTool && !force {
		return fmt.Errorf("diff.tool is already set to %q; pass --force to overwrite", existing)
	}

	// D-08: each of the four canonical keys is written through its own
	// gitConfigSet call, so git — not alturd — edits the file and preserves
	// unrelated keys, comments and ordering (T-04-04-03).
	//
	// difftool.trustExitCode is written as false (G-04-1 correction; was
	// true), and deliberately as an explicit value rather than left unset:
	// this key was verified empirically against installed git 2.39.5 to
	// resolve across the system/global/local precedence chain, so a scoped
	// `install-difftool --scope global` that merely removed the key would
	// still lose to a lower-precedence scope still holding the superseded
	// `true` — the fatal would come straight back. An explicit value at the
	// install scope overrides everything below it and converges any machine
	// that already ran the previous version of install-difftool, which is
	// what D-10's "the command converges the config" means.
	// git-difftool--helper only forwards the backend's exit status when the
	// resolved value is exactly the string "true", so writing "false"
	// restores git's own default masking behaviour and stops a trusted
	// non-zero exit (alturd's abort convention) from ever reaching diff.c's
	// unconditional die()-on-non-zero-exit path. See the file header comment
	// and .planning/debug/DEBUG-difftool-trustexitcode-fatal.md for the full
	// trace, and T-04-04-* / the Pitfall references already noted above for
	// the surrounding key-writing contract.
	if err := gitConfigSet(scopeFlag, "diff.tool", name); err != nil {
		return err
	}
	if err := gitConfigSet(scopeFlag, fmt.Sprintf("difftool.%s.cmd", name), difftoolCmdTemplate); err != nil {
		return err
	}
	if err := gitConfigSet(scopeFlag, "difftool.prompt", "false"); err != nil {
		return err
	}
	if err := gitConfigSet(scopeFlag, "difftool.trustExitCode", "false"); err != nil {
		return err
	}

	switch {
	case overwritingForeignTool:
		_, _ = fmt.Fprintf(os.Stdout, "Overwrote existing diff.tool %q with %q (scope: %s, --force).\n", existing, name, scope)
	case present:
		// present && existing == name: idempotent re-run — the four keys were
		// still re-written above (that is what "idempotent" means under D-10:
		// the command converges the config), but the copy signals no change.
		_, _ = fmt.Fprintf(os.Stdout, "alturd is already configured as git difftool %q (scope: %s).\n", name, scope)
	default:
		_, _ = fmt.Fprintf(os.Stdout, "Installed alturd as git difftool %q (scope: %s).\n", name, scope)
	}
	return nil
}

// gitConfigRun executes "git config <args...>" in argv form and returns
// stdout, git's own exit code, and an error only for conditions gitConfigRun
// itself recognizes as unconditionally exceptional (git not on PATH, or the
// --local-outside-a-repo case). Any other non-zero exit code is returned as
// exitCode with a nil error so the caller (gitConfigGet/gitConfigSet)
// decides what that code means per git config's own contract. Callers pass
// only the arguments that follow the "config" subcommand (e.g. scopeFlag,
// "--get", key) — gitConfigRun prepends "config" itself.
func gitConfigRun(args ...string) (stdout string, exitCode int, err error) {
	// SECURITY: exec.Command uses argv form — each element is a separate argument.
	// Shell metacharacters in gitconfig keys or values are never interpreted.
	fullArgs := append([]string{"config"}, args...)
	cmd := exec.Command("git", fullArgs...) //nolint:gosec
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	if runErr == nil {
		return outBuf.String(), 0, nil
	}

	var execErr *exec.Error
	if errors.As(runErr, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
		return "", 0, git.ErrGitNotFound
	}

	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		stderrStr := errBuf.String()
		// git's exact wording for "--local outside a repository" varies by
		// version (git 2.39.5 in this environment says "--local can only be
		// used inside a git repository"; other versions/docs describe "not a
		// git repository"). Both phrasings share "git repository", so match on
		// that substring rather than the narrower literal string from
		// 04-RESEARCH.md Pitfall D (deviation: Rule 1 auto-fix, verified
		// against the actual installed git binary).
		if strings.Contains(strings.ToLower(stderrStr), "git repository") {
			return "", exitErr.ExitCode(), git.ErrLocalScopeOutsideRepo
		}
		return outBuf.String(), exitErr.ExitCode(), nil
	}

	if msg := strings.TrimRight(errBuf.String(), "\n"); msg != "" {
		return "", 0, fmt.Errorf("git config: %s", msg)
	}
	return "", 0, fmt.Errorf("git config: %w", runErr)
}

// gitConfigGet reads a single gitconfig key via `git config <scopeFlag> --get
// <key>`, interpreting git config's own exit-code contract: 0=success,
// 1=key/section not found (a normal "not configured" outcome, never an
// error — 04-RESEARCH.md Pitfall C), 2=invalid config file, anything else is
// surfaced as an error.
func gitConfigGet(scopeFlag, key string) (value string, present bool, err error) {
	out, code, runErr := gitConfigRun(scopeFlag, "--get", key)
	if runErr != nil {
		return "", false, runErr
	}
	switch code {
	case 0:
		return strings.TrimRight(out, "\n"), true, nil
	case 1:
		return "", false, nil
	case 2:
		return "", false, fmt.Errorf("git config: invalid config file")
	default:
		return "", false, fmt.Errorf("git config --get %s: exit code %d", key, code)
	}
}

// gitConfigSet writes a single gitconfig key via `git config <scopeFlag>
// <key> <value>`, treating any non-zero exit as an error wrapped with the key
// name so a failure says which key it was writing.
func gitConfigSet(scopeFlag, key, value string) error {
	_, code, err := gitConfigRun(scopeFlag, key, value)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("git config --set %s: exit code %d", key, code)
	}
	return nil
}

// validateScope maps the --scope flag value to git's own --global/--local
// flag, rejecting anything else (including "system", which install-difftool
// deliberately never targets) with a single-line error.
func validateScope(scope string) (string, error) {
	switch scope {
	case "global":
		return "--global", nil
	case "local":
		return "--local", nil
	default:
		return "", errors.New(`--scope must be "global" or "local"`)
	}
}

// validateToolName rejects an empty name and anchors non-empty names to
// toolNamePattern. name is compared and stored as an exact byte string —
// never case-folded, never Unicode-normalised — because git treats the
// subsection component of a config key as case-sensitive, so any folding
// here would make the idempotency check in runInstallDifftool disagree with
// what git actually stored.
func validateToolName(name string) error {
	if name == "" {
		return errors.New("--name must not be empty")
	}
	if !toolNamePattern.MatchString(name) {
		return errors.New("--name must be 1-64 characters of letters, digits, dot, dash or underscore")
	}
	return nil
}
