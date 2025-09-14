package specui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	ansi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	"github.com/charmbracelet/x/xpty"
	runewidth "github.com/mattn/go-runewidth"
	"syscall"

	"codectl/internal/system"
	uistyle "codectl/internal/ui"
	fsnotify "github.com/fsnotify/fsnotify"
	zone "github.com/lrstanley/bubblezone"
)

// Start runs the spec UI program
func Start() error {
	m := initialModel()
	// Ensure global zone manager exists (idempotent if already created).
	zone.NewGlobal()
	// Disable mouse so users can select and copy freely in terminal
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

type page int

const (
	pageSelect page = iota
	pageDetail
)

type specItem struct {
	Path  string
	Title string
}

type model struct {
	// env
	cwd    string
	root   string
	width  int
	height int
	// cached pane widths for stable layout
	leftW  int
	rightW int

	// flow
	page page
	// legacy spec list (unused when using file manager)
	items []specItem
	table table.Model
	// file explorer (tree) state
	fmRoot    string
	fmCwd     string
	fileTable table.Model
	treeRoot  *treeNode
	expanded  map[string]bool
	visible   []treeRow
	selected  *specItem
	mdVP      viewport.Model
	logVP     viewport.Model
	logs      []string
	ti        textinput.Model
	statusMsg string
	now       time.Time
	// rendering options
	fastMode bool
	// terminal mode state
	termMode   bool
	cmdRunning bool
	pty        xpty.Pty
	termVT     *vt.Emulator
	termFocus  bool
	termDirty  bool
	// track OSC sequence state across PTY read chunks
	oscPending bool
	// markdown cache: path -> width -> entry
	mdCache map[string]map[int]mdEntry
	// tab bar: explorer vs diff
	tab activeTab
	// Diff view state (File Diff tab)
	diffTable          table.Model
	diffItems          []diffRow
	diffVP             viewport.Model
	diffMode           diffMode
	diffFilterSpecOnly bool
	gitInRepo          bool
	hasDelta           bool
	diffErr            string
	// current diff target file + last known stat
	diffCurrentFile string
	diffFileModUnix int64
	diffFileSize    int64
	// focus management
	focus        focusArea
	lastTopFocus focusArea
	// file tree change detection
	treeStamp int64
	treeCount int

	// fsnotify watcher for live updates
	watcher *fsnotify.Watcher
	watchCh chan struct{}

	// Work tab: tasks + filters
	workTaskTable table.Model
	workTasks     []taskItem
	workFiltered  []taskItem
	workOwnerList []string
	workOwnerIdx  int
	workStatus    string // All|backlog|in-progress|blocked|done|draft|accepted
	workOwner     string // All|<owner>
	workPriority  string // All|P0|P1|P2
	workSearch    string
}

type mdEntry struct {
	out     string
	modUnix int64
	size    int64
}

// fileEntry removed (unused)

type treeNode struct {
	Name     string
	Path     string
	IsDir    bool
	Parent   *treeNode
	Children []*treeNode
}

type treeRow struct {
	Node *treeNode
	Text string
}

type focusArea int

const (
	focusFiles focusArea = iota
	focusPreview
	focusInput
)

// top-level tabs
type activeTab int

const (
	tabExplorer activeTab = iota
	tabDiff
	tabWork
)

// diff compare modes
type diffMode int

const (
	diffAll      diffMode = iota // HEAD -> working tree (combined)
	diffStaged                   // HEAD -> index (staged only)
	diffWorktree                 // index -> working tree (unstaged only)
)

type changeItem struct {
	Path   string
	Status string // e.g., M/A/D/?? combined short
	Group  string // Staged / Unstaged / Untracked
}

type diffRow struct {
	Header string // non-empty means a header row (group title)
	Item   *changeItem
	Text   string
}

// work tab task item
type taskItem struct {
	Path         string
	Title        string
	Status       string
	Owner        string
	Priority     string
	Due          string
	RelatedSpecs []string
}

func initialModel() model {
	wd, _ := os.Getwd()
	root := wd
	if r, err := system.GitRoot(context.Background(), wd); err == nil && strings.TrimSpace(r) != "" {
		root = r
	}
	// build file table (left file manager)
	ft := table.New(
		table.WithColumns([]table.Column{{Title: "Files", Width: 32}}),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	ts := table.DefaultStyles()
	ts.Header = ts.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(uistyle.Vitesse.Border).
		BorderBottom(true).
		Bold(false).
		Padding(0, 0).
		Foreground(uistyle.Vitesse.Secondary).
		Background(uistyle.Vitesse.Bg)
	ts.Cell = ts.Cell.
		Padding(0, 0).
		Foreground(uistyle.Vitesse.Text).
		Background(uistyle.Vitesse.Bg)

	ts.Selected = ts.Selected.
		Foreground(uistyle.Vitesse.OnAccent).
		Background(uistyle.Vitesse.Primary).
		Bold(false)
	ft.SetStyles(ts)

	// input for conversation
	ti := textinput.New()
	ti.Placeholder = "输入对话（前缀 ! 执行命令），Esc 返回列表"
	ti.Prompt = "> "
	ti.CharLimit = 4096

	m := model{
		cwd:       wd,
		root:      root,
		page:      pageDetail,
		fileTable: ft,
		ti:        ti,
		logs:      make([]string, 0, 64),
		mdCache:   make(map[string]map[int]mdEntry),
	}
	// file manager root/cwd
	m.fmRoot = filepath.Join(root, "vibe-docs")
	m.fmCwd = m.fmRoot
	// initial focus on files
	m.focus = focusFiles
	m.fileTable.Focus()
	m.ti.Blur()
	// default tab from env (SPECUI_DEFAULT_TAB=explorer|diff|work)
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SPECUI_DEFAULT_TAB"))) {
	case "diff":
		m.tab = tabDiff
	case "work":
		m.tab = tabWork
	default:
		m.tab = tabExplorer
	}
	// init diff table
	dt := table.New(
		table.WithColumns([]table.Column{{Title: uistyle.IconDiff() + " Changes", Width: 32}}),
		table.WithHeight(20),
	)
	dts := table.DefaultStyles()
	dts.Header = dts.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(uistyle.Vitesse.Border).
		BorderBottom(true).
		Bold(false).
		Padding(0, 0).
		Foreground(uistyle.Vitesse.Secondary).
		Background(uistyle.Vitesse.Bg)
	dts.Cell = dts.Cell.
		Padding(0, 0).
		Foreground(uistyle.Vitesse.Text).
		Background(uistyle.Vitesse.Bg)
	dts.Selected = dts.Selected.
		Foreground(uistyle.Vitesse.OnAccent).
		Background(uistyle.Vitesse.Primary).
		Bold(false)
	dt.SetStyles(dts)
	m.diffTable = dt
	// detect git repo (best-effort)
	if r, err := system.GitRoot(context.Background(), wd); err == nil && strings.TrimSpace(r) != "" {
		m.gitInRepo = true
	}
	m.diffMode = diffAll
	m.diffFilterSpecOnly = false
	m.expanded = map[string]bool{}
	m.expanded[m.fmRoot] = true
	m.reloadTree()
	// If SPECUI_OPEN_PATH is set, try to preselect that file in Explorer tab
	if p := strings.TrimSpace(os.Getenv("SPECUI_OPEN_PATH")); p != "" {
		// ensure explorer tab and focus on files
		m.tab = tabExplorer
		m.setFocus(focusFiles)
		if idx := m.findVisibleIndexByPath(p); idx >= 0 {
			m.fileTable.SetCursor(idx)
			it := specItem{Path: p, Title: filepath.Base(p)}
			m.selected = &it
			m.mdVP.SetContent("渲染中…")
		}
	}
	// init Work tab task table and filters
	m.workTaskTable = table.New(
		table.WithColumns([]table.Column{
			{Title: "Task", Width: 36},
			{Title: "Status", Width: 10},
			{Title: "Owner", Width: 10},
			{Title: "Pri", Width: 4},
			{Title: "Due", Width: 10},
		}),
	)
	ws := table.DefaultStyles()
	ws.Selected = ws.Selected.
		Foreground(uistyle.Vitesse.OnAccent).
		Background(uistyle.Vitesse.Primary)
	m.workTaskTable.SetStyles(ws)
	m.workStatus = "All"
	m.workOwner = "All"
	m.workPriority = "All"
	return m
}

func (m model) Init() tea.Cmd { return tea.Batch(tickCmd(), startWatchCmd(m.root)) }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case watchStartedMsg:
		m.watcher = msg.w
		m.watchCh = msg.ch
		// subscribe to first event
		return m, watchSubscribeCmd(m.watchCh)
	case fileChangedMsg:
		// refresh tree and diff list; if work tab, reload tasks
		m.reloadTree()
		cmds := []tea.Cmd{checkDeltaCmd(), refreshDiffListCmd(m.root, m.diffFilterSpecOnly)}
		if m.tab == tabWork {
			cmds = append(cmds, loadTasksCmd(m.root, m.currentSelectedSpec()))
		}
		// resubscribe for more events
		cmds = append(cmds, watchSubscribeCmd(m.watchCh))
		return m, tea.Batch(cmds...)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// update table height (leave some room for header/help)
		m.recalcViewports()
		// adjust left tables height to match right box
		// Explorer/Diff right box overhead: header+divider (2)
		// Work right box overhead: header+filter+divider (3)
		desired := m.mdVP.Height + 2
		if m.tab == tabWork {
			desired = m.mdVP.Height + 3
		}
		if m.fileTable.Height() != desired {
			m.fileTable.SetHeight(desired)
		}
		if m.diffTable.Height() != m.diffVP.Height+2 {
			m.diffTable.SetHeight(m.diffVP.Height + 2)
		}
		// async re-render for new width
		if m.selected != nil {
			return m, renderMarkdownCmd(m.selected.Path, m.mdVP.Width, m.fastMode)
		}
		return m, nil
	case tea.MouseMsg:
		// Focus panes by clicking their zones
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			// tab bar clicks
			if zone.Get("spec.tab.explorer").InBounds(msg) {
				m.tab = tabExplorer
				m.setFocus(focusFiles)
				return m, nil
			}
			if zone.Get("spec.tab.diff").InBounds(msg) {
				m.tab = tabDiff
				m.setFocus(focusFiles)
				return m, tea.Batch(checkDeltaCmd(), refreshDiffListCmd(m.root, m.diffFilterSpecOnly))
			}
			if zone.Get("spec.tab.work").InBounds(msg) {
				m.tab = tabWork
				m.setFocus(focusFiles)
				return m, tea.Batch(loadTasksCmd(m.root, ""))
			}
			if zone.Get("spec.input").InBounds(msg) {
				m.setFocus(focusInput)
				return m, nil
			}
			if zone.Get("spec.files").InBounds(msg) {
				m.setFocus(focusFiles)
				// Map click Y to a visible row index. Account for left box
				// top border (1) + table header (1) + header divider (1).
				_, zy := zone.Get("spec.files").Pos(msg)
				row := zy - 3
				if row < 0 {
					return m, nil
				}
				// If the list is longer than the viewport, this approximation
				// selects from the top of the visible list. This works when
				// the list fits; for long lists we can later enhance by
				// custom-rendering rows with per-row zones.
				if row >= len(m.visible) {
					row = len(m.visible) - 1
				}
				if row < 0 {
					return m, nil
				}
				cur := m.fileTable.Cursor()
				if row != cur {
					// Move cursor to clicked row (approximate absolute index)
					m.fileTable.SetCursor(row)
					return m, nil
				}
				// Clicking the already-selected row: perform Enter behavior
				// (toggle dir open/collapse or open file)
				if len(m.visible) == 0 {
					return m, nil
				}
				if cur < 0 || cur >= len(m.visible) {
					return m, nil
				}
				node := m.visible[cur].Node
				if node.IsDir {
					m.expanded[node.Path] = !m.expanded[node.Path]
					m.reloadTree()
					m.fileTable.SetRows(m.visibleRows())
					// keep cursor at same index when possible
					if cur >= len(m.visible) {
						cur = len(m.visible) - 1
					}
					if cur >= 0 {
						m.fileTable.SetCursor(cur)
					}
					return m, nil
				}
				// Open file in preview
				it := specItem{Path: node.Path, Title: filepath.Base(node.Path)}
				m.selected = &it
				m.ti.Reset()
				m.ti.Blur()
				m.recalcViewports()
				m.statusMsg = "按 Esc 返回"
				m.mdVP.SetContent("渲染中…")
				return m, renderMarkdownCmd(node.Path, m.mdVP.Width, m.fastMode)
			}
			if zone.Get("spec.preview").InBounds(msg) {
				m.setFocus(focusPreview)
				return m, nil
			}
		}
		return m, nil
	case tea.KeyMsg:
		// when terminal has focus, forward keys directly to PTY
		if m.page == pageDetail && m.termMode && m.termFocus && m.pty != nil {
			// focus escape and app quit
			switch msg.String() {
			case "esc":
				// exit terminal focus back to input
				m.termFocus = false
				return m, nil
			case "ctrl+c":
				return m, writePTYCmd(m.pty, []byte{0x03})
			case "ctrl+l":
				return m, writePTYCmd(m.pty, []byte{0x0c})
			case "ctrl+z":
				return m, writePTYCmd(m.pty, []byte{0x1a})
			}
			if data := keyToPTYBytes(msg); len(data) > 0 {
				return m, writePTYCmd(m.pty, data)
			}
			return m, nil
		}
		// global quit when not in terminal focus
		if key := msg.String(); key == "ctrl+c" || key == "q" {
			return m, tea.Quit
		}
		// global tab switching
		switch msg.String() {
		case "1":
			m.tab = tabExplorer
			m.setFocus(focusFiles)
			return m, nil
		case "2":
			m.tab = tabDiff
			m.setFocus(focusFiles)
			return m, tea.Batch(checkDeltaCmd(), refreshDiffListCmd(m.root, m.diffFilterSpecOnly))
		case "3":
			m.tab = tabWork
			m.setFocus(focusFiles)
			return m, loadTasksCmd(m.root, "")
		case "tab":
			// cycle explorer -> diff -> work -> explorer
			switch m.tab {
			case tabExplorer:
				m.tab = tabDiff
				return m, tea.Batch(checkDeltaCmd(), refreshDiffListCmd(m.root, m.diffFilterSpecOnly))
			case tabDiff:
				m.tab = tabWork
				return m, loadTasksCmd(m.root, "")
			default:
				m.tab = tabExplorer
				return m, nil
			}
		}
		switch m.page {
		case pageDetail:
			switch msg.String() {
			case "j", "k":
				// Vim-style navigation
				if m.focus == focusFiles {
					switch m.tab {
					case tabExplorer:
						n := m.fileTable.Cursor()
						if msg.String() == "j" {
							n++
						} else {
							n--
						}
						if n < 0 {
							n = 0
						}
						if n >= len(m.visible) {
							n = len(m.visible) - 1
						}
						if n >= 0 && n != m.fileTable.Cursor() {
							m.fileTable.SetCursor(n)
							// auto-open markdown files on move
							node := m.visible[n].Node
							if !node.IsDir && isMarkdown(node.Path) {
								it := specItem{Path: node.Path, Title: filepath.Base(node.Path)}
								m.selected = &it
								m.ti.Reset()
								m.ti.Blur()
								m.recalcViewports()
								m.statusMsg = "按 Esc 返回"
								m.mdVP.SetContent("渲染中…")
								return m, renderMarkdownCmd(node.Path, m.mdVP.Width, m.fastMode)
							}
						}
						return m, nil
					case tabDiff:
						// Diff tab
						n := m.diffTable.Cursor()
						if msg.String() == "j" {
							n++
						} else {
							n--
						}
						if n < 0 {
							n = 0
						}
						if n >= len(m.diffItems) {
							n = len(m.diffItems) - 1
						}
						if n >= 0 && n != m.diffTable.Cursor() {
							m.diffTable.SetCursor(n)
							if it := m.diffItems[n].Item; it != nil {
								m.diffVP.SetContent("载入中…")
								return m, renderDiffCmd(m.root, it.Path, m.diffMode, m.diffVP.Width, m.hasDelta)
							}
						}
						return m, nil
					default:
						// Work tab (right pane is tasks table) — j/k navigate when right focused
						return m, nil
					}
				}
				if m.focus == focusPreview {
					switch m.tab {
					case tabExplorer:
						if msg.String() == "j" {
							m.mdVP.ScrollDown(1)
						} else {
							m.mdVP.ScrollUp(1)
						}
					case tabDiff:
						if msg.String() == "j" {
							m.diffVP.ScrollDown(1)
						} else {
							m.diffVP.ScrollUp(1)
						}
					default:
						// Work tab: no scrollable viewport on right
						return m, nil
					}
					return m, nil
				}
				return m, nil
			case "enter":
				// Only handle Enter for the left pane when it is focused.
				if m.focus == focusFiles && m.tab == tabExplorer {
					// open selection: dir -> toggle expand; file -> render
					if len(m.visible) == 0 {
						return m, nil
					}
					idx := m.fileTable.Cursor()
					if idx < 0 || idx >= len(m.visible) {
						return m, nil
					}
					node := m.visible[idx].Node
					if node.IsDir {
						m.expanded[node.Path] = !m.expanded[node.Path]
						m.reloadTree()
						// keep cursor position (clamped by table)
						if idx >= len(m.visible) {
							idx = len(m.visible) - 1
						}
						m.fileTable.SetRows(m.visibleRows())
						m.fileTable.SetCursor(idx)
						return m, nil
					}
					// render file
					it := specItem{Path: node.Path, Title: filepath.Base(node.Path)}
					m.selected = &it
					m.ti.Reset()
					m.ti.Blur()
					m.recalcViewports()
					m.statusMsg = "按 Esc 返回"
					m.mdVP.SetContent("渲染中…")
					cmds := []tea.Cmd{renderMarkdownCmd(node.Path, m.mdVP.Width, m.fastMode)}
					if m.tab == tabWork {
						cmds = append(cmds, loadTasksCmd(m.root, m.currentSelectedSpec()))
					}
					return m, tea.Batch(cmds...)
				}
				if m.tab == tabWork && m.focus == focusPreview && msg.String() == "enter" {
					idx := m.workTaskTable.Cursor()
					if idx >= 0 && idx < len(m.workFiltered) {
						return m, openInOSCmd(m.workFiltered[idx].Path)
					}
					return m, nil
				}
				if m.focus == focusFiles && m.tab == tabDiff {
					// open diff for selected change item
					if len(m.diffItems) == 0 {
						return m, nil
					}
					idx := m.diffTable.Cursor()
					if idx < 0 || idx >= len(m.diffItems) {
						return m, nil
					}
					row := m.diffItems[idx]
					if row.Item == nil {
						return m, nil
					}
					// trigger diff render and move focus to preview
					m.diffVP.SetContent("载入中…")
					m.diffCurrentFile = row.Item.Path
					if fi, err := os.Stat(filepath.Join(m.root, row.Item.Path)); err == nil {
						m.diffFileModUnix = fi.ModTime().Unix()
						m.diffFileSize = fi.Size()
					}
					m.setFocus(focusPreview)
					return m, renderDiffCmd(m.root, row.Item.Path, m.diffMode, m.diffVP.Width, m.hasDelta)
				}
				// If input is focused, let the input handler process Enter below.
			case "left", "backspace":
				// Only treat Left/Backspace as tree navigation when file pane focused.
				if m.focus == focusFiles && m.tab == tabExplorer {
					// collapse dir or move to parent
					if len(m.visible) == 0 {
						return m, nil
					}
					idx := m.fileTable.Cursor()
					if idx < 0 || idx >= len(m.visible) {
						return m, nil
					}
					node := m.visible[idx].Node
					if node.IsDir && m.expanded[node.Path] {
						m.expanded[node.Path] = false
						m.reloadTree()
						m.fileTable.SetRows(m.visibleRows())
						m.fileTable.SetCursor(idx)
						return m, nil
					}
					if node.Parent != nil {
						parent := node.Parent
						// ensure parent visible
						m.expanded[parent.Path] = true
						m.reloadTree()
						for i, r := range m.visible {
							if r.Node == parent {
								m.fileTable.SetCursor(i)
								break
							}
						}
					}
					return m, nil
				}
				// In Diff tab: Left cycles diff mode backwards; Backspace ignored
				if m.focus == focusFiles && m.tab == tabDiff && msg.String() == "left" {
					if m.diffMode == diffAll {
						m.diffMode = diffWorktree
					} else {
						m.diffMode--
					}
					idx := m.diffTable.Cursor()
					if idx >= 0 && idx < len(m.diffItems) {
						if it := m.diffItems[idx].Item; it != nil {
							m.diffVP.SetContent("载入中…")
							return m, renderDiffCmd(m.root, it.Path, m.diffMode, m.diffVP.Width, m.hasDelta)
						}
					}
					return m, nil
				}
				// If input is focused, allow textinput to receive Backspace/Left.
			case "tab":
				// terminal focus toggle disabled (no terminal binding)
				return m, nil
			case "ctrl+h":
				// Avoid stealing common Backspace mapping (Ctrl+H) from input
				if m.focus != focusInput {
					m.setFocus(focusFiles)
					return m, nil
				}
				// let input handle as delete when focused
			case "ctrl+l":
				if m.focus != focusInput {
					m.setFocus(focusPreview)
					return m, nil
				}
				// let input handle when focused
			case "ctrl+j":
				if m.focus == focusFiles || m.focus == focusPreview {
					m.lastTopFocus = m.focus
				}
				m.setFocus(focusInput)
				return m, nil
			case "ctrl+k":
				if m.lastTopFocus != focusFiles && m.lastTopFocus != focusPreview {
					m.lastTopFocus = focusFiles
				}
				m.setFocus(m.lastTopFocus)
				return m, nil
			case "esc":
				// ensure focus stays on file manager
				m.ti.Reset()
				m.ti.Blur()
				m.statusMsg = ""
				return m, nil
			case "r":
				// reload based on tab
				if m.tab == tabExplorer {
					if m.selected != nil {
						m.mdVP.SetContent("渲染中…")
						return m, renderMarkdownCmd(m.selected.Path, m.mdVP.Width, m.fastMode)
					}
					return m, nil
				}
				// Diff: refresh list and current diff
				cmds := []tea.Cmd{refreshDiffListCmd(m.root, m.diffFilterSpecOnly)}
				idx := m.diffTable.Cursor()
				if idx >= 0 && idx < len(m.diffItems) {
					if it := m.diffItems[idx].Item; it != nil {
						m.diffCurrentFile = it.Path
						if fi, err := os.Stat(filepath.Join(m.root, it.Path)); err == nil {
							m.diffFileModUnix = fi.ModTime().Unix()
							m.diffFileSize = fi.Size()
						}
						cmds = append(cmds, renderDiffCmd(m.root, it.Path, m.diffMode, m.diffVP.Width, m.hasDelta))
					}
				}
				return m, tea.Batch(cmds...)
			case "f":
				// Explorer: toggle fast markdown mode; Diff: toggle spec-only filter
				if m.tab == tabExplorer { //nolint:staticcheck // keep explicit if-else for readability
					m.fastMode = !m.fastMode
					if m.selected != nil {
						m.mdVP.SetContent("渲染中…")
						return m, renderMarkdownCmd(m.selected.Path, m.mdVP.Width, m.fastMode)
					}
					return m, nil
				}
				m.diffFilterSpecOnly = !m.diffFilterSpecOnly
				return m, refreshDiffListCmd(m.root, m.diffFilterSpecOnly)
			case "/":
				if m.tab == tabWork {
					m.setFocus(focusInput)
					m.ti.Placeholder = "搜索任务…回车提交"
					return m, nil
				}
			case "s":
				if m.tab == tabWork {
					m.workStatus = cycle([]string{"All", "backlog", "in-progress", "blocked", "done", "draft", "accepted"}, m.workStatus)
					m.applyWorkFilter()
					return m, nil
				}
			case "o":
				if m.tab == tabWork {
					if len(m.workOwnerList) == 0 {
						m.workOwnerList = []string{"All"}
					}
					m.workOwnerIdx = (m.workOwnerIdx + 1) % len(m.workOwnerList)
					m.workOwner = m.workOwnerList[m.workOwnerIdx]
					m.applyWorkFilter()
					return m, nil
				}
			case "p":
				if m.tab == tabWork {
					m.workPriority = cycle([]string{"All", "P0", "P1", "P2"}, m.workPriority)
					m.applyWorkFilter()
					return m, nil
				}
			case "right":
				if m.tab == tabDiff {
					if m.diffMode == diffWorktree {
						m.diffMode = diffAll
					} else {
						m.diffMode++
					}
					idx := m.diffTable.Cursor()
					if idx >= 0 && idx < len(m.diffItems) {
						if it := m.diffItems[idx].Item; it != nil {
							m.diffVP.SetContent("载入中…")
							m.diffCurrentFile = it.Path
							if fi, err := os.Stat(filepath.Join(m.root, it.Path)); err == nil {
								m.diffFileModUnix = fi.ModTime().Unix()
								m.diffFileSize = fi.Size()
							}
							return m, renderDiffCmd(m.root, it.Path, m.diffMode, m.diffVP.Width, m.hasDelta)
						}
					}
					return m, nil
				}
			case "t":
				// terminal toggle disabled (no terminal binding)
				return m, nil
			}
			// input handling
			if msg.Type == tea.KeyEnter && m.ti.Focused() {
				val := strings.TrimSpace(m.ti.Value())
				// one-shot shell command: prefix with '!'
				if strings.HasPrefix(val, "!") {
					cmdline := strings.TrimSpace(strings.TrimPrefix(val, "!"))
					m.ti.SetValue("")
					if cmdline == "" {
						return m, nil
					}
					// echo the command into the log
					m.logs = append(m.logs, "> "+cmdline)
					m.logVP.SetContent(strings.Join(m.logs, "\n"))
					m.logVP.GotoBottom()
					return m, runShellCmd(m.cwd, cmdline, 20*time.Second)
				}
				// chat mode: append to log
				if val != "" {
					stamp := time.Now().Format("15:04:05")
					m.logs = append(m.logs, fmt.Sprintf("[%s] %s", stamp, val))
					// update log viewport
					m.logVP.SetContent(strings.Join(m.logs, "\n"))
					m.logVP.GotoBottom()
				}
				m.ti.SetValue("")
				return m, nil
			}
			var cmds []tea.Cmd
			var cmd tea.Cmd
			// route updates based on focus
			if m.focus == focusFiles {
				//nolint:staticcheck // prefer explicit if-else here
				if m.tab == tabExplorer {
					prev := m.fileTable.Cursor()
					m.fileTable, cmd = m.fileTable.Update(msg)
					cmds = append(cmds, cmd)
					// When moving selection, auto-open markdown files on the right
					cur := m.fileTable.Cursor()
					if cur != prev && cur >= 0 && cur < len(m.visible) {
						node := m.visible[cur].Node
						if !node.IsDir && isMarkdown(node.Path) {
							it := specItem{Path: node.Path, Title: filepath.Base(node.Path)}
							m.selected = &it
							m.ti.Reset()
							m.ti.Blur()
							m.recalcViewports()
							m.statusMsg = "按 Esc 返回"
							m.mdVP.SetContent("渲染中…")
							cmds = append(cmds, renderMarkdownCmd(node.Path, m.mdVP.Width, m.fastMode))
							return m, tea.Batch(cmds...)
						}
					}
				} else if m.tab == tabDiff {
					prev := m.diffTable.Cursor()
					m.diffTable, cmd = m.diffTable.Update(msg)
					cmds = append(cmds, cmd)
					cur := m.diffTable.Cursor()
					if cur != prev && cur >= 0 && cur < len(m.diffItems) {
						if it := m.diffItems[cur].Item; it != nil {
							m.diffVP.SetContent("载入中…")
							m.diffCurrentFile = it.Path
							if fi, err := os.Stat(filepath.Join(m.root, it.Path)); err == nil {
								m.diffFileModUnix = fi.ModTime().Unix()
								m.diffFileSize = fi.Size()
							}
							cmds = append(cmds, renderDiffCmd(m.root, it.Path, m.diffMode, m.diffVP.Width, m.hasDelta))
							return m, tea.Batch(cmds...)
						}
					} else {
						// Work tab: left is file table as well
						prev := m.fileTable.Cursor()
						m.fileTable, cmd = m.fileTable.Update(msg)
						cmds = append(cmds, cmd)
						cur := m.fileTable.Cursor()
						if cur != prev {
							// selection changed -> re-apply task filter
							m.applyWorkFilter()
							return m, tea.Batch(cmds...)
						}
					}
				}
			}
			if m.focus == focusInput {
				m.ti, cmd = m.ti.Update(msg)
				cmds = append(cmds, cmd)
			}
			if m.focus == focusPreview {
				if m.tab == tabExplorer { //nolint:staticcheck
					m.mdVP, cmd = m.mdVP.Update(msg)
					cmds = append(cmds, cmd)
				} else if m.tab == tabDiff {
					m.diffVP, cmd = m.diffVP.Update(msg)
					cmds = append(cmds, cmd)
				} else {
					m.workTaskTable, cmd = m.workTaskTable.Update(msg)
					cmds = append(cmds, cmd)
				}
			}
			return m, tea.Batch(cmds...)
		}
	case tickMsg:
		m.now = time.Time(msg)
		// Periodically ensure preview reflects on-disk changes.
		// If the selected file's modtime/size changed, re-render asynchronously.
		if m.page == pageDetail && m.tab == tabExplorer && m.selected != nil {
			path := m.selected.Path
			width := m.mdVP.Width
			if width <= 0 {
				width = 80
			}
			if fi, err := os.Stat(path); err == nil {
				mod := fi.ModTime().Unix()
				size := fi.Size()
				var have bool
				var cached mdEntry
				if wm, ok := m.mdCache[path]; ok {
					if e, ok2 := wm[width]; ok2 {
						cached = e
						have = true
					} else {
						// fallback: any cached width for this path
						for _, e := range wm {
							cached = e
							have = true
							break
						}
					}
				}
				if !have || cached.modUnix != mod || cached.size != size {
					// Kick a render; also keep ticking clock
					return m, tea.Batch(tickCmd(), renderMarkdownCmd(path, width, m.fastMode))
				}
			}
		}
		// Auto-refresh diff preview when current file changes on disk.
		if m.page == pageDetail && m.tab == tabDiff && m.diffCurrentFile != "" {
			abs := filepath.Join(m.root, m.diffCurrentFile)
			if fi, err := os.Stat(abs); err == nil {
				mod := fi.ModTime().Unix()
				size := fi.Size()
				if mod != m.diffFileModUnix || size != m.diffFileSize {
					m.diffFileModUnix, m.diffFileSize = mod, size
					return m, tea.Batch(tickCmd(), renderDiffCmd(m.root, m.diffCurrentFile, m.diffMode, m.diffVP.Width, m.hasDelta))
				}
			}
		}

		// Periodic fallback when fsnotify is unavailable
		if m.page == pageDetail && m.treeRoot != nil && m.watcher == nil {
			newStamp, newCount := m.computeTreeFingerprint()
			if newStamp != m.treeStamp || newCount != m.treeCount {
				// try to preserve cursor by path
				cur := m.fileTable.Cursor()
				var curPath string
				if cur >= 0 && cur < len(m.visible) {
					if n := m.visible[cur].Node; n != nil {
						curPath = n.Path
					}
				}
				m.reloadTree()
				if curPath != "" {
					if idx := m.findVisibleIndexByPath(curPath); idx >= 0 {
						m.fileTable.SetCursor(idx)
					}
				}
			}
		}
		return m, tickCmd()
	case tasksLoadedMsg:
		m.workTasks = msg.items
		// owners
		set := map[string]struct{}{}
		for _, t := range m.workTasks {
			if strings.TrimSpace(t.Owner) != "" {
				set[t.Owner] = struct{}{}
			}
		}
		m.workOwnerList = make([]string, 0, len(set)+1)
		m.workOwnerList = append(m.workOwnerList, "All")
		for o := range set {
			m.workOwnerList = append(m.workOwnerList, o)
		}
		sort.Strings(m.workOwnerList[1:])
		m.workOwnerIdx = 0
		m.applyWorkFilter()
		return m, nil
	case diffListMsg:
		if msg.Err != "" {
			m.diffErr = msg.Err
			m.diffItems = nil
			m.diffTable.SetRows(nil)
			return m, nil
		}
		m.diffErr = ""
		m.diffItems = buildDiffRows(msg.Items)
		m.diffTable.SetRows(diffRowsToTable(m.diffItems))
		// clamp cursor
		if c := m.diffTable.Cursor(); c >= len(m.diffItems) {
			m.diffTable.SetCursor(len(m.diffItems) - 1)
		}
		return m, nil
	case diffRenderedMsg:
		if msg.Err != "" {
			m.diffVP.SetContent("渲染失败：" + msg.Err)
			return m, nil
		}
		m.diffVP.SetContent(trimEdgeBlankLines(msg.Out))
		return m, nil
	case deltaCheckMsg:
		m.hasDelta = bool(msg)
		return m, nil
	case ptyStartErrMsg:
		m.logs = append(m.logs, "[pty error] "+msg.Err)
		m.logVP.SetContent(strings.Join(m.logs, "\n"))
		m.termMode = false
		return m, nil
	case ptyStartedMsg:
		// initialize VT emulator for right pane
		cols, rows := m.termSize()
		emu := vt.NewEmulator(cols, rows)
		m.pty = msg.Pty
		m.termVT = emu
		// kick off first PTY read and render tick
		return m, tea.Batch(readPTYOnceCmd(m.pty), termTickCmd())
	case ptyChunkMsg:
		if m.termVT != nil && len(msg.Data) > 0 {
			data := stripOSCBytesState(msg.Data, &m.oscPending)
			if len(data) > 0 {
				_, _ = m.termVT.Write(data)
			}
			// mark dirty to re-render into viewport
			m.termDirty = true
		}
		// schedule next read while PTY exists
		if m.pty != nil {
			return m, readPTYOnceCmd(m.pty)
		}
		return m, nil
	case termRenderTickMsg:
		if m.termMode && m.pty != nil {
			if m.termVT != nil && (m.termDirty || m.termFocus) {
				m.logVP.SetContent(renderVTRightPane(&m))
				m.termDirty = false
			}
			// continue ticking
			return m, termTickCmd()
		}
		return m, nil
	case termDoneMsg:
		// legacy one-shot command result (kept for fallback)
		m.cmdRunning = false
		if strings.TrimSpace(msg.Out) != "" {
			outs := strings.Split(strings.ReplaceAll(msg.Out, "\r\n", "\n"), "\n")
			for _, ln := range outs {
				if ln != "" {
					m.logs = append(m.logs, ln)
				}
			}
		}
		if msg.Exit != 0 {
			m.logs = append(m.logs, fmt.Sprintf("[exit %d]", msg.Exit))
		}
		m.logVP.SetContent(strings.Join(m.logs, "\n"))
		m.logVP.GotoBottom()
		return m, nil
	case renderDoneMsg:
		// apply only if still on same file and width
		if m.selected != nil && m.selected.Path == msg.Path {
			if msg.Err != "" {
				m.mdVP.SetContent(fmt.Sprintf("读取/渲染失败：%s", msg.Err))
				return m, nil
			}
			// Always show latest render result, even if width changed meanwhile.
			// If width mismatched, schedule a re-render with current width to reflow.
			m.mdVP.SetContent(msg.Out)
			if _, ok := m.mdCache[msg.Path]; !ok {
				m.mdCache[msg.Path] = make(map[int]mdEntry)
			}
			m.mdCache[msg.Path][msg.Width] = mdEntry{out: msg.Out, modUnix: msg.ModUnix, size: msg.Size}
			if m.mdVP.Width != msg.Width {
				return m, renderMarkdownCmd(msg.Path, m.mdVP.Width, m.fastMode)
			}
		}
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	switch m.page {
	case pageSelect:
		// legacy select page; unused with file manager
		return ""
	case pageDetail:
		// choose styles; highlight border of focused pane
		leftBox := boxStyle
		rightBox := boxStyle
		inputBox := boxStyle
		// Remove inner horizontal padding on left/right panes so width is stable regardless of focus
		leftBox = leftBox.Padding(0, 0)
		rightBox = rightBox.Padding(0, 0)
		switch m.focus {
		case focusFiles:
			leftBox = boxStyleFocus
		case focusPreview:
			rightBox = boxStyleFocus
		case focusInput:
			inputBox = boxStyleFocus
		}
		// Ensure focused pane styles also keep zero inner padding (avoid width jitter)
		leftBox = leftBox.Padding(0, 0)
		rightBox = rightBox.Padding(0, 0)
		// top: split left/right; plus a tabs bar above
		lw := m.leftW
		rw := m.rightW
		// Fixed-width render to avoid jitter when selection changes
		if lw < 3 {
			lw = 3
		}
		if rw < 3 {
			rw = 3
		}
		var left string
		var right string
		if m.tab == tabExplorer { //nolint:staticcheck
			left = leftBox.Width(lw).Render(m.fileTable.View())
			left = zone.Mark("spec.files", left)
			// right: header with filename + divider + markdown viewport
			var fname string
			if m.selected != nil {
				fname = relFrom(m.root, m.selected.Path)
			} else {
				fname = "(未选择文件)"
			}
			// divider under filename sized to preview width; clip long names
			sepWidth := rw - 2
			if sepWidth <= 0 {
				sepWidth = 1
			}
			clipW := sepWidth - 2
			if clipW < 1 {
				clipW = sepWidth
			}
			nameClipped := fname
			if xansi.StringWidth(nameClipped) > clipW {
				nameClipped = xansi.Truncate(nameClipped, clipW, "…")
			}
			divider := lipgloss.NewStyle().Foreground(uistyle.Vitesse.Border).Render(strings.Repeat("─", sepWidth))
			rightInner := lipgloss.JoinVertical(lipgloss.Left,
				headerStyle.Render(" "+nameClipped+" "),
				divider,
				m.mdVP.View(),
			)
			right = rightBox.Width(rw).Render(rightInner)
			right = zone.Mark("spec.preview", right)
		} else if m.tab == tabDiff {
			// Diff tab
			left = leftBox.Width(lw).Render(m.diffTable.View())
			left = zone.Mark("spec.files", left)
			// right: header shows mode/delta
			sepWidth := rw - 2
			if sepWidth <= 0 {
				sepWidth = 1
			}
			title := fmt.Sprintf(" %s Diff — %s — Delta:%s ", uistyle.IconDiff(), diffModeLabel(m.diffMode), onOff(m.hasDelta))
			divider := lipgloss.NewStyle().Foreground(uistyle.Vitesse.Border).Render(strings.Repeat("─", sepWidth))
			rightInner := lipgloss.JoinVertical(lipgloss.Left,
				headerStyle.Render(title),
				divider,
				m.diffVP.View(),
			)
			right = rightBox.Width(rw).Render(rightInner)
			right = zone.Mark("spec.preview", right)
		} else {
			// Work tab: left = fileTable; right = tasks table with filter chips
			left = leftBox.Width(lw).Render(m.fileTable.View())
			left = zone.Mark("spec.files", left)
			// title and filter line
			sepWidth := rw - 2
			if sepWidth <= 0 {
				sepWidth = 1
			}
			// build filter chips
			filter := m.renderWorkFilters()
			divider := lipgloss.NewStyle().Foreground(uistyle.Vitesse.Border).Render(strings.Repeat("─", sepWidth))
			rightInner := lipgloss.JoinVertical(lipgloss.Left,
				headerStyle.Render(" Tasks "),
				filter,
				divider,
				m.workTaskTable.View(),
			)
			right = rightBox.Width(rw).Render(rightInner)
			right = zone.Mark("spec.preview", right)
		}
		// tabs bar on top
		tabs := m.renderTabs()
		top := tabs + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, left, right)
		// bottom: input and a lipgloss-rendered work bar
		bottomInput := inputBox.Render(m.ti.View())
		bottom := zone.Mark("spec.input", bottomInput) + "\n" + m.renderWorkbar()
		out := lipgloss.JoinVertical(lipgloss.Left, top, bottom)
		// When the input is focused, move the REAL terminal cursor to the
		// input caret line so OS IME candidate windows anchor correctly.
		// We approximate the column using prompt + visible value width.
		if m.focus == focusInput {
			// Move cursor: up 2 lines (from status bar to input content row),
			// CR to column 1, then forward past border + left pad + prompt.
			// Note: boxStyle has Padding(0,1) so we add one space after the
			// left border. This positioning is intentionally conservative; it
			// avoids drawing past the right border.
			// Calculate prompt and value widths (display width).
			pw := xansi.StringWidth(m.ti.Prompt)
			vw := xansi.StringWidth(m.ti.Value())
			// Max visible content inside textinput width minus prompt
			maxVisible := m.ti.Width - pw
			if maxVisible < 0 {
				maxVisible = 0
			}
			if vw > maxVisible {
				vw = maxVisible
			}
			// base offset: border(1) + left pad(1)
			base := 2
			col := base + pw + vw
			if col < 0 {
				col = 0
			}
			// emit ANSI to show cursor, move up two lines, CR, and move right
			// by computed columns.
			out += "\x1b[?25h\x1b[2A\r"
			if col > 0 {
				out += fmt.Sprintf("\x1b[%dC", col)
			}
		} else {
			// Hide the real cursor when not in input to prevent stray IME anchors
			out += "\x1b[?25l"
		}
		// Zone scan strips markers and records positions.
		return zone.Scan(out)
	default:
		return ""
	}
}

