// Package tui implements the bubbletea v2 terminal UI for alturd.
// It owns all interactive state: pane layout, file selection, search mode,
// and hunk navigation. Data is pre-loaded by cmd/alturd/main.go before
// tea.NewProgram is called (D-06).
package tui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"golang.org/x/term"

	"github.com/alturd/alturd/internal/diff"
	"github.com/alturd/alturd/internal/git"
)

type pane int

const (
	treeFocused pane = iota
	diffFocused
)

const (
	treeWidthUnfocused  = 24
	treeWidthFocused    = 45
	windowsPollInterval = time.Second / 4
)

// resizePollMsg is emitted by the Windows resize tick (issue #1601).
type resizePollMsg struct{}

type model struct {
	files []*gitdiff.File

	ready       bool
	termWidth   int
	termHeight  int
	focusedPane pane
	treeWidth   int

	treeVP    viewport.Model
	treeNodes []*TreeNode
	treeFlat  []flatRow
	treeIdx   int
	allFiles  bool
	allFilePaths []string

	diffVP      viewport.Model
	currentFile int
	renderMode  diff.RenderMode
	hunkRows    []int
	currentHunk int

	searchMode      bool
	searchInput     textinput.Model
	diffContent     string // unhighlighted rendered diff content
	searchMatches   []Match
	searchMatchIdx  int
}

// NewModel creates the initial bubbletea model. files must be non-nil (may be empty).
// Called from cmd/alturd/main.go after git+parse complete (D-06).
func NewModel(files []*gitdiff.File) model {
	ti := textinput.New()
	ti.Prompt = "/"
	ti.Placeholder = "search..."

	statusMap := buildStatusMap(files)
	nodes := buildTree(filePaths(files), statusMap)
	flat := flattenTree(nodes, 0)

	// HighlightStyle: reverse video so matches are visible on any background
	// (RESEARCH Open Question 2).
	highlightStyle := lipgloss.NewStyle().Reverse(true)
	diffVP := viewport.New(viewport.WithWidth(0), viewport.WithHeight(0))
	diffVP.HighlightStyle = highlightStyle
	diffVP.SelectedHighlightStyle = highlightStyle

	return model{
		files:       files,
		focusedPane: diffFocused,
		treeWidth:   treeWidthUnfocused,
		searchInput: ti,
		treeNodes:   []*TreeNode{nodes},
		treeFlat:    flat,
		treeVP:      viewport.New(viewport.WithWidth(0), viewport.WithHeight(0)),
		diffVP:      diffVP,
	}
}

func (m model) Init() tea.Cmd {
	if runtime.GOOS == "windows" {
		return tea.Tick(windowsPollInterval, func(_ time.Time) tea.Msg {
			return resizePollMsg{}
		})
	}
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleResize(msg.Width, msg.Height)
		return m, nil

	case resizePollMsg:
		w, h, err := term.GetSize(int(os.Stdout.Fd()))
		if err == nil && (w != m.termWidth || h != m.termHeight) {
			m.handleResize(w, h)
		}
		if runtime.GOOS == "windows" {
			return m, tea.Tick(windowsPollInterval, func(_ time.Time) tea.Msg {
				return resizePollMsg{}
			})
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	// Forward non-key messages to the search textinput when search is active.
	// This allows bubbletea internal cursor blink and focus messages to reach
	// the textinput even when they are not tea.KeyPressMsg.
	if m.searchMode {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.recomputeSearch()
		return m, cmd
	}

	return m, nil
}

func (m model) View() tea.View {
	if !m.ready {
		return tea.NewView("")
	}

	fileName := ""
	if len(m.files) > 0 {
		f := m.files[m.currentFile]
		if f.NewName != "" && f.NewName != "/dev/null" {
			fileName = f.NewName
		} else {
			fileName = f.OldName
		}
	}

	statusBar := fmt.Sprintf("alturd — %s (%d of %d changed files)",
		fileName, m.currentFile+1, len(m.files))
	if m.searchMode {
		statusBar += " [SEARCH]"
	}
	statusBar = lipgloss.NewStyle().Width(m.termWidth).Render(statusBar)

	treeStr := m.treeVP.View()
	diffStr := m.diffVP.View()

	var searchBar string
	if m.searchMode {
		searchBar = "\n" + m.searchInput.View()
	}

	// Build a full-height separator column: JoinHorizontal pads a single "│"
	// with empty strings for all rows beyond the first, making the separator
	// invisible on all but the top row. Repeating "│" for each content row
	// ensures the vertical divider spans the full pane height (Bug 1 fix).
	contentH := m.termHeight - 1
	if m.searchMode {
		contentH--
	}
	sep := strings.Repeat("│\n", contentH-1) + "│"

	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		treeStr,
		sep,
		diffStr+searchBar,
	)

	v := tea.NewView(statusBar + "\n" + body)
	// AltScreen declared per-View as required by bubbletea v2 (v1's
	// tea.WithAltScreen() program option no longer exists in v2).
	v.AltScreen = true
	return v
}

