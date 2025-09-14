package ui

import (
	"strings"

	xansi "github.com/charmbracelet/x/ansi"
)

// asciiLogo returns a 6-line ASCII art for "CODECTL" using plain ASCII.
// asciiLogo removed (unused)

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
// renderLogoTopThird removed (unused)

// colorizeLine applies a simple horizontal gradient using lipgloss foreground colors.
func colorizeLine(s string) string {
	// Vitesse accent
	st := AccentBold()
	return st.Render(s)
}

// renderLogoCard renders a full-width card at the top containing a centered
// ASCII "CODECTL" banner. It returns the rendered string and the number of
// lines the card occupies on screen.
// renderLogoCard removed (unused; use renderLogoCardSized)

// renderLogoCardSized renders the top logo card with an explicit total height.
// totalHeight includes the top and bottom borders; the content area will be
// padded or clipped to fit. Returns the rendered string and the final height.
func renderLogoCardSized(totalWidth, totalHeight int) (string, int) {
	if totalWidth <= 0 {
		totalWidth = 80
	}
	if totalHeight < 3 { // at least top+bottom+1 content
		totalHeight = 3
	}
	inner := totalWidth - 2
	if inner < 16 {
		inner = 16
	}
	// content height inside borders
	innerLines := totalHeight - 2
	if innerLines < 1 {
		innerLines = 1
	}
	raw := composeLogoLines(asciiLogoBlocks(), false)
	padLeft := 2
	cw := inner - padLeft
	if cw < 1 {
		cw = 1
	}
	// horizontally center each logo line
	centered := make([]string, len(raw))
	for i, ln := range raw {
		colored := colorizeLine(ln)
		w := xansi.StringWidth(colored)
		if w > cw {
			colored = clipToWidth(colored, cw)
			w = xansi.StringWidth(colored)
		}
		left := 0
		if cw > w {
			left = (cw - w) / 2
		}
		centered[i] = strings.Repeat(" ", left) + colored
	}
	// fit into target content height, attempt vertical centering when there's room
	var lines []string
	if innerLines <= len(centered) {
		lines = centered[:innerLines]
	} else {
		topPad := (innerLines - len(centered)) / 2
		if topPad < 0 {
			topPad = 0
		}
		lines = make([]string, 0, innerLines)
		// top padding
		for i := 0; i < topPad; i++ {
			lines = append(lines, "")
		}
		// logo lines
		lines = append(lines, centered...)
		// bottom padding
		for len(lines) < innerLines {
			lines = append(lines, "")
		}
	}
	card := renderTitledBoxFixed(inner, "", lines, innerLines)
	return card, innerLines + 2
}
