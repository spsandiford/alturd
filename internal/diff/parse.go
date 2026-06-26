package diff

import (
	"fmt"
	"io"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
)

// Parse wraps gitdiff.Parse, returning typed File structs.
// It never panics — malformed input surfaces as a returned error (ASVS V5).
// Phase 2 (internal/git) calls this with an io.Reader from a git subprocess.
//
// The preamble (non-diff text before the first diff --git header) is discarded.
// On error, Parse returns nil and a wrapped error matchable via errors.Is / %w.
func Parse(r io.Reader) ([]*gitdiff.File, error) {
	files, _, err := gitdiff.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parsing diff: %w", err)
	}
	return files, nil
}