var (
	boxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(uistyle.Vitesse.Border).
			Background(uistyle.Vitesse.Bg).
			Padding(0, 1)
	boxStyleFocus = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(uistyle.Vitesse.Primary).
			Background(uistyle.Vitesse.Bg).
			Padding(0, 1)
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(uistyle.Vitesse.Text)
)

func (m *model) recalcViewports() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	// Reserve bottom for input (3 lines box) + status bar (1 line) = 4 lines
	bottomH := 4
	if bottomH >= m.height {
		bottomH = 1
	}
	topH := m.height - bottomH
	if topH < 3 {
		topH = 3
	}

	// top split: left (file list) fixed width, right (markdown) rest
	innerW := m.width - 2 // borders padding approx
	if innerW < 20 {
		innerW = m.width
	}
	const fileTreeFixed = 36
	lw := fileTreeFixed
	// Ensure right pane keeps at least 20 cols; clamp when terminal is narrow
	if lw > innerW-20 {
		if innerW-20 > 20 {
			lw = innerW - 20
		} else {
			lw = 20
		}
	}
	if lw < 20 {
		lw = 20
	}
	rw := innerW - lw
	// cache for stable rendering widths (total pane widths including borders)
	m.leftW = lw
	m.rightW = rw
	if lw < 20 {
		lw = 20
	}
	if rw < 20 {
		rw = 20
	}
	// Adjust for lipgloss border padding by setting slightly smaller dimensions
	// left (file list) uses table height directly; right is viewport content
	// Deduct: 1 line for tabs bar + 2 lines for right header/divider inside box
	mdW, mdH := rw-2, topH-3
	if mdH > 2 {
		mdH -= 2
	} else if mdH > 0 {
		mdH = 1
	}
	if mdW < 10 {
		mdW = lw
	}
	if mdH < 3 {
		mdH = topH
	}
	if m.mdVP.Width == 0 && m.mdVP.Height == 0 {
		m.mdVP = viewport.New(mdW, mdH)
	} else {
		m.mdVP.Width = mdW
		m.mdVP.Height = mdH
	}
	if m.diffVP.Width == 0 && m.diffVP.Height == 0 {
		m.diffVP = viewport.New(mdW, mdH)
	} else {
		m.diffVP.Width = mdW
		m.diffVP.Height = mdH
	}
	// work tab right table height aligns to mdH
	if m.workTaskTable.Height() != mdH {
		m.workTaskTable.SetHeight(mdH)
	}
	// viewport Y position left default; header/divider are inside the right pane
	// Sync file table column width and viewport width with left pane width so
	// that selection highlight covers the whole line and truncation is reasonable.
	colW := lw - 2 // account for left/right borders; left pane padding set to 0
	if colW < 10 {
		if lw-2 > 10 {
			colW = lw - 2
		} else {
			colW = 10
		}
	}
	// Only one column: Files / Changes
	m.fileTable.SetColumns([]table.Column{{Title: "Files", Width: colW}})
	m.fileTable.SetWidth(colW)
	m.diffTable.SetColumns([]table.Column{{Title: "Changes", Width: colW}})
	m.diffTable.SetWidth(colW)
	// Work tab right table height adjusts implicitly in view; set width hint via columns
	// Split right pane in half for rough task name width budget
	half := (rw - 4) / 2
	if half < 16 {
		half = 16
	}
	m.workTaskTable.SetColumns([]table.Column{
		{Title: "Task", Width: half}, {Title: "Status", Width: 10}, {Title: "Owner", Width: 10}, {Title: "Pri", Width: 4}, {Title: "Due", Width: 10},
	})
	// log viewport unused in this layout

	// input width adjust
	m.ti.Width = m.width - 6

	// viewport-only adjustments; caller decides whether to rerender
}

