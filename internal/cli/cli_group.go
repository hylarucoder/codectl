package cli

import (
	"github.com/spf13/cobra"
)

// cliCmd is a group command to organize CLI management subcommands.
var cliCmd = &cobra.Command{
	Use:   "cli",
	Short: "管理受支持的 CLI 工具",
	Long:  "对受支持的开发者 CLI 工具进行添加、卸载与升级等操作。",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	// Attach the group to root; subcommands are added in their respective files.
	rootCmd.AddCommand(cliCmd)
}
