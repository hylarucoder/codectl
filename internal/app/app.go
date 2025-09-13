package app

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"codectl/internal/ui"
)

// Start runs the TUI program and returns any error.
func Start() error {
	if _, err := tea.NewProgram(ui.InitialModel()).Run(); err != nil {
		return err
	}
	return nil
}

// Main is a helper to use as entry-point from cmd.
func Main() {
	if err := Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
