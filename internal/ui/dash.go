package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"

	"codectl/internal/tools"
)

// renderDashFixed enforces equal card heights by fixing the number of inner lines
// for cards in each row. Use this when you want three rows evenly divided.
func renderDashFixed(m model, innerLinesPerCard int) string {
	var b strings.Builder
	// Grid cards (homepage dashboard) — 3-column layout
	b.WriteString(renderDashThreeColsFixed(m, innerLinesPerCard))
	b.WriteString("\n")
	return b.String()
}

// renderDashThreeColsFixed renders a single row with 3 equal-height cards.
// Each card content area is fixed to innerLines lines.
func renderDashThreeColsFixed(m model, innerLines int) string {
	if innerLines < 1 {
		innerLines = 1
	}
	W := m.width
	if W <= 0 {
		W = 80
	}
	gap := 2
	w := calcInnerWidths(W, 3, gap)
	// Column 1: Spec/Task view (stats + recent accepted) — add top padding
	col1 := renderLabeledCardFixedFocus(w[0], "Spec · Tasks", withTopPad(linesSpecOverview(w[0], m), 1), innerLines, m.focusedPane == 0)
	// Column 2: Config overview (CLI + Models + MCP) — add top padding
	col2 := renderLabeledCardFixedFocus(w[1], "配置总览", withTopPad(linesConfigOverview(w[1], m), 1), innerLines, m.focusedPane == 1)
	// Column 3: Operations list (flattened actions)
	// Hide the embedded colored title for this card by passing an empty title
	col3 := renderLabeledCardFixedFocus(w[2], "", linesOpsEmbedded(w[2], innerLines, m), innerLines, m.focusedPane == 2)
	return joinCols([]string{col1, col2, col3}, w, gap)
}

// withTopPad returns a new slice with n empty lines prefixed before lines.
func withTopPad(lines []string, n int) []string {
	if n <= 0 {
		return lines
	}
	pad := make([]string, n)
	out := make([]string, 0, len(pad)+len(lines))
	out = append(out, pad...)
	out = append(out, lines...)
	return out
}

// calcInnerWidths computes inner content widths (excluding the 2 border characters) for n columns.
func calcInnerWidths(totalW, cols, gap int) []int {
	if cols <= 0 {
		return []int{}
	}
	// Each card has outer width = inner + 2. Total gaps = gap*(cols-1)
	avail := totalW - gap*(cols-1) - 2*cols
	if avail < cols*10 {
		// ensure at least minimal width, fallback to 10 each
		avail = cols * 10
	}
	base := avail / cols
	rem := avail % cols
	out := make([]int, cols)
	for i := 0; i < cols; i++ {
		w := base
		if i < rem {
			w++
		}
		if w < 16 {
			w = 16
		}
		out[i] = w
	}
	return out
}

// renderLabeledCard draws a title line above a bordered box. Content lines are rendered inside.
// renderLabeledCardFixed draws a title line above a bordered box, fixing the
// content area to exactly innerLines lines (padding or clipping as needed).
// (renderLabeledCardFixed removed; focus-aware version used)

// renderLabeledCardFixedFocus draws a title line above a bordered box, fixing
// height and allowing a highlight border when focused.
func renderLabeledCardFixedFocus(inner int, title string, lines []string, innerLines int, focused bool) string {
	if inner < 16 {
		inner = 16
	}
	if innerLines < 1 {
		innerLines = 1
	}
	t := strings.TrimSpace(title)
	if focused {
		return renderTitledBoxFixedWithBorder(inner, t, lines, innerLines, Vitesse.Primary)
	}
	return renderTitledBoxFixed(inner, t, lines, innerLines)
}

// renderTitledBox draws a card with the title embedded on the top border.
// renderTitledBoxFixed draws a fixed-height content card with title on the top border.
func renderTitledBoxFixed(inner int, title string, lines []string, innerLines int) string {
    body := renderBodyBox(inner, lines, innerLines)
    top := renderTopBorderWithTitle(inner, title)
    return top + "\n" + body
}

// renderTitledBoxFixedWithBorder is like renderTitledBoxFixed but with a custom border color.
func renderTitledBoxFixedWithBorder(inner int, title string, lines []string, innerLines int, borderColor lipgloss.Color) string {
	body := renderBodyBoxWithBorder(inner, lines, innerLines, borderColor)
	top := renderTopBorderWithTitleColor(inner, title, borderColor)
	return top + "\n" + body
}

