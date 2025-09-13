package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"codectl/internal/tools"
)

func init() {
	cliCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update [tool|all]...",
	Short: "升级受支持的 CLI 工具到最新版本",
	Long:  "为指定或全部工具执行全局升级（npm -g pkg@latest）。支持：all、codex、claude、gemini。",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		selected := selectTools(args)
		if len(selected) == 0 {
			fmt.Println("未选择任何工具")
			return nil
		}
		for i, t := range selected {
			fmt.Printf("[%d/%d] %s 升级检查…\n", i+1, len(selected), t.DisplayName)
			res := tools.CheckTool(t)
			if !res.Installed {
				fmt.Println("  • 未安装，跳过（请使用 add）")
				continue
			}
			// already latest?
			if res.Latest != "" && tools.NormalizeVersion(res.Version) == tools.NormalizeVersion(res.Latest) {
				fmt.Printf("  ✓ 已是最新 %s\n", res.Version)
				continue
			}
			// If latest unknown, still attempt upgrade
			needUpgrade := true
			if res.Latest != "" {
				needUpgrade = tools.VersionLess(res.Version, res.Latest)
			}
			if !needUpgrade {
				fmt.Printf("  • 当前 %s（不低于 Registry）\n", res.Version)
				continue
			}
			fmt.Println("  → 执行升级…")
			var err error
			// Prefer the tool's own updater when applicable
			if t.ID == tools.ToolClaude && hasBin("claude") {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				err = runSelfUpdater(ctx, "claude", "update")
				cancel()
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				err = tools.NpmUpgradeLatest(ctx, t.Package)
				cancel()
			}
			if err != nil {
				fmt.Printf("  × 升级失败：%v\n", err)
				continue
			}
			// Recheck and report
			res2 := tools.CheckTool(t)
			newVer := strings.TrimSpace(res2.Version)
			if newVer == "" {
				if res2.Latest != "" {
					newVer = res2.Latest
				} else {
					newVer = "latest"
				}
			}
			fmt.Printf("  ✓ 升级成功 → %s\n", newVer)
		}
		return nil
	},
}

// hasBin checks if a binary is available in PATH.
func hasBin(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// runSelfUpdater runs a tool's self-update command with a safe environment.
func runSelfUpdater(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := cmd.CombinedOutput()
	// Always show output for transparency
	s := strings.TrimSpace(string(out))
	if s != "" {
		for _, line := range strings.Split(s, "\n") {
			fmt.Printf("    %s\n", line)
		}
	}
	return err
}
