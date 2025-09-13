package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
)

// Tabs
type tabKind int

const (
	tabDash tabKind = iota
	tabInstall
	tabUpdate
	tabSync
	tabClean
)

func (t tabKind) String() string {
	switch t {
	case tabDash:
		return "dash"
	case tabInstall:
		return "install"
	case tabUpdate:
		return "update"
	case tabSync:
		return "sync"
	case tabClean:
		return "clean"
	default:
		return "?"
	}
}

func renderTabs(width int, active tabKind) string {
	w := width
	if w <= 0 {
		w = 100
	}
	inner := w

	base := lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Background(lipgloss.Color("236")).Padding(0, 1)
	hl := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("86")).Padding(0, 1)

	items := []struct {
		k   tabKind
		txt string
	}{
		{tabDash, "dash"},
		{tabInstall, "install"},
		{tabUpdate, "update"},
		{tabSync, "sync"},
		{tabClean, "clean"},
	}

	parts := make([]string, 0, len(items))
	for _, it := range items {
		if it.k == active {
			parts = append(parts, hl.Render(it.txt))
		} else {
			parts = append(parts, base.Render(it.txt))
		}
	}
	line := strings.Join(parts, " ")
	// Trim if necessary
	if xansi.StringWidth(line) > inner {
		// Naively drop from the right until it fits
		for len(parts) > 0 && xansi.StringWidth(strings.Join(parts, " ")) > inner {
			parts = parts[:len(parts)-1]
		}
		line = strings.Join(parts, " ")
	}
	return line + "\n"
}

// renderBanner creates a welcome banner and can include additional lines inside the box.
func renderBanner(cwd string, extra []string) string {
	lines := []string{
		"✻ Welcome to codectl!",
		"",
		"/help for help, /status for your current setup",
		"",
	}
	if len(extra) > 0 {
		lines = append(lines, extra...)
		lines = append(lines, "")
	}
	lines = append(lines, fmt.Sprintf("cwd: %s", cwd))

	// compute max display width (ignore ANSI codes)
	max := 0
	for _, ln := range lines {
		if w := xansi.StringWidth(ln); w > max {
			max = w
		}
	}
	border := BorderStyle()
	fillBG := FillBG()
	top := border.Render("╭"+strings.Repeat("─", max+2)+"╮") + "\n"
	bot := border.Render("╰"+strings.Repeat("─", max+2)+"╯") + "\n"
	var sb strings.Builder
	sb.WriteString(top)
	for _, ln := range lines {
		sb.WriteString(border.Render("│"))
		// ANSI-safe width: background fill handles padding/truncation
		sb.WriteString(fillBG.Width(max + 2).Render(" " + ln))
		sb.WriteString(border.Render("│\n"))
	}
	sb.WriteString(bot)
	return sb.String()
}

// renderInputBox draws a single-line bordered input hint box at the given width.
func renderInputUI(width int, content string) string {
	// Provide a reasonable fallback width
	w := width
	if w <= 0 {
		w = 100
	}
	// Minimum box width to safely draw borders and one space
	if w < 10 {
		w = 10
	}
	inner := w - 2
	// ANSI-safe width handled by lipgloss Width
	border := BorderStyle()
	fillBG := FillBG()
	top := border.Render("╭"+strings.Repeat("─", inner)+"╮") + "\n"
	bot := border.Render("╰"+strings.Repeat("─", inner)+"╯") + "\n"
	var sb strings.Builder
	sb.WriteString(top)
	sb.WriteString(border.Render("│"))
	sb.WriteString(fillBG.Width(inner).Render(content))
	sb.WriteString(border.Render("│\n"))
	sb.WriteString(bot)
	return sb.String()
}

