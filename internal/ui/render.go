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

// renderTabs removed (not used in current dash-only UI)

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
// renderInputUI removed (no persistent input box in dash)

// renderStatusBar draws a single-line status bar at the given width
// with left/right-aligned content.
// renderStatusBar removed (status bar uses segmented style)

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
