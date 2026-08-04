// Package main is the entrypoint for the alturd binary.
// It wires the cobra root command to the internal/git runner, diff parser,
// and bubbletea TUI model, launching an interactive side-by-side diff viewer.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	tea "charm.land/bubbletea/v2"
	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"github.com/alturd/alturd/internal/config"
	"github.com/alturd/alturd/internal/diff"
	"github.com/alturd/alturd/internal/git"
	applog "github.com/alturd/alturd/internal/log"
	"github.com/alturd/alturd/internal/tui"
)

// version is overridden at build time by goreleaser via -ldflags "-X main.version=<tag>".
// Declared as var (not const) so the linker can set it (D-03).
var version = "dev"

// rootCmd is the single cobra command — alturd has no subcommands in Phase 2.
// Subcommands (install-difftool) arrive in Phase 4 (D-02).
var rootCmd = &cobra.Command{
	Use:   "alturd [ref] [ref1..ref2] [-- paths]",
	Short: "Side-by-side terminal diff viewer",
	// Version enables --version flag; the value is set from the version var above.
	Version: version,
	// SilenceErrors/SilenceUsage required so main() fully controls all error output
	// (no second "Error: ..." line from cobra, no usage dump on flag errors — D-11).
	SilenceErrors: true,
	SilenceUsage:  true,
	Args:          cobra.ArbitraryArgs,
	RunE:          run,
}

// init registers rootCmd's persistent flags. Flag registration only wires
// cobra's flag parser; it does not read the filesystem, so --help/--version
// remain side-effect-free (D-03 is enforced later, inside run(), by
// config.Load itself never being called from init or PersistentPreRunE).
//
// The three --difftool-* flags have no short forms: each is either rarely
// typed by hand or invoked by git itself, so a single-letter alias would
// only burn namespace.
func init() {
	rootCmd.PersistentFlags().String("config", "", "path to a TOML config file (default: $XDG_CONFIG_HOME/alturd/config.toml)")
	rootCmd.PersistentFlags().String("theme", "", "theme override: light|dark|auto")
	rootCmd.PersistentFlags().String("difftool-local", "", "difftool mode: path to the local (old) file (git's $LOCAL)")
	rootCmd.PersistentFlags().String("difftool-remote", "", "difftool mode: path to the remote (new) file (git's $REMOTE)")
	rootCmd.PersistentFlags().String("difftool-path", "", "difftool mode: the real working-tree filename (git's $MERGED)")
}

