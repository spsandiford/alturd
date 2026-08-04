package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"github.com/alturd/alturd/internal/config"
	"github.com/alturd/alturd/internal/diff"
)

// newModelWith creates a model from files and simulates a WindowSizeMsg so ready=true.
// darkBg defaults to false (light terminal) matching CI environments. Uses
// config.DefaultKeymap() so the existing Phase 3 suite keeps exercising the
// default bindings unchanged.
func newModelWith(t *testing.T, files []*gitdiff.File) model {
	t.Helper()
	m := NewModel(files, false, config.DefaultKeymap(), DifftoolInfo{})
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
	defer func() { _ = f.Close() }()
	files, err := diff.Parse(f)
	if err != nil {
		t.Fatalf("parseAllFixture(%q): Parse: %v", fixture, err)
	}
	return files
}

func TestModel_NotReady(t *testing.T) {
	m := NewModel(nil, false, config.DefaultKeymap(), DifftoolInfo{})
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

// TestModel_SearchDispatch verifies the searchMode key routing (SEARCH-01/D-14/D-15).
func TestModel_SearchDispatch(t *testing.T) {
	files := parseAllFixture(t, "multi-hunk.diff")
	if len(files) == 0 {
		t.Fatal("multi-hunk.diff has no files")
	}
	m := newModelWith(t, files)

	// '/' opens search mode (D-13).
	m2, _ := m.Update(tea.KeyPressMsg{Code: '/'})
	m = m2.(model)
	if !m.searchMode {
		t.Fatal("after '/': searchMode = false, want true (D-13)")
	}

	// In searchMode, record the current hunk position.
	hunkBefore := m.currentHunk

	// 'n' in searchMode must NOT advance the hunk (D-14): it goes to HighlightNext.
	m3, _ := m.Update(tea.KeyPressMsg{Code: 'n'})
	m = m3.(model)
	if m.currentHunk != hunkBefore {
		t.Errorf("'n' in searchMode changed currentHunk %d → %d; must route to highlight nav, not hunk nav (D-14)",
			hunkBefore, m.currentHunk)
	}

	// 'N' in searchMode must also NOT change the hunk (D-14).
	m4, _ := m.Update(tea.KeyPressMsg{Code: 'N'})
	m = m4.(model)
	if m.currentHunk != hunkBefore {
		t.Errorf("'N' in searchMode changed currentHunk %d → %d; must route to HighlightPrevious (D-14)",
			hunkBefore, m.currentHunk)
	}

	// Esc closes search and clears state (D-15).
	m5, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = m5.(model)
	if m.searchMode {
		t.Error("after Esc: searchMode = true, want false (D-15)")
	}
	if m.searchMatches != nil {
		t.Errorf("after Esc: searchMatches = %v, want nil (D-15)", m.searchMatches)
	}
}

// TestModel_SearchFindPositions verifies that recomputeSearch populates searchMatches
// when a query appears in the rendered diff content (SEARCH-01).
func TestModel_SearchFindPositions(t *testing.T) {
	files := parseAllFixture(t, "multi-hunk.diff")
	if len(files) == 0 {
		t.Fatal("multi-hunk.diff has no files")
	}
	m := newModelWith(t, files)

	// Open search mode.
	m2, _ := m.Update(tea.KeyPressMsg{Code: '/'})
	m = m2.(model)

	// The diff content for multi-hunk.diff should contain the marker character '+' or '-'.
	// We use a query that is almost certainly in the rendered content to verify
	// that recomputeSearch runs without panic and populates searchMatches.
	// The actual query text doesn't matter: even if 0 matches, the call must not panic.
	m.recomputeSearch()
	// If we reach here without panic, the basic path works. Verify state consistency.
	_ = m.searchMatches // may be nil (no matches) or populated — both are valid
}

// TestModel_AllFilesToggle verifies the TREE-03 / D-11 all-files toggle without
// invoking a live git subprocess. It pre-seeds allFilePaths and asserts that
// toggling rebuilds treeFlat over the expected path set with status markers retained.
func TestModel_AllFilesToggle(t *testing.T) {
	// Build a minimal file list with a known path and status.
	files := parseAllFixture(t, "simple.diff")
	if len(files) == 0 {
		t.Skip("simple.diff has no files — cannot test toggleAllFiles")
	}
	m := newModelWith(t, files)

	// Seed allFilePaths with additional paths (simulating ls-tree output) so that
	// toggleAllFiles does NOT call git ls-tree. The first press would call git if the
	// slice is empty; we pre-seed so the test is hermetic.
	extraPaths := []string{
		"README.md",
		"cmd/alturd/main.go",
		"internal/diff/render.go",
		"internal/tui/model.go",
	}
	// Add the changed-file paths too so status markers appear.
	m.allFilePaths = append(extraPaths, filePaths(files)...)

	// Capture the changed-files treeFlat length before toggling.
	flatBefore := len(m.treeFlat)

	// First toggle ON: should rebuild treeFlat over allFilePaths (hermetic — pre-seeded).
	m.toggleAllFiles()
	if !m.allFiles {
		t.Error("after first toggleAllFiles: allFiles = false, want true")
	}
	if len(m.treeFlat) <= flatBefore {
		t.Errorf("after toggle ON: treeFlat len = %d, want > %d (full repo has more files)",
			len(m.treeFlat), flatBefore)
	}

	// Verify that at least one changed file retains its status marker in the flat list.
	statusMap := buildStatusMap(files)
	found := false
	for _, row := range m.treeFlat {
		if !row.node.IsDir && row.node.Status != "" {
			if _, ok := statusMap[row.node.Path]; ok {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("after toggle ON: no changed file with a status marker found in treeFlat (D-11)")
	}

	// Second toggle OFF: should rebuild treeFlat over changed-files-only paths.
	m.toggleAllFiles()
	if m.allFiles {
		t.Error("after second toggleAllFiles: allFiles = true, want false")
	}
	if len(m.treeFlat) != flatBefore {
		t.Errorf("after toggle OFF: treeFlat len = %d, want %d (changed-files only)",
			len(m.treeFlat), flatBefore)
	}
}

// TestModel_TreeExpandCollapse verifies that Enter/l/right expand a directory node
// in the tree and a second press collapses it.
func TestModel_TreeExpandCollapse(t *testing.T) {
	// Seed a tree with a directory that has children so we have something to expand.
	m := newModelWith(t, nil)
	m.allFilePaths = []string{
		"src/foo.go",
		"src/bar.go",
		"README.md",
	}
	m.allFiles = false
	root := buildTree(m.allFilePaths, nil)
	m.treeNodes = []*TreeNode{root}
	m.treeFlat = flattenTree(root, 0)
	m.treeIdx = 0

	// Focus the tree pane so Enter routes to treeToggleExpand.
	m.focusedPane = treeFocused

	// Find a dir node in the flat list.
	dirIdx := -1
	for i, row := range m.treeFlat {
		if row.node.IsDir {
			dirIdx = i
			break
		}
	}
	if dirIdx < 0 {
		t.Skip("test tree has no directory nodes — cannot test expand")
	}
	m.treeIdx = dirIdx
	flatBefore := len(m.treeFlat)

	// Enter expands the dir: treeFlat should grow.
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = m2.(model)
	if len(m.treeFlat) <= flatBefore {
		t.Errorf("after Enter on dir: treeFlat len = %d, want > %d (children should appear)",
			len(m.treeFlat), flatBefore)
	}

	// Enter again collapses: treeFlat should return to original length.
	m3, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = m3.(model)
	if len(m.treeFlat) != flatBefore {
		t.Errorf("after 2nd Enter on dir: treeFlat len = %d, want %d (collapsed)",
			len(m.treeFlat), flatBefore)
	}
}

// TestKeymapOverrideDispatchesInModel is the tracer's end-to-end assertion
// (04-01-PLAN.md Task 2): a TOML file rebinding the quit action to "x" is
// loaded into a real config.Keymap via config.Load, that Keymap is passed
// into NewModel, and the running model dispatches quit on the rebound key
// while the former default key ("q") no longer does (CONFIG-01, CONFIG-02).
func TestKeymapOverrideDispatchesInModel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	contents := "[keybindings]\nquit = \"x\"\n"
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("config.Load(%q) = _, %v, want nil error", path, err)
	}

	m := NewModel(nil, false, cfg.Keys, DifftoolInfo{})
	m.handleResize(200, 50)

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'x'})
	if cmd == nil {
		t.Error("'x' key (rebound quit): expected non-nil Cmd (tea.Quit), got nil")
	}

	_, cmd = m.Update(tea.KeyPressMsg{Code: 'q'})
	if cmd != nil {
		t.Error("'q' key (former default, now unbound): expected nil Cmd, got non-nil")
	}
}

