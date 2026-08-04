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
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/term"

	"github.com/alturd/alturd/internal/config"
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

// DifftoolInfo carries the difftool-mode-only data cmd/alturd/main.go
// resolves before constructing the model: whether difftool mode is active,
// the git-supplied N-of-M counters, the real working-tree filename (git's
// $MERGED), and the pre-loaded post-image lines for full-file rendering. A
// struct rather than four positional NewModel parameters keeps the
// constructor signature from growing a fifth, sixth and seventh argument,
// and lets the zero value mean "standalone mode" (DIFFTOOL-01, DIFFTOOL-02).
type DifftoolInfo struct {
	Enabled      bool
	Counter      int
	Total        int
	Filename     string
	NewFileLines []string
}

type model struct {
	files    []*gitdiff.File
	darkBg   bool
	keys     config.Keymap
	difftool DifftoolInfo

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
	searchTyping    bool // true while textinput is focused for input; false in n/N navigation phase
	searchInput     textinput.Model
	diffContent     string // unhighlighted rendered diff content
	searchMatches   []Match
	searchMatchIdx  int

	aborted bool // true when the user pressed the abort key (config.ActionAbort, CR-02)
}

// Aborted reports whether the user pressed the abort key. See WasAborted for
// the exported boundary-crossing accessor cmd/alturd uses.
func (m model) Aborted() bool { return m.aborted }

// WasAborted reports whether the final tea.Model returned by tea.Program.Run
// represents an abort (the user pressed the documented abort key, default
// "Q") rather than a normal quit. A true result means the caller must exit
// with status 1 (code review CR-02, D-08: difftool.trustExitCode = true
// reads only the process exit status); by the time tea.Program.Run returns,
// the terminal has already been restored by bubbletea's own unwind.
//
// NewModel returns the unexported model type, so cmd/alturd cannot name it
// directly — asserting the returned tea.Model against a locally declared
// method interface is the idiomatic way to cross that package boundary in
// Go. A final model that doesn't implement Aborted() (should not happen in
// practice, since NewModel is the only constructor) is treated as false.
func WasAborted(final tea.Model) bool {
	a, ok := final.(interface{ Aborted() bool })
	return ok && a.Aborted()
}

