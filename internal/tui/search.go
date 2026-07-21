package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Match represents a single search hit with per-line display positions.
// ColStart and ColEnd are grapheme-width column offsets within the line,
// compatible with lipgloss.StyleRanges.
type Match struct {
	Line     int // 0-based line index in the ANSI-stripped content
	ColStart int // grapheme-width column where the match begins (inclusive)
	ColEnd   int // grapheme-width column where the match ends (exclusive)
}

// findMatches returns all non-overlapping occurrences of query in content.
// content may contain ANSI escape codes; searching is done on the stripped text.
// ColStart/ColEnd are grapheme-width positions compatible with lipgloss.StyleRanges.
// Returns nil when query is empty.
func findMatches(content, query string) []Match {
	if query == "" {
		return nil
	}
	plain := ansi.Strip(content)
	lines := strings.Split(plain, "\n")
	var matches []Match
	for lineN, line := range lines {
		i := 0
		for len(line)-i >= len(query) {
			j := strings.Index(line[i:], query)
			if j < 0 {
				break
			}
			byteStart := i + j
			byteEnd := byteStart + len(query)
			colStart := ansi.StringWidth(line[:byteStart])
			colEnd := colStart + ansi.StringWidth(query)
			matches = append(matches, Match{lineN, colStart, colEnd})
			i = byteEnd
		}
	}
	return matches
}
