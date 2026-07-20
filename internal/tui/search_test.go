package tui

import (
	"testing"
)

func TestFindMatches(t *testing.T) {
	t.Run("empty_query_returns_nil", func(t *testing.T) {
		got := findMatches("hello world", "")
		if got != nil {
			t.Errorf("findMatches(empty query): got %v, want nil", got)
		}
	})

	t.Run("no_match_returns_nil", func(t *testing.T) {
		got := findMatches("hello world", "xyz")
		if got != nil {
			t.Errorf("findMatches(no match): got %v, want nil", got)
		}
	})

	t.Run("plain_text_positions", func(t *testing.T) {
		got := findMatches("abcabc", "bc")
		if len(got) != 2 {
			t.Fatalf("findMatches(plain): got %d matches, want 2", len(got))
		}
		if got[0][0] != 1 || got[0][1] != 3 {
			t.Errorf("findMatches(plain): first match = %v, want [1 3]", got[0])
		}
		if got[1][0] != 4 || got[1][1] != 6 {
			t.Errorf("findMatches(plain): second match = %v, want [4 6]", got[1])
		}
	})

	t.Run("ansi_stripped_positions", func(t *testing.T) {
		// Build a string with ANSI reverse-video sequence before the query text.
		// The ANSI bytes must NOT be counted in the returned positions.
		const reverseOn = "\x1b[7m"
		const reverseOff = "\x1b[27m"
		content := reverseOn + "prefix" + reverseOff + "hello"
		got := findMatches(content, "hello")
		if len(got) != 1 {
			t.Fatalf("findMatches(ansi): got %d matches, want 1", len(got))
		}
		// After stripping ANSI, plain = "prefixhello". "hello" starts at index 6.
		if got[0][0] != 6 || got[0][1] != 11 {
			t.Errorf("findMatches(ansi): match = %v, want [6 11]", got[0])
		}
	})

	t.Run("non_overlapping", func(t *testing.T) {
		// "aaaa" with query "aa" should yield [0,2] and [2,4], not [0,2],[1,3],[2,4].
		got := findMatches("aaaa", "aa")
		if len(got) != 2 {
			t.Fatalf("findMatches(non_overlapping): got %d matches, want 2", len(got))
		}
		if got[0][0] != 0 || got[0][1] != 2 {
			t.Errorf("findMatches(non_overlapping): first match = %v, want [0 2]", got[0])
		}
		if got[1][0] != 2 || got[1][1] != 4 {
			t.Errorf("findMatches(non_overlapping): second match = %v, want [2 4]", got[1])
		}
	})
}