// NewModel creates the initial bubbletea model. files must be non-nil (may be empty).
// darkBg should be true when the terminal background is dark; it controls status
// marker colours in the tree pane. keys resolves every normal-mode keypress to an
// action (config.Keymap.Lookup); a nil keys defaults to config.DefaultKeymap() so
// existing callers that pass no override still get Phase 3's default bindings.
// dt carries difftool-mode data; the zero value (DifftoolInfo{}) means standalone
// mode (DIFFTOOL-01, DIFFTOOL-02).
// Called from cmd/alturd/main.go after git+parse and background detection complete (D-06).
//
//nolint:revive // model is intentionally unexported: callers only need the tea.Model interface.
func NewModel(files []*gitdiff.File, darkBg bool, keys config.Keymap, dt DifftoolInfo) model {
	if keys == nil {
		keys = config.DefaultKeymap()
	}

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
		darkBg:      darkBg,
		keys:        keys,
		difftool:    dt,
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

	// Forward non-key messages to the textinput only in the typing phase.
	// In navigation phase the textinput is blurred so no blink commands are pending.
	if m.searchMode && m.searchTyping {
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

	diffStr := m.diffVP.View()

	var searchBar string
	if m.searchMode {
		searchBar = "\n" + m.searchInput.View()
	}

	// Difftool mode (DIFFTOOL-01): reduced-chrome variant — the difftool
	// title bar replaces the standalone status bar entirely (the two
	// templates are alternatives, never concatenated), and the body is the
	// diff viewport plus the optional search bar only: no treeVP.View(),
	// no separator column, no lipgloss.JoinHorizontal call at all.
	if m.difftool.Enabled {
		v := tea.NewView(m.difftoolTitleBar() + "\n" + diffStr + searchBar)
		v.AltScreen = true
		return v
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

	// Build a full-height separator column: JoinHorizontal pads a single "│"
	// with empty strings for all rows beyond the first, making the separator
	// invisible on all but the top row. Repeating "│" for each content row
	// ensures the vertical divider spans the full pane height (Bug 1 fix).
	contentH := m.termHeight - 1
	if m.searchMode {
		contentH--
	}
	// CR-01 (code review): m.termHeight arrives unvalidated from
	// tea.WindowSizeMsg and the Windows resizePollMsg tick, so a one-row
	// terminal (or two rows with search open, or any transient degenerate
	// resize report) can make contentH-1 negative — strings.Repeat panics
	// on a negative count, taking down the whole render path with no
	// recover anywhere above it. Reproduced via handleResize(80, 1) then
	// View(). Clamp at zero so the separator degenerates to a single "│"
	// instead of crashing.
	sepLines := contentH - 1
	if sepLines < 0 {
		sepLines = 0
	}
	sep := strings.Repeat("│\n", sepLines) + "│"

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

// difftoolTitleBar implements the DIFFTOOL-02 Copywriting Contract. Segment
// order is fixed and never varies: the literal "alturd (difftool)", then
// " — {Counter} of {Total}" only when both Counter and Total are greater
// than zero, then " — {Filename}", then " [SEARCH]" only when m.searchMode.
// Counter and Total render with %d exactly as received — including when
// equal or adjacent — never clamped, wrapped or special-cased. A long
// filename is truncated to m.termWidth via ansi.Truncate with an explicit
// U+2026 ("…") tail rather than wrapping onto a second row, then padded to
// m.termWidth the same way the standalone status bar is.
//
// lipgloss.Style.MaxWidth is NOT used here: it truncates via
// ansi.Truncate(line, maxWidth, "") with a hardcoded empty tail internally
// (charm.land/lipgloss/v2@v2.0.5/style.go:510) and has no user-facing tail
// parameter, so it cannot append an ellipsis. Calling ansi.Truncate directly
// gives the same ANSI-aware, wide-character-aware truncation with an
// explicit tail, satisfying 04-UI-SPEC.md's Copywriting Contract ("ellipsis
// appended") and closing the gap 04-VERIFICATION.md recorded against
// DIFFTOOL-02 (04-05-PLAN.md Task 1).
func (m model) difftoolTitleBar() string {
	title := "alturd (difftool)"
	if m.difftool.Counter > 0 && m.difftool.Total > 0 {
		title += fmt.Sprintf(" — %d of %d", m.difftool.Counter, m.difftool.Total)
	}
	title += " — " + m.difftool.Filename
	if m.searchMode {
		title += " [SEARCH]"
	}
	title = ansi.Truncate(title, m.termWidth, "…")
	return lipgloss.NewStyle().Width(m.termWidth).Render(title)
}

func (m *model) handleResize(w, h int) {
	m.termWidth = w
	m.termHeight = h
	m.ready = true

	contentH := h - 1
	if m.searchMode {
		contentH--
	}
	// CR-01 (code review): clamp the same way View()'s independent contentH
	// computation is clamped — an unvalidated terminal height reaching
	// treeVP.SetHeight/diffVP.SetHeight is the width twin of the panic this
	// clamp prevents in View(). Height only; diffW (WR-02) stays untouched.
	if contentH < 0 {
		contentH = 0
	}

	// Difftool mode (DIFFTOOL-01): the diff pane takes the full width — no
	// treeWidth/separator subtraction — and no tree viewport exists to size
	// or refresh.
	if m.difftool.Enabled {
		m.diffVP.SetWidth(w)
		m.diffVP.SetHeight(contentH)
		m.refreshDiffContent()
		return
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
	diffW := m.termWidth
	if !m.difftool.Enabled {
		diffW = m.termWidth - m.treeWidth - 1
	}
	file := m.files[m.currentFile]

	var rows []string
	if m.renderMode == diff.FullFile {
		var fileLines []string
		var err error
		if m.difftool.Enabled {
			// Difftool mode: use the already-loaded post-image lines instead
			// of fetchFileLines, which shells out to `git show HEAD:<name>` —
			// meaningless for a file git handed us as a temp path. This keeps
			// full-file mode (DIFF-05) working in difftool mode rather than
			// silently degrading to hunk-context rendering.
			fileLines = m.difftool.NewFileLines
		} else {
			fileLines, err = fetchFileLines(file)
		}
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

// statusMarkerStyle returns a lipgloss Style that applies an appropriate foreground
// colour to a file-status marker based on the marker text and terminal background.
// Colours use 256-colour ANSI codes chosen to be readable on both dark and light
// backgrounds: darker shades for light terminals, brighter shades for dark terminals.
func statusMarkerStyle(status string, darkBg bool) lipgloss.Style {
	var code string
	switch status {
	case "[A]":
		if darkBg {
			code = "82" // bright green
		} else {
			code = "28" // dark green
		}
	case "[D]":
		if darkBg {
			code = "203" // bright red
		} else {
			code = "88" // dark red
		}
	case "[M]":
		if darkBg {
			code = "220" // bright yellow
		} else {
			code = "136" // dark amber
		}
	case "[R]", "[C]":
		if darkBg {
			code = "81" // bright cyan
		} else {
			code = "26" // dark blue
		}
	default: // [B], [S], unknown
		if darkBg {
			code = "245" // medium gray
		} else {
			code = "240" // dark gray
		}
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(code))
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
			status := row.node.Status
			if status == "" {
				line = indent + "    " + row.node.Name
			} else {
				colored := statusMarkerStyle(status, m.darkBg).Render(status)
				line = indent + colored + " " + row.node.Name
			}
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
	// Search mode: two-phase dispatch (SEARCH-01, D-13/D-14/D-15/D-16).
	// Phase 1 — typing (searchTyping=true): esc closes; enter commits to navigation;
	//   all other keys (including 'n') forwarded to the textinput so the user can
	//   type any character in their query.
	// Phase 2 — navigation (searchTyping=false): n/N cycle matches; [/] close and
	//   cycle files; any other character re-enters typing phase.
	if m.searchMode {
		if m.searchTyping {
			switch msg.String() {
			case "esc":
				// D-15: close search, clear state, restore viewport height.
				m.searchMode = false
				m.searchTyping = false
				m.searchInput.Reset()
				m.searchMatches = nil
				m.searchMatchIdx = 0
				m.handleResize(m.termWidth, m.termHeight)
			case "enter":
				// Commit search — switch to navigation phase; blur cursor.
				m.searchTyping = false
				m.searchInput.Blur()
			default:
				// Forward all typed characters to the textinput (SEARCH-01, D-13).
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.recomputeSearch()
				return m, cmd
			}
		} else {
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
				// Any typed character re-enters typing phase.
				m.searchTyping = true
				focusCmd := m.searchInput.Focus()
				var updateCmd tea.Cmd
				m.searchInput, updateCmd = m.searchInput.Update(msg)
				m.recomputeSearch()
				return m, tea.Batch(focusCmd, updateCmd)
			}
		}
		return m, nil
	}

	// Resolve the pressed key to a rebindable action through the single
	// config.Keymap lookup (D-04 assumption-delta: promote, not add-alongside —
	// every normal-mode key passes through exactly one Lookup call, so a
	// rebound key and its former default can never both fire).
	action := m.keys.Lookup(msg.String())

	// Difftool mode (DIFFTOOL-01): Tab and 'a' are no-ops — no tree pane
	// exists to focus or toggle. This guard must run before any tree-scoped
	// action executes, so it prevents a resize of a viewport that was never
	// sized rather than merely hiding a cosmetic affordance.
	if m.difftool.Enabled && (action == config.ActionToggleFocus || action == config.ActionToggleAllFiles) {
		return m, nil
	}

	switch action {
	case config.ActionQuit:
		return m, tea.Quit
	case config.ActionAbort:
		// CR-02 (code review): terminating the process directly here (the
		// former os.Exit(1)) bypasses bubbletea's terminal restore — the
		// user is left with raw mode / the alternate screen still active
		// until reset/stty sane. Route through the normal tea.Quit path
		// instead so tea.Program.Run unwinds and restores the terminal;
		// cmd/alturd/main.go inspects the final model via tui.WasAborted
		// and applies exit status 1 only after Run() returns.
		m.aborted = true
		return m, tea.Quit
	case config.ActionToggleFocus:
		m.toggleFocus()
		m.handleResize(m.termWidth, m.termHeight)
	case config.ActionToggleRenderMode:
		if m.renderMode == diff.FullFile {
			m.renderMode = diff.HunkOnly
		} else {
			m.renderMode = diff.FullFile
		}
		m.refreshDiffContent()
		m.scrollToFirstHunk()
	case config.ActionNextHunk:
		m.hunkNext()
	case config.ActionPrevHunk:
		m.hunkPrev()
	case config.ActionNextFile:
		m.handleFileCycle(true)
	case config.ActionPrevFile:
		m.handleFileCycle(false)
	case config.ActionOpenSearch:
		// D-13: open search — shrink viewport by 1 row, focus textinput (SEARCH-01).
		m.searchMode = true
		m.searchTyping = true
		m.handleResize(m.termWidth, m.termHeight)
		return m, m.searchInput.Focus()
	case config.ActionToggleAllFiles:
		// D-11: toggle between changed-files-only and full-repo tree (TREE-03).
		m.toggleAllFiles()
	default:
		// config.ActionNone — not one of the ten rebindable actions. These
		// pane-scroll keys and the viewport-forwarding fallback are NOT
		// rebindable (03-UI-SPEC.md scopes them outside the Key Binding
		// Contract), so they stay a literal switch on msg.String().
		switch msg.String() {
		case "enter", "l", "right":
			if m.focusedPane == treeFocused {
				m.treeToggleExpand()
			}
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
			idx = m.treeIdx
			for idx >= 0 && m.treeFlat[idx].node.IsDir {
				idx--
			}
		}
	} else {
		for idx >= 0 && m.treeFlat[idx].node.IsDir {
			idx--
		}
		if idx < 0 {
			// No file backward — walk forward
			idx = m.treeIdx
			for idx < len(m.treeFlat) && m.treeFlat[idx].node.IsDir {
				idx++
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

// treeToggleExpand toggles the expanded state of the directory node at the
// current treeIdx. If the node is a file leaf, it is ignored. After toggling,
// treeFlat is rebuilt and treeIdx is kept on the same node.
func (m *model) treeToggleExpand() {
	if len(m.treeFlat) == 0 || m.treeIdx >= len(m.treeFlat) {
		return
	}
	node := m.treeFlat[m.treeIdx].node
	if !node.IsDir {
		return
	}
	node.expanded = !node.expanded
	m.treeFlat = flattenTree(m.treeNodes[0], 0)
	// Keep treeIdx within the new flat list bounds.
	if m.treeIdx >= len(m.treeFlat) {
		m.treeIdx = max(0, len(m.treeFlat)-1)
	}
	m.refreshTreeContent()
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