// termSize returns terminal columns and rows for the right pane
func (m model) termSize() (cols, rows int) {
	cols = m.logVP.Width
	rows = m.logVP.Height
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	return
}

// buildRenderer removed (unused)

// Background render command and message
type renderDoneMsg struct {
	Path    string
	Width   int
	Out     string
	Err     string
	ModUnix int64
	Size    int64
}

const fastThresholdBytes = 64 * 1024 // 64KB (aggressive for snappier UI)

// vitesseGlamour returns a glamour ANSI style config adapted to the
// centralized Vitesse theme used by the main UI.
func vitesseGlamour() ansi.StyleConfig {
	// helper: take lipgloss.Color -> hex without alpha
	hex := func(c lipgloss.Color) string {
		s := string(c)
		if strings.HasPrefix(s, "#") && len(s) == 9 { // #RRGGBBAA
			return s[:7]
		}
		return s
	}
	sp := func(s string) *string { return &s }
	bp := func(b bool) *bool { return &b }

	// palette
	text := hex(uistyle.Vitesse.Text)
	secondary := hex(uistyle.Vitesse.Secondary)
	muted := hex(uistyle.Vitesse.Muted)
	primary := hex(uistyle.Vitesse.Primary)
	blue := hex(uistyle.Vitesse.Blue)
	yellow := hex(uistyle.Vitesse.Yellow)
	magenta := hex(uistyle.Vitesse.Magenta)
	red := hex(uistyle.Vitesse.Red)
	bg := hex(uistyle.Vitesse.Bg)
	bgSoft := hex(uistyle.Vitesse.BgSoft)

	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: sp(text), BackgroundColor: sp(bg)},
		},
		Paragraph: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: sp(text)},
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: sp(secondary), Italic: bp(true)},
		},
		// Markdown headings use theme blue consistently
		Heading: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: sp(blue), Bold: bp(true)}},
		H1:      ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: sp(blue), Bold: bp(true)}},
		H2:      ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: sp(blue), Bold: bp(true)}},
		H3:      ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: sp(blue), Bold: bp(true)}},
		H4:      ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: sp(blue), Bold: bp(true)}},
		H5:      ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: sp(blue), Bold: bp(true)}},
		H6:      ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: sp(blue), Bold: bp(true)}},

		Text:           ansi.StylePrimitive{Color: sp(text)},
		Emph:           ansi.StylePrimitive{Italic: bp(true)},
		Strong:         ansi.StylePrimitive{Bold: bp(true)},
		Strikethrough:  ansi.StylePrimitive{CrossedOut: bp(true)},
		HorizontalRule: ansi.StylePrimitive{Color: sp(secondary)},

		Link:     ansi.StylePrimitive{Color: sp(blue), Underline: bp(true)},
		LinkText: ansi.StylePrimitive{Color: sp(blue), Underline: bp(true)},

		Code: ansi.StyleBlock{ // inline code
			StylePrimitive: ansi.StylePrimitive{Color: sp(yellow), BackgroundColor: sp(bgSoft)},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{Color: sp(text), BackgroundColor: sp(bgSoft)},
			},
			// Basic chroma mapping tuned to Vitesse accents
			Chroma: &ansi.Chroma{
				Text:              ansi.StylePrimitive{Color: sp(text)},
				Comment:           ansi.StylePrimitive{Color: sp(muted), Italic: bp(true)},
				Keyword:           ansi.StylePrimitive{Color: sp(primary), Bold: bp(true)},
				NameFunction:      ansi.StylePrimitive{Color: sp(blue)},
				NameBuiltin:       ansi.StylePrimitive{Color: sp(magenta)},
				LiteralString:     ansi.StylePrimitive{Color: sp(yellow)},
				LiteralNumber:     ansi.StylePrimitive{Color: sp(magenta)},
				NameAttribute:     ansi.StylePrimitive{Color: sp(blue)},
				Operator:          ansi.StylePrimitive{Color: sp(secondary)},
				Punctuation:       ansi.StylePrimitive{Color: sp(secondary)},
				GenericDeleted:    ansi.StylePrimitive{Color: sp(red)},
				GenericInserted:   ansi.StylePrimitive{Color: sp(primary)},
				GenericStrong:     ansi.StylePrimitive{Bold: bp(true)},
				GenericSubheading: ansi.StylePrimitive{Color: sp(secondary)},
				Background:        ansi.StylePrimitive{BackgroundColor: sp(bgSoft)},
			},
		},

		Table: ansi.StyleTable{
			StyleBlock:      ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: sp(text)}},
			CenterSeparator: sp("│"),
			ColumnSeparator: sp("│"),
			RowSeparator:    sp("─"),
		},

		DefinitionTerm:        ansi.StylePrimitive{Bold: bp(true)},
		DefinitionDescription: ansi.StylePrimitive{Color: sp(secondary)},
	}
}

