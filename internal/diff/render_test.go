package diff_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/alturd/alturd/internal/diff"
)

// testWidth is the fixed terminal width used in all Render tests.
// Phase 3 passes the actual terminal width; tests use a fixed value (D-04).
const testWidth = 160

// ansiResetStr is the ANSI SGR full-reset sequence used in column-boundary assertions.
// Using a named constant keeps the raw escape byte out of string comparison calls.
const ansiResetStr = "\x1b[0m"

// intraLineSpanRe matches the brighter background ANSI sequences used for
// intra-line span highlights. We assert their PRESENCE on small Modified rows and
// their ABSENCE on guard-tripped rows. The pattern is intentionally broad so it
// catches any SGR sequence that is NOT the standard line-level diff colour
// (we use line-level diff colours that are distinguishable from intra-line spans
// in the implementation; here we assert span presence vs absence structurally).
//
// The guard-path assertion approach: for large-line.diff and many-tokens.diff we
// parse the Render output and verify that NO intra-line span markers appear
// (those rows should show line-level diff colour only). For simple.diff Modified
// rows we assert that span markers ARE present.
//
// This is built as two separate regexps to keep assertions readable.
var (
	// ansiRenderRe matches any ANSI escape sequence in a rendered row.
	ansiRenderRe = regexp.MustCompile("\x1b\\[[0-9;]+m")

	// intraLineRe matches the specific brighter-highlight sequences emitted for
	// intra-line spans. The render implementation uses a distinct SGR code for
	// intra-line span backgrounds vs line-level diff backgrounds; this regexp
	// matches the intra-line marker sequence.
	// NOTE: the actual pattern depends on the render implementation; the test
	// asserts presence/absence of this pattern to verify guard behaviour (D-07).
	// We will refine this after the GREEN implementation is in place.
	intraLineRe = regexp.MustCompile("\x1b\\[1m")
)

// renderFile is a test helper that parses a fixture and calls diff.Render.
func renderFile(t *testing.T, fixture string, width int) ([]string, string) {
	t.Helper()
	file := parseFirst(t, fixture)
	rows := diff.Render(file, width, diff.FullFile)
	return rows, file.NewName
}

func TestRender(t *testing.T) {
	t.Run("non_empty_rows_simple", func(t *testing.T) {
		// Render on simple.diff must return a non-empty []string (DIFF-01, D-03).
		rows, _ := renderFile(t, "simple.diff", testWidth)
		if len(rows) == 0 {
			t.Error("Render(simple.diff): expected non-empty []string, got 0 rows")
		}
	})

	t.Run("rows_contain_ansi_codes", func(t *testing.T) {
		// At least one row in simple.diff must contain ANSI escape codes (DIFF-02, DIFF-03).
		rows, _ := renderFile(t, "simple.diff", testWidth)
		hasANSI := false
		for _, row := range rows {
			if ansiRenderRe.MatchString(row) {
				hasANSI = true
				break
			}
		}
		if !hasANSI {
			t.Error("Render(simple.diff): no ANSI codes found in any rendered row")
		}
	})

	t.Run("reset_at_column_boundary", func(t *testing.T) {
		// Every rendered row must contain the ANSI reset at the left/right column
		// boundary to prevent chroma colour bleed (RESEARCH Pitfall 1, DIFF-01).
		rows, _ := renderFile(t, "simple.diff", testWidth)
		if len(rows) == 0 {
			t.Fatal("Render(simple.diff): got 0 rows")
		}
		for i, row := range rows {
			if !strings.Contains(row, ansiResetStr) {
				t.Errorf("row[%d]: does not contain ANSI reset at column boundary: %q", i, row[:min(len(row), 60)])
			}
		}
	})

	t.Run("intra_line_markers_on_modified_row", func(t *testing.T) {
		// A Modified row in simple.diff must contain intra-line span markers
		// in addition to the line-level diff background colour (DIFF-04, D-06).
		rows, _ := renderFile(t, "simple.diff", testWidth)
		foundModifiedWithSpan := false
		for _, row := range rows {
			// intraLineRe matches the bold/bright sequence used for intra-line spans.
			if intraLineRe.MatchString(row) {
				foundModifiedWithSpan = true
				break
			}
		}
		if !foundModifiedWithSpan {
			t.Error("Render(simple.diff): no intra-line span markers found on any Modified row (DIFF-04)")
		}
	})

	t.Run("large_line_guard_no_intra_markers", func(t *testing.T) {
		// large-line.diff has a Modified row > 1000 chars; the 1000-char guard must
		// trip and suppress intra-line markers (D-07). Only line-level diff colour
		// should be present.
		rows, _ := renderFile(t, "large-line.diff", testWidth)
		for _, row := range rows {
			if intraLineRe.MatchString(row) {
				t.Error("Render(large-line.diff): intra-line span markers present but 1000-char guard should have suppressed them (D-07)")
				break
			}
		}
	})

	t.Run("many_tokens_guard_no_intra_markers", func(t *testing.T) {
		// many-tokens.diff has a Modified row with > 200 tokens; the token guard
		// must trip and suppress intra-line markers (D-07).
		rows, _ := renderFile(t, "many-tokens.diff", testWidth)
		for _, row := range rows {
			if intraLineRe.MatchString(row) {
				t.Error("Render(many-tokens.diff): intra-line span markers present but 200-token guard should have suppressed them (D-07)")
				break
			}
		}
	})

	t.Run("binary_no_panic", func(t *testing.T) {
		// Render on binary.diff must return without panic and emit the placeholder row.
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Render(binary.diff) panicked: %v", r)
			}
		}()
		rows, _ := renderFile(t, "binary.diff", testWidth)
		if len(rows) == 0 {
			t.Error("Render(binary.diff): expected at least 1 placeholder row")
		}
	})

	t.Run("no_panic_all_fixtures", func(t *testing.T) {
		// Render must not panic on any fixture in the corpus (DIFF-07).
		fixtures := []string{
			"simple.diff", "binary.diff", "rename.diff", "mode-only.diff",
			"submodule.diff", "no-newline.diff", "new-file.diff", "deleted-file.diff",
			"multi-file.diff", "multi-hunk.diff", "large-line.diff",
			"many-tokens.diff", "multiline-string.diff",
		}
		for _, fixture := range fixtures {
			fixture := fixture
			t.Run(fixture, func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Render(%s) panicked: %v", fixture, r)
					}
				}()
				file := parseFirst(t, fixture)
				_ = diff.Render(file, testWidth, diff.FullFile)
			})
		}
	})

	t.Run("width_independence", func(t *testing.T) {
		// Calling Render twice with different widths must produce independent
		// results — no global state mutation (D-04).
		file := parseFirst(t, "simple.diff")
		rows80 := diff.Render(file, 80, diff.FullFile)
		rows160 := diff.Render(file, testWidth, diff.FullFile)
		if len(rows80) == 0 || len(rows160) == 0 {
			t.Fatal("Render: got 0 rows for at least one width")
		}
		// Row counts must be the same (width changes column widths, not row count).
		if len(rows80) != len(rows160) {
			t.Errorf("Render: different row counts for width 80 (%d) vs 160 (%d)", len(rows80), len(rows160))
		}
		// Re-calling must not change the row count (no global mutation).
		rows160b := diff.Render(file, testWidth, diff.FullFile)
		if len(rows160) != len(rows160b) {
			t.Errorf("Render: second call with same width produces different count (%d vs %d)", len(rows160), len(rows160b))
		}
	})
}

// min returns the smaller of a and b.
// Provided as a local helper since Go 1.21+ builtin min is available,
// but we include it for explicitness in test output truncation.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
