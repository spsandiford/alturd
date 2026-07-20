package tui

import (
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"github.com/alturd/alturd/internal/diff"
)

// newModelWith creates a model from files and simulates a WindowSizeMsg so ready=true.
func newModelWith(t *testing.T, files []*gitdiff.File) model {
	t.Helper()
	m := NewModel(files)
	m.handleResize(200, 50)
	return m
}

// parseAllFixture parses a multi-file diff fixture from internal/diff/testdata/.
func parseAllFixture(t *testing.T, fixture string) []*gitdiff.File {
	t.Helper()
	f, err := os.Open("../diff/testdata/" + fixture)
	if err != nil {
		t.Fatalf("parseAllFixture(%q): %v", fixture, err)
	}
	defer f.Close()
	files, err := diff.Parse(f)
	if err != nil {
		t.Fatalf("parseAllFixture(%q): Parse: %v", fixture, err)
	}
	return files
}

func TestModel_NotReady(t *testing.T) {
	m := NewModel(nil)
	v := m.View()
	if v.Content != "" {
		t.Errorf("View() before WindowSizeMsg: got %q, want empty string (D-07)", v.Content)
	}
}

func TestModel_Quit(t *testing.T) {
	m := newModelWith(t, nil)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	if cmd == nil {
		t.Error("'q' key: expected non-nil Cmd (tea.Quit), got nil")
	}
}

func TestModel_FocusToggle(t *testing.T) {
	m := newModelWith(t, nil)
	// Initial state: diffFocused, treeWidth=24.
	if m.treeWidth != treeWidthUnfocused {
		t.Fatalf("initial treeWidth = %d, want %d", m.treeWidth, treeWidthUnfocused)
	}

	// Tab → treeFocused, treeWidth=45.
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = m2.(model)
	if m.focusedPane != treeFocused {
		t.Errorf("after Tab: focusedPane = %v, want treeFocused", m.focusedPane)
	}
	if m.treeWidth != treeWidthFocused {
		t.Errorf("after Tab: treeWidth = %d, want %d", m.treeWidth, treeWidthFocused)
	}

	// Tab again → diffFocused, treeWidth=24.
	m3, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = m3.(model)
	if m.focusedPane != diffFocused {
		t.Errorf("after 2nd Tab: focusedPane = %v, want diffFocused", m.focusedPane)
	}
	if m.treeWidth != treeWidthUnfocused {
		t.Errorf("after 2nd Tab: treeWidth = %d, want %d", m.treeWidth, treeWidthUnfocused)
	}
}

func TestModel_FileCycle(t *testing.T) {
	files := parseAllFixture(t, "multi-file.diff")
	if len(files) < 2 {
		t.Skip("multi-file.diff has fewer than 2 files")
	}
	m := newModelWith(t, files)
	if m.currentFile != 0 {
		t.Fatalf("initial currentFile = %d, want 0", m.currentFile)
	}

	// ']' → advance to file 1.
	m2, _ := m.Update(tea.KeyPressMsg{Code: ']'})
	m = m2.(model)
	if m.currentFile != 1 {
		t.Errorf("after ']': currentFile = %d, want 1", m.currentFile)
	}

	// '[' from file 0 → wraparound to last.
	m.currentFile = 0
	m.refreshDiffContent()
	m3, _ := m.Update(tea.KeyPressMsg{Code: '['})
	m = m3.(model)
	if m.currentFile != len(files)-1 {
		t.Errorf("'[' from 0: currentFile = %d, want %d (last)", m.currentFile, len(files)-1)
	}
}

func TestModel_HunkNav(t *testing.T) {
	files := parseAllFixture(t, "multi-hunk.diff")
	if len(files) == 0 {
		t.Fatal("multi-hunk.diff has no files")
	}
	m := newModelWith(t, files)
	if len(m.hunkRows) < 2 {
		t.Skip("multi-hunk.diff has fewer than 2 hunks")
	}

	initialHunk := m.currentHunk
	m2, _ := m.Update(tea.KeyPressMsg{Code: 'n'})
	m = m2.(model)
	if m.currentHunk != initialHunk+1 {
		t.Errorf("after 'n': currentHunk = %d, want %d", m.currentHunk, initialHunk+1)
	}
	wantOffset := m.hunkRows[m.currentHunk] - m.diffVP.Height()/2
	if wantOffset < 0 {
		wantOffset = 0
	}
	if m.diffVP.YOffset() != wantOffset {
		t.Errorf("YOffset = %d, want %d (hunk centering)", m.diffVP.YOffset(), wantOffset)
	}
}

func TestModel_ModeToggle(t *testing.T) {
	files := parseAllFixture(t, "multi-hunk.diff")
	m := newModelWith(t, files)
	if m.renderMode != diff.FullFile {
		t.Fatalf("initial renderMode = %v, want FullFile", m.renderMode)
	}

	m2, _ := m.Update(tea.KeyPressMsg{Code: 'v'})
	m = m2.(model)
	if m.renderMode != diff.HunkOnly {
		t.Errorf("after 'v': renderMode = %v, want HunkOnly", m.renderMode)
	}

	m3, _ := m.Update(tea.KeyPressMsg{Code: 'v'})
	m = m3.(model)
	if m.renderMode != diff.FullFile {
		t.Errorf("after 2nd 'v': renderMode = %v, want FullFile", m.renderMode)
	}
}
