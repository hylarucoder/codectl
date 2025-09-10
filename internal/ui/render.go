package ui

import (
    "fmt"
    "strings"
    "unicode/utf8"

    xansi "github.com/charmbracelet/x/ansi"
)

// renderBanner creates a welcome banner similar to Claude Code, customized for codectl.
func renderBanner(cwd string) string {
    lines := []string{
        "✻ Welcome to codectl!",
        "",
        "/help for help, /status for your current setup",
        "",
        fmt.Sprintf("cwd: %s", cwd),
    }
    // compute max rune width
    max := 0
    for _, ln := range lines {
        if w := utf8.RuneCountInString(ln); w > max {
            max = w
        }
    }
    top := "╭" + strings.Repeat("─", max+2) + "╮\n"
    bot := "╰" + strings.Repeat("─", max+2) + "╯\n"
    var sb strings.Builder
    sb.WriteString(top)
    for _, ln := range lines {
        pad := max - utf8.RuneCountInString(ln)
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

