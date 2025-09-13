package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"codectl/internal/tools"
	appver "codectl/internal/version"
)

func (m model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	b := &strings.Builder{}
	if m.upgrading {
		// header and tabs
		b.WriteString(renderBanner(m.cwd, nil))
		b.WriteString("\n")
		b.WriteString(renderTabs(m.width, m.activeTab))
		b.WriteString("\n  codectl — 正在升级 CLI\n\n")

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
		pkgName := lipgloss.NewStyle().Foreground(lipgloss.Color("211")).Render(current)
		info := lipgloss.NewStyle().MaxWidth(cellsAvail).Render("Upgrading " + pkgName)
		cellsRemaining := maxInt(0, m.width-lipgloss.Width(spin+info+prog+pkgCount))
		gap := strings.Repeat(" ", cellsRemaining)
		b.WriteString("  ")
		b.WriteString(spin + info + gap + prog + pkgCount)
		b.WriteString("\n\n")

		// message above input (optional)
		if m.notice != "" {
			fmt.Fprintf(b, "  %s\n\n", m.notice)
		} else if m.lastInput != "" {
			fmt.Fprintf(b, "  %s\n\n", m.lastInput)
		}
		b.WriteString(renderInputUI(m.width, m.ti.View()))
		if !(m.ti.Focused() && m.slashVisible) {
			b.WriteString(m.renderStatusBarLine())
		}
		return b.String()
	}
	// Build status lines to render inside the banner
	var status []string
	status = append(status, "codectl — CLI 版本检测")
	status = append(status, "")
	for _, t := range tools.Tools {
		res, ok := m.results[t.ID]
		if !ok && m.checking {
			status = append(status, fmt.Sprintf("  • %-12s: 检测中…", t.ID))
			continue
		}
		if !res.Installed {
			if res.Err != "" {
				if res.Latest != "" {
					status = append(status, fmt.Sprintf("  • %-12s: 未安装 (最新 %s, %s)", t.ID, res.Latest, res.Err))
				} else {
					status = append(status, fmt.Sprintf("  • %-12s: 未安装 (%s)", t.ID, res.Err))
				}
			} else {
				if res.Latest != "" {
					status = append(status, fmt.Sprintf("  • %-12s: 未安装 (最新 %s)", t.ID, res.Latest))
				} else {
					status = append(status, fmt.Sprintf("  • %-12s: 未安装", t.ID))
				}
			}
			continue
		}
		// Installed
		ver := res.Version
		if ver == "" {
			ver = "(未知版本)"
		}
		// show latest and highlight when update available
		latest := res.Latest
		if latest == "" {
			status = append(status, fmt.Sprintf("  • %-12s: %s  [%s]", t.ID, ver, res.Source))
			continue
		}
		if tools.VersionLess(res.Version, latest) {
			// highlight latest in red if update available
			red := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render
			status = append(status, fmt.Sprintf("  • %-12s: %s → %s  [%s]", t.ID, ver, red(latest), res.Source))
		} else if tools.NormalizeVersion(res.Version) == tools.NormalizeVersion(latest) {
			// equal to latest; avoid redundant latest notation
			status = append(status, fmt.Sprintf("  • %-12s: %s  [%s]", t.ID, ver, res.Source))
		} else {
			// current version is newer (rare); show both
			status = append(status, fmt.Sprintf("  • %-12s: %s (最新 %s)  [%s]", t.ID, ver, latest, res.Source))
		}
	}

	b.WriteString(renderBanner(m.cwd, status))
	// tabs under banner
	b.WriteString("\n")
	b.WriteString(renderTabs(m.width, m.activeTab))

	// no "上次更新" display per requirement
	// message line just above input: prefer notice (if any), else lastInput
	b.WriteString("\n")
	if m.notice != "" {
		fmt.Fprintf(b, "  %s\n\n", m.notice)
	} else if m.lastInput != "" {
		fmt.Fprintf(b, "  %s\n\n", m.lastInput)
	}
	b.WriteString(renderInputUI(m.width, m.ti.View()))
	// slash command overlay
	if m.ti.Focused() && m.slashVisible {
		b.WriteString(renderSlashHelp(m.width, m.slashFiltered, m.slashIndex))
	}
	// status bar just below input (hidden when slash dropdown is visible)
	if !(m.ti.Focused() && m.slashVisible) {
		b.WriteString(m.renderStatusBarLine())
	}
	// no persistent operations hint; shown transiently in status bar
	return b.String()
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
	leftParts := []string{now.Format("2006-01-02 15:04:05")}
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