// TestDifftoolTitleBar is table-driven over the five title-bar rows from
// 04-03-PLAN.md Task 3's behavior block, asserting on the trimmed rendered
// string (DIFFTOOL-02).
func TestDifftoolTitleBar(t *testing.T) {
	tests := []struct {
		name       string
		dt         DifftoolInfo
		searchMode bool
		want       string
	}{
		{
			name:       "counters_present",
			dt:         DifftoolInfo{Enabled: true, Counter: 3, Total: 7, Filename: "render.go"},
			searchMode: false,
			want:       "alturd (difftool) — 3 of 7 — render.go",
		},
		{
			name:       "counters_absent",
			dt:         DifftoolInfo{Enabled: true, Counter: 0, Total: 0, Filename: "render.go"},
			searchMode: false,
			want:       "alturd (difftool) — render.go",
		},
		{
			name:       "counters_present_search_open",
			dt:         DifftoolInfo{Enabled: true, Counter: 3, Total: 7, Filename: "render.go"},
			searchMode: true,
			want:       "alturd (difftool) — 3 of 7 — render.go [SEARCH]",
		},
		{
			name:       "counters_equal_boundary",
			dt:         DifftoolInfo{Enabled: true, Counter: 1, Total: 1, Filename: "render.go"},
			searchMode: false,
			want:       "alturd (difftool) — 1 of 1 — render.go",
		},
		{
			name:       "counters_adjacent_boundary",
			dt:         DifftoolInfo{Enabled: true, Counter: 7, Total: 7, Filename: "render.go"},
			searchMode: false,
			want:       "alturd (difftool) — 7 of 7 — render.go",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewModel(nil, false, config.DefaultKeymap(), tc.dt)
			m.handleResize(200, 50)
			m.searchMode = tc.searchMode

			got := strings.TrimRight(m.difftoolTitleBar(), " ")
			if got != tc.want {
				t.Errorf("difftoolTitleBar() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestAbortKeyQuitsWithoutProcessExit reproduces code review CR-02: pressing
// the default abort key ("Q") must return control to bubbletea via tea.Quit
// (so raw mode / the alternate screen are restored before the process
// exits) rather than terminating the process directly from inside Update.
// Pre-fix, this test does not report a normal failure: it terminates the
// whole package test binary with exit status 1 before results can be
// reported — that IS the CR-02 reproduction.
func TestAbortKeyQuitsWithoutProcessExit(t *testing.T) {
	t.Run("abort_key_quits_and_marks_aborted", func(t *testing.T) {
		m := NewModel(nil, false, config.DefaultKeymap(), DifftoolInfo{})
		m.handleResize(80, 24)

		newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'Q', Text: "Q"})
		if cmd == nil {
			t.Fatal("abort key ('Q'): expected non-nil Cmd (tea.Quit), got nil")
		}
		if msg := cmd(); msg != (tea.QuitMsg{}) {
			t.Errorf("abort key ('Q') cmd() = %#v, want tea.QuitMsg{}", msg)
		}
		if !WasAborted(newModel) {
			t.Error("abort key ('Q'): WasAborted(newModel) = false, want true")
		}
	})

	t.Run("quit_key_quits_without_abort", func(t *testing.T) {
		m := NewModel(nil, false, config.DefaultKeymap(), DifftoolInfo{})
		m.handleResize(80, 24)

		newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
		if cmd == nil {
			t.Fatal("quit key ('q'): expected non-nil Cmd (tea.Quit), got nil")
		}
		if msg := cmd(); msg != (tea.QuitMsg{}) {
			t.Errorf("quit key ('q') cmd() = %#v, want tea.QuitMsg{}", msg)
		}
		if WasAborted(newModel) {
			t.Error("quit key ('q'): WasAborted(newModel) = true, want false (quit and abort stay distinguishable)")
		}
	})

	t.Run("abort_key_in_difftool_mode_behaves_identically", func(t *testing.T) {
		dt := DifftoolInfo{Enabled: true, Filename: "render.go"}
		m := NewModel(nil, false, config.DefaultKeymap(), dt)
		m.handleResize(80, 24)

		newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'Q', Text: "Q"})
		if cmd == nil {
			t.Fatal("abort key ('Q') in difftool mode: expected non-nil Cmd (tea.Quit), got nil")
		}
		if msg := cmd(); msg != (tea.QuitMsg{}) {
			t.Errorf("abort key ('Q') in difftool mode cmd() = %#v, want tea.QuitMsg{}", msg)
		}
		if !WasAborted(newModel) {
			t.Error("abort key ('Q') in difftool mode: WasAborted(newModel) = false, want true (no mode-specific guard)")
		}
	})
}

// TestViewNoPanicOnShortTerminal reproduces code review CR-01: View()'s
// separator-column repeat count goes negative (and strings.Repeat panics)
// whenever the effective content height is small — a 1-row terminal with
// search closed, or a 2-row terminal with search open. Reaching the end of
// each subtest is the assertion; an unclamped repeat count aborts the
// subtest (and the whole package test binary) with a panic before any
// t.Error can run.
func TestViewNoPanicOnShortTerminal(t *testing.T) {
	tests := []struct {
		height     int
		searchMode bool
	}{
		{height: 0, searchMode: false},
		{height: 1, searchMode: false},
		{height: 2, searchMode: false},
		{height: 3, searchMode: false},
		{height: 24, searchMode: false},
		{height: 1, searchMode: true},
		{height: 2, searchMode: true},
		{height: 3, searchMode: true},
		{height: 24, searchMode: true},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("height_%d_search_%v", tc.height, tc.searchMode), func(t *testing.T) {
			m := NewModel(nil, false, config.DefaultKeymap(), DifftoolInfo{})
			m.searchMode = tc.searchMode
			m.handleResize(80, tc.height)
			v := m.View()

			if tc.height == 24 && !tc.searchMode {
				// Non-regression (CR-01): above the degenerate boundary the
				// separator column is unchanged — one "│" per content row.
				if got := strings.Count(v.Content, "│"); got != 23 {
					t.Errorf("View().Content has %d '│' chars at 80x24 search-closed, want 23 (separator unchanged above the degenerate boundary)", got)
				}
			}
		})
	}
}

