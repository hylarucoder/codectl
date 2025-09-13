package cli

import (
	"fmt"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// codexFinishedMsg is emitted when the spawned codex process exits.
type codexFinishedMsg struct{ err error }

// codexModel runs the external codex CLI via Bubble Tea's ExecProcess so the
// terminal state is properly restored when the process exits.
type codexModel struct {
	cmd *exec.Cmd
	err error
}

func (m codexModel) Init() tea.Cmd {
	// Start the external process immediately on program start.
	return tea.ExecProcess(m.cmd, func(err error) tea.Msg {
		return codexFinishedMsg{err: err}
	})
}

func (m codexModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case codexFinishedMsg:
		m.err = msg.err
		return m, tea.Quit
	}
	return m, nil
}

func (m codexModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("codex 运行出错: %v\n", m.err)
	}
	return "正在启动 codex ...\n"
}

// codexCmd wires `codectl codex` to `codex --dangerously-bypass-approvals-and-sandbox -m gpt-5 -c model_reasoning_effort=high`.
var codexCmd = &cobra.Command{
	Use:                "codex [args...]",
	Short:              "以预设参数启动 Codex (gpt-5, high)",
	Long:               "等价于：codex --dangerously-bypass-approvals-and-sandbox -m gpt-5 -c model_reasoning_effort=high",
	DisableFlagParsing: true, // pass through all flags/args to the underlying codex
	RunE: func(cmd *cobra.Command, args []string) error {
		base := []string{
			"--dangerously-bypass-approvals-and-sandbox",
			"-m", "gpt-5",
			"-c", "model_reasoning_effort=high",
		}
		finalArgs := append(base, args...)

		c := exec.Command("codex", finalArgs...) //nolint:gosec
		m := codexModel{cmd: c}
		_, err := tea.NewProgram(m).Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(codexCmd)
}
