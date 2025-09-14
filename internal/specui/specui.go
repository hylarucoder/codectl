package specui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	zone "github.com/lrstanley/bubblezone"
)

// Start runs the spec UI program
func Start() error {
	m := initialModel()
	// Ensure global zone manager exists (idempotent if already created).
	zone.NewGlobal()
	_, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
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
	renderer  *glamour.TermRenderer
	statusMsg string
	errMsg    string
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
	// focus management
	focus        focusArea
	lastTopFocus focusArea
}

type mdEntry struct {
	out     string
	modUnix int64
	size    int64
}

type fileEntry struct {
	Name  string
	Path  string
	IsDir bool
}

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
	m.expanded = map[string]bool{}
	m.expanded[m.fmRoot] = true
	m.reloadTree()
	return m
}

func (m model) Init() tea.Cmd { return tickCmd() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// update table height (leave some room for header/help)
		m.recalcViewports()
		// adjust file table height to match right box (header+divider+viewport)
		if m.fileTable.Height() != m.mdVP.Height+2 {
			m.fileTable.SetHeight(m.mdVP.Height + 2)
		}
		// async re-render for new width
		if m.selected != nil {
			return m, renderMarkdownCmd(m.selected.Path, m.mdVP.Width, m.fastMode)
		}
		return m, nil
	case tea.MouseMsg:
		// Focus panes by clicking their zones
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
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
		switch m.page {
		case pageDetail:
			switch msg.String() {
			case "enter":
				// Only handle Enter for the file tree when the file pane is focused.
				if m.focus == focusFiles {
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
					return m, renderMarkdownCmd(node.Path, m.mdVP.Width, m.fastMode)
				}
				// If input is focused, let the input handler process Enter below.
			case "left", "backspace":
				// Only treat Left/Backspace as tree navigation when file pane focused.
				if m.focus == focusFiles {
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
				// reload md (async)
				if m.selected != nil {
					m.mdVP.SetContent("渲染中…")
					return m, renderMarkdownCmd(m.selected.Path, m.mdVP.Width, m.fastMode)
				}
				return m, nil
			case "f":
				// toggle fast mode and re-render
				m.fastMode = !m.fastMode
				if m.selected != nil {
					m.mdVP.SetContent("渲染中…")
					return m, renderMarkdownCmd(m.selected.Path, m.mdVP.Width, m.fastMode)
				}
				return m, nil
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
				prev := m.fileTable.Cursor()
				m.fileTable, cmd = m.fileTable.Update(msg)
				cmds = append(cmds, cmd)
				// When moving selection in the file tree, auto-open markdown files on the right
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
			}
			if m.focus == focusInput {
				m.ti, cmd = m.ti.Update(msg)
				cmds = append(cmds, cmd)
			}
			if m.focus == focusPreview {
				m.mdVP, cmd = m.mdVP.Update(msg)
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}
	case tickMsg:
		m.now = time.Time(msg)
		return m, tickCmd()
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
        // For file tree, remove inner horizontal padding so selection can highlight full width
        leftBox = leftBox.Padding(0, 0)
		switch m.focus {
		case focusFiles:
			leftBox = boxStyleFocus
		case focusPreview:
			rightBox = boxStyleFocus
		case focusInput:
			inputBox = boxStyleFocus
		}
        // top: split left (file manager) and right (markdown)
        // Fill file tree background to card bg across the inner width
        ftw := m.fileTable.Width()
        if ftw <= 0 {
            ftw = 32
        }
        left := leftBox.Render(uistyle.FillBG().Width(ftw).Render(m.fileTable.View()))
		left = zone.Mark("spec.files", left)
		// right: header with filename + markdown viewport
		var fname string
		if m.selected != nil {
			fname = relFrom(m.root, m.selected.Path)
		} else {
			fname = "(未选择文件)"
		}
		// divider under filename sized to preview width; clip long names
		sepWidth := m.mdVP.Width
		if sepWidth <= 0 {
			sepWidth = 80
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
		right := rightBox.Render(rightInner)
		right = zone.Mark("spec.preview", right)
		top := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
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
	dimStyle    = lipgloss.NewStyle().Foreground(uistyle.Vitesse.Secondary)
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

    // top split: left (file list) ~40% cols, right (markdown) rest
    innerW := m.width - 2 // borders padding approx
    if innerW < 20 {
        innerW = m.width
    }
    lw := innerW * 2 / 5 // 40%
    if lw < 32 {
        lw = 32
    }
    if lw > innerW-20 { // keep room for right pane
        lw = innerW - 20
    }
    rw := innerW - lw
	if lw < 20 {
		lw = 20
	}
	if rw < 20 {
		rw = 20
	}
    // Adjust for lipgloss border padding by setting slightly smaller dimensions
    // left (file list) uses table height directly; right is markdown viewport
    // right also reserves 2 lines for filename header and divider inside the box
	mdW, mdH := rw-4, topH-2
	if mdH > 2 {
		mdH -= 2
	} else if mdH > 0 {
		mdH = 1
	}
	lgW, lgH := 0, 0
	if mdW < 10 {
		mdW = lw
	}
	if mdH < 3 {
		mdH = topH
	}
	if lgW < 10 {
		lgW = rw
	}
	if lgH < 3 {
		lgH = topH
	}
    if m.mdVP.Width == 0 && m.mdVP.Height == 0 {
        m.mdVP = viewport.New(mdW, mdH)
    } else {
        m.mdVP.Width = mdW
        m.mdVP.Height = mdH
    }
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
    // Only one column: Files
    m.fileTable.SetColumns([]table.Column{{Title: "Files", Width: colW}})
    m.fileTable.SetWidth(colW)
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

func (m *model) buildRenderer() {
	// Always rebuild with current width to ensure proper wrapping
	width := m.mdVP.Width
	if width <= 0 {
		width = 80
	}
	// Adopt demo markdown rendering: account for Glamour's internal gutter
	// to avoid jagged wrapping.
	const glamourGutter = 2
	wrap := width - glamourGutter
	if wrap < 10 {
		wrap = 10
	}
	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(vitesseGlamour()),
		glamour.WithWordWrap(wrap),
	)
	m.renderer = r
}

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
func startPTYCmd(cwd string, cols, rows int) tea.Cmd {
	return func() tea.Msg {
		p, err := xpty.NewPty(cols, rows)
		if err != nil {
			return ptyStartErrMsg{Err: err.Error()}
		}
		sh := os.Getenv("SHELL")
		if sh == "" {
			if _, err := exec.LookPath("bash"); err == nil {
				sh = "bash"
			} else {
				sh = "sh"
			}
		}
		cmd := exec.Command(sh, "-i")
		cmd.Dir = cwd
		cmd.Env = append(os.Environ(), "TERM=xterm-256color")
		if err := p.Start(cmd); err != nil {
			_ = p.Close()
			return ptyStartErrMsg{Err: err.Error()}
		}
		return ptyStartedMsg{Pty: p, Cols: cols, Rows: rows}
	}
}

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
    md := ""      // nf-custom-markdown
    gof := ""      // nf-custom-go
    json := ""    // nf-custom-json
    yml := ""     // generic config/gear
    sh := ""      // nf-dev-terminal
    js := ""      // nf-custom-js
    ts := ""      // nf-custom-ts
    py := ""      // nf-custom-python
    html := ""    // nf-custom-html5
    css := ""     // nf-custom-css3
    img := ""     // nf-custom-image
    lock := ""     // nf-fa-lock
    txt := ""      // nf-fa-file

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

func (m *model) refreshTableRows() {
	// legacy: unused
}

func (m model) loadSpecItems() []specItem {
	return nil
}

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

// setFocus updates UI focus across panes and applies component focus state.
func (m *model) setFocus(f focusArea) {
	m.focus = f
	if f == focusInput {
		m.ti.Focus()
	} else {
		m.ti.Blur()
	}
	if f == focusFiles {
		m.fileTable.Focus()
	} else {
		m.fileTable.Blur()
	}
}

// parseFrontmatterTitle extracts `title:` from the first frontmatter block
func parseFrontmatterTitle(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	s := string(b)
	lines := strings.Split(s, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r") != "---" {
		return ""
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return ""
	}
	keyRe := regexp.MustCompile(`^([A-Za-z0-9_-]+)\s*:\s*(.*)$`)
	for _, ln := range lines[1:end] {
		l := strings.TrimSpace(ln)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		if m := keyRe.FindStringSubmatch(l); len(m) == 3 {
			key := strings.ToLower(strings.TrimSpace(m[1]))
			if key == "title" {
				v := strings.TrimSpace(m[2])
				if len(v) >= 2 {
					if (v[0] == '\'' && v[len(v)-1] == '\'') || (v[0] == '"' && v[len(v)-1] == '"') {
						v = v[1 : len(v)-1]
					}
				}
				return v
			}
		}
	}
	return ""
}

// work/status bar at bottom using lipgloss
func (m model) renderWorkbar() string {
	// left segments depend on current page
	left := []string{}
	if m.page == pageSelect {
		label := "No files"
		if len(m.items) > 0 {
			cur := m.table.Cursor()
			if cur >= 0 && cur < len(m.items) {
				label = relFrom(m.root, m.items[cur].Path)
			}
		}
		left = append(left, label)
		left = append(left, "↑/↓ 选择")
		left = append(left, "Enter 打开")
	} else {
		if m.selected != nil {
			left = append(left, filepath.Base(m.selected.Path))
		} else {
			left = append(left, "No file selected")
		}
		left = append(left, "↵ 记录")
		left = append(left, "! 执行")
		left = append(left, "r 载入")
		left = append(left, "f 快速")
		// terminal binding removed: no 't'/'Tab' hints
		left = append(left, "Esc 返回")
	}
	// right segments
	right := []string{}
	if m.page == pageDetail {
		if m.fastMode {
			right = append(right, "快速:开")
		} else {
			right = append(right, "快速:关")
		}
	}
	if !m.now.IsZero() {
		right = append(right, m.now.Format("15:04"))
	} else {
		right = append(right, time.Now().Format("15:04"))
	}
	return renderStatusBarStyled(m.width, left, right)
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
