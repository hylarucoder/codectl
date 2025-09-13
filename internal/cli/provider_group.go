package cli

import "github.com/spf13/cobra"

// providerCmd groups provider-related commands.
var providerCmd = &cobra.Command{
    Use:   "provider",
    Short: "管理 provider 清单",
    Long:  "管理远端清单 provider.json（模型与 MCP 列表）。",
}

func init() {
    rootCmd.AddCommand(providerCmd)
}
