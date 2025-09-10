package main

import (
    "fmt"
    "os"

    tea "github.com/charmbracelet/bubbletea"
)

type model struct {
    count   int
    quitting bool
}

func initialModel() model {
    return model{count: 0}
}

func (m model) Init() tea.Cmd {
    // No I/O on start
    return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "q":
            m.quitting = true
            return m, tea.Quit
        case "+", "=", "up", "k":
            m.count++
        case "-", "_", "down", "j":
            if m.count > 0 {
                m.count--
            }
        case "0":
            m.count = 0
        }
    }
    return m, nil
}

func (m model) View() string {
    if m.quitting {
        return "Goodbye!\n"
    }
    return fmt.Sprintf(
        "\n  Simple Counter (Bubble Tea)\n\n  Count: %d\n\n  [+]/[up]/k to increment\n  [-]/[down]/j to decrement\n  [0] to reset\n  q to quit\n\n",
        m.count,
    )
}

func main() {
    if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}

