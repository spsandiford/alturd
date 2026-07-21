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

	t.Run("plain_text_single_line", func(t *testing.T) {
		got := findMatches("abcabc", "bc")
		if len(got) != 2 {
			t.Fatalf("findMatches(plain): got %d matches, want 2", len(got))
		}
		if got[0].Line != 0 || got[0].ColStart != 1 || got[0].ColEnd != 3 {
			t.Errorf("findMatches(plain): first match = %+v, want {Line:0 ColStart:1 ColEnd:3}", got[0])
		}
		if got[1].Line != 0 || got[1].ColStart != 4 || got[1].ColEnd != 6 {
			t.Errorf("findMatches(plain): second match = %+v, want {Line:0 ColStart:4 ColEnd:6}", got[1])
		}
	})

	t.Run("multi_line", func(t *testing.T) {
		got := findMatches("hello\nworld", "world")
		if len(got) != 1 {
			t.Fatalf("findMatches(multi_line): got %d matches, want 1", len(got))
		}
		if got[0].Line != 1 || got[0].ColStart != 0 || got[0].ColEnd != 5 {
			t.Errorf("findMatches(multi_line): match = %+v, want {Line:1 ColStart:0 ColEnd:5}", got[0])
		}
	})

	t.Run("ansi_stripped_positions", func(t *testing.T) {
		const reverseOn = "\x1b[7m"
		const reverseOff = "\x1b[27m"
		// After stripping: "prefixhello"; "hello" starts at col 6.
		content := reverseOn + "prefix" + reverseOff + "hello"
		got := findMatches(content, "hello")
		if len(got) != 1 {
			t.Fatalf("findMatches(ansi): got %d matches, want 1", len(got))
		}
		if got[0].Line != 0 || got[0].ColStart != 6 || got[0].ColEnd != 11 {
			t.Errorf("findMatches(ansi): match = %+v, want {Line:0 ColStart:6 ColEnd:11}", got[0])
		}
	})

	t.Run("ansi_multi_line", func(t *testing.T) {
		// Simulate two diff rows: ANSI-coded, separated by \n.
		content := "\x1b[31m-old line\x1b[0m\n\x1b[32m+new world\x1b[0m"
		got := findMatches(content, "world")
		if len(got) != 1 {
			t.Fatalf("findMatches(ansi_multi_line): got %d matches, want 1", len(got))
		}
		// Stripped line 1: "+new world". "world" starts at col 5.
		if got[0].Line != 1 || got[0].ColStart != 5 || got[0].ColEnd != 10 {
			t.Errorf("findMatches(ansi_multi_line): match = %+v, want {Line:1 ColStart:5 ColEnd:10}", got[0])
		}
	})

	t.Run("non_overlapping", func(t *testing.T) {
		got := findMatches("aaaa", "aa")
		if len(got) != 2 {
			t.Fatalf("findMatches(non_overlapping): got %d matches, want 2", len(got))
		}
		if got[0].ColStart != 0 || got[0].ColEnd != 2 {
			t.Errorf("findMatches(non_overlapping): first match = %+v, want {ColStart:0 ColEnd:2}", got[0])
		}
		if got[1].ColStart != 2 || got[1].ColEnd != 4 {
			t.Errorf("findMatches(non_overlapping): second match = %+v, want {ColStart:2 ColEnd:4}", got[1])
		}
	})
}
