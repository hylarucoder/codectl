package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	appver "codectl/internal/version"
)

func (m model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	// We'll build into a body buffer and then pin the status bar to the bottom.
	if m.upgrading {
		// header and tabs
		var body strings.Builder
		// top command palette when focused
		if m.ti.Focused() {
			body.WriteString(renderCommandPaletteTop(m.width, m.ti.Value(), m.slashFiltered, m.slashIndex))
		}
		body.WriteString(renderBanner(m.cwd, nil))
		body.WriteString("\n  codectl — 正在升级 CLI\n\n")

		// Draw spinner + info + progress bar + count, inspired by package-manager example
		// current package name
		current := ""
		if m.upIndex < len(m.upList) {
			current = string(m.upList[m.upIndex].ID)
		}
		// available cells for info text between spinner and progress
		spin := m.upSpinner.View() + " "
		prog := m.upProgress.View()
		n := m.upgradeTotal
		wnum := lipgloss.Width(fmt.Sprintf("%d", n))
		pkgCount := fmt.Sprintf(" %*d/%*d", wnum, m.upgradeDone, wnum, n)
		cellsAvail := maxInt(0, m.width-lipgloss.Width(spin+prog+pkgCount))
		pkgName := lipgloss.NewStyle().Foreground(Vitesse.Blue).Render(current)
		info := lipgloss.NewStyle().MaxWidth(cellsAvail).Render("Upgrading " + pkgName)
		cellsRemaining := maxInt(0, m.width-lipgloss.Width(spin+info+prog+pkgCount))
		gap := strings.Repeat(" ", cellsRemaining)
		body.WriteString("  ")
		body.WriteString(spin + info + gap + prog + pkgCount)
		body.WriteString("\n\n")

		// message above input (optional)
		if m.notice != "" {
			fmt.Fprintf(&body, "  %s\n\n", m.notice)
		} else if m.lastInput != "" {
			fmt.Fprintf(&body, "  %s\n\n", m.lastInput)
		}
		// no persistent input; palette shown at top when focused
		// pin status bar to bottom
		viewBody := body.String()
		bottom := m.renderStatusBarLine() // includes trailing \n
		// compute padding lines to push bottom bar to last line
		linesUsed := strings.Count(viewBody, "\n")
		linesBottom := strings.Count(bottom, "\n")
		h := m.height
		if h <= 0 {
			// fallback when height unknown
			return zone.Scan(viewBody + bottom)
		}
		pad := h - linesUsed - linesBottom
		if pad < 0 {
			pad = 0
		}
		return zone.Scan(viewBody + strings.Repeat("\n", pad) + bottom)
	}
	// Compose body content first; we'll pin status bar at bottom later
	var body strings.Builder
	// Single-screen: dash only (remove ASCII banner). Use equal-height rows.
	paletteLines := 0
	topPad := 2 // keep two lines breathing room below palette/top
	if m.ti.Focused() {
		pal := renderCommandPaletteTop(m.width, m.ti.Value(), m.slashFiltered, m.slashIndex)
		paletteLines = strings.Count(pal, "\n")
		body.WriteString(pal)
	}
	// visual spacer after palette (or at very top when no palette)
	body.WriteString("\n\n")
	// Compute equal row heights.
	// Reserve space for message block and bottom status bar.
	msgBlock := 2 // default when no message
	if m.notice != "" || m.lastInput != "" {
		msgBlock = 3
	}
	// 1 line bottom bar (added later)
	bottomBar := 1
	// we insert 2 row separators between the three rows
	rowSeps := 2
	h := m.height
	if h <= 0 {
		h = 24
	}
	avail := h - paletteLines - msgBlock - bottomBar - rowSeps - topPad
	if avail < 6 {
		avail = 6
	}
	rowTotal := avail / 3
	if rowTotal < 3 {
		rowTotal = 3
	}
	// Each card content lines = rowTotal - (title + top + bottom)
	innerLines := rowTotal - 3
	if innerLines < 1 {
		innerLines = 1
	}
	body.WriteString(renderDashFixed(m, innerLines))
	body.WriteString("\n")

	// message line just above input: prefer notice (if any), else lastInput
	if m.notice != "" {
		fmt.Fprintf(&body, "  %s\n\n", m.notice)
	} else if m.lastInput != "" {
		fmt.Fprintf(&body, "  %s\n\n", m.lastInput)
	} else {
		body.WriteString("\n")
	}
	// No persistent input box on dash or other tabs. Command palette shows at top when focused.
	// pin status bar to bottom
	viewBody := body.String()
	bottom := m.renderStatusBarLine() // includes trailing \n
	linesUsed := strings.Count(viewBody, "\n")
	linesBottom := strings.Count(bottom, "\n")
	if h <= 0 {
		return zone.Scan(viewBody + bottom)
	}
	pad := h - linesUsed - linesBottom
	if pad < 0 {
		pad = 0
	}
	return zone.Scan(viewBody + strings.Repeat("\n", pad) + bottom)
}

// renderStatusBarLine builds the status bar string (one line plus a newline)
// to be placed directly under the input (and slash overlay if visible).
func (m model) renderStatusBarLine() string {
	// show transient hint if active
	now := m.now
	if now.IsZero() {
		now = time.Now()
	}
	if m.hintText != "" && now.Before(m.hintUntil) {
		leftParts := []string{m.hintText}
		rightParts := []string{appver.AppVersion}
		return renderStatusBarStyled(m.width, leftParts, rightParts) + "\n"
	}
	leftParts := []string{"codectl", now.Format("15:04")}
	// right segments: version + git info (if available)
	rightParts := []string{"v" + appver.AppVersion}
	if m.git.InRepo {
		rightParts = append(rightParts, "git")
		if m.git.Branch != "" {
			rightParts = append(rightParts, m.git.Branch)
		}
		if m.git.ShortSHA != "" {
			rightParts = append(rightParts, m.git.ShortSHA)
		}
		if m.git.Dirty {
			rightParts = append(rightParts, "*")
		}
	}
	return renderStatusBarStyled(m.width, leftParts, rightParts) + "\n"
}

// helper used locally for layout
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