func renderMarkdownCmd(path string, width int, forceFast bool) tea.Cmd {
	return func() tea.Msg {
		fi, statErr := os.Stat(path)
		b, err := os.ReadFile(path)
		if err != nil {
			return renderDoneMsg{Path: path, Width: width, Err: err.Error()}
		}
		content := stripFrontmatter(string(b))
		var modUnix int64
		var size int64
		if statErr == nil && fi != nil {
			modUnix = fi.ModTime().Unix()
			size = fi.Size()
		}
		fast := forceFast || len(b) >= fastThresholdBytes
		if fast {
			return renderDoneMsg{Path: path, Width: width, Out: trimEdgeBlankLines(content), ModUnix: modUnix, Size: size}
		}
		// Match demo rendering: subtract Glamour gutter from wrap width
		const glamourGutter = 2
		wrap := width - glamourGutter
		if wrap < 10 {
			wrap = 10
		}
		r, _ := glamour.NewTermRenderer(
			glamour.WithStyles(vitesseGlamour()),
			glamour.WithWordWrap(wrap),
		)
		if out, err := r.Render(content); err == nil {
			return renderDoneMsg{Path: path, Width: width, Out: trimEdgeBlankLines(out), ModUnix: modUnix, Size: size}
		}
		return renderDoneMsg{Path: path, Width: width, Out: trimEdgeBlankLines(content), ModUnix: modUnix, Size: size}
	}
}