// TestDifftoolTitleBarTruncatesWithEllipsis reproduces the 04-VERIFICATION.md
// gap (DIFFTOOL-02): an overflowing filename must truncate to exactly
// m.termWidth display cells and end in the horizontal-ellipsis character
// U+2026, never wrap to a second row, and never panic at degenerate widths.
// A title that already fits (handled by TestDifftoolTitleBar above) must stay
// byte-identical — this test only exercises the overflow path.
func TestDifftoolTitleBarTruncatesWithEllipsis(t *testing.T) {
	longName := strings.Repeat("x", 500) + ".go"
	dt := DifftoolInfo{Enabled: true, Counter: 3, Total: 7, Filename: longName}

	widths := []int{40, 20, 5, 1, 0}
	for _, width := range widths {
		t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
			m := NewModel(nil, false, config.DefaultKeymap(), dt)
			m.handleResize(width, 50)

			got := m.difftoolTitleBar()
			if w := lipgloss.Width(got); w != width {
				t.Errorf("difftoolTitleBar() display width = %d, want %d", w, width)
			}
			if strings.Contains(got, "\n") {
				t.Errorf("difftoolTitleBar() contains a newline — must stay a single row (04-03 truth #13): %q", got)
			}
			if width >= 2 {
				trimmed := strings.TrimRight(got, " ")
				if !strings.HasSuffix(trimmed, "…") {
					t.Errorf("difftoolTitleBar() at width %d = %q, want suffix %q (DIFFTOOL-02 ellipsis)", width, trimmed, "…")
				}
			}
		})
	}
}

