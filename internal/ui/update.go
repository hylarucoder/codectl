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
	zone "github.com/lrstanley/bubblezone"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		// Focus input when clicking on the input zone, and handle dash buttons
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			if zone.Get("cli.input").InBounds(msg) {
				if !m.ti.Focused() {
					m.ti.Focus()
				}
				m.refreshSlash()
				return m, nil
			}
			// Dash buttons (clickable CTAs)
			if zone.Get("dash.btn.settings").InBounds(msg) {
				return m, m.execSlashCmd("/settings", "", false)
			}
			if zone.Get("dash.btn.sync").InBounds(msg) {
				return m, m.execSlashCmd("/sync", "", false)
			}
			if zone.Get("dash.btn.upgrade").InBounds(msg) {
				return m, func() tea.Msg { return startUpgradeMsg{} }
			}
			if zone.Get("dash.btn.doctor").InBounds(msg) {
				return m, tea.Batch(func() tea.Msg { return noticeMsg("正在诊断…") }, checkAllCmd(), configInfoCmd())
			}
			if zone.Get("dash.btn.specui").InBounds(msg) {
				return m, m.execSlashCmd("/specui", "", false)
			}
			if zone.Get("dash.btn.work").InBounds(msg) {
				// open Spec UI (Spec/Task is the 3rd tab inside)
				return m, m.execSlashCmd("/specui", "", false)
			}
		}
		return m, nil
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
		// size ops panel width reasonably; height is set during render pass
		rw := opsRightWidth(m.width)
		if rw < 16 {
			rw = 16
		}
		m.ops.SetSize(rw, maxInt(6, m.height-6))
		return m, nil
	case tea.KeyMsg:
		// Always allow Ctrl+C to quit, even when input is focused
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		// Quit confirmation overlay handling has highest priority
		if m.confirmQuit {
			switch msg.String() {
			case "left", "right", "tab", "shift+tab":
				if m.confirmIndex == 0 {
					m.confirmIndex = 1
				} else {
					m.confirmIndex = 0
				}
				return m, nil
			case "y":
				m.quitting = true
				return m, tea.Quit
			case "n", "esc":
				m.confirmQuit = false
				return m, nil
			}
			if msg.Type == tea.KeyEnter {
				if m.confirmIndex == 0 { // confirm
					m.quitting = true
					return m, tea.Quit
				}
				// cancel
				m.confirmQuit = false
				return m, nil
			}
			// ignore other keys when confirming
			return m, nil
		}
		// Open command palette: Cmd+P / Cmd+Shift+P on mac (if terminal forwards), or Ctrl+P fallback
		sw := msg.String()
		if sw == "cmd+p" || sw == "cmd+shift+p" || sw == "shift+cmd+p" || sw == "ctrl+p" || sw == "alt+shift+p" {
			if !m.ti.Focused() {
				m.ti.Focus()
			}
			// Explicitly open palette as floating overlay, clear prior input.
			m.paletteOpen = true
			m.ti.SetValue("")
			m.slashIndex = 0
			m.refreshSlash()
			return m, nil
		}
		// Note: Do not short-circuit Enter here when input is unfocused.
		// We handle Enter for the ops list below so Exit and other actions work.
		// Do not auto-focus input or open palette by typing directly.
		// Palette must be explicitly opened via Ctrl/Cmd+P.
		if msg.String() == "esc" && (m.ti.Focused() || m.paletteOpen) {
			m.ti.Blur()
			m.slashVisible = false
			m.paletteOpen = false
			m.ti.SetValue("")
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
					m.paletteOpen = false
					return m, func() tea.Msg { return quitMsg{} }
				case "settings":
					m.ti.SetValue("")
					m.slashVisible = false
					m.paletteOpen = false
					return m, m.execSlashCmd("/settings", "", true)
				}
				// execute selection if visible
				if m.slashVisible && len(m.slashFiltered) > 0 {
					cmdStr := m.slashFiltered[m.slashIndex].Name
					m.ti.SetValue("")
					m.slashVisible = false
					m.paletteOpen = false
					return m, m.execSlashCmd(cmdStr, "", true)
				}
				// execute typed slash command
				if strings.HasPrefix(val, "/") {
					m.ti.SetValue("")
					m.slashVisible = false
					m.paletteOpen = false
					return m, m.execSlashLine(val, true)
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
		// otherwise, handle global shortcuts, focus switching, and ops list navigation
		if !m.paletteOpen {
			// Ctrl+H/J/K/L switch focus across dashboard panes
			switch msg.String() {
			case "ctrl+h":
				if m.focusedPane > 0 {
					m.focusedPane--
				}
				return m, nil
			case "ctrl+l":
				if m.focusedPane < 2 {
					m.focusedPane++
				}
				return m, nil
			case "ctrl+j":
				// single row; treat as move right
				if m.focusedPane < 2 {
					m.focusedPane++
				}
				return m, nil
			case "ctrl+k":
				// single row; treat as move left
				if m.focusedPane > 0 {
					m.focusedPane--
				}
				return m, nil
			}
			// Route ops list navigation keys only when Ops pane is focused
			if m.focusedPane == 2 {
				switch msg.Type {
				case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown:
					var cmd tea.Cmd
					m.ops, cmd = m.ops.Update(msg)
					return m, cmd
				}
				// also accept 'k'/'j' vim keys
				if s := msg.String(); s == "k" || s == "j" {
					var cmd tea.Cmd
					m.ops, cmd = m.ops.Update(msg)
					return m, cmd
				}
				// Enter triggers the selected ops action
				if msg.Type == tea.KeyEnter {
					if oi, ok := m.getSelectedOps(); ok {
						// Execute mapped slash command in non-quiet mode (show notices)
						return m, m.execSlashCmd(oi.cmd, "", false)
					}
					// Fallback: quick diagnose
					return m, tea.Batch(func() tea.Msg { return noticeMsg("正在运行诊断…") }, checkAllCmd(), configInfoCmd())
				}
			}
		}
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
	case configInfoMsg:
		m.config = msg.info
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
		sp.Style = lipgloss.NewStyle().Foreground(Vitesse.Primary)
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
		// Show confirmation overlay instead of immediate quit
		m.paletteOpen = false
		m.slashVisible = false
		m.ti.Blur()
		m.confirmQuit = true
		m.confirmIndex = 1 // default to cancel for safety
		return m, nil
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
		if !msg.quiet {
			if msg.err != nil {
				m.notice = fmt.Sprintf("codex 退出（错误）：%v", msg.err)
			} else {
				m.notice = "codex 已退出"
			}
		}
		return m, nil
	case settingsFinishedMsg:
		if !msg.quiet {
			if msg.err != nil {
				m.notice = fmt.Sprintf("设置退出（错误）：%v", msg.err)
			} else {
				m.notice = "设置已关闭"
			}
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
// performActiveTab removed (tabs not used in dash layout)