// terminal command result
type termDoneMsg struct {
	Out  string
	Exit int
}

func runShellCmd(cwd string, line string, timeout time.Duration) tea.Cmd {
	return func() tea.Msg {
		sh := os.Getenv("SHELL")
		var cmd *exec.Cmd
		if sh != "" {
			cmd = exec.Command(sh, "-lc", line)
		} else {
			// fallback
			if _, err := exec.LookPath("bash"); err == nil {
				cmd = exec.Command("bash", "-lc", line)
			} else {
				cmd = exec.Command("sh", "-lc", line)
			}
		}
		cmd.Dir = cwd
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
		cmd.Dir = cwd
		out, err := cmd.CombinedOutput()
		exit := 0
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				if ws, ok2 := ee.Sys().(syscall.WaitStatus); ok2 {
					exit = ws.ExitStatus()
				} else {
					exit = 1
				}
			} else if ctx.Err() == context.DeadlineExceeded {
				exit = 124
			} else {
				exit = 1
			}
		}
		return termDoneMsg{Out: string(out), Exit: exit}
	}
}

// PTY startup message
type ptyStartedMsg struct {
	Pty  xpty.Pty
	Cols int
	Rows int
}

type ptyStartErrMsg struct{ Err string }

type ptyChunkMsg struct{ Data []byte }

// startPTYCmd starts a persistent shell on a PTY with given size
// startPTYCmd removed (unused)

// schedule a single PTY read
func readPTYOnceCmd(p xpty.Pty) tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, 4096)
		n, err := p.Read(buf)
		if n > 0 {
			return ptyChunkMsg{Data: buf[:n]}
		}
		if err != nil {
			return termDoneMsg{Out: err.Error(), Exit: 0}
		}
		return nil
	}
}

// write to PTY
func writePTYCmd(p xpty.Pty, data []byte) tea.Cmd {
	return func() tea.Msg {
		_, _ = p.Write(data)
		return nil
	}
}

// (no VT input pumping; keys write directly to PTY)
// --- Diff integration commands & helpers ---

// diff messages
type diffListMsg struct {
	Items []changeItem
	Err   string
}
type diffRenderedMsg struct {
	Out string
	Err string
}
type deltaCheckMsg bool

// check if delta exists in PATH
func checkDeltaCmd() tea.Cmd {
	return func() tea.Msg { _, err := exec.LookPath("delta"); return deltaCheckMsg(err == nil) }
}

// refreshDiffListCmd lists changed/untracked files via porcelain -z
func refreshDiffListCmd(root string, onlySpec bool) tea.Cmd {
	return func() tea.Msg {
		if _, err := exec.LookPath("git"); err != nil {
			return diffListMsg{Err: "未检测到 Git"}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "git", "-C", root, "status", "--porcelain=v1", "-z")
		out, err := cmd.Output()
		if err != nil {
			return diffListMsg{Err: "Git 状态获取失败"}
		}
		items := parsePorcelainZ(out)
		if onlySpec {
			prefix := filepath.ToSlash(filepath.Join("vibe-docs", "spec")) + "/"
			filtered := make([]changeItem, 0, len(items))
			for _, it := range items {
				if strings.HasPrefix(filepath.ToSlash(it.Path), prefix) {
					filtered = append(filtered, it)
				}
			}
			items = filtered
		}
		return diffListMsg{Items: items}
	}
}

// render diff for single file; optionally pipe through delta
func renderDiffCmd(root, file string, mode diffMode, width int, useDelta bool) tea.Cmd {
	return func() tea.Msg {
		if _, err := exec.LookPath("git"); err != nil {
			return diffRenderedMsg{Err: "未检测到 Git"}
		}
		args := []string{"-C", root, "--no-pager", "diff", "--color=always"}
		switch mode {
		case diffAll:
			args = append(args, "HEAD")
		case diffStaged:
			args = []string{"-C", root, "--no-pager", "diff", "--color=always", "--staged"}
		case diffWorktree:
			// default index..worktree
		}
		args = append(args, "--", file)
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		gitOut, err := exec.CommandContext(ctx, "git", args...).CombinedOutput()
		if err != nil && len(gitOut) == 0 {
			return diffRenderedMsg{Err: fmt.Sprintf("git diff 失败: %v", err)}
		}
		if useDelta {
			if _, err2 := exec.LookPath("delta"); err2 == nil {
				dargs := []string{"--paging=never"}
				if width > 0 {
					dargs = append(dargs, fmt.Sprintf("--width=%d", width))
				}
				dctx, dcancel := context.WithTimeout(context.Background(), 6*time.Second)
				defer dcancel()
				dc := exec.CommandContext(dctx, "delta", dargs...)
				dc.Stdin = strings.NewReader(string(gitOut))
				dout, derr := dc.CombinedOutput()
				if derr == nil && len(dout) > 0 {
					return diffRenderedMsg{Out: string(dout)}
				}
			}
		}
		return diffRenderedMsg{Out: string(gitOut)}
	}
}

