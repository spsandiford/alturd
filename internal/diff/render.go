package diff

import (
	"strings"
	"time"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// dmp is the package-level diff-match-patch instance reused across Render calls.
// Allocating once at package init avoids per-call overhead (PATTERNS anti-pattern:
// do NOT allocate diffmatchpatch.New() per call).
var dmp = diffmatchpatch.New()

// Line-level diff background colour sequences using 256-colour ANSI (DIFF-03).
// Applied to the full content of each side based on LineKind.
// Phase 4 will make these theme-configurable.
const (
	bgAdded    = "\x1b[48;5;22m"  // dark green
	bgRemoved  = "\x1b[48;5;52m"  // dark red
	bgModified = "\x1b[48;5;58m"  // dark amber/olive
)

// Intra-line span markers (DIFF-04): bold weight (\x1b[1m) starts the brighter
// highlight on a changed character span; normal-weight (\x1b[22m) ends it.
// Phase 4 may switch to a different highlight (e.g. bright background).
const (
	spanStart = "\x1b[1m"
	spanEnd   = "\x1b[22m"
)

// Render produces the []string side-by-side render contract for a gitdiff.File.
//
// It is the public phase API consumed by Phase 3 (TUI) via:
//
//	viewport.SetContent(strings.Join(rows, "\n"))
//
// Each element is one fully-composed ANSI row (D-03):
//
//	<left-column> + ansiReset + " " + <right-column>
//
// The ansiReset at the column boundary prevents chroma foreground colour from
// the left column from bleeding into the right column (RESEARCH Pitfall 1).
//
// width is the total terminal width; each column receives width/2-1 characters.
// Tests pass testWidth=160; Phase 3 passes the actual terminal width. Render
// is pure and re-callable with different widths — it mutates no package state (D-04).
//
// ANSI layer order:
//  1. Syntax highlighting foreground (chroma, via Highlight)
//  2. Line-level diff background (DIFF-03) wraps the chroma content
//  3. Intra-line bold spans (DIFF-04) on Modified rows when guards permit
func Render(file *gitdiff.File, width int) []string {
	if width < 4 {
		width = 4
	}

	// Align produces side-by-side RowPairs; Highlight populates ANSI fields.
	pairs := Align(file, FullFile)
	_ = Highlight(pairs, file.NewName) // non-fatal; ANSI may fall back to Content

	rows := make([]string, 0, len(pairs))
	for _, p := range pairs {
		rows = append(rows, renderPair(p))
	}
	return rows
}

// renderPair composes one side-by-side ANSI row from a RowPair. For Modified
// pairs it delegates to applyIntraLine; all others use renderSide directly.
func renderPair(p RowPair) string {
	var left, right string
	if p.Left.Kind == LineModifiedOld && p.Right.Kind == LineModifiedNew {
		left, right = applyIntraLine(p)
	} else {
		left = renderSide(p.Left)
		right = renderSide(p.Right)
	}
	return joinColumns(left, right)
}

// renderSide produces the ANSI string for one column side. It applies the
// line-level diff background colour over the syntax-highlighted ANSI content.
// For context and blank lines no background is added.
func renderSide(line RenderedLine) string {
	bg := lineBg(line.Kind)

	// Prefer the Highlight-populated ANSI field; fall back to raw Content.
	content := line.ANSI
	if content == "" {
		content = line.Content
	}

	if bg == "" {
		// Context or blank: return content as-is (no diff colouring).
		return content
	}
	// Wrap with diff background colour; reset at end of side.
	return bg + content + ansiReset
}

// applyIntraLine computes character-level intra-line span markers for a Modified
// RowPair and returns the coloured left and right column strings.
//
// Guards (D-07): if either line exceeds 1000 chars, exceeds 200 tokens, or
// DiffMain takes longer than 100ms, the function falls back to renderSide with
// line-level colour only — no intra-line markers, no visual indicator.
//
// DiffMain is called with checklines=false (D-06) for character-level diff.
func applyIntraLine(p RowPair) (left, right string) {
	oldRaw := p.Left.Content
	newRaw := p.Right.Content

	if shouldSkipIntraLine(oldRaw, newRaw) {
		return renderSide(p.Left), renderSide(p.Right)
	}

	diffs, timedOut := computeIntraLineWithTimeout(oldRaw, newRaw)
	if timedOut {
		return renderSide(p.Left), renderSide(p.Right)
	}

	// Walk diffs to build span-highlighted content for each side.
	// DiffDelete spans appear on the old (left) side; DiffInsert on the new (right).
	var lb, rb strings.Builder
	for _, d := range diffs {
		switch d.Type {
		case diffmatchpatch.DiffEqual:
			lb.WriteString(d.Text)
			rb.WriteString(d.Text)
		case diffmatchpatch.DiffDelete:
			lb.WriteString(spanStart)
			lb.WriteString(d.Text)
			lb.WriteString(spanEnd)
		case diffmatchpatch.DiffInsert:
			rb.WriteString(spanStart)
			rb.WriteString(d.Text)
			rb.WriteString(spanEnd)
		}
	}

	// Apply line-level Modified background over span-highlighted content.
	left = bgModified + lb.String() + ansiReset
	right = bgModified + rb.String() + ansiReset
	return left, right
}

// joinColumns assembles the final rendered row from the two column strings.
// CRITICAL: ansiReset is inserted at the column boundary to prevent chroma
// foreground colour in the left column from bleeding into the right column
// (RESEARCH Pitfall 1, Threat T-01-08).
func joinColumns(left, right string) string {
	return left + ansiReset + " " + right
}

// lineBg returns the ANSI background colour sequence for the given LineKind.
// Returns empty string for context and blank lines (no background change).
func lineBg(kind LineKind) string {
	switch kind {
	case LineAdded:
		return bgAdded
	case LineRemoved:
		return bgRemoved
	case LineModifiedOld, LineModifiedNew:
		return bgModified
	default:
		return ""
	}
}

// shouldSkipIntraLine returns true when either line is too long (> 1000 chars)
// or has too many whitespace-delimited tokens (> 200), short-circuiting the
// computationally expensive DiffMain call (D-07, T-01-06).
func shouldSkipIntraLine(oldLine, newLine string) bool {
	return len(oldLine) > 1000 || len(newLine) > 1000 ||
		countTokens(oldLine) > 200 || countTokens(newLine) > 200
}

// countTokens counts whitespace-delimited fields in s as a fast proxy for
// lexical token count. Used by shouldSkipIntraLine to guard DiffMain (D-07).
func countTokens(s string) int {
	return len(strings.Fields(s))
}

// computeIntraLineWithTimeout runs dmp.DiffMain(old, new, false) on a goroutine
// and returns the result, or (nil, true) when DiffMain takes longer than 100ms.
// checklines is always false (D-06: character-level diff, not word-level).
func computeIntraLineWithTimeout(old, new string) ([]diffmatchpatch.Diff, bool) {
	type result struct{ diffs []diffmatchpatch.Diff }
	done := make(chan result, 1)
	go func() {
		done <- result{diffs: dmp.DiffMain(old, new, false)}
	}()
	select {
	case r := <-done:
		return r.diffs, false
	case <-time.After(100 * time.Millisecond):
		return nil, true // timed out — skip intra-line (D-07)
	}
}
