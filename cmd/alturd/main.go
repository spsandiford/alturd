// Package main is the entrypoint for the alturd binary.
// It wires the cobra root command to the internal/git runner, diff parser,
// and renderer, producing side-by-side ANSI output on stdout.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/alturd/alturd/internal/diff"
	"github.com/alturd/alturd/internal/git"
	applog "github.com/alturd/alturd/internal/log"
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

// run is the cobra RunE handler. The FIRST statement must be applog.Init() so
// that --version and --help (which cobra handles before RunE is called) never
// create a log file or any side effects (D-10, T-02-08, GIT-04).
func run(cmd *cobra.Command, args []string) error {
	// FIRST: initialise log. Non-fatal — continue without logging if init fails.
	logFile, err := applog.Init()
	if err == nil {
		defer logFile.Close()
	}

	// Build the git diff argument slice: literal "diff" subcommand prepended to
	// the parsed ref/path arguments (ExecRunner.Run does not add "diff" itself).
	gitArgs := append([]string{"diff"}, git.ParseRefArgs(args, cmd.ArgsLenAtDash())...)

	// Run git and obtain an io.Reader over the raw diff output.
	reader, err := git.ExecRunner{}.Run(gitArgs)
	if err != nil {
		// Return unchanged — ExitCodeError sentinels are already typed correctly
		// for main() to extract the exit code via errors.As.
		return err
	}

	// Parse the unified diff into typed File structs.
	files, err := diff.Parse(reader)
	if err != nil {
		return fmt.Errorf("parsing diff output: %w", err)
	}

	// Determine terminal width for column sizing.
	width := terminalWidth()

	// Render each file to stdout as ANSI rows (D-06).
	for _, file := range files {
		rows := diff.Render(file, width)
		for _, row := range rows {
			if _, err := fmt.Fprintln(os.Stdout, row); err != nil {
				return fmt.Errorf("writing output: %w", err)
			}
		}
	}

	return nil
}

// terminalWidth returns the current terminal width when stdout is a TTY, or
// falls back to 160 columns when stdout is redirected or GetSize fails (D-07).
func terminalWidth() int {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
			return w
		}
	}
	return 160
}

// main executes the root command and routes exit codes.
// All error output is a single line to stderr (D-11, T-02-07):
//   - *git.ExitCodeError → print Msg, exit with Code (distinguishes no-repo from no-git)
//   - other error       → print err.Error(), exit 1
func main() {
	if err := rootCmd.Execute(); err != nil {
		var exitErr *git.ExitCodeError
		if errors.As(err, &exitErr) {
			fmt.Fprintln(os.Stderr, exitErr.Msg)
			os.Exit(exitErr.Code)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
