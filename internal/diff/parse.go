package diff

import (
	"fmt"
	"io"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
)

// Parse wraps gitdiff.Parse, returning typed File structs.
// It never panics — malformed input surfaces as a returned error.
// Phase 2 (internal/git) calls this with an io.Reader from a git subprocess.
func Parse(r io.Reader) ([]*gitdiff.File, error) {
	// STUB: will be fully implemented in GREEN step
	_ = r
	return nil, fmt.Errorf("parse: not implemented")
}
