package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	store "codectl/internal/mcp"
)

func init() {
	mcpCmd.AddCommand(mcpAddCmd)
}

var mcpAddCmd = &cobra.Command{
	Use:   "add <name>...",
	Short: "添加 MCP 服务端",
	Long:  "按名称添加 MCP 服务端（默认使用 npx -y <name> --stdio 启动，可手动编辑 ~/.codectl/mcp.json 自定义 command/args）。",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		items := make([]string, 0, len(args))
		for _, a := range args {
			a = strings.TrimSpace(a)
			if a != "" {
				items = append(items, a)
			}
		}
		if len(items) == 0 {
			fmt.Println("未提供有效服务端名称")
			return nil
		}
		added, existed, err := store.Add(items)
		if err != nil {
			return err
		}
		for _, s := range added {
			fmt.Printf("✓ 已添加：%s\n", s)
		}
		for _, s := range existed {
			fmt.Printf("• 已存在：%s\n", s)
		}
		if len(added) == 0 && len(existed) == 0 {
			fmt.Println("无变更")
		}
		return nil
	},
}
