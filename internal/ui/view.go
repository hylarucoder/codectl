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
		// Build base body first (no palette here; we'll overlay later)
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
			screen := viewBody + bottom
			// overlay palette last (only when palette is open)
			if m.paletteOpen {
				if pal, top := renderPlacedPalette(m); pal != "" {
					screen = overlayAt(screen, pal, top)
				}
			}
			return zone.Scan(screen)
		}
		pad := h - linesUsed - linesBottom
		if pad < 0 {
			pad = 0
		}
		screen := viewBody + strings.Repeat("\n", pad) + bottom
		// overlay palette last (only when palette is open)
		if m.paletteOpen {
			if pal, top := renderPlacedPalette(m); pal != "" {
				screen = overlayAt(screen, pal, top)
			}
		}
		return zone.Scan(screen)
	}
	// Compose body content first; we'll pin status bar at bottom later
	var body strings.Builder
	// Single-screen: dash only (remove ASCII banner). Use equal-height rows.
	topPad := 0 // overlay floats; don't push content
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
	avail := h - msgBlock - bottomBar - rowSeps - topPad
	if avail < 6 {
		avail = 6
	}
	rowTotal := avail / 3
	if rowTotal < 3 {
		rowTotal = 3
	}
	// Title is embedded into the top border now, so per card overhead is: top(=title) + bottom = 2
	innerLines := rowTotal - 2
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
		screen := viewBody + bottom
		if m.paletteOpen {
			if pal, top := renderPlacedPalette(m); pal != "" {
				screen = overlayAt(screen, pal, top)
			}
		}
		return zone.Scan(screen)
	}
	pad := h - linesUsed - linesBottom
	if pad < 0 {
		pad = 0
	}
	screen := viewBody + strings.Repeat("\n", pad) + bottom
	// overlay palette last (only when palette is open)
	if m.paletteOpen {
		if pal, top := renderPlacedPalette(m); pal != "" {
			screen = overlayAt(screen, pal, top)
		}
	}
	return zone.Scan(screen)
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

// renderPlacedPalette renders the command palette as an overlay positioned
// roughly at top 1/5 of the screen and using ~3/5 of the terminal width,
// centered horizontally. It returns the rendered string and the total line
// count it occupies (including vertical padding).
func renderPlacedPalette(m model) (string, int) {
	// Compute overlay outer width (~60% of terminal width), with sane bounds.
	ow := (m.width * 3) / 5
	if ow < 20 {
		if m.width >= 20 {
			ow = 20
		} else {
			ow = m.width // very narrow terminals
		}
	}
	if ow > m.width {
		ow = m.width
	}
	// Vertical offset (line index): ~20% of terminal height.
	top := m.height / 5
	if top < 0 {
		top = 0
	}
	// Build the palette box for the target width.
	pal := renderCommandPaletteTop(ow, m.ti.Value(), m.slashFiltered, m.slashIndex)
	// Center horizontally by indenting each line with left margin.
	left := (m.width - ow) / 2
	if left < 0 {
		left = 0
	}
	pal = indentLines(pal, left)
	// Ensure overlay lines fully cover the screen width to avoid bleed-through
	pal = padLinesToWidth(pal, m.width)
	// Return palette string (indented) and line offset (no vertical padding applied).
	return pal, top
}

// indentLines prefixes each line in s with n spaces.
func indentLines(s string, n int) string {
	if n <= 0 || s == "" {
		return s
	}
	pad := strings.Repeat(" ", n)
	// Preserve trailing newline structure by splitting and re-joining.
	lines := strings.Split(s, "\n")
	for i := range lines {
		if lines[i] == "" {
			// still indent to keep left border aligned
			lines[i] = pad + lines[i]
			continue
		}
		lines[i] = pad + lines[i]
	}
	return strings.Join(lines, "\n")
}

// padLinesToWidth right-pads each line (considering ANSI width) to 'width'.
func padLinesToWidth(s string, width int) string {
	if width <= 0 || s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	for i := range lines {
		w := lipgloss.Width(lines[i])
		if w < width {
			lines[i] = lines[i] + strings.Repeat(" ", width-w)
		}
	}
	return strings.Join(lines, "\n")
}

// overlayAt replaces lines in the base screen starting at line index 'top'
// with the provided overlay lines. It does not change the total line count.
func overlayAt(screen string, overlay string, top int) string {
	if overlay == "" || top < 0 {
		return screen
	}
	baseLines := strings.Split(screen, "\n")
	ovLines := strings.Split(strings.TrimRight(overlay, "\n"), "\n")
	for i := 0; i < len(ovLines); i++ {
		idx := top + i
		if idx < 0 || idx >= len(baseLines) {
			break
		}
		baseLines[idx] = ovLines[i]
	}
	return strings.Join(baseLines, "\n")
}
