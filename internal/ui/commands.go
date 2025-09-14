package ui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"codectl/internal/tools"
)

// Commands
func checkAllCmd() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(tools.Tools))
	for _, t := range tools.Tools {
		t := t
		cmds = append(cmds, func() tea.Msg {
			res := tools.CheckTool(t)
			return versionMsg{id: t.ID, result: res}
		})
	}
	return tea.Batch(cmds...)
}

func upgradeOneCmd(t tools.ToolInfo) tea.Cmd {
	return func() tea.Msg {
		// initial check
		res := tools.CheckTool(t)
		if !res.Installed {
			return upgradeProgressMsg{id: t.ID, note: "未安装，跳过"}
		}
		// already latest?
		if res.Latest != "" && tools.NormalizeVersion(res.Version) == tools.NormalizeVersion(res.Latest) {
			return upgradeProgressMsg{id: t.ID, note: fmt.Sprintf("已是最新 %s", res.Version)}
		}
		// If latest unknown, still attempt upgrade to latest
		needUpgrade := true
		if res.Latest != "" {
			needUpgrade = tools.VersionLess(res.Version, res.Latest)
		}
		if !needUpgrade {
			return upgradeProgressMsg{id: t.ID, note: fmt.Sprintf("当前 %s (不低于 Registry)", res.Version)}
		}
		// Perform upgrade
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		err := tools.NpmUpgradeLatest(ctx, t.Package)
		cancel()
		if err != nil {
			return upgradeProgressMsg{id: t.ID, note: fmt.Sprintf("升级失败：%v", err)}
		}
		// optional: quick re-check for new version
		res2 := tools.CheckTool(t)
		newVer := res2.Version
		if newVer == "" {
			if res.Latest != "" {
				newVer = res.Latest
			} else {
				newVer = "latest"
			}
		}
		return upgradeProgressMsg{id: t.ID, note: fmt.Sprintf("升级成功 → %s", newVer)}
	}
}
