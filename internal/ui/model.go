package ui

import (
	"os"
	"time"

	"github.com/charmbracelet/bubbles/list"
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
	// explicit palette open (Ctrl/Cmd+P); keeps overlay visible even without '/'
	paletteOpen bool

	// transient status-bar hint
	hintText  string
	hintUntil time.Time

	// tabs
	activeTab tabKind
	// focused dashboard pane: 0=Spec/Tasks, 1=Config, 2=Ops
	focusedPane int

	// config info (for dash)
	config ConfigInfo

	// right-side operations panel (grouped list)
	ops list.Model

	// selection state for dash recent specs table
	recentIndex int

	// quit confirmation overlay
	confirmQuit  bool
	confirmIndex int // 0 = confirm, 1 = cancel

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
	ti.Blur() // start blurred; use Ctrl/Cmd+P to open palette
	m.ti = ti
	// initialize slash suggestions (hidden at start)
	m.refreshSlash()

	// operations panel list
	m.ops = newOpsList()

	// transient operations hint in status bar (single-screen, no tabs)
	m.hintText = "操作: Enter 执行右侧操作 · ⌘P/Ctrl+P 命令面板 · Esc 关闭 · Ctrl+C 退出"
	m.hintUntil = time.Now().Add(6 * time.Second)
	// default tab
	m.activeTab = tabDash
	// default focus to bottom-right card (Ops)
	m.focusedPane = 2
	return m
}

// public constructor for app
func InitialModel() tea.Model { return initialModel() }

func (m model) Init() tea.Cmd {
	return tea.Batch(checkAllCmd(), configInfoCmd(), tickCmd(), gitInfoCmd(m.cwd))
}

// initCmd removed (unused)
