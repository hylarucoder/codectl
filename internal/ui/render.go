package ui

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
    xansi "github.com/charmbracelet/x/ansi"
)

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
    top := "╭" + strings.Repeat("─", max+2) + "╮\n"
    bot := "╰" + strings.Repeat("─", max+2) + "╯\n"
    var sb strings.Builder
    sb.WriteString(top)
    for _, ln := range lines {
        pad := max - xansi.StringWidth(ln)
        sb.WriteString("│ ")
        sb.WriteString(ln)
        if pad > 0 {
            sb.WriteString(strings.Repeat(" ", pad))
        }
        sb.WriteString(" │\n")
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
    // compute display width ignoring ANSI escape codes
    cw := xansi.StringWidth(content)
    if cw > inner {
        cw = inner
    }
    pad := inner - cw
    top := "╭" + strings.Repeat("─", inner) + "╮\n"
    bot := "╰" + strings.Repeat("─", inner) + "╯\n"
    var sb strings.Builder
    sb.WriteString(top)
    sb.WriteString("│")
    sb.WriteString(content)
    if pad > 0 {
        sb.WriteString(strings.Repeat(" ", pad))
    }
    sb.WriteString("│\n")
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
    style := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("236"))
    return style.Render(line)
}

// renderStatusBarStyled renders a segmented status bar where each segment has
// its own background color. Left segments are left-aligned, right segments
// right-aligned. Spacing is ANSI-safe.
func renderStatusBarStyled(width int, leftParts, rightParts []string) string {
    w := width
    if w <= 0 {
        w = 100
    }

    // Define palettes for segments
    // Left palette (e.g., time)
    leftBG := []string{"24", "30", "60", "66"}
    // Right palette (e.g., git segments)
    rightBG := []string{"22", "23", "28", "29", "57", "60"}

    seg := func(text, bg string) string {
        style := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color(bg))
        return style.Render(" " + text + " ")
    }

    // Render segments with alternating palettes
    lrender := make([]string, 0, len(leftParts))
    for i, s := range leftParts {
        bg := leftBG[i%len(leftBG)]
        lrender = append(lrender, seg(s, bg))
    }
    rrender := make([]string, 0, len(rightParts))
    for i, s := range rightParts {
        bg := rightBG[i%len(rightBG)]
        rrender = append(rrender, seg(s, bg))
    }
    lstr := strings.Join(lrender, " ")
    rstr := strings.Join(rrender, " ")

    // Shrink from left if overflow
    lw := xansi.StringWidth(lstr)
    rw := xansi.StringWidth(rstr)
    inner := w
    // ensure at least one space between sides when both present
    minGap := 1
    if lstr == "" || rstr == "" {
        minGap = 0
    }
    for lw+rw+minGap > inner && len(lrender) > 0 {
        lrender = lrender[:len(lrender)-1]
        lstr = strings.Join(lrender, " ")
        lw = xansi.StringWidth(lstr)
    }
    // If still overflow, trim right-most segment text (naive cut)
    if lw+rw+minGap > inner && len(rrender) > 0 {
        // convert last right segment to plain text approximation
        // We can't easily strip ANSI here, so instead try reducing spacing by using no joins
        // As a fallback, drop entire right segments until it fits
        for lw+rw+minGap > inner && len(rrender) > 0 {
            rrender = rrender[:len(rrender)-1]
            rstr = strings.Join(rrender, " ")
            rw = xansi.StringWidth(rstr)
        }
    }
    // Compose
    lw = xansi.StringWidth(lstr)
    rw = xansi.StringWidth(rstr)
    pad := inner - lw - rw
    if pad < 0 {
        pad = 0
    }
    gap := pad
    if lstr != "" && rstr != "" && gap > 0 {
        // keep at least one space between sides when both present
        // If gap is zero, they will butt together; acceptable when width is tiny
    }
    return lstr + strings.Repeat(" ", gap) + rstr
}
