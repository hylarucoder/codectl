package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// opsItem represents an item in the right-side operations panel.
type opsItem struct {
	title    string
	desc     string
	cmd      string // slash command to execute (e.g., "/specui"); empty means non-action header
	isHeader bool
}

func (i opsItem) Title() string       { return i.title }
func (i opsItem) Description() string { return i.desc }
func (i opsItem) FilterValue() string { return i.title + " " + i.desc }

// newOpsList constructs the grouped list with desired sections.
func newOpsList() list.Model {
	// Flattened actionable list (no group headers)
	items := []list.Item{
		opsItem{title: "Spec UI", desc: "Open Spec UI", cmd: "/specui"},
		opsItem{title: "Update CLI", desc: "Upgrade all CLIs", cmd: "/upgrade"},
		opsItem{title: "Model Settings", desc: "Open settings", cmd: "/settings"},
		opsItem{title: "MCP Settings", desc: "Open settings", cmd: "/settings"},
		opsItem{title: "Sync Providers", desc: "Sync provider.json", cmd: "/sync"},
		opsItem{title: "Exit", desc: "Quit codectl", cmd: "/exit"},
	}

	// Use default delegate and adapt styles to Vitesse theme
	d := list.NewDefaultDelegate()
	s := list.NewDefaultItemStyles()
	// Normal item
	s.NormalTitle = s.NormalTitle.Foreground(Vitesse.Text)
	s.NormalDesc = s.NormalDesc.Foreground(Vitesse.Secondary)
	// Selected item: accent colored left border and title/desc
	s.SelectedTitle = s.SelectedTitle.
		BorderForeground(Vitesse.Primary).
		Foreground(Vitesse.Primary)
	s.SelectedDesc = s.SelectedDesc.
		Foreground(Vitesse.Primary)
	// Dimmed when filtering (not commonly used since filter hidden)
	s.DimmedTitle = s.DimmedTitle.Foreground(Vitesse.Secondary)
	s.DimmedDesc = s.DimmedDesc.Foreground(Vitesse.Muted)
	// Highlight filter matches
	s.FilterMatch = lipgloss.NewStyle().Foreground(Vitesse.Yellow).Underline(true)
	d.Styles = s
	l := list.New(items, d, 28, 12)
	// List chrome styles (title/help/status/pagination) use theme colors if shown
	ls := list.DefaultStyles()
	ls.Title = ls.Title.Foreground(Vitesse.Text)
	ls.PaginationStyle = ls.PaginationStyle.Foreground(Vitesse.Secondary)
	ls.HelpStyle = ls.HelpStyle.Foreground(Vitesse.Muted)
	ls.StatusBar = ls.StatusBar.Foreground(Vitesse.Secondary)
	l.Styles = ls
	// Do not render internal title; the card handles captioning itself
	l.Title = ""
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetShowPagination(false)
	// Default to first item
	l.Select(0)
	// Custom styling: dim non-action headers in the delegate's Render function is invasive.
	// For simplicity, we keep default rendering; we will skip executing headers on Enter.
	return l
}

// opsRightWidth returns the desired width for the right operations panel.
func opsRightWidth(total int) int {
	// Target ~30% of total width within sane bounds
	w := total / 3
	if w < 24 {
		w = 24
	}
	if w > 36 {
		w = 36
	}
	if w > total-20 {
		// leave at least 20 cols for the left content
		w = total - 20
	}
	if w < 16 {
		w = 16
	}
	return w
}

// renderOpsPanel returns the right-side operations list view padded to width.
func (m *model) renderOpsPanel(width, height int) string {
	if height < 3 {
		height = 3
	}
	if width < 16 {
		width = 16
	}
	m.ops.SetSize(width, height)
	s := m.ops.View()
	// pad each line to width to avoid bleed-through when joining columns
	return padLinesToWidth(s, width)
}

// handleOpsKey updates the ops list selection for a key. Returns a tea.Cmd, but
// we avoid importing tea here; caller handles Enter behavior.
func (m *model) handleOpsKey(msg any) any {
	// This is just a typed passthrough; Update is invoked from Update() directly.
	return msg
}

// getSelectedOps returns the current selected actionable item, or ok=false.
func (m *model) getSelectedOps() (opsItem, bool) {
	it := m.ops.SelectedItem()
	if it == nil {
		return opsItem{}, false
	}
	oi, ok := it.(opsItem)
	if !ok || strings.TrimSpace(oi.cmd) == "" {
		return opsItem{}, false
	}
	return oi, true
}