// parsePorcelainZ parses `git status --porcelain=v1 -z` output into items.
func parsePorcelainZ(b []byte) []changeItem {
	items := make([]changeItem, 0, 16)
	i := 0
	for i < len(b) {
		if i+3 > len(b) {
			break
		}
		X := b[i]
		Y := b[i+1]
		// skip status and space
		i += 3
		start := i
		for i < len(b) && b[i] != 0x00 {
			i++
		}
		path := string(b[start:i])
		if i < len(b) && b[i] == 0x00 {
			i++
		}
		group := ""
		if X == '?' && Y == '?' {
			group = "Untracked"
		} else if X != ' ' && X != '?' {
			group = "Staged"
		} else if Y != ' ' {
			group = "Unstaged"
		} else {
			group = "Unstaged"
		}
		status := strings.TrimSpace(string([]byte{X}))
		if status == "" || status == "?" {
			status = string([]byte{Y})
		}
		items = append(items, changeItem{Path: path, Status: status, Group: group})
	}
	return items
}

// buildDiffRows groups change items by Group and injects headers
func buildDiffRows(items []changeItem) []diffRow {
	groups := []string{"Unstaged", "Staged", "Untracked"}
	out := make([]diffRow, 0, len(items)+3)
	for _, g := range groups {
		addedHeader := false
		for _, it := range items {
			if it.Group != g {
				continue
			}
			if !addedHeader {
				ghIcon := ""
				switch g {
				case "Unstaged":
					ghIcon = uistyle.IconDiffModified()
				case "Staged":
					ghIcon = uistyle.IconAccepted()
				case "Untracked":
					ghIcon = uistyle.IconDiffUntracked()
				}
				hdr := lipgloss.NewStyle().Foreground(uistyle.Vitesse.Secondary).Render(
					fmt.Sprintf("%s %s", ghIcon, g),
				)
				out = append(out, diffRow{Header: g, Text: hdr})
				addedHeader = true
			}
			// add a Nerd Font icon per change status and colorize status/icon
			icon := uistyle.IconDiffByStatus(it.Status)
			var col lipgloss.Color
			switch it.Status {
			case "A":
				col = uistyle.Vitesse.Primary
			case "M":
				col = uistyle.Vitesse.Yellow
			case "D":
				col = uistyle.Vitesse.Red
			case "R":
				col = uistyle.Vitesse.Blue
			case "C":
				col = uistyle.Vitesse.Cyan
			case "U":
				col = uistyle.Vitesse.Magenta
			case "?":
				col = uistyle.Vitesse.Secondary
			default:
				col = uistyle.Vitesse.Text
			}
			part := lipgloss.NewStyle().Foreground(col).Render(fmt.Sprintf("%s [%s]", icon, it.Status))
			label := fmt.Sprintf("%s %s", part, it.Path)
			copy := it // avoid aliasing pointer to loop var
			out = append(out, diffRow{Item: &copy, Text: label})
		}
	}
	if len(out) == 0 {
		out = append(out, diffRow{Header: "No changes", Text: "(no changes)"})
	}
	return out
}

func diffRowsToTable(rows []diffRow) []table.Row {
	out := make([]table.Row, 0, len(rows))
	for _, r := range rows {
		out = append(out, table.Row{r.Text})
	}
	return out
}

// (VT key mapping removed; we directly convert keys to PTY bytes below)

// keyToVTEvents maps Bubble Tea KeyMsg into VT key events
func keyToPTYBytes(k tea.KeyMsg) []byte {
	// runes typing
	if k.Type == tea.KeyRunes && len(k.Runes) > 0 {
		return []byte(string(k.Runes))
	}
	switch k.Type {
	case tea.KeySpace:
		return []byte(" ")
	case tea.KeyEnter:
		return []byte("\r")
	case tea.KeyBackspace:
		return []byte{0x7f}
	}
	// named keys
	switch k.String() {
	case "up":
		return []byte("\x1b[A")
	case "down":
		return []byte("\x1b[B")
	case "right":
		return []byte("\x1b[C")
	case "left":
		return []byte("\x1b[D")
	case "home":
		return []byte("\x1b[H")
	case "end":
		return []byte("\x1b[F")
	case "pgup":
		return []byte("\x1b[5~")
	case "pgdown":
		return []byte("\x1b[6~")
	case "tab":
		return []byte("\t")
	}
	return nil
}

// stripFrontmatter removes the first frontmatter block if present
func stripFrontmatter(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return s
	}
	if strings.TrimRight(lines[0], "\r") != "---" {
		return s
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return s
	}
	return strings.Join(lines[end+1:], "\n")
}

// trimEdgeBlankLines removes leading and trailing blank lines from a string

// isMarkdown reports whether the given path looks like a Markdown document
// that we can render on the right pane.
func isMarkdown(path string) bool {
	p := strings.ToLower(path)
	return strings.HasSuffix(p, ".md") || strings.HasSuffix(p, ".mdx") || strings.HasSuffix(p, ".markdown")
}

// nerdIcon returns a Nerd Font icon for a given file or directory name.
// It uses a conservative subset of common glyphs to avoid width issues.
func nerdIcon(name string, isDir bool, expanded bool) string {
	// Choose icon codepoints from Nerd Font (v3) commonly available
	// Folders
	folderClosed := "" // nf-custom-folder
	folderOpen := ""   // nf-custom-folder_open
	// Files
	md := ""   // nf-custom-markdown
	gof := ""  // nf-custom-go
	json := "" // nf-custom-json
	yml := ""  // generic config/gear
	sh := ""   // nf-dev-terminal
	js := ""   // nf-custom-js
	ts := ""   // nf-custom-ts
	py := ""   // nf-custom-python
	html := "" // nf-custom-html5
	css := ""  // nf-custom-css3
	img := ""  // nf-custom-image
	lock := "" // nf-fa-lock
	txt := ""  // nf-fa-file

	icon := txt
	lower := strings.ToLower(name)
	if isDir {
		if expanded {
			icon = folderOpen
		} else {
			icon = folderClosed
		}
	} else if strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".mdx") || strings.HasSuffix(lower, ".markdown") {
		icon = md
	} else if strings.HasSuffix(lower, ".go") {
		icon = gof
	} else if strings.HasSuffix(lower, ".json") {
		icon = json
	} else if strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".toml") {
		icon = yml
	} else if strings.HasSuffix(lower, ".sh") || strings.HasSuffix(lower, ".bash") || strings.HasSuffix(lower, ".zsh") {
		icon = sh
	} else if strings.HasSuffix(lower, ".js") || strings.HasSuffix(lower, ".cjs") || strings.HasSuffix(lower, ".mjs") {
		icon = js
	} else if strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx") {
		icon = ts
	} else if strings.HasSuffix(lower, ".py") {
		icon = py
	} else if strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm") {
		icon = html
	} else if strings.HasSuffix(lower, ".css") || strings.HasSuffix(lower, ".scss") || strings.HasSuffix(lower, ".less") {
		icon = css
	} else if strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".gif") || strings.HasSuffix(lower, ".svg") || strings.HasSuffix(lower, ".ico") {
		icon = img
	} else if strings.HasSuffix(lower, ".lock") || strings.Contains(lower, "lock") {
		icon = lock
	}
	// Return raw glyph without ANSI color to avoid width miscalculation
	// inside bubbles table (which truncates before styling).
	return icon
}

// while preserving inner spacing and ANSI sequences.
func trimEdgeBlankLines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	i := 0
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	j := len(lines) - 1
	for j >= i && strings.TrimSpace(lines[j]) == "" {
		j--
	}
	if i == 0 && j == len(lines)-1 {
		return s
	}
	return strings.Join(lines[i:j+1], "\n")
}

// refreshTableRows removed (unused)
// loadSpecItems removed (unused)

func relFrom(root, p string) string {
	if r, err := filepath.Rel(root, p); err == nil {
		return r
	}
	return p
}

// reloadFileTable populates the file manager table for fmCwd.
func (m *model) reloadTree() {
	m.treeRoot = m.buildTree(m.fmRoot, nil)
	m.visible = m.buildVisible()
	m.fileTable.SetRows(m.visibleRows())
	// after rebuilding, compute new fingerprint for change detection
	m.treeStamp, m.treeCount = m.computeTreeFingerprint()
}

