// theme.go implements THEME-01's terminal background resolution — a
// 50ms-bounded OSC 11 auto-detection with a dark fallback, overridable by
// an explicit --theme flag or the config `theme` key, and skipped entirely
// in difftool mode (D-06).

package config

import (
	"fmt"
	"io"
	"time"

	"github.com/muesli/termenv"
)

// Theme identifies one of the three valid `theme` config-key / --theme flag
// values. The zero value (empty string) is treated as ThemeAuto by ParseTheme
// so that an unset flag and an unset config key both mean "not specified".
type Theme string

// The three valid theme values (04-UI-SPEC.md Color — Theme Resolution).
const (
	ThemeLight Theme = "light"
	ThemeDark  Theme = "dark"
	ThemeAuto  Theme = "auto"
)

// DetectTimeout bounds how long DetectDarkBackground may block on an OSC 11
// terminal round-trip before falling back to dark (THEME-01). termenv's own
// internal OSCTimeout is a 5-second `const`, not a `var` — importing code
// cannot lower it (04-RESEARCH.md Pitfall A) — so DetectDarkBackground
// imposes this external, lower bound via its own goroutine race instead.
const DetectTimeout = 50 * time.Millisecond

// ParseTheme validates s against the three known theme values. The empty
// string is treated as ThemeAuto so an unset --theme flag or unset config
// `theme` key both mean "not specified" rather than requiring callers to
// special-case emptiness separately. Any other value is rejected with the
// exact 04-UI-SPEC.md Copywriting Contract error string.
func ParseTheme(s string) (Theme, error) {
	switch Theme(s) {
	case "":
		return ThemeAuto, nil
	case ThemeLight, ThemeDark, ThemeAuto:
		return Theme(s), nil
	default:
		return "", fmt.Errorf("config: invalid theme %q (must be \"light\", \"dark\", or \"auto\")", s)
	}
}

// DetectDarkBackground queries the terminal's background color via OSC 11
// (through termenv.HasDarkBackground) and returns whether it is dark,
// bounded to DetectTimeout (50ms) rather than termenv's own 5-second
// internal bound.
//
// Implementation note (FA-04-02): the result channel is buffered so the
// query goroutine's send never blocks after the select has already taken the
// timeout branch — this lets the abandoned goroutine complete and exit
// (bounded by termenv's own OSCTimeout const) instead of leaking forever.
// The residual risk, not fully resolved by this design, is that the
// abandoned goroutine still holds a read on the same terminal file
// descriptor that bubbletea is about to take over immediately after this
// function returns; RESEARCH.md asserts this exits cleanly without
// consuming a keystroke destined for the TUI, but that has not been
// independently verified here (flagged assumption FA-04-02). This function
// must only be called before tea.NewProgram takes the TTY (D-06).
func DetectDarkBackground(w io.Writer) bool {
	result := make(chan bool, 1)
	go func() {
		result <- termenv.NewOutput(w).HasDarkBackground()
	}()

	select {
	case dark := <-result:
		return dark
	case <-time.After(DetectTimeout):
		return true
	}
}

// ResolveDarkBackground implements the D-07 precedence chain as a pure,
// table-testable function with an injectable detector (so tests never need a
// real terminal):
//
//  1. flagTheme is light or dark → return accordingly (an explicit --theme
//     flag always wins).
//  2. cfgTheme is light or dark → return accordingly (an explicit config
//     `theme` key wins over auto-detection).
//  3. difftoolMode is true → return true (dark) without calling detect at
//     all — D-06 forbids the OSC 11 terminal round-trip when git is the
//     parent process (PITFALLS.md Pitfall 10).
//  4. detect is non-nil → return detect().
//  5. otherwise → return true, the dark fallback (D-07).
func ResolveDarkBackground(flagTheme, cfgTheme Theme, difftoolMode bool, detect func() bool) bool {
	switch flagTheme {
	case ThemeLight:
		return false
	case ThemeDark:
		return true
	}

	switch cfgTheme {
	case ThemeLight:
		return false
	case ThemeDark:
		return true
	}

	if difftoolMode {
		return true
	}

	if detect != nil {
		return detect()
	}

	return true
}
