package ui

import "github.com/charmbracelet/lipgloss"

// Design centralizes the TUI color palette and common styles.
//
// Palette is based on Vitesse Dark Soft:
// https://github.com/antfu/vscode-theme-vitesse/blob/main/themes/vitesse-dark-soft.json
type designTheme struct {
	// Core brand/semantic colors
	Primary lipgloss.Color // #4d9375
	Blue    lipgloss.Color // #6394bf
	Yellow  lipgloss.Color // #e6cc77
	Magenta lipgloss.Color // #d9739f
	Cyan    lipgloss.Color // #5eaab5
	Red     lipgloss.Color // #cb7676

	// Text colors
	Text      lipgloss.Color // #dbd7caee
	Secondary lipgloss.Color // #bfbaaa
	Muted     lipgloss.Color // #dedcd590

	// Surfaces
	Bg     lipgloss.Color // #222
	BgSoft lipgloss.Color // #292929
	Border lipgloss.Color // #252525

	// Text on accent backgrounds (e.g., buttons/chips)
	OnAccent lipgloss.Color // #222 (vitesse button.foreground)

	// Status bar colors
	BarFG lipgloss.AdaptiveColor // light/dark
	BarBG lipgloss.AdaptiveColor // light/dark
}

// Vitesse defines the current global design theme for the TUI.
var Vitesse = designTheme{
	Primary: lipgloss.Color("#4d9375"),
	Blue:    lipgloss.Color("#6394bf"),
	Yellow:  lipgloss.Color("#e6cc77"),
	Magenta: lipgloss.Color("#d9739f"),
	Cyan:    lipgloss.Color("#5eaab5"),
	Red:     lipgloss.Color("#cb7676"),

	Text:      lipgloss.Color("#dbd7caee"),
	Secondary: lipgloss.Color("#bfbaaa"),
	Muted:     lipgloss.Color("#dedcd590"),

	Bg:     lipgloss.Color("#181818"),
	BgSoft: lipgloss.Color("#292929"),
	Border: lipgloss.Color("#252525"),

	OnAccent: lipgloss.Color("#222"),

	BarFG: lipgloss.AdaptiveColor{Light: "#343433", Dark: "#bfbaaa"},
	BarBG: lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#222"},
}

// Convenience style helpers

// BorderStyle returns a style with the standard border color.
func BorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Vitesse.Border)
}

// FillBG returns a style with the base background color.
func FillBG() lipgloss.Style {
	return lipgloss.NewStyle().Background(Vitesse.Bg)
}

// AccentBold returns a bold style using the primary accent color.
func AccentBold() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(Vitesse.Primary)
}

// ChipKeyStyle returns a style for the left-most highlighted chip in the status bar.
func ChipKeyStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Vitesse.OnAccent).
		Background(Vitesse.Primary).
		Padding(0, 1)
}

// ChipStyle returns a style for colored nuggets (right/left segments).
func ChipStyle(bg lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Vitesse.OnAccent).Background(bg).Padding(0, 1)
}

// StatusBarBase returns the base style for the status bar background/foreground.
func StatusBarBase() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Vitesse.BarFG).Background(Vitesse.BarBG)
}

// Button renders a small accent button label with consistent styling.
func Button(s string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(Vitesse.OnAccent).Background(Vitesse.Primary).Padding(0, 1).Render(s)
}

// AfterButton wraps following text with the base background to prevent
// accent background from visually bleeding beyond the button area.
// Use with a leading space, e.g., AfterButton("  Description"). For lines
// that end right after a button, you can pass a single space: AfterButton(" ").
func AfterButton(s string) string {
	if s == "" {
		return ""
	}
	return lipgloss.NewStyle().Background(Vitesse.Bg).Render(s)
}
