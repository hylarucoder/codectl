package cli

import "github.com/spf13/cobra"

// mcpCmd groups MCP management commands.
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "管理 MCP 服务端",
	Long:  "添加、移除与列出 MCP 服务端配置与远端清单。",
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