// run is the cobra RunE handler. The FIRST statement must be applog.Init() so
// that --version and --help (which cobra handles before RunE is called) never
// create a log file or any side effects (D-10, T-02-08, GIT-04).
func run(cmd *cobra.Command, args []string) error {
	// FIRST: initialise log. Non-fatal — continue without logging if init fails.
	logFile, err := applog.Init()
	if err == nil {
		defer func() { _ = logFile.Close() }()
	}

	// Load configuration (keybinding overrides, theme). configFlag is empty
	// unless --config was passed, in which case config.Load reads exactly
	// that path and never silently falls back to defaults on a missing file
	// (CONFIG-01). Called here, inside RunE, never at package init or
	// PersistentPreRunE, so --help/--version never touch the filesystem (D-03).
	configFlag, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}
	cfg, err := config.Load(configFlag)
	if err != nil {
		return err
	}

	// Resolve the --theme flag and the three --difftool-* flags immediately
	// after config.Load (04-03-PLAN.md Task 2).
	themeFlag, err := cmd.Flags().GetString("theme")
	if err != nil {
		return err
	}
	flagTheme, err := config.ParseTheme(themeFlag)
	if err != nil {
		return err
	}

	difftoolLocal, err := cmd.Flags().GetString("difftool-local")
	if err != nil {
		return err
	}
	difftoolRemote, err := cmd.Flags().GetString("difftool-remote")
	if err != nil {
		return err
	}
	difftoolPath, err := cmd.Flags().GetString("difftool-path")
	if err != nil {
		return err
	}

	// Difftool mode is active when any of the three --difftool-* flags is
	// non-empty; all three are then required (DIFFTOOL-01).
	difftoolMode := difftoolLocal != "" || difftoolRemote != "" || difftoolPath != ""
	if difftoolMode && (difftoolLocal == "" || difftoolRemote == "" || difftoolPath == "") {
		return errors.New("--difftool-local, --difftool-remote and --difftool-path must all be provided")
	}

	// Resolve the terminal background before bubbletea takes over the TTY,
	// through the D-05/D-06/D-07 precedence chain. The detector closure is
	// constructed but only invoked when ResolveDarkBackground actually needs
	// it — in difftool mode it is never called, which is what D-06 requires
	// (the OSC 11 query is skipped, not merely time-boxed). Detection is now
	// owned entirely by internal/config; main.go no longer imports termenv
	// directly.
	darkBg := config.ResolveDarkBackground(flagTheme, cfg.Theme, difftoolMode, func() bool {
		return config.DetectDarkBackground(os.Stdout)
	})
	diff.SetDarkBackground(darkBg)

	var files []*gitdiff.File
	var dt tui.DifftoolInfo

	if difftoolMode {
		files, dt, err = loadDifftoolFiles(difftoolLocal, difftoolRemote, difftoolPath)
		if err != nil {
			return err
		}
		if files == nil {
			// Empty-state guard already printed by loadDifftoolFiles.
			return nil
		}
	} else {
		// Build the git diff argument slice: literal "diff" subcommand
		// prepended to the parsed ref/path arguments (ExecRunner.Run does not
		// add "diff" itself).
		gitArgs := append([]string{"diff"}, git.ParseRefArgs(args, cmd.ArgsLenAtDash())...)

		// Run git and obtain an io.Reader over the raw diff output.
		reader, runErr := git.ExecRunner{}.Run(gitArgs)
		if runErr != nil {
			// Return unchanged — ExitCodeError sentinels are already typed
			// correctly for main() to extract the exit code via errors.As.
			return runErr
		}

		// Parse the unified diff into typed File structs.
		files, err = diff.Parse(reader)
		if err != nil {
			return fmt.Errorf("parsing diff output: %w", err)
		}

		// Empty-state guard: if there are no changed files, inform the user
		// and exit cleanly without starting the TUI (UI-SPEC Empty State).
		if len(files) == 0 {
			fmt.Fprintln(os.Stderr, "No changes found.")
			return nil
		}
	}

	// Phase 3: launch bubbletea TUI (D-06).
	// Data is pre-loaded; no async loading inside the model.
	// In bubbletea v2, alternate screen is declared in the model's View()
	// via view.AltScreen=true rather than as a program option (v2 upgrade guide).
	m := tui.NewModel(files, darkBg, cfg.Keys, dt)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	// CR-02 (code review): tea.Program.Run has already restored the
	// terminal (raw mode, alternate screen) by the time it returns, since
	// internal/tui now routes the abort key through tea.Quit instead of
	// calling os.Exit directly. Returning errAborted through run() (rather
	// than exiting here) lets the deferred logFile.Close() above still run.
	if tui.WasAborted(finalModel) {
		return errAborted
	}
	return nil
}

// errAborted is the sentinel run() returns when the user pressed the abort
// key. Its Msg is deliberately empty: the abort path has always printed
// nothing to stderr, and difftool.trustExitCode = true (D-08) reads only the
// process exit status, never any output. reportError's empty-message
// suppression is what keeps this silent (code review CR-02).
var errAborted = &git.ExitCodeError{Code: 1}

// loadDifftoolFiles builds the single-file gitdiff.File and tui.DifftoolInfo
// for difftool mode (DIFFTOOL-01, DIFFTOOL-02). local and remote are git's
// $LOCAL/$REMOTE temp files; path is git's $MERGED, the real working-tree
// filename. A nil files return (with nil error) means the empty-state
// message was already printed and the caller should exit 0 without starting
// the TUI, mirroring the standalone empty-state guard.
func loadDifftoolFiles(local, remote, path string) ([]*gitdiff.File, tui.DifftoolInfo, error) {
	reader, err := difftoolDiff(local, remote)
	if err != nil {
		return nil, tui.DifftoolInfo{}, err
	}

	files, err := diff.Parse(reader)
	if err != nil {
		return nil, tui.DifftoolInfo{}, fmt.Errorf("parsing diff output: %w", err)
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "No changes found.")
		return nil, tui.DifftoolInfo{}, nil
	}

	// --difftool-path is git's $MERGED, the real working-tree filename;
	// $LOCAL/$REMOTE are temp files with meaningless generated names.
	// Without this rewrite, Chroma's filename-keyed lexer selection would
	// fail to detect the language and the title bar would show a temp path
	// (04-RESEARCH.md Pattern 4).
	base := filepath.Base(path)
	files[0].OldName = base
	files[0].NewName = base

	counter, total := difftoolCounters()

	// Load the post-image lines for full-file rendering directly from the
	// already-materialised --difftool-remote file — git show has no meaning
	// for a temp path (04-RESEARCH.md Pattern 4). Opened read-only; never
	// written to (T-04-03-03). A /dev/null remote (a deletion) legitimately
	// yields an empty slice.
	var newFileLines []string
	if data, readErr := os.ReadFile(remote); readErr == nil {
		newFileLines = splitDifftoolFileLines(data)
	}

	dt := tui.DifftoolInfo{
		Enabled:      true,
		Counter:      counter,
		Total:        total,
		Filename:     base,
		NewFileLines: newFileLines,
	}
	return files, dt, nil
}