func (m *model) handleResize(w, h int) {
	m.termWidth = w
	m.termHeight = h
	m.ready = true

	contentH := h - 1
	if m.searchMode {
		contentH--
	}
	diffW := w - m.treeWidth - 1

	m.treeVP.SetWidth(m.treeWidth)
	m.treeVP.SetHeight(contentH)
	m.diffVP.SetWidth(diffW)
	m.diffVP.SetHeight(contentH)

	m.refreshDiffContent()
	m.refreshTreeContent()
}

func (m *model) refreshDiffContent() {
	if len(m.files) == 0 {
		return
	}
	diffW := m.termWidth - m.treeWidth - 1
	file := m.files[m.currentFile]

	var rows []string
	if m.renderMode == diff.FullFile {
		fileLines, err := fetchFileLines(file)
		if err == nil {
			rows = diff.RenderFull(file, diffW, fileLines)
			m.hunkRows = diff.HunkStartRowsFull(file)
		} else {
			// Fall back to hunk-context rendering if the file cannot be read.
			rows = diff.Render(file, diffW, diff.FullFile)
			m.hunkRows = diff.HunkStartRows(file, diff.FullFile)
		}
	} else {
		rows = diff.Render(file, diffW, m.renderMode)
		m.hunkRows = diff.HunkStartRows(file, m.renderMode)
	}
	m.currentHunk = 0
	// Expand tab characters to a single space before storing in the viewport.
	// lipgloss.Style.Width() — used internally by viewport.View() — converts
	// tabs to 4 spaces, which inflates line width beyond the viewport width and
	// causes word-wrap to fire, adding spurious newlines and corrupting layout.
	// Using a single space keeps intent (indentation present) while avoiding
	// the expansion mismatch (tabs are already truncated to 1 visible rune by
	// truncateANSI in diff.Render, so a space is the correct replacement).
	for i, r := range rows {
		rows[i] = strings.ReplaceAll(r, "\t", " ")
	}
	m.diffContent = strings.Join(rows, "\n")
	m.diffVP.SetContent(m.diffContent)
	if m.searchMode && m.searchInput.Value() != "" {
		m.recomputeSearch()
	}
}

func (m *model) refreshTreeContent() {
	m.diffVP.SetWidth(m.termWidth - m.treeWidth - 1)
	content := m.renderTree()
	m.treeVP.SetContent(content)
}

func (m *model) renderTree() string {
	var sb strings.Builder
	for i, row := range m.treeFlat {
		indent := strings.Repeat("  ", row.depth)
		var line string
		if row.node.IsDir {
			glyph := "▸"
			if row.node.expanded {
				glyph = "▾"
			}
			line = indent + glyph + " " + row.node.Name
		} else {
			marker := row.node.Status
			if marker == "" {
				marker = "   "
			} else {
				marker = marker + " "
			}
			line = indent + marker + row.node.Name
		}
		line = lipgloss.NewStyle().MaxWidth(m.treeWidth).Render(line)
		if i == m.treeIdx {
			line = lipgloss.NewStyle().Reverse(true).Render(line)
		}
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(line)
	}
	return sb.String()
}

