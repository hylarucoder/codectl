package ui

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"

    "codectl/internal/tools"
)

func (m model) View() string {
    if m.quitting {
        return "Goodbye!\n"
    }

    b := &strings.Builder{}
    if m.upgrading {
        b.WriteString(renderBanner(m.cwd))
        fmt.Fprintf(b, "\n  codectl — 正在升级\n\n")
        // show per-tool upgrade status for tools with npm package
        for _, t := range tools.Tools {
            if t.Package == "" {
                continue
            }
            note, ok := m.upgradeNotes[t.ID]
            if !ok || note == "" {
                note = "…"
            }
            fmt.Fprintf(b, "  • %-12s: %s\n", t.ID, note)
        }
        fmt.Fprintf(b, "\n  进度: %d/%d 完成\n", m.upgradeDone, m.upgradeTotal)
        b.WriteString("\n")
        b.WriteString(renderInputUI(m.width, m.ti.View()))
        fmt.Fprintf(b, "\n  操作: q 退出\n\n")
        return b.String()
    }

    b.WriteString(renderBanner(m.cwd))
    fmt.Fprintf(b, "\n  codectl — CLI 版本检测\n\n")
    for _, t := range tools.Tools {
        res, ok := m.results[t.ID]
        if !ok && m.checking {
            fmt.Fprintf(b, "  • %-12s: 检测中…\n", t.ID)
            continue
        }
        if !res.Installed {
            if res.Err != "" {
                if res.Latest != "" {
                    fmt.Fprintf(b, "  • %-12s: 未安装 (最新 %s, %s)\n", t.ID, res.Latest, res.Err)
                } else {
                    fmt.Fprintf(b, "  • %-12s: 未安装 (%s)\n", t.ID, res.Err)
                }
            } else {
                if res.Latest != "" {
                    fmt.Fprintf(b, "  • %-12s: 未安装 (最新 %s)\n", t.ID, res.Latest)
                } else {
                    fmt.Fprintf(b, "  • %-12s: 未安装\n", t.ID)
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
            fmt.Fprintf(b, "  • %-12s: %s  [%s]\n", t.ID, ver, res.Source)
            continue
        }
        if tools.VersionLess(res.Version, latest) {
            // highlight latest in red if update available
            red := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render
            fmt.Fprintf(b, "  • %-12s: %s → %s  [%s]\n", t.ID, ver, red(latest), res.Source)
        } else if tools.NormalizeVersion(res.Version) == tools.NormalizeVersion(latest) {
            // equal to latest; avoid redundant latest notation
            fmt.Fprintf(b, "  • %-12s: %s  [%s]\n", t.ID, ver, res.Source)
        } else {
            // current version is newer (rare); show both
            fmt.Fprintf(b, "  • %-12s: %s (最新 %s)  [%s]\n", t.ID, ver, latest, res.Source)
        }
    }

    if !m.updatedAt.IsZero() {
        fmt.Fprintf(b, "\n  上次更新: %s\n", m.updatedAt.Format("2006-01-02 15:04:05"))
    }
    b.WriteString("\n")
    b.WriteString(renderInputUI(m.width, m.ti.View()))
    // slash command overlay
    if m.ti.Focused() && m.slashVisible {
        b.WriteString("\n")
        b.WriteString(renderSlashHelp(m.width, m.slashFiltered, m.slashIndex))
    }
    if m.lastInput != "" {
        fmt.Fprintf(b, "\n  已提交: %s\n", m.lastInput)
    }
    if m.notice != "" {
        fmt.Fprintf(b, "\n  %s\n", m.notice)
    }
    fmt.Fprintf(b, "\n  操作: r 重新检测 · u 升级到最新 · q 退出 · / 聚焦输入 · Esc 取消输入\n\n")
    return b.String()
}
