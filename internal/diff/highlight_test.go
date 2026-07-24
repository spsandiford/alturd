package diff_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/alturd/alturd/internal/diff"
)

// ansiColorRe matches any ANSI SGR colour/style escape sequence.
// Constructed via a string literal rather than a raw backtick literal so the
// ESC byte (\x1b) is unambiguous in the source (D-05: no golden ANSI snapshots,
// but ANSI presence assertions are required).
var ansiColorRe = regexp.MustCompile("\x1b\\[[0-9;]+m")

func TestHighlight(t *testing.T) {
	t.Run("detectable_language_has_ansi", func(t *testing.T) {
		// simple.diff is a Markdown file (README.md); Highlight must produce
		// ANSI codes in at least one non-blank row's ANSI field (DIFF-02).
		file := parseFirst(t, "simple.diff")
		pairs := diff.Align(file, diff.FullFile)
		if err := diff.Highlight(pairs, file.NewName); err != nil {
			t.Fatalf("Highlight(simple.diff): %v", err)
		}
		hasANSI := false
		for _, p := range pairs {
			if p.Left.Kind != diff.LineBlank && ansiColorRe.MatchString(p.Left.ANSI) {
				hasANSI = true
				break
			}
			if p.Right.Kind != diff.LineBlank && ansiColorRe.MatchString(p.Right.ANSI) {
				hasANSI = true
				break
			}
		}
		if !hasANSI {
			t.Error("Highlight(simple.diff): expected ANSI colour codes in at least one non-blank row")
		}
	})

	t.Run("multiline_string_lines_end_with_reset", func(t *testing.T) {
		// multiline-string.diff contains a Go backtick multi-line string.
		// After Highlight + per-line split, every non-empty ANSI field must end
		// with the reset sequence to prevent colour bleed (RESEARCH Pitfall 2).
		file := parseFirst(t, "multiline-string.diff")
		pairs := diff.Align(file, diff.FullFile)
		if err := diff.Highlight(pairs, file.NewName); err != nil {
			t.Fatalf("Highlight(multiline-string.diff): %v", err)
		}
		for i, p := range pairs {
			for side, ansi := range []string{p.Left.ANSI, p.Right.ANSI} {
				if ansi == "" {
					continue
				}
				if !strings.HasSuffix(ansi, "\x1b[0m") {
					// Show a suffix for diagnosis without printing the whole escape string.
					suffix := ansi
					if len(suffix) > 20 {
						suffix = "…" + suffix[len(suffix)-20:]
					}
					t.Errorf("pair[%d] side=%d: ANSI does not end with reset; suffix=%q", i, side, suffix)
				}
			}
		}
	})

	t.Run("binary_placeholder_passed_through", func(t *testing.T) {
		// binary.diff produces a single placeholder RowPair via Align.
		// Highlight must NOT apply chroma — ANSI must equal raw Content (DIFF-07).
		file := parseFirst(t, "binary.diff")
		pairs := diff.Align(file, diff.FullFile)
		if err := diff.Highlight(pairs, file.NewName); err != nil {
			t.Fatalf("Highlight(binary.diff): %v", err)
		}
		if len(pairs) == 0 {
			t.Fatal("binary.diff Align: expected at least 1 pair")
		}
		// The placeholder row must have ANSI == Content (no chroma processing).
		got := pairs[0].Left.ANSI
		want := pairs[0].Left.Content
		if got != want {
			t.Errorf("binary placeholder: ANSI %q != Content %q (chroma was applied)", got, want)
		}
	})

	t.Run("mode_only_placeholder_passed_through", func(t *testing.T) {
		// mode-only.diff: single placeholder RowPair; Highlight must bypass chroma.
		file := parseFirst(t, "mode-only.diff")
		pairs := diff.Align(file, diff.FullFile)
		if err := diff.Highlight(pairs, file.NewName); err != nil {
			t.Fatalf("Highlight(mode-only.diff): %v", err)
		}
		if len(pairs) == 0 {
			t.Fatal("mode-only.diff Align: expected at least 1 pair")
		}
		if pairs[0].Left.ANSI != pairs[0].Left.Content {
			t.Errorf("mode-only placeholder: ANSI != Content (chroma was applied)")
		}
	})

	t.Run("submodule_placeholder_passed_through", func(t *testing.T) {
		// submodule.diff: commit SHA lines should not be chroma-highlighted.
		// Verify that no ANSI colour codes appear in submodule row ANSI fields.
		file := parseFirst(t, "submodule.diff")
		pairs := diff.Align(file, diff.FullFile)
		if err := diff.Highlight(pairs, file.NewName); err != nil {
			t.Fatalf("Highlight(submodule.diff): %v", err)
		}
		if len(pairs) == 0 {
			t.Fatal("submodule.diff Align: expected at least 1 pair")
		}
		for i, p := range pairs {
			// ANSI must equal Content (no chroma colour codes injected).
			if p.Left.Kind != diff.LineBlank && p.Left.ANSI != p.Left.Content {
				t.Errorf("submodule pair[%d].Left: ANSI != Content (chroma applied to SHA)", i)
			}
			if p.Right.Kind != diff.LineBlank && p.Right.ANSI != p.Right.Content {
				t.Errorf("submodule pair[%d].Right: ANSI != Content (chroma applied to SHA)", i)
			}
		}
	})

	t.Run("empty_pairs_no_error", func(t *testing.T) {
		// Highlight on nil/empty pairs must return nil (no-op, no panic).
		if err := diff.Highlight(nil, "file.go"); err != nil {
			t.Errorf("Highlight(nil, ...): expected nil, got %v", err)
		}
		if err := diff.Highlight([]diff.RowPair{}, "file.go"); err != nil {
			t.Errorf("Highlight(empty, ...): expected nil, got %v", err)
		}
	})

	t.Run("set_dark_background_selects_style", func(t *testing.T) {
		// SetDarkBackground must switch style so subsequent Highlight calls use ANSI
		// output appropriate for the terminal background. We verify indirectly: both
		// styles must produce ANSI output (non-empty), and the ANSI strings from each
		// style must differ (confirming the switch actually changes output).
		file := parseFirst(t, "simple.diff")

		diff.SetDarkBackground(true)
		pairsDark := diff.Align(file, diff.FullFile)
		if err := diff.Highlight(pairsDark, file.NewName); err != nil {
			t.Fatalf("Highlight(dark): %v", err)
		}

		diff.SetDarkBackground(false)
		pairsLight := diff.Align(file, diff.FullFile)
		if err := diff.Highlight(pairsLight, file.NewName); err != nil {
			t.Fatalf("Highlight(light): %v", err)
		}

		// Restore default so other tests are not affected.
		diff.SetDarkBackground(true)

		// Both styles must produce at least one ANSI-coloured row.
		hasDark, hasLight := false, false
		for i := range pairsDark {
			if ansiColorRe.MatchString(pairsDark[i].Left.ANSI) {
				hasDark = true
			}
			if ansiColorRe.MatchString(pairsLight[i].Left.ANSI) {
				hasLight = true
			}
		}
		if !hasDark {
			t.Error("SetDarkBackground(true): expected ANSI colour in at least one row")
		}
		if !hasLight {
			t.Error("SetDarkBackground(false): expected ANSI colour in at least one row")
		}

		// The two styles must produce different ANSI output.
		same := true
		for i := range pairsDark {
			if pairsDark[i].Left.ANSI != pairsLight[i].Left.ANSI {
				same = false
				break
			}
		}
		if same {
			t.Error("SetDarkBackground: dark and light styles produced identical ANSI output — style switch had no effect")
		}
	})
}