func (m model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Search mode: dispatch before the normal-mode switch (SEARCH-01, D-13/D-14/D-15/D-16).
	if m.searchMode {
		switch msg.String() {
		case "esc":
			// D-15: close search, clear state, restore viewport height.
			m.searchMode = false
			m.searchInput.Reset()
			m.searchMatches = nil
			m.searchMatchIdx = 0
			m.handleResize(m.termWidth, m.termHeight)
		case "n":
			// D-14: navigate to next match.
			m.searchNextMatch(1)
		case "N":
			// D-14: navigate to previous match.
			m.searchNextMatch(-1)
		case "]":
			// D-16: close search then cycle to next file.
			m.searchMode = false
			m.searchInput.Reset()
			m.searchMatches = nil
			m.searchMatchIdx = 0
			m.handleResize(m.termWidth, m.termHeight)
			m.handleFileCycle(true)
		case "[":
			// D-16: close search then cycle to previous file.
			m.searchMode = false
			m.searchInput.Reset()
			m.searchMatches = nil
			m.searchMatchIdx = 0
			m.handleResize(m.termWidth, m.termHeight)
			m.handleFileCycle(false)
		default:
			// Forward typed characters to the textinput (SEARCH-01, D-13).
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			m.recomputeSearch()
			return m, cmd
		}
		return m, nil
	}

	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "Q":
		os.Exit(1)
	case "tab":
		m.toggleFocus()
		m.handleResize(m.termWidth, m.termHeight)
	case "v":
		if m.renderMode == diff.FullFile {
			m.renderMode = diff.HunkOnly
		} else {
			m.renderMode = diff.FullFile
		}
		m.refreshDiffContent()
		m.scrollToFirstHunk()
	case "n":
		m.hunkNext()
	case "N":
		m.hunkPrev()
	case "]":
		m.handleFileCycle(true)
	case "[":
		m.handleFileCycle(false)
	case "j", "down":
		if m.focusedPane == treeFocused {
			m.treeIdxMove(1)
		} else {
			m.diffVP.ScrollDown(1)
		}
	case "k", "up":
		if m.focusedPane == treeFocused {
			m.treeIdxMove(-1)
		} else {
			m.diffVP.ScrollUp(1)
		}
	case "/":
		// D-13: open search — shrink viewport by 1 row, focus textinput (SEARCH-01).
		m.searchMode = true
		m.handleResize(m.termWidth, m.termHeight)
		return m, m.searchInput.Focus()
	case "a":
		// D-11: toggle between changed-files-only and full-repo tree (TREE-03).
		m.toggleAllFiles()
	default:
		// Route scroll/navigation keys to the focused pane. When the tree pane
		// is focused (Tab pressed), arrow/vim keys scroll the tree; otherwise
		// they scroll the diff. This ensures j/k/arrows work in both panes.
		var cmd tea.Cmd
		if m.focusedPane == treeFocused {
			m.treeVP, cmd = m.treeVP.Update(msg)
		} else {
			m.diffVP, cmd = m.diffVP.Update(msg)
		}
		return m, cmd
	}
	return m, nil
}

func (m *model) toggleFocus() {
	if m.focusedPane == treeFocused {
		m.focusedPane = diffFocused
		m.treeWidth = treeWidthUnfocused
	} else {
		m.focusedPane = treeFocused
		m.treeWidth = treeWidthFocused
	}
}

func (m *model) handleFileCycle(forward bool) {
	if len(m.files) == 0 {
		return
	}
	if forward {
		m.currentFile = (m.currentFile + 1) % len(m.files)
	} else {
		m.currentFile = (m.currentFile - 1 + len(m.files)) % len(m.files)
	}
	m.currentHunk = 0
	m.refreshDiffContent()
	m.scrollToFirstHunk()
}

func (m *model) hunkNext() {
	if len(m.hunkRows) == 0 {
		return
	}
	if m.currentHunk < len(m.hunkRows)-1 {
		m.currentHunk++
	}
	m.diffVP.SetYOffset(max(0, m.hunkRows[m.currentHunk]-m.diffVP.Height()/2))
}

func (m *model) hunkPrev() {
	if len(m.hunkRows) == 0 {
		return
	}
	if m.currentHunk > 0 {
		m.currentHunk--
	}
	m.diffVP.SetYOffset(max(0, m.hunkRows[m.currentHunk]-m.diffVP.Height()/2))
}

// recomputeSearch refreshes searchMatches from the current textinput value and
// bakes highlights into the diff viewport content (SEARCH-01, D-13).
// Bypasses viewport.SetHighlights because its internal parseMatches function
// incorrectly detects newlines when content contains ANSI codes (the function
// checks content[bytePos] where bytePos advances through stripped-content positions,
// not original-content positions). We compute per-line grapheme positions ourselves
// and apply them via lipgloss.StyleRanges instead.
func (m *model) recomputeSearch() {
	query := m.searchInput.Value()
	m.searchMatches = findMatches(m.diffContent, query)
	m.searchMatchIdx = 0
	highlighted := m.applySearchHighlights()
	m.diffVP.SetContent(highlighted)
	if len(m.searchMatches) > 0 {
		m.scrollToMatch(0)
	}
}