func (m *model) buildTree(dir string, parent *treeNode) *treeNode {
	name := filepath.Base(dir)
	if strings.TrimSpace(name) == "" {
		name = dir
	}
	root := &treeNode{Name: name, Path: dir, IsDir: true, Parent: parent}
	entries, _ := os.ReadDir(dir)
	// sort: dirs first, then files, by name
	sort.SliceStable(entries, func(i, j int) bool {
		di, dj := entries[i].IsDir(), entries[j].IsDir()
		if di != dj {
			return di && !dj
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		if e.IsDir() {
			child := m.buildTree(p, root)
			root.Children = append(root.Children, child)
		} else {
			n := &treeNode{Name: e.Name(), Path: p, IsDir: false, Parent: root}
			root.Children = append(root.Children, n)
		}
	}
	return root
}

func (m *model) buildVisible() []treeRow {
	out := make([]treeRow, 0, 64)
	// root line
	{
		icon := nerdIcon(m.treeRoot.Name+"/", true, true)
		out = append(out, treeRow{Node: m.treeRoot, Text: icon + " " + m.treeRoot.Name + "/"})
	}
	var walk func(parent *treeNode, prefixKinds []bool)
	walk = func(parent *treeNode, prefixKinds []bool) {
		if !m.expanded[parent.Path] {
			return
		}
		for i, ch := range parent.Children {
			last := i == len(parent.Children)-1
			var b strings.Builder
			for _, cont := range prefixKinds {
				if cont {
					b.WriteString("│ ")
				} else {
					b.WriteString("  ")
				}
			}
			if last {
				b.WriteString("└╴ ")
			} else {
				b.WriteString("├╴ ")
			}
			name := ch.Name
			if ch.IsDir {
				name += "/"
			}
			// Nerd Font icon based on filetype/dir status
			icon := nerdIcon(name, ch.IsDir, m.expanded[ch.Path])
			out = append(out, treeRow{Node: ch, Text: b.String() + icon + " " + name})
			if ch.IsDir {
				next := append(append([]bool(nil), prefixKinds...), !last)
				walk(ch, next)
			}
		}
	}
	walk(m.treeRoot, nil)
	return out
}

func (m *model) visibleRows() []table.Row {
	rows := make([]table.Row, 0, len(m.visible))
	for _, r := range m.visible {
		rows = append(rows, table.Row{r.Text})
	}
	return rows
}

// computeTreeFingerprint returns a lightweight snapshot of the current
// visible tree state: the maximum directory mtime among visible directories
// (root + expanded directories) and the total child-entry count across those
// directories. It avoids deep recursion into collapsed folders.
func (m *model) computeTreeFingerprint() (int64, int) {
	if m.treeRoot == nil {
		// fallback to root dir only
		fi, err := os.Stat(m.fmRoot)
		if err != nil || fi == nil {
			return 0, 0
		}
		entries, _ := os.ReadDir(m.fmRoot)
		return fi.ModTime().Unix(), len(entries)
	}
	var maxMod int64
	total := 0
	var walk func(n *treeNode)
	walk = func(n *treeNode) {
		if n == nil || !n.IsDir {
			return
		}
		// consider this dir if it's root or expanded
		consider := n == m.treeRoot || m.expanded[n.Path]
		if consider {
			if fi, err := os.Stat(n.Path); err == nil && fi != nil {
				if ts := fi.ModTime().Unix(); ts > maxMod {
					maxMod = ts
				}
			}
			total += len(n.Children)
		}
		// recurse only into expanded nodes to avoid deep cost
		if m.expanded[n.Path] {
			for _, ch := range n.Children {
				if ch.IsDir {
					walk(ch)
				}
			}
		}
	}
	walk(m.treeRoot)
	return maxMod, total
}

// findVisibleIndexByPath finds the row index of a node path in visible list.
func (m *model) findVisibleIndexByPath(path string) int {
	for i, r := range m.visible {
		if r.Node != nil && r.Node.Path == path {
			return i
		}
	}
	return -1
}

// setFocus updates UI focus across panes and applies component focus state.
func (m *model) setFocus(f focusArea) {
	m.focus = f
	if f == focusInput {
		m.ti.Focus()
	} else {
		m.ti.Blur()
	}
	if f == focusFiles {
		switch m.tab {
		case tabExplorer:
			m.fileTable.Focus()
			m.diffTable.Blur()
		case tabDiff:
			m.diffTable.Focus()
			m.fileTable.Blur()
		default:
			m.fileTable.Focus()
			m.diffTable.Blur()
		}
	} else {
		m.fileTable.Blur()
		m.diffTable.Blur()
	}
}

// parseFrontmatterTitle extracts `title:` from the first frontmatter block
// parseFrontmatterTitle removed (unused)

// work/status bar at bottom using lipgloss
func (m model) renderWorkbar() string {
	// left segments depend on current page/tab
	left := []string{}
	if m.page == pageSelect {
		label := "No files"
		if len(m.items) > 0 {
			cur := m.table.Cursor()
			if cur >= 0 && cur < len(m.items) {
				label = relFrom(m.root, m.items[cur].Path)
			}
		}
		left = append(left, uistyle.IconDoc()+" "+label)
		left = append(left, "↑/↓ 选择")
		left = append(left, "↵ 打开")
	} else {
		switch m.tab {
		case tabExplorer:
			if m.selected != nil {
				left = append(left, uistyle.IconDoc()+" "+filepath.Base(m.selected.Path))
			} else {
				left = append(left, uistyle.IconDoc()+" No file selected")
			}
			left = append(left, "↵ 记录")
			left = append(left, uistyle.IconTerminal()+" 执行")
			left = append(left, uistyle.IconRefresh()+" 载入")
			left = append(left, uistyle.IconFastBolt()+" 快速")
			left = append(left, "Esc 返回")
		case tabDiff:
			left = append(left, uistyle.IconDiff()+" File Diff")
			left = append(left, fmt.Sprintf("模式:%s", diffModeLabel(m.diffMode)))
			if m.diffFilterSpecOnly {
				left = append(left, uistyle.IconFilter()+" 过滤:spec/开")
			} else {
				left = append(left, uistyle.IconFilter()+" 过滤:关")
			}
			left = append(left, uistyle.IconRefresh()+" 刷新")
			left = append(left, uistyle.IconFilter()+" 过滤")
			left = append(left, "←/→ 模式")
		default:
			left = append(left, uistyle.IconTasksWork()+" Spec • Task")
			left = append(left, "S 状态 "+uistyle.IconUser()+" 所有人  优先级")
		}
	}
	// right segments
	right := []string{}
	if m.page == pageDetail {
		switch m.tab {
		case tabExplorer:
			if m.fastMode {
				right = append(right, uistyle.IconFastBolt()+" 快速:开")
			} else {
				right = append(right, uistyle.IconFastBolt()+" 快速:关")
			}
		case tabDiff:
			right = append(right, uistyle.IconDiff()+" Delta:"+onOff(m.hasDelta))
		default:
			right = append(right, uistyle.IconClock()+" "+time.Now().Format(""))
		}
	}
	if !m.now.IsZero() {
		right = append(right, m.now.Format("15:04"))
	} else {
		right = append(right, time.Now().Format("15:04"))
	}
	return renderStatusBarStyled(m.width, left, right)
}

func (m model) renderTabs() string {
	// three-tab bar
	active := lipgloss.NewStyle().Bold(true).
		Foreground(uistyle.Vitesse.OnAccent).
		Background(uistyle.Vitesse.Primary).
		Padding(0, 1)
	inactive := lipgloss.NewStyle().
		Foreground(uistyle.Vitesse.Secondary).
		Background(uistyle.Vitesse.Bg)
	sep := lipgloss.NewStyle().Foreground(uistyle.Vitesse.Border).Render("│")
	var exp, dif, work string
	switch m.tab {
	case tabExplorer:
		exp = active.Render("1 File Explorer")
		dif = inactive.Render(" 2 File Diff ")
		work = inactive.Render(" 3 Spec • Task ")
	case tabDiff:
		exp = inactive.Render(" 1 File Explorer ")
		dif = active.Render("2 File Diff")
		work = inactive.Render(" 3 Spec • Task ")
	default:
		exp = inactive.Render(" 1 File Explorer ")
		dif = inactive.Render(" 2 File Diff ")
		work = active.Render("3 Spec • Task")
	}
	// mark zones for click
	exp = zone.Mark("spec.tab.explorer", exp)
	dif = zone.Mark("spec.tab.diff", dif)
	work = zone.Mark("spec.tab.work", work)
	line := exp + sep + dif + sep + work
	// extend to full width with background
	return lipgloss.NewStyle().Background(uistyle.Vitesse.Bg).Width(m.width).Render(line)
}

// tiny helpers
func cycle(arr []string, cur string) string {
	if len(arr) == 0 {
		return cur
	}
	idx := 0
	for i, v := range arr {
		if strings.EqualFold(v, cur) {
			idx = i
			break
		}
	}
	return arr[(idx+1)%len(arr)]
}

// parse frontmatter helpers (quick & lenient)
func parseFMQuick(path string) map[string]string {
	b, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}
	}
	s := string(b)
	lines := strings.Split(s, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r") != "---" {
		return map[string]string{}
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return map[string]string{}
	}
	out := map[string]string{}
	for _, ln := range lines[1:end] {
		l := strings.TrimSpace(ln)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		if strings.HasSuffix(l, ":") {
			continue
		}
		if i := strings.Index(l, ":"); i >= 0 {
			k := strings.ToLower(strings.TrimSpace(l[:i]))
			v := strings.TrimSpace(l[i+1:])
			if len(v) >= 2 {
				if (v[0] == '\'' && v[len(v)-1] == '\'') || (v[0] == '"' && v[len(v)-1] == '"') {
					v = v[1 : len(v)-1]
				}
			}
			out[k] = v
		}
	}
	return out
}

func parseFMArray(path, key string) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	s := string(b)
	lines := strings.Split(s, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return nil
	}
	lk := strings.ToLower(strings.TrimSpace(key))
	var res []string
	for i := 1; i < end; i++ {
		l := strings.TrimSpace(lines[i])
		if strings.HasPrefix(strings.ToLower(l), lk+":") {
			v := strings.TrimSpace(l[len(lk)+1:])
			if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
				v = strings.TrimSuffix(strings.TrimPrefix(v, "["), "]")
				parts := strings.Split(v, ",")
				for _, p := range parts {
					pp := strings.Trim(strings.TrimSpace(p), "\"'")
					if pp != "" {
						res = append(res, pp)
					}
				}
				return res
			}
			for j := i + 1; j < end; j++ {
				ln := lines[j]
				t := strings.TrimSpace(ln)
				if !strings.HasPrefix(ln, " ") && !strings.HasPrefix(ln, "\t") {
					break
				}
				if strings.HasPrefix(t, "- ") {
					vv := strings.Trim(strings.TrimPrefix(t, "- "), "\"'")
					if vv != "" {
						res = append(res, vv)
					}
				}
			}
			return res
		}
	}
	return res
}

func scanTasks(dir string) []taskItem {
	entries, _ := filepath.Glob(filepath.Join(dir, "*.task.mdx"))
	if len(entries) == 0 {
		return nil
	}
	out := make([]taskItem, 0, len(entries))
	for _, p := range entries {
		fm := parseFMQuick(p)
		t := strings.TrimSpace(fm["title"])
		if t == "" {
			t = filepath.Base(p)
		}
		out = append(out, taskItem{
			Path:         p,
			Title:        t,
			Status:       strings.TrimSpace(fm["status"]),
			Owner:        strings.TrimSpace(fm["owner"]),
			Priority:     strings.ToUpper(strings.TrimSpace(fm["priority"])),
			Due:          strings.TrimSpace(fm["due"]),
			RelatedSpecs: parseFMArray(p, "relatedspec"),
		})
	}
	// newest first by filename
	sort.SliceStable(out, func(i, j int) bool { return out[i].Path > out[j].Path })
	return out
}

func containsPath(arr []string, want string) bool {
	w := filepath.ToSlash(want)
	for _, s := range arr {
		if filepath.ToSlash(s) == w {
			return true
		}
	}
	return false
}

// openInOSCmd opens a file via OS default handler
func openInOSCmd(path string) tea.Cmd {
	return func() tea.Msg {
		exe := ""
		var args []string
		switch runtime.GOOS {
		case "darwin":
			exe = "open"
			args = []string{path}
		case "linux":
			exe = "xdg-open"
			args = []string{path}
		case "windows":
			exe = "cmd"
			args = []string{"/c", "start", "", path}
		default:
			// fallback: no-op
			return nil
		}
		cmd := exec.Command(exe, args...)
		cmd.Env = os.Environ()
		_ = cmd.Start()
		return nil
	}
}

// ---------- fsnotify integration ----------

type watchStartedMsg struct {
	w  *fsnotify.Watcher
	ch chan struct{}
}
type fileChangedMsg struct{}

func startWatchCmd(root string) tea.Cmd {
	return func() tea.Msg {
		w, err := fsnotify.NewWatcher()
		if err != nil {
			return nil
		}
		// watch vibe-docs and subdirs (spec, task) best-effort
		_ = w.Add(filepath.Join(root, "vibe-docs"))
		_ = w.Add(filepath.Join(root, "vibe-docs", "spec"))
		_ = w.Add(filepath.Join(root, "vibe-docs", "task"))
		ch := make(chan struct{}, 1)
		go func() {
			for {
				select {
				case _, ok := <-w.Events:
					if !ok {
						return
					}
					select {
					case ch <- struct{}{}:
					default:
					}
				case _, ok := <-w.Errors:
					if !ok {
						return
					}
				}
			}
		}()
		return watchStartedMsg{w: w, ch: ch}
	}
}

func watchSubscribeCmd(ch <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		<-ch
		time.Sleep(120 * time.Millisecond)
		return fileChangedMsg{}
	}
}