// renderBodyBox renders a box with no top border (left/right/bottom only).
// If fixedLines > 0, uses that many content lines; otherwise variable height.
func renderBodyBox(inner int, lines []string, fixedLines int) string {
	if inner < 1 {
		inner = 1
	}
	padLeft := 2
	cw := inner - padLeft
	if cw < 1 {
		cw = 1
	}
	contentStyle := lipgloss.NewStyle().PaddingLeft(padLeft).Width(cw)
	var content string
	if fixedLines > 0 {
		rows := make([]string, fixedLines)
		for i := 0; i < fixedLines; i++ {
			var ln string
			if i < len(lines) {
				ln = lines[i]
			}
			rows[i] = contentStyle.Render(ln)
		}
		content = strings.Join(rows, "\n")
	} else {
		rows := make([]string, 0, maxInt(1, len(lines)))
		if len(lines) == 0 {
			rows = append(rows, contentStyle.Render(""))
		} else {
			for _, ln := range lines {
				rows = append(rows, contentStyle.Render(ln))
			}
		}
		content = strings.Join(rows, "\n")
	}
	card := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(Vitesse.Border).
		Background(Vitesse.Bg).
		BorderTop(false).BorderLeft(true).BorderRight(true).BorderBottom(true).
		Width(inner)
	if fixedLines > 0 {
		card = card.Height(fixedLines)
	}
	return card.Render(content)
}

// renderBodyBoxWithBorder renders a box (no top border) with a custom border color.
func renderBodyBoxWithBorder(inner int, lines []string, fixedLines int, borderColor lipgloss.Color) string {
	if inner < 1 {
		inner = 1
	}
	padLeft := 2
	cw := inner - padLeft
	if cw < 1 {
		cw = 1
	}
	contentStyle := lipgloss.NewStyle().PaddingLeft(padLeft).Width(cw)
	var content string
	if fixedLines > 0 {
		rows := make([]string, fixedLines)
		for i := 0; i < fixedLines; i++ {
			var ln string
			if i < len(lines) {
				ln = lines[i]
			}
			rows[i] = contentStyle.Render(ln)
		}
		content = strings.Join(rows, "\n")
	} else {
		rows := make([]string, 0, maxInt(1, len(lines)))
		if len(lines) == 0 {
			rows = append(rows, contentStyle.Render(""))
		} else {
			for _, ln := range lines {
				rows = append(rows, contentStyle.Render(ln))
			}
		}
		content = strings.Join(rows, "\n")
	}
	card := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Background(Vitesse.Bg).
		BorderTop(false).BorderLeft(true).BorderRight(true).BorderBottom(true).
		Width(inner)
	if fixedLines > 0 {
		card = card.Height(fixedLines)
	}
	return card.Render(content)
}

// renderTopBorderWithTitle composes the top border line with the title embedded.
func renderTopBorderWithTitle(inner int, title string) string {
	if inner < 1 {
		inner = 1
	}
	border := BorderStyle()
	t := strings.TrimSpace(title)
	// If title is empty, render a plain top border without an embedded title.
	if t == "" {
		return border.Render("╭" + strings.Repeat("─", inner) + "╮")
	}
	// Styled title (no background to blend with border line)
	tStyled := AccentBold().Render(t)
	tW := xansi.StringWidth(tStyled)
	// Draw at least one dash before the title to make the header obvious.
	leftFill := 1
	// Max title width considering: left dashes + space + title + space + right dashes = inner
	maxTitleW := inner - leftFill - 2
	if maxTitleW < 0 {
		maxTitleW = 0
	}
	if tW > maxTitleW {
		tStyled = clipToWidth(tStyled, maxTitleW)
		tW = xansi.StringWidth(tStyled)
	}
	rightFill := inner - leftFill - tW - 2
	if rightFill < 1 {
		rightFill = 1
	}
	left := border.Render("╭")
	pre := border.Render(strings.Repeat("─", leftFill) + " ")
	post := border.Render(" " + strings.Repeat("─", rightFill) + "╮")
	return left + pre + tStyled + post
}

// renderTopBorderWithTitleColor composes the top border line with the title embedded,
// using the provided border color.
func renderTopBorderWithTitleColor(inner int, title string, color lipgloss.Color) string {
	if inner < 1 {
		inner = 1
	}
	border := lipgloss.NewStyle().Foreground(color)
	t := strings.TrimSpace(title)
	if t == "" {
		return border.Render("╭" + strings.Repeat("─", inner) + "╮")
	}
	tStyled := AccentBold().Render(t)
	tW := xansi.StringWidth(tStyled)
	leftFill := 1
	maxTitleW := inner - leftFill - 2
	if maxTitleW < 0 {
		maxTitleW = 0
	}
	if tW > maxTitleW {
		tStyled = clipToWidth(tStyled, maxTitleW)
		tW = xansi.StringWidth(tStyled)
	}
	rightFill := inner - leftFill - tW - 2
	if rightFill < 1 {
		rightFill = 1
	}
	left := border.Render("╭")
	pre := border.Render(strings.Repeat("─", leftFill) + " ")
	post := border.Render(" " + strings.Repeat("─", rightFill) + "╮")
	return left + pre + tStyled + post
}