// applySearchHighlights returns m.diffContent with all search matches highlighted
// using lipgloss.StyleRanges. The selected match (searchMatchIdx) uses a bolder style.
func (m *model) applySearchHighlights() string {
	if len(m.searchMatches) == 0 {
		return m.diffContent
	}
	hl := lipgloss.NewStyle().Reverse(true)
	hlSel := lipgloss.NewStyle().Reverse(true).Bold(true)

	lines := strings.Split(m.diffContent, "\n")
	type colRange struct {
		start, end int
		sel        bool
	}
	byLine := make(map[int][]colRange)
	for i, match := range m.searchMatches {
		byLine[match.Line] = append(byLine[match.Line], colRange{match.ColStart, match.ColEnd, i == m.searchMatchIdx})
	}
	for lineN, ranges := range byLine {
		if lineN >= len(lines) {
			continue
		}
		lgRanges := make([]lipgloss.Range, 0, len(ranges))
		for _, r := range ranges {
			style := hl
			if r.sel {
				style = hlSel
			}
			lgRanges = append(lgRanges, lipgloss.NewRange(r.start, r.end, style))
		}
		lines[lineN] = lipgloss.StyleRanges(lines[lineN], lgRanges...)
	}
	return strings.Join(lines, "\n")
}

// searchNextMatch advances the current match by delta (±1, wrapping) and scrolls
// the viewport to centre it.
func (m *model) searchNextMatch(delta int) {
	if len(m.searchMatches) == 0 {
		return
	}
	m.searchMatchIdx = (m.searchMatchIdx + delta + len(m.searchMatches)) % len(m.searchMatches)
	m.diffVP.SetContent(m.applySearchHighlights())
	m.scrollToMatch(m.searchMatchIdx)
}

// scrollToMatch scrolls the diff viewport so that match idx is roughly centred.
func (m *model) scrollToMatch(idx int) {
	if idx < 0 || idx >= len(m.searchMatches) {
		return
	}
	line := m.searchMatches[idx].Line
	offset := line - m.diffVP.Height()/2
	if offset < 0 {
		offset = 0
	}
	m.diffVP.SetYOffset(offset)
}

// toggleAllFiles toggles between the changed-files-only tree and the full-repo
// tree (TREE-03, D-11). On the first 'a' press it runs git ls-tree to load all
// repo paths and caches them in allFilePaths; subsequent presses reuse the cache.
// If the git subprocess fails, the toggle is silently undone so the TUI keeps
// showing the changed-files tree without crashing.
func (m *model) toggleAllFiles() {
	m.allFiles = !m.allFiles

	if m.allFiles && len(m.allFilePaths) == 0 {
		// Lazy-load full repo path list via git ls-tree (RESEARCH git ls-tree section).
		// --full-tree makes paths root-relative regardless of cwd.
		reader, err := git.ExecRunner{}.Run([]string{
			"ls-tree", "-r", "--full-tree", "--name-only", "HEAD",
		})
		if err != nil {
			// Do not crash the TUI; revert the toggle.
			m.allFiles = false
			return
		}
		var paths []string
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				paths = append(paths, line)
			}
		}
		if err := scanner.Err(); err != nil {
			m.allFiles = false
			return
		}
		m.allFilePaths = paths
	}

	// Rebuild the tree over either the full repo paths or the changed-files paths.
	statusMap := buildStatusMap(m.files)
	var paths []string
	if m.allFiles {
		paths = m.allFilePaths
	} else {
		paths = filePaths(m.files)
	}
	root := buildTree(paths, statusMap)
	m.treeNodes = []*TreeNode{root}
	m.treeFlat = flattenTree(root, 0)
	// Clamp treeIdx into the valid range.
	if m.treeIdx >= len(m.treeFlat) {
		m.treeIdx = max(0, len(m.treeFlat)-1)
	}
	m.refreshTreeContent()
}

