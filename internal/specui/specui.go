package specui

import (
	"context"
	"fmt"
	"io/fs"
	"os"
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
		cwd:   wd,
		root:  root,
		page:  pageSelect,
		table: t,
		ti:    ti,
		logs:  make([]string, 0, 64),
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
		}
		return m, nil
	case tea.KeyMsg:
		// global quit
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
					m.buildRenderer()
					m.page = pageDetail
					m.ti.Focus()
					m.statusMsg = "已进入详情视图。按 Esc 返回"
					// load content
					m.reloadContent()
					m.recalcViewports()
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		case pageDetail:
			switch msg.String() {
			case "esc":
				// back to list
				m.page = pageSelect
				m.ti.Blur()
				m.statusMsg = ""
				return m, nil
			case "r":
				// reload md
				m.reloadContent()
				return m, nil
			}
			// input handling
			if msg.Type == tea.KeyEnter && m.ti.Focused() {
				val := strings.TrimSpace(m.ti.Value())
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
		// top: split left (md) and right (log)
		left := boxStyle.Render(m.mdVP.View())
		right := boxStyle.Render(m.logVP.View())
		top := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
		// bottom: input and a lipgloss-rendered work bar
		bottom := boxStyle.Render(m.ti.View()) + "\n" + m.renderWorkbar()
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
	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

func (m *model) recalcViewports() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	// Split: top (60%) and bottom (40%)
	topH := int(float64(m.height) * 0.6)
	if topH < 6 {
		topH = m.height - 5
	}
	if topH < 3 {
		topH = 3
	}
	bottomH := m.height - topH
	if bottomH < 3 {
		bottomH = 3
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

	// rebuild renderer for new width and re-render content
	m.buildRenderer()
	if m.selected != nil {
		m.reloadContent()
	}
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

func (m *model) reloadContent() {
	if m.selected == nil {
		return
	}
	b, err := os.ReadFile(m.selected.Path)
	if err != nil {
		m.errMsg = err.Error()
		m.mdVP.SetContent(fmt.Sprintf("读取失败：%v", err))
		return
	}
	content := string(b)
	// render MDX as markdown (best effort)
	if m.renderer != nil {
		if out, err := m.renderer.Render(content); err == nil {
			m.mdVP.SetContent(out)
		} else {
			m.mdVP.SetContent(content)
		}
	} else {
		m.mdVP.SetContent(content)
	}
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
		left = append(left, "↵ 记录")
		left = append(left, "r 载入")
		left = append(left, "Esc 返回")
	}
	// right segments
	right := []string{}
	if !m.now.IsZero() {
		right = append(right, m.now.Format("15:04:05"))
	} else {
		right = append(right, time.Now().Format("15:04:05"))
	}
	return renderStatusBarStyled(m.width, left, right)
}

// periodic tick for clock updates
type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
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
