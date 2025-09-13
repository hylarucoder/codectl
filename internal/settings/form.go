package settings

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	mdl "codectl/internal/models"
)

// Run launches an interactive settings form to configure models.json.
// It loads remote models as options, preselects current local models,
// and saves the selection on submit.
func Run() error {
	// Load current and remote models
	current, _ := mdl.Load()
	remote := mdl.ListRemote()

	// Selection bound to the MultiSelect
	selected := make([]string, len(current))
	copy(selected, current)

	// Light theme tweaks inspired by freeze/interactive.go
	green := lipgloss.Color("#03BF87")
	theme := huh.ThemeCharm()
	theme.FieldSeparator = lipgloss.NewStyle()
	theme.Blurred.Title = theme.Blurred.Title.Width(18).Foreground(lipgloss.Color("7"))
	theme.Focused.Title = theme.Focused.Title.Width(18).Foreground(green).Bold(true)
	theme.Blurred.SelectedOption = theme.Blurred.SelectedOption.Foreground(lipgloss.Color("243"))
	theme.Focused.SelectedOption = lipgloss.NewStyle().Foreground(green)
	theme.Focused.Base.BorderForeground(green)

	// Build options
	opts := make([]huh.Option[string], 0, len(remote))
	for _, r := range remote {
		opts = append(opts, huh.NewOption(r, r))
	}

	// Height: adaptive for readability
	height := 10
	switch n := len(opts); {
	case n == 0:
		height = 3
	case n < 10:
		height = n
	case n > 18:
		height = 18
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title("Settings").Description("选择需要的模型并保存到本地 models.json"),
			huh.NewMultiSelect[string]().
				Title("Models").
				Options(opts...).
				Height(height).
				Value(&selected),
		),
	).WithTheme(theme).WithWidth(60)

	if err := form.Run(); err != nil {
		return err // form canceled or failed
	}

	if err := mdl.Save(selected); err != nil {
		return err
	}
	fmt.Printf("\n✓ 已保存 models.json（%d 项）\n\n", len(selected))
	return nil
}
