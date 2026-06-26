package diff

import (
	"github.com/bluekeyes/go-gitdiff/gitdiff"
)

// Align converts a parsed gitdiff.File into []RowPair for side-by-side rendering.
// STUB: full implementation pending.
func Align(file *gitdiff.File, mode RenderMode) []RowPair {
	return nil
}

// FileStatus returns a short bracketed status marker for a file.
// STUB: full implementation pending.
func FileStatus(f *gitdiff.File) string {
	return "[M]"
}