// difftoolDiff runs `git diff --no-index -- local remote` and returns its
// stdout as an io.Reader over the resulting diff, interpreting exit codes
// per git diff --no-index's own contract (NOT per internal/git.ExecRunner's,
// which hardcodes a diff-runner-specific "not a git repository" branch that
// does not apply here):
//   - exit 0: the two files are identical — stdout is empty, and the caller's
//     diff.Parse(...) correctly returns zero files.
//   - exit 1: the files differ — stdout holds the diff to parse.
//   - anything else: a real failure, surfaced with git's stderr.
//
// Deliberately does NOT apply internal/git.NormalizeCRLF: difftool mode
// compares two already-materialised local files (git's own $LOCAL/$REMOTE
// temp files) whose carriage returns are legitimate file content, not a
// git-subprocess line-ending artifact — stripping them would corrupt the
// diff of any CRLF-formatted source file (04-RESEARCH.md Pitfall F).
func difftoolDiff(local, remote string) (io.Reader, error) {
	// SECURITY: exec.Command uses argv form — each element is a separate
	// argument. Shell metacharacters in local/remote (user-supplied paths
	// crossing a subprocess boundary) are never interpreted (ASVS V5,
	// T-04-03-02). The "--" separator precedes the two path arguments so a
	// path beginning with a hyphen cannot be parsed as a git option.
	cmd := exec.Command("git", "diff", "--no-index", "--", local, remote) //nolint:gosec
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		// Exit 0: files are identical.
		return &stdout, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		// Exit 1: files differ — stdout holds the diff to parse.
		return &stdout, nil
	}

	// Any other outcome (unexpected exit code, or a non-ExitError failure
	// such as git not being on PATH) is a real failure.
	if msg := strings.TrimRight(stderr.String(), "\n"); msg != "" {
		return nil, fmt.Errorf("git diff --no-index: %s", msg)
	}
	return nil, fmt.Errorf("git diff --no-index: %w", err)
}

// difftoolCounters reads GIT_DIFF_PATH_COUNTER and GIT_DIFF_PATH_TOTAL and
// validates both as positive integers (T-04-03-01). An absent, empty,
// non-numeric, or non-positive value on either variable is treated as
// "counters unavailable" and both are set to 0, which selects the
// counter-less title-bar template. Neither raw env string is ever rendered:
// they arrive from outside the process and must reach the terminal only as
// validated integers.
func difftoolCounters() (counter, total int) {
	c, cErr := strconv.Atoi(os.Getenv("GIT_DIFF_PATH_COUNTER"))
	t, tErr := strconv.Atoi(os.Getenv("GIT_DIFF_PATH_TOTAL"))
	if cErr != nil || tErr != nil || c <= 0 || t <= 0 {
		return 0, 0
	}
	return c, t
}

// splitDifftoolFileLines splits raw file bytes into lines for full-file
// difftool rendering, normalising CRLF and stripping the final newline so
// each element holds one logical line. Mirrors internal/tui.splitFileBytes's
// logic; replicated locally rather than exported across the package
// boundary (04-03-PLAN.md Task 2).
func splitDifftoolFileLines(data []byte) []string {
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	if len(content) > 0 && content[len(content)-1] == '\n' {
		content = content[:len(content)-1]
	}
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}

// reportError routes an error to its exit code, writing at most one line to
// w (D-11, T-02-07):
//   - *git.ExitCodeError with a non-empty Msg → print Msg, return Code
//     (distinguishes no-repo from no-git; also covers errAborted's Code: 1)
//   - *git.ExitCodeError with an empty Msg    → print nothing, return Code
//     (the abort path, CR-02: exit status 1 is the entire signal, exactly
//     as the pre-fix os.Exit(1) path behaved — a stray blank line would be
//     new, unwanted output)
//   - any other error                         → print err.Error(), return 1
func reportError(err error, w io.Writer) int {
	var exitErr *git.ExitCodeError
	if errors.As(err, &exitErr) {
		if exitErr.Msg != "" {
			fmt.Fprintln(w, exitErr.Msg)
		}
		return exitErr.Code
	}
	fmt.Fprintln(w, err.Error())
	return 1
}

// main executes the root command and routes exit codes via reportError.
func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(reportError(err, os.Stderr))
	}
}