// ---------- work tab: tasks + filters ----------

type tasksLoadedMsg struct{ items []taskItem }

func loadTasksCmd(root, selectedSpec string) tea.Cmd {
	return func() tea.Msg {
		dir := filepath.Join(root, "vibe-docs", "task")
		items := scanTasks(dir)
		// no filtering here; we filter in Update based on current selection
		return tasksLoadedMsg{items: items}
	}
}

func (m *model) applyWorkFilter() {
	sel := m.currentSelectedSpec()
	out := make([]taskItem, 0, len(m.workTasks))
	for _, t := range m.workTasks {
		if sel != "" && !containsPath(t.RelatedSpecs, sel) {
			continue
		}
		if m.workStatus != "All" && !strings.EqualFold(t.Status, m.workStatus) {
			continue
		}
		if m.workOwner != "All" && !strings.EqualFold(t.Owner, m.workOwner) {
			continue
		}
		if m.workPriority != "All" && !strings.EqualFold(t.Priority, m.workPriority) {
			continue
		}
		if s := strings.TrimSpace(m.workSearch); s != "" {
			ls := strings.ToLower(s)
			if !strings.Contains(strings.ToLower(t.Title), ls) && !strings.Contains(strings.ToLower(filepath.Base(t.Path)), ls) {
				continue
			}
		}
		out = append(out, t)
	}
	m.workFiltered = out
	// rows
	rows := make([]table.Row, 0, len(out))
	for _, t := range out {
		rows = append(rows, table.Row{t.Title, t.Status, t.Owner, t.Priority, t.Due})
	}
	m.workTaskTable.SetRows(rows)
}

func (m model) currentSelectedSpec() string {
	if m.focus == focusFiles || true {
		// derive from file table selection
		idx := m.fileTable.Cursor()
		if idx >= 0 && idx < len(m.visible) {
			node := m.visible[idx].Node
			if node != nil && !node.IsDir {
				// ensure under vibe-docs/spec
				rel := filepath.ToSlash(relFrom(m.root, node.Path))
				if strings.HasPrefix(rel, "vibe-docs/spec/") {
					return node.Path
				}
			}
		}
	}
	return ""
}

func (m model) renderWorkFilters() string {
	// chips with current filters; simple text line
	chip := func(txt string) string {
		return lipgloss.NewStyle().Foreground(uistyle.Vitesse.OnAccent).Background(uistyle.Vitesse.Blue).Padding(0, 1).Render(txt)
	}
	parts := []string{"Spec:", chip(baseOr(m.currentSelectedSpec())),
		" Status:", chip(m.workStatus),
		" Owner:", chip(m.workOwner),
		" Pri:", chip(m.workPriority)}
	if s := strings.TrimSpace(m.workSearch); s != "" {
		parts = append(parts, " Search:", chip(s))
	}
	return lipgloss.NewStyle().Background(uistyle.Vitesse.Bg).Render(strings.Join(parts, " "))
}

func baseOr(p string) string {
	if strings.TrimSpace(p) == "" {
		return "All"
	}
	return filepath.Base(p)
}

func diffModeLabel(mo diffMode) string {
	switch mo {
	case diffAll:
		return "HEAD→工作区"
	case diffStaged:
		return "HEAD→暂存区"
	case diffWorktree:
		return "暂存区→工作区"
	}
	return "HEAD→工作区"
}

func onOff(b bool) string {
	if b {
		return "开"
	}
	return "关"
}

// periodic tick for clock updates
type tickMsg time.Time
type termRenderTickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// render tick for terminal pane (throttled ~30fps)
func termTickCmd() tea.Cmd {
	return tea.Tick(33*time.Millisecond, func(time.Time) tea.Msg { return termRenderTickMsg{} })
}

// Render a segmented status bar with lipgloss backgrounds.
func renderStatusBarStyled(width int, leftParts, rightParts []string) string {
	// Lip Gloss layout example-inspired status bar
	w := width
	if w <= 0 {
		w = 100
	}

	// Adopt global Vitesse theme from main UI
	statusBarStyle := uistyle.StatusBarBase()

	keyStyle := uistyle.ChipKeyStyle().Inherit(statusBarStyle).MarginRight(1)

	nugget := lipgloss.NewStyle().
		Foreground(uistyle.Vitesse.OnAccent).
		Padding(0, 1)

	nuggetBG := []lipgloss.Color{
		uistyle.Vitesse.Primary,
		uistyle.Vitesse.Blue,
		uistyle.Vitesse.Yellow,
		uistyle.Vitesse.Magenta,
	}

	centerStyle := lipgloss.NewStyle().Inherit(statusBarStyle)

	leftItems := make([]string, 0, len(leftParts))
	for i, s := range leftParts {
		if i == 0 {
			leftItems = append(leftItems, keyStyle.Render(s))
			continue
		}
		bg := nuggetBG[(i-1)%len(nuggetBG)]
		leftItems = append(leftItems, nugget.Background(bg).Render(s))
	}
	leftStr := strings.Join(leftItems, "")

	rightItems := make([]string, 0, len(rightParts))
	for i, s := range rightParts {
		bg := nuggetBG[i%len(nuggetBG)]
		rightItems = append(rightItems, nugget.Background(bg).Render(s))
	}
	rightStr := strings.Join(rightItems, "")

	lw := xansi.StringWidth(leftStr)
	rw := xansi.StringWidth(rightStr)
	inner := w

	rebuild := func(parts []string) (string, int) {
		s := strings.Join(parts, "")
		return s, xansi.StringWidth(s)
	}

	for lw+rw > inner && len(leftItems) > 1 {
		leftItems = leftItems[:len(leftItems)-1]
		leftStr, lw = rebuild(leftItems)
	}
	for lw+rw > inner && len(rightItems) > 0 {
		rightItems = rightItems[:len(rightItems)-1]
		rightStr, rw = rebuild(rightItems)
	}

	centerWidth := inner - lw - rw
	if centerWidth < 0 {
		centerWidth = 0
	}
	center := centerStyle.Width(centerWidth).Render("")

	bar := leftStr + center + rightStr
	return statusBarStyle.Width(w).Render(bar)
}

// renderVTRightPane renders VT screen to string, and when terminal has focus
// overlays a visible cursor at the emulator cursor position by inverting the
// cell at that position. This is a presentation-only overlay; it does not
// mutate the emulator state.
func renderVTRightPane(m *model) string {
	if m.termVT == nil {
		return ""
	}
	out := m.termVT.Render()
	// Strip OSC (Operating System Control) sequences like OSC 11 that some
	// shells emit (e.g., setting terminal background colors). Rendering these
	// in our TUI can leak into the real terminal and even appear as stray
	// characters if partially interpreted. We keep CSI/SGR for styling.
	out = stripOSC(out)
	if !m.termFocus {
		return out
	}
	// Cursor column/row
	pos := m.termVT.CursorPosition()
	cx, cy := pos.X, pos.Y
	if cx < 0 {
		cx = 0
	}
	if cy < 0 {
		cy = 0
	}
	lines := strings.Split(out, "\r\n")
	// ensure enough lines
	for len(lines) <= cy {
		lines = append(lines, "")
	}
	lines[cy] = overlayCursorOnAnsiLine(lines[cy], cx)
	return strings.Join(lines, "\r\n")
}

// overlayCursorOnAnsiLine returns the line with an inverse-video cursor at
// the given column. It preserves existing ANSI SGR sequences and counts
// display width correctly across runes. If the column is past the end, pads
// spaces and appends an inverse space.
func overlayCursorOnAnsiLine(line string, col int) string {
	if col < 0 {
		col = 0
	}
	// Fast path: if no ANSI and short
	// Walk the string tracking visible column, skipping ANSI sequences
	var b strings.Builder
	b.Grow(len(line) + 16)
	visible := 0
	i := 0
	for i < len(line) {
		if line[i] == 0x1b { // ESC ... CSI or SGR
			j := i + 1
			if j < len(line) && (line[j] == '[' || line[j] == ']' || line[j] == '(' || line[j] == ')' || line[j] == 'P') {
				j++
				for j < len(line) {
					ch := line[j]
					// OSC (ESC]) ends with BEL (0x07) or ST (ESC\)
					if line[i+1] == ']' {
						if ch == 0x07 {
							j++
							break
						}
						if ch == '\\' && j > i+1 && line[j-1] == 0x1b {
							j++
							break
						}
					}
					// CSI/other: final byte in 0x40..0x7E
					if ch >= 0x40 && ch <= 0x7e {
						j++
						break
					}
					j++
				}
			}
			b.WriteString(line[i:j])
			i = j
			continue
		}
		r, sz := utf8.DecodeRuneInString(line[i:])
		if r == utf8.RuneError && sz == 1 {
			// write raw byte
			if visible == col {
				b.WriteString("\x1b[7m")
				b.WriteByte(line[i])
				b.WriteString("\x1b[27m")
			} else {
				b.WriteByte(line[i])
			}
			visible++
			i++
			continue
		}
		width := runewidth.RuneWidth(r)
		if width <= 0 {
			width = 1
		}
		if visible == col {
			b.WriteString("\x1b[7m")
			b.WriteString(line[i : i+sz])
			b.WriteString("\x1b[27m")
		} else {
			b.WriteString(line[i : i+sz])
		}
		visible += width
		i += sz
	}
	if col >= visible {
		// pad spaces up to col, then inverse a space
		if pad := col - visible; pad > 0 {
			b.WriteString(strings.Repeat(" ", pad))
		}
		b.WriteString("\x1b[7m \x1b[27m")
	}
	return b.String()
}

// stripOSC removes OSC escape sequences from a string. OSC sequences start with
// ESC ] and end with BEL (0x07) or ST (ESC \). This prevents terminal control
// codes like OSC 11;rgb:... from affecting the host terminal.
func stripOSC(s string) string {
	b := strings.Builder{}
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == 0x1b { // ESC
			// Look ahead for OSC introducer ']'
			if i+1 < len(s) && s[i+1] == ']' {
				// skip until BEL or ST (ESC \)
				j := i + 2
				for j < len(s) {
					if s[j] == 0x07 { // BEL
						j++
						break
					}
					if s[j] == '\\' && j > i+1 && s[j-1] == 0x1b { // ESC \
						j++
						break
					}
					j++
				}
				i = j
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// stripOSCBytesState removes OSC sequences from byte stream while tracking state
// across chunks. If an OSC sequence starts in this chunk and doesn't end here,
// oscPending will be set to true so that subsequent chunks continue skipping
// until a BEL (0x07) or ST (ESC \) is found.
func stripOSCBytesState(b []byte, oscPending *bool) []byte {
	out := make([]byte, 0, len(b))
	i := 0
	for i < len(b) {
		if *oscPending {
			// Skip until BEL or ST (ESC \)
			for i < len(b) {
				if b[i] == 0x07 { // BEL
					i++
					*oscPending = false
					break
				}
				if b[i] == '\\' && i > 0 && b[i-1] == 0x1b { // ESC \
					i++
					*oscPending = false
					break
				}
				i++
			}
			continue
		}
		if b[i] == 0x1b && i+1 < len(b) && b[i+1] == ']' { // OSC start
			*oscPending = true
			i += 2
			continue
		}
		out = append(out, b[i])
		i++
	}
	return out
}