// treeIdxMove moves the tree selection by delta rows (±1), updates treeIdx,
// finds the nearest file in that direction to update currentFile, and scrolls
// the tree viewport to keep the selected row in view.
func (m *model) treeIdxMove(delta int) {
	if len(m.treeFlat) == 0 {
		return
	}
	m.treeIdx = max(0, min(m.treeIdx+delta, len(m.treeFlat)-1))

	// Find the nearest file node at or past treeIdx in the direction of movement.
	idx := m.treeIdx
	if delta >= 0 {
		for idx < len(m.treeFlat) && m.treeFlat[idx].node.IsDir {
			idx++
		}
		if idx >= len(m.treeFlat) {
			// No file forward — walk backward
			for idx = m.treeIdx; idx >= 0 && m.treeFlat[idx].node.IsDir; idx-- {
			}
		}
	} else {
		for idx >= 0 && m.treeFlat[idx].node.IsDir {
			idx--
		}
		if idx < 0 {
			// No file backward — walk forward
			for idx = m.treeIdx; idx < len(m.treeFlat) && m.treeFlat[idx].node.IsDir; idx++ {
			}
		}
	}

	// If we found a file node, update currentFile.
	if idx >= 0 && idx < len(m.treeFlat) && !m.treeFlat[idx].node.IsDir {
		path := m.treeFlat[idx].node.Path
		for i, f := range m.files {
			name := f.NewName
			if name == "" || name == "/dev/null" {
				name = f.OldName
			}
			if name == path {
				m.currentFile = i
				m.refreshDiffContent()
				m.scrollToFirstHunk()
				break
			}
		}
	}

	// Scroll treeVP to keep treeIdx visible.
	m.refreshTreeContent()
	if m.treeIdx < m.treeVP.YOffset() {
		m.treeVP.SetYOffset(m.treeIdx)
	} else if m.treeIdx >= m.treeVP.YOffset()+m.treeVP.Height() {
		m.treeVP.SetYOffset(m.treeIdx - m.treeVP.Height() + 1)
	}
}

// scrollToFirstHunk centres the diff viewport on the first hunk when in FullFile
// mode. Called after any operation that changes the active file or render mode so
// that the first change is immediately visible instead of the user having to scroll
// through unchanged lines at the top of the file.
func (m *model) scrollToFirstHunk() {
	if m.renderMode == diff.FullFile && len(m.hunkRows) > 0 {
		m.diffVP.SetYOffset(max(0, m.hunkRows[0]-m.diffVP.Height()/2))
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// fetchFileLines reads the complete line content of a file for full-file rendering.
// For deleted files it reads the old version via git show HEAD:OldName.
// For all other files it tries, in order:
//  1. git show HEAD:path  (reliable — path is repo-root-relative; works for any CWD)
//  2. git show :path      (staged index version — for newly-added or staged files)
//  3. os.ReadFile(path)   (working-tree fallback)
//
// Returns nil, nil for files with no usable name (binary specials etc.) where
// AlignFull handles rendering via its own placeholder logic.
func fetchFileLines(f *gitdiff.File) ([]string, error) {
	if f.IsDelete {
		r, err := git.ExecRunner{}.Run([]string{"show", "HEAD:" + f.OldName})
		if err != nil {
			return nil, err
		}
		return scanLines(r), nil
	}

	name := f.NewName
	if name == "" || name == "/dev/null" {
		name = f.OldName
	}
	if name == "" {
		return nil, nil
	}

	// Try the committed version first — path is relative to repo root so this
	// works regardless of the user's current working directory.
	r, err := git.ExecRunner{}.Run([]string{"show", "HEAD:" + name})
	if err == nil {
		return scanLines(r), nil
	}

	// Try the staged (index) version — handles newly-added files that are staged
	// but not yet committed.
	r, err = git.ExecRunner{}.Run([]string{"show", ":" + name})
	if err == nil {
		return scanLines(r), nil
	}

	// Last resort: read from the working tree.
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return splitFileBytes(data), nil
}

// scanLines reads all lines from r using bufio.Scanner, stripping trailing \r.
func scanLines(r io.Reader) []string {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, strings.TrimRight(scanner.Text(), "\r"))
	}
	return lines
}

// splitFileBytes splits raw file bytes into lines, normalising CRLF and
// stripping the final newline so each element holds one logical line.
func splitFileBytes(data []byte) []string {
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	if len(content) > 0 && content[len(content)-1] == '\n' {
		content = content[:len(content)-1]
	}
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}
