package ui

import (
	"os"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"codectl/internal/system"
	"codectl/internal/tools"
)

// Model for TUI
type model struct {
	results   map[tools.ToolID]tools.CheckResult
	checking  bool
	updatedAt time.Time
	quitting  bool
	// upgrade state
	upgrading bool
	// Sequential upgrade flow
	upList       []tools.ToolInfo
	upIndex      int
	upSpinner    spinner.Model
	upProgress   progress.Model
	upgradeDone  int
	upgradeTotal int
	cwd          string
	width        int
	height       int
	// input
	ti        textinput.Model
	lastInput string

	// status bar state
	now          time.Time
	git          system.GitInfo
	lastGitCheck time.Time

	// slash commands UI state
	slashVisible  bool
	slashFiltered []SlashCmd
	slashIndex    int
	notice        string

	// transient status-bar hint
	hintText  string
	hintUntil time.Time

	// tabs
	activeTab tabKind

	// config info (for dash)
	config ConfigInfo
}

func initialModel() model {
	wd, _ := os.Getwd()
	m := model{
		results:  make(map[tools.ToolID]tools.CheckResult, len(tools.Tools)),
		checking: true,
		cwd:      wd,
	}
	// text input setup
	ti := textinput.New()
	ti.Prompt = " > "
	ti.Placeholder = "Try \"write a test for <filepath>\""
	ti.CharLimit = 4096
	ti.Blur() // start blurred; press '/' to focus
	m.ti = ti
	// initialize slash suggestions (hidden at start)
	m.refreshSlash()

	// transient operations hint in status bar (single-screen, no tabs)
	m.hintText = "操作: Enter 运行诊断 · ⌘⇧P/‘/’ 命令面板 · Esc 取消输入 · Ctrl+C 退出"
	m.hintUntil = time.Now().Add(6 * time.Second)
	// default tab
	m.activeTab = tabDash
	return m
}

// public constructor for app
func InitialModel() tea.Model { return initialModel() }

func (m model) Init() tea.Cmd {
	return tea.Batch(checkAllCmd(), configInfoCmd(), tickCmd(), gitInfoCmd(m.cwd))
}

// public constructor for app
func (m model) initCmd() tea.Cmd {
	return tea.Batch(checkAllCmd(), configInfoCmd(), tickCmd(), gitInfoCmd(m.cwd))
}
