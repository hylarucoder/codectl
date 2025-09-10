package ui

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
    xansi "github.com/charmbracelet/x/ansi"
    tea "github.com/charmbracelet/bubbletea"
)

type SlashCmd struct {
    Name    string
    Aliases []string
    Desc    string
}

var slashCmds = []SlashCmd{
    {Name: "/add-dir", Desc: "Add a new working directory"},
    {Name: "/agents", Desc: "Manage agent configurations"},
    {Name: "/bashes", Desc: "List and manage background tasks"},
    {Name: "/clear", Aliases: []string{"/reset", "/new"}, Desc: "Clear conversation history and free up context"},
    {Name: "/compact", Desc: "Clear history but keep a summary"},
    {Name: "/config", Aliases: []string{"/theme"}, Desc: "Open config panel"},
    {Name: "/context", Desc: "Visualize current context usage"},
    {Name: "/cost", Desc: "Show total cost and duration"},
    {Name: "/doctor", Desc: "Diagnose and verify installation"},
    {Name: "/exit", Aliases: []string{"/quit"}, Desc: "Exit the REPL"},
}

func (m *model) refreshSlash() {
    v := m.ti.Value()
    // slash visible only when input starts with '/'
    if !strings.HasPrefix(v, "/") {
        m.slashVisible = false
        m.slashFiltered = nil
        m.slashIndex = 0
        return
    }
    m.slashVisible = true
    // filter by prefix token (first word)
    q := strings.TrimSpace(v)
    want := q
    // if there are spaces, only use the first token for filtering
    if sp := strings.IndexAny(q, " \t"); sp >= 0 {
        want = q[:sp]
    }
    m.slashFiltered = filterSlashCommands(want)
    if m.slashIndex >= len(m.slashFiltered) {
        m.slashIndex = 0
    }
}

func filterSlashCommands(prefix string) []SlashCmd {
    // Show all when prefix is just '/'
    if prefix == "/" {
        return slashCmds
    }
    res := make([]SlashCmd, 0, len(slashCmds))
    p := strings.ToLower(prefix)
    for _, c := range slashCmds {
        if strings.HasPrefix(strings.ToLower(c.Name), p) {
            res = append(res, c)
            continue
        }
        for _, a := range c.Aliases {
            if strings.HasPrefix(strings.ToLower(a), p) {
                res = append(res, c)
                break
            }
        }
    }
    if len(res) == 0 {
        return slashCmds
    }
    return res
}

func renderSlashHelp(width int, cmds []SlashCmd, sel int) string {
    if len(cmds) == 0 {
        cmds = slashCmds
    }
    // limit list size for readability
    maxItems := 10
    if len(cmds) > maxItems {
        cmds = cmds[:maxItems]
        if sel >= maxItems {
            sel = maxItems - 1
        }
    }
    // compute widths
    nameWidth := 16
    inner := width - 2
    if inner < 20 {
        inner = 20
    }
    // styles
    hl := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render
    dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render
    var b strings.Builder
    // top border
    b.WriteString("╭" + strings.Repeat("─", inner) + "╮\n")
    for i, c := range cmds {
        line := fmt.Sprintf("  %-*s  %s", nameWidth, c.Name, dim(c.Desc))
        // trim to inner width
        w := xansi.StringWidth(line)
        if w > inner {
            // naive trim to fit
            diff := w - inner
            if diff > 0 && diff < len(line) {
                line = line[:len(line)-diff]
            }
        }
        if i == sel {
            line = hl(line)
        }
        b.WriteString("│")
        b.WriteString(line)
        // pad to inner
        pad := inner - xansi.StringWidth(line)
        if pad > 0 {
            b.WriteString(strings.Repeat(" ", pad))
        }
        b.WriteString("│\n")
    }
    // bottom border
    b.WriteString("╰" + strings.Repeat("─", inner) + "╯\n")
    // hint line
    b.WriteString("  ↑/↓ 选择 · Enter 执行 · Esc 关闭\n")
    return b.String()
}

// execSlashLine parses and executes a typed slash command line.
func (m model) execSlashLine(line string) tea.Cmd {
    s := strings.TrimSpace(line)
    if s == "" || !strings.HasPrefix(s, "/") {
        return nil
    }
    parts := strings.Fields(s)
    cmd := parts[0]
    args := ""
    if len(parts) > 1 {
        args = strings.Join(parts[1:], " ")
    }
    return m.execSlashCmd(cmd, args)
}

// execSlashCmd executes a slash command by name and optional args.
func (m model) execSlashCmd(cmd string, args string) tea.Cmd {
    c := canonicalSlash(cmd)
    switch c {
    case "/exit", "/quit":
        return func() tea.Msg { return quitMsg{} }
    case "/clear", "/reset", "/new":
        return func() tea.Msg { return noticeMsg("已清空会话（占位实现）") }
    case "/doctor":
        // Trigger re-check and show notice
        return tea.Batch(
            func() tea.Msg { return noticeMsg("正在运行诊断…") },
            checkAllCmd(),
        )
    default:
        // not implemented
        return func() tea.Msg {
            // find description if exists
            var desc string
            for _, sc := range slashCmds {
                if sc.Name == c {
                    desc = sc.Desc
                    break
                }
            }
            if desc == "" {
                desc = "未实现"
            }
            return noticeMsg(fmt.Sprintf("命令 %s：%s (尚未实现)", c, desc))
        }
    }
}

// canonicalize command including aliases
func canonicalSlash(name string) string {
    n := strings.ToLower(name)
    for _, c := range slashCmds {
        if strings.ToLower(c.Name) == n {
            return c.Name
        }
        for _, a := range c.Aliases {
            if strings.ToLower(a) == n {
                return c.Name
            }
        }
    }
    return n
}
