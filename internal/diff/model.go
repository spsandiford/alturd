// Package diff provides utilities for parsing git diff output and rendering
// side-by-side ANSI output with syntax highlighting and intra-line change markers.
//
// Pipeline: Parse → Align → Highlight → Render
//
//   - Parse: converts raw git diff output into typed []*gitdiff.File structs
//   - Align: groups adjacent delete+add lines into side-by-side RowPairs
//   - Highlight: populates the ANSI field of each RenderedLine via Chroma
//   - Render: composes the final []string rows for display in a terminal viewport
package diff

// LineKind classifies a single line within the side-by-side diff view,
// indicating how it should be colored and whether it participates in
// intra-line character-level diff processing.
type LineKind int

const (
	// LineContext is an unchanged context line that appears on both sides.
	LineContext LineKind = iota

	// LineAdded is a line present only on the new (right) side.
	LineAdded

	// LineRemoved is a line present only on the old (left) side.
	LineRemoved

	// LineModifiedOld is the old version of a changed line, paired with
	// the corresponding LineModifiedNew on the right side.
	// Intra-line character diff is applied to Modified pairs when guards permit.
	LineModifiedOld

	// LineModifiedNew is the new version of a changed line, paired with
	// the corresponding LineModifiedOld on the left side.
	LineModifiedNew

	// LineBlank is an alignment filler inserted on the side that has no
	// corresponding line. Used when an added or removed line has no pair.
	LineBlank
)

// RenderMode controls how much content is included in the rendered output.
type RenderMode int

const (
	// FullFile renders all lines in the diff output including unchanged context
	// lines, giving a complete view of the file with changes inline.
	// This is the default mode (iota 0) per DIFF-05.
	FullFile RenderMode = iota

	// HunkOnly renders only the changed hunks, omitting unchanged context lines
	// between hunks. Used as an alternate view toggled by the 'v' key (DIFF-06).
	HunkOnly
)

// RenderedLine represents a single line on one side (left or right) of the
// side-by-side diff view. It carries both the raw text content and the
// pre-computed ANSI-highlighted output.
//
// Content is populated by Align; ANSI is populated by Highlight and Render.
type RenderedLine struct {
	// Kind classifies this line for coloring and intra-line diff eligibility.
	Kind LineKind

	// Content is the raw text of the line before syntax highlighting or
	// diff coloring is applied.
	Content string

	// ANSI is the fully-composed terminal escape sequence string for this line,
	// including syntax highlighting foreground color and diff background color.
	// It is populated by highlight.go and render.go; it is empty after Align.
	ANSI string
}

// RowPair represents one row in the side-by-side diff view, pairing the old
// (left) and new (right) renderings of the same position in the file.
//
// RowPairs are produced by Align and consumed by Render. Both sides are always
// present: when a line exists on only one side, the other side is a LineBlank.
type RowPair struct {
	// Left holds the old-file content for this row position.
	Left RenderedLine

	// Right holds the new-file content for this row position.
	Right RenderedLine
}
