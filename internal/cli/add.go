package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"codectl/internal/tools"
)

func init() {
	cliCmd.AddCommand(addCmd)
}

var addCmd = &cobra.Command{
	Use:   "add [tool|all]...",
	Short: "添加（安装）受支持的 CLI 工具",
	Long:  "为指定或全部工具执行全局安装（npm -g）。支持：all、codex、claude、gemini。",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		selected := selectTools(args)
		if len(selected) == 0 {
			fmt.Println("未选择任何工具")
			return nil
		}
		for i, t := range selected {
			// prefix like: [1/3] Gemini …
			fmt.Printf("[%d/%d] %s 安装中…\n", i+1, len(selected), t.DisplayName)
			res := tools.CheckTool(t)
			if res.Installed {
				// already installed, skip (show version if available)
				ver := res.Version
				if ver == "" {
					ver = "已安装"
				}
				fmt.Printf("  • 跳过：%s\n", ver)
				continue
			}
			// Install latest
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			err := tools.NpmUpgradeLatest(ctx, t.Package)
			cancel()
			if err != nil {
				fmt.Printf("  × 安装失败：%v\n", err)
				continue
			}
			// Recheck and report
			res2 := tools.CheckTool(t)
			ver := res2.Version
			if strings.TrimSpace(ver) == "" {
				if res2.Latest != "" {
					ver = res2.Latest
				} else {
					ver = "latest"
				}
			}
			fmt.Printf("  ✓ 安装成功 → %s\n", ver)
		}
		return nil
	},
}

// selectTools parses args into a list of ToolInfo.
// Accepts: none (defaults to all), or any of: all, codex, claude, gemini.
func selectTools(args []string) []tools.ToolInfo {
	if len(args) == 0 {
		return tools.Tools
	}
	// normalize args
	m := map[string]bool{}
	for _, a := range args {
		aa := strings.TrimSpace(strings.ToLower(a))
		if aa == "" {
			continue
		}
		m[aa] = true
	}
	if m["all"] {
		return tools.Tools
	}
	sel := make([]tools.ToolInfo, 0, len(tools.Tools))
	for _, t := range tools.Tools {
		id := strings.ToLower(string(t.ID))
		// allow id, display shortnames, and common aliases
		names := []string{
			id,
			strings.ToLower(t.DisplayName),
		}
		switch t.ID {
		case tools.ToolCodex:
			names = append(names, "codex", "openai", "openai-codex")
		case tools.ToolClaude:
			names = append(names, "claude", "claude-code", "anthropic")
		case tools.ToolGemini:
			names = append(names, "gemini", "google")
		}
		match := false
		for _, n := range names {
			if m[n] {
				match = true
				break
			}
		}
		if match {
			sel = append(sel, t)
		}
	}
	return sel
}
