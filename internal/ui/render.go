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
