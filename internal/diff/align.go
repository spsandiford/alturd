package diff

import (
	"fmt"
	"os"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
)

// submoduleMode is the git file mode for submodule entries (octal 0160000).
const submoduleMode = os.FileMode(0160000)

// isSubmodule reports whether the file represents a git submodule.
// Detection is via OldMode or NewMode == 0160000 because go-gitdiff v0.8.1
// exposes no explicit IsSubmodule flag (RESEARCH Assumption A6).
func isSubmodule(f *gitdiff.File) bool {
	return f.OldMode == submoduleMode || f.NewMode == submoduleMode
}

// isModeOnly reports whether the file's diff is a pure mode change (e.g.
// chmod +x) with no textual content change. Such files have differing modes
// but zero TextFragments (RESEARCH Pitfall 4).
//
// The IsDelete and IsNew guards prevent a false-positive for empty deleted or
// newly-created files: an empty deleted file has OldMode set, NewMode=0, and
// zero TextFragments — identical to a mode-only change, but it is a deletion.
func isModeOnly(f *gitdiff.File) bool {
	return !f.IsDelete && !f.IsNew &&
		f.OldMode != f.NewMode && f.OldMode != 0 && len(f.TextFragments) == 0
}

// FileStatus returns a short bracketed status marker for a file suitable for
// display in the file-tree pane. Used by Plan 03 and Phase 3.
//
// Markers: [A]=added, [D]=deleted, [R]=renamed, [C]=copied, [B]=binary,
// [S]=submodule, [M]=modified or mode-only change.
func FileStatus(f *gitdiff.File) string {
	switch {
	case f.IsNew:
		return "[A]"
	case f.IsDelete:
		return "[D]"
	case f.IsRename:
		return "[R]"
	case f.IsCopy:
		return "[C]"
	case f.IsBinary:
		return "[B]"
	case isSubmodule(f):
		return "[S]"
	default:
		return "[M]"
	}
}

// stripNewline removes a trailing "\n" from s so that per-row Content holds
// one logical line without a newline suffix.
func stripNewline(s string) string {
	return strings.TrimSuffix(s, "\n")
}

// Align converts a parsed gitdiff.File into a []RowPair suitable for
// side-by-side rendering. It must never panic on nil or empty TextFragments.
//
// Edge-case handling (DIFF-07):
//   - Binary files return a single placeholder RowPair ("Binary file changed").
//   - Mode-only files return a single placeholder describing the mode change.
//   - Submodule files pass the raw Subproject-commit lines through as context
//     rows WITHOUT pairing them as Modified (Plan 03 skips intra-line diff on SHAs).
//
// Mode (DIFF-05):
//   - FullFile: all lines present in the diff output (context + changed) are included.
//   - HunkOnly: same as FullFile for Phase 1; in Phase 1 the diff already contains
//     only the hunk context, so there is no additional "full file" source to draw from.
//     The distinction is preserved here for Plan 03 to implement viewport toggling.
//
// Adjacent OpDelete + OpAdd sequences are paired as Modified RowPairs for intra-line
// diff processing. Multi-delete + multi-add runs are paired positionally (first delete
// → first add, etc.); leftovers are emitted as one-sided rows (RESEARCH Assumption A1).
func Align(file *gitdiff.File, mode RenderMode) []RowPair {
	// --- Early-exit edge cases (DIFF-07) ---

	// Binary files: do not attempt to render content; return a single placeholder.
	if file.IsBinary {
		var notice string
		if file.BinaryFragment != nil {
			notice = fmt.Sprintf("[Binary file changed — %d bytes]", file.BinaryFragment.Size)
		} else {
			notice = "[Binary file changed]"
		}
		return []RowPair{{
			Left:  RenderedLine{Kind: LineContext, Content: notice},
			Right: RenderedLine{Kind: LineBlank},
		}}
	}

	// Mode-only: pure permission change with no textual diff.
	if isModeOnly(file) {
		notice := fmt.Sprintf("[Mode changed: %04o → %04o]", file.OldMode, file.NewMode)
		return []RowPair{{
			Left:  RenderedLine{Kind: LineContext, Content: notice},
			Right: RenderedLine{Kind: LineBlank},
		}}
	}

	// Submodule: pass raw lines through as context rows.
	// Intra-line diff on 40-char SHA strings would be meaningless.
	if isSubmodule(file) {
		return alignSubmodule(file)
	}

	// --- Normal text diff ---
	return alignText(file, mode)
}