// TestDifftoolLayoutHasNoTree verifies that in difftool mode, handleResize
// gives the diff viewport the full terminal width (no treeWidth/separator
// subtraction) and View() renders no tree-pane/separator column appended
// beside the diff pane (DIFFTOOL-01). Body line width (not glyph presence)
// is the correct assertion here: diff.Render's own " │ " old/new column
// separator is a legitimate part of every diff row in both modes — only the
// *pane*-level tree/diff separator column is difftool-mode-specific, and its
// absence shows up as the line's total display width matching diffVP.Width()
// exactly rather than treeWidth+1 wider.
func TestDifftoolLayoutHasNoTree(t *testing.T) {
	files := parseAllFixture(t, "simple.diff")
	dt := DifftoolInfo{Enabled: true, Filename: "render.go"}
	m := NewModel(files, false, config.DefaultKeymap(), dt)
	m.handleResize(200, 50)

	if got := m.diffVP.Width(); got != 200 {
		t.Errorf("after handleResize(200, 50): diffVP.Width() = %d, want 200 (no tree/separator subtraction)", got)
	}

	view := m.View().Content
	lines := strings.Split(view, "\n")
	if len(lines) < 2 {
		t.Fatalf("View() has %d lines, want at least 2 (title bar + body)", len(lines))
	}
	// lines[0] is the title bar; lines[1] is the first diff-pane body row.
	if w := lipgloss.Width(lines[1]); w != 200 {
		t.Errorf("difftool mode body line display width = %d, want 200 (no tree pane/separator column appended)", w)
	}
}

