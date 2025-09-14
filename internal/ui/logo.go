package ui

import (
	"strings"

	xansi "github.com/charmbracelet/x/ansi"
)

// asciiLogo returns a 6-line ASCII art for "CODECTL" using plain ASCII.
func asciiLogo() []string {
	// Height 6, simple banner-style letters
	return []string{
		"  _____    ____    _____   _______  _______  _        ",
		" / ____|  / __ \\  |  __\\ |__   __||__   __|| |       ",
		"| |      | |  | | | |  | |   | |       | |   | |      ",
		"| |      | |  | | | |  | |   | |       | |   | |      ",
		"| |____  | |__| | | |__| |   | |       | |   | |____   ",
		" \\_____|  \\____/  |_____/    |_|       |_|   |______|  ",
	}
}

// asciiLogoBlocks returns 6xN blocks for CODECTL letters.
func asciiLogoBlocks() [][]string {
	C := []string{
		"  #####  ",
		" ####### ",
		" ###     ",
		" ###     ",
		" ####### ",
		"  #####  ",
	}
	O := []string{
		"  #####  ",
		" ####### ",
		" ### ### ",
		" ### ### ",
		" ####### ",
		"  #####  ",
	}
	D := []string{
		" #####   ",
		" ######  ",
		" ### ### ",
		" ### ### ",
		" ######  ",
		" #####   ",
	}
	E := []string{
		" ####### ",
		" ###     ",
		" #####   ",
		" ###     ",
		" ###     ",
		" ####### ",
	}
	T := []string{
		" ####### ",
		"   ###   ",
		"   ###   ",
		"   ###   ",
		"   ###   ",
		"   ###   ",
	}
	L := []string{
		" ###     ",
		" ###     ",
		" ###     ",
		" ###     ",
		" ###     ",
		" ####### ",
	}
	return [][]string{C, O, D, E, C, T, L}
}

// composeLogoLines joins blocks horizontally; when solid=true, fills inner spaces of each block row.
func composeLogoLines(blocks [][]string, solid bool) []string {
	sep := "  "
	out := make([]string, 6)
	for row := 0; row < 6; row++ {
		var parts []string
		for _, blk := range blocks {
			s := blk[row]
			if solid {
				// Fill between first and last non-space with full blocks
				bRunes := []rune(s)
				first, last := -1, -1
				for i, r := range bRunes {
					if r != ' ' {
						first = i
						break
					}
				}
				for i := len(bRunes) - 1; i >= 0; i-- {
					if bRunes[i] != ' ' {
						last = i
						break
					}
				}
				if first >= 0 && last >= first {
					for i := first; i <= last; i++ {
						bRunes[i] = '█'
					}
					s = string(bRunes)
				}
			}
			parts = append(parts, s)
		}
		out[row] = strings.Join(parts, sep)
	}
	return out
}

// renderLogoTopThird centers the ASCII logo horizontally and vertically within the top third.
// Returns the string including the necessary leading newlines.
func renderLogoTopThird(width, height int) string {
	lines := composeLogoLines(asciiLogoBlocks(), true)
	h := len(lines)
	if h == 0 {
		return ""
	}
	// compute top area
	topArea := height / 3
	if topArea < h+1 { // ensure at least room for logo
		topArea = h + 1
	}
	// vertical centering within top third
	padTop := (topArea - h) / 2
	if padTop < 0 {
		padTop = 0
	}
	var b strings.Builder
	if padTop > 0 {
		b.WriteString(strings.Repeat("\n", padTop))
	}
	// horizontal centering/trim
	inner := width
	if inner <= 0 {
		inner = 80
	}
	for _, ln := range lines {
		w := xansi.StringWidth(ln)
		if w >= inner {
			// naive trim
			if len(ln) > inner {
				ln = ln[:inner]
			}
			b.WriteString(colorizeLine(ln))
			b.WriteString("\n")
			continue
		}
		pad := (inner - w) / 2
		if pad < 0 {
			pad = 0
		}
		b.WriteString(strings.Repeat(" ", pad))
		b.WriteString(colorizeLine(ln))
		b.WriteString("\n")
	}
	return b.String()
}

// colorizeLine applies a simple horizontal gradient using lipgloss foreground colors.
func colorizeLine(s string) string {
	// Vitesse accent
	st := AccentBold()
	return st.Render(s)
}

// renderLogoCard renders a full-width card at the top containing a centered
// ASCII "CODECTL" banner. It returns the rendered string and the number of
// lines the card occupies on screen.
func renderLogoCard(totalWidth int) (string, int) {
	if totalWidth <= 0 {
		totalWidth = 80
	}
	// The card box outer width equals inner content width + 2 (borders).
	inner := totalWidth - 2
	if inner < 16 {
		inner = 16
	}

	// Build banner lines and center them within the card's content area.
	// Use ASCII-only variant with '#' glyphs to avoid non-ASCII blocks.
	raw := composeLogoLines(asciiLogoBlocks(), false)
	// renderBodyBox applies a left padding of 2; mirror that to compute centering.
	padLeft := 2
	cw := inner - padLeft
	if cw < 1 {
		cw = 1
	}
	lines := make([]string, len(raw))
	for i, ln := range raw {
		colored := colorizeLine(ln)
		w := xansi.StringWidth(colored)
		if w > cw {
			// Trim to fit content width (ANSI-aware trimming via clipToWidth)
			colored = clipToWidth(colored, cw)
			w = xansi.StringWidth(colored)
		}
		left := 0
		if cw > w {
			left = (cw - w) / 2
		}
		lines[i] = strings.Repeat(" ", left) + colored
	}

	// Fixed content height equals number of banner rows
	innerLines := len(lines)
	if innerLines < 1 {
		innerLines = 1
	}
	card := renderTitledBoxFixed(inner, "", lines, innerLines)
	// Total visual height = top border + content rows + bottom border
	height := innerLines + 2
	return card, height
}