// alignSubmodule builds RowPairs for a submodule diff by emitting each line as
// a one-sided row: deletes on the left (LineRemoved + LineBlank), adds on the
// right (LineBlank + LineAdded), context on both sides. No Modified pairing.
func alignSubmodule(file *gitdiff.File) []RowPair {
	var result []RowPair
	for _, frag := range file.TextFragments {
		for _, l := range frag.Lines {
			content := stripNewline(l.Line)
			switch l.Op {
			case gitdiff.OpContext:
				result = append(result, RowPair{
					Left:  RenderedLine{Kind: LineContext, Content: content},
					Right: RenderedLine{Kind: LineContext, Content: content},
				})
			case gitdiff.OpDelete:
				result = append(result, RowPair{
					Left:  RenderedLine{Kind: LineRemoved, Content: content},
					Right: RenderedLine{Kind: LineBlank},
				})
			case gitdiff.OpAdd:
				result = append(result, RowPair{
					Left:  RenderedLine{Kind: LineBlank},
					Right: RenderedLine{Kind: LineAdded, Content: content},
				})
			}
		}
	}
	return result
}

// alignText is the main alignment algorithm for normal text diffs. It walks
// TextFragments, collecting consecutive delete+add runs and pairing them
// positionally into Modified RowPairs. Leftover deletes or adds become
// one-sided rows. Context lines appear on both sides.
//
// In HunkOnly mode, only the lines that are part of changed hunks are included
// (context lines between hunks that are NOT part of a hunk are omitted). In
// Phase 1 this is equivalent to FullFile because the diff output already
// contains only hunk-local context; the distinction matters in Phase 3 when
// the original file is available. The mode parameter is wired here for forward
// compatibility.
func alignText(file *gitdiff.File, mode RenderMode) []RowPair {
	var result []RowPair
	for _, frag := range file.TextFragments {
		lines := frag.Lines
		i := 0
		for i < len(lines) {
			switch lines[i].Op {
			case gitdiff.OpContext:
				content := stripNewline(lines[i].Line)
				// In HunkOnly mode we still include hunk-local context lines.
				if mode == HunkOnly || mode == FullFile {
					result = append(result, RowPair{
						Left:  RenderedLine{Kind: LineContext, Content: content},
						Right: RenderedLine{Kind: LineContext, Content: content},
					})
				}
				i++

			case gitdiff.OpDelete:
				// Collect all consecutive deletes.
				var deletes []string
				for i < len(lines) && lines[i].Op == gitdiff.OpDelete {
					deletes = append(deletes, stripNewline(lines[i].Line))
					i++
				}
				// Collect all consecutive adds immediately following the deletes.
				var adds []string
				for i < len(lines) && lines[i].Op == gitdiff.OpAdd {
					adds = append(adds, stripNewline(lines[i].Line))
					i++
				}
				// Pair positionally.
				n := len(deletes)
				if len(adds) < n {
					n = len(adds)
				}
				for j := 0; j < n; j++ {
					result = append(result, RowPair{
						Left:  RenderedLine{Kind: LineModifiedOld, Content: deletes[j]},
						Right: RenderedLine{Kind: LineModifiedNew, Content: adds[j]},
					})
				}
				// Leftover deletes (more deletes than adds).
				for j := n; j < len(deletes); j++ {
					result = append(result, RowPair{
						Left:  RenderedLine{Kind: LineRemoved, Content: deletes[j]},
						Right: RenderedLine{Kind: LineBlank},
					})
				}
				// Leftover adds (more adds than deletes).
				for j := n; j < len(adds); j++ {
					result = append(result, RowPair{
						Left:  RenderedLine{Kind: LineBlank},
						Right: RenderedLine{Kind: LineAdded, Content: adds[j]},
					})
				}

			case gitdiff.OpAdd:
				// Standalone add (not preceded by deletes in this run).
				result = append(result, RowPair{
					Left:  RenderedLine{Kind: LineBlank},
					Right: RenderedLine{Kind: LineAdded, Content: stripNewline(lines[i].Line)},
				})
				i++
			}
		}
	}
	return result
}
