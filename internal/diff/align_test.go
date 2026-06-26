package diff_test

import (
	"os"
	"strings"
	"testing"

	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"github.com/alturd/alturd/internal/diff"
)

// parseFirst opens a fixture, calls diff.Parse, and returns the first file.
// It fails the test if anything goes wrong.
func parseFirst(t *testing.T, fixture string) *gitdiff.File {
	t.Helper()
	f, err := os.Open("testdata/" + fixture)
	if err != nil {
		t.Fatalf("parseFirst(%q): %v", fixture, err)
	}
	defer f.Close()
	files, err := diff.Parse(f)
	if err != nil {
		t.Fatalf("parseFirst(%q): Parse: %v", fixture, err)
	}
	if len(files) == 0 {
		t.Fatalf("parseFirst(%q): no files returned", fixture)
	}
	return files[0]
}

func TestAlign(t *testing.T) {
	t.Run("simple_modified_pair", func(t *testing.T) {
		// simple.diff has one delete/add pair; Align must produce at least one
		// Modified RowPair (LineModifiedOld on left, LineModifiedNew on right).
		file := parseFirst(t, "simple.diff")
		rows := diff.Align(file, diff.FullFile)

		if len(rows) == 0 {
			t.Fatal("Align(simple.diff, FullFile): got 0 rows, want > 0")
		}
		found := false
		for _, row := range rows {
			if row.Left.Kind == diff.LineModifiedOld && row.Right.Kind == diff.LineModifiedNew {
				found = true
				break
			}
		}
		if !found {
			t.Error("Align(simple.diff, FullFile): no LineModifiedOld+LineModifiedNew pair found")
		}
	})

	t.Run("lone_delete_gets_blank_right", func(t *testing.T) {
		// deleted-file.diff has only OpDelete lines; each must yield
		// {Left: LineRemoved, Right: LineBlank}.
		file := parseFirst(t, "deleted-file.diff")
		rows := diff.Align(file, diff.FullFile)

		if len(rows) == 0 {
			t.Fatal("Align(deleted-file.diff): got 0 rows")
		}
		for i, row := range rows {
			if row.Left.Kind != diff.LineRemoved {
				t.Errorf("row[%d].Left.Kind = %v, want LineRemoved", i, row.Left.Kind)
			}
			if row.Right.Kind != diff.LineBlank {
				t.Errorf("row[%d].Right.Kind = %v, want LineBlank", i, row.Right.Kind)
			}
		}
	})

	t.Run("lone_add_gets_blank_left", func(t *testing.T) {
		// new-file.diff has only OpAdd lines; each must yield
		// {Left: LineBlank, Right: LineAdded}.
		file := parseFirst(t, "new-file.diff")
		rows := diff.Align(file, diff.FullFile)

		if len(rows) == 0 {
			t.Fatal("Align(new-file.diff): got 0 rows")
		}
		for i, row := range rows {
			if row.Left.Kind != diff.LineBlank {
				t.Errorf("row[%d].Left.Kind = %v, want LineBlank", i, row.Left.Kind)
			}
			if row.Right.Kind != diff.LineAdded {
				t.Errorf("row[%d].Right.Kind = %v, want LineAdded", i, row.Right.Kind)
			}
		}
	})

	t.Run("multi_delete_add_positional_pairing", func(t *testing.T) {
		// multi-hunk.diff has runs of 2+ delete then 2+ add lines.
		// First delete pairs with first add (Modified), leftovers are one-sided.
		file := parseFirst(t, "multi-hunk.diff")
		rows := diff.Align(file, diff.FullFile)

		if len(rows) == 0 {
			t.Fatal("Align(multi-hunk.diff): got 0 rows")
		}
		// Must have at least one Modified pair.
		modifiedCount := 0
		standaloneAddCount := 0
		for _, row := range rows {
			if row.Left.Kind == diff.LineModifiedOld && row.Right.Kind == diff.LineModifiedNew {
				modifiedCount++
			}
			if row.Left.Kind == diff.LineBlank && row.Right.Kind == diff.LineAdded {
				standaloneAddCount++
			}
		}
		if modifiedCount == 0 {
			t.Error("Align(multi-hunk.diff): no Modified pairs found")
		}
		// The multi-hunk fixture has leftover adds (more adds than deletes in each block).
		if standaloneAddCount == 0 {
			t.Error("Align(multi-hunk.diff): no standalone-add rows found (expected leftover adds from multi-add blocks)")
		}
	})

	t.Run("binary_placeholder", func(t *testing.T) {
		// binary.diff must return exactly 1 placeholder RowPair; no line-diff attempted.
		file := parseFirst(t, "binary.diff")
		rows := diff.Align(file, diff.FullFile)

		if len(rows) != 1 {
			t.Fatalf("Align(binary.diff): got %d rows, want exactly 1", len(rows))
		}
		// Placeholder must contain "[Binary" somewhere in the content.
		content := rows[0].Left.Content + rows[0].Right.Content
		if !strings.Contains(content, "[Binary") {
			t.Errorf("Align(binary.diff): placeholder content %q does not contain '[Binary'", content)
		}
		// Must not contain LineModified kinds.
		if rows[0].Left.Kind == diff.LineModifiedOld || rows[0].Right.Kind == diff.LineModifiedNew {
			t.Error("Align(binary.diff): got Modified kinds in binary placeholder")
		}
	})

	t.Run("mode_only_placeholder", func(t *testing.T) {
		// mode-only.diff (chmod +x, no text) must return exactly 1 placeholder
		// describing the mode change.
		file := parseFirst(t, "mode-only.diff")
		rows := diff.Align(file, diff.FullFile)

		if len(rows) != 1 {
			t.Fatalf("Align(mode-only.diff): got %d rows, want exactly 1", len(rows))
		}
		content := rows[0].Left.Content + rows[0].Right.Content
		if !strings.Contains(content, "Mode") && !strings.Contains(content, "mode") {
			t.Errorf("Align(mode-only.diff): placeholder %q does not mention mode", content)
		}
	})

	t.Run("submodule_raw_context_no_modified", func(t *testing.T) {
		// submodule.diff must pass lines through as context rows without pairing
		// the Subproject-commit SHA lines as LineModifiedOld/New.
		file := parseFirst(t, "submodule.diff")
		rows := diff.Align(file, diff.FullFile)

		if len(rows) == 0 {
			t.Fatal("Align(submodule.diff): got 0 rows")
		}
		for i, row := range rows {
			if row.Left.Kind == diff.LineModifiedOld || row.Right.Kind == diff.LineModifiedNew {
				t.Errorf("row[%d]: submodule lines must not produce Modified pairs (got Left=%v Right=%v)",
					i, row.Left.Kind, row.Right.Kind)
			}
		}
	})

	t.Run("hunkonly_le_fullfile", func(t *testing.T) {
		// For multi-hunk.diff, HunkOnly row count must be <= FullFile row count.
		file := parseFirst(t, "multi-hunk.diff")
		fullRows := diff.Align(file, diff.FullFile)
		hunkRows := diff.Align(file, diff.HunkOnly)

		if len(hunkRows) > len(fullRows) {
			t.Errorf("Align HunkOnly (%d rows) > FullFile (%d rows); want HunkOnly <= FullFile",
				len(hunkRows), len(fullRows))
		}
	})

	t.Run("no_panic_empty_fragments", func(t *testing.T) {
		// rename.diff has no TextFragments; Align must return without panicking.
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Align(rename.diff) panicked: %v", r)
			}
		}()
		file := parseFirst(t, "rename.diff")
		rows := diff.Align(file, diff.FullFile)
		// A pure rename with no content produces zero rows (or a placeholder).
		_ = rows
	})

	t.Run("context_lines_populated", func(t *testing.T) {
		// In FullFile mode, context lines must appear with LineContext on BOTH sides.
		// Blank context lines may have empty Content (they represent blank lines in the file).
		// At least one non-blank context line must have non-empty Content.
		file := parseFirst(t, "simple.diff")
		rows := diff.Align(file, diff.FullFile)

		foundContext := false
		foundNonBlankContent := false
		for _, row := range rows {
			if row.Left.Kind == diff.LineContext {
				foundContext = true
				if row.Right.Kind != diff.LineContext {
					t.Errorf("LineContext row: Left=LineContext but Right=%v", row.Right.Kind)
				}
				if row.Left.Content != "" {
					foundNonBlankContent = true
				}
			}
		}
		if !foundContext {
			t.Error("Align(simple.diff, FullFile): no LineContext rows found")
		}
		if !foundNonBlankContent {
			t.Error("Align(simple.diff, FullFile): all context rows have empty Content")
		}
	})

	t.Run("filestatus_helper", func(t *testing.T) {
		// FileStatus is exercised by Align internally; verify via observable behaviour:
		// a new-file fixture should produce only LineAdded/LineBlank rows (status [A]).
		file := parseFirst(t, "new-file.diff")
		rows := diff.Align(file, diff.FullFile)
		for i, row := range rows {
			if row.Right.Kind != diff.LineAdded {
				t.Errorf("new-file row[%d].Right.Kind = %v, want LineAdded", i, row.Right.Kind)
			}
		}
	})
}
