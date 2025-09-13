package cli

import (
    "github.com/spf13/cobra"

    "codectl/internal/app"
)

// cliCmd is a group command to organize CLI management subcommands.
var cliCmd = &cobra.Command{
    Use:   "cli",
    Short: "打开 CLI 工具管理 TUI",
    Long:  "以 TUI 方式检查与管理受支持的开发者 CLI 工具（安装/卸载/升级通过斜杠命令）。",
    RunE: func(cmd *cobra.Command, args []string) error {
        return app.Start()
    },
}

func init() {
	// Attach the group to root; subcommands are added in their respective files.
	rootCmd.AddCommand(cliCmd)
}
