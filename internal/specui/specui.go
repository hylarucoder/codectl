package specui

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/charmbracelet/x/xpty"
	"syscall"

	"codectl/internal/system"
)

// Start runs the spec UI program
func Start() error {
	m := initialModel()
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

	// flow
	page      page
	items     []specItem
	table     table.Model
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
	termScr    *cellbuf.Screen
	termWr     *cellbuf.ScreenWriter
	termFocus  bool
	termDirty  bool
	// markdown cache: path -> width -> entry
	mdCache map[string]map[int]mdEntry
}

type mdEntry struct {
	out     string
	modUnix int64
	size    int64
}

func initialModel() model {
	wd, _ := os.Getwd()
	root := wd
	if r, err := system.GitRoot(context.Background(), wd); err == nil && strings.TrimSpace(r) != "" {
		root = r
	}
	// build table
	columns := []table.Column{
		{Title: "File", Width: 36},
		{Title: "Title", Width: 40},
	}
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(12),
	)
	// style similar to bubbletea example
	ts := table.DefaultStyles()
	ts.Header = ts.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	ts.Selected = ts.Selected.
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(ts)

	// input for conversation
	ti := textinput.New()
	ti.Placeholder = "输入对话并回车，Esc 返回列表"
	ti.Prompt = "> "
	ti.CharLimit = 4096

	m := model{
		cwd:     wd,
		root:    root,
		page:    pageSelect,
		table:   t,
		ti:      ti,
		logs:    make([]string, 0, 64),
		mdCache: make(map[string]map[int]mdEntry),
	}
	// preload items
	m.items = m.loadSpecItems()
	m.refreshTableRows()
	return m
}

