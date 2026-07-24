package diff

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// ansiReset is the ANSI SGR full-reset sequence. It clears all terminal colour
// and style attributes. Appended to every per-line highlighted string to prevent
// multi-line token colour from bleeding across the "\n" split (RESEARCH Pitfall 2),
// and emitted at the left-column boundary in render.go to prevent cross-column
// colour bleed (RESEARCH Pitfall 1).
const ansiReset = "\x1b[0m"

// chromaStyleName is the Chroma syntax-highlight style applied to all diff content.
// Defaults to "monokai" (tuned for dark terminal backgrounds). Call SetDarkBackground
// once at startup — before tea.NewProgram — to select the appropriate style.
var chromaStyleName = "monokai"

// SetDarkBackground switches the Chroma style to one suited for the terminal
// background. dark=true keeps "monokai"; dark=false switches to "github" whose
// default-text token uses the terminal foreground colour and is readable on light
// backgrounds. Must be called before the first Highlight invocation.
func SetDarkBackground(dark bool) {
	if dark {
		chromaStyleName = "monokai"
	} else {
		chromaStyleName = "github"
	}
}

// Highlight applies Chroma syntax highlighting to every RowPair in pairs,
// populating the ANSI field of each non-blank RenderedLine. It must be called
// after Align and before Render. Highlight mutates pairs in-place.
//
// # Tokenisation strategy
//
// All left-side content is joined with "\n", tokenised once per side, formatted,
// and split back on "\n" to recover per-line ANSI strings. The same process
// applies independently to the right side. This prevents the anti-pattern of
// re-tokenising per render frame (PATTERNS §highlight.go).
//
// # Placeholder rows
//
// Placeholder rows produced by Align for binary/mode-only/submodule files are
// detected by isPlaceholderPairs and passed through with raw Content copied to
// ANSI unchanged — these strings are not source code and must not be highlighted
// (DIFF-07).
//
// # Multi-line token colour bleed (RESEARCH Pitfall 2)
//
// After splitting chroma output on "\n", any line that does not already end
// with ansiReset gets one appended. This prevents a multi-line string literal
// token from leaving terminal colour set across the line boundary.
//
// # Phase 4 note
//
// The formatter ("terminal16m") and the two style names are hardcoded here.
// Phase 4 will make both fully configurable via the user's config.toml.
func Highlight(pairs []RowPair, filename string) error {
	if len(pairs) == 0 {
		return nil
	}

	// Placeholder files (binary/mode-only/submodule): copy Content to ANSI unchanged.
	if isPlaceholderPairs(pairs) {
		for i := range pairs {
			pairs[i].Left.ANSI = pairs[i].Left.Content
			pairs[i].Right.ANSI = pairs[i].Right.Content
		}
		return nil
	}

	// Select lexer — fallback to plaintext when no lexer matches the filename.
	lex := lexers.Match(filename)
	if lex == nil {
		lex = lexers.Fallback
	}

	sty := styles.Get(chromaStyleName)
	if sty == nil {
		sty = styles.Fallback
	}

	// Phase 4 will auto-select formatter based on terminal colour depth.
	fmtr := formatters.Get("terminal16m")
	if fmtr == nil {
		fmtr = formatters.Fallback
	}

	// --- Highlight left side ---
	// Collect content from all non-blank left sides, maintaining pair-index mapping.
	var leftContents []string
	var leftPairIdxs []int
	for i, p := range pairs {
		if p.Left.Kind != LineBlank {
			leftPairIdxs = append(leftPairIdxs, i)
			leftContents = append(leftContents, p.Left.Content)
		}
	}
	if len(leftContents) > 0 {
		iterator, err := lex.Tokenise(nil, strings.Join(leftContents, "\n"))
		if err != nil {
			return fmt.Errorf("highlight left side of %q: tokenise: %w", filename, err)
		}
		var sb strings.Builder
		if err := fmtr.Format(&sb, sty, iterator); err != nil {
			return fmt.Errorf("highlight left side of %q: format: %w", filename, err)
		}
		highlighted := splitAndReset(sb.String())
		for j, pairIdx := range leftPairIdxs {
			if j < len(highlighted) {
				pairs[pairIdx].Left.ANSI = highlighted[j]
			}
		}
	}

	// --- Highlight right side ---
	var rightContents []string
	var rightPairIdxs []int
	for i, p := range pairs {
		if p.Right.Kind != LineBlank {
			rightPairIdxs = append(rightPairIdxs, i)
			rightContents = append(rightContents, p.Right.Content)
		}
	}
	if len(rightContents) > 0 {
		iterator, err := lex.Tokenise(nil, strings.Join(rightContents, "\n"))
		if err != nil {
			return fmt.Errorf("highlight right side of %q: tokenise: %w", filename, err)
		}
		var sb strings.Builder
		if err := fmtr.Format(&sb, sty, iterator); err != nil {
			return fmt.Errorf("highlight right side of %q: format: %w", filename, err)
		}
		highlighted := splitAndReset(sb.String())
		for j, pairIdx := range rightPairIdxs {
			if j < len(highlighted) {
				pairs[pairIdx].Right.ANSI = highlighted[j]
			}
		}
	}

	return nil
}

// splitAndReset splits a chroma-formatted string on "\n" to produce per-line
// ANSI strings, then ensures every line ends with ansiReset. This guards against
// multi-line token colour bleed: chroma may emit a single colour sequence around
// a multi-line string literal; splitting mid-sequence leaves the terminal colour
// set for subsequent lines (RESEARCH Pitfall 2).
func splitAndReset(formatted string) []string {
	lines := strings.Split(formatted, "\n")
	for i, line := range lines {
		if !strings.HasSuffix(line, ansiReset) {
			lines[i] = line + ansiReset
		}
	}
	return lines
}

// isPlaceholderPairs reports whether every RowPair in pairs was produced by
// Align as a non-source placeholder row (binary notice, mode-only notice,
// submodule commit reference). Placeholder pairs bypass Chroma highlighting.
//
// Detection is structural: it reads the IsPlaceholder field set by Align,
// avoiding content-prefix heuristics that can misclassify real source files
// (e.g. TOML config files where every changed line starts with "[").
//
// If ANY row is not a placeholder, the file is eligible for highlighting.
func isPlaceholderPairs(pairs []RowPair) bool {
	for _, p := range pairs {
		if !p.IsPlaceholder {
			return false
		}
	}
	return true
}
