package ui

import (
    "strings"
    "time"

    tea "github.com/charmbracelet/bubbletea"
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
    case noticeMsg:
        m.notice = string(msg)
        return m, nil
    case quitMsg:
        m.quitting = true
        return m, tea.Quit
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
