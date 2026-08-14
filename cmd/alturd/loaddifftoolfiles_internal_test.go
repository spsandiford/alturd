// Package main (internal test file) guards WR-07: loadDifftoolFiles must
// source the reference-version content for full-file rendering from
// --difftool-local, not --difftool-remote, when the file is a deletion —
// honouring AlignFull's documented old-file-for-deletions contract
// (internal/diff/align.go:254-261, "fileLines contains the full content of
// the reference version of the file ... old file for deleted files"). A
// dedicated file (not main_internal_test.go, scoped to reportError/diffArgs,
// nor difftooldiff_internal_test.go, scoped to the external-diff recursion
// guard) so this regression test stays surgically separate from the
// existing internal-test suite.
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/alturd/alturd/internal/diff"
)

// TestLoadDifftoolFilesDeletedFileUsesLocalSide asserts that a deleted
// file's reference content is read from --difftool-local (the pre-image),
// not --difftool-remote (git's /dev/null for a deletion), and that the
// content survives all the way through diff.RenderFull — the exact call
// refreshDiffContent makes. A sensitivity-control subtest asserts the
// unmodified case (a modification) still reads from --difftool-remote, so
// inverting the IsDelete branch — not just deleting it — is also caught.
func TestLoadDifftoolFilesDeletedFileUsesLocalSide(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH — skipping")
	}

	// Isolate gitconfig: a developer's real gitconfig must not be able to
	// turn this test green or red by accident (mirrors
	// difftooldiff_internal_test.go).
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)

	// Deterministic counter-less title-bar template regardless of the
	// environment this test runs in.
	t.Setenv("GIT_DIFF_PATH_COUNTER", "")
	t.Setenv("GIT_DIFF_PATH_TOTAL", "")

	t.Run("deleted_file_uses_local", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("needs git's literal /dev/null semantics to produce a deletion; an empty temp file parses as a modification on Windows")
		}

		dir := t.TempDir()
		localPath := filepath.Join(dir, "local.txt")
		wantLines := []string{"line one", "line two", "line three"}
		if err := os.WriteFile(localPath, []byte("line one\nline two\nline three\n"), 0o644); err != nil {
			t.Fatalf("write local file: %v", err)
		}
		mergedPath := filepath.Join(t.TempDir(), "deleted.txt")

		files, dt, err := loadDifftoolFiles(localPath, os.DevNull, mergedPath)
		if err != nil {
			t.Fatalf("loadDifftoolFiles() error = %v, want nil", err)
		}
		if len(files) != 1 {
			t.Fatalf("loadDifftoolFiles() returned %d files, want 1", len(files))
		}
		if !files[0].IsDelete {
			t.Fatalf("files[0].IsDelete = false, want true")
		}
		if files[0].NewName != "deleted.txt" {
			t.Errorf("files[0].NewName = %q, want %q", files[0].NewName, "deleted.txt")
		}
		if !slices.Equal(dt.RefFileLines, wantLines) {
			t.Fatalf("dt.RefFileLines = %v, want %v (empty RefFileLines here is the WR-07 regression: loadDifftoolFiles read --difftool-remote instead of --difftool-local for a deleted file)", dt.RefFileLines, wantLines)
		}

		// Close the end-to-end loop: prove the content survives the exact
		// call refreshDiffContent makes.
		rows := diff.RenderFull(files[0], 120, dt.RefFileLines)
		joined := ""
		for _, row := range rows {
			joined += row + "\n"
		}
		for _, want := range wantLines {
			if !strings.Contains(joined, want) {
				t.Errorf("rendered output missing %q; got:\n%s", want, joined)
			}
		}
	})

	t.Run("modified_file_uses_remote", func(t *testing.T) {
		dir := t.TempDir()
		localPath := filepath.Join(dir, "local.txt")
		remotePath := filepath.Join(dir, "remote.txt")
		if err := os.WriteFile(localPath, []byte("old line one\nold line two\n"), 0o644); err != nil {
			t.Fatalf("write local file: %v", err)
		}
		wantLines := []string{"new line one", "new line two"}
		if err := os.WriteFile(remotePath, []byte("new line one\nnew line two\n"), 0o644); err != nil {
			t.Fatalf("write remote file: %v", err)
		}
		mergedPath := filepath.Join(t.TempDir(), "modified.txt")

		files, dt, err := loadDifftoolFiles(localPath, remotePath, mergedPath)
		if err != nil {
			t.Fatalf("loadDifftoolFiles() error = %v, want nil", err)
		}
		if len(files) != 1 {
			t.Fatalf("loadDifftoolFiles() returned %d files, want 1", len(files))
		}
		if files[0].IsDelete {
			t.Fatalf("files[0].IsDelete = true, want false")
		}
		if !slices.Equal(dt.RefFileLines, wantLines) {
			t.Fatalf("dt.RefFileLines = %v, want %v (the remote/post-image lines)", dt.RefFileLines, wantLines)
		}
	})
}
