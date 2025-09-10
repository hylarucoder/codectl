package ui

import (
    "context"
    "strings"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "codectl/internal/system"
    "codectl/internal/tools"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        inner := msg.Width - 2
        if inner < 10 {
            inner = 10
        }
        tiw := inner - 3 // account for " > " prompt
        if tiw < 5 {
            tiw = 5
        }
        m.ti.Width = tiw
        return m, nil
    case tea.KeyMsg:
        // Always allow Ctrl+C to quit, even when input is focused
        if msg.String() == "ctrl+c" {
            m.quitting = true
            return m, tea.Quit
        }
        // When input is not focused, start typing on any rune/space to auto-focus
        if !m.ti.Focused() {
            switch msg.Type {
            case tea.KeyRunes:
                m.ti.Focus()
                m.ti.SetValue(m.ti.Value() + string(msg.Runes))
                m.refreshSlash()
                return m, nil
            case tea.KeySpace:
                m.ti.Focus()
                m.ti.SetValue(m.ti.Value() + " ")
                m.refreshSlash()
                return m, nil
            }
        }
        // global input focus toggles
        if msg.String() == "/" && !m.ti.Focused() {
            m.ti.Focus()
            // prefill '/'
            m.ti.SetValue("/")
            m.refreshSlash()
            return m, nil
        }
        if msg.String() == "esc" && m.ti.Focused() {
            m.ti.Blur()
            m.slashVisible = false
            return m, nil
        }
        // when input focused, handle typing, slash nav, and submit
        if m.ti.Focused() {
            // selection navigation when slash menu visible
            if m.slashVisible {
                switch msg.String() {
                case "up":
                    if len(m.slashFiltered) > 0 {
                        m.slashIndex--
                        if m.slashIndex < 0 {
                            m.slashIndex = len(m.slashFiltered) - 1
                        }
                    }
                    return m, nil
                case "down", "tab":
                    if len(m.slashFiltered) > 0 {
                        m.slashIndex++
                        if m.slashIndex >= len(m.slashFiltered) {
                            m.slashIndex = 0
                        }
                    }
                    return m, nil
                }
            }
            // capture Enter to submit or execute
            if msg.Type == tea.KeyEnter {
                val := strings.TrimSpace(m.ti.Value())
                // built-in plain commands (no slash needed)
                switch strings.ToLower(val) {
                case "exit", "quit":
                    m.ti.SetValue("")
                    m.slashVisible = false
                    return m, func() tea.Msg { return quitMsg{} }
                }
                // execute selection if visible
                if m.slashVisible && len(m.slashFiltered) > 0 {
                    cmdStr := m.slashFiltered[m.slashIndex].Name
                    m.ti.SetValue("")
                    m.slashVisible = false
                    return m, m.execSlashCmd(cmdStr, "")
                }
                // execute typed slash command
                if strings.HasPrefix(val, "/") {
                    m.ti.SetValue("")
                    m.slashVisible = false
                    return m, m.execSlashLine(val)
                }
                // empty message: no feedback and do nothing
                if val == "" {
                    return m, nil
                }
                // normal submit
                m.lastInput = m.ti.Value()
                m.ti.SetValue("")
                m.notice = ""
                return m, nil
            }
            var cmd tea.Cmd
            m.ti, cmd = m.ti.Update(msg)
            m.refreshSlash()
            return m, cmd
        }
        // otherwise, handle global shortcuts
        switch msg.String() {
        case "ctrl+c", "q":
            m.quitting = true
            return m, tea.Quit
        case "r":
            // re-run checks
            if !m.upgrading {
                m.results = make(map[tools.ToolID]tools.CheckResult, len(tools.Tools))
                m.checking = true
                return m, checkAllCmd()
            }
        case "u":
            // upgrade tools to latest using npm and then re-check
            if m.upgrading {
                return m, nil
            }
            m.upgrading = true
            // transient hint for upgrade mode
            m.hintText = "操作: q/Ctrl+C 退出"
            m.hintUntil = time.Now().Add(6 * time.Second)
            m.upgradeNotes = make(map[tools.ToolID]string, len(tools.Tools))
            total := 0
            for _, t := range tools.Tools {
                if t.Package != "" {
                    total++
                    m.upgradeNotes[t.ID] = "升级中…"
                }
            }
            m.upgradeTotal = total
            m.upgradeDone = 0
            return m, upgradeAllCmd()
        }
    case versionMsg:
        m.results[msg.id] = msg.result
        if len(m.results) == len(tools.Tools) {
            m.checking = false
            m.updatedAt = time.Now()
        }
        return m, nil
    case tickMsg:
        m.now = time.Time(msg)
        // Throttle git checks to every 10 seconds
        var cmd tea.Cmd
        if m.lastGitCheck.IsZero() || time.Since(m.lastGitCheck) >= 10*time.Second {
            m.lastGitCheck = time.Now()
            cmd = gitInfoCmd(m.cwd)
        }
        // schedule next tick
        if cmd != nil {
            return m, tea.Batch(tickCmd(), cmd)
        }
        return m, tickCmd()
    case noticeMsg:
        m.notice = string(msg)
        return m, nil
    case quitMsg:
        m.quitting = true
        return m, tea.Quit
    case gitInfoMsg:
        m.git = msg.info
        return m, nil
    case upgradeProgressMsg:
        // update individual tool upgrade note
        m.upgradeNotes[msg.id] = msg.note
        m.upgradeDone++
        if m.upgradeDone >= m.upgradeTotal {
            // all done; trigger re-check
            m.upgrading = false
            m.results = make(map[tools.ToolID]tools.CheckResult, len(tools.Tools))
            m.checking = true
            return m, checkAllCmd()
        }
        return m, nil
    }
    return m, nil
}

// periodic tick command
func tickCmd() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// git info command
func gitInfoCmd(dir string) tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
        defer cancel()
        gi, _ := system.GetGitInfo(ctx, dir)
        return gitInfoMsg{info: gi}
    }
}
