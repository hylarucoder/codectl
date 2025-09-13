package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"

	"codectl/internal/system"
	"codectl/internal/tools"
	tea "github.com/charmbracelet/bubbletea"
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
		// Global tab navigation when input is not focused
		if !m.ti.Focused() {
			s := msg.String()
			switch s {
			case "left", "h":
				if m.activeTab == tabInstall {
					m.activeTab = tabClean
				} else {
					m.activeTab--
				}
				return m, nil
			case "right", "l", "tab":
				if m.activeTab == tabClean {
					m.activeTab = tabInstall
				} else {
					m.activeTab++
				}
				return m, nil
			case "shift+tab", "backtab":
				if m.activeTab == tabInstall {
					m.activeTab = tabClean
				} else {
					m.activeTab--
				}
				return m, nil
			case "enter":
				return m, m.performActiveTab()
			case "1":
				m.activeTab = tabInstall
				return m, nil
			case "2":
				m.activeTab = tabUpdate
				return m, nil
			case "3":
				m.activeTab = tabSync
				return m, nil
			case "4":
				m.activeTab = tabClean
				return m, nil
			}
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
				case "down":
					if len(m.slashFiltered) > 0 {
						m.slashIndex++
						if m.slashIndex >= len(m.slashFiltered) {
							m.slashIndex = 0
						}
					}
					return m, nil
				case "tab":
					// Autocomplete to the selected slash command
					if len(m.slashFiltered) > 0 {
						sel := m.slashFiltered[m.slashIndex].Name
						v := m.ti.Value()
						// replace the first token with the selected command name
						if sp := strings.IndexAny(v, " \t"); sp >= 0 {
							// keep existing suffix (including leading space)
							v = sel + v[sp:]
						} else {
							// add a space to allow typing args
							v = sel + " "
						}
						m.ti.SetValue(v)
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
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
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
	case startUpgradeMsg:
		if m.upgrading {
			return m, nil
		}
		// Build sequential upgrade list (tools with npm package)
		list := make([]tools.ToolInfo, 0, len(tools.Tools))
		for _, t := range tools.Tools {
			if t.Package != "" {
				list = append(list, t)
			}
		}
		if len(list) == 0 {
			return m, func() tea.Msg { return noticeMsg("没有可升级的 CLI") }
		}
		m.upgrading = true
		m.upList = list
		m.upIndex = 0
		m.upgradeTotal = len(list)
		m.upgradeDone = 0
		// init spinner and progress
		sp := spinner.New()
		sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
		m.upSpinner = sp
		pr := progress.New(progress.WithDefaultGradient(), progress.WithWidth(40), progress.WithoutPercentage())
		m.upProgress = pr
		// transient hint for upgrade mode
		m.hintText = "升级中: Esc/Ctrl+C 取消"
		m.hintUntil = time.Now().Add(6 * time.Second)
		// kick off first upgrade and spinner
		cmds := []tea.Cmd{m.upSpinner.Tick}
		// start with 0%
		cmds = append(cmds, m.upProgress.SetPercent(0))
		cmds = append(cmds, upgradeOneCmd(m.upList[m.upIndex]))
		return m, tea.Batch(cmds...)
	case quitMsg:
		m.quitting = true
		return m, tea.Quit
	case gitInfoMsg:
		m.git = msg.info
		return m, nil
	case upgradeProgressMsg:
		// Sequentially process next tool and update progress
		if m.upgrading && len(m.upList) > 0 {
			// Print a success/failure line above the TUI
			_ = tea.Printf("✓ %s %s", string(msg.id), msg.note)
			m.upgradeDone++
			m.upIndex++
			// progress percent
			var cmds []tea.Cmd
			if m.upgradeTotal > 0 {
				pct := float64(m.upgradeDone) / float64(m.upgradeTotal)
				cmds = append(cmds, m.upProgress.SetPercent(pct))
			}
			if m.upIndex >= len(m.upList) {
				// Completed; refresh checks and exit upgrade mode
				m.upgrading = false
				m.results = make(map[tools.ToolID]tools.CheckResult, len(tools.Tools))
				m.checking = true
				cmds = append(cmds, checkAllCmd())
				return m, tea.Batch(cmds...)
			}
			// Trigger next upgrade
			cmds = append(cmds, upgradeOneCmd(m.upList[m.upIndex]))
			return m, tea.Batch(cmds...)
		}
		return m, nil
	case spinner.TickMsg:
		if m.upgrading {
			var cmd tea.Cmd
			m.upSpinner, cmd = m.upSpinner.Update(msg)
			return m, cmd
		}
		return m, nil
	case progress.FrameMsg:
		if m.upgrading {
			nm, cmd := m.upProgress.Update(msg)
			if newModel, ok := nm.(progress.Model); ok {
				m.upProgress = newModel
			}
			return m, cmd
		}
		return m, nil
	case codexFinishedMsg:
		if msg.err != nil {
			m.notice = fmt.Sprintf("codex 退出（错误）：%v", msg.err)
		} else {
			m.notice = "codex 已退出"
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

// performActiveTab triggers the action for the current tab.
func (m model) performActiveTab() tea.Cmd {
	switch m.activeTab {
	case tabInstall:
		return m.execSlashCmd("/add", "all")
	case tabUpdate:
		return m.execSlashCmd("/upgrade", "")
	case tabSync:
		return m.execSlashCmd("/sync", "")
	case tabClean:
		return m.execSlashCmd("/remove", "all")
	default:
		return nil
	}
}
