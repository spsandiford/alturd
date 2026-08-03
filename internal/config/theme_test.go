package config_test

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/alturd/alturd/internal/config"
)

// TestParseTheme covers the three valid values, the empty string (treated as
// ThemeAuto), and one invalid value asserted against the full error string
// (04-UI-SPEC.md Copywriting Contract, "Invalid theme value" row).
func TestParseTheme(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    config.Theme
		wantErr string
	}{
		{name: "light", in: "light", want: config.ThemeLight},
		{name: "dark", in: "dark", want: config.ThemeDark},
		{name: "auto", in: "auto", want: config.ThemeAuto},
		{name: "empty_treated_as_auto", in: "", want: config.ThemeAuto},
		{
			name:    "invalid",
			in:      "solarized",
			wantErr: `config: invalid theme "solarized" (must be "light", "dark", or "auto")`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := config.ParseTheme(tc.in)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("ParseTheme(%q) = _, nil, want error %q", tc.in, tc.wantErr)
				}
				if got := err.Error(); got != tc.wantErr {
					t.Errorf("ParseTheme(%q) error = %q, want %q", tc.in, got, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseTheme(%q) = _, %v, want nil error", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("ParseTheme(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestTheme_Precedence walks the full D-07 precedence matrix using a closure
// detector that increments a counter, asserting both the returned value and
// the exact call count (0 or 1) for every row — the call count is what
// proves D-06 and the explicit-override rows skip detection rather than
// merely ignoring its result.
func TestTheme_Precedence(t *testing.T) {
	tests := []struct {
		name         string
		flagTheme    config.Theme
		cfgTheme     config.Theme
		difftoolMode bool
		detectResult bool
		nilDetector  bool
		wantDark     bool
		wantCalls    int
	}{
		{
			name:      "flag_light_wins_over_config_dark",
			flagTheme: config.ThemeLight,
			cfgTheme:  config.ThemeDark,
			wantDark:  false,
			wantCalls: 0,
		},
		{
			name:      "flag_dark_wins_over_config_light",
			flagTheme: config.ThemeDark,
			cfgTheme:  config.ThemeLight,
			wantDark:  true,
			wantCalls: 0,
		},
		{
			name:      "config_light_wins_over_auto_detect",
			flagTheme: config.ThemeAuto,
			cfgTheme:  config.ThemeLight,
			wantDark:  false,
			wantCalls: 0,
		},
		{
			name:      "config_dark_wins_over_auto_detect",
			flagTheme: config.ThemeAuto,
			cfgTheme:  config.ThemeDark,
			wantDark:  true,
			wantCalls: 0,
		},
		{
			name:         "auto_calls_detect_once_true",
			flagTheme:    config.ThemeAuto,
			cfgTheme:     config.ThemeAuto,
			detectResult: true,
			wantDark:     true,
			wantCalls:    1,
		},
		{
			name:         "auto_calls_detect_once_false",
			flagTheme:    config.ThemeAuto,
			cfgTheme:     config.ThemeAuto,
			detectResult: false,
			wantDark:     false,
			wantCalls:    1,
		},
		{
			name:         "difftool_mode_skips_detect_entirely",
			flagTheme:    config.ThemeAuto,
			cfgTheme:     config.ThemeAuto,
			difftoolMode: true,
			wantDark:     true,
			wantCalls:    0,
		},
		{
			name:        "nil_detector_falls_back_to_dark",
			flagTheme:   config.ThemeAuto,
			cfgTheme:    config.ThemeAuto,
			nilDetector: true,
			wantDark:    true,
			wantCalls:   0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			var detect func() bool
			if !tc.nilDetector {
				detect = func() bool {
					calls++
					return tc.detectResult
				}
			}

			got := config.ResolveDarkBackground(tc.flagTheme, tc.cfgTheme, tc.difftoolMode, detect)
			if got != tc.wantDark {
				t.Errorf("ResolveDarkBackground(%q, %q, %v, ...) = %v, want %v",
					tc.flagTheme, tc.cfgTheme, tc.difftoolMode, got, tc.wantDark)
			}
			if calls != tc.wantCalls {
				t.Errorf("ResolveDarkBackground(%q, %q, %v, ...) called detect %d times, want %d",
					tc.flagTheme, tc.cfgTheme, tc.difftoolMode, calls, tc.wantCalls)
			}
		})
	}
}

// TestDetectDarkBackground_TimeoutFallback passes an io.Writer backed by a
// pipe whose read end is never written to, asserts the result is true (dark
// fallback), and asserts the call completes well under 200ms — a bound, not
// an exact timing assertion, to avoid flakiness on loaded CI runners.
func TestDetectDarkBackground_TimeoutFallback(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer func() { _ = r.Close() }()
	defer func() { _ = w.Close() }()

	var writer io.Writer = w

	start := time.Now()
	got := config.DetectDarkBackground(writer)
	elapsed := time.Since(start)

	if !got {
		t.Errorf("DetectDarkBackground(never-responding writer) = false, want true (dark fallback)")
	}
	if elapsed >= 200*time.Millisecond {
		t.Errorf("DetectDarkBackground(never-responding writer) took %v, want < 200ms", elapsed)
	}
}