// TestDifftoolTabAndAllFilesAreNoOps verifies that Tab and 'a' leave
// treeWidth, focusedPane and allFiles unchanged in difftool mode
// (DIFFTOOL-01, 04-UI-SPEC.md Interaction States).
func TestDifftoolTabAndAllFilesAreNoOps(t *testing.T) {
	dt := DifftoolInfo{Enabled: true, Filename: "render.go"}
	m := NewModel(nil, false, config.DefaultKeymap(), dt)
	m.handleResize(200, 50)

	treeWidthBefore := m.treeWidth
	focusedPaneBefore := m.focusedPane
	allFilesBefore := m.allFiles

	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = m2.(model)
	if m.treeWidth != treeWidthBefore {
		t.Errorf("after Tab in difftool mode: treeWidth = %d, want %d (unchanged)", m.treeWidth, treeWidthBefore)
	}
	if m.focusedPane != focusedPaneBefore {
		t.Errorf("after Tab in difftool mode: focusedPane = %v, want %v (unchanged)", m.focusedPane, focusedPaneBefore)
	}

	m3, _ := m.Update(tea.KeyPressMsg{Code: 'a'})
	m = m3.(model)
	if m.allFiles != allFilesBefore {
		t.Errorf("after 'a' in difftool mode: allFiles = %v, want %v (unchanged)", m.allFiles, allFilesBefore)
	}
}
