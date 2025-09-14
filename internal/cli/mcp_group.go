package cli

import "github.com/spf13/cobra"

// mcpCmd groups MCP management commands.
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "管理 MCP 服务端",
	Long:  "MCP 管理命令入口（占位，当前无子命令）。",
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
