package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
)

// renderCommandPaletteTop draws a command palette overlay at the very top.
// It includes an input echo line and the filtered commands list.
func renderCommandPaletteTop(width int, value string, cmds []SlashCmd, sel int) string {
	inner := width - 2
	if inner < 20 {
		inner = 20
	}
	nameWidth := 16
	// Vitesse Dark palette (centralized)
	accent := Vitesse.Primary
	muted := Vitesse.Muted
	border := BorderStyle()
	fillBG := FillBG()
	text := lipgloss.NewStyle().Foreground(Vitesse.Text)
	// styles
	prompt := lipgloss.NewStyle().Bold(true).Foreground(accent).Render("›")
	hl := lipgloss.NewStyle().Bold(true).Foreground(accent).Render
	dim := lipgloss.NewStyle().Foreground(muted).Render

	var b strings.Builder
	// top border
	b.WriteString(border.Render("╭"+strings.Repeat("─", inner)+"╮") + "\n")
	// input line (trim to inner)
	valStyled := text.Render(value)
	in := fmt.Sprintf(" %s %s", prompt, valStyled)
	if xansi.StringWidth(in) > inner {
		in = xansi.Truncate(in, inner, "")
	}
	// width handled by lipgloss .Width
	// left border
	b.WriteString(border.Render("│"))
	// inside fill with bg (ANSI-safe width)
	b.WriteString(fillBG.Width(inner).Render(in))
	// right border
	b.WriteString(border.Render("│\n"))

	// items (limit)
	maxItems := 10
	if len(cmds) > maxItems {
		cmds = cmds[:maxItems]
		if sel >= maxItems {
			sel = maxItems - 1
		}
	}
	if len(cmds) == 0 {
		line := "  no matches"
		if xansi.StringWidth(line) > inner {
			line = xansi.Truncate(line, inner, "")
		}
		b.WriteString(border.Render("│"))
		b.WriteString(fillBG.Width(inner).Render(line))
		b.WriteString(border.Render("│\n"))
		// bottom border and hint
		b.WriteString(border.Render("╰"+strings.Repeat("─", inner)+"╯") + "\n")
		b.WriteString("  ↑/↓ 选择 · Tab 补全 · Enter 执行 · Esc 关闭\n")
		return b.String()
	}
	for i, c := range cmds {
		line := fmt.Sprintf("  %-*s  %s", nameWidth, c.Name, dim(c.Desc))
		if xansi.StringWidth(line) > inner {
			line = xansi.Truncate(line, inner, "")
		}
		if i == sel {
			line = hl(line)
		}
		b.WriteString(border.Render("│"))
		b.WriteString(fillBG.Width(inner).Render(line))
		b.WriteString(border.Render("│\n"))
	}
	// bottom border and hint
	b.WriteString(border.Render("╰"+strings.Repeat("─", inner)+"╯") + "\n")
	b.WriteString("  ↑/↓ 选择 · Tab 补全 · Enter 执行 · Esc 关闭\n")
	return b.String()
}