// renderStatusBar draws a single-line status bar at the given width
// with left/right-aligned content.
func renderStatusBar(width int, left, right string) string {
	w := width
	if w <= 0 {
		w = 100
	}
	inner := w
	// Ensure right fits, then trim left if necessary
	lw := xansi.StringWidth(left)
	rw := xansi.StringWidth(right)
	if lw+rw > inner {
		// space between sections at least 1 when possible
		maxL := inner - rw - 1
		if maxL < 0 {
			maxL = 0
		}
		// naive rune-based trim to approx width
		if xansi.StringWidth(left) > maxL {
			// cut by bytes; acceptable for ASCII-heavy content
			if maxL <= 1 {
				left = ""
			} else if len(left) > maxL {
				left = left[:maxL]
			}
		}
		lw = xansi.StringWidth(left)
	}
	pad := inner - lw - rw
	if pad < 0 {
		pad = 0
	}
	line := left + strings.Repeat(" ", pad) + right
	// Vitesse Dark: fg and bg from design
	style := lipgloss.NewStyle().Foreground(Vitesse.Secondary).Background(Vitesse.Bg)
	return style.Render(line)
}

// renderStatusBarStyled renders a segmented status bar where each segment has
// its own background color. Left segments are left-aligned, right segments
// right-aligned. Spacing is ANSI-safe.
func renderStatusBarStyled(width int, leftParts, rightParts []string) string {
	// Lip Gloss layout example-inspired status bar:
	// - Subtle bar background spans full width
	// - First left item is a highlighted “key” chip
	// - Remaining items (left and right) are colored nuggets
	// - Center area flex-fills with bar background

	w := width
	if w <= 0 {
		w = 100
	}

	// Vitesse bar colors
	statusBarStyle := StatusBarBase()

	// Key chip (left-most) and generic nugget chips
	keyStyle := ChipKeyStyle().Inherit(statusBarStyle).MarginRight(1)

	nugget := lipgloss.NewStyle().
		Foreground(Vitesse.OnAccent).
		Padding(0, 1)

		// Palette for nuggets (alternating)
	nuggetBG := []lipgloss.Color{
		Vitesse.Primary, // green
		Vitesse.Blue,    // blue
		Vitesse.Yellow,  // yellow
		Vitesse.Magenta, // magenta
	}

	// Center fill inherits bar style and expands to fill remaining width
	centerStyle := lipgloss.NewStyle().Inherit(statusBarStyle)

	// Render left chips
	leftItems := make([]string, 0, len(leftParts))
	for i, s := range leftParts {
		if i == 0 {
			leftItems = append(leftItems, keyStyle.Render(s))
			continue
		}
		bg := nuggetBG[(i-1)%len(nuggetBG)]
		leftItems = append(leftItems, nugget.Background(bg).Render(s))
	}
	leftStr := strings.Join(leftItems, "")

	// Render right chips
	rightItems := make([]string, 0, len(rightParts))
	for i, s := range rightParts {
		bg := nuggetBG[i%len(nuggetBG)]
		rightItems = append(rightItems, nugget.Background(bg).Render(s))
	}
	rightStr := strings.Join(rightItems, "")

	// If overflow, trim from the left tail, then from right, preserving the first left key when possible
	lw := xansi.StringWidth(leftStr)
	rw := xansi.StringWidth(rightStr)
	inner := w

	// Helper to rebuild strings from items
	rebuild := func(parts []string) (string, int) {
		s := strings.Join(parts, "")
		return s, xansi.StringWidth(s)
	}

	// Protect the first left chip if present
	for lw+rw > inner && len(leftItems) > 1 {
		leftItems = leftItems[:len(leftItems)-1]
		leftStr, lw = rebuild(leftItems)
	}
	for lw+rw > inner && len(rightItems) > 0 {
		rightItems = rightItems[:len(rightItems)-1]
		rightStr, rw = rebuild(rightItems)
	}

	// Compute center width (can be zero)
	centerWidth := inner - lw - rw
	if centerWidth < 0 {
		centerWidth = 0
	}
	center := centerStyle.Width(centerWidth).Render("")

	bar := leftStr + center + rightStr
	// Ensure base background across the whole bar
	return statusBarStyle.Width(w).Render(bar)
}