func (m model) Init() tea.Cmd { return tickCmd() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// update table height (leave some room for header/help)
		if m.page == pageSelect {
			h := m.height - 6
			if h < 6 {
				h = 6
			}
			m.table.SetHeight(h)
		} else {
			m.recalcViewports()
			// async re-render for new width
			if m.selected != nil {
				return m, renderMarkdownCmd(m.selected.Path, m.mdVP.Width, m.fastMode)
			}
			// resize PTY and cell screen if terminal mode is on
			if m.termMode && m.pty != nil {
				cols, rows := m.termSize()
				_ = m.pty.Resize(cols, rows)
				// reset screen to new size
				scr := cellbuf.NewScreen(nil, cols, rows, &cellbuf.ScreenOptions{Term: "xterm-256color"})
				m.termScr = scr
				m.termWr = cellbuf.NewScreenWriter(scr)
			}
		}
		return m, nil
	case tea.KeyMsg:
		// when terminal has focus, forward most keys to PTY
		if m.page == pageDetail && m.termMode && m.termFocus && m.pty != nil {
			// Special-case focus management and program quit
			switch msg.String() {
			case "esc":
				// exit terminal focus back to input
				m.termFocus = false
				return m, nil
			case "ctrl+c":
				// send SIGINT to PTY instead of quitting app
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
		case pageSelect:
			switch msg.String() {
			case "enter":
				if len(m.items) == 0 {
					return m, nil
				}
				row := m.table.SelectedRow()
				if row == nil {
					return m, nil
				}
				idx := m.table.Cursor()
				if idx >= 0 && idx < len(m.items) {
					it := m.items[idx]
                    m.selected = &it
                    m.page = pageDetail
                    // default focus to input (line-mode typing)
                    m.ti.Focus()
                    m.termFocus = false
					// layout first so content isn't lost when creating viewports
					m.recalcViewports()
					// async render (use cache when possible)
					m.statusMsg = "已进入详情视图。按 Esc 返回"
					// default enable terminal mode and start PTY
					m.termMode = true
					cols, rows := m.termSize()
					// check cache before rendering
					var cmds []tea.Cmd
					if m.selected != nil {
						p := m.selected.Path
						w := m.mdVP.Width
						if cW, ok := m.mdCache[p]; ok {
							if ce, ok2 := cW[w]; ok2 {
								// verify file unchanged
								if fi, err := os.Stat(p); err == nil && fi.ModTime().Unix() == ce.modUnix && fi.Size() == ce.size {
									m.mdVP.SetContent(ce.out)
								} else {
									m.mdVP.SetContent("渲染中…")
									cmds = append(cmds, renderMarkdownCmd(p, w, m.fastMode))
								}
							} else {
								m.mdVP.SetContent("渲染中…")
								cmds = append(cmds, renderMarkdownCmd(p, w, m.fastMode))
							}
						} else {
							m.mdVP.SetContent("渲染中…")
							cmds = append(cmds, renderMarkdownCmd(p, w, m.fastMode))
						}
					}
					cmds = append(cmds, startPTYCmd(m.cwd, cols, rows))
					return m, tea.Batch(cmds...)
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		case pageDetail:
			switch msg.String() {
			case "tab":
				if m.termMode && m.pty != nil {
					m.termFocus = !m.termFocus
					if m.termFocus {
						m.ti.Blur()
					} else {
						m.ti.Focus()
					}
					return m, nil
				}
			case "esc":
				// back to list
				m.page = pageSelect
				m.ti.Blur()
				m.statusMsg = ""
				// stop PTY if running
				if m.pty != nil {
					_ = m.pty.Close()
					m.pty = nil
				}
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
				// toggle terminal mode (right pane behavior)
				m.termMode = !m.termMode
				if m.termMode && m.pty == nil {
					// start persistent PTY session
					cols, rows := m.termSize()
					return m, startPTYCmd(m.cwd, cols, rows)
				}
				if !m.termMode && m.pty != nil {
					_ = m.pty.Close()
					m.pty = nil
				}
				return m, nil
			}
			// input handling
			if msg.Type == tea.KeyEnter && m.ti.Focused() {
				val := strings.TrimSpace(m.ti.Value())
				if m.termMode && m.pty != nil {
					if val == "" {
						return m, nil
					}
					// echo input into screen like a terminal prompt
					m.logs = append(m.logs, ">$ "+val)
					m.logVP.SetContent(strings.Join(m.logs, "\n"))
					m.logVP.GotoBottom()
					// write to PTY
					line := val + "\n"
					m.ti.SetValue("")
					return m, writePTYCmd(m.pty, []byte(line))
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
			m.ti, cmd = m.ti.Update(msg)
			cmds = append(cmds, cmd)
			// allow scrolling in viewports
			m.mdVP, cmd = m.mdVP.Update(msg)
			cmds = append(cmds, cmd)
			m.logVP, cmd = m.logVP.Update(msg)
			cmds = append(cmds, cmd)
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
		// initialize cell screen for right pane
		cols, rows := m.termSize()
		scr := cellbuf.NewScreen(nil, cols, rows, &cellbuf.ScreenOptions{Term: "xterm-256color"})
		wr := cellbuf.NewScreenWriter(scr)
		m.pty = msg.Pty
		m.termScr = scr
		m.termWr = wr
		// kick off first read
		return m, tea.Batch(readPTYOnceCmd(m.pty), termTickCmd())
	case ptyChunkMsg:
		if m.termWr != nil && len(msg.Data) > 0 {
			_, _ = m.termWr.Write(msg.Data)
			// render screen into viewport
			m.termDirty = true
		}
		// schedule next read while PTY exists
		if m.pty != nil {
			return m, readPTYOnceCmd(m.pty)
		}
		return m, nil
	case termRenderTickMsg:
		if m.termMode && m.pty != nil {
			if m.termDirty && m.termScr != nil {
				m.logVP.SetContent(cellbuf.Render(m.termScr))
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
		if m.selected != nil && m.selected.Path == msg.Path && m.mdVP.Width == msg.Width {
			if msg.Err != "" {
				m.mdVP.SetContent(fmt.Sprintf("读取/渲染失败：%s", msg.Err))
			} else {
				m.mdVP.SetContent(msg.Out)
				// cache rendered content
				if _, ok := m.mdCache[msg.Path]; !ok {
					m.mdCache[msg.Path] = make(map[int]mdEntry)
				}
				m.mdCache[msg.Path][msg.Width] = mdEntry{out: msg.Out, modUnix: msg.ModUnix, size: msg.Size}
			}
		}
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	switch m.page {
	case pageSelect:
		title := lipgloss.NewStyle().Bold(true).Render("选择一个规范文件 (Enter 确认，Ctrl+C 退出)")
		help := "来源：vibe-docs/spec/*.spec.mdx"
		return strings.Join([]string{
			title,
			"",
			m.table.View(),
			"",
			help,
			"",
			m.renderWorkbar(),
		}, "\n")
	case pageDetail:
		// choose styles based on focus
		rightBox := boxStyle
		inputBox := boxStyle
		if m.termMode && m.termFocus {
			rightBox = boxStyleFocus
		} else if m.ti.Focused() {
			inputBox = boxStyleFocus
		}
		// top: split left (md) and right (log/terminal)
		left := boxStyle.Render(m.mdVP.View())
		right := rightBox.Render(m.logVP.View())
		top := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
		// bottom: input and a lipgloss-rendered work bar
		bottom := inputBox.Render(m.ti.View()) + "\n" + m.renderWorkbar()
		return lipgloss.JoinVertical(lipgloss.Left, top, bottom)
	default:
		return ""
	}
}

var (
	boxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("243")).
			Padding(0, 1)
	boxStyleFocus = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)
	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
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

	// top split left/right 50/50
	innerW := m.width - 2 // borders padding approx
	if innerW < 20 {
		innerW = m.width
	}
	lw := innerW / 2
	rw := innerW - lw
	if lw < 20 {
		lw = 20
	}
	if rw < 20 {
		rw = 20
	}
	// Adjust for lipgloss border padding by setting slightly smaller dimensions
	mdW, mdH := lw-4, topH-2
	lgW, lgH := rw-4, topH-2
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
	if m.logVP.Width == 0 && m.logVP.Height == 0 {
		m.logVP = viewport.New(lgW, lgH)
	} else {
		m.logVP.Width = lgW
		m.logVP.Height = lgH
	}

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
	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
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

func renderMarkdownCmd(path string, width int, forceFast bool) tea.Cmd {
	return func() tea.Msg {
		fi, statErr := os.Stat(path)
		b, err := os.ReadFile(path)
		if err != nil {
			return renderDoneMsg{Path: path, Width: width, Err: err.Error()}
		}
		content := string(b)
		var modUnix int64
		var size int64
		if statErr == nil && fi != nil {
			modUnix = fi.ModTime().Unix()
			size = fi.Size()
		}
		fast := forceFast || len(b) >= fastThresholdBytes
		if fast {
			return renderDoneMsg{Path: path, Width: width, Out: stripFrontmatter(content), ModUnix: modUnix, Size: size}
		}
		r, _ := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(width),
		)
		if out, err := r.Render(content); err == nil {
			return renderDoneMsg{Path: path, Width: width, Out: out, ModUnix: modUnix, Size: size}
		}
		return renderDoneMsg{Path: path, Width: width, Out: content, ModUnix: modUnix, Size: size}
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

// keyToPTYBytes maps Bubble Tea KeyMsg into terminal byte sequences
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

func (m *model) refreshTableRows() {
	rows := make([]table.Row, 0, len(m.items))
	for _, it := range m.items {
		rows = append(rows, table.Row{relFrom(m.root, it.Path), it.Title})
	}
	m.table.SetRows(rows)
}

func (m model) loadSpecItems() []specItem {
	// scan vibe-docs/spec under repo root
	dir := filepath.Join(m.root, "vibe-docs", "spec")
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return nil
	}
	res := make([]specItem, 0, 8)
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if !strings.HasSuffix(name, ".spec.mdx") {
			return nil
		}
		// parse title from frontmatter
		title := parseFrontmatterTitle(path)
		if strings.TrimSpace(title) == "" {
			title = filepath.Base(path)
		}
		res = append(res, specItem{Path: path, Title: title})
		return nil
	})
	sort.Slice(res, func(i, j int) bool { return res[i].Path < res[j].Path })
	return res
}

func relFrom(root, p string) string {
	if r, err := filepath.Rel(root, p); err == nil {
		return r
	}
	return p
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
		if m.termMode {
			if m.termFocus {
				left = append(left, "键入→终端")
			} else {
				left = append(left, "↵ 执行")
			}
		} else {
			left = append(left, "↵ 记录")
		}
		left = append(left, "r 载入")
		left = append(left, "f 快速")
		left = append(left, "t 终端")
		if m.termMode {
			left = append(left, "Tab 焦点")
		}
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
		if m.termMode {
			right = append(right, "终端:开")
		} else {
			right = append(right, "终端:关")
		}
		if m.termMode {
			if m.termFocus {
				right = append(right, "焦点:终端")
			} else {
				right = append(right, "焦点:输入")
			}
		}
	}
	if !m.now.IsZero() {
		right = append(right, m.now.Format("15:04:05"))
	} else {
		right = append(right, time.Now().Format("15:04:05"))
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
	w := width
	if w <= 0 {
		w = 100
	}

	// color palettes
	leftBG := []string{"24", "30", "60", "66"}
	rightBG := []string{"22", "23", "28", "29", "57", "60"}

	seg := func(text, bg string) string {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color(bg))
		return style.Render(" " + text + " ")
	}

	lrender := make([]string, 0, len(leftParts))
	for i, s := range leftParts {
		lrender = append(lrender, seg(s, leftBG[i%len(leftBG)]))
	}
	rrender := make([]string, 0, len(rightParts))
	for i, s := range rightParts {
		rrender = append(rrender, seg(s, rightBG[i%len(rightBG)]))
	}
	lstr := strings.Join(lrender, " ")
	rstr := strings.Join(rrender, " ")

	lw := xansi.StringWidth(lstr)
	rw := xansi.StringWidth(rstr)
	inner := w
	minGap := 1
	if lstr == "" || rstr == "" {
		minGap = 0
	}
	for lw+rw+minGap > inner && len(lrender) > 0 {
		lrender = lrender[:len(lrender)-1]
		lstr = strings.Join(lrender, " ")
		lw = xansi.StringWidth(lstr)
	}
	for lw+rw+minGap > inner && len(rrender) > 0 {
		rrender = rrender[:len(rrender)-1]
		rstr = strings.Join(rrender, " ")
		rw = xansi.StringWidth(rstr)
	}
	pad := inner - lw - rw
	if pad < 0 {
		pad = 0
	}
	return lstr + strings.Repeat(" ", pad) + rstr
}
