package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// findMatches returns plain-text (ANSI-stripped) byte positions of all
// non-overlapping occurrences of query in content.
//
// The returned positions are in the same coordinate space that
// viewport.SetHighlights() expects: character offsets in the ANSI-stripped
// string (Assumption A2). If query is empty, nil is returned.
func findMatches(content, query string) [][]int {
	if query == "" {
		return nil
	}
	plain := ansi.Strip(content)
	var matches [][]int
	for i := 0; i < len(plain); {
		j := strings.Index(plain[i:], query)
		if j < 0 {
			break
		}
		start := i + j
		end := start + len(query)
		matches = append(matches, []int{start, end})
		i = end
	}
	return matches
}