// clipToWidth trims a string to the given display width (ANSI-safe).
func clipToWidth(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	if xansi.StringWidth(s) <= maxW {
		return s
	}
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := xansi.StringWidth(string(r))
		if w+rw > maxW {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String()
}

// renderBox draws a bordered box with given inner width and content lines.
// (renderBox variants removed; consolidated into renderBodyBox family)

// joinCols aligns multiple card blocks horizontally with fixed gap.
func joinCols(cols []string, innerWidths []int, gap int) string {
	if len(cols) == 0 {
		return ""
	}
	// Split into lines and compute per-column heights
	split := make([][]string, len(cols))
	heights := make([]int, len(cols))
	outerW := make([]int, len(cols))
	for i, c := range cols {
		lines := strings.Split(strings.TrimRight(c, "\n"), "\n")
		split[i] = lines
		heights[i] = len(lines)
		// outer width = inner + 2
		iw := 16
		if i < len(innerWidths) {
			iw = innerWidths[i]
		}
		outerW[i] = iw + 2
	}
	// Find max height
	maxH := 0
	for _, h := range heights {
		if h > maxH {
			maxH = h
		}
	}
	var b strings.Builder
	for row := 0; row < maxH; row++ {
		for i := range cols {
			var cell string
			if row < len(split[i]) {
				cell = split[i][row]
			} else {
				cell = strings.Repeat(" ", outerW[i])
			}
			b.WriteString(cell)
			if i != len(cols)-1 {
				b.WriteString(strings.Repeat(" ", gap))
			}
		}
		if row != maxH-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// Content providers for cards
func linesSpecStats(m model) []string {
	c := m.config
	lines := []string{}
	total := c.SpecTotal
	if total == 0 {
		lines = append(lines, "暂无 spec")
		return lines
	}
	lines = append(lines, fmt.Sprintf("Total: %d", total))
	if c.SpecDraft > 0 {
		lines = append(lines, fmt.Sprintf("Draft: %d", c.SpecDraft))
	}
	if c.SpecProposal > 0 {
		lines = append(lines, fmt.Sprintf("Proposal: %d", c.SpecProposal))
	}
	if c.SpecAccepted > 0 {
		lines = append(lines, fmt.Sprintf("Accepted: %d", c.SpecAccepted))
	}
	if c.SpecDeprecated > 0 {
		lines = append(lines, fmt.Sprintf("Deprecated: %d", c.SpecDeprecated))
	}
	if c.SpecRetired > 0 {
		lines = append(lines, fmt.Sprintf("Retired: %d", c.SpecRetired))
	}
	return lines
}

// linesRecentSpecsTable renders a small two-column table: [#] [标题]
func linesRecentSpecsTable(inner int, m model) []string {
    arr := m.config.SpecRecent
	// content width inside card (account for left padding in box)
	cw := inner - 2
	if cw < 8 {
		cw = inner // fallback
	}
	headers := []string{"#", "标题"}
	rows := make([][]string, 0, len(arr))
	if len(arr) == 0 {
		rows = append(rows, []string{"-", "暂无已完成 spec"})
	} else {
		for i, t := range arr {
			rows = append(rows, []string{fmt.Sprintf("%d", i+1), t})
		}
	}
	return renderTable(cw, headers, rows)
}

// linesCliStatusTable renders a table for codex/claude/gemini status.
func linesCliStatusTable(inner int, m model) []string {
	cw := inner - 2
	if cw < 12 {
		cw = inner
	}
	headers := []string{"工具", "安装", "版本", "最新"}
	ids := []tools.ToolID{tools.ToolCodex, tools.ToolClaude, tools.ToolGemini}
	rows := make([][]string, 0, len(ids))
	for _, id := range ids {
		res, ok := m.results[id]
		name := string(id)
		installed := "×"
		ver := ""
		latest := ""
		if ok {
			if res.Installed {
				installed = "✓"
			}
			ver = strings.TrimSpace(res.Version)
			latest = strings.TrimSpace(res.Latest)
		} else if m.checking {
			installed = "…"
		}
		if ver == "" && installed == "✓" {
			ver = "(未知)"
		}
		rows = append(rows, []string{name, installed, ver, latest})
	}
	out := renderTable(cw, headers, rows)
	return out
}

// linesSpecOverview combines spec stats and a small recent table.
func linesSpecOverview(inner int, m model) []string {
	out := make([]string, 0, 64)
	// Stats
	stats := linesSpecStats(m)
	if len(stats) > 0 {
		out = append(out, stats...)
	}
	// Recent accepted table
	table := linesRecentSpecsTable(inner, m)
	if len(table) > 0 {
		if len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, table...)
	}
	return out
}

// linesConfigOverview merges CLI status table and brief model/MCP lists.
func linesConfigOverview(inner int, m model) []string {
	out := make([]string, 0, 64)
	// CLI tools table
	out = append(out, linesCliStatusTable(inner, m)...)
	// Models
	if len(out) > 0 {
		out = append(out, "")
	}
	out = append(out, linesModels(m)...)
	// MCP services
	if len(out) > 0 {
		out = append(out, "")
	}
	out = append(out, linesMCP(m)...)
	return out
}

// linesQuickActions provides primary actions and a short slogan.
// linesOpsEmbedded renders the right-side ops list inside the card.
func linesOpsEmbedded(inner int, innerLines int, m model) []string {
	if inner < 8 {
		inner = 8
	}
	if innerLines < 1 {
		innerLines = 1
	}
	// Reserve 1 line as top padding inside the card
	listHeight := innerLines - 1
	if listHeight < 1 {
		listHeight = 1
	}
	s := m.renderOpsPanel(inner, listHeight)
	arr := strings.Split(s, "\n")
	// Ensure exactly innerLines lines, with first line as top padding
	out := make([]string, innerLines)
	out[0] = ""
	for i := 1; i < innerLines; i++ {
		idx := i - 1
		if idx < len(arr) {
			out[i] = arr[idx]
		} else {
			out[i] = ""
		}
	}
	return out
}

// renderTable builds simple left-aligned table lines that fit exactly into cw.
// It computes per-column widths from content with graceful truncation.
func renderTable(cw int, headers []string, rows [][]string) []string {
	if cw < 4 || len(headers) == 0 {
		// fallback: join cells
		out := make([]string, 0, len(rows)+1)
		out = append(out, strings.Join(headers, " "))
		for _, r := range rows {
			out = append(out, strings.Join(r, " "))
		}
		return out
	}
	cols := len(headers)
	sep := "  "
	sepW := xansi.StringWidth(sep)
	avail := cw - sepW*(cols-1)
	if avail < cols {
		avail = cols
	}
	// desired widths = max width per column from content
	desired := make([]int, cols)
	// helper to consider string widths (ANSI-safe)
	widen := func(i int, s string) {
		if w := xansi.StringWidth(s); w > desired[i] {
			desired[i] = w
		}
	}
	for i, h := range headers {
		widen(i, h)
	}
	for _, r := range rows {
		for i := 0; i < cols && i < len(r); i++ {
			widen(i, r[i])
		}
	}
	// allocate widths ensuring sum = avail
	widths := make([]int, cols)
	remaining := avail
	remainCols := cols
	for i := 0; i < cols; i++ {
		maxForThis := remaining - (remainCols - 1) // leave at least 1 for each remaining col
		if maxForThis < 1 {
			maxForThis = 1
		}
		w := desired[i]
		if w > maxForThis {
			w = maxForThis
		}
		if w < 1 {
			w = 1
		}
		widths[i] = w
		remaining -= w
		remainCols--
	}
	// clip/pad cell to width
	clip := func(s string, w int) string {
		if w <= 0 {
			return ""
		}
		sw := xansi.StringWidth(s)
		if sw == w {
			return s
		}
		if sw < w {
			return s + strings.Repeat(" ", w-sw)
		}
		// trim runes until fit (rough, but ANSI-safe width check below)
		var b strings.Builder
		count := 0
		for _, r := range s {
			// treat most runes width=1; xansi width used for final padding
			if count+1 > w {
				break
			}
			b.WriteRune(r)
			count++
		}
		out := b.String()
		if xansi.StringWidth(out) < w {
			out += strings.Repeat(" ", w-xansi.StringWidth(out))
		}
		return out
	}
	// build lines
	out := make([]string, 0, len(rows)+1)
	// header (bold)
	hcells := make([]string, cols)
	for i, h := range headers {
		hcells[i] = AccentBold().Render(clip(h, widths[i]))
	}
	out = append(out, strings.Join(hcells, sep))
	// rows
	for _, r := range rows {
		cells := make([]string, cols)
		for i := 0; i < cols; i++ {
			var val string
			if i < len(r) {
				val = r[i]
			}
			cells[i] = clip(val, widths[i])
		}
		out = append(out, strings.Join(cells, sep))
	}
	return out
}

func linesModels(m model) []string {
	if m.config.ModelsCount == 0 || len(m.config.Models) == 0 {
		return []string{"暂无已选择的模型"}
	}
	lines := []string{"已选择模型:"}
	for _, s := range m.config.Models {
		lines = append(lines, "• "+s)
	}
	return lines
}

func linesMCP(m model) []string {
	if m.config.MCPCount == 0 || len(m.config.MCPNames) == 0 {
		return []string{"暂无 MCP 服务"}
	}
	lines := []string{"服务列表:"}
	for _, n := range m.config.MCPNames {
		lines = append(lines, "• "+n)
	}
	return lines
}

// (misc unused helpers removed)
